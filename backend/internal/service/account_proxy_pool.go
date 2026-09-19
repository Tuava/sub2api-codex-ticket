package service

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type accountProxyPoolRuntime struct {
	proxies []*Proxy
}

var accountProxyPools sync.Map        // accountID -> *accountProxyPoolRuntime
var accountProxyPoolCounters sync.Map // accountID -> *atomic.Uint64

type accountProxyPoolResolvedContextKey struct{}
type accountProxyPoolEligibleContextKey struct{}

// WithAccountProxyPoolEligible marks a real user-facing upstream request as
// eligible for proxy-pool rotation. Maintenance traffic deliberately does not
// carry this marker and therefore keeps using the account's primary proxy.
func WithAccountProxyPoolEligible(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, accountProxyPoolEligibleContextKey{}, true)
}

func AccountProxyPoolEligible(ctx context.Context) bool {
	return ctx != nil && ctx.Value(accountProxyPoolEligibleContextKey{}) == true
}

func WithAccountProxyPoolResolved(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, accountProxyPoolResolvedContextKey{}, true)
}

func AccountProxyPoolResolved(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	resolved, _ := ctx.Value(accountProxyPoolResolvedContextKey{}).(bool)
	return resolved
}

// RegisterAccountProxyPool publishes the hydrated proxy pool for central
// HTTP-upstream selection. Accounts without additional proxies keep the legacy
// caller-provided proxy URL and are removed from this registry.
func RegisterAccountProxyPool(account *Account) {
	if account == nil || account.ID <= 0 || len(AccountProxyPoolIDs(account.Extra)) == 0 {
		if account != nil && account.ID > 0 {
			accountProxyPools.Delete(account.ID)
			accountProxyPoolCounters.Delete(account.ID)
		}
		return
	}
	proxies := make([]*Proxy, 0, len(account.ProxyPool)+1)
	seen := map[int64]struct{}{}
	appendProxy := func(proxy *Proxy) {
		if proxy == nil || proxy.ID <= 0 || proxy.URL() == "" {
			return
		}
		if _, exists := seen[proxy.ID]; exists {
			return
		}
		seen[proxy.ID] = struct{}{}
		proxies = append(proxies, proxy)
	}
	appendProxy(account.Proxy)
	for _, proxy := range account.ProxyPool {
		appendProxy(proxy)
	}
	accountProxyPools.Store(account.ID, &accountProxyPoolRuntime{proxies: proxies})
}

// ResolveAccountProxyPoolURL returns the next proxy for an account. The
// fallback is returned unchanged when no hydrated multi-proxy pool exists.
func ResolveAccountProxyPoolURL(accountID int64, fallback string) string {
	raw, ok := accountProxyPools.Load(accountID)
	if !ok {
		return fallback
	}
	pool, ok := raw.(*accountProxyPoolRuntime)
	if !ok || pool == nil {
		return fallback
	}
	candidates := healthyAccountProxyCandidates(pool.proxies, timeNow())
	if len(candidates) == 0 {
		return ""
	}
	index := nextAccountProxyPoolIndex(accountID, len(candidates))
	return candidates[index].URL()
}

func resolveAccountProxyPoolProxy(account *Account) *Proxy {
	if account == nil {
		return nil
	}
	proxies := make([]*Proxy, 0, len(account.ProxyPool)+1)
	proxies = append(proxies, account.Proxy)
	proxies = append(proxies, account.ProxyPool...)
	candidates := healthyAccountProxyCandidates(proxies, timeNow())
	if len(candidates) == 0 {
		return nil
	}
	raw, _ := accountProxyPools.LoadOrStore(account.ID, &accountProxyPoolRuntime{proxies: proxies})
	pool, ok := raw.(*accountProxyPoolRuntime)
	if !ok || pool == nil {
		return candidates[0]
	}
	index := nextAccountProxyPoolIndex(account.ID, len(candidates))
	return candidates[index]
}

func nextAccountProxyPoolIndex(accountID int64, size int) int {
	if size <= 1 {
		return 0
	}
	raw, _ := accountProxyPoolCounters.LoadOrStore(accountID, &atomic.Uint64{})
	counter, ok := raw.(*atomic.Uint64)
	if !ok || counter == nil {
		return 0
	}
	return int(counter.Add(1)-1) % size
}

func healthyAccountProxyCandidates(proxies []*Proxy, now time.Time) []*Proxy {
	seen := make(map[int64]struct{}, len(proxies))
	candidates := make([]*Proxy, 0, len(proxies))
	for _, proxy := range proxies {
		if proxy == nil || proxy.ID <= 0 || !proxy.IsActive() || proxy.IsExpired(now) || proxy.URL() == "" {
			continue
		}
		if _, exists := seen[proxy.ID]; exists {
			continue
		}
		seen[proxy.ID] = struct{}{}
		candidates = append(candidates, proxy)
	}
	return candidates
}

var timeNow = func() time.Time { return time.Now() }

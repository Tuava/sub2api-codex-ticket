package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func testAccountProxy(id int64, status string, expiresAt *time.Time) *Proxy {
	return &Proxy{
		ID:        id,
		Name:      "proxy",
		Protocol:  "http",
		Host:      "127.0.0.1",
		Port:      8000 + int(id),
		Status:    status,
		ExpiresAt: expiresAt,
	}
}

func TestAccountProxyPoolIDsNormalizesJSONValues(t *testing.T) {
	extra := map[string]any{
		AccountProxyPoolIDsExtraKey: []any{
			float64(2), json.Number("3"), int64(2), int(4), float64(-1), float64(6.5), "5",
		},
	}
	require.Equal(t, []int64{2, 3, 4}, AccountProxyPoolIDs(extra))
}

func TestAccountNextProxyRoundRobin(t *testing.T) {
	account := &Account{
		ID:        91001,
		Proxy:     testAccountProxy(1, StatusActive, nil),
		ProxyPool: []*Proxy{testAccountProxy(2, StatusActive, nil), testAccountProxy(3, StatusActive, nil)},
		Extra: map[string]any{
			AccountProxyPoolIDsExtraKey: []int64{2, 3},
		},
	}
	accountProxyPools.Delete(account.ID)
	accountProxyPoolCounters.Delete(account.ID)
	RegisterAccountProxyPool(account)
	t.Cleanup(func() {
		accountProxyPools.Delete(account.ID)
		accountProxyPoolCounters.Delete(account.ID)
	})

	got := make([]int64, 0, 4)
	for i := 0; i < 4; i++ {
		got = append(got, account.NextProxy().ID)
	}
	require.Equal(t, []int64{1, 2, 3, 1}, got)
}

func TestAccountNextProxySkipsUnavailablePoolMembers(t *testing.T) {
	expiredAt := time.Now().Add(-time.Minute)
	account := &Account{
		ID:    91002,
		Proxy: testAccountProxy(1, StatusActive, nil),
		ProxyPool: []*Proxy{
			testAccountProxy(2, StatusDisabled, nil),
			testAccountProxy(3, StatusActive, &expiredAt),
			testAccountProxy(4, StatusActive, nil),
		},
		Extra: map[string]any{
			AccountProxyPoolIDsExtraKey: []int64{2, 3, 4, 99},
		},
	}
	accountProxyPools.Delete(account.ID)
	accountProxyPoolCounters.Delete(account.ID)
	RegisterAccountProxyPool(account)
	t.Cleanup(func() {
		accountProxyPools.Delete(account.ID)
		accountProxyPoolCounters.Delete(account.ID)
	})

	require.Equal(t, int64(1), account.NextProxy().ID)
	require.Equal(t, int64(4), account.NextProxy().ID)
	require.Equal(t, int64(1), account.NextProxy().ID)
}

func TestAccountNextProxyLegacyPrimaryOnly(t *testing.T) {
	primary := testAccountProxy(1, StatusActive, nil)
	account := &Account{ID: 91003, Proxy: primary}
	require.Same(t, primary, account.NextProxy())
	require.Equal(t, primary.URL(), account.NextProxyURL())
}

func TestNormalizeAccountProxyPoolExtraDropsPrimaryAndDuplicates(t *testing.T) {
	primaryID := int64(2)
	extra := NormalizeAccountProxyPoolExtra(map[string]any{
		AccountProxyPoolIDsExtraKey: []any{float64(2), float64(3), float64(3), float64(4)},
	}, &primaryID)
	require.Equal(t, []int64{3, 4}, AccountProxyPoolIDs(extra))
}

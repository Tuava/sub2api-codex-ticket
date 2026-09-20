package repository

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type githubReleaseClient struct {
	httpClient         *http.Client
	downloadHTTPClient *http.Client
	updateGitHubToken  string
}

type githubReleaseClientError struct {
	err error
}

type githubAPIError struct {
	StatusCode      int
	Message         string
	Remaining       string
	ResetUnix       int64
	TokenConfigured bool
}

func (e *githubAPIError) Error() string {
	if e == nil {
		return "GitHub API request failed"
	}
	message := strings.TrimSpace(e.Message)
	if e.StatusCode == http.StatusForbidden && e.Remaining == "0" {
		reset := ""
		if e.ResetUnix > 0 {
			reset = time.Unix(e.ResetUnix, 0).UTC().Format(time.RFC3339)
		}
		if e.TokenConfigured {
			if reset != "" {
				return fmt.Sprintf("GitHub API rate limit exceeded; configured token was rejected or exhausted; retry after %s", reset)
			}
			return "GitHub API rate limit exceeded; configured token was rejected or exhausted"
		}
		if reset != "" {
			return fmt.Sprintf("GitHub anonymous API rate limit exceeded; retry after %s or configure UPDATE_GITHUB_TOKEN", reset)
		}
		return "GitHub anonymous API rate limit exceeded; configure UPDATE_GITHUB_TOKEN and retry"
	}
	if message == "" {
		message = http.StatusText(e.StatusCode)
	}
	return fmt.Sprintf("GitHub API returned %d: %s", e.StatusCode, message)
}

func (e *githubAPIError) GitHubAPIStatus() int             { return e.StatusCode }
func (e *githubAPIError) GitHubRateLimitRemaining() string { return e.Remaining }
func (e *githubAPIError) GitHubRateLimitResetUnix() int64  { return e.ResetUnix }
func (e *githubAPIError) GitHubTokenConfigured() bool      { return e.TokenConfigured }

type githubAtomFeed struct {
	Entries []githubAtomEntry `xml:"entry"`
}

type githubAtomEntry struct {
	ID      string           `xml:"id"`
	Updated string           `xml:"updated"`
	Title   string           `xml:"title"`
	Content string           `xml:"content"`
	Links   []githubAtomLink `xml:"link"`
}

type githubAtomLink struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}

// NewGitHubReleaseClient 创建 GitHub Release 客户端
// proxyURL 为空时直连 GitHub，支持 http/https/socks5/socks5h 协议
// 代理配置失败时行为由 allowDirectOnProxyError 控制：
//   - false（默认）：返回错误占位客户端，禁止回退到直连
//   - true：回退到直连（仅限管理员显式开启）
func NewGitHubReleaseClient(proxyURL string, allowDirectOnProxyError bool) service.GitHubReleaseClient {
	// 安全说明：httpclient.GetClient 的错误链（url.Parse / proxyutil）不含明文代理凭据，
	// 但仍通过 slog 仅在服务端日志记录，不会暴露给 HTTP 响应。
	sharedClient, err := httpclient.GetClient(httpclient.Options{
		Timeout:  30 * time.Second,
		ProxyURL: proxyURL,
	})
	if err != nil {
		if strings.TrimSpace(proxyURL) != "" && !allowDirectOnProxyError {
			slog.Warn("proxy client init failed, all requests will fail", "service", "github_release", "error", err)
			return &githubReleaseClientError{err: fmt.Errorf("proxy client init failed and direct fallback is disabled; set security.proxy_fallback.allow_direct_on_error=true to allow fallback: %w", err)}
		}
		sharedClient = &http.Client{Timeout: 30 * time.Second}
	}
	apiClient := cloneHTTPClient(sharedClient)
	apiClient.CheckRedirect = githubAPICheckRedirect(apiClient.CheckRedirect)

	// 下载客户端需要更长的超时时间
	downloadClient, err := httpclient.GetClient(httpclient.Options{
		Timeout:  10 * time.Minute,
		ProxyURL: proxyURL,
	})
	if err != nil {
		if strings.TrimSpace(proxyURL) != "" && !allowDirectOnProxyError {
			slog.Warn("proxy download client init failed, all requests will fail", "service", "github_release", "error", err)
			return &githubReleaseClientError{err: fmt.Errorf("proxy client init failed and direct fallback is disabled; set security.proxy_fallback.allow_direct_on_error=true to allow fallback: %w", err)}
		}
		downloadClient = &http.Client{Timeout: 10 * time.Minute}
	}
	downloadClient = cloneHTTPClient(downloadClient)

	return &githubReleaseClient{
		httpClient:         apiClient,
		downloadHTTPClient: downloadClient,
		updateGitHubToken:  firstGitHubToken("UPDATE_GITHUB_TOKEN", "GITHUB_TOKEN", "GH_TOKEN"),
	}
}

func firstGitHubToken(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func cloneHTTPClient(client *http.Client) *http.Client {
	cloned := *client
	return &cloned
}

func isGitHubAPIURL(url *url.URL) bool {
	return url != nil && strings.EqualFold(url.Scheme, "https") && url.User == nil &&
		strings.EqualFold(url.Host, "api.github.com")
}

func githubAPICheckRedirect(previous func(*http.Request, []*http.Request) error) func(*http.Request, []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if !isGitHubAPIURL(req.URL) {
			req.Header.Del("Authorization")
		}
		if previous != nil {
			return previous(req, via)
		}
		return nil
	}
}

func (c *githubReleaseClient) newAPIRequest(ctx context.Context, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Sub2API-Updater")
	if c.updateGitHubToken != "" && isGitHubAPIURL(req.URL) {
		req.Header.Set("Authorization", "Bearer "+c.updateGitHubToken)
	}
	return req, nil
}

func (c *githubReleaseClientError) FetchLatestRelease(ctx context.Context, repo string) (*service.GitHubRelease, error) {
	return nil, c.err
}

func (c *githubReleaseClientError) FetchRecentReleases(ctx context.Context, repo string, perPage int) ([]*service.GitHubRelease, error) {
	return nil, c.err
}

func (c *githubReleaseClientError) DownloadFile(ctx context.Context, url, dest string, maxSize int64) error {
	return c.err
}

func (c *githubReleaseClientError) FetchChecksumFile(ctx context.Context, url string) ([]byte, error) {
	return nil, c.err
}

func (c *githubReleaseClient) FetchLatestRelease(ctx context.Context, repo string) (*service.GitHubRelease, error) {
	normalizedRepo, err := normalizeGitHubRepository(repo)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", normalizedRepo)

	req, err := c.newAPIRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		apiErr := c.parseAPIError(resp)
		if shouldFallbackToGitHubAtom(resp.StatusCode) {
			releases, atomErr := c.fetchAtomReleases(ctx, repo, 1)
			if atomErr == nil && len(releases) > 0 {
				slog.Warn("github API unavailable; using releases Atom fallback",
					"service", "github_release", "repo", repo, "error", apiErr)
				return releases[0], nil
			}
			return nil, fmt.Errorf("%w; Atom fallback failed: %v", apiErr, atomErr)
		}
		return nil, apiErr
	}

	var release service.GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

func (c *githubReleaseClient) FetchRecentReleases(ctx context.Context, repo string, perPage int) ([]*service.GitHubRelease, error) {
	if perPage <= 0 {
		perPage = 10
	}
	if perPage > 100 {
		perPage = 100 // GitHub API hard limit
	}
	normalizedRepo, err := normalizeGitHubRepository(repo)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=%d", normalizedRepo, perPage)

	req, err := c.newAPIRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		apiErr := c.parseAPIError(resp)
		if shouldFallbackToGitHubAtom(resp.StatusCode) {
			releases, atomErr := c.fetchAtomReleases(ctx, repo, perPage)
			if atomErr == nil {
				slog.Warn("github API unavailable; using releases Atom fallback",
					"service", "github_release", "repo", repo, "error", apiErr)
				return releases, nil
			}
			return nil, fmt.Errorf("%w; Atom fallback failed: %v", apiErr, atomErr)
		}
		return nil, apiErr
	}

	var releases []*service.GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}

	return releases, nil
}

func shouldFallbackToGitHubAtom(statusCode int) bool {
	return statusCode == http.StatusForbidden || statusCode == http.StatusTooManyRequests || statusCode >= 500
}

func (c *githubReleaseClient) parseAPIError(resp *http.Response) error {
	if resp == nil {
		return &githubAPIError{Message: "empty GitHub API response", TokenConfigured: c.updateGitHubToken != ""}
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 32*1024))
	var payload struct {
		Message string `json:"message"`
	}
	_ = json.Unmarshal(body, &payload)
	resetUnix, _ := strconv.ParseInt(strings.TrimSpace(resp.Header.Get("X-RateLimit-Reset")), 10, 64)
	return &githubAPIError{
		StatusCode:      resp.StatusCode,
		Message:         payload.Message,
		Remaining:       strings.TrimSpace(resp.Header.Get("X-RateLimit-Remaining")),
		ResetUnix:       resetUnix,
		TokenConfigured: c.updateGitHubToken != "",
	}
}

func normalizeGitHubRepository(repo string) (string, error) {
	repo = strings.TrimSpace(repo)
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("invalid GitHub repository %q", repo)
	}
	for _, part := range parts {
		if strings.ContainsAny(part, "?#@\\") || part == "." || part == ".." {
			return "", fmt.Errorf("invalid GitHub repository %q", repo)
		}
	}
	return url.PathEscape(parts[0]) + "/" + url.PathEscape(parts[1]), nil
}

func (c *githubReleaseClient) fetchAtomReleases(ctx context.Context, repo string, limit int) ([]*service.GitHubRelease, error) {
	normalizedRepo, err := normalizeGitHubRepository(repo)
	if err != nil {
		return nil, err
	}
	feedURL := "https://github.com/" + normalizedRepo + "/releases.atom"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/atom+xml, application/xml;q=0.9")
	req.Header.Set("User-Agent", "Sub2API-Updater")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub releases Atom returned %d", resp.StatusCode)
	}
	var feed githubAtomFeed
	if err := xml.NewDecoder(io.LimitReader(resp.Body, 4*1024*1024)).Decode(&feed); err != nil {
		return nil, fmt.Errorf("decode GitHub releases Atom: %w", err)
	}
	if limit <= 0 || limit > len(feed.Entries) {
		limit = len(feed.Entries)
	}
	releases := make([]*service.GitHubRelease, 0, limit)
	for _, entry := range feed.Entries {
		htmlURL := ""
		for _, link := range entry.Links {
			if link.Rel == "alternate" || (link.Rel == "" && htmlURL == "") {
				htmlURL = strings.TrimSpace(link.Href)
			}
		}
		tagName := githubReleaseTagFromURL(htmlURL)
		if tagName == "" {
			continue
		}
		version := strings.TrimPrefix(tagName, "v")
		releases = append(releases, &service.GitHubRelease{
			TagName:     tagName,
			Name:        strings.TrimSpace(entry.Title),
			Body:        strings.TrimSpace(html.UnescapeString(entry.Content)),
			PublishedAt: strings.TrimSpace(entry.Updated),
			HTMLURL:     htmlURL,
			Prerelease:  isGitHubPrereleaseTag(version),
			Assets:      synthesizedGitHubReleaseAssets(repo, tagName, version),
		})
		if len(releases) == limit {
			break
		}
	}
	if len(releases) == 0 {
		return nil, fmt.Errorf("GitHub releases Atom contains no usable releases")
	}
	return releases, nil
}

func githubReleaseTagFromURL(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return ""
	}
	const marker = "/releases/tag/"
	index := strings.Index(parsed.Path, marker)
	if index < 0 {
		return ""
	}
	tag, err := url.PathUnescape(strings.Trim(parsed.Path[index+len(marker):], "/"))
	if err != nil {
		return ""
	}
	return tag
}

func isGitHubPrereleaseTag(version string) bool {
	lower := strings.ToLower(version)
	for _, marker := range []string{"-alpha", "-beta", "-rc", "-pre", "-preview"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func synthesizedGitHubReleaseAssets(repo, tagName, version string) []service.GitHubAsset {
	base := fmt.Sprintf("https://github.com/%s/releases/download/%s/", repo, url.PathEscape(tagName))
	names := []string{
		fmt.Sprintf("sub2api_%s_linux_amd64.tar.gz", version),
		fmt.Sprintf("sub2api_%s_linux_arm64.tar.gz", version),
		fmt.Sprintf("sub2api_%s_darwin_amd64.tar.gz", version),
		fmt.Sprintf("sub2api_%s_darwin_arm64.tar.gz", version),
		fmt.Sprintf("sub2api_%s_windows_amd64.zip", version),
		"checksums.txt",
	}
	assets := make([]service.GitHubAsset, 0, len(names))
	for _, name := range names {
		assets = append(assets, service.GitHubAsset{Name: name, BrowserDownloadURL: base + url.PathEscape(name)})
	}
	return assets
}

func (c *githubReleaseClient) DownloadFile(ctx context.Context, url, dest string, maxSize int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	// 使用预配置的下载客户端（已包含代理配置）
	resp, err := c.downloadHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	// SECURITY: Check Content-Length if available
	if resp.ContentLength > maxSize {
		return fmt.Errorf("file too large: %d bytes (max %d)", resp.ContentLength, maxSize)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}

	// SECURITY: Use LimitReader to enforce max download size even if Content-Length is missing/wrong
	limited := io.LimitReader(resp.Body, maxSize+1)
	written, err := io.Copy(out, limited)

	// Close file before attempting to remove (required on Windows)
	_ = out.Close()

	if err != nil {
		_ = os.Remove(dest) // Clean up partial file (best-effort)
		return err
	}

	// Check if we hit the limit (downloaded more than maxSize)
	if written > maxSize {
		_ = os.Remove(dest) // Clean up partial file (best-effort)
		return fmt.Errorf("download exceeded maximum size of %d bytes", maxSize)
	}

	return nil
}

func (c *githubReleaseClient) FetchChecksumFile(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

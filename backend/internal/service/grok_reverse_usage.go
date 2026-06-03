package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	httppool "github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"golang.org/x/sync/errgroup"
)

const grokWebRateLimitsPath = "/rest/rate-limits"

type grokQuotaMode struct {
	Key                  string
	DefaultWindowSeconds int
}

type grokRateLimitsResponse struct {
	WindowSizeSeconds *int `json:"windowSizeSeconds"`
	RemainingQueries  *int `json:"remainingQueries"`
	TotalQueries      *int `json:"totalQueries"`
}

type grokRateLimitsHTTPError struct {
	status int
	body   string
}

func (e *grokRateLimitsHTTPError) Error() string {
	msg := strings.TrimSpace(e.body)
	if msg == "" {
		msg = http.StatusText(e.status)
	}
	return fmt.Sprintf("grok rate-limits returned %d: %s", e.status, msg)
}

var grokQuotaModes = []grokQuotaMode{
	{Key: "auto", DefaultWindowSeconds: 7200},
	{Key: "fast", DefaultWindowSeconds: 86400},
	{Key: "expert", DefaultWindowSeconds: 7200},
	{Key: "heavy", DefaultWindowSeconds: 7200},
	{Key: "grok-420-computer-use-sa", DefaultWindowSeconds: 7200},
}

func (s *AccountUsageService) getGrokUsage(ctx context.Context, account *Account, force bool) (*UsageInfo, error) {
	if account == nil || !account.IsXAICookie() {
		now := time.Now()
		return &UsageInfo{UpdatedAt: &now}, nil
	}

	cache := s.cache
	if cache == nil {
		return s.fetchGrokUsage(ctx, account)
	}

	if !force {
		if cached, ok := cache.grokCache.Load(account.ID); ok {
			if entry, ok := cached.(*grokUsageCache); ok && time.Since(entry.timestamp) < grokCacheTTL(entry.usageInfo) {
				return entry.usageInfo, nil
			}
		}
	}

	flightKey := fmt.Sprintf("grok-usage:%d", account.ID)
	result, flightErr, _ := cache.grokFlight.Do(flightKey, func() (any, error) {
		if !force {
			if cached, ok := cache.grokCache.Load(account.ID); ok {
				if entry, ok := cached.(*grokUsageCache); ok && time.Since(entry.timestamp) < grokCacheTTL(entry.usageInfo) {
					return entry.usageInfo, nil
				}
			}
		}

		fetchCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		usage, err := s.fetchGrokUsage(fetchCtx, account)
		if err != nil {
			degraded := buildGrokDegradedUsage(err)
			cache.grokCache.Store(account.ID, &grokUsageCache{
				usageInfo: degraded,
				timestamp: time.Now(),
			})
			return degraded, nil
		}

		cache.grokCache.Store(account.ID, &grokUsageCache{
			usageInfo: usage,
			timestamp: time.Now(),
		})
		s.persistGrokQuotaSnapshot(account.ID, usage)
		return usage, nil
	})
	if flightErr != nil {
		return nil, flightErr
	}
	usage, ok := result.(*UsageInfo)
	if !ok || usage == nil {
		now := time.Now()
		return &UsageInfo{UpdatedAt: &now}, nil
	}
	return usage, nil
}

func (s *AccountUsageService) fetchGrokUsage(ctx context.Context, account *Account) (*UsageInfo, error) {
	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	quotas, err := fetchGrokRateLimits(ctx, account, proxyURL)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	return &UsageInfo{
		UpdatedAt: &now,
		GrokQuota: quotas,
	}, nil
}

func fetchGrokRateLimits(ctx context.Context, account *Account, proxyURL string) (map[string]*GrokModelQuota, error) {
	client, err := httppool.GetClient(httppool.Options{
		ProxyURL:              proxyURL,
		Timeout:               25 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("build grok rate-limits client: %w", err)
	}

	quotas := make(map[string]*GrokModelQuota, len(grokQuotaModes))
	var (
		mu       sync.Mutex
		firstErr error
	)
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(len(grokQuotaModes))
	for _, mode := range grokQuotaModes {
		mode := mode
		g.Go(func() error {
			quota, err := fetchGrokRateLimitMode(gctx, client, account, mode)
			if err != nil {
				if isInvalidGrokCredentialsError(err) {
					return err
				}
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return nil
			}
			if quota == nil {
				return nil
			}
			mu.Lock()
			quotas[mode.Key] = quota
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	if len(quotas) == 0 {
		if firstErr != nil {
			return nil, firstErr
		}
		return nil, errors.New("grok rate-limits response missing quota fields")
	}
	return quotas, nil
}

func fetchGrokRateLimitMode(ctx context.Context, client *http.Client, account *Account, mode grokQuotaMode) (*GrokModelQuota, error) {
	body, err := buildGrokRateLimitsPayload(mode.Key)
	if err != nil {
		return nil, err
	}
	req, err := buildGrokWebRateLimitsRequest(ctx, account, body)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("grok rate-limits request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read grok rate-limits response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, &grokRateLimitsHTTPError{
			status: resp.StatusCode,
			body:   string(respBody),
		}
	}

	quota, err := parseGrokRateLimitsResponse(respBody, mode, time.Now())
	if err != nil {
		return nil, err
	}
	return quota, nil
}

func buildGrokRateLimitsPayload(modelName string) ([]byte, error) {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return nil, errors.New("grok rate-limits modelName is required")
	}
	return json.Marshal(map[string]string{"modelName": modelName})
}

func buildGrokWebRateLimitsRequest(ctx context.Context, account *Account, body []byte) (*http.Request, error) {
	targetURL, err := grokWebEndpointURL(account, grokWebRateLimitsPath)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", grokFirstNonEmpty(account.GetCredential("accept_language"), "zh-CN,zh;q=0.9,en;q=0.8"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", grokFirstNonEmpty(account.GetCredential("origin"), grokWebDefaultBaseURL))
	req.Header.Set("Referer", grokFirstNonEmpty(account.GetCredential("referer"), grokWebDefaultBaseURL+"/"))
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("User-Agent", grokFirstNonEmpty(account.GetCredential("user_agent"), defaultGrokWebUserAgent()))
	req.Header.Set("x-xai-request-id", newGrokRequestID())
	req.Header.Set("Cookie", buildGrokWebCookieHeader(account))
	if req.Header.Get("Cookie") == "" {
		return nil, errors.New("sso_token or cookie is required for Grok Cookie account")
	}
	return req, nil
}

func parseGrokRateLimitsResponse(body []byte, mode grokQuotaMode, now time.Time) (*GrokModelQuota, error) {
	var parsed grokRateLimitsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse grok rate-limits response: %w", err)
	}
	if parsed.RemainingQueries == nil {
		return nil, nil
	}
	remaining := *parsed.RemainingQueries
	if remaining < 0 {
		remaining = 0
	}
	total := remaining
	if parsed.TotalQueries != nil {
		total = *parsed.TotalQueries
	}
	if total < remaining {
		total = remaining
	}
	if total < 0 {
		total = 0
	}
	windowSeconds := mode.DefaultWindowSeconds
	if parsed.WindowSizeSeconds != nil && *parsed.WindowSizeSeconds > 0 {
		windowSeconds = *parsed.WindowSizeSeconds
	}

	utilization := 0
	if total > 0 {
		used := total - remaining
		utilization = int((float64(used)/float64(total))*100 + 0.5)
	}

	resetAt := ""
	if windowSeconds > 0 {
		resetAt = now.Add(time.Duration(windowSeconds) * time.Second).UTC().Format(time.RFC3339)
	}

	return &GrokModelQuota{
		Utilization:       utilization,
		ResetTime:         resetAt,
		RemainingQueries:  remaining,
		TotalQueries:      total,
		WindowSizeSeconds: windowSeconds,
	}, nil
}

func grokCacheTTL(info *UsageInfo) time.Duration {
	if info == nil {
		return antigravityErrorTTL
	}
	if info.ErrorCode != "" || info.Error != "" {
		return antigravityErrorTTL
	}
	return apiCacheTTL
}

func buildGrokDegradedUsage(err error) *UsageInfo {
	now := time.Now()
	info := &UsageInfo{
		UpdatedAt: &now,
		Error:     fmt.Sprintf("usage API error: %v", err),
	}
	slog.Warn("grok usage fetch failed, returning degraded response", "error", err)

	var httpErr *grokRateLimitsHTTPError
	if errors.As(err, &httpErr) {
		switch {
		case isInvalidGrokCredentialsBody(httpErr.body):
			info.ErrorCode = errorCodeUnauthenticated
			info.NeedsReauth = true
		case httpErr.status == http.StatusTooManyRequests:
			info.ErrorCode = errorCodeRateLimited
		case httpErr.status == http.StatusUnauthorized || httpErr.status == http.StatusForbidden:
			info.ErrorCode = errorCodeUnauthenticated
			info.NeedsReauth = true
		default:
			info.ErrorCode = errorCodeNetworkError
		}
		return info
	}

	if isInvalidGrokCredentialsError(err) {
		info.ErrorCode = errorCodeUnauthenticated
		info.NeedsReauth = true
		return info
	}
	info.ErrorCode = errorCodeNetworkError
	return info
}

func isInvalidGrokCredentialsError(err error) bool {
	var httpErr *grokRateLimitsHTTPError
	if !errors.As(err, &httpErr) {
		return false
	}
	if httpErr.status != http.StatusBadRequest && httpErr.status != http.StatusUnauthorized && httpErr.status != http.StatusForbidden {
		return false
	}
	return isInvalidGrokCredentialsBody(httpErr.body)
}

func isInvalidGrokCredentialsBody(body string) bool {
	text := strings.ToLower(body)
	return strings.Contains(text, "invalid-credentials") ||
		strings.Contains(text, "bad-credentials") ||
		strings.Contains(text, "failed to look up session id") ||
		strings.Contains(text, "blocked-user") ||
		strings.Contains(text, "email-domain-rejected") ||
		strings.Contains(text, "session not found") ||
		strings.Contains(text, "account suspended") ||
		strings.Contains(text, "token revoked") ||
		strings.Contains(text, "token expired")
}

func (s *AccountUsageService) persistGrokQuotaSnapshot(accountID int64, usage *UsageInfo) {
	if s == nil || s.accountRepo == nil || accountID <= 0 || usage == nil || len(usage.GrokQuota) == 0 {
		return
	}
	updates := grokQuotaExtraUpdates(usage)
	if len(updates) == 0 {
		return
	}
	go func() {
		updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.accountRepo.UpdateExtra(updateCtx, accountID, updates); err != nil {
			slog.Warn("persist_grok_quota_snapshot_failed", "account_id", accountID, "error", err)
		}
	}()
}

func grokQuotaExtraUpdates(usage *UsageInfo) map[string]any {
	if usage == nil || len(usage.GrokQuota) == 0 {
		return nil
	}
	quota := make(map[string]any, len(usage.GrokQuota))
	for mode, q := range usage.GrokQuota {
		if q == nil {
			continue
		}
		quota[mode] = map[string]any{
			"utilization":         q.Utilization,
			"reset_time":          q.ResetTime,
			"remaining_queries":   q.RemainingQueries,
			"total_queries":       q.TotalQueries,
			"window_size_seconds": q.WindowSizeSeconds,
		}
	}
	if len(quota) == 0 {
		return nil
	}
	updatedAt := time.Now().UTC()
	if usage.UpdatedAt != nil {
		updatedAt = usage.UpdatedAt.UTC()
	}
	return map[string]any{
		"grok_quota":            quota,
		"grok_quota_updated_at": updatedAt.Format(time.RFC3339),
	}
}

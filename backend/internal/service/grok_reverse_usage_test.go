package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildGrokWebRateLimitsRequest(t *testing.T) {
	account := &Account{
		Platform: PlatformXAI,
		Type:     AccountTypeCookie,
		Credentials: map[string]any{
			"base_url":        "https://grok.example",
			"sso_token":       "tok",
			"accept_language": "en-US,en;q=0.9",
			"user_agent":      "ua",
		},
	}
	payload, err := buildGrokRateLimitsPayload("fast")
	require.NoError(t, err)

	req, err := buildGrokWebRateLimitsRequest(context.Background(), account, payload)
	require.NoError(t, err)

	require.Equal(t, "https://grok.example/rest/rate-limits", req.URL.String())
	require.Equal(t, http.MethodPost, req.Method)
	require.Equal(t, "application/json", req.Header.Get("Content-Type"))
	require.Equal(t, "en-US,en;q=0.9", req.Header.Get("Accept-Language"))
	require.Equal(t, "ua", req.Header.Get("User-Agent"))
	require.Contains(t, req.Header.Get("Cookie"), "sso=tok")
	require.Contains(t, req.Header.Get("Cookie"), "sso-rw=tok")

	body, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"modelName":"fast"}`, string(body))
}

func TestParseGrokRateLimitsResponse(t *testing.T) {
	now := time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC)

	quota, err := parseGrokRateLimitsResponse(
		[]byte(`{"windowSizeSeconds":3600,"remainingQueries":5,"totalQueries":20}`),
		grokQuotaMode{Key: "fast", DefaultWindowSeconds: 86400},
		now,
	)
	require.NoError(t, err)
	require.NotNil(t, quota)
	require.Equal(t, 75, quota.Utilization)
	require.Equal(t, 5, quota.RemainingQueries)
	require.Equal(t, 20, quota.TotalQueries)
	require.Equal(t, 3600, quota.WindowSizeSeconds)
	require.Equal(t, "2026-06-03T11:00:00Z", quota.ResetTime)
}

func TestAccountUsageServiceGetGrokUsageFetchesQuotasAndPersists(t *testing.T) {
	var (
		mu   sync.Mutex
		seen = map[string]int{}
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/rate-limits" {
			http.Error(w, "bad path", http.StatusNotFound)
			return
		}
		if !stringsContainsAll(r.Header.Get("Cookie"), "sso=tok", "sso-rw=tok") {
			http.Error(w, "missing cookie", http.StatusUnauthorized)
			return
		}
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		mode := payload["modelName"]
		mu.Lock()
		seen[mode]++
		mu.Unlock()

		resp := map[string]int{
			"windowSizeSeconds": 7200,
			"remainingQueries":  8,
			"totalQueries":      10,
		}
		if mode == "fast" {
			resp["windowSizeSeconds"] = 3600
			resp["remainingQueries"] = 5
			resp["totalQueries"] = 20
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	account := Account{
		ID:       701,
		Platform: PlatformXAI,
		Type:     AccountTypeCookie,
		Credentials: map[string]any{
			"base_url":  server.URL,
			"sso_token": "tok",
		},
		Concurrency: 1,
	}
	repo := &accountUsageCodexProbeRepo{
		stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: []Account{account}},
		updateExtraCh:         make(chan map[string]any, 1),
	}
	svc := &AccountUsageService{
		accountRepo: repo,
		cache:       NewUsageCache(),
	}

	usage, err := svc.GetUsage(context.Background(), account.ID, true)
	require.NoError(t, err)
	require.NotNil(t, usage)
	require.Len(t, usage.GrokQuota, 5)
	require.Equal(t, 75, usage.GrokQuota["fast"].Utilization)
	require.Equal(t, 5, usage.GrokQuota["fast"].RemainingQueries)
	require.Equal(t, 20, usage.GrokQuota["fast"].TotalQueries)
	require.NotEmpty(t, usage.GrokQuota["auto"].ResetTime)

	mu.Lock()
	require.Equal(t, 1, seen["auto"])
	require.Equal(t, 1, seen["fast"])
	require.Equal(t, 1, seen["expert"])
	require.Equal(t, 1, seen["heavy"])
	require.Equal(t, 1, seen["grok-420-computer-use-sa"])
	mu.Unlock()

	select {
	case updates := <-repo.updateExtraCh:
		require.NotEmpty(t, updates["grok_quota_updated_at"])
		quota, ok := updates["grok_quota"].(map[string]any)
		require.True(t, ok)
		require.Contains(t, quota, "fast")
	case <-time.After(2 * time.Second):
		t.Fatal("等待 Grok quota 快照写入 extra 超时")
	}
}

func TestAccountUsageServiceGetGrokUsageInvalidCredentialsDegrades(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"invalid-credentials"}`, http.StatusUnauthorized)
	}))
	defer server.Close()

	account := Account{
		ID:       702,
		Platform: PlatformXAI,
		Type:     AccountTypeCookie,
		Credentials: map[string]any{
			"base_url":  server.URL,
			"sso_token": "bad",
		},
		Concurrency: 1,
	}
	repo := &accountUsageCodexProbeRepo{
		stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: []Account{account}},
	}
	svc := &AccountUsageService{
		accountRepo: repo,
		cache:       NewUsageCache(),
	}

	usage, err := svc.GetUsage(context.Background(), account.ID, true)
	require.NoError(t, err)
	require.NotNil(t, usage)
	require.Equal(t, errorCodeUnauthenticated, usage.ErrorCode)
	require.True(t, usage.NeedsReauth)
	require.Contains(t, usage.Error, "invalid-credentials")
}

func stringsContainsAll(s string, needles ...string) bool {
	for _, needle := range needles {
		if !strings.Contains(s, needle) {
			return false
		}
	}
	return true
}

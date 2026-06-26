//go:build unit

package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/imroc/req/v3"
	"github.com/stretchr/testify/require"
)

func TestOpenAIQuotaServicePrepareUpstreamCall_ValidOAuthAccount(t *testing.T) {
	account := &Account{
		ID:       1,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token":       " access-token ",
			"chatgpt_account_id": " account-id ",
		},
	}
	service := newOpenAIQuotaServiceForTest(account)

	accessToken, chatGPTAccountID, proxyURL, err := service.prepareUpstreamCall(context.Background(), account.ID)
	require.NoError(t, err)
	require.Equal(t, " access-token ", accessToken)
	require.Equal(t, "account-id", chatGPTAccountID)
	require.Empty(t, proxyURL)
}

func TestOpenAIQuotaServicePrepareUpstreamCall_OrganizationFallback(t *testing.T) {
	account := &Account{
		ID:       1,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token":    "access-token",
			"organization_id": " org-id ",
		},
	}
	service := newOpenAIQuotaServiceForTest(account)

	_, chatGPTAccountID, _, err := service.prepareUpstreamCall(context.Background(), account.ID)
	require.NoError(t, err)
	require.Equal(t, "org-id", chatGPTAccountID)
}

func TestOpenAIQuotaServicePrepareUpstreamCall_RejectsInvalidAccounts(t *testing.T) {
	tests := []struct {
		name    string
		account *Account
		reason  string
	}{
		{
			name: "non openai",
			account: &Account{
				ID:       1,
				Platform: PlatformGemini,
				Type:     AccountTypeOAuth,
			},
			reason: "OPENAI_QUOTA_INVALID_PLATFORM",
		},
		{
			name: "api key",
			account: &Account{
				ID:       1,
				Platform: PlatformOpenAI,
				Type:     AccountTypeAPIKey,
			},
			reason: "OPENAI_QUOTA_INVALID_TYPE",
		},
		{
			name: "missing chatgpt account id",
			account: &Account{
				ID:       1,
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
				Credentials: map[string]any{
					"access_token": "access-token",
				},
			},
			reason: "OPENAI_QUOTA_MISSING_ACCOUNT_ID",
		},
		{
			name: "missing access token",
			account: &Account{
				ID:       1,
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
				Credentials: map[string]any{
					"chatgpt_account_id": "account-id",
				},
			},
			reason: "OPENAI_QUOTA_TOKEN_UNAVAILABLE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := newOpenAIQuotaServiceForTest(tt.account)

			_, _, _, err := service.prepareUpstreamCall(context.Background(), tt.account.ID)
			require.Error(t, err)
			require.Equal(t, tt.reason, infraerrors.Reason(err))
		})
	}
}

func TestBuildCodexCommonHeaders(t *testing.T) {
	headers := buildCodexCommonHeaders("token", "account-id")

	require.Equal(t, "Bearer token", headers["authorization"])
	require.Equal(t, "account-id", headers["chatgpt-account-id"])
	require.Equal(t, "zh-CN", headers["oai-language"])
	require.Equal(t, "Codex Desktop", headers["originator"])
	require.Equal(t, "application/json", headers["accept"])
}

func TestGenerateRedeemRequestID(t *testing.T) {
	id, err := generateRedeemRequestID()
	require.NoError(t, err)
	require.Regexp(t, regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`), id)
}

func TestMapOpenAIQuotaUpstreamStatus(t *testing.T) {
	require.Equal(t, http.StatusUnauthorized, mapOpenAIQuotaUpstreamStatus(http.StatusUnauthorized))
	require.Equal(t, http.StatusForbidden, mapOpenAIQuotaUpstreamStatus(http.StatusForbidden))
	require.Equal(t, http.StatusTooManyRequests, mapOpenAIQuotaUpstreamStatus(http.StatusTooManyRequests))
	require.Equal(t, http.StatusBadGateway, mapOpenAIQuotaUpstreamStatus(http.StatusBadRequest))
	require.Equal(t, http.StatusBadGateway, mapOpenAIQuotaUpstreamStatus(http.StatusInternalServerError))
}

func TestOpenAIQuotaServiceQueryResetCredits(t *testing.T) {
	account := &Account{
		ID:       1,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token":       "access-token",
			"chatgpt_account_id": "account-id",
		},
	}
	service := newOpenAIQuotaServiceForTest(account)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/backend-api/wham/rate-limit-reset-credits" {
			t.Errorf("path = %q, want /backend-api/wham/rate-limit-reset-credits", r.URL.Path)
		}
		assertRequestHeader(t, r, "Authorization", "Bearer access-token")
		assertRequestHeader(t, r, "chatgpt-account-id", "account-id")
		assertRequestHeader(t, r, "originator", openaiQuotaCodexOriginator)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"available_count":1,"total_earned_count":2,"credits":[{"id":"credit_1","status":"available","title":"Invite reset"}]}`))
	}))
	defer server.Close()
	restore := overrideOpenAIQuotaURLsForTest(t, server.URL)
	defer restore()

	credits, err := service.QueryResetCredits(context.Background(), account.ID)
	require.NoError(t, err)
	require.Equal(t, 1, credits.AvailableCount)
	require.Equal(t, 2, credits.TotalEarnedCount)
	require.Len(t, credits.Credits, 1)
	require.Equal(t, "credit_1", credits.Credits[0].ID)
	require.NotZero(t, credits.FetchedAt)
}

func assertRequestHeader(t *testing.T, r *http.Request, key, want string) {
	t.Helper()
	if got := r.Header.Get(key); got != want {
		t.Errorf("header %s = %q, want %q", key, got, want)
	}
}

func TestOpenAIQuotaServiceResetCreditBlocksWhenNoCredits(t *testing.T) {
	account := &Account{
		ID:       1,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token":       "access-token",
			"chatgpt_account_id": "account-id",
		},
	}
	service := newOpenAIQuotaServiceForTest(account)

	var consumeCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/backend-api/wham/rate-limit-reset-credits":
			_, _ = w.Write([]byte(`{"available_count":0,"total_earned_count":0,"credits":[]}`))
		case "/backend-api/wham/rate-limit-reset-credits/consume":
			consumeCalls.Add(1)
			_, _ = w.Write([]byte(`{"code":"ok","windows_reset":1}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	restore := overrideOpenAIQuotaURLsForTest(t, server.URL)
	defer restore()

	_, err := service.ResetCredit(context.Background(), account.ID)
	require.Error(t, err)
	require.Equal(t, "OPENAI_QUOTA_NO_RESET_CREDITS", infraerrors.Reason(err))
	require.Equal(t, int32(0), consumeCalls.Load())
}

func newOpenAIQuotaServiceForTest(account *Account) *OpenAIQuotaService {
	accountRepo := &mockAccountRepoForGemini{
		accountsByID: map[int64]*Account{account.ID: account},
	}
	return NewOpenAIQuotaService(
		accountRepo,
		nil,
		NewOpenAITokenProvider(accountRepo, nil, nil),
		func(string) (*req.Client, error) { return req.C(), nil },
	)
}

func overrideOpenAIQuotaURLsForTest(t *testing.T, baseURL string) func() {
	t.Helper()
	oldUsageURL := chatGPTUsageURL
	oldResetCreditsURL := chatGPTRateLimitResetCreditsURL
	oldResetURL := chatGPTRateLimitResetURL
	chatGPTUsageURL = baseURL + "/backend-api/wham/usage"
	chatGPTRateLimitResetCreditsURL = baseURL + "/backend-api/wham/rate-limit-reset-credits"
	chatGPTRateLimitResetURL = baseURL + "/backend-api/wham/rate-limit-reset-credits/consume"
	return func() {
		chatGPTUsageURL = oldUsageURL
		chatGPTRateLimitResetCreditsURL = oldResetCreditsURL
		chatGPTRateLimitResetURL = oldResetURL
	}
}

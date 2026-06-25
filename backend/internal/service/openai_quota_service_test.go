//go:build unit

package service

import (
	"context"
	"net/http"
	"regexp"
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

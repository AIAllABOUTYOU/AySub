package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type grokOAuthSyncClientStub struct {
	refreshResponse *xai.TokenResponse
}

func (s *grokOAuthSyncClientStub) ExchangeCode(context.Context, string, string, string, string, string) (*xai.TokenResponse, error) {
	return &xai.TokenResponse{}, nil
}

func (s *grokOAuthSyncClientStub) RefreshToken(context.Context, string, string, string) (*xai.TokenResponse, error) {
	return s.refreshResponse, nil
}

func TestGrokAccountBaseURLRoutesOAuthToCLIProxy(t *testing.T) {
	tests := []struct {
		name        string
		accountType string
		baseURL     string
		want        string
	}{
		{name: "oauth default", accountType: AccountTypeOAuth, want: xai.DefaultCLIBaseURL},
		{name: "oauth legacy official base", accountType: AccountTypeOAuth, baseURL: xai.DefaultBaseURL, want: xai.DefaultCLIBaseURL},
		{name: "oauth explicit cli base", accountType: AccountTypeOAuth, baseURL: xai.DefaultCLIBaseURL, want: xai.DefaultCLIBaseURL},
		{name: "api key default", accountType: AccountTypeAPIKey, want: xai.DefaultBaseURL},
		{name: "api key custom", accountType: AccountTypeAPIKey, baseURL: "https://xai.example.com/v1", want: "https://xai.example.com/v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := &Account{Platform: PlatformXAI, Type: tt.accountType, Credentials: map[string]any{}}
			if tt.baseURL != "" {
				account.Credentials["base_url"] = tt.baseURL
			}
			require.Equal(t, tt.want, account.GetGrokBaseURL())
		})
	}
}

func TestGrokOfficialMediaCapabilityExcludesCookieAccounts(t *testing.T) {
	for _, accountType := range []string{AccountTypeOAuth, AccountTypeAPIKey} {
		account := &Account{Platform: PlatformXAI, Type: accountType}
		require.True(t, account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityGrokMedia))
	}

	cookie := &Account{Platform: PlatformXAI, Type: AccountTypeCookie}
	require.False(t, cookie.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityGrokMedia))
	require.True(t, cookie.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityVideos))
}

func TestGrokDefaultModelMappingDoesNotOverrideCookieArchitecture(t *testing.T) {
	oauth := &Account{Platform: PlatformXAI, Type: AccountTypeOAuth, Credentials: map[string]any{}}
	require.Equal(t, "grok-4.5", oauth.GetMappedModel("grok"))
	require.Equal(t, "grok-composer-2.5-fast", oauth.GetMappedModel("grok-composer"))

	cookie := &Account{Platform: PlatformXAI, Type: AccountTypeCookie, Credentials: map[string]any{}}
	require.Empty(t, cookie.GetModelMapping())
	require.Equal(t, "grok", cookie.GetMappedModel("grok"))
}

func TestGrokOAuthRefreshPreservesRefreshTokenAndCLIBaseURL(t *testing.T) {
	svc := NewGrokOAuthService(nil, &grokOAuthSyncClientStub{refreshResponse: &xai.TokenResponse{
		AccessToken: "new-access-token",
		TokenType:   "Bearer",
		ExpiresIn:   3600,
	}})
	defer svc.Stop()

	info, err := svc.RefreshToken(context.Background(), "original-refresh-token", "", "client-id")
	require.NoError(t, err)
	require.Equal(t, "original-refresh-token", info.RefreshToken)
	credentials := svc.BuildAccountCredentials(info)
	require.Equal(t, "original-refresh-token", credentials["refresh_token"])
	require.Equal(t, xai.DefaultCLIBaseURL, credentials["base_url"])
}

func TestGrokAccessTokenRejectsExpiredOAuthWithoutProviderRefresh(t *testing.T) {
	account := &Account{
		Platform: PlatformXAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "expired-token",
			"expires_at":   time.Now().Add(-time.Minute).UTC().Format(time.RFC3339),
		},
	}
	token, err := (&OpenAIGatewayService{}).getGrokAccessToken(context.Background(), account)
	require.Error(t, err)
	require.Empty(t, token)
	require.Contains(t, err.Error(), "token provider refresh is required")
}

func TestGrokResponsesSanitizesComposerReasoningParameters(t *testing.T) {
	body, err := patchGrokResponsesBody([]byte(`{
		"model":"grok-composer",
		"input":"hi",
		"reasoning":{"effort":"high","summary":"auto"},
		"reasoning_effort":"high",
		"prompt_cache_retention":"24h"
	}`), "grok-composer-2.5-fast")
	require.NoError(t, err)
	require.Equal(t, "grok-composer-2.5-fast", gjson.GetBytes(body, "model").String())
	require.False(t, gjson.GetBytes(body, "reasoning.effort").Exists())
	require.False(t, gjson.GetBytes(body, "reasoning_effort").Exists())
	require.False(t, gjson.GetBytes(body, "prompt_cache_retention").Exists())
}

func TestGrokPromptCacheIdentityIsTenantAndModelIsolated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	newContext := func(apiKeyID int64) *gin.Context {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
		c.Set("api_key", &APIKey{ID: apiKeyID, Group: &Group{Platform: PlatformXAI}})
		return c
	}
	body := []byte(`{"model":"grok","input":"same prompt"}`)
	base := resolveGrokCacheIdentity(newContext(101), body, "session", "grok-4.5")
	require.NotEmpty(t, base)
	require.Equal(t, base, resolveGrokCacheIdentity(newContext(101), body, "session", "grok-4.5"))
	require.NotEqual(t, base, resolveGrokCacheIdentity(newContext(102), body, "session", "grok-4.5"))
	require.NotEqual(t, base, resolveGrokCacheIdentity(newContext(101), body, "session", "grok-4.3"))
}

func TestGrokChatResponsesBridgeEligibilityIsConservative(t *testing.T) {
	eligible, reason := grokChatResponsesBridgeEligibility([]byte(`{"model":"grok","messages":[{"role":"user","content":"hi"}],"stream":true}`))
	require.True(t, eligible)
	require.Empty(t, reason)

	eligible, reason = grokChatResponsesBridgeEligibility([]byte(`{"model":"grok","messages":[{"role":"user","content":"hi"}],"tools":[{"type":"function","function":{"name":"lookup"}}]}`))
	require.False(t, eligible)
	require.Equal(t, "unsupported_tools", reason)
}

func TestGrokRateLimitUsesLatestExhaustedWindowReset(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	requestsRemaining := int64(0)
	tokensRemaining := int64(0)
	requestReset := now.Add(time.Minute).Unix()
	tokenReset := now.Add(3 * time.Minute).Unix()
	snapshot := &xai.QuotaSnapshot{
		StatusCode: http.StatusTooManyRequests,
		UpdatedAt:  now.Format(time.RFC3339),
		Requests:   &xai.QuotaWindow{Remaining: &requestsRemaining, ResetUnix: &requestReset},
		Tokens:     &xai.QuotaWindow{Remaining: &tokensRemaining, ResetUnix: &tokenReset},
	}
	resetAt, limited := grokRateLimitResetAt(snapshot, now)
	require.True(t, limited)
	require.Equal(t, time.Unix(tokenReset, 0), resetAt)
}

func TestBuildGrokUpstreamModelsRequestForAPIKey(t *testing.T) {
	svc := &AccountTestService{cfg: upstreamModelSyncTestConfig()}
	req, err := svc.buildGrokUpstreamModelsRequest(context.Background(), &Account{
		Platform: PlatformXAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "xai-key",
			"base_url": "https://api.x.ai/v1",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "https://api.x.ai/v1/models", req.URL.String())
	require.Equal(t, "Bearer xai-key", req.Header.Get("Authorization"))
}

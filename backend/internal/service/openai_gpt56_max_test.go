package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestNormalizeOpenAIReasoningEffortForGPT56(t *testing.T) {
	tests := []struct {
		name, raw, model, want string
	}{
		{name: "sol", raw: "max", model: "gpt-5.6-sol", want: "max"},
		{name: "terra provider prefix", raw: "max", model: "openai/gpt-5.6-terra", want: "max"},
		{name: "luna version suffix", raw: "max", model: "gpt-5.6-luna-2026-07-09", want: "max"},
		{name: "other model", raw: "max", model: "deepseek-v4-pro", want: "xhigh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, normalizeOpenAIReasoningEffortForModel(tt.raw, tt.model))
		})
	}
}

func TestNormalizeOpenAICodexCompactReasoningEffortDowngradesMax(t *testing.T) {
	body := []byte(`{"model":"gpt-5.6-sol","input":"compact me","reasoning":{"effort":"max","summary":"auto"}}`)

	normalized, changed, err := normalizeOpenAICodexCompactReasoningEffort(body, "gpt-5.6-sol")

	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, "xhigh", gjson.GetBytes(normalized, "reasoning.effort").String())
	require.Equal(t, "auto", gjson.GetBytes(normalized, "reasoning.summary").String())
}

func TestOpenAIGatewayServiceForwardPreservesMappedGPT56MaxEffort(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"usage":{"input_tokens":1,"output_tokens":2}}`)),
	}}
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream}
	account := &Account{
		ID: 9, Name: "openai-apikey-mapped", Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Concurrency: 1,
		Credentials: map[string]any{
			"api_key": "sk-test", "base_url": "https://example.com",
			"model_mapping": map[string]any{"sol": "gpt-5.6-sol"},
		},
		Extra: map[string]any{"use_responses_api": true},
	}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses", nil)
	SetOpenAIClientTransport(c, OpenAIClientTransportHTTP)

	result, err := svc.Forward(context.Background(), c, account, []byte(`{"model":"sol","stream":false,"reasoning":{"effort":"max"},"input":"hello"}`))

	require.NoError(t, err)
	require.Equal(t, "gpt-5.6-sol", gjson.GetBytes(upstream.lastBody, "model").String())
	require.Equal(t, "max", gjson.GetBytes(upstream.lastBody, "reasoning.effort").String())
	require.NotNil(t, result.ReasoningEffort)
	require.Equal(t, "max", *result.ReasoningEffort)
}

func TestOpenAIGatewayServiceForwardOAuthCompactDowngradesMaxEffort(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"usage":{"input_tokens":1,"output_tokens":2}}`)),
	}}
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream}
	account := &Account{
		ID: 8, Name: "openai-oauth", Platform: PlatformOpenAI, Type: AccountTypeOAuth, Concurrency: 1,
		Credentials: map[string]any{"access_token": "oauth-token", "chatgpt_account_id": "chatgpt-acc"},
		Status:      StatusActive, Schedulable: true,
	}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses/compact", nil)
	SetOpenAIClientTransport(c, OpenAIClientTransportHTTP)

	result, err := svc.Forward(context.Background(), c, account, []byte(`{"model":"gpt-5.6-sol","instructions":"compact-test","input":"hello","reasoning":{"effort":"max"}}`))

	require.NoError(t, err)
	require.Equal(t, chatgptCodexURL+"/compact", upstream.lastReq.URL.String())
	require.Equal(t, "xhigh", gjson.GetBytes(upstream.lastBody, "reasoning.effort").String())
	require.NotNil(t, result.ReasoningEffort)
	require.Equal(t, "xhigh", *result.ReasoningEffort)
}

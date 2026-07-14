package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestForwardXAIAPIKeyUsesGrokResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"grok","input":"hello","stream":false}`)
	c, recorder := newGrokDispatchContext(http.MethodPost, "/v1/responses", body, 6101)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"rid_grok_apikey"}},
		Body: io.NopCloser(strings.NewReader(
			`{"id":"resp_grok_apikey","object":"response","model":"grok-4.5","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ok"}],"status":"completed"}],"usage":{"input_tokens":3,"output_tokens":2,"total_tokens":5}}`,
		)),
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream}
	account := newGrokDispatchAccount(AccountTypeAPIKey)
	account.Credentials["api_key"] = "xai-api-key"
	account.Credentials["base_url"] = xai.DefaultBaseURL

	result, err := svc.Forward(context.Background(), c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, xai.DefaultBaseURL+"/responses", upstream.lastReq.URL.String())
	require.Equal(t, "Bearer xai-api-key", upstream.lastReq.Header.Get("Authorization"))
	require.Equal(t, grokUpstreamUserAgent, upstream.lastReq.Header.Get("User-Agent"))
	require.Equal(t, "grok-4.5", gjson.GetBytes(upstream.lastBody, "model").String())
	require.True(t, gjson.GetBytes(upstream.lastBody, "input").Exists())
	require.False(t, gjson.GetBytes(upstream.lastBody, "messages").Exists())
	require.Equal(t, "resp_grok_apikey", gjson.Get(recorder.Body.String(), "id").String())
}

func TestForwardAsChatCompletionsXAIOAuthCacheableUsesResponsesBridge(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"grok","messages":[{"role":"user","content":"hello"}],"stream":false}`)
	c, recorder := newGrokDispatchContext(http.MethodPost, "/v1/chat/completions", body, 6102)
	upstream := &httpUpstreamRecorder{resp: grokDispatchSSEResponse("resp_grok_chat", "ok")}
	svc := &OpenAIGatewayService{httpUpstream: upstream}
	account := newGrokDispatchAccount(AccountTypeOAuth)

	result, err := svc.ForwardAsChatCompletions(context.Background(), c, account, body, "", "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, xai.DefaultCLIBaseURL+"/responses", upstream.lastReq.URL.String())
	require.Equal(t, "Bearer xai-access-token", upstream.lastReq.Header.Get("Authorization"))
	require.Equal(t, grokUpstreamUserAgent, upstream.lastReq.Header.Get("User-Agent"))
	require.Empty(t, upstream.lastReq.Header.Get("originator"))
	require.NotEmpty(t, gjson.GetBytes(upstream.lastBody, "prompt_cache_key").String())
	require.Equal(t, "ok", gjson.Get(recorder.Body.String(), "choices.0.message.content").String())
}

func TestForwardAsAnthropicXAIOAuthUsesResponsesAdapter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"grok","max_tokens":32,"stream":false,"messages":[{"role":"user","content":"hello"}]}`)
	c, recorder := newGrokDispatchContext(http.MethodPost, "/v1/messages", body, 6103)
	c.Request.Header.Set("originator", "opencode")
	c.Request.Header.Set("OpenAI-Beta", "grok-experimental")
	upstream := &httpUpstreamRecorder{resp: grokDispatchSSEResponse("resp_grok_messages", "ok")}
	svc := &OpenAIGatewayService{httpUpstream: upstream}
	account := newGrokDispatchAccount(AccountTypeOAuth)

	result, err := svc.ForwardAsAnthropic(context.Background(), c, account, body, "", "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, xai.DefaultCLIBaseURL+"/responses", upstream.lastReq.URL.String())
	require.Equal(t, "Bearer xai-access-token", upstream.lastReq.Header.Get("Authorization"))
	require.Equal(t, grokUpstreamUserAgent, upstream.lastReq.Header.Get("User-Agent"))
	require.Equal(t, "grok-experimental", upstream.lastReq.Header.Get("OpenAI-Beta"))
	require.Empty(t, upstream.lastReq.Header.Get("originator"))
	require.Empty(t, upstream.lastReq.Header.Get("session_id"))
	require.Equal(t, "grok-4.5", gjson.GetBytes(upstream.lastBody, "model").String())
	require.True(t, gjson.GetBytes(upstream.lastBody, "input").IsArray())
	require.NotEmpty(t, gjson.GetBytes(upstream.lastBody, "prompt_cache_key").String())
	require.Equal(t, "message", gjson.Get(recorder.Body.String(), "type").String())
	require.Equal(t, "ok", gjson.Get(recorder.Body.String(), "content.0.text").String())
}

func newGrokDispatchContext(method, path string, body []byte, apiKeyID int64) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(method, path, bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("api_key", &APIKey{ID: apiKeyID, Group: &Group{Platform: PlatformXAI}})
	return c, recorder
}

func newGrokDispatchAccount(accountType string) *Account {
	return &Account{
		ID:          6100,
		Name:        "xai-dispatch",
		Platform:    PlatformXAI,
		Type:        accountType,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "xai-access-token",
			"expires_at":   time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
		},
	}
}

func grokDispatchSSEResponse(responseID, text string) *http.Response {
	body := strings.Join([]string{
		`event: response.completed`,
		`data: {"type":"response.completed","response":{"id":"` + responseID + `","object":"response","model":"grok-4.5","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"` + text + `"}],"status":"completed"}],"usage":{"input_tokens":3,"output_tokens":2,"total_tokens":5}}}`,
		"",
	}, "\n")
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}, "x-request-id": []string{"rid_grok_sse"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

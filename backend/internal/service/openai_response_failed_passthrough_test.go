package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestOpenAIStreamFailedEventSemanticStatus(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    int
	}{
		{name: "context", payload: `{"type":"response.failed","response":{"error":{"type":"invalid_request_error","code":"context_length_exceeded"}}}`, want: http.StatusBadRequest},
		{name: "rate limit", payload: `{"response":{"error":{"type":"rate_limit_error"}}}`, want: http.StatusTooManyRequests},
		{name: "auth", payload: `{"response":{"error":{"code":"invalid_api_key"}}}`, want: http.StatusUnauthorized},
		{name: "permission", payload: `{"response":{"error":{"type":"permission_error"}}}`, want: http.StatusForbidden},
		{name: "overloaded", payload: `{"response":{"error":{"code":"server_is_overloaded"}}}`, want: http.StatusServiceUnavailable},
		{name: "unknown", payload: `{"response":{"error":{"type":"server_error"}}}`, want: http.StatusBadGateway},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, openAIStreamFailedEventSemanticStatus([]byte(tt.payload), ""))
		})
	}
}

func TestOpenAIStreamFailedEventPassthroughBody(t *testing.T) {
	payload := []byte(`{"type":"response.failed","response":{"error":{"type":"invalid_request_error","code":"context_length_exceeded","message":"too long"}}}`)
	body := openAIStreamFailedEventPassthroughBody(payload, "")
	require.Equal(t, "invalid_request_error", gjson.GetBytes(body, "error.type").String())
	require.Equal(t, "context_length_exceeded", gjson.GetBytes(body, "error.code").String())
	require.Equal(t, "too long", gjson.GetBytes(body, "error.message").String())
}

func TestApplyOpenAIStreamFailedErrorPassthroughRule(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	ruleSvc := &ErrorPassthroughService{}
	rule := newNonFailoverPassthroughRule(http.StatusBadRequest, "context_length_exceeded", http.StatusBadRequest, "上下文超限")
	ruleSvc.setLocalCache([]*model.ErrorPassthroughRule{rule})
	BindErrorPassthroughService(c, ruleSvc)

	payload := []byte(`{"type":"response.failed","response":{"error":{"type":"invalid_request_error","code":"context_length_exceeded","message":"too long"}}}`)
	status, errType, errMsg, matched := applyOpenAIStreamFailedErrorPassthroughRule(c, payload, "too long")
	require.True(t, matched)
	require.Equal(t, http.StatusBadRequest, status)
	require.NotEmpty(t, errType)
	require.Equal(t, "上下文超限", errMsg)
}

func TestSanitizeOpenAIResponseFailedEventForClient(t *testing.T) {
	payload := []byte(`{"type":"response.failed","response":{"id":"resp_1","instructions":"secret","output":[{"type":"message"}],"usage":{"input_tokens":10},"error":{"type":"server_error","message":"input exceeds context window"}}}`)
	updated, changed := sanitizeOpenAIResponseFailedEventForClient(payload, "response.failed", true)
	require.True(t, changed)
	require.False(t, gjson.GetBytes(updated, "response.instructions").Exists())
	require.False(t, gjson.GetBytes(updated, "response.output").Exists())
	require.False(t, gjson.GetBytes(updated, "response.usage").Exists())
	require.Equal(t, "invalid_request_error", gjson.GetBytes(updated, "response.error.type").String())
	require.Equal(t, "context_length_exceeded", gjson.GetBytes(updated, "response.error.code").String())
}

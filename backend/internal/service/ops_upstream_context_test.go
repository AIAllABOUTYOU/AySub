package service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSafeUpstreamURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"strips query", "https://api.anthropic.com/v1/messages?beta=true", "https://api.anthropic.com/v1/messages"},
		{"strips fragment", "https://api.openai.com/v1/responses#frag", "https://api.openai.com/v1/responses"},
		{"strips both", "https://host/path?token=secret#x", "https://host/path"},
		{"no query or fragment", "https://host/path", "https://host/path"},
		{"empty string", "", ""},
		{"whitespace only", "  ", ""},
		{"query before fragment", "https://h/p?a=1#f", "https://h/p"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, safeUpstreamURL(tt.input))
		})
	}
}

func TestNewUpstreamRequestFailoverErrorDoesNotWriteClientResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	upstreamReq, err := http.NewRequest(http.MethodPost, "https://upstream.example/v1/responses?token=secret", nil)
	require.NoError(t, err)

	account := &Account{
		ID:       123,
		Name:     "openai-pool-1",
		Platform: PlatformOpenAI,
	}

	failoverErr := newUpstreamRequestFailoverError(
		c,
		account,
		upstreamReq,
		errors.New("dial tcp 10.0.0.1:443: connect: connection refused"),
		true,
	)

	require.NotNil(t, failoverErr)
	require.Equal(t, http.StatusBadGateway, failoverErr.StatusCode)
	require.Contains(t, string(failoverErr.ResponseBody), "Upstream request failed")
	require.False(t, c.Writer.Written(), "request-stage failover must not start the client response")
	require.Empty(t, recorder.Body.String())

	rawEvents, ok := c.Get(OpsUpstreamErrorsKey)
	require.True(t, ok)
	events, ok := rawEvents.([]*OpsUpstreamErrorEvent)
	require.True(t, ok)
	require.Len(t, events, 1)
	require.Equal(t, "request_error", events[0].Kind)
	require.Equal(t, int64(123), events[0].AccountID)
	require.Equal(t, PlatformOpenAI, events[0].Platform)
	require.True(t, events[0].Passthrough)
	require.Equal(t, "https://upstream.example/v1/responses", events[0].UpstreamURL)
	require.Contains(t, events[0].Message, "connection refused")
}

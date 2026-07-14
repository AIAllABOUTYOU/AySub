package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newGatewayRoutesTestRouter() *gin.Engine {
	return newGatewayRoutesTestRouterForPlatform(service.PlatformOpenAI)
}

func newGatewayRoutesTestRouterForPlatform(platform string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	RegisterGatewayRoutes(
		router,
		&handler.Handlers{
			Gateway:       &handler.GatewayHandler{},
			OpenAIGateway: &handler.OpenAIGatewayHandler{},
		},
		servermiddleware.APIKeyAuthMiddleware(func(c *gin.Context) {
			groupID := int64(1)
			c.Set(string(servermiddleware.ContextKeyAPIKey), &service.APIKey{
				GroupID: &groupID,
				Group:   &service.Group{Platform: platform},
			})
			c.Next()
		}),
		nil,
		nil,
		nil,
		nil,
		&config.Config{},
	)

	return router
}

func TestGatewayRoutesOpenAIResponsesCompactPathIsRegistered(t *testing.T) {
	router := newGatewayRoutesTestRouter()

	for _, path := range []string{
		"/v1/responses/compact",
		"/responses/compact",
		"/backend-api/codex/responses",
		"/backend-api/codex/responses/compact",
	} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"model":"gpt-5"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.NotEqual(t, http.StatusNotFound, w.Code, "path=%s should hit OpenAI responses handler", path)
	}
}

func TestGatewayRoutesOpenAIAlphaSearchPathsAreRegistered(t *testing.T) {
	router := newGatewayRoutesTestRouter()

	for _, path := range []string{
		"/v1/alpha/search",
		"/alpha/search",
		"/backend-api/codex/alpha/search",
	} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"model":"gpt-5.6-sol"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.NotEqual(t, http.StatusNotFound, w.Code, "path=%s should hit alpha/search handler", path)
	}
}

func TestGatewayRoutesOpenAIImagesPathsAreRegistered(t *testing.T) {
	router := newGatewayRoutesTestRouter()

	for _, path := range []string{
		"/v1/images/generations",
		"/v1/images/edits",
		"/images/generations",
		"/images/edits",
	} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"model":"gpt-image-2","prompt":"draw a cat"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.NotEqual(t, http.StatusNotFound, w.Code, "path=%s should hit OpenAI images handler", path)
	}
}

func TestGatewayRoutesOpenAIAudioPathsAreRegistered(t *testing.T) {
	router := newGatewayRoutesTestRouter()

	for _, path := range []string{
		"/v1/audio/speech",
		"/v1/audio/transcriptions",
		"/v1/audio/translations",
		"/audio/speech",
		"/audio/transcriptions",
		"/audio/translations",
	} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"model":"gpt-4o-mini-tts","input":"hello"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.NotEqual(t, http.StatusNotFound, w.Code, "path=%s should hit OpenAI audio handler", path)
	}
}

func TestGatewayRoutesOpenAIVideosPathsAreRegistered(t *testing.T) {
	router := newGatewayRoutesTestRouterForPlatform(service.PlatformXAI)

	for _, tc := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/v1/videos", `{"model":"grok-imagine-video","prompt":"make a video"}`},
		{http.MethodPost, "/v1/videos/generations", `{"model":"grok-imagine-video","prompt":"make a video"}`},
		{http.MethodGet, "/v1/videos/video_test", ``},
		{http.MethodGet, "/v1/videos/video_test/content", ``},
		{http.MethodGet, "/v1/files/image?id=0123456789abcdef", ``},
		{http.MethodGet, "/v1/files/video?id=0123456789abcdef", ``},
		{http.MethodPost, "/videos", `{"model":"grok-imagine-video","prompt":"make a video"}`},
		{http.MethodPost, "/videos/generations", `{"model":"grok-imagine-video","prompt":"make a video"}`},
		{http.MethodGet, "/videos/video_test", ``},
		{http.MethodGet, "/videos/video_test/content", ``},
		{http.MethodGet, "/files/image?id=0123456789abcdef", ``},
		{http.MethodGet, "/files/video?id=0123456789abcdef", ``},
	} {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.NotEqual(t, http.StatusNotFound, w.Code, "path=%s should hit OpenAI videos handler", tc.path)
	}
}

func TestGatewayRoutesGrokLiveKitTokenPathsAreRegistered(t *testing.T) {
	router := newGatewayRoutesTestRouterForPlatform(service.PlatformXAI)

	for _, path := range []string{
		"/v1/livekit/tokens",
		"/livekit/tokens",
	} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"voice":"ara"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.NotEqual(t, http.StatusNotFound, w.Code, "path=%s should hit OpenAI LiveKit token handler", path)
	}
}

func TestGatewayRoutesGrokLiveKitRTCPathsAreRegistered(t *testing.T) {
	router := newGatewayRoutesTestRouterForPlatform(service.PlatformXAI)

	for _, path := range []string{
		"/v1/livekit/rtc?access_token=lk-test",
		"/livekit/rtc?access_token=lk-test",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.NotEqual(t, http.StatusNotFound, w.Code, "path=%s should hit OpenAI LiveKit RTC handler", path)
	}
}

func TestGatewayRoutesXAIUsesOpenAICompatibleHandlers(t *testing.T) {
	router := newGatewayRoutesTestRouterForPlatform(service.PlatformXAI)

	for _, path := range []string{
		"/v1/messages",
		"/v1/responses",
		"/v1/chat/completions",
		"/v1/embeddings",
		"/v1/videos",
		"/v1/livekit/tokens",
		"/responses",
		"/chat/completions",
		"/embeddings",
		"/videos",
		"/livekit/tokens",
	} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"model":"grok-4","input":"hello"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.NotEqual(t, http.StatusNotFound, w.Code, "path=%s should hit OpenAI-compatible handler for xAI", path)
	}
}

func TestGatewayRoutesXAIMessagesIgnoresDispatchSwitch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	RegisterGatewayRoutes(
		router,
		&handler.Handlers{
			Gateway:       &handler.GatewayHandler{},
			OpenAIGateway: &handler.OpenAIGatewayHandler{},
		},
		servermiddleware.APIKeyAuthMiddleware(func(c *gin.Context) {
			groupID := int64(1)
			c.Set(string(servermiddleware.ContextKeyAPIKey), &service.APIKey{
				GroupID: &groupID,
				Group: &service.Group{
					Platform:              service.PlatformXAI,
					AllowMessagesDispatch: false,
				},
			})
			c.Next()
		}),
		nil,
		nil,
		nil,
		nil,
		&config.Config{},
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"claude-sonnet-4-5","messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	require.NotEqual(t, http.StatusForbidden, w.Code)
	require.NotContains(t, w.Body.String(), "does not allow /v1/messages dispatch")
}

func TestGatewayRoutesXAIStillRejectsCountTokens(t *testing.T) {
	router := newGatewayRoutesTestRouterForPlatform(service.PlatformXAI)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages/count_tokens", strings.NewReader(`{"model":"grok-4","messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
	require.Contains(t, w.Body.String(), "Token counting is not supported for this platform")
}

func TestGatewayRoutesOpenAICountTokensPathIsRegistered(t *testing.T) {
	router := newGatewayRoutesTestRouterForPlatform(service.PlatformOpenAI)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages/count_tokens", strings.NewReader(`{"model":"claude-sonnet-4-5","messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	require.NotEqual(t, http.StatusNotFound, w.Code)
}

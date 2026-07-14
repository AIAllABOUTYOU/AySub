package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func init() { gin.SetMode(gin.TestMode) }

// ──────────────────────────────────────────────────────────
// NormalizeInboundEndpoint
// ──────────────────────────────────────────────────────────

func TestNormalizeInboundEndpoint(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		// Direct canonical paths.
		{"/v1/messages", EndpointMessages},
		{"/v1/chat/completions", EndpointChatCompletions},
		{"/v1/embeddings", EndpointEmbeddings},
		{"/v1/alpha/search", EndpointAlphaSearch},
		{"/alpha/search", EndpointAlphaSearch},
		{"/backend-api/codex/alpha/search", EndpointAlphaSearch},
		{"/v1/responses", EndpointResponses},
		{"/v1/responses/compact", EndpointResponsesCompact},
		{"/v1/responses/compact/detail", EndpointResponsesCompact},
		{"/v1/images/generations", EndpointImagesGenerations},
		{"/v1/images/edits", EndpointImagesEdits},
		{"/v1/videos", EndpointVideos},
		{"/v1/audio/speech", EndpointAudioSpeech},
		{"/v1/audio/transcriptions", EndpointAudioTranscriptions},
		{"/v1/audio/translations", EndpointAudioTranslations},
		{"/v1/livekit/tokens", EndpointLiveKitTokens},
		{"/v1/livekit/rtc", EndpointLiveKitRTC},
		{"/v1beta/models", EndpointGeminiModels},

		// Prefixed paths (antigravity, openai).
		{"/antigravity/v1/messages", EndpointMessages},
		{"/openai/v1/responses", EndpointResponses},
		{"/openai/v1/responses/compact", EndpointResponsesCompact},
		{"/openai/v1/responses/compact/detail", EndpointResponsesCompact},
		{"/openai/v1/images/generations", EndpointImagesGenerations},
		{"/openai/v1/images/edits", EndpointImagesEdits},
		{"/openai/v1/audio/speech", EndpointAudioSpeech},
		{"/openai/v1/audio/transcriptions", EndpointAudioTranscriptions},
		{"/openai/v1/audio/translations", EndpointAudioTranslations},
		{"/openai/v1/livekit/rtc", EndpointLiveKitRTC},
		{"/antigravity/v1beta/models/gemini:generateContent", EndpointGeminiModels},
		{"/responses", EndpointResponses},
		{"/responses/compact", EndpointResponsesCompact},
		{"/responses/compact/detail", EndpointResponsesCompact},
		{"/backend-api/codex/responses", EndpointResponses},
		{"/backend-api/codex/responses/compact", EndpointResponsesCompact},
		{"/backend-api/codex/responses/compact/detail", EndpointResponsesCompact},
		{"/foo/responses", "/foo/responses"},
		{"/foo/responses/compact", "/foo/responses/compact"},

		// Gin route patterns with wildcards.
		{"/v1beta/models/*modelAction", EndpointGeminiModels},
		{"/v1/responses/*subpath", EndpointResponses},

		// Unknown path is returned as-is.
		{"/v1/embeddings", "/v1/embeddings"},
		{"", ""},
		{"  /v1/messages  ", EndpointMessages},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			require.Equal(t, tt.want, NormalizeInboundEndpoint(tt.path))
		})
	}
}

// ──────────────────────────────────────────────────────────
// DeriveUpstreamEndpoint
// ──────────────────────────────────────────────────────────

func TestDeriveUpstreamEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		inbound  string
		rawPath  string
		platform string
		want     string
	}{
		// Anthropic.
		{"anthropic messages", EndpointMessages, "/v1/messages", service.PlatformAnthropic, EndpointMessages},

		// Gemini.
		{"gemini models", EndpointGeminiModels, "/v1beta/models/gemini:gen", service.PlatformGemini, EndpointGeminiModels},

		// OpenAI — always /v1/responses.
		{"openai responses root", EndpointResponses, "/v1/responses", service.PlatformOpenAI, EndpointResponses},
		{"openai responses compact", EndpointResponsesCompact, "/openai/v1/responses/compact", service.PlatformOpenAI, "/v1/responses/compact"},
		{"openai responses nested", EndpointResponsesCompact, "/openai/v1/responses/compact/detail", service.PlatformOpenAI, "/v1/responses/compact/detail"},
		{"openai bare responses compact", EndpointResponsesCompact, "/responses/compact", service.PlatformOpenAI, "/v1/responses/compact"},
		{"openai codex direct responses compact", EndpointResponsesCompact, "/backend-api/codex/responses/compact", service.PlatformOpenAI, "/v1/responses/compact"},
		{"openai compact inbound only", EndpointResponsesCompact, "/v1/messages", service.PlatformOpenAI, EndpointResponsesCompact},
		{"openai from messages", EndpointMessages, "/v1/messages", service.PlatformOpenAI, EndpointResponses},
		{"openai from completions", EndpointChatCompletions, "/v1/chat/completions", service.PlatformOpenAI, EndpointResponses},
		{"openai embeddings", EndpointEmbeddings, "/v1/embeddings", service.PlatformOpenAI, EndpointEmbeddings},
		{"openai alpha search", EndpointAlphaSearch, "/backend-api/codex/alpha/search", service.PlatformOpenAI, EndpointAlphaSearch},
		{"openai image generations", EndpointImagesGenerations, "/v1/images/generations", service.PlatformOpenAI, EndpointImagesGenerations},
		{"openai image edits", EndpointImagesEdits, "/openai/v1/images/edits", service.PlatformOpenAI, EndpointImagesEdits},
		{"openai audio speech", EndpointAudioSpeech, "/v1/audio/speech", service.PlatformOpenAI, EndpointAudioSpeech},
		{"openai audio transcriptions", EndpointAudioTranscriptions, "/v1/audio/transcriptions", service.PlatformOpenAI, EndpointAudioTranscriptions},
		{"openai audio translations", EndpointAudioTranslations, "/v1/audio/translations", service.PlatformOpenAI, EndpointAudioTranslations},
		{"xai from messages", EndpointMessages, "/v1/messages", service.PlatformXAI, EndpointChatCompletions},
		{"xai responses fallback", EndpointResponses, "/v1/responses", service.PlatformXAI, EndpointChatCompletions},
		{"xai chat completions", EndpointChatCompletions, "/v1/chat/completions", service.PlatformXAI, EndpointChatCompletions},
		{"xai embeddings", EndpointEmbeddings, "/v1/embeddings", service.PlatformXAI, EndpointEmbeddings},
		{"xai videos reverse", EndpointVideos, "/v1/videos", service.PlatformXAI, EndpointChatCompletions},
		{"xai livekit token reverse", EndpointLiveKitTokens, "/v1/livekit/tokens", service.PlatformXAI, EndpointChatCompletions},
		{"xai livekit rtc reverse", EndpointLiveKitRTC, "/v1/livekit/rtc", service.PlatformXAI, EndpointChatCompletions},

		// Antigravity — uses inbound to pick Claude vs Gemini upstream.
		{"antigravity claude", EndpointMessages, "/antigravity/v1/messages", service.PlatformAntigravity, EndpointMessages},
		{"antigravity gemini", EndpointGeminiModels, "/antigravity/v1beta/models", service.PlatformAntigravity, EndpointGeminiModels},

		// Unknown platform — passthrough.
		{"unknown platform", "/v1/embeddings", "/v1/embeddings", "unknown", "/v1/embeddings"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, DeriveUpstreamEndpoint(tt.inbound, tt.rawPath, tt.platform))
		})
	}
}

// ──────────────────────────────────────────────────────────
// responsesSubpathSuffix
// ──────────────────────────────────────────────────────────

func TestResponsesSubpathSuffix(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"/v1/responses", ""},
		{"/v1/responses/", ""},
		{"/v1/responses/compact", "/compact"},
		{"/openai/v1/responses/compact/detail", "/compact/detail"},
		{"/responses", ""},
		{"/responses/compact", "/compact"},
		{"/responses/compact/detail", "/compact/detail"},
		{"/backend-api/codex/responses", ""},
		{"/backend-api/codex/responses/compact", "/compact"},
		{"/backend-api/codex/responses/compact/detail", "/compact/detail"},
		{"/v1/messages", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			require.Equal(t, tt.want, responsesSubpathSuffix(tt.raw))
		})
	}
}

// ──────────────────────────────────────────────────────────
// InboundEndpointMiddleware + context helpers
// ──────────────────────────────────────────────────────────

func TestInboundEndpointMiddleware(t *testing.T) {
	router := gin.New()
	router.Use(InboundEndpointMiddleware())

	var captured string
	router.POST("/v1/messages", func(c *gin.Context) {
		captured = GetInboundEndpoint(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, EndpointMessages, captured)
}

func TestGetInboundEndpoint_FallbackWithoutMiddleware(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/antigravity/v1/messages", nil)

	// Middleware did not run — fallback to normalizing c.Request.URL.Path.
	got := GetInboundEndpoint(c)
	require.Equal(t, EndpointMessages, got)
}

func TestInboundEndpointMiddleware_WildcardRoutesUseRawPath(t *testing.T) {
	tests := []struct {
		name        string
		routePath   string
		requestPath string
		want        string
	}{
		{"v1 compact", "/v1/responses/*subpath", "/v1/responses/compact", EndpointResponsesCompact},
		{"bare compact", "/responses/*subpath", "/responses/compact", EndpointResponsesCompact},
		{"codex compact", "/backend-api/codex/responses/*subpath", "/backend-api/codex/responses/compact", EndpointResponsesCompact},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(InboundEndpointMiddleware())

			var captured string
			router.POST(tt.routePath, func(c *gin.Context) {
				captured = GetInboundEndpoint(c)
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodPost, tt.requestPath, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			require.Equal(t, tt.want, captured)
		})
	}
}

func TestGetInboundEndpoint_FallbackWildcardRouteWithoutMiddleware(t *testing.T) {
	router := gin.New()

	var captured string
	router.POST("/v1/responses/*subpath", func(c *gin.Context) {
		require.Equal(t, "/v1/responses/*subpath", c.FullPath())
		captured = GetInboundEndpoint(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, EndpointResponsesCompact, captured)
}

func TestGetUpstreamEndpoint_FullFlow(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses/compact", nil)

	// Simulate middleware.
	c.Set(ctxKeyInboundEndpoint, NormalizeInboundEndpoint(c.Request.URL.Path))

	got := GetUpstreamEndpoint(c, service.PlatformOpenAI)
	require.Equal(t, "/v1/responses/compact", got)
}

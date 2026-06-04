package routes

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterGatewayRoutes 注册 API 网关路由（Claude/OpenAI/Gemini 兼容）
func RegisterGatewayRoutes(
	r *gin.Engine,
	h *handler.Handlers,
	apiKeyAuth middleware.APIKeyAuthMiddleware,
	apiKeyService *service.APIKeyService,
	subscriptionService *service.SubscriptionService,
	opsService *service.OpsService,
	settingService *service.SettingService,
	cfg *config.Config,
	auditServices ...*service.SecurityAuditService,
) {
	auditService := firstGatewaySecurityAuditService(auditServices)
	bodyLimit := middleware.RequestBodyLimit(cfg.Gateway.MaxBodySize)
	clientRequestID := middleware.ClientRequestID()
	opsErrorLogger := handler.OpsErrorLoggerMiddleware(opsService)
	endpointNorm := handler.InboundEndpointMiddleware()

	// 未分组 Key 拦截中间件（按协议格式区分错误响应）
	requireGroupAnthropic := middleware.RequireGroupAssignment(settingService, middleware.AnthropicErrorWriter)
	requireGroupGoogle := middleware.RequireGroupAssignment(settingService, middleware.GoogleErrorWriter)

	openAICompatiblePlatform := func(c *gin.Context) string {
		platform := getGroupPlatform(c)
		if service.IsOpenAICompatiblePlatform(platform) {
			return platform
		}
		return ""
	}
	withOpenAICompatiblePlatform := func(c *gin.Context, platform string) {
		if c == nil || c.Request == nil || platform == "" {
			return
		}
		ctx := service.WithOpenAICompatiblePlatform(c.Request.Context(), platform)
		c.Request = c.Request.WithContext(ctx)
	}

	// API网关（Claude API兼容）
	gateway := r.Group("/v1")
	gateway.Use(bodyLimit)
	gateway.Use(clientRequestID)
	gateway.Use(opsErrorLogger)
	gateway.Use(endpointNorm)
	gateway.Use(gin.HandlerFunc(apiKeyAuth))
	gateway.Use(requireAPIKeyGatewayPermission(gatewayPermissionProtocolAuto, auditService))
	gateway.Use(requireGroupAnthropic)
	{
		// /v1/messages: auto-route based on group platform
		gateway.POST("/messages", func(c *gin.Context) {
			if platform := openAICompatiblePlatform(c); platform != "" {
				withOpenAICompatiblePlatform(c, platform)
				h.OpenAIGateway.Messages(c)
				return
			}
			h.Gateway.Messages(c)
		})
		// /v1/messages/count_tokens: OpenAI groups get 404
		gateway.POST("/messages/count_tokens", func(c *gin.Context) {
			if openAICompatiblePlatform(c) != "" {
				service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
				c.JSON(http.StatusNotFound, gin.H{
					"type": "error",
					"error": gin.H{
						"type":    "not_found_error",
						"message": "Token counting is not supported for this platform",
					},
				})
				return
			}
			h.Gateway.CountTokens(c)
		})
		gateway.GET("/models", h.Gateway.Models)
		gateway.GET("/usage", h.Gateway.Usage)
		// OpenAI Responses API: auto-route based on group platform
		gateway.POST("/responses", func(c *gin.Context) {
			if platform := openAICompatiblePlatform(c); platform != "" {
				withOpenAICompatiblePlatform(c, platform)
				h.OpenAIGateway.Responses(c)
				return
			}
			h.Gateway.Responses(c)
		})
		gateway.POST("/responses/*subpath", func(c *gin.Context) {
			if platform := openAICompatiblePlatform(c); platform != "" {
				withOpenAICompatiblePlatform(c, platform)
				h.OpenAIGateway.Responses(c)
				return
			}
			h.Gateway.Responses(c)
		})
		gateway.GET("/responses", h.OpenAIGateway.ResponsesWebSocket)
		// OpenAI Chat Completions API: auto-route based on group platform
		gateway.POST("/chat/completions", func(c *gin.Context) {
			if platform := openAICompatiblePlatform(c); platform != "" {
				withOpenAICompatiblePlatform(c, platform)
				h.OpenAIGateway.ChatCompletions(c)
				return
			}
			h.Gateway.ChatCompletions(c)
		})
		gateway.POST("/embeddings", func(c *gin.Context) {
			platform := openAICompatiblePlatform(c)
			if platform == "" {
				service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
				c.JSON(http.StatusNotFound, gin.H{
					"error": gin.H{
						"type":    "not_found_error",
						"message": "Embeddings API is not supported for this platform",
					},
				})
				return
			}
			withOpenAICompatiblePlatform(c, platform)
			h.OpenAIGateway.Embeddings(c)
		})
		gateway.POST("/images/generations", func(c *gin.Context) {
			platform := openAICompatiblePlatform(c)
			if platform == "" {
				service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
				c.JSON(http.StatusNotFound, gin.H{
					"error": gin.H{
						"type":    "not_found_error",
						"message": "Images API is not supported for this platform",
					},
				})
				return
			}
			withOpenAICompatiblePlatform(c, platform)
			h.OpenAIGateway.Images(c)
		})
		gateway.POST("/images/edits", func(c *gin.Context) {
			platform := openAICompatiblePlatform(c)
			if platform == "" {
				service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
				c.JSON(http.StatusNotFound, gin.H{
					"error": gin.H{
						"type":    "not_found_error",
						"message": "Images API is not supported for this platform",
					},
				})
				return
			}
			withOpenAICompatiblePlatform(c, platform)
			h.OpenAIGateway.Images(c)
		})
		gateway.POST("/audio/speech", func(c *gin.Context) {
			platform := openAICompatiblePlatform(c)
			if platform == "" {
				service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
				c.JSON(http.StatusNotFound, gin.H{
					"error": gin.H{
						"type":    "not_found_error",
						"message": "Audio API is not supported for this platform",
					},
				})
				return
			}
			withOpenAICompatiblePlatform(c, platform)
			h.OpenAIGateway.Audio(c)
		})
		gateway.POST("/audio/transcriptions", func(c *gin.Context) {
			platform := openAICompatiblePlatform(c)
			if platform == "" {
				service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
				c.JSON(http.StatusNotFound, gin.H{
					"error": gin.H{
						"type":    "not_found_error",
						"message": "Audio API is not supported for this platform",
					},
				})
				return
			}
			withOpenAICompatiblePlatform(c, platform)
			h.OpenAIGateway.Audio(c)
		})
		gateway.POST("/audio/translations", func(c *gin.Context) {
			platform := openAICompatiblePlatform(c)
			if platform == "" {
				service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
				c.JSON(http.StatusNotFound, gin.H{
					"error": gin.H{
						"type":    "not_found_error",
						"message": "Audio API is not supported for this platform",
					},
				})
				return
			}
			withOpenAICompatiblePlatform(c, platform)
			h.OpenAIGateway.Audio(c)
		})
		gateway.POST("/videos", func(c *gin.Context) {
			platform := openAICompatiblePlatform(c)
			if platform == "" {
				service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
				c.JSON(http.StatusNotFound, gin.H{
					"error": gin.H{
						"type":    "not_found_error",
						"message": "Videos API is not supported for this platform",
					},
				})
				return
			}
			withOpenAICompatiblePlatform(c, platform)
			h.OpenAIGateway.Videos(c)
		})
		gateway.GET("/videos/:video_id", h.OpenAIGateway.VideoJob)
		gateway.GET("/videos/:video_id/content", h.OpenAIGateway.VideoContent)
		gateway.GET("/files/image", h.OpenAIGateway.LocalImageFile)
		gateway.GET("/files/video", h.OpenAIGateway.LocalVideoFile)
		gateway.POST("/livekit/tokens", func(c *gin.Context) {
			platform := openAICompatiblePlatform(c)
			if platform == "" {
				service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
				c.JSON(http.StatusNotFound, gin.H{
					"error": gin.H{
						"type":    "not_found_error",
						"message": "LiveKit API is not supported for this platform",
					},
				})
				return
			}
			withOpenAICompatiblePlatform(c, platform)
			h.OpenAIGateway.LiveKitToken(c)
		})
		gateway.GET("/livekit/rtc", func(c *gin.Context) {
			platform := openAICompatiblePlatform(c)
			if platform == "" {
				service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
				c.JSON(http.StatusNotFound, gin.H{
					"error": gin.H{
						"type":    "not_found_error",
						"message": "LiveKit API is not supported for this platform",
					},
				})
				return
			}
			withOpenAICompatiblePlatform(c, platform)
			h.OpenAIGateway.LiveKitRTC(c)
		})
	}

	// Gemini 原生 API 兼容层（Gemini SDK/CLI 直连）
	gemini := r.Group("/v1beta")
	gemini.Use(bodyLimit)
	gemini.Use(clientRequestID)
	gemini.Use(opsErrorLogger)
	gemini.Use(endpointNorm)
	gemini.Use(middleware.APIKeyAuthWithSubscriptionGoogle(apiKeyService, subscriptionService, cfg, auditService))
	gemini.Use(requireAPIKeyGatewayPermission(gatewayPermissionProtocolGoogle, auditService))
	gemini.Use(requireGroupGoogle)
	{
		gemini.GET("/models", h.Gateway.GeminiV1BetaListModels)
		gemini.GET("/models/:model", h.Gateway.GeminiV1BetaGetModel)
		// Gin treats ":" as a param marker, but Gemini uses "{model}:{action}" in the same segment.
		gemini.POST("/models/*modelAction", h.Gateway.GeminiV1BetaModels)
	}

	// OpenAI Responses API（不带v1前缀的别名）— auto-route based on group platform
	responsesHandler := func(c *gin.Context) {
		if platform := openAICompatiblePlatform(c); platform != "" {
			withOpenAICompatiblePlatform(c, platform)
			h.OpenAIGateway.Responses(c)
			return
		}
		h.Gateway.Responses(c)
	}
	permissionAnthropic := requireAPIKeyGatewayPermission(gatewayPermissionProtocolAuto, auditService)
	permissionGoogle := requireAPIKeyGatewayPermission(gatewayPermissionProtocolGoogle, auditService)
	r.POST("/responses", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), permissionAnthropic, requireGroupAnthropic, responsesHandler)
	r.POST("/responses/*subpath", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), permissionAnthropic, requireGroupAnthropic, responsesHandler)
	r.GET("/responses", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), permissionAnthropic, requireGroupAnthropic, h.OpenAIGateway.ResponsesWebSocket)
	codexDirect := r.Group("/backend-api/codex")
	codexDirect.Use(bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), permissionAnthropic, requireGroupAnthropic)
	{
		codexDirect.POST("/responses", responsesHandler)
		codexDirect.POST("/responses/*subpath", responsesHandler)
		codexDirect.GET("/responses", h.OpenAIGateway.ResponsesWebSocket)
	}
	// OpenAI Chat Completions API（不带v1前缀的别名）— auto-route based on group platform
	r.POST("/chat/completions", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), permissionAnthropic, requireGroupAnthropic, func(c *gin.Context) {
		if platform := openAICompatiblePlatform(c); platform != "" {
			withOpenAICompatiblePlatform(c, platform)
			h.OpenAIGateway.ChatCompletions(c)
			return
		}
		h.Gateway.ChatCompletions(c)
	})
	r.POST("/embeddings", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), permissionAnthropic, requireGroupAnthropic, func(c *gin.Context) {
		platform := openAICompatiblePlatform(c)
		if platform == "" {
			service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"type":    "not_found_error",
					"message": "Embeddings API is not supported for this platform",
				},
			})
			return
		}
		withOpenAICompatiblePlatform(c, platform)
		h.OpenAIGateway.Embeddings(c)
	})
	r.POST("/images/generations", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), permissionAnthropic, requireGroupAnthropic, func(c *gin.Context) {
		platform := openAICompatiblePlatform(c)
		if platform == "" {
			service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"type":    "not_found_error",
					"message": "Images API is not supported for this platform",
				},
			})
			return
		}
		withOpenAICompatiblePlatform(c, platform)
		h.OpenAIGateway.Images(c)
	})
	r.POST("/images/edits", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), permissionAnthropic, requireGroupAnthropic, func(c *gin.Context) {
		platform := openAICompatiblePlatform(c)
		if platform == "" {
			service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"type":    "not_found_error",
					"message": "Images API is not supported for this platform",
				},
			})
			return
		}
		withOpenAICompatiblePlatform(c, platform)
		h.OpenAIGateway.Images(c)
	})
	r.POST("/audio/speech", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), permissionAnthropic, requireGroupAnthropic, func(c *gin.Context) {
		platform := openAICompatiblePlatform(c)
		if platform == "" {
			service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"type":    "not_found_error",
					"message": "Audio API is not supported for this platform",
				},
			})
			return
		}
		withOpenAICompatiblePlatform(c, platform)
		h.OpenAIGateway.Audio(c)
	})
	r.POST("/audio/transcriptions", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), permissionAnthropic, requireGroupAnthropic, func(c *gin.Context) {
		platform := openAICompatiblePlatform(c)
		if platform == "" {
			service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"type":    "not_found_error",
					"message": "Audio API is not supported for this platform",
				},
			})
			return
		}
		withOpenAICompatiblePlatform(c, platform)
		h.OpenAIGateway.Audio(c)
	})
	r.POST("/audio/translations", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), permissionAnthropic, requireGroupAnthropic, func(c *gin.Context) {
		platform := openAICompatiblePlatform(c)
		if platform == "" {
			service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"type":    "not_found_error",
					"message": "Audio API is not supported for this platform",
				},
			})
			return
		}
		withOpenAICompatiblePlatform(c, platform)
		h.OpenAIGateway.Audio(c)
	})
	r.POST("/videos", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), permissionAnthropic, requireGroupAnthropic, func(c *gin.Context) {
		platform := openAICompatiblePlatform(c)
		if platform == "" {
			service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"type":    "not_found_error",
					"message": "Videos API is not supported for this platform",
				},
			})
			return
		}
		withOpenAICompatiblePlatform(c, platform)
		h.OpenAIGateway.Videos(c)
	})
	r.GET("/videos/:video_id", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), permissionAnthropic, requireGroupAnthropic, h.OpenAIGateway.VideoJob)
	r.GET("/videos/:video_id/content", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), permissionAnthropic, requireGroupAnthropic, h.OpenAIGateway.VideoContent)
	r.GET("/files/image", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), permissionAnthropic, requireGroupAnthropic, h.OpenAIGateway.LocalImageFile)
	r.GET("/files/video", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), permissionAnthropic, requireGroupAnthropic, h.OpenAIGateway.LocalVideoFile)
	r.POST("/livekit/tokens", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), permissionAnthropic, requireGroupAnthropic, func(c *gin.Context) {
		platform := openAICompatiblePlatform(c)
		if platform == "" {
			service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"type":    "not_found_error",
					"message": "LiveKit API is not supported for this platform",
				},
			})
			return
		}
		withOpenAICompatiblePlatform(c, platform)
		h.OpenAIGateway.LiveKitToken(c)
	})
	r.GET("/livekit/rtc", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), permissionAnthropic, requireGroupAnthropic, func(c *gin.Context) {
		platform := openAICompatiblePlatform(c)
		if platform == "" {
			service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"type":    "not_found_error",
					"message": "LiveKit API is not supported for this platform",
				},
			})
			return
		}
		withOpenAICompatiblePlatform(c, platform)
		h.OpenAIGateway.LiveKitRTC(c)
	})

	// Antigravity 模型列表
	r.GET("/antigravity/models", gin.HandlerFunc(apiKeyAuth), permissionAnthropic, requireGroupAnthropic, h.Gateway.AntigravityModels)

	// Antigravity 专用路由（仅使用 antigravity 账户，不混合调度）
	antigravityV1 := r.Group("/antigravity/v1")
	antigravityV1.Use(bodyLimit)
	antigravityV1.Use(clientRequestID)
	antigravityV1.Use(opsErrorLogger)
	antigravityV1.Use(endpointNorm)
	antigravityV1.Use(middleware.ForcePlatform(service.PlatformAntigravity))
	antigravityV1.Use(gin.HandlerFunc(apiKeyAuth))
	antigravityV1.Use(permissionAnthropic)
	antigravityV1.Use(requireGroupAnthropic)
	{
		antigravityV1.POST("/messages", h.Gateway.Messages)
		antigravityV1.POST("/messages/count_tokens", h.Gateway.CountTokens)
		antigravityV1.GET("/models", h.Gateway.AntigravityModels)
		antigravityV1.GET("/usage", h.Gateway.Usage)
	}

	antigravityV1Beta := r.Group("/antigravity/v1beta")
	antigravityV1Beta.Use(bodyLimit)
	antigravityV1Beta.Use(clientRequestID)
	antigravityV1Beta.Use(opsErrorLogger)
	antigravityV1Beta.Use(endpointNorm)
	antigravityV1Beta.Use(middleware.ForcePlatform(service.PlatformAntigravity))
	antigravityV1Beta.Use(middleware.APIKeyAuthWithSubscriptionGoogle(apiKeyService, subscriptionService, cfg, auditService))
	antigravityV1Beta.Use(permissionGoogle)
	antigravityV1Beta.Use(requireGroupGoogle)
	{
		antigravityV1Beta.GET("/models", h.Gateway.GeminiV1BetaListModels)
		antigravityV1Beta.GET("/models/:model", h.Gateway.GeminiV1BetaGetModel)
		antigravityV1Beta.POST("/models/*modelAction", h.Gateway.GeminiV1BetaModels)
	}

}

// getGroupPlatform extracts the group platform from the API Key stored in context.
func getGroupPlatform(c *gin.Context) string {
	apiKey, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || apiKey.Group == nil {
		return ""
	}
	return apiKey.Group.Platform
}

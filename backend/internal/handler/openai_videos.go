package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Videos handles OpenAI-style video generation for Grok Cookie accounts.
// POST /v1/videos
func (h *OpenAIGatewayHandler) Videos(c *gin.Context) {
	streamStarted := false
	defer h.recoverResponsesPanic(c, &streamStarted)

	requestStart := time.Now()
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	reqLog := requestLogger(
		c,
		"handler.openai_gateway.videos",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)
	if !h.ensureResponsesDependencies(c, reqLog) {
		return
	}

	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			h.errorResponse(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
			return
		}
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	if len(body) == 0 {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return
	}
	parsed, err := h.gatewayService.ParseOpenAIVideoRequestWithContentType(c.GetHeader("Content-Type"), body)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}
	reqLog = reqLog.With(zap.String("model", parsed.Model))
	setOpsRequestContext(c, parsed.Model, false)
	setOpsEndpointContext(c, "", int16(service.RequestTypeFromLegacy(false, false)))

	if decision := h.checkContentModeration(c, reqLog, apiKey, subject, service.ContentModerationProtocolOpenAIImages, parsed.Model, body); decision != nil && decision.Blocked {
		h.errorResponse(c, contentModerationStatus(decision), contentModerationErrorCode(decision), decision.Message)
		return
	}
	channelMapping, _ := h.gatewayService.ResolveChannelMappingAndRestrict(c.Request.Context(), apiKey.GroupID, parsed.Model)
	if h.errorPassthroughService != nil {
		service.BindErrorPassthroughService(c, h.errorPassthroughService)
	}
	subscription, _ := middleware2.GetSubscriptionFromContext(c)
	service.SetOpsLatencyMs(c, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())
	routingStart := time.Now()

	userReleaseFunc, acquired := h.acquireResponsesUserSlot(c, subject.UserID, subject.Concurrency, false, &streamStarted, reqLog)
	if !acquired {
		return
	}
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}
	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription, service.QuotaPlatform(c.Request.Context(), apiKey)); err != nil {
		reqLog.Info("openai.videos.billing_eligibility_check_failed", zap.Error(err))
		status, code, message, retryAfter := billingErrorDetails(err)
		if retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		h.handleStreamingAwareError(c, status, code, message, streamStarted)
		return
	}

	sessionHash := h.gatewayService.GenerateExplicitSessionHash(c, body)
	maxAccountSwitches := h.maxAccountSwitches
	switchCount := 0
	failedAccountIDs := make(map[int64]struct{})
	sameAccountRetryCount := make(map[int64]int)
	var lastFailoverErr *service.UpstreamFailoverError

	for {
		selection, _, err := h.gatewayService.SelectAccountWithSchedulerForCapability(
			c.Request.Context(),
			apiKey.GroupID,
			"",
			sessionHash,
			parsed.Model,
			failedAccountIDs,
			service.OpenAIUpstreamTransportHTTPSSE,
			service.OpenAIEndpointCapabilityVideos,
			false,
		)
		if err != nil {
			reqLog.Warn("openai.videos.account_select_failed", zap.Error(err), zap.Int("excluded_account_count", len(failedAccountIDs)))
			if len(failedAccountIDs) == 0 {
				markOpsRoutingCapacityLimitedIfNoAvailable(c, err)
				h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available Grok Cookie accounts", streamStarted)
				return
			}
			if lastFailoverErr != nil {
				h.handleFailoverExhausted(c, lastFailoverErr, streamStarted)
			} else {
				h.handleFailoverExhaustedSimple(c, 502, streamStarted)
			}
			return
		}
		if selection == nil || selection.Account == nil {
			markOpsRoutingCapacityLimited(c)
			h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available Grok Cookie accounts", streamStarted)
			return
		}
		account := selection.Account
		sessionHash = ensureOpenAIPoolModeSessionHash(sessionHash, account)
		setOpsSelectedAccount(c, account.ID, account.Platform)

		accountReleaseFunc, acquireFailoverErr, acquired := h.acquireResponsesAccountSlotForFailover(c, apiKey.GroupID, sessionHash, selection, false, &streamStarted, reqLog)
		if acquireFailoverErr != nil {
			if !h.handleOpenAIAccountFailover(c, reqLog, account, acquireFailoverErr, failedAccountIDs, &lastFailoverErr, &switchCount, maxAccountSwitches) {
				h.handleFailoverExhausted(c, acquireFailoverErr, streamStarted)
				return
			}
			continue
		}
		if !acquired {
			return
		}
		service.SetOpsLatencyMs(c, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())
		forwardStart := time.Now()
		result, err := func() (*service.OpenAIForwardResult, error) {
			defer func() {
				if accountReleaseFunc != nil {
					accountReleaseFunc()
				}
			}()
			return h.gatewayService.ForwardVideos(c.Request.Context(), c, account, parsed, channelMapping.MappedModel)
		}()
		service.SetOpsLatencyMs(c, service.OpsResponseLatencyMsKey, time.Since(forwardStart).Milliseconds())
		if err != nil {
			if failoverErr, ok := ClassifyUpstreamFailoverError(c.Request.Context(), err); ok {
				if failoverErr.RetryableOnSameAccount {
					retryLimit := account.GetPoolModeRetryCount()
					if sameAccountRetryCount[account.ID] < retryLimit {
						sameAccountRetryCount[account.ID]++
						if !sleepOpenAISameAccountRetry(c.Request.Context(), sameAccountRetryDelay) {
							return
						}
						continue
					}
				}
				if !h.handleOpenAIAccountFailover(c, reqLog, account, failoverErr, failedAccountIDs, &lastFailoverErr, &switchCount, maxAccountSwitches) {
					h.handleFailoverExhausted(c, failoverErr, streamStarted)
					return
				}
				continue
			}
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
			if !c.Writer.Written() {
				h.errorResponse(c, http.StatusBadGateway, "upstream_error", "Upstream request failed")
			}
			reqLog.Warn("openai.videos.forward_failed", zap.Int64("account_id", account.ID), zap.Error(err))
			return
		}
		h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, nil)

		userAgent := c.GetHeader("User-Agent")
		clientIP := ip.GetClientIP(c)
		inboundEndpoint := GetInboundEndpoint(c)
		upstreamEndpoint := GetUpstreamEndpoint(c, account.Platform)
		requestPayloadHash := service.HashUsageRequestPayload(body)
		h.submitOpenAIUsageRecordTask(c.Request.Context(), result, func(ctx context.Context) {
			if err := h.gatewayService.RecordUsage(ctx, &service.OpenAIRecordUsageInput{
				Result:             result,
				APIKey:             apiKey,
				User:               apiKey.User,
				Account:            account,
				Subscription:       subscription,
				InboundEndpoint:    inboundEndpoint,
				UpstreamEndpoint:   upstreamEndpoint,
				UserAgent:          userAgent,
				IPAddress:          clientIP,
				RequestPayloadHash: requestPayloadHash,
				APIKeyService:      h.apiKeyService,
				ChannelUsageFields: channelMapping.ToUsageFields(parsed.Model, result.UpstreamModel),
			}); err != nil {
				logger.L().With(
					zap.String("component", "handler.openai_gateway.videos"),
					zap.Int64("user_id", subject.UserID),
					zap.Int64("api_key_id", apiKey.ID),
					zap.Int64("account_id", account.ID),
				).Error("openai.videos.record_usage_failed", zap.Error(err))
			}
		})
		reqLog.Debug("openai.videos.request_completed", zap.Int64("account_id", account.ID), zap.Int("switch_count", switchCount))
		return
	}
}

// VideoJob handles GET /v1/videos/{video_id}.
func (h *OpenAIGatewayHandler) VideoJob(c *gin.Context) {
	if h.gatewayService == nil {
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "Service temporarily unavailable")
		return
	}
	videoID := strings.TrimSpace(c.Param("video_id"))
	job, ok := h.gatewayService.GetVideoJob(videoID)
	if !ok {
		if apiKey, exists := middleware2.GetAPIKeyFromContext(c); exists && apiKey.Group != nil && apiKey.Group.Platform == service.PlatformXAI {
			h.GrokVideoStatus(c)
			return
		}
		h.errorResponse(c, http.StatusNotFound, "not_found_error", "Video not found")
		return
	}
	c.JSON(http.StatusOK, job.PublicPayload())
}

// VideoContent redirects to the generated Grok asset URL.
func (h *OpenAIGatewayHandler) VideoContent(c *gin.Context) {
	if h.gatewayService == nil {
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "Service temporarily unavailable")
		return
	}
	videoID := strings.TrimSpace(c.Param("video_id"))
	contentURL, found, err := h.gatewayService.GetVideoContentURL(videoID)
	if !found {
		h.errorResponse(c, http.StatusNotFound, "not_found_error", "Video not found")
		return
	}
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}
	c.Redirect(http.StatusTemporaryRedirect, contentURL)
}

// LocalImageFile serves a locally cached generated image.
func (h *OpenAIGatewayHandler) LocalImageFile(c *gin.Context) {
	if h.gatewayService == nil {
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "Service temporarily unavailable")
		return
	}
	id := strings.TrimSpace(c.Query("id"))
	file, err := h.gatewayService.GetLocalImageFile(id)
	if err != nil {
		status := http.StatusNotFound
		if strings.Contains(strings.ToLower(err.Error()), "invalid") {
			status = http.StatusBadRequest
		}
		h.errorResponse(c, status, "not_found_error", err.Error())
		return
	}
	c.Header("Content-Type", file.ContentType)
	c.File(file.Path)
}

// LocalVideoFile serves a locally cached generated video.
func (h *OpenAIGatewayHandler) LocalVideoFile(c *gin.Context) {
	if h.gatewayService == nil {
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "Service temporarily unavailable")
		return
	}
	id := strings.TrimSpace(c.Query("id"))
	file, err := h.gatewayService.GetLocalVideoFile(id)
	if err != nil {
		status := http.StatusNotFound
		if strings.Contains(strings.ToLower(err.Error()), "invalid") {
			status = http.StatusBadRequest
		}
		h.errorResponse(c, status, "not_found_error", err.Error())
		return
	}
	c.Header("Content-Type", file.ContentType)
	c.File(file.Path)
}

// LiveKitToken fetches a short-lived Grok LiveKit token for realtime clients.
// POST /v1/livekit/tokens
func (h *OpenAIGatewayHandler) LiveKitToken(c *gin.Context) {
	streamStarted := false
	defer h.recoverResponsesPanic(c, &streamStarted)

	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	reqLog := requestLogger(
		c,
		"handler.openai_gateway.livekit_token",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)
	if !h.ensureResponsesDependencies(c, reqLog) {
		return
	}

	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			h.errorResponse(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
			return
		}
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	parsed, err := h.gatewayService.ParseGrokLiveKitTokenRequest(body)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}
	setOpsRequestContext(c, "grok-livekit", false)
	setOpsEndpointContext(c, "", int16(service.RequestTypeFromLegacy(false, false)))

	sessionHash := h.gatewayService.GenerateExplicitSessionHash(c, body)
	selection, _, err := h.gatewayService.SelectAccountWithSchedulerForCapability(
		c.Request.Context(),
		apiKey.GroupID,
		"",
		sessionHash,
		"grok-livekit",
		map[int64]struct{}{},
		service.OpenAIUpstreamTransportHTTPSSE,
		service.OpenAIEndpointCapabilityVideos,
		false,
	)
	if err != nil || selection == nil || selection.Account == nil {
		markOpsRoutingCapacityLimitedIfNoAvailable(c, err)
		h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available Grok Cookie accounts", streamStarted)
		return
	}

	account := selection.Account
	setOpsSelectedAccount(c, account.ID, account.Platform)
	accountReleaseFunc, acquired := h.acquireResponsesAccountSlot(c, apiKey.GroupID, sessionHash, selection, false, &streamStarted, reqLog)
	if !acquired {
		return
	}
	defer func() {
		if accountReleaseFunc != nil {
			accountReleaseFunc()
		}
	}()

	result, err := h.gatewayService.ForwardGrokLiveKitToken(c.Request.Context(), c, account, parsed)
	if err != nil {
		h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
		if !c.Writer.Written() {
			h.errorResponse(c, http.StatusBadGateway, "upstream_error", "Upstream request failed")
		}
		reqLog.Warn("openai.livekit_token.forward_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		return
	}
	h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, nil)

	subscription, _ := middleware2.GetSubscriptionFromContext(c)
	userAgent := c.GetHeader("User-Agent")
	clientIP := ip.GetClientIP(c)
	inboundEndpoint := GetInboundEndpoint(c)
	upstreamEndpoint := GetUpstreamEndpoint(c, account.Platform)
	requestPayloadHash := service.HashUsageRequestPayload(body)
	h.submitOpenAIUsageRecordTask(c.Request.Context(), result, func(ctx context.Context) {
		if err := h.gatewayService.RecordUsage(ctx, &service.OpenAIRecordUsageInput{
			Result:             result,
			APIKey:             apiKey,
			User:               apiKey.User,
			Account:            account,
			Subscription:       subscription,
			InboundEndpoint:    inboundEndpoint,
			UpstreamEndpoint:   upstreamEndpoint,
			UserAgent:          userAgent,
			IPAddress:          clientIP,
			RequestPayloadHash: requestPayloadHash,
			APIKeyService:      h.apiKeyService,
		}); err != nil {
			logger.L().With(
				zap.String("component", "handler.openai_gateway.livekit_token"),
				zap.Int64("user_id", subject.UserID),
				zap.Int64("api_key_id", apiKey.ID),
				zap.Int64("account_id", account.ID),
			).Error("openai.livekit_token.record_usage_failed", zap.Error(err))
		}
	})
}

// LiveKitRTC transparently proxies a client WebSocket to Grok LiveKit RTC.
// GET /v1/livekit/rtc?access_token=...
func (h *OpenAIGatewayHandler) LiveKitRTC(c *gin.Context) {
	if !isOpenAIWSUpgradeRequest(c.Request) {
		h.errorResponse(c, http.StatusUpgradeRequired, "invalid_request_error", "WebSocket upgrade required (Upgrade: websocket)")
		return
	}
	setOpenAIClientTransportWS(c)

	streamStarted := false
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	accessToken := strings.TrimSpace(c.Query("access_token"))
	if accessToken == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "access_token is required")
		return
	}

	reqLog := requestLogger(
		c,
		"handler.openai_gateway.livekit_rtc",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
		zap.Bool("openai_ws_mode", true),
	)
	if !h.ensureResponsesDependencies(c, reqLog) {
		return
	}

	setOpsRequestContext(c, "grok-livekit-rtc", true)
	setOpsEndpointContext(c, "", int16(service.RequestTypeWSV2))

	subscription, _ := middleware2.GetSubscriptionFromContext(c)
	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription, service.QuotaPlatform(c.Request.Context(), apiKey)); err != nil {
		reqLog.Info("openai.livekit_rtc.billing_eligibility_check_failed", zap.Error(err))
		status, code, message, retryAfter := billingErrorDetails(err)
		if retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		h.handleStreamingAwareError(c, status, code, message, streamStarted)
		return
	}

	userReleaseFunc, acquired := h.acquireResponsesUserSlot(c, subject.UserID, subject.Concurrency, true, &streamStarted, reqLog)
	if !acquired {
		return
	}
	defer func() {
		if userReleaseFunc != nil {
			userReleaseFunc()
		}
	}()

	sessionHash := h.gatewayService.GenerateExplicitSessionHash(c, []byte(accessToken))
	selection, _, err := h.gatewayService.SelectAccountWithSchedulerForCapability(
		c.Request.Context(),
		apiKey.GroupID,
		"",
		sessionHash,
		"grok-livekit-rtc",
		map[int64]struct{}{},
		service.OpenAIUpstreamTransportResponsesWebsocketV2,
		service.OpenAIEndpointCapabilityLiveKit,
		false,
	)
	if err != nil || selection == nil || selection.Account == nil {
		markOpsRoutingCapacityLimitedIfNoAvailable(c, err)
		h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available Grok Cookie accounts", streamStarted)
		return
	}

	account := selection.Account
	setOpsSelectedAccount(c, account.ID, account.Platform)
	accountReleaseFunc, acquired := h.acquireResponsesAccountSlot(c, apiKey.GroupID, sessionHash, selection, true, &streamStarted, reqLog)
	if !acquired {
		return
	}
	defer func() {
		if accountReleaseFunc != nil {
			accountReleaseFunc()
		}
	}()

	wsConn, err := coderws.Accept(c.Writer, c.Request, &coderws.AcceptOptions{
		CompressionMode: coderws.CompressionContextTakeover,
	})
	if err != nil {
		h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
		reqLog.Warn("openai.livekit_rtc.accept_failed", zap.Error(err))
		return
	}
	defer func() {
		_ = wsConn.CloseNow()
	}()
	wsConn.SetReadLimit(service.ResolveOpenAIWSClientReadLimitBytes(h.cfg))

	result, err := h.gatewayService.ProxyGrokLiveKitRTC(c.Request.Context(), wsConn, account, accessToken)
	if err != nil {
		h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
		reqLog.Warn("openai.livekit_rtc.proxy_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		closeOpenAIClientWS(wsConn, coderws.StatusInternalError, "upstream websocket proxy failed")
		return
	}
	h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, nil)

	userAgent := c.GetHeader("User-Agent")
	clientIP := ip.GetClientIP(c)
	inboundEndpoint := GetInboundEndpoint(c)
	upstreamEndpoint := GetUpstreamEndpoint(c, account.Platform)
	usageResult := &service.OpenAIForwardResult{
		RequestID:       "grok-livekit-rtc",
		ResponseID:      "grok-livekit-rtc",
		Usage:           service.OpenAIUsage{},
		Model:           "grok-livekit-rtc",
		BillingModel:    "grok-livekit-rtc",
		UpstreamModel:   "grok-livekit-rtc",
		ResponseHeaders: http.Header{},
		Duration:        result.Duration,
	}
	h.submitOpenAIUsageRecordTask(c.Request.Context(), usageResult, func(ctx context.Context) {
		if err := h.gatewayService.RecordUsage(ctx, &service.OpenAIRecordUsageInput{
			Result:             usageResult,
			APIKey:             apiKey,
			User:               apiKey.User,
			Account:            account,
			Subscription:       subscription,
			InboundEndpoint:    inboundEndpoint,
			UpstreamEndpoint:   upstreamEndpoint,
			UserAgent:          userAgent,
			IPAddress:          clientIP,
			RequestPayloadHash: service.HashUsageRequestPayload([]byte("grok-livekit-rtc")),
			APIKeyService:      h.apiKeyService,
		}); err != nil {
			reqLog.Error("openai.livekit_rtc.record_usage_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		}
	})
	reqLog.Info(
		"openai.livekit_rtc.closed",
		zap.Int64("account_id", account.ID),
		zap.Int64("client_to_upstream_frames", result.ClientToUpstreamFrames),
		zap.Int64("upstream_to_client_frames", result.UpstreamToClientFrames),
		zap.Int64("dropped_downstream_frames", result.DroppedDownstreamFrames),
	)
}

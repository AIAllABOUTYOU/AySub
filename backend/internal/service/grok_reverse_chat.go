package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

const (
	grokWebDefaultBaseURL = "https://grok.com"
	grokWebChatPath       = "/rest/app-chat/conversations/new"
	grokWebUploadPath     = "/rest/app-chat/upload-file"
	grokWebAssetsBaseURL  = "https://assets.grok.com/"
	grokImagineWSURL      = "wss://grok.com/ws/imagine/listen"
	grokWebMaxUploadBytes = 25 << 20
)

var (
	grokRenderTagRE       = regexp.MustCompile(`(?s)<grok:render\s+[^>]*>.*?</grok:render>`)
	grokRenderTagDetailRE = regexp.MustCompile(`(?s)<grok:render\s+card_id="([^"]+)"\s+card_type="([^"]+)"\s+type="([^"]+)"[^>]*>.*?</grok:render>`)
	grokImagineImageURLRE = regexp.MustCompile(`/images/([A-Za-z0-9_-]+)\.(png|jpg|jpeg|webp|gif|bmp)`)
	whitespaceRE          = regexp.MustCompile(`\s+`)
)

type grokStreamEvent struct {
	Kind       string
	Content    string
	Annotation grokAnnotation
}

type grokAnnotation struct {
	URL        string
	Title      string
	StartIndex int
	EndIndex   int
}

type grokPendingCitation struct {
	URL    string
	Title  string
	Needle string
}

type grokSearchSource struct {
	URL   string
	Title string
	Type  string
}

type grokGeneratedImage struct {
	URL  string
	ID   string
	Blob string
}

type grokWebFileInput struct {
	Filename string
	Mime     string
	Content  string
}

type grokWebStreamAdapter struct {
	cardCache         map[string]string
	citationOrder     []string
	citationMap       map[string]int
	lastCitationIndex int
	pendingCitations  []grokPendingCitation
	annotations       []grokAnnotation
	textOffset        int
	searchSources     []grokSearchSource
	searchSourceURLs  map[string]struct{}
	imageURLs         []grokGeneratedImage
	contentStarted    bool
}

func (s *OpenAIGatewayService) ForwardGrokAnthropicMessages(ctx context.Context, c *gin.Context, account *Account, body []byte, defaultMappedModel string) (*OpenAIForwardResult, error) {
	startTime := time.Now()
	if account == nil || !account.IsXAICookie() {
		writeAnthropicError(c, http.StatusBadRequest, "invalid_request_error", "Grok Messages forwarding requires an xAI cookie account")
		return nil, errors.New("grok messages forwarding requires xai cookie account")
	}
	originalModel := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	if originalModel == "" {
		writeAnthropicError(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return nil, errors.New("missing model in request")
	}
	clientStream := gjson.GetBytes(body, "stream").Bool()

	billingModel := resolveOpenAIForwardModel(account, originalModel, defaultMappedModel)
	upstreamModel := normalizeOpenAIModelForUpstream(account, billingModel)
	if isGrokConsoleModel(billingModel) {
		return s.ForwardGrokConsoleAnthropicMessages(ctx, c, account, body, originalModel, billingModel, upstreamModel, startTime)
	}
	prompt, fileInputs, err := flattenGrokChatMessages(body)
	if err != nil {
		writeAnthropicError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return nil, err
	}
	if strings.TrimSpace(prompt) == "" && len(fileInputs) > 0 {
		prompt = "[user]: Please analyze the attached file."
	}
	fileAttachments, err := s.uploadGrokWebFileAttachments(ctx, account, fileInputs)
	if err != nil {
		writeAnthropicError(c, http.StatusBadGateway, "api_error", "Grok file upload failed")
		return nil, err
	}

	upstreamBody, err := buildGrokWebChatPayload(account, upstreamModel, prompt, fileAttachments)
	if err != nil {
		writeAnthropicError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return nil, err
	}
	req, err := s.buildGrokWebChatRequest(ctx, account, upstreamBody)
	if err != nil {
		writeAnthropicError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return nil, err
	}
	if s.httpUpstream == nil {
		writeAnthropicError(c, http.StatusBadGateway, "api_error", "HTTP upstream is not configured")
		return nil, errors.New("http upstream is not configured")
	}
	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.doGrokWebRequest(req, account, proxyURL)
	if err != nil {
		safeErr := sanitizeUpstreamErrorMessage(err.Error())
		setOpsUpstreamError(c, 0, safeErr, "")
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: 0,
			Kind:               "request_error",
			Message:            safeErr,
		})
		writeAnthropicError(c, http.StatusBadGateway, "api_error", "Grok upstream request failed")
		return nil, fmt.Errorf("grok messages upstream request failed: %s", safeErr)
	}
	defer func() { _ = resp.Body.Close() }()

	resp = s.retryGrokWebResponseAfterCloudflareChallenge(ctx, account, resp, proxyURL, "messages", func() (*http.Request, error) {
		return s.buildGrokWebChatRequest(ctx, account, upstreamBody)
	})
	if resp.StatusCode >= 400 {
		respBody := s.readUpstreamErrorBody(resp)
		upstreamMsg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
		if upstreamMsg == "" {
			upstreamMsg = http.StatusText(resp.StatusCode)
		}
		setOpsUpstreamError(c, resp.StatusCode, upstreamMsg, truncateString(string(respBody), 2048))
		if s.shouldFailoverOpenAIUpstreamResponse(resp.StatusCode, upstreamMsg, respBody) {
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
				Platform:           account.Platform,
				AccountID:          account.ID,
				AccountName:        account.Name,
				UpstreamStatusCode: resp.StatusCode,
				UpstreamRequestID:  resp.Header.Get("x-request-id"),
				Kind:               "failover",
				Message:            upstreamMsg,
				Detail:             truncateString(string(respBody), 2048),
			})
			s.handleOpenAIAccountUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody, upstreamModel)
			return nil, &UpstreamFailoverError{
				StatusCode:             resp.StatusCode,
				ResponseBody:           respBody,
				RetryableOnSameAccount: false,
			}
		}
		writeAnthropicError(c, resp.StatusCode, "api_error", upstreamMsg)
		return nil, fmt.Errorf("grok messages upstream returned %d: %s", resp.StatusCode, upstreamMsg)
	}

	responseID := "msg_grok_" + newGrokRequestID()
	promptTokens := estimateGrokTextTokens(prompt)
	if clientStream {
		return s.streamGrokWebAnthropicMessages(c, resp, responseID, originalModel, billingModel, upstreamModel, promptTokens, startTime)
	}
	return s.bufferGrokWebAnthropicMessages(c, resp, responseID, originalModel, billingModel, upstreamModel, promptTokens, startTime)
}

func (s *OpenAIGatewayService) ForwardGrokChatCompletions(ctx context.Context, c *gin.Context, account *Account, body []byte, defaultMappedModel string) (*OpenAIForwardResult, error) {
	startTime := time.Now()
	if account == nil || !account.IsXAICookie() {
		writeChatCompletionsError(c, http.StatusBadRequest, "invalid_request_error", "Grok Cookie forwarding requires an xAI cookie account")
		return nil, errors.New("grok cookie forwarding requires xai cookie account")
	}
	originalModel := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	if originalModel == "" {
		writeChatCompletionsError(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return nil, errors.New("missing model in request")
	}
	clientStream := gjson.GetBytes(body, "stream").Bool()

	billingModel := resolveOpenAIForwardModel(account, originalModel, defaultMappedModel)
	upstreamModel := normalizeOpenAIModelForUpstream(account, billingModel)
	if isGrokConsoleModel(billingModel) {
		return s.ForwardGrokConsoleChatCompletions(ctx, c, account, body, originalModel, billingModel, upstreamModel, startTime)
	}
	prompt, fileInputs, err := flattenGrokChatMessages(body)
	if err != nil {
		writeChatCompletionsError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return nil, err
	}
	if strings.TrimSpace(prompt) == "" && len(fileInputs) > 0 {
		prompt = "[user]: Please analyze the attached file."
	}
	fileAttachments, err := s.uploadGrokWebFileAttachments(ctx, account, fileInputs)
	if err != nil {
		writeChatCompletionsError(c, http.StatusBadGateway, "upstream_error", "Grok file upload failed")
		return nil, err
	}

	upstreamBody, err := buildGrokWebChatPayload(account, upstreamModel, prompt, fileAttachments)
	if err != nil {
		writeChatCompletionsError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return nil, err
	}

	req, err := s.buildGrokWebChatRequest(ctx, account, upstreamBody)
	if err != nil {
		writeChatCompletionsError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return nil, err
	}

	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	if s.httpUpstream == nil {
		writeChatCompletionsError(c, http.StatusBadGateway, "upstream_error", "HTTP upstream is not configured")
		return nil, errors.New("http upstream is not configured")
	}
	resp, err := s.doGrokWebRequest(req, account, proxyURL)
	if err != nil {
		safeErr := sanitizeUpstreamErrorMessage(err.Error())
		setOpsUpstreamError(c, 0, safeErr, "")
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: 0,
			Kind:               "request_error",
			Message:            safeErr,
		})
		writeChatCompletionsError(c, http.StatusBadGateway, "upstream_error", "Grok upstream request failed")
		return nil, fmt.Errorf("grok upstream request failed: %s", safeErr)
	}
	defer func() { _ = resp.Body.Close() }()

	resp = s.retryGrokWebResponseAfterCloudflareChallenge(ctx, account, resp, proxyURL, "chat_completions", func() (*http.Request, error) {
		return s.buildGrokWebChatRequest(ctx, account, upstreamBody)
	})
	if resp.StatusCode >= 400 {
		respBody := s.readUpstreamErrorBody(resp)
		upstreamMsg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
		if upstreamMsg == "" {
			upstreamMsg = http.StatusText(resp.StatusCode)
		}
		setOpsUpstreamError(c, resp.StatusCode, upstreamMsg, truncateString(string(respBody), 2048))
		if s.shouldFailoverOpenAIUpstreamResponse(resp.StatusCode, upstreamMsg, respBody) {
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
				Platform:           account.Platform,
				AccountID:          account.ID,
				AccountName:        account.Name,
				UpstreamStatusCode: resp.StatusCode,
				UpstreamRequestID:  resp.Header.Get("x-request-id"),
				Kind:               "failover",
				Message:            upstreamMsg,
				Detail:             truncateString(string(respBody), 2048),
			})
			s.handleOpenAIAccountUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody, upstreamModel)
			return nil, &UpstreamFailoverError{
				StatusCode:             resp.StatusCode,
				ResponseBody:           respBody,
				RetryableOnSameAccount: false,
			}
		}
		writeChatCompletionsError(c, resp.StatusCode, "upstream_error", upstreamMsg)
		return nil, fmt.Errorf("grok upstream returned %d: %s", resp.StatusCode, upstreamMsg)
	}

	responseID := newGrokChatCompletionID()
	promptTokens := estimateGrokTextTokens(prompt)
	logger.L().Debug("grok chat_completions reverse: forwarding through web cookie",
		zap.Int64("account_id", account.ID),
		zap.String("original_model", originalModel),
		zap.String("billing_model", billingModel),
		zap.String("upstream_model", upstreamModel),
		zap.Bool("stream", clientStream),
	)

	if clientStream {
		return s.streamGrokWebChatCompletions(c, resp, account, responseID, originalModel, billingModel, upstreamModel, promptTokens, startTime)
	}
	return s.bufferGrokWebChatCompletions(c, resp, responseID, originalModel, billingModel, upstreamModel, promptTokens, startTime)
}

func (s *OpenAIGatewayService) ForwardGrokResponses(ctx context.Context, c *gin.Context, account *Account, body []byte, defaultMappedModel string) (*OpenAIForwardResult, error) {
	startTime := time.Now()
	if account == nil || !account.IsXAICookie() {
		writeResponsesError(c, http.StatusBadRequest, "invalid_request_error", "Grok Responses forwarding requires an xAI cookie account")
		return nil, errors.New("grok responses forwarding requires xai cookie account")
	}
	originalModel := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	if originalModel == "" {
		writeResponsesError(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return nil, errors.New("missing model in request")
	}
	clientStream := gjson.GetBytes(body, "stream").Bool()

	billingModel := resolveOpenAIForwardModel(account, originalModel, defaultMappedModel)
	upstreamModel := normalizeOpenAIModelForUpstream(account, billingModel)
	if isGrokConsoleModel(billingModel) {
		return s.ForwardGrokConsoleResponses(ctx, c, account, body, originalModel, billingModel, upstreamModel, startTime)
	}
	prompt, fileInputs, err := flattenGrokChatMessages(body)
	if err != nil {
		writeResponsesError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return nil, err
	}
	if strings.TrimSpace(prompt) == "" && len(fileInputs) > 0 {
		prompt = "[user]: Please analyze the attached file."
	}
	fileAttachments, err := s.uploadGrokWebFileAttachments(ctx, account, fileInputs)
	if err != nil {
		writeResponsesError(c, http.StatusBadGateway, "upstream_error", "Grok file upload failed")
		return nil, err
	}

	upstreamBody, err := buildGrokWebChatPayload(account, upstreamModel, prompt, fileAttachments)
	if err != nil {
		writeResponsesError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return nil, err
	}
	req, err := s.buildGrokWebChatRequest(ctx, account, upstreamBody)
	if err != nil {
		writeResponsesError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return nil, err
	}
	if s.httpUpstream == nil {
		writeResponsesError(c, http.StatusBadGateway, "upstream_error", "HTTP upstream is not configured")
		return nil, errors.New("http upstream is not configured")
	}
	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.doGrokWebRequest(req, account, proxyURL)
	if err != nil {
		safeErr := sanitizeUpstreamErrorMessage(err.Error())
		setOpsUpstreamError(c, 0, safeErr, "")
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: 0,
			Kind:               "request_error",
			Message:            safeErr,
		})
		writeResponsesError(c, http.StatusBadGateway, "upstream_error", "Grok upstream request failed")
		return nil, fmt.Errorf("grok upstream request failed: %s", safeErr)
	}
	defer func() { _ = resp.Body.Close() }()

	resp = s.retryGrokWebResponseAfterCloudflareChallenge(ctx, account, resp, proxyURL, "responses", func() (*http.Request, error) {
		return s.buildGrokWebChatRequest(ctx, account, upstreamBody)
	})
	if resp.StatusCode >= 400 {
		respBody := s.readUpstreamErrorBody(resp)
		upstreamMsg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
		if upstreamMsg == "" {
			upstreamMsg = http.StatusText(resp.StatusCode)
		}
		setOpsUpstreamError(c, resp.StatusCode, upstreamMsg, truncateString(string(respBody), 2048))
		if s.shouldFailoverOpenAIUpstreamResponse(resp.StatusCode, upstreamMsg, respBody) {
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
				Platform:           account.Platform,
				AccountID:          account.ID,
				AccountName:        account.Name,
				UpstreamStatusCode: resp.StatusCode,
				UpstreamRequestID:  resp.Header.Get("x-request-id"),
				Kind:               "failover",
				Message:            upstreamMsg,
				Detail:             truncateString(string(respBody), 2048),
			})
			s.handleOpenAIAccountUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody, upstreamModel)
			return nil, &UpstreamFailoverError{
				StatusCode:             resp.StatusCode,
				ResponseBody:           respBody,
				RetryableOnSameAccount: false,
			}
		}
		writeResponsesError(c, resp.StatusCode, "upstream_error", upstreamMsg)
		return nil, fmt.Errorf("grok upstream returned %d: %s", resp.StatusCode, upstreamMsg)
	}

	responseID := newGrokResponseID()
	messageID := newGrokResponseMessageID()
	promptTokens := estimateGrokTextTokens(prompt)
	var result *OpenAIForwardResult
	if clientStream {
		result, err = s.streamGrokWebResponses(c, resp, account, responseID, messageID, originalModel, billingModel, upstreamModel, promptTokens, startTime)
	} else {
		result, err = s.bufferGrokWebResponses(c, resp, responseID, messageID, originalModel, billingModel, upstreamModel, promptTokens, startTime)
	}
	if result != nil {
		result.ReasoningEffort = extractOpenAIReasoningEffortFromBody(body, originalModel)
	}
	return result, err
}

func (s *OpenAIGatewayService) ForwardGrokImages(ctx context.Context, c *gin.Context, account *Account, parsed *OpenAIImagesRequest, channelMappedModel string) (*OpenAIForwardResult, error) {
	startTime := time.Now()
	if account == nil || !account.IsXAICookie() {
		return nil, errors.New("grok images forwarding requires xai cookie account")
	}
	if parsed == nil {
		return nil, errors.New("parsed images request is required")
	}
	requestModel := strings.TrimSpace(parsed.Model)
	if mapped := strings.TrimSpace(channelMappedModel); mapped != "" {
		requestModel = mapped
	}
	if requestModel == "" {
		requestModel = "gpt-image-2"
	}
	billingModel := requestModel
	upstreamModel := normalizeOpenAIModelForUpstream(account, account.GetMappedModel(billingModel))
	prompt, files := buildGrokImagesPromptAndFiles(parsed)
	if strings.TrimSpace(prompt) == "" {
		return nil, errors.New("prompt is required")
	}
	if !parsed.IsEdits() && len(files) == 0 {
		return s.forwardGrokImagineImages(ctx, c, account, requestModel, billingModel, upstreamModel, parsed, prompt, startTime)
	}
	fileAttachments, err := s.uploadGrokWebFileAttachments(ctx, account, files)
	if err != nil {
		return nil, err
	}
	upstreamBody, err := buildGrokWebChatPayload(account, upstreamModel, prompt, fileAttachments)
	if err != nil {
		return nil, err
	}
	upstreamBody, err = setGrokWebImageGenerationCount(upstreamBody, parsed.N)
	if err != nil {
		return nil, err
	}
	req, err := s.buildGrokWebChatRequest(ctx, account, upstreamBody)
	if err != nil {
		return nil, err
	}
	if s.httpUpstream == nil {
		return nil, errors.New("http upstream is not configured")
	}
	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.doGrokWebRequest(req, account, proxyURL)
	if err != nil {
		safeErr := sanitizeUpstreamErrorMessage(err.Error())
		setOpsUpstreamError(c, 0, safeErr, "")
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: 0,
			Kind:               "request_error",
			Message:            safeErr,
		})
		return nil, fmt.Errorf("grok images upstream request failed: %s", safeErr)
	}
	defer func() { _ = resp.Body.Close() }()
	resp = s.retryGrokWebResponseAfterCloudflareChallenge(ctx, account, resp, proxyURL, "images", func() (*http.Request, error) {
		return s.buildGrokWebChatRequest(ctx, account, upstreamBody)
	})
	if resp.StatusCode >= 400 {
		respBody := s.readUpstreamErrorBody(resp)
		upstreamMsg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
		if upstreamMsg == "" {
			upstreamMsg = http.StatusText(resp.StatusCode)
		}
		setOpsUpstreamError(c, resp.StatusCode, upstreamMsg, truncateString(string(respBody), 2048))
		if s.shouldFailoverOpenAIUpstreamResponse(resp.StatusCode, upstreamMsg, respBody) {
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
				Platform:           account.Platform,
				AccountID:          account.ID,
				AccountName:        account.Name,
				UpstreamStatusCode: resp.StatusCode,
				UpstreamRequestID:  resp.Header.Get("x-request-id"),
				Kind:               "failover",
				Message:            upstreamMsg,
				Detail:             truncateString(string(respBody), 2048),
			})
			s.handleOpenAIAccountUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody, upstreamModel)
			return nil, &UpstreamFailoverError{
				StatusCode:             resp.StatusCode,
				ResponseBody:           respBody,
				RetryableOnSameAccount: false,
			}
		}
		return nil, fmt.Errorf("grok images upstream returned %d: %s", resp.StatusCode, upstreamMsg)
	}
	if parsed.Stream {
		return s.streamGrokWebImages(ctx, c, resp, account, requestModel, billingModel, upstreamModel, parsed, prompt, startTime)
	}
	return s.bufferGrokWebImages(ctx, c, resp, account, requestModel, billingModel, upstreamModel, parsed, prompt, startTime)
}

func (s *OpenAIGatewayService) forwardGrokImagineImages(ctx context.Context, c *gin.Context, account *Account, requestModel, billingModel, upstreamModel string, parsed *OpenAIImagesRequest, prompt string, startTime time.Time) (*OpenAIForwardResult, error) {
	requestID := newGrokRequestID()
	images, err := s.collectGrokImagineImages(ctx, account, prompt, upstreamModel, parsed.N, grokImagineAspectRatio(parsed.Size), requestID)
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, errors.New("grok imagine websocket did not include generated images")
	}
	resolved, err := s.resolveGrokImageOutputs(ctx, account, images, parsed)
	if err != nil {
		return nil, err
	}
	if len(resolved) == 0 {
		return nil, errors.New("grok imagine websocket did not resolve generated images")
	}
	if parsed.Stream {
		if s.responseHeaderFilter != nil {
			_ = s.responseHeaderFilter
		}
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Status(http.StatusOK)
		flusher, _ := c.Writer.(http.Flusher)
		firstTokenMs := int(time.Since(startTime).Milliseconds())
		for idx, image := range resolved {
			event := map[string]any{
				"type":           "image_generation.completed",
				"id":             fmt.Sprintf("grok_img_%d", idx),
				"created":        time.Now().Unix(),
				"revised_prompt": prompt,
				"size":           grokImagesOutputSize(parsed),
			}
			for key, value := range image {
				event[key] = value
			}
			if err := writeGrokSSEJSON(c.Writer, event); err != nil {
				return nil, err
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if _, err := io.WriteString(c.Writer, "data: [DONE]\n\n"); err != nil {
			return nil, err
		}
		if flusher != nil {
			flusher.Flush()
		}
		return &OpenAIForwardResult{
			RequestID:        requestID,
			Usage:            OpenAIUsage{InputTokens: estimateGrokTextTokens(prompt), OutputTokens: len(resolved)},
			Model:            requestModel,
			BillingModel:     billingModel,
			UpstreamModel:    upstreamModel,
			Stream:           true,
			ResponseHeaders:  http.Header{"Content-Type": []string{"text/event-stream"}},
			Duration:         time.Since(startTime),
			FirstTokenMs:     &firstTokenMs,
			ImageCount:       len(resolved),
			ImageSize:        parsed.SizeTier,
			ImageInputSize:   parsed.Size,
			ImageOutputSizes: grokImagesOutputSizes(parsed, len(resolved)),
		}, nil
	}
	data := make([]any, 0, len(resolved))
	for _, image := range resolved {
		item := map[string]any{
			"revised_prompt": prompt,
			"size":           grokImagesOutputSize(parsed),
		}
		for key, value := range image {
			item[key] = value
		}
		data = append(data, item)
	}
	c.JSON(http.StatusOK, map[string]any{
		"created": time.Now().Unix(),
		"data":    data,
		"usage": map[string]any{
			"input_tokens":  estimateGrokTextTokens(prompt),
			"output_tokens": len(resolved),
			"total_tokens":  estimateGrokTextTokens(prompt) + len(resolved),
		},
	})
	return &OpenAIForwardResult{
		RequestID:        requestID,
		Usage:            OpenAIUsage{InputTokens: estimateGrokTextTokens(prompt), OutputTokens: len(resolved)},
		Model:            requestModel,
		BillingModel:     billingModel,
		UpstreamModel:    upstreamModel,
		Stream:           false,
		ResponseHeaders:  http.Header{"Content-Type": []string{"application/json"}},
		Duration:         time.Since(startTime),
		ImageCount:       len(resolved),
		ImageSize:        parsed.SizeTier,
		ImageInputSize:   parsed.Size,
		ImageOutputSizes: grokImagesOutputSizes(parsed, len(resolved)),
	}, nil
}

func (s *OpenAIGatewayService) collectGrokImagineImages(ctx context.Context, account *Account, prompt, upstreamModel string, wantCount int, aspectRatio, requestID string) ([]grokGeneratedImage, error) {
	if wantCount <= 0 {
		wantCount = 1
	}
	headers, err := buildGrokImagineWSHeaders(account, s.cfg)
	if err != nil {
		return nil, err
	}
	headers.Set("x-xai-request-id", requestID)
	dialer := s.getOpenAIWSPassthroughDialer()
	if dialer == nil {
		return nil, errors.New("websocket dialer is not configured")
	}
	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	dialCtx, cancelDial := context.WithTimeout(ctx, 30*time.Second)
	conn, status, _, err := dialer.Dial(dialCtx, grokImagineWSURL, headers, proxyURL)
	cancelDial()
	if err != nil {
		if status > 0 {
			return nil, fmt.Errorf("grok imagine websocket dial returned %d: %s", status, sanitizeUpstreamErrorMessage(err.Error()))
		}
		return nil, fmt.Errorf("grok imagine websocket dial failed: %s", sanitizeUpstreamErrorMessage(err.Error()))
	}
	defer func() { _ = conn.Close() }()
	if err := conn.WriteJSON(ctx, buildGrokImagineResetMessage()); err != nil {
		return nil, fmt.Errorf("write grok imagine reset message: %w", err)
	}
	if err := conn.WriteJSON(ctx, buildGrokImagineRequestMessage(requestID, prompt, aspectRatio, grokImagineEnablePro(upstreamModel))); err != nil {
		return nil, fmt.Errorf("write grok imagine request message: %w", err)
	}

	type slot struct {
		image     grokGeneratedImage
		completed bool
		moderated bool
	}
	slots := make(map[string]*slot)
	images := make([]grokGeneratedImage, 0, wantCount)
	deadline := time.Now().Add(120 * time.Second)
	for len(images) < wantCount && time.Now().Before(deadline) {
		readCtx, cancelRead := context.WithTimeout(ctx, 15*time.Second)
		payload, err := conn.ReadMessage(readCtx)
		cancelRead()
		if err != nil {
			if len(images) > 0 {
				break
			}
			return nil, fmt.Errorf("read grok imagine websocket message: %w", err)
		}
		msg := gjson.ParseBytes(payload)
		switch msg.Get("type").String() {
		case "error":
			code := strings.TrimSpace(msg.Get("err_code").String())
			message := sanitizeUpstreamErrorMessage(strings.TrimSpace(grokFirstNonEmpty(msg.Get("err_msg").String(), string(payload))))
			if code == "" {
				code = "upstream_error"
			}
			return nil, fmt.Errorf("grok imagine websocket %s: %s", code, message)
		case "json":
			imageID := strings.TrimSpace(grokFirstNonEmpty(msg.Get("image_id").String(), msg.Get("job_id").String()))
			if imageID == "" {
				continue
			}
			current := slots[imageID]
			if current == nil {
				current = &slot{image: grokGeneratedImage{ID: imageID}}
				slots[imageID] = current
			}
			status := strings.TrimSpace(msg.Get("current_status").String())
			if status == "completed" {
				current.completed = true
				current.moderated = msg.Get("moderated").Bool()
				if !current.moderated && current.image.URL != "" {
					images = append(images, current.image)
					delete(slots, imageID)
				}
			}
		case "image":
			imageURL := strings.TrimSpace(msg.Get("url").String())
			imageID := grokImagineImageIDFromMessage(msg)
			if imageID == "" {
				continue
			}
			current := slots[imageID]
			if current == nil {
				current = &slot{image: grokGeneratedImage{ID: imageID}}
				slots[imageID] = current
			}
			current.image.URL = imageURL
			current.image.Blob = strings.TrimSpace(msg.Get("blob").String())
			if current.completed && !current.moderated {
				images = append(images, current.image)
				delete(slots, imageID)
			}
		}
	}
	return images, nil
}

func (s *OpenAIGatewayService) buildGrokWebChatRequest(ctx context.Context, account *Account, body []byte) (*http.Request, error) {
	targetURL, err := grokWebChatURL(account)
	if err != nil {
		return nil, err
	}
	upstreamCtx, releaseUpstreamCtx := detachUpstreamContext(ctx)
	req, err := http.NewRequestWithContext(upstreamCtx, http.MethodPost, targetURL, bytes.NewReader(body))
	releaseUpstreamCtx()
	if err != nil {
		return nil, err
	}
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", grokFirstNonEmpty(account.GetCredential("accept_language"), "zh-CN,zh;q=0.9,en;q=0.8"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", grokFirstNonEmpty(account.GetCredential("origin"), grokWebDefaultBaseURL))
	req.Header.Set("Referer", grokFirstNonEmpty(account.GetCredential("referer"), grokWebDefaultBaseURL+"/"))
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("User-Agent", grokFirstNonEmpty(account.GetCredential("user_agent"), defaultGrokWebUserAgent()))
	req.Header.Set("x-xai-request-id", newGrokRequestID())
	req.Header.Set("Cookie", buildGrokWebCookieHeader(account))
	if req.Header.Get("Cookie") == "" {
		return nil, errors.New("sso_token or cookie is required for Grok Cookie account")
	}
	applyGrokWebCompatibilityHeaders(req.Header, account, s.cfg)
	return req, nil
}

func (s *OpenAIGatewayService) doGrokWebRequest(req *http.Request, account *Account, proxyURL string) (*http.Response, error) {
	if s == nil || s.httpUpstream == nil {
		return nil, errors.New("http upstream is not configured")
	}
	accountID := int64(0)
	concurrency := 0
	if account != nil {
		accountID = account.ID
		concurrency = account.Concurrency
	}
	return s.httpUpstream.DoWithTLS(req, proxyURL, accountID, concurrency, grokWebTLSProfile(account))
}

func grokWebTLSProfile(account *Account) *tlsfingerprint.Profile {
	if account == nil || !account.IsXAICookie() {
		return nil
	}
	for _, key := range []string{"disable_tls_fingerprint", "grok_disable_tls_fingerprint"} {
		if v, ok := accountBoolOverride(account.Credentials, key); ok && v {
			return nil
		}
		if v, ok := accountBoolOverride(account.Extra, key); ok && v {
			return nil
		}
	}
	return &tlsfingerprint.Profile{Name: "Grok Web Built-in Default (Node.js 24.x)"}
}

func grokWebChatURL(account *Account) (string, error) {
	return grokWebEndpointURL(account, grokWebChatPath)
}

func grokWebUploadURL(account *Account) (string, error) {
	return grokWebEndpointURL(account, grokWebUploadPath)
}

func grokWebEndpointURL(account *Account, path string) (string, error) {
	base := strings.TrimSpace(account.GetCredential("base_url"))
	if base == "" {
		base = grokWebDefaultBaseURL
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid Grok base_url: %s", base)
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + path
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func (s *OpenAIGatewayService) uploadGrokWebFileAttachments(ctx context.Context, account *Account, files []grokWebFileInput) ([]string, error) {
	if len(files) == 0 {
		return nil, nil
	}
	if s.httpUpstream == nil {
		return nil, errors.New("http upstream is not configured")
	}
	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	attachments := make([]string, 0, len(files))
	for _, file := range files {
		req, err := s.buildGrokWebUploadRequest(ctx, account, file)
		if err != nil {
			return nil, err
		}
		resp, err := s.doGrokWebRequest(req, account, proxyURL)
		if err != nil {
			return nil, fmt.Errorf("grok file upload request failed: %w", err)
		}
		respBody, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read grok file upload response: %w", readErr)
		}
		if resp.StatusCode >= 400 {
			msg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
			if msg == "" {
				msg = http.StatusText(resp.StatusCode)
			}
			return nil, fmt.Errorf("grok file upload returned %d: %s", resp.StatusCode, msg)
		}
		fileID := strings.TrimSpace(grokFirstNonEmpty(
			gjson.GetBytes(respBody, "fileMetadataId").String(),
			gjson.GetBytes(respBody, "fileId").String(),
		))
		if fileID == "" {
			return nil, errors.New("grok file upload response missing fileMetadataId")
		}
		attachments = append(attachments, fileID)
	}
	return attachments, nil
}

func (s *OpenAIGatewayService) buildGrokWebUploadRequest(ctx context.Context, account *Account, file grokWebFileInput) (*http.Request, error) {
	targetURL, err := grokWebUploadURL(account)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(map[string]any{
		"fileName":     file.Filename,
		"fileMimeType": file.Mime,
		"content":      file.Content,
	})
	if err != nil {
		return nil, err
	}
	upstreamCtx, releaseUpstreamCtx := detachUpstreamContext(ctx)
	req, err := http.NewRequestWithContext(upstreamCtx, http.MethodPost, targetURL, bytes.NewReader(payload))
	releaseUpstreamCtx()
	if err != nil {
		return nil, err
	}
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", grokFirstNonEmpty(account.GetCredential("accept_language"), "zh-CN,zh;q=0.9,en;q=0.8"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", grokFirstNonEmpty(account.GetCredential("origin"), grokWebDefaultBaseURL))
	req.Header.Set("Referer", grokFirstNonEmpty(account.GetCredential("referer"), grokWebDefaultBaseURL+"/"))
	req.Header.Set("User-Agent", grokFirstNonEmpty(account.GetCredential("user_agent"), defaultGrokWebUserAgent()))
	req.Header.Set("x-xai-request-id", newGrokRequestID())
	req.Header.Set("Cookie", buildGrokWebCookieHeader(account))
	if req.Header.Get("Cookie") == "" {
		return nil, errors.New("sso_token or cookie is required for Grok Cookie account")
	}
	applyGrokWebCompatibilityHeaders(req.Header, account, s.cfg)
	return req, nil
}

func buildGrokWebChatPayload(account *Account, model string, message string, fileAttachments []string) ([]byte, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		return nil, errors.New("messages must contain text content")
	}
	modeID := resolveGrokWebModeID(account, model)
	attachments := make([]string, 0, len(fileAttachments))
	attachments = append(attachments, fileAttachments...)
	payload := map[string]any{
		"collectionIds": []any{},
		"connectors":    []any{},
		"deviceEnvInfo": map[string]any{
			"darkModeEnabled":  false,
			"devicePixelRatio": 2,
			"screenHeight":     1329,
			"screenWidth":      2056,
			"viewportHeight":   1083,
			"viewportWidth":    2056,
		},
		"disableMemory":               credentialBool(account, "disable_memory", true),
		"disableSearch":               credentialBool(account, "disable_search", false),
		"disableSelfHarmShortCircuit": false,
		"disableTextFollowUps":        false,
		"enableImageGeneration":       true,
		"enableImageStreaming":        true,
		"enableSideBySide":            true,
		"fileAttachments":             attachments,
		"forceConcise":                false,
		"forceSideBySide":             false,
		"imageAttachments":            []any{},
		"imageGenerationCount":        2,
		"isAsyncChat":                 false,
		"message":                     message,
		"modeId":                      modeID,
		"responseMetadata":            map[string]any{},
		"returnImageBytes":            false,
		"returnRawGrokInXaiRequest":   false,
		"searchAllConnectors":         false,
		"sendFinalMetadata":           true,
		"temporary":                   credentialBool(account, "temporary", true),
		"toolOverrides": map[string]any{
			"gmailSearch":           false,
			"googleCalendarSearch":  false,
			"outlookSearch":         false,
			"outlookCalendarSearch": false,
			"googleDriveSearch":     false,
		},
	}
	if custom := strings.TrimSpace(account.GetCredential("custom_instruction")); custom != "" {
		payload["customPersonality"] = custom
	}
	return json.Marshal(payload)
}

func (s *OpenAIGatewayService) streamGrokWebChatCompletions(c *gin.Context, resp *http.Response, account *Account, responseID, originalModel, billingModel, upstreamModel string, promptTokens int, startTime time.Time) (*OpenAIForwardResult, error) {
	if s.responseHeaderFilter != nil {
		responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)
	flusher, _ := c.Writer.(http.Flusher)

	// 发送心跳以防止在上游思考期间连接超时
	if _, err := io.WriteString(c.Writer, ": heartbeat\n\n"); err != nil {
		return nil, fmt.Errorf("write initial heartbeat: %w", err)
	}
	if flusher != nil {
		flusher.Flush()
	}

	var output strings.Builder
	var reasoning strings.Builder
	firstTokenMs := 0
	seenFirst := false
	adapter := newGrokWebStreamAdapter()
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		events, done, err := adapter.FeedLine(line)
		if err != nil {
			return nil, err
		}
		for _, ev := range events {
			switch ev.Kind {
			case "text":
				if ev.Content == "" {
					continue
				}
				if !seenFirst {
					seenFirst = true
					firstTokenMs = int(time.Since(startTime).Milliseconds())
				}
				output.WriteString(ev.Content)
				if err := writeGrokSSEJSON(c.Writer, grokChatCompletionChunk(responseID, originalModel, ev.Content, false)); err != nil {
					return nil, err
				}
				if flusher != nil {
					flusher.Flush()
				}
			case "thinking":
				if ev.Content == "" {
					continue
				}
				reasoning.WriteString(ev.Content)
				if err := writeGrokSSEJSON(c.Writer, grokChatCompletionThinkingChunk(responseID, originalModel, ev.Content)); err != nil {
					return nil, err
				}
				if flusher != nil {
					flusher.Flush()
				}
			}
		}
		if done {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read grok stream: %w", err)
	}
	if imageMarkdown := adapter.ImageMarkdownSuffix(); imageMarkdown != "" {
		output.WriteString(imageMarkdown)
		if err := writeGrokSSEJSON(c.Writer, grokChatCompletionChunk(responseID, originalModel, imageMarkdown, false)); err != nil {
			return nil, err
		}
		if flusher != nil {
			flusher.Flush()
		}
	}
	if err := writeGrokSSEJSON(c.Writer, grokChatCompletionFinalChunk(responseID, originalModel, adapter.ChatAnnotations(), adapter.SearchSources())); err != nil {
		return nil, err
	}
	if _, err := io.WriteString(c.Writer, "data: [DONE]\n\n"); err != nil {
		return nil, err
	}
	if flusher != nil {
		flusher.Flush()
	}

	outputTokens := estimateGrokTextTokens(output.String())
	reasoningTokens := estimateGrokTextTokens(reasoning.String())
	var firstTokenPtr *int
	if seenFirst {
		firstTokenPtr = &firstTokenMs
	}
	return &OpenAIForwardResult{
		RequestID:        resp.Header.Get("x-request-id"),
		ResponseID:       responseID,
		Usage:            OpenAIUsage{InputTokens: promptTokens, OutputTokens: outputTokens + reasoningTokens},
		Model:            originalModel,
		BillingModel:     billingModel,
		UpstreamModel:    upstreamModel,
		Stream:           true,
		ResponseHeaders:  http.Header{"Content-Type": []string{"text/event-stream"}},
		Duration:         time.Since(startTime),
		FirstTokenMs:     firstTokenPtr,
		ImageOutputSizes: []string{},
	}, nil
}

func (s *OpenAIGatewayService) bufferGrokWebChatCompletions(c *gin.Context, resp *http.Response, responseID, originalModel, billingModel, upstreamModel string, promptTokens int, startTime time.Time) (*OpenAIForwardResult, error) {
	var output strings.Builder
	var reasoning strings.Builder
	adapter := newGrokWebStreamAdapter()
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		events, done, err := adapter.FeedLine(scanner.Text())
		if err != nil {
			return nil, err
		}
		for _, ev := range events {
			switch ev.Kind {
			case "text":
				output.WriteString(ev.Content)
			case "thinking":
				reasoning.WriteString(ev.Content)
			}
		}
		if done {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read grok response: %w", err)
	}
	content := output.String()
	if imageMarkdown := adapter.ImageMarkdownSuffix(); imageMarkdown != "" {
		content += imageMarkdown
	}
	reasoningContent := reasoning.String()
	usage := OpenAIUsage{
		InputTokens:  promptTokens,
		OutputTokens: estimateGrokTextTokens(content) + estimateGrokTextTokens(reasoningContent),
	}
	c.JSON(http.StatusOK, grokChatCompletionResponse(responseID, originalModel, content, reasoningContent, usage, adapter.ChatAnnotations(), adapter.SearchSources()))
	return &OpenAIForwardResult{
		RequestID:        resp.Header.Get("x-request-id"),
		ResponseID:       responseID,
		Usage:            usage,
		Model:            originalModel,
		BillingModel:     billingModel,
		UpstreamModel:    upstreamModel,
		Stream:           false,
		ResponseHeaders:  http.Header{"Content-Type": []string{"application/json"}},
		Duration:         time.Since(startTime),
		ImageOutputSizes: []string{},
	}, nil
}

func (s *OpenAIGatewayService) streamGrokWebAnthropicMessages(c *gin.Context, resp *http.Response, responseID, originalModel, billingModel, upstreamModel string, promptTokens int, startTime time.Time) (*OpenAIForwardResult, error) {
	if s.responseHeaderFilter != nil {
		responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)
	flusher, _ := c.Writer.(http.Flusher)

	// 发送心跳以防止在上游思考期间连接超时
	if _, err := io.WriteString(c.Writer, ": heartbeat\n\n"); err != nil {
		return nil, fmt.Errorf("write initial heartbeat: %w", err)
	}
	if flusher != nil {
		flusher.Flush()
	}

	writeEvent := func(evt apicompat.AnthropicStreamEvent) error {
		sse, err := apicompat.ResponsesAnthropicEventToSSE(evt)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(c.Writer, sse); err != nil {
			return err
		}
		if flusher != nil {
			flusher.Flush()
		}
		return nil
	}

	if err := writeEvent(grokAnthropicMessageStartEvent(responseID, originalModel, promptTokens)); err != nil {
		return nil, err
	}

	var output strings.Builder
	var reasoning strings.Builder
	nextBlockIndex := 0
	textBlockOpen := false
	thinkingBlockOpen := false
	textIndex := -1
	thinkingIndex := -1
	firstTokenMs := 0
	seenFirst := false
	adapter := newGrokWebStreamAdapter()
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		events, done, err := adapter.FeedLine(scanner.Text())
		if err != nil {
			return nil, err
		}
		for _, ev := range events {
			switch ev.Kind {
			case "text":
				if ev.Content == "" {
					continue
				}
				if thinkingBlockOpen {
					if err := writeEvent(grokAnthropicContentBlockStopEvent(thinkingIndex)); err != nil {
						return nil, err
					}
					thinkingBlockOpen = false
				}
				if !seenFirst {
					seenFirst = true
					firstTokenMs = int(time.Since(startTime).Milliseconds())
				}
				if !textBlockOpen {
					textIndex = nextBlockIndex
					nextBlockIndex++
					if err := writeEvent(grokAnthropicContentBlockStartEvent(textIndex, "text")); err != nil {
						return nil, err
					}
					textBlockOpen = true
				}
				output.WriteString(ev.Content)
				if err := writeEvent(grokAnthropicTextDeltaEvent(textIndex, ev.Content)); err != nil {
					return nil, err
				}
			case "thinking":
				if ev.Content == "" {
					continue
				}
				if !thinkingBlockOpen {
					thinkingIndex = nextBlockIndex
					nextBlockIndex++
					if err := writeEvent(grokAnthropicContentBlockStartEvent(thinkingIndex, "thinking")); err != nil {
						return nil, err
					}
					thinkingBlockOpen = true
				}
				reasoning.WriteString(ev.Content)
				if err := writeEvent(grokAnthropicThinkingDeltaEvent(thinkingIndex, ev.Content)); err != nil {
					return nil, err
				}
			}
		}
		if done {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read grok messages stream: %w", err)
	}
	if imageMarkdown := adapter.ImageMarkdownSuffix(); imageMarkdown != "" {
		if thinkingBlockOpen {
			if err := writeEvent(grokAnthropicContentBlockStopEvent(thinkingIndex)); err != nil {
				return nil, err
			}
			thinkingBlockOpen = false
		}
		if !textBlockOpen {
			textIndex = nextBlockIndex
			nextBlockIndex++
			if err := writeEvent(grokAnthropicContentBlockStartEvent(textIndex, "text")); err != nil {
				return nil, err
			}
			textBlockOpen = true
		}
		output.WriteString(imageMarkdown)
		if err := writeEvent(grokAnthropicTextDeltaEvent(textIndex, imageMarkdown)); err != nil {
			return nil, err
		}
	}
	if textBlockOpen {
		if err := writeEvent(grokAnthropicContentBlockStopEvent(textIndex)); err != nil {
			return nil, err
		}
	}
	if thinkingBlockOpen {
		if err := writeEvent(grokAnthropicContentBlockStopEvent(thinkingIndex)); err != nil {
			return nil, err
		}
	}
	content := output.String()
	reasoningContent := reasoning.String()
	usage := OpenAIUsage{
		InputTokens:  promptTokens,
		OutputTokens: estimateGrokTextTokens(content) + estimateGrokTextTokens(reasoningContent),
	}
	if err := writeEvent(grokAnthropicMessageDeltaEvent(usage)); err != nil {
		return nil, err
	}
	if err := writeEvent(apicompat.AnthropicStreamEvent{Type: "message_stop"}); err != nil {
		return nil, err
	}

	var firstTokenPtr *int
	if seenFirst {
		firstTokenPtr = &firstTokenMs
	}
	return &OpenAIForwardResult{
		RequestID:        resp.Header.Get("x-request-id"),
		ResponseID:       responseID,
		Usage:            usage,
		Model:            originalModel,
		BillingModel:     billingModel,
		UpstreamModel:    upstreamModel,
		Stream:           true,
		ResponseHeaders:  http.Header{"Content-Type": []string{"text/event-stream"}},
		Duration:         time.Since(startTime),
		FirstTokenMs:     firstTokenPtr,
		ImageOutputSizes: []string{},
	}, nil
}

func (s *OpenAIGatewayService) bufferGrokWebAnthropicMessages(c *gin.Context, resp *http.Response, responseID, originalModel, billingModel, upstreamModel string, promptTokens int, startTime time.Time) (*OpenAIForwardResult, error) {
	var output strings.Builder
	var reasoning strings.Builder
	adapter := newGrokWebStreamAdapter()
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		events, done, err := adapter.FeedLine(scanner.Text())
		if err != nil {
			return nil, err
		}
		for _, ev := range events {
			switch ev.Kind {
			case "text":
				output.WriteString(ev.Content)
			case "thinking":
				reasoning.WriteString(ev.Content)
			}
		}
		if done {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read grok messages response: %w", err)
	}
	content := output.String()
	if imageMarkdown := adapter.ImageMarkdownSuffix(); imageMarkdown != "" {
		content += imageMarkdown
	}
	reasoningContent := reasoning.String()
	usage := OpenAIUsage{
		InputTokens:  promptTokens,
		OutputTokens: estimateGrokTextTokens(content) + estimateGrokTextTokens(reasoningContent),
	}
	c.JSON(http.StatusOK, grokAnthropicMessageResponse(responseID, originalModel, content, reasoningContent, usage))
	return &OpenAIForwardResult{
		RequestID:        resp.Header.Get("x-request-id"),
		ResponseID:       responseID,
		Usage:            usage,
		Model:            originalModel,
		BillingModel:     billingModel,
		UpstreamModel:    upstreamModel,
		Stream:           false,
		ResponseHeaders:  http.Header{"Content-Type": []string{"application/json"}},
		Duration:         time.Since(startTime),
		ImageOutputSizes: []string{},
	}, nil
}

func (s *OpenAIGatewayService) streamGrokWebResponses(c *gin.Context, resp *http.Response, account *Account, responseID, messageID, originalModel, billingModel, upstreamModel string, promptTokens int, startTime time.Time) (*OpenAIForwardResult, error) {
	if s.responseHeaderFilter != nil {
		responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)
	flusher, _ := c.Writer.(http.Flusher)

	// 发送心跳以防止在上游思考期间连接超时
	if _, err := io.WriteString(c.Writer, ": heartbeat\n\n"); err != nil {
		return nil, fmt.Errorf("write initial heartbeat: %w", err)
	}
	if flusher != nil {
		flusher.Flush()
	}

	writeEvent := func(v any) error {
		if err := writeGrokSSEJSON(c.Writer, v); err != nil {
			return err
		}
		if flusher != nil {
			flusher.Flush()
		}
		return nil
	}
	if err := writeEvent(grokResponsesCreatedEvent(responseID, originalModel)); err != nil {
		return nil, err
	}
	if err := writeEvent(grokResponsesOutputItemAddedEvent(responseID, messageID)); err != nil {
		return nil, err
	}
	if err := writeEvent(grokResponsesContentPartAddedEvent(responseID, messageID)); err != nil {
		return nil, err
	}

	var output strings.Builder
	var reasoning strings.Builder
	firstTokenMs := 0
	seenFirst := false
	adapter := newGrokWebStreamAdapter()
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		events, done, err := adapter.FeedLine(scanner.Text())
		if err != nil {
			return nil, err
		}
		for _, ev := range events {
			switch ev.Kind {
			case "text":
				if ev.Content == "" {
					continue
				}
				if !seenFirst {
					seenFirst = true
					firstTokenMs = int(time.Since(startTime).Milliseconds())
				}
				output.WriteString(ev.Content)
				if err := writeEvent(grokResponsesTextDeltaEvent(responseID, messageID, ev.Content)); err != nil {
					return nil, err
				}
			case "thinking":
				reasoning.WriteString(ev.Content)
			}
		}
		if done {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read grok responses stream: %w", err)
	}
	if imageMarkdown := adapter.ImageMarkdownSuffix(); imageMarkdown != "" {
		output.WriteString(imageMarkdown)
		if err := writeEvent(grokResponsesTextDeltaEvent(responseID, messageID, imageMarkdown)); err != nil {
			return nil, err
		}
	}
	content := output.String()
	reasoningContent := reasoning.String()
	usage := OpenAIUsage{
		InputTokens:  promptTokens,
		OutputTokens: estimateGrokTextTokens(content) + estimateGrokTextTokens(reasoningContent),
	}
	response := grokResponsesResponse(responseID, messageID, originalModel, content, reasoningContent, usage, adapter.ChatAnnotations(), adapter.SearchSources())
	if err := writeEvent(grokResponsesTextDoneEvent(responseID, messageID, content)); err != nil {
		return nil, err
	}
	if err := writeEvent(grokResponsesContentPartDoneEvent(responseID, messageID, content, adapter.ChatAnnotations())); err != nil {
		return nil, err
	}
	if err := writeEvent(grokResponsesOutputItemDoneEvent(responseID, messageID, content, adapter.ChatAnnotations())); err != nil {
		return nil, err
	}
	if err := writeEvent(grokResponsesCompletedEvent(response)); err != nil {
		return nil, err
	}
	if _, err := io.WriteString(c.Writer, "data: [DONE]\n\n"); err != nil {
		return nil, err
	}
	if flusher != nil {
		flusher.Flush()
	}

	var firstTokenPtr *int
	if seenFirst {
		firstTokenPtr = &firstTokenMs
	}
	return &OpenAIForwardResult{
		RequestID:        resp.Header.Get("x-request-id"),
		ResponseID:       responseID,
		Usage:            usage,
		Model:            originalModel,
		BillingModel:     billingModel,
		UpstreamModel:    upstreamModel,
		Stream:           true,
		ResponseHeaders:  http.Header{"Content-Type": []string{"text/event-stream"}},
		Duration:         time.Since(startTime),
		FirstTokenMs:     firstTokenPtr,
		ImageOutputSizes: []string{},
	}, nil
}

func (s *OpenAIGatewayService) bufferGrokWebResponses(c *gin.Context, resp *http.Response, responseID, messageID, originalModel, billingModel, upstreamModel string, promptTokens int, startTime time.Time) (*OpenAIForwardResult, error) {
	var output strings.Builder
	var reasoning strings.Builder
	adapter := newGrokWebStreamAdapter()
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		events, done, err := adapter.FeedLine(scanner.Text())
		if err != nil {
			return nil, err
		}
		for _, ev := range events {
			switch ev.Kind {
			case "text":
				output.WriteString(ev.Content)
			case "thinking":
				reasoning.WriteString(ev.Content)
			}
		}
		if done {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read grok responses response: %w", err)
	}
	content := output.String()
	if imageMarkdown := adapter.ImageMarkdownSuffix(); imageMarkdown != "" {
		content += imageMarkdown
	}
	reasoningContent := reasoning.String()
	usage := OpenAIUsage{
		InputTokens:  promptTokens,
		OutputTokens: estimateGrokTextTokens(content) + estimateGrokTextTokens(reasoningContent),
	}
	c.JSON(http.StatusOK, grokResponsesResponse(responseID, messageID, originalModel, content, reasoningContent, usage, adapter.ChatAnnotations(), adapter.SearchSources()))
	return &OpenAIForwardResult{
		RequestID:        resp.Header.Get("x-request-id"),
		ResponseID:       responseID,
		Usage:            usage,
		Model:            originalModel,
		BillingModel:     billingModel,
		UpstreamModel:    upstreamModel,
		Stream:           false,
		ResponseHeaders:  http.Header{"Content-Type": []string{"application/json"}},
		Duration:         time.Since(startTime),
		ImageOutputSizes: []string{},
	}, nil
}

func (s *OpenAIGatewayService) streamGrokWebImages(ctx context.Context, c *gin.Context, resp *http.Response, account *Account, requestModel, billingModel, upstreamModel string, parsed *OpenAIImagesRequest, prompt string, startTime time.Time) (*OpenAIForwardResult, error) {
	if s.responseHeaderFilter != nil {
		responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)
	flusher, _ := c.Writer.(http.Flusher)

	adapter, err := readGrokWebStream(resp.Body)
	if err != nil {
		return nil, err
	}
	images, err := s.resolveGrokImageOutputs(ctx, account, adapter.imageURLs, parsed)
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, errors.New("grok images response did not include generated images")
	}
	firstTokenMs := int(time.Since(startTime).Milliseconds())
	for idx, image := range images {
		event := map[string]any{
			"type":           "image_generation.completed",
			"id":             fmt.Sprintf("grok_img_%d", idx),
			"created":        time.Now().Unix(),
			"revised_prompt": prompt,
			"size":           grokImagesOutputSize(parsed),
		}
		for key, value := range image {
			event[key] = value
		}
		if err := writeGrokSSEJSON(c.Writer, event); err != nil {
			return nil, err
		}
		if flusher != nil {
			flusher.Flush()
		}
	}
	if _, err := io.WriteString(c.Writer, "data: [DONE]\n\n"); err != nil {
		return nil, err
	}
	if flusher != nil {
		flusher.Flush()
	}
	return &OpenAIForwardResult{
		RequestID:        resp.Header.Get("x-request-id"),
		Usage:            OpenAIUsage{InputTokens: estimateGrokTextTokens(prompt), OutputTokens: len(images)},
		Model:            requestModel,
		BillingModel:     billingModel,
		UpstreamModel:    upstreamModel,
		Stream:           true,
		ResponseHeaders:  http.Header{"Content-Type": []string{"text/event-stream"}},
		Duration:         time.Since(startTime),
		FirstTokenMs:     &firstTokenMs,
		ImageCount:       len(images),
		ImageSize:        parsed.SizeTier,
		ImageInputSize:   parsed.Size,
		ImageOutputSizes: grokImagesOutputSizes(parsed, len(images)),
	}, nil
}

func (s *OpenAIGatewayService) bufferGrokWebImages(ctx context.Context, c *gin.Context, resp *http.Response, account *Account, requestModel, billingModel, upstreamModel string, parsed *OpenAIImagesRequest, prompt string, startTime time.Time) (*OpenAIForwardResult, error) {
	adapter, err := readGrokWebStream(resp.Body)
	if err != nil {
		return nil, err
	}
	images, err := s.resolveGrokImageOutputs(ctx, account, adapter.imageURLs, parsed)
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, errors.New("grok images response did not include generated images")
	}
	data := make([]any, 0, len(images))
	for _, image := range images {
		item := map[string]any{
			"revised_prompt": prompt,
			"size":           grokImagesOutputSize(parsed),
		}
		for key, value := range image {
			item[key] = value
		}
		data = append(data, item)
	}
	body := map[string]any{
		"created": time.Now().Unix(),
		"data":    data,
		"usage": map[string]any{
			"input_tokens":  estimateGrokTextTokens(prompt),
			"output_tokens": len(images),
			"total_tokens":  estimateGrokTextTokens(prompt) + len(images),
		},
	}
	c.JSON(http.StatusOK, body)
	return &OpenAIForwardResult{
		RequestID:        resp.Header.Get("x-request-id"),
		Usage:            OpenAIUsage{InputTokens: estimateGrokTextTokens(prompt), OutputTokens: len(images)},
		Model:            requestModel,
		BillingModel:     billingModel,
		UpstreamModel:    upstreamModel,
		Stream:           false,
		ResponseHeaders:  http.Header{"Content-Type": []string{"application/json"}},
		Duration:         time.Since(startTime),
		ImageCount:       len(images),
		ImageSize:        parsed.SizeTier,
		ImageInputSize:   parsed.Size,
		ImageOutputSizes: grokImagesOutputSizes(parsed, len(images)),
	}, nil
}

func readGrokWebStream(r io.Reader) (*grokWebStreamAdapter, error) {
	adapter := newGrokWebStreamAdapter()
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		_, done, err := adapter.FeedLine(scanner.Text())
		if err != nil {
			return adapter, err
		}
		if done {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return adapter, fmt.Errorf("read grok image stream: %w", err)
	}
	return adapter, nil
}

func newGrokWebStreamAdapter() *grokWebStreamAdapter {
	return &grokWebStreamAdapter{
		cardCache:         make(map[string]string),
		citationMap:       make(map[string]int),
		lastCitationIndex: -1,
		searchSourceURLs:  make(map[string]struct{}),
	}
}

func (a *grokWebStreamAdapter) FeedLine(line string) ([]grokStreamEvent, bool, error) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "event:") {
		return nil, false, nil
	}
	if strings.HasPrefix(line, "data:") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	}
	if line == "[DONE]" {
		return nil, true, nil
	}
	if !strings.HasPrefix(line, "{") {
		return nil, false, nil
	}
	if errObj := gjson.Get(line, "error"); errObj.Exists() {
		message := strings.TrimSpace(grokFirstNonEmpty(errObj.Get("message").String(), errObj.Get("error").String(), "Grok stream error"))
		status := http.StatusBadGateway
		lower := strings.ToLower(message)
		if errObj.Get("code").Int() == 8 || strings.Contains(lower, "rate limit") || strings.Contains(lower, "too many requests") {
			status = http.StatusTooManyRequests
		}
		return nil, false, &UpstreamFailoverError{StatusCode: status, ResponseBody: []byte(line)}
	}
	resp := gjson.Get(line, "result.response")
	if !resp.Exists() {
		return nil, false, nil
	}
	events := make([]grokStreamEvent, 0, 2)
	a.collectCardAttachment(resp, &events)
	a.collectSearchSources(resp)

	tag := resp.Get("messageTag").String()
	if tag == "tool_usage_card" {
		if !a.contentStarted {
			if text := a.formatToolUsage(resp); text != "" {
				events = append(events, grokStreamEvent{Kind: "thinking", Content: text})
			}
		}
		return events, false, nil
	}
	if tag == "raw_function_result" {
		return events, false, nil
	}
	if resp.Get("isSoftStop").Bool() || resp.Get("finalMetadata").Exists() {
		events = append(events, grokStreamEvent{Kind: "soft_stop"})
		return events, true, nil
	}
	token := resp.Get("token")
	if !token.Exists() {
		return events, false, nil
	}
	if resp.Get("isThinking").Bool() {
		content := cleanGrokWebToken(token.String())
		if content != "" {
			events = append(events, grokStreamEvent{Kind: "thinking", Content: content})
		}
		return events, false, nil
	}
	if tag == "final" {
		a.contentStarted = true
		content, annotations := a.cleanToken(token.String())
		if content != "" {
			events = append(events, grokStreamEvent{Kind: "text", Content: content})
			for _, ann := range annotations {
				a.annotations = append(a.annotations, ann)
				events = append(events, grokStreamEvent{Kind: "annotation", Annotation: ann})
			}
			a.textOffset += len([]rune(content))
		}
	}
	return events, false, nil
}

func (a *grokWebStreamAdapter) collectCardAttachment(resp gjson.Result, events *[]grokStreamEvent) {
	cardRaw := resp.Get("cardAttachment.jsonData").String()
	if cardRaw == "" {
		return
	}
	card := gjson.Parse(cardRaw)
	cardID := card.Get("id").String()
	if cardID != "" {
		a.cardCache[cardID] = cardRaw
	}
	chunk := card.Get("image_chunk")
	if !chunk.Exists() || chunk.Get("moderated").Bool() || chunk.Get("progress").Int() != 100 {
		return
	}
	imagePath := strings.TrimSpace(chunk.Get("imageUrl").String())
	if imagePath == "" {
		return
	}
	imageURL := imagePath
	if !strings.HasPrefix(imageURL, "http://") && !strings.HasPrefix(imageURL, "https://") {
		imageURL = grokWebAssetsBaseURL + strings.TrimLeft(imageURL, "/")
	}
	image := grokGeneratedImage{URL: imageURL, ID: chunk.Get("imageUuid").String()}
	a.imageURLs = append(a.imageURLs, image)
	*events = append(*events, grokStreamEvent{Kind: "image", Content: imageURL})
}

func (a *grokWebStreamAdapter) collectSearchSources(resp gjson.Result) {
	for _, item := range resp.Get("webSearchResults.results").Array() {
		url := strings.TrimSpace(item.Get("url").String())
		if url == "" {
			continue
		}
		title := grokFirstNonEmpty(item.Get("title").String(), url)
		a.addSearchSource(url, title, "web")
	}
	for _, item := range resp.Get("xSearchResults.results").Array() {
		postID := strings.TrimSpace(item.Get("postId").String())
		username := strings.TrimSpace(item.Get("username").String())
		if postID == "" || username == "" {
			continue
		}
		url := fmt.Sprintf("https://x.com/%s/status/%s", username, postID)
		text := whitespaceRE.ReplaceAllString(strings.TrimSpace(item.Get("text").String()), " ")
		title := "X/@" + username
		if text != "" {
			if len([]rune(text)) > 50 {
				runes := []rune(text)
				text = string(runes[:50]) + "..."
			}
			title += ": " + text
		}
		a.addSearchSource(url, title, "x_post")
	}
}

func (a *grokWebStreamAdapter) addSearchSource(url, title, sourceType string) {
	if _, exists := a.searchSourceURLs[url]; exists {
		return
	}
	a.searchSourceURLs[url] = struct{}{}
	a.searchSources = append(a.searchSources, grokSearchSource{
		URL:   url,
		Title: grokFirstNonEmpty(title, url),
		Type:  grokFirstNonEmpty(sourceType, "web"),
	})
}

func (a *grokWebStreamAdapter) cleanToken(token string) (string, []grokAnnotation) {
	if !strings.Contains(token, "<grok:render") {
		return token, nil
	}
	cleaned := grokRenderTagDetailRE.ReplaceAllStringFunc(token, func(match string) string {
		parts := grokRenderTagDetailRE.FindStringSubmatch(match)
		if len(parts) < 4 {
			return ""
		}
		return a.renderReplace(parts[1], parts[3])
	})
	cleaned = grokRenderTagRE.ReplaceAllString(cleaned, "")
	if strings.HasPrefix(cleaned, "\n") && strings.Contains(cleaned, "[[") {
		cleaned = strings.TrimLeft(cleaned, "\n")
	}
	annotations := a.resolvePendingCitations(cleaned)
	return cleaned, annotations
}

func (a *grokWebStreamAdapter) renderReplace(cardID, renderType string) string {
	cardRaw := a.cardCache[cardID]
	if cardRaw == "" {
		return ""
	}
	card := gjson.Parse(cardRaw)
	switch renderType {
	case "render_searched_image":
		img := card.Get("image")
		title := grokFirstNonEmpty(img.Get("title").String(), "image")
		thumb := grokFirstNonEmpty(img.Get("thumbnail").String(), img.Get("original").String())
		link := strings.TrimSpace(img.Get("link").String())
		if thumb == "" {
			return ""
		}
		if link != "" {
			return fmt.Sprintf("[![%s](%s)](%s)", escapeGrokMarkdownText(title), thumb, link)
		}
		return fmt.Sprintf("![%s](%s)", escapeGrokMarkdownText(title), thumb)
	case "render_generated_image":
		return ""
	case "render_inline_citation":
		url := strings.TrimSpace(card.Get("url").String())
		if url == "" {
			return ""
		}
		index, ok := a.citationMap[url]
		if !ok {
			a.citationOrder = append(a.citationOrder, url)
			index = len(a.citationOrder)
			a.citationMap[url] = index
		}
		if index == a.lastCitationIndex {
			return ""
		}
		a.lastCitationIndex = index
		citationText := fmt.Sprintf(" [[%d]](%s)", index, url)
		title := strings.TrimSpace(card.Get("title").String())
		if title == "" {
			for _, source := range a.searchSources {
				if source.URL == url {
					title = source.Title
					break
				}
			}
		}
		a.pendingCitations = append(a.pendingCitations, grokPendingCitation{
			URL:    url,
			Title:  grokFirstNonEmpty(title, url),
			Needle: citationText,
		})
		return citationText
	default:
		return ""
	}
}

func (a *grokWebStreamAdapter) resolvePendingCitations(cleaned string) []grokAnnotation {
	if len(a.pendingCitations) == 0 || cleaned == "" {
		a.pendingCitations = nil
		return nil
	}
	annotations := make([]grokAnnotation, 0, len(a.pendingCitations))
	searchStart := 0
	for _, cite := range a.pendingCitations {
		pos := strings.Index(cleaned[searchStart:], cite.Needle)
		if pos < 0 {
			continue
		}
		startByte := searchStart + pos
		endByte := startByte + len(cite.Needle)
		startIndex := len([]rune(cleaned[:startByte]))
		endIndex := len([]rune(cleaned[:endByte]))
		annotations = append(annotations, grokAnnotation{
			URL:        cite.URL,
			Title:      cite.Title,
			StartIndex: a.textOffset + startIndex,
			EndIndex:   a.textOffset + endIndex,
		})
		searchStart = endByte
	}
	a.pendingCitations = nil
	return annotations
}

func (a *grokWebStreamAdapter) formatToolUsage(resp gjson.Result) string {
	card := resp.Get("toolUsageCard")
	if !card.Exists() {
		return ""
	}
	for key, value := range card.Map() {
		if key == "toolUsageCardId" || value.Type != gjson.JSON || !strings.HasPrefix(strings.TrimSpace(value.Raw), "{") {
			continue
		}
		toolName := camelToSnake(key)
		args := value.Get("args")
		display := grokFirstNonEmpty(args.Get("query").String(), args.Get("q").String(), args.Get("url").String(), args.Get("imageDescription").String(), args.Get("image_description").String())
		if display != "" {
			return fmt.Sprintf("%s: %s\n", toolName, display)
		}
		return toolName + "\n"
	}
	return ""
}

func (a *grokWebStreamAdapter) ImageMarkdownSuffix() string {
	if len(a.imageURLs) == 0 {
		return ""
	}
	lines := make([]string, 0, len(a.imageURLs))
	for _, image := range a.imageURLs {
		if strings.TrimSpace(image.URL) != "" {
			lines = append(lines, "![image]("+image.URL+")")
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return "\n\n" + strings.Join(lines, "\n")
}

func (a *grokWebStreamAdapter) ChatAnnotations() []map[string]any {
	if len(a.annotations) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(a.annotations))
	for _, ann := range a.annotations {
		out = append(out, map[string]any{
			"type": "url_citation",
			"url_citation": map[string]any{
				"url":         ann.URL,
				"title":       ann.Title,
				"start_index": ann.StartIndex,
				"end_index":   ann.EndIndex,
			},
		})
	}
	return out
}

func (a *grokWebStreamAdapter) SearchSources() []map[string]any {
	if len(a.searchSources) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(a.searchSources))
	for _, source := range a.searchSources {
		out = append(out, map[string]any{
			"url":   source.URL,
			"title": source.Title,
			"type":  source.Type,
		})
	}
	return out
}

func parseGrokWebStreamLine(line string) (grokStreamEvent, bool, bool, error) {
	events, done, err := newGrokWebStreamAdapter().FeedLine(line)
	if err != nil {
		return grokStreamEvent{}, false, false, err
	}
	for _, ev := range events {
		if ev.Kind == "text" || ev.Kind == "thinking" {
			return ev, true, done, nil
		}
	}
	return grokStreamEvent{}, false, done, nil
}

func flattenGrokChatMessages(body []byte) (string, []grokWebFileInput, error) {
	messages := gjson.GetBytes(body, "messages")
	if !messages.Exists() {
		if input := gjson.GetBytes(body, "input"); input.Exists() {
			prompt, files := flattenGrokResponsesInput(input)
			return prompt, files, nil
		}
		return "", nil, errors.New("messages is required")
	}
	if !messages.IsArray() {
		return "", nil, errors.New("messages must be an array")
	}
	var parts []string
	var files []grokWebFileInput
	for _, msg := range messages.Array() {
		role := strings.TrimSpace(msg.Get("role").String())
		if role == "" {
			role = "user"
		}
		if toolCalls := msg.Get("tool_calls"); toolCalls.Exists() {
			parts = append(parts, fmt.Sprintf("[%s tool calls]: %s", role, toolCalls.Raw))
			continue
		}
		content, contentFiles := flattenGrokMessageContent(msg.Get("content"))
		files = append(files, contentFiles...)
		if content == "" {
			continue
		}
		if role == "tool" {
			if id := strings.TrimSpace(msg.Get("tool_call_id").String()); id != "" {
				parts = append(parts, fmt.Sprintf("[tool result for %s]:\n%s", id, content))
			} else {
				parts = append(parts, "[tool result]:\n"+content)
			}
			continue
		}
		parts = append(parts, fmt.Sprintf("[%s]: %s", role, content))
	}
	prompt := strings.TrimSpace(strings.Join(parts, "\n\n"))
	if prompt == "" && len(files) == 0 {
		return "", nil, errors.New("messages must contain text content")
	}
	return prompt, files, nil
}

func flattenGrokResponsesInput(input gjson.Result) (string, []grokWebFileInput) {
	if input.Type == gjson.String {
		return strings.TrimSpace(input.String()), nil
	}
	var parts []string
	var files []grokWebFileInput
	if input.IsArray() {
		for _, item := range input.Array() {
			if text, contentFiles := flattenGrokMessageContent(item.Get("content")); text != "" || len(contentFiles) > 0 {
				role := grokFirstNonEmpty(item.Get("role").String(), "user")
				if text != "" {
					parts = append(parts, fmt.Sprintf("[%s]: %s", role, text))
				}
				files = append(files, contentFiles...)
			} else if item.Type == gjson.String {
				parts = append(parts, item.String())
			}
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n")), files
}

func flattenGrokMessageContent(content gjson.Result) (string, []grokWebFileInput) {
	switch content.Type {
	case gjson.String:
		return strings.TrimSpace(content.String()), nil
	case gjson.JSON:
		if content.IsArray() {
			var parts []string
			var files []grokWebFileInput
			for _, block := range content.Array() {
				switch block.Get("type").String() {
				case "text", "input_text":
					if text := strings.TrimSpace(block.Get("text").String()); text != "" {
						parts = append(parts, text)
					}
				case "image_url":
					if url := strings.TrimSpace(block.Get("image_url.url").String()); url != "" {
						if file, err := parseGrokWebDataURI(url); err == nil {
							files = append(files, file)
						} else {
							parts = append(parts, "[image] "+url)
						}
					}
				case "input_image":
					if url := strings.TrimSpace(block.Get("image_url").String()); url != "" {
						if file, err := parseGrokWebDataURI(url); err == nil {
							files = append(files, file)
						} else {
							parts = append(parts, "[image] "+url)
						}
					}
				case "image":
					if source := block.Get("source"); source.Exists() {
						mime := grokFirstNonEmpty(source.Get("media_type").String(), "application/octet-stream")
						data := strings.TrimSpace(source.Get("data").String())
						if data != "" {
							if file, err := parseGrokWebDataURI("data:" + mime + ";base64," + data); err == nil {
								files = append(files, file)
							}
						}
					}
				case "file":
					data := grokFirstNonEmpty(block.Get("file.file_data").String(), block.Get("file.data").String())
					if file, err := parseGrokWebDataURI(data); err == nil {
						files = append(files, file)
					}
				case "input_audio":
					data := grokFirstNonEmpty(block.Get("input_audio.data").String(), block.Get("input_audio.file_data").String())
					if file, err := parseGrokWebDataURI(data); err == nil {
						files = append(files, file)
					}
				}
			}
			return strings.TrimSpace(strings.Join(parts, "\n")), files
		}
	}
	return "", nil
}

func parseGrokWebDataURI(input string) (grokWebFileInput, error) {
	input = strings.TrimSpace(input)
	if !strings.HasPrefix(input, "data:") {
		return grokWebFileInput{}, errors.New("not a data uri")
	}
	header, content, ok := strings.Cut(input, ",")
	if !ok {
		return grokWebFileInput{}, errors.New("malformed data uri")
	}
	if !strings.Contains(strings.ToLower(header), ";base64") {
		return grokWebFileInput{}, errors.New("data uri must be base64 encoded")
	}
	mime := strings.TrimSpace(strings.TrimPrefix(strings.SplitN(header, ";", 2)[0], "data:"))
	if mime == "" {
		mime = "application/octet-stream"
	}
	content = whitespaceRE.ReplaceAllString(content, "")
	if content == "" {
		return grokWebFileInput{}, errors.New("empty data uri")
	}
	decoded, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return grokWebFileInput{}, fmt.Errorf("invalid base64 data uri: %w", err)
	}
	if len(decoded) > grokWebMaxUploadBytes {
		return grokWebFileInput{}, fmt.Errorf("data uri exceeds %d bytes", grokWebMaxUploadBytes)
	}
	ext := "bin"
	if slash := strings.LastIndex(mime, "/"); slash >= 0 && slash+1 < len(mime) {
		ext = sanitizeGrokFilenameExt(mime[slash+1:])
	}
	return grokWebFileInput{
		Filename: "file." + ext,
		Mime:     mime,
		Content:  content,
	}, nil
}

func sanitizeGrokFilenameExt(ext string) string {
	ext = strings.ToLower(strings.TrimSpace(ext))
	if ext == "" {
		return "bin"
	}
	var b strings.Builder
	for _, r := range ext {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "bin"
	}
	return b.String()
}

func buildGrokImagesPromptAndFiles(parsed *OpenAIImagesRequest) (string, []grokWebFileInput) {
	if parsed == nil {
		return "", nil
	}
	prompt := strings.TrimSpace(parsed.Prompt)
	if prompt == "" {
		prompt = "Generate an image."
	}
	files := make([]grokWebFileInput, 0, len(parsed.InputImageURLs)+len(parsed.Uploads)+1)
	var refs []string
	for _, imageURL := range parsed.InputImageURLs {
		imageURL = strings.TrimSpace(imageURL)
		if imageURL == "" {
			continue
		}
		if file, err := parseGrokWebDataURI(imageURL); err == nil {
			files = append(files, file)
			continue
		}
		refs = append(refs, "[reference image] "+imageURL)
	}
	for _, upload := range parsed.Uploads {
		if len(upload.Data) == 0 {
			continue
		}
		mimeType := strings.TrimSpace(upload.ContentType)
		if mimeType == "" {
			mimeType = http.DetectContentType(upload.Data)
		}
		filename := strings.TrimSpace(upload.FileName)
		if filename == "" {
			filename = "image." + sanitizeGrokFilenameExt(grokMimeExt(mimeType))
		}
		files = append(files, grokWebFileInput{
			Filename: filename,
			Mime:     mimeType,
			Content:  base64.StdEncoding.EncodeToString(upload.Data),
		})
	}
	if parsed.MaskUpload != nil && len(parsed.MaskUpload.Data) > 0 {
		mimeType := strings.TrimSpace(parsed.MaskUpload.ContentType)
		if mimeType == "" {
			mimeType = http.DetectContentType(parsed.MaskUpload.Data)
		}
		filename := strings.TrimSpace(parsed.MaskUpload.FileName)
		if filename == "" {
			filename = "mask." + sanitizeGrokFilenameExt(grokMimeExt(mimeType))
		}
		files = append(files, grokWebFileInput{
			Filename: filename,
			Mime:     mimeType,
			Content:  base64.StdEncoding.EncodeToString(parsed.MaskUpload.Data),
		})
		refs = append(refs, "[mask image attached]")
	}
	if maskURL := strings.TrimSpace(parsed.MaskImageURL); maskURL != "" {
		if file, err := parseGrokWebDataURI(maskURL); err == nil {
			files = append(files, file)
			refs = append(refs, "[mask image attached]")
		} else {
			refs = append(refs, "[mask image] "+maskURL)
		}
	}
	if len(refs) > 0 {
		prompt = strings.TrimSpace(prompt + "\n\n" + strings.Join(refs, "\n"))
	}
	if parsed.IsEdits() && len(files) > 0 {
		prompt = "Use the attached image(s) as reference and apply this edit:\n" + prompt
	}
	return prompt, files
}

func grokMimeExt(mimeType string) string {
	mimeType = strings.TrimSpace(mimeType)
	if slash := strings.LastIndex(mimeType, "/"); slash >= 0 && slash+1 < len(mimeType) {
		return mimeType[slash+1:]
	}
	return "bin"
}

func setGrokWebImageGenerationCount(body []byte, count int) ([]byte, error) {
	if count <= 0 {
		count = 1
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	payload["imageGenerationCount"] = count
	return json.Marshal(payload)
}

func (s *OpenAIGatewayService) resolveGrokImageOutputs(ctx context.Context, account *Account, images []grokGeneratedImage, parsed *OpenAIImagesRequest) ([]map[string]any, error) {
	if len(images) == 0 {
		return nil, nil
	}
	wantB64 := parsed != nil && strings.EqualFold(strings.TrimSpace(parsed.ResponseFormat), "b64_json")
	out := make([]map[string]any, 0, len(images))
	for _, image := range images {
		url := strings.TrimSpace(image.URL)
		if url == "" {
			continue
		}
		if !wantB64 {
			localURL, err := s.cacheGrokImageAsLocalURL(ctx, account, image)
			if err != nil {
				return nil, err
			}
			out = append(out, map[string]any{"url": localURL})
			continue
		}
		if blob := strings.TrimSpace(image.Blob); blob != "" {
			out = append(out, map[string]any{"b64_json": blob})
			continue
		}
		b64, err := s.downloadGrokImageAsBase64(ctx, account, url)
		if err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"b64_json": b64})
	}
	return out, nil
}

func (s *OpenAIGatewayService) cacheGrokImageAsLocalURL(ctx context.Context, account *Account, image grokGeneratedImage) (string, error) {
	var (
		raw         []byte
		contentType string
		err         error
	)
	if blob := strings.TrimSpace(image.Blob); blob != "" {
		raw, err = base64.StdEncoding.DecodeString(blob)
		if err != nil {
			return "", fmt.Errorf("decode grok image blob: %w", err)
		}
		contentType = grokContentTypeFromImageURL(image.URL)
	} else {
		raw, contentType, err = s.downloadGrokImageBytes(ctx, account, image.URL)
		if err != nil {
			return "", err
		}
	}
	seed := grokFirstNonEmpty(image.ID, image.URL)
	id, err := saveLocalImage(raw, contentType, seed)
	if err != nil {
		return "", err
	}
	return localImageURL(id), nil
}

func (s *OpenAIGatewayService) downloadGrokImageAsBase64(ctx context.Context, account *Account, imageURL string) (string, error) {
	raw, _, err := s.downloadGrokImageBytes(ctx, account, imageURL)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}

func (s *OpenAIGatewayService) downloadGrokImageBytes(ctx context.Context, account *Account, imageURL string) ([]byte, string, error) {
	if !isAllowedGrokAssetURL(imageURL) {
		return nil, "", fmt.Errorf("unsupported Grok image URL: %s", imageURL)
	}
	if s.httpUpstream == nil {
		return nil, "", errors.New("http upstream is not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
	req.Header.Set("User-Agent", grokFirstNonEmpty(account.GetCredential("user_agent"), defaultGrokWebUserAgent()))
	req.Header.Set("Cookie", buildGrokWebCookieHeader(account))
	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.doGrokWebRequest(req, account, proxyURL)
	if err != nil {
		return nil, "", err
	}
	if resp == nil {
		return nil, "", errors.New("grok image download returned empty response")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return nil, "", fmt.Errorf("grok image download returned %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, openAIImageMaxDownloadBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(raw) > openAIImageMaxDownloadBytes {
		return nil, "", fmt.Errorf("grok image download exceeded %d bytes", openAIImageMaxDownloadBytes)
	}
	return raw, responseContentType(resp, "image/jpeg"), nil
}

func isAllowedGrokAssetURL(imageURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(imageURL))
	if err != nil || parsed.Scheme != "https" {
		return false
	}
	return strings.EqualFold(parsed.Hostname(), "assets.grok.com")
}

func grokContentTypeFromImageURL(imageURL string) string {
	lower := strings.ToLower(strings.TrimSpace(imageURL))
	switch {
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".webp"):
		return "image/webp"
	case strings.HasSuffix(lower, ".gif"):
		return "image/gif"
	case strings.HasSuffix(lower, ".bmp"):
		return "image/bmp"
	default:
		return "image/jpeg"
	}
}

func grokImagesOutputSize(parsed *OpenAIImagesRequest) string {
	if parsed == nil {
		return ""
	}
	if size := strings.TrimSpace(parsed.Size); size != "" {
		return size
	}
	return parsed.SizeTier
}

func grokImagesOutputSizes(parsed *OpenAIImagesRequest, count int) []string {
	size := grokImagesOutputSize(parsed)
	if size == "" || count <= 0 {
		return nil
	}
	out := make([]string, count)
	for i := range out {
		out[i] = size
	}
	return out
}

func buildGrokImagineWSHeaders(account *Account, cfg *config.Config) (http.Header, error) {
	if account == nil {
		return nil, errors.New("account is required")
	}
	cookie := buildGrokWebCookieHeader(account)
	if cookie == "" {
		return nil, errors.New("sso_token or cookie is required for Grok Cookie account")
	}
	headers := http.Header{}
	headers.Set("Accept-Language", grokFirstNonEmpty(account.GetCredential("accept_language"), "zh-CN,zh;q=0.9,en;q=0.8"))
	headers.Set("Cookie", cookie)
	headers.Set("Origin", grokFirstNonEmpty(account.GetCredential("origin"), grokWebDefaultBaseURL))
	headers.Set("Referer", grokFirstNonEmpty(account.GetCredential("referer"), "https://grok.com/imagine"))
	headers.Set("User-Agent", grokFirstNonEmpty(account.GetCredential("user_agent"), defaultGrokWebUserAgent()))
	applyGrokWebCompatibilityHeaders(headers, account, cfg)
	return headers, nil
}

func buildGrokImagineResetMessage() map[string]any {
	return map[string]any{
		"type":      "conversation.item.create",
		"timestamp": time.Now().UnixMilli(),
		"item": map[string]any{
			"type":    "message",
			"content": []map[string]any{{"type": "reset"}},
		},
	}
}

func buildGrokImagineRequestMessage(requestID, prompt, aspectRatio string, enablePro bool) map[string]any {
	return map[string]any{
		"type":      "conversation.item.create",
		"timestamp": time.Now().UnixMilli(),
		"item": map[string]any{
			"type": "message",
			"content": []map[string]any{{
				"requestId": requestID,
				"text":      prompt,
				"type":      "input_text",
				"properties": map[string]any{
					"section_count":       0,
					"is_kids_mode":        false,
					"enable_nsfw":         true,
					"skip_upsampler":      false,
					"enable_side_by_side": true,
					"is_initial":          false,
					"aspect_ratio":        aspectRatio,
					"enable_pro":          enablePro,
				},
			}},
		},
	}
}

func grokImagineAspectRatio(size string) string {
	switch strings.TrimSpace(strings.ToLower(size)) {
	case "1280x720", "16:9":
		return "16:9"
	case "720x1280", "9:16":
		return "9:16"
	case "1792x1024", "3:2":
		return "3:2"
	case "1024x1792", "2:3":
		return "2:3"
	case "1024x1024", "1:1":
		return "1:1"
	default:
		return "2:3"
	}
}

func grokImagineEnablePro(model string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(model)), "pro")
}

func grokImagineImageIDFromMessage(msg gjson.Result) string {
	if imageID := strings.TrimSpace(grokFirstNonEmpty(msg.Get("image_id").String(), msg.Get("job_id").String())); imageID != "" {
		return imageID
	}
	imageURL := strings.TrimSpace(msg.Get("url").String())
	matches := grokImagineImageURLRE.FindStringSubmatch(imageURL)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func buildGrokWebCookieHeader(account *Account) string {
	fullCookie := sanitizeGrokHeaderValue(account.GetCredential("cookie"))
	sso := grokFirstNonEmpty(
		extractCookieValue(fullCookie, "sso"),
		account.GetCredential("sso_token"),
		account.GetCredential("sso"),
	)
	sso = sanitizeGrokHeaderValue(strings.TrimPrefix(sso, "sso="))
	var cookie string
	if fullCookie != "" {
		cookie = fullCookie
		if sso != "" && extractCookieValue(cookie, "sso-rw") == "" {
			cookie = appendCookiePair(cookie, "sso-rw", sso)
		}
	} else if sso != "" {
		cookie = "sso=" + sso + "; sso-rw=" + sso
	}
	cfCookies := sanitizeGrokHeaderValue(account.GetCredential("cf_cookies"))
	if cfCookies != "" {
		cookie = appendRawCookies(cookie, cfCookies)
	}
	if cfClearance := sanitizeGrokHeaderValue(account.GetCredential("cf_clearance")); cfClearance != "" {
		cookie = upsertCookiePair(cookie, "cf_clearance", cfClearance)
	}
	return strings.Trim(cookie, "; ")
}

func resolveGrokWebModeID(account *Account, model string) string {
	if account != nil {
		if override := strings.TrimSpace(account.GetCredential("mode_id")); override != "" {
			return override
		}
	}
	m := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.Contains(m, "heavy") || strings.Contains(m, "multi-agent"):
		return "heavy"
	case strings.Contains(m, "expert") || strings.Contains(m, "reasoning"):
		return "expert"
	case strings.Contains(m, "fast") || strings.Contains(m, "non-reasoning") || strings.Contains(m, "lite"):
		return "fast"
	case strings.Contains(m, "4.3") || strings.Contains(m, "grok-4-3"):
		return "grok-420-computer-use-sa"
	default:
		return "auto"
	}
}

func grokAnthropicMessageResponse(responseID, model, content, reasoning string, usage OpenAIUsage) map[string]any {
	blocks := make([]map[string]any, 0, 2)
	if strings.TrimSpace(reasoning) != "" {
		blocks = append(blocks, map[string]any{
			"type":     "thinking",
			"thinking": reasoning,
		})
	}
	blocks = append(blocks, map[string]any{
		"type": "text",
		"text": content,
	})
	return map[string]any{
		"id":            responseID,
		"type":          "message",
		"role":          "assistant",
		"model":         model,
		"content":       blocks,
		"stop_reason":   "end_turn",
		"stop_sequence": nil,
		"usage": map[string]any{
			"input_tokens":                usage.InputTokens,
			"output_tokens":               usage.OutputTokens,
			"cache_creation_input_tokens": 0,
			"cache_read_input_tokens":     0,
		},
	}
}

func grokAnthropicMessageStartEvent(responseID, model string, inputTokens int) apicompat.AnthropicStreamEvent {
	return apicompat.AnthropicStreamEvent{
		Type: "message_start",
		Message: &apicompat.AnthropicResponse{
			ID:         responseID,
			Type:       "message",
			Role:       "assistant",
			Content:    []apicompat.AnthropicContentBlock{},
			Model:      model,
			StopReason: "",
			Usage: apicompat.AnthropicUsage{
				InputTokens: inputTokens,
			},
		},
	}
}

func grokAnthropicContentBlockStartEvent(index int, blockType string) apicompat.AnthropicStreamEvent {
	return apicompat.AnthropicStreamEvent{
		Type:  "content_block_start",
		Index: &index,
		ContentBlock: &apicompat.AnthropicContentBlock{
			Type: blockType,
		},
	}
}

func grokAnthropicTextDeltaEvent(index int, text string) apicompat.AnthropicStreamEvent {
	return apicompat.AnthropicStreamEvent{
		Type:  "content_block_delta",
		Index: &index,
		Delta: &apicompat.AnthropicDelta{
			Type: "text_delta",
			Text: text,
		},
	}
}

func grokAnthropicThinkingDeltaEvent(index int, thinking string) apicompat.AnthropicStreamEvent {
	return apicompat.AnthropicStreamEvent{
		Type:  "content_block_delta",
		Index: &index,
		Delta: &apicompat.AnthropicDelta{
			Type:     "thinking_delta",
			Thinking: thinking,
		},
	}
}

func grokAnthropicContentBlockStopEvent(index int) apicompat.AnthropicStreamEvent {
	return apicompat.AnthropicStreamEvent{
		Type:  "content_block_stop",
		Index: &index,
	}
}

func grokAnthropicMessageDeltaEvent(usage OpenAIUsage) apicompat.AnthropicStreamEvent {
	return apicompat.AnthropicStreamEvent{
		Type: "message_delta",
		Delta: &apicompat.AnthropicDelta{
			StopReason: "end_turn",
		},
		Usage: &apicompat.AnthropicUsage{
			OutputTokens: usage.OutputTokens,
		},
	}
}

func grokChatCompletionChunk(responseID, model, content string, final bool) map[string]any {
	choice := map[string]any{
		"index": 0,
		"delta": map[string]any{
			"role":    "assistant",
			"content": content,
		},
	}
	if final {
		choice["finish_reason"] = "stop"
	}
	return map[string]any{
		"id":      responseID,
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []any{choice},
	}
}

func grokChatCompletionThinkingChunk(responseID, model, content string) map[string]any {
	return map[string]any{
		"id":      responseID,
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []any{
			map[string]any{
				"index": 0,
				"delta": map[string]any{
					"role":              "assistant",
					"reasoning_content": content,
				},
			},
		},
	}
}

func grokChatCompletionFinalChunk(responseID, model string, annotations []map[string]any, searchSources []map[string]any) map[string]any {
	chunk := grokChatCompletionChunk(responseID, model, "", true)
	if len(annotations) > 0 {
		if choices, ok := chunk["choices"].([]any); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]any); ok {
				if delta, ok := choice["delta"].(map[string]any); ok {
					delta["annotations"] = annotations
				}
			}
		}
	}
	if len(searchSources) > 0 {
		chunk["search_sources"] = searchSources
	}
	return chunk
}

func grokChatCompletionResponse(responseID, model, content, reasoning string, usage OpenAIUsage, annotations []map[string]any, searchSources []map[string]any) map[string]any {
	message := map[string]any{
		"role":    "assistant",
		"content": content,
	}
	if strings.TrimSpace(reasoning) != "" {
		message["reasoning_content"] = reasoning
	}
	if len(annotations) > 0 {
		message["annotations"] = annotations
	}
	resp := map[string]any{
		"id":      responseID,
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []any{
			map[string]any{
				"index":         0,
				"message":       message,
				"finish_reason": "stop",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     usage.InputTokens,
			"completion_tokens": usage.OutputTokens,
			"total_tokens":      usage.InputTokens + usage.OutputTokens,
			"prompt_tokens_details": map[string]any{
				"cached_tokens": 0,
				"text_tokens":   usage.InputTokens,
				"audio_tokens":  0,
				"image_tokens":  0,
			},
			"completion_tokens_details": map[string]any{
				"text_tokens":      usage.OutputTokens,
				"audio_tokens":     0,
				"reasoning_tokens": 0,
			},
		},
	}
	if len(searchSources) > 0 {
		resp["search_sources"] = searchSources
	}
	return resp
}

func grokResponsesResponse(responseID, messageID, model, content, reasoning string, usage OpenAIUsage, annotations []map[string]any, searchSources []map[string]any) map[string]any {
	textPart := map[string]any{
		"type": "output_text",
		"text": content,
	}
	if len(annotations) > 0 {
		textPart["annotations"] = annotations
	}
	output := []any{
		map[string]any{
			"id":      messageID,
			"type":    "message",
			"status":  "completed",
			"role":    "assistant",
			"content": []any{textPart},
		},
	}
	if strings.TrimSpace(reasoning) != "" {
		output = append([]any{
			map[string]any{
				"id":      "rs_" + messageID,
				"type":    "reasoning",
				"summary": []any{map[string]any{"type": "summary_text", "text": reasoning}},
			},
		}, output...)
	}
	resp := map[string]any{
		"id":                  responseID,
		"object":              "response",
		"created_at":          time.Now().Unix(),
		"status":              "completed",
		"model":               model,
		"parallel_tool_calls": true,
		"output":              output,
		"usage": map[string]any{
			"input_tokens":  usage.InputTokens,
			"output_tokens": usage.OutputTokens,
			"total_tokens":  usage.InputTokens + usage.OutputTokens,
			"input_tokens_details": map[string]any{
				"cached_tokens": 0,
			},
			"output_tokens_details": map[string]any{
				"reasoning_tokens": 0,
			},
		},
	}
	if len(searchSources) > 0 {
		resp["search_sources"] = searchSources
	}
	return resp
}

func grokResponsesCreatedEvent(responseID, model string) map[string]any {
	return map[string]any{
		"type":        "response.created",
		"response_id": responseID,
		"response": map[string]any{
			"id":         responseID,
			"object":     "response",
			"created_at": time.Now().Unix(),
			"status":     "in_progress",
			"model":      model,
			"output":     []any{},
		},
	}
}

func grokResponsesOutputItemAddedEvent(responseID, messageID string) map[string]any {
	return map[string]any{
		"type":         "response.output_item.added",
		"response_id":  responseID,
		"output_index": 0,
		"item": map[string]any{
			"id":      messageID,
			"type":    "message",
			"status":  "in_progress",
			"role":    "assistant",
			"content": []any{},
		},
	}
}

func grokResponsesContentPartAddedEvent(responseID, messageID string) map[string]any {
	return map[string]any{
		"type":          "response.content_part.added",
		"response_id":   responseID,
		"item_id":       messageID,
		"output_index":  0,
		"content_index": 0,
		"part": map[string]any{
			"type":        "output_text",
			"text":        "",
			"annotations": []any{},
		},
	}
}

func grokResponsesTextDeltaEvent(responseID, messageID, delta string) map[string]any {
	return map[string]any{
		"type":          "response.output_text.delta",
		"response_id":   responseID,
		"item_id":       messageID,
		"output_index":  0,
		"content_index": 0,
		"delta":         delta,
	}
}

func grokResponsesTextDoneEvent(responseID, messageID, text string) map[string]any {
	return map[string]any{
		"type":          "response.output_text.done",
		"response_id":   responseID,
		"item_id":       messageID,
		"output_index":  0,
		"content_index": 0,
		"text":          text,
	}
}

func grokResponsesContentPartDoneEvent(responseID, messageID, text string, annotations []map[string]any) map[string]any {
	part := map[string]any{
		"type": "output_text",
		"text": text,
	}
	if len(annotations) > 0 {
		part["annotations"] = annotations
	}
	return map[string]any{
		"type":          "response.content_part.done",
		"response_id":   responseID,
		"item_id":       messageID,
		"output_index":  0,
		"content_index": 0,
		"part":          part,
	}
}

func grokResponsesOutputItemDoneEvent(responseID, messageID, text string, annotations []map[string]any) map[string]any {
	part := map[string]any{
		"type": "output_text",
		"text": text,
	}
	if len(annotations) > 0 {
		part["annotations"] = annotations
	}
	return map[string]any{
		"type":         "response.output_item.done",
		"response_id":  responseID,
		"output_index": 0,
		"item": map[string]any{
			"id":      messageID,
			"type":    "message",
			"status":  "completed",
			"role":    "assistant",
			"content": []any{part},
		},
	}
}

func grokResponsesCompletedEvent(response map[string]any) map[string]any {
	return map[string]any{
		"type":     "response.completed",
		"response": response,
	}
}

func writeGrokSSEJSON(w io.Writer, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", b)
	return err
}

func cleanGrokWebToken(token string) string {
	return grokRenderTagRE.ReplaceAllString(token, "")
}

func escapeGrokMarkdownText(text string) string {
	text = strings.ReplaceAll(text, `\`, `\\`)
	text = strings.ReplaceAll(text, `[`, `\[`)
	text = strings.ReplaceAll(text, `]`, `\]`)
	return text
}

func camelToSnake(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r + ('a' - 'A'))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func estimateGrokTextTokens(text string) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}
	n := len([]rune(text))
	tokens := n / 4
	if tokens < 1 {
		return 1
	}
	return tokens
}

func newGrokChatCompletionID() string {
	return fmt.Sprintf("chatcmpl-grok-%d", time.Now().UnixNano())
}

func newGrokResponseID() string {
	return fmt.Sprintf("resp_grok_%d", time.Now().UnixNano())
}

func newGrokResponseMessageID() string {
	return fmt.Sprintf("msg_grok_%d", time.Now().UnixNano())
}

func newGrokRequestID() string {
	return uuid.NewString()
}

func defaultGrokWebUserAgent() string {
	return "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36"
}

func credentialBool(account *Account, key string, fallback bool) bool {
	if account == nil || account.Credentials == nil {
		return fallback
	}
	v, ok := account.Credentials[key]
	if !ok || v == nil {
		return fallback
	}
	switch x := v.(type) {
	case bool:
		return x
	case string:
		switch strings.ToLower(strings.TrimSpace(x)) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}
	return fallback
}

func grokFirstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func sanitizeGrokHeaderValue(v string) string {
	v = strings.ReplaceAll(v, "\r", "")
	v = strings.ReplaceAll(v, "\n", "")
	return strings.TrimSpace(v)
}

func appendRawCookies(cookie, raw string) string {
	raw = strings.Trim(raw, "; ")
	if raw == "" {
		return strings.Trim(cookie, "; ")
	}
	if strings.TrimSpace(cookie) == "" {
		return raw
	}
	return strings.Trim(cookie, "; ") + "; " + raw
}

func appendCookiePair(cookie, key, value string) string {
	if key == "" || value == "" {
		return strings.Trim(cookie, "; ")
	}
	return appendRawCookies(cookie, key+"="+value)
}

func upsertCookiePair(cookie, key, value string) string {
	cookie = strings.Trim(cookie, "; ")
	if cookie == "" {
		return key + "=" + value
	}
	parts := strings.Split(cookie, ";")
	found := false
	for i, part := range parts {
		p := strings.TrimSpace(part)
		if strings.HasPrefix(p, key+"=") {
			parts[i] = key + "=" + value
			found = true
		} else {
			parts[i] = p
		}
	}
	if !found {
		parts = append(parts, key+"="+value)
	}
	return strings.Join(parts, "; ")
}

func extractCookieValue(cookieHeader, name string) string {
	if cookieHeader == "" || name == "" {
		return ""
	}
	for _, part := range strings.Split(cookieHeader, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, name+"=") {
			return strings.TrimSpace(strings.TrimPrefix(part, name+"="))
		}
	}
	return ""
}

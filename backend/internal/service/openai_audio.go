package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"go.uber.org/zap"
)

type OpenAIAudioEndpoint string

const (
	OpenAIAudioEndpointSpeech         OpenAIAudioEndpoint = "/v1/audio/speech"
	OpenAIAudioEndpointTranscriptions OpenAIAudioEndpoint = "/v1/audio/transcriptions"
	OpenAIAudioEndpointTranslations   OpenAIAudioEndpoint = "/v1/audio/translations"
)

func (e OpenAIAudioEndpoint) Capability() OpenAIEndpointCapability {
	switch e {
	case OpenAIAudioEndpointSpeech:
		return OpenAIEndpointCapabilityAudioSpeech
	case OpenAIAudioEndpointTranscriptions:
		return OpenAIEndpointCapabilityAudioTranscribe
	case OpenAIAudioEndpointTranslations:
		return OpenAIEndpointCapabilityAudioTranslate
	default:
		return ""
	}
}

func (s *OpenAIGatewayService) ForwardAudio(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	contentType string,
	endpoint OpenAIAudioEndpoint,
	defaultMappedModel string,
) (*OpenAIForwardResult, error) {
	startTime := time.Now()
	if account == nil {
		return nil, fmt.Errorf("account is required")
	}
	if account.Type != AccountTypeAPIKey {
		writeOpenAIAudioError(c, http.StatusBadRequest, "invalid_request_error", "Audio API requires an OpenAI-compatible API key account")
		return nil, fmt.Errorf("unsupported account type for audio: %s", account.Type)
	}
	requestModel := extractOpenAIAudioModel(body, contentType)
	if strings.TrimSpace(requestModel) == "" {
		writeOpenAIAudioError(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return nil, fmt.Errorf("missing model in audio request")
	}

	billingModel := resolveOpenAIForwardModel(account, requestModel, defaultMappedModel)
	upstreamModel := normalizeOpenAIModelForUpstream(account, billingModel)
	forwardBody, forwardContentType, err := rewriteOpenAIAudioModel(body, contentType, upstreamModel)
	if err != nil {
		writeOpenAIAudioError(c, http.StatusBadRequest, "invalid_request_error", "Failed to rewrite audio request")
		return nil, err
	}

	logger.L().Debug("openai audio: forwarding",
		zap.Int64("account_id", account.ID),
		zap.String("endpoint", string(endpoint)),
		zap.String("original_model", requestModel),
		zap.String("billing_model", billingModel),
		zap.String("upstream_model", upstreamModel),
	)

	apiKey := account.GetOpenAIApiKey()
	if apiKey == "" {
		return nil, fmt.Errorf("account %d missing api_key", account.ID)
	}
	baseURL := account.GetOpenAIBaseURL()
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	validatedURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base_url: %w", err)
	}
	targetURL := buildOpenAIAudioURL(validatedURL, endpoint)

	upstreamCtx, releaseUpstreamCtx := detachUpstreamContext(ctx)
	upstreamReq, err := http.NewRequestWithContext(upstreamCtx, http.MethodPost, targetURL, bytes.NewReader(forwardBody))
	releaseUpstreamCtx()
	if err != nil {
		return nil, fmt.Errorf("build upstream request: %w", err)
	}
	upstreamReq = upstreamReq.WithContext(WithHTTPUpstreamProfile(upstreamReq.Context(), HTTPUpstreamProfileOpenAI))
	upstreamReq.Header.Set("Authorization", "Bearer "+apiKey)
	upstreamReq.Header.Set("Accept", "*/*")
	if strings.TrimSpace(forwardContentType) != "" {
		upstreamReq.Header.Set("Content-Type", forwardContentType)
	} else {
		upstreamReq.Header.Set("Content-Type", "application/json")
	}
	for key, values := range c.Request.Header {
		lowerKey := strings.ToLower(key)
		if lowerKey == "content-type" || lowerKey == "accept" || lowerKey == "authorization" {
			continue
		}
		if openaiCCRawAllowedHeaders[lowerKey] {
			for _, value := range values {
				upstreamReq.Header.Add(key, value)
			}
		}
	}
	if customUA := account.GetOpenAIUserAgent(); customUA != "" {
		upstreamReq.Header.Set("user-agent", customUA)
	}

	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	upstreamStart := time.Now()
	resp, err := s.httpUpstream.Do(upstreamReq, proxyURL, account.ID, account.Concurrency)
	SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, time.Since(upstreamStart).Milliseconds())
	if err != nil {
		safeErr := sanitizeUpstreamErrorMessage(err.Error())
		setOpsUpstreamError(c, 0, safeErr, "")
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: 0,
			UpstreamURL:        safeUpstreamURL(upstreamReq.URL.String()),
			Kind:               "request_error",
			Message:            safeErr,
		})
		writeOpenAIAudioError(c, http.StatusBadGateway, "upstream_error", "Upstream request failed")
		return nil, fmt.Errorf("upstream request failed: %s", safeErr)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		respBody := s.readUpstreamErrorBody(resp)
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(respBody))

		upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(respBody))
		upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
		if s.shouldFailoverOpenAIUpstreamResponse(resp.StatusCode, upstreamMsg, respBody) {
			upstreamDetail := ""
			if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
				maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
				if maxBytes <= 0 {
					maxBytes = 2048
				}
				upstreamDetail = truncateString(string(respBody), maxBytes)
			}
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
				Platform:           account.Platform,
				AccountID:          account.ID,
				AccountName:        account.Name,
				UpstreamStatusCode: resp.StatusCode,
				UpstreamRequestID:  resp.Header.Get("x-request-id"),
				UpstreamURL:        safeUpstreamURL(upstreamReq.URL.String()),
				Kind:               "failover",
				Message:            upstreamMsg,
				Detail:             upstreamDetail,
			})
			s.handleOpenAIAccountUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody, upstreamModel)
			return nil, &UpstreamFailoverError{
				StatusCode:             resp.StatusCode,
				ResponseBody:           respBody,
				RetryableOnSameAccount: account.IsPoolMode() && account.IsPoolModeRetryableStatus(resp.StatusCode),
			}
		}
		writeOpenAIAudioUpstreamResponse(c, resp, respBody, s.responseHeaderFilter)
		return nil, fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}

	respBody, err := ReadUpstreamResponseBody(resp.Body, s.cfg, c, openAITooLargeError)
	if err != nil {
		if err != ErrUpstreamResponseBodyTooLarge {
			writeOpenAIAudioError(c, http.StatusBadGateway, "api_error", "Failed to read upstream response")
		}
		return nil, fmt.Errorf("read upstream body: %w", err)
	}
	writeOpenAIAudioUpstreamResponse(c, resp, respBody, s.responseHeaderFilter)

	return &OpenAIForwardResult{
		RequestID:       firstNonEmptyString(resp.Header.Get("x-request-id"), resp.Header.Get("request-id")),
		Usage:           extractOpenAIAudioUsage(respBody),
		Model:           requestModel,
		BillingModel:    billingModel,
		UpstreamModel:   upstreamModel,
		Stream:          false,
		ResponseHeaders: resp.Header.Clone(),
		Duration:        time.Since(startTime),
	}, nil
}

func buildOpenAIAudioURL(base string, endpoint OpenAIAudioEndpoint) string {
	return buildOpenAIEndpointURL(base, string(endpoint))
}

func extractOpenAIAudioModel(body []byte, contentType string) string {
	if isMultipartContentType(contentType) {
		return extractMultipartTextField(body, contentType, "model")
	}
	return strings.TrimSpace(gjson.GetBytes(body, "model").String())
}

func ExtractOpenAIAudioModelForGateway(body []byte, contentType string) string {
	return extractOpenAIAudioModel(body, contentType)
}

func rewriteOpenAIAudioModel(body []byte, contentType string, model string) ([]byte, string, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return body, contentType, nil
	}
	if isMultipartContentType(contentType) {
		return rewriteMultipartTextField(body, contentType, "model", model)
	}
	rewritten, err := sjson.SetBytes(body, "model", model)
	if err != nil {
		return nil, "", fmt.Errorf("rewrite audio request model: %w", err)
	}
	return rewritten, contentType, nil
}

func isMultipartContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && strings.HasPrefix(strings.ToLower(mediaType), "multipart/")
}

func extractMultipartTextField(body []byte, contentType string, fieldName string) string {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ""
	}
	boundary := strings.TrimSpace(params["boundary"])
	if boundary == "" {
		return ""
	}
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	for {
		part, err := reader.NextPart()
		if err != nil {
			return ""
		}
		if strings.TrimSpace(part.FormName()) != fieldName || part.FileName() != "" {
			_ = part.Close()
			continue
		}
		data, readErr := io.ReadAll(io.LimitReader(part, 4096))
		_ = part.Close()
		if readErr != nil {
			return ""
		}
		return strings.TrimSpace(string(data))
	}
}

func rewriteMultipartTextField(body []byte, contentType string, fieldName string, value string) ([]byte, string, error) {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, "", fmt.Errorf("parse multipart content-type: %w", err)
	}
	boundary := strings.TrimSpace(params["boundary"])
	if boundary == "" {
		return nil, "", fmt.Errorf("multipart boundary is required")
	}

	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	fieldWritten := false

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", fmt.Errorf("read multipart body: %w", err)
		}
		formName := strings.TrimSpace(part.FormName())
		partHeader := cloneMultipartHeader(part.Header)
		target, err := writer.CreatePart(partHeader)
		if err != nil {
			_ = part.Close()
			return nil, "", fmt.Errorf("create multipart part: %w", err)
		}
		if formName == fieldName && part.FileName() == "" {
			if _, err := target.Write([]byte(value)); err != nil {
				_ = part.Close()
				return nil, "", fmt.Errorf("rewrite multipart field: %w", err)
			}
			fieldWritten = true
			_ = part.Close()
			continue
		}
		if _, err := io.Copy(target, part); err != nil {
			_ = part.Close()
			return nil, "", fmt.Errorf("copy multipart part: %w", err)
		}
		_ = part.Close()
	}
	if !fieldWritten {
		if err := writer.WriteField(fieldName, value); err != nil {
			return nil, "", fmt.Errorf("append multipart field: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("finalize multipart body: %w", err)
	}
	return buffer.Bytes(), writer.FormDataContentType(), nil
}

func writeOpenAIAudioUpstreamResponse(c *gin.Context, resp *http.Response, body []byte, filter *responseheaders.CompiledHeaderFilter) {
	if c == nil || resp == nil || c.Writer.Written() {
		return
	}
	responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, filter)
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "application/json"
	}
	c.Data(resp.StatusCode, contentType, body)
}

func writeOpenAIAudioError(c *gin.Context, statusCode int, errType, message string) {
	c.JSON(statusCode, gin.H{
		"error": gin.H{
			"type":    errType,
			"message": message,
		},
	})
}

func extractOpenAIAudioUsage(body []byte) OpenAIUsage {
	usage := gjson.GetBytes(body, "usage")
	if !usage.Exists() || !usage.IsObject() {
		return OpenAIUsage{}
	}
	return OpenAIUsage{
		InputTokens: firstPositiveGJSONInt(
			usage.Get("input_tokens"),
			usage.Get("prompt_tokens"),
			usage.Get("total_tokens"),
		),
		OutputTokens: firstPositiveGJSONInt(
			usage.Get("output_tokens"),
			usage.Get("completion_tokens"),
		),
	}
}

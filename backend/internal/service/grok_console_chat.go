package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

const (
	grokConsoleBaseURL = "https://console.x.ai"
	grokConsolePath    = "/v1/responses"
)

var grokConsoleModelMap = map[string]string{
	"grok-4.3-console":                     "grok-4.3",
	"grok-4.3-low":                         "grok-4.3",
	"grok-4.3-medium":                      "grok-4.3",
	"grok-4.3-high":                        "grok-4.3",
	"grok-4.20-0309-reasoning-console":     "grok-4.20-0309-reasoning",
	"grok-4.20-0309-console":               "grok-4.20-0309",
	"grok-4.20-0309-non-reasoning-console": "grok-4.20-0309-non-reasoning",
	"grok-4.20-multi-agent-console":        "grok-4.20-multi-agent-0309",
	"grok-4.20-multi-agent-low":            "grok-4.20-multi-agent-0309",
	"grok-4.20-multi-agent-medium":         "grok-4.20-multi-agent-0309",
	"grok-4.20-multi-agent-high":           "grok-4.20-multi-agent-0309",
	"grok-4.20-multi-agent-xhigh":          "grok-4.20-multi-agent-0309",
	"grok-build-console":                   "grok-build-0.1",
}

var grokConsoleFixedEffort = map[string]string{
	"grok-4.3-low":                 "low",
	"grok-4.3-medium":              "medium",
	"grok-4.3-high":                "high",
	"grok-4.20-multi-agent-low":    "low",
	"grok-4.20-multi-agent-medium": "medium",
	"grok-4.20-multi-agent-high":   "high",
	"grok-4.20-multi-agent-xhigh":  "xhigh",
}

var grokConsoleMaxOutputTokens = map[string]int{
	"grok-4.20-multi-agent-0309": 2000000,
	"grok-build-0.1":             256000,
}

func isGrokConsoleModel(model string) bool {
	_, ok := grokConsoleModelMap[strings.TrimSpace(model)]
	return ok
}

func normalizeGrokConsoleModel(model string) string {
	if mapped, ok := grokConsoleModelMap[strings.TrimSpace(model)]; ok {
		return mapped
	}
	return strings.TrimSpace(model)
}

func (s *OpenAIGatewayService) ForwardGrokConsoleChatCompletions(ctx context.Context, c *gin.Context, account *Account, body []byte, originalModel, billingModel, upstreamModel string, startTime time.Time) (*OpenAIForwardResult, error) {
	clientStream := gjson.GetBytes(body, "stream").Bool()
	upstreamBody, promptText, err := buildGrokConsolePayloadFromChat(body, upstreamModel, clientStream)
	if err != nil {
		writeChatCompletionsError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return nil, err
	}
	resp, err := s.doGrokConsoleRequest(ctx, c, account, upstreamBody, upstreamModel, "chat_completions")
	if err != nil {
		writeChatCompletionsError(c, http.StatusBadGateway, "upstream_error", "Grok console upstream request failed")
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if handled, err := s.handleGrokConsoleError(ctx, c, account, resp, upstreamModel, writeChatCompletionsError); handled {
		return nil, err
	}

	responseID := newGrokChatCompletionID()
	promptTokens := estimateGrokTextTokens(promptText)
	if clientStream {
		return s.streamGrokConsoleChatCompletions(c, resp, responseID, originalModel, billingModel, upstreamModel, promptTokens, startTime)
	}
	return s.bufferGrokConsoleChatCompletions(c, resp, responseID, originalModel, billingModel, upstreamModel, promptTokens, startTime)
}

func (s *OpenAIGatewayService) ForwardGrokConsoleResponses(ctx context.Context, c *gin.Context, account *Account, body []byte, originalModel, billingModel, upstreamModel string, startTime time.Time) (*OpenAIForwardResult, error) {
	clientStream := gjson.GetBytes(body, "stream").Bool()
	upstreamBody, promptText, err := buildGrokConsolePayloadFromResponses(body, upstreamModel, clientStream)
	if err != nil {
		writeResponsesError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return nil, err
	}
	resp, err := s.doGrokConsoleRequest(ctx, c, account, upstreamBody, upstreamModel, "responses")
	if err != nil {
		writeResponsesError(c, http.StatusBadGateway, "upstream_error", "Grok console upstream request failed")
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if handled, err := s.handleGrokConsoleError(ctx, c, account, resp, upstreamModel, writeResponsesError); handled {
		return nil, err
	}

	responseID := newGrokResponseID()
	messageID := newGrokResponseMessageID()
	promptTokens := estimateGrokTextTokens(promptText)
	if clientStream {
		return s.streamGrokConsoleResponses(c, resp, responseID, messageID, originalModel, billingModel, upstreamModel, promptTokens, startTime)
	}
	return s.bufferGrokConsoleResponses(c, resp, responseID, messageID, originalModel, billingModel, upstreamModel, promptTokens, startTime)
}

func (s *OpenAIGatewayService) ForwardGrokConsoleAnthropicMessages(ctx context.Context, c *gin.Context, account *Account, body []byte, originalModel, billingModel, upstreamModel string, startTime time.Time) (*OpenAIForwardResult, error) {
	clientStream := gjson.GetBytes(body, "stream").Bool()
	upstreamBody, promptText, err := buildGrokConsolePayloadFromAnthropic(body, upstreamModel, clientStream)
	if err != nil {
		writeAnthropicError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return nil, err
	}
	resp, err := s.doGrokConsoleRequest(ctx, c, account, upstreamBody, upstreamModel, "messages")
	if err != nil {
		writeAnthropicError(c, http.StatusBadGateway, "api_error", "Grok console upstream request failed")
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if handled, err := s.handleGrokConsoleError(ctx, c, account, resp, upstreamModel, writeAnthropicError); handled {
		return nil, err
	}

	responseID := "msg_grok_" + newGrokRequestID()
	promptTokens := estimateGrokTextTokens(promptText)
	if clientStream {
		return s.streamGrokConsoleAnthropicMessages(c, resp, responseID, originalModel, billingModel, upstreamModel, promptTokens, startTime)
	}
	return s.bufferGrokConsoleAnthropicMessages(c, resp, responseID, originalModel, billingModel, upstreamModel, promptTokens, startTime)
}

func buildGrokConsolePayloadFromChat(body []byte, model string, stream bool) ([]byte, string, error) {
	messages := gjson.GetBytes(body, "messages")
	if !messages.Exists() || !messages.IsArray() {
		return nil, "", errors.New("messages is required")
	}
	input, promptText := buildGrokConsoleInputFromMessages(messages)
	if len(input) == 0 {
		return nil, "", errors.New("messages must contain text or image content")
	}
	return buildGrokConsolePayload(body, model, input, stream), promptText, nil
}

func buildGrokConsolePayloadFromResponses(body []byte, model string, stream bool) ([]byte, string, error) {
	inputValue := gjson.GetBytes(body, "input")
	if !inputValue.Exists() {
		return nil, "", errors.New("input is required")
	}
	input, promptText := buildGrokConsoleInputFromResponses(inputValue)
	if len(input) == 0 {
		return nil, "", errors.New("input must contain text or image content")
	}
	return buildGrokConsolePayload(body, model, input, stream), promptText, nil
}

func buildGrokConsolePayloadFromAnthropic(body []byte, model string, stream bool) ([]byte, string, error) {
	var req apicompat.AnthropicRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, "", fmt.Errorf("invalid Anthropic request: %w", err)
	}
	responsesReq, err := apicompat.AnthropicToResponses(&req)
	if err != nil {
		return nil, "", err
	}
	raw, err := json.Marshal(responsesReq)
	if err != nil {
		return nil, "", err
	}
	return buildGrokConsolePayloadFromResponses(raw, model, stream)
}

func buildGrokConsolePayload(sourceBody []byte, model string, input []map[string]any, stream bool) []byte {
	consoleModel := normalizeGrokConsoleModel(model)
	temperature := grokConsoleFloat(sourceBody, "temperature", 0.7)
	topP := grokConsoleFloat(sourceBody, "top_p", 0.95)
	maxOutputTokens := grokConsoleMaxOutputTokens[consoleModel]
	if maxOutputTokens == 0 {
		maxOutputTokens = 1000000
	}
	payload := map[string]any{
		"model":             consoleModel,
		"input":             input,
		"max_output_tokens": maxOutputTokens,
		"temperature":       temperature,
		"top_p":             topP,
		"store":             false,
		"include":           []string{"reasoning.encrypted_content"},
		"stream":            stream,
	}
	if consoleModel == "grok-4.3" || consoleModel == "grok-4.20-multi-agent-0309" {
		payload["reasoning"] = map[string]any{"effort": grokConsoleReasoningEffort(sourceBody, model)}
	}
	if grokConsoleSupportsSearchTools(consoleModel) {
		payload["tools"] = []map[string]any{
			{"type": "web_search", "enable_image_understanding": true},
			{"type": "x_search", "enable_video_understanding": true},
		}
		payload["tool_choice"] = "auto"
	}
	raw, _ := json.Marshal(payload)
	return raw
}

func buildGrokConsoleInputFromMessages(messages gjson.Result) ([]map[string]any, string) {
	input := make([]map[string]any, 0, len(messages.Array()))
	var promptParts []string
	for _, msg := range messages.Array() {
		role := grokConsoleRole(msg.Get("role").String())
		content, text := grokConsoleContentParts(msg.Get("content"))
		if len(content) == 0 {
			continue
		}
		input = append(input, map[string]any{
			"role":    role,
			"content": content,
		})
		if text != "" {
			promptParts = append(promptParts, fmt.Sprintf("[%s]: %s", role, text))
		}
	}
	return input, strings.TrimSpace(strings.Join(promptParts, "\n\n"))
}

func buildGrokConsoleInputFromResponses(inputValue gjson.Result) ([]map[string]any, string) {
	switch {
	case inputValue.Type == gjson.String:
		text := strings.TrimSpace(inputValue.String())
		if text == "" {
			return nil, ""
		}
		return []map[string]any{{"role": "user", "content": []map[string]any{{"type": "input_text", "text": text}}}}, text
	case inputValue.IsArray():
		input := make([]map[string]any, 0, len(inputValue.Array()))
		var promptParts []string
		for _, item := range inputValue.Array() {
			if item.Type == gjson.String {
				text := strings.TrimSpace(item.String())
				if text == "" {
					continue
				}
				input = append(input, map[string]any{"role": "user", "content": []map[string]any{{"type": "input_text", "text": text}}})
				promptParts = append(promptParts, text)
				continue
			}
			role := grokConsoleRole(item.Get("role").String())
			content, text := grokConsoleContentParts(item.Get("content"))
			if len(content) == 0 {
				continue
			}
			input = append(input, map[string]any{"role": role, "content": content})
			if text != "" {
				promptParts = append(promptParts, fmt.Sprintf("[%s]: %s", role, text))
			}
		}
		return input, strings.TrimSpace(strings.Join(promptParts, "\n\n"))
	default:
		return nil, ""
	}
}

func grokConsoleContentParts(content gjson.Result) ([]map[string]any, string) {
	switch {
	case content.Type == gjson.String:
		text := strings.TrimSpace(content.String())
		if text == "" {
			return nil, ""
		}
		return []map[string]any{{"type": "input_text", "text": text}}, text
	case content.IsArray():
		parts := make([]map[string]any, 0, len(content.Array()))
		var texts []string
		for _, block := range content.Array() {
			switch block.Get("type").String() {
			case "text", "input_text", "output_text":
				if text := strings.TrimSpace(block.Get("text").String()); text != "" {
					parts = append(parts, map[string]any{"type": "input_text", "text": text})
					texts = append(texts, text)
				}
			case "image_url", "input_image":
				url := strings.TrimSpace(grokFirstNonEmpty(block.Get("image_url.url").String(), block.Get("image_url").String()))
				if url != "" {
					parts = append(parts, map[string]any{"type": "input_image", "image_url": url})
				}
			case "image":
				source := block.Get("source")
				if source.Get("type").String() == "base64" {
					mime := grokFirstNonEmpty(source.Get("media_type").String(), "image/png")
					data := strings.TrimSpace(source.Get("data").String())
					if data != "" {
						parts = append(parts, map[string]any{"type": "input_image", "image_url": "data:" + mime + ";base64," + data})
					}
				}
			}
		}
		return parts, strings.TrimSpace(strings.Join(texts, "\n"))
	default:
		return nil, ""
	}
}

func grokConsoleRole(role string) string {
	switch strings.TrimSpace(role) {
	case "system", "developer":
		return "system"
	case "assistant":
		return "assistant"
	default:
		return "user"
	}
}

func grokConsoleFloat(body []byte, path string, fallback float64) float64 {
	value := gjson.GetBytes(body, path)
	if !value.Exists() || value.Type != gjson.Number {
		return fallback
	}
	return value.Float()
}

func grokConsoleReasoningEffort(body []byte, model string) string {
	if fixed := grokConsoleFixedEffort[strings.TrimSpace(model)]; fixed != "" {
		return fixed
	}
	raw := strings.TrimSpace(gjson.GetBytes(body, "reasoning.effort").String())
	switch raw {
	case "none":
		return "none"
	case "minimal", "low":
		return "low"
	case "high":
		return "high"
	case "xhigh", "max":
		return "xhigh"
	default:
		return "medium"
	}
}

func grokConsoleSupportsSearchTools(model string) bool {
	switch model {
	case "grok-4.20-multi-agent-0309", "grok-4.20-0309", "grok-4.20-0309-reasoning", "grok-4.20-0309-non-reasoning", "grok-4.3", "grok-build-0.1":
		return true
	default:
		return false
	}
}

func (s *OpenAIGatewayService) doGrokConsoleRequest(ctx context.Context, c *gin.Context, account *Account, body []byte, upstreamModel, kind string) (*http.Response, error) {
	req, err := s.buildGrokConsoleRequest(ctx, account, body)
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
		return nil, fmt.Errorf("grok console %s upstream request failed: %s", kind, safeErr)
	}
	return resp, nil
}

func (s *OpenAIGatewayService) buildGrokConsoleRequest(ctx context.Context, account *Account, body []byte) (*http.Request, error) {
	upstreamCtx, releaseUpstreamCtx := detachUpstreamContext(ctx)
	req, err := http.NewRequestWithContext(upstreamCtx, http.MethodPost, grokConsoleBaseURL+grokConsolePath, bytes.NewReader(body))
	releaseUpstreamCtx()
	if err != nil {
		return nil, err
	}
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	req.Header.Set("Accept", "text/event-stream, application/json")
	req.Header.Set("Accept-Language", grokFirstNonEmpty(account.GetCredential("accept_language"), "zh-CN,zh;q=0.9,en;q=0.8"))
	req.Header.Set("Authorization", "Bearer anonymous")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", grokConsoleBaseURL)
	req.Header.Set("Referer", grokConsoleBaseURL+"/")
	req.Header.Set("User-Agent", grokFirstNonEmpty(account.GetCredential("user_agent"), defaultGrokWebUserAgent()))
	req.Header.Set("Cookie", buildGrokWebCookieHeader(account))
	if req.Header.Get("Cookie") == "" {
		return nil, errors.New("sso_token or cookie is required for Grok Console account")
	}
	applyGrokWebCompatibilityHeaders(req.Header, account, s.cfg)
	return req, nil
}

func (s *OpenAIGatewayService) handleGrokConsoleError(ctx context.Context, c *gin.Context, account *Account, resp *http.Response, upstreamModel string, writeErr func(*gin.Context, int, string, string)) (bool, error) {
	if resp.StatusCode < 400 {
		return false, nil
	}
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
		return true, &UpstreamFailoverError{
			StatusCode:             resp.StatusCode,
			ResponseBody:           respBody,
			RetryableOnSameAccount: false,
		}
	}
	writeErr(c, resp.StatusCode, "upstream_error", upstreamMsg)
	return true, fmt.Errorf("grok console upstream returned %d: %s", resp.StatusCode, upstreamMsg)
}

type grokConsoleCollectedResponse struct {
	Text  string
	Usage OpenAIUsage
}

type grokConsoleStreamAdapter struct {
	event string
	data  strings.Builder
	text  strings.Builder
	usage OpenAIUsage
	done  bool
}

func (a *grokConsoleStreamAdapter) FeedLine(line string) ([]string, bool, error) {
	line = strings.TrimRight(line, "\r")
	if strings.TrimSpace(line) == "" {
		return a.flush()
	}
	if strings.HasPrefix(line, "event:") {
		a.event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		return nil, false, nil
	}
	if strings.HasPrefix(line, "data:") {
		if a.data.Len() > 0 {
			a.data.WriteByte('\n')
		}
		a.data.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
	}
	return nil, false, nil
}

func (a *grokConsoleStreamAdapter) flush() ([]string, bool, error) {
	data := strings.TrimSpace(a.data.String())
	event := strings.TrimSpace(a.event)
	a.event = ""
	a.data.Reset()
	if data == "" {
		return nil, a.done, nil
	}
	if data == "[DONE]" {
		a.done = true
		return nil, true, nil
	}
	if event == "" {
		event = gjson.Get(data, "type").String()
	}
	if errObj := gjson.Get(data, "error"); errObj.Exists() {
		message := strings.TrimSpace(grokFirstNonEmpty(errObj.Get("message").String(), errObj.String(), "Grok console stream error"))
		return nil, false, fmt.Errorf("grok console stream error: %s", message)
	}
	switch event {
	case "response.output_text.delta":
		delta := gjson.Get(data, "delta").String()
		if delta == "" {
			return nil, false, nil
		}
		a.text.WriteString(delta)
		return []string{delta}, false, nil
	case "response.completed":
		a.usage = parseGrokConsoleUsage(gjson.Get(data, "response.usage"))
		a.done = true
		return nil, true, nil
	default:
		return nil, a.done, nil
	}
}

func readGrokConsoleResponse(r io.Reader) (grokConsoleCollectedResponse, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return grokConsoleCollectedResponse{}, err
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return grokConsoleCollectedResponse{}, nil
	}
	if strings.HasPrefix(trimmed, "{") {
		return parseGrokConsoleJSONResponse(trimmed), nil
	}
	adapter := &grokConsoleStreamAdapter{}
	scanner := bufio.NewScanner(strings.NewReader(trimmed))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if _, done, err := adapter.FeedLine(scanner.Text()); err != nil {
			return grokConsoleCollectedResponse{}, err
		} else if done {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return grokConsoleCollectedResponse{}, err
	}
	if _, _, err := adapter.flush(); err != nil {
		return grokConsoleCollectedResponse{}, err
	}
	return grokConsoleCollectedResponse{Text: adapter.text.String(), Usage: adapter.usage}, nil
}

func parseGrokConsoleJSONResponse(raw string) grokConsoleCollectedResponse {
	var text strings.Builder
	for _, item := range gjson.Get(raw, "output").Array() {
		for _, part := range item.Get("content").Array() {
			if part.Get("type").String() == "output_text" || part.Get("text").Exists() {
				text.WriteString(part.Get("text").String())
			}
		}
	}
	return grokConsoleCollectedResponse{
		Text:  text.String(),
		Usage: parseGrokConsoleUsage(gjson.Get(raw, "usage")),
	}
}

func parseGrokConsoleUsage(usage gjson.Result) OpenAIUsage {
	if !usage.Exists() {
		return OpenAIUsage{}
	}
	return OpenAIUsage{
		InputTokens:  int(usage.Get("input_tokens").Int()),
		OutputTokens: int(usage.Get("output_tokens").Int()),
	}
}

func completeGrokConsoleUsage(usage OpenAIUsage, inputTokens int, output string) OpenAIUsage {
	if usage.InputTokens <= 0 {
		usage.InputTokens = inputTokens
	}
	if usage.OutputTokens <= 0 {
		usage.OutputTokens = estimateGrokTextTokens(output)
	}
	return usage
}

func (s *OpenAIGatewayService) streamGrokConsoleChatCompletions(c *gin.Context, resp *http.Response, responseID, originalModel, billingModel, upstreamModel string, promptTokens int, startTime time.Time) (*OpenAIForwardResult, error) {
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
	firstTokenMs := 0
	seenFirst := false
	adapter := &grokConsoleStreamAdapter{}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		deltas, done, err := adapter.FeedLine(scanner.Text())
		if err != nil {
			return nil, err
		}
		for _, delta := range deltas {
			if !seenFirst {
				seenFirst = true
				firstTokenMs = int(time.Since(startTime).Milliseconds())
			}
			output.WriteString(delta)
			if err := writeGrokSSEJSON(c.Writer, grokChatCompletionChunk(responseID, originalModel, delta, false)); err != nil {
				return nil, err
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if done {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read grok console stream: %w", err)
	}
	if deltas, _, err := adapter.flush(); err != nil {
		return nil, err
	} else {
		for _, delta := range deltas {
			if !seenFirst {
				seenFirst = true
				firstTokenMs = int(time.Since(startTime).Milliseconds())
			}
			output.WriteString(delta)
			if err := writeGrokSSEJSON(c.Writer, grokChatCompletionChunk(responseID, originalModel, delta, false)); err != nil {
				return nil, err
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
	}
	usage := completeGrokConsoleUsage(adapter.usage, promptTokens, output.String())
	if err := writeGrokSSEJSON(c.Writer, grokChatCompletionFinalChunk(responseID, originalModel, nil, nil)); err != nil {
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

func (s *OpenAIGatewayService) bufferGrokConsoleChatCompletions(c *gin.Context, resp *http.Response, responseID, originalModel, billingModel, upstreamModel string, promptTokens int, startTime time.Time) (*OpenAIForwardResult, error) {
	collected, err := readGrokConsoleResponse(resp.Body)
	if err != nil {
		return nil, err
	}
	usage := completeGrokConsoleUsage(collected.Usage, promptTokens, collected.Text)
	c.JSON(http.StatusOK, grokChatCompletionResponse(responseID, originalModel, collected.Text, "", usage, nil, nil))
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

func (s *OpenAIGatewayService) streamGrokConsoleResponses(c *gin.Context, resp *http.Response, responseID, messageID, originalModel, billingModel, upstreamModel string, promptTokens int, startTime time.Time) (*OpenAIForwardResult, error) {
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
	firstTokenMs := 0
	seenFirst := false
	adapter := &grokConsoleStreamAdapter{}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		deltas, done, err := adapter.FeedLine(scanner.Text())
		if err != nil {
			return nil, err
		}
		for _, delta := range deltas {
			if !seenFirst {
				seenFirst = true
				firstTokenMs = int(time.Since(startTime).Milliseconds())
			}
			output.WriteString(delta)
			if err := writeEvent(grokResponsesTextDeltaEvent(responseID, messageID, delta)); err != nil {
				return nil, err
			}
		}
		if done {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read grok console responses stream: %w", err)
	}
	if deltas, _, err := adapter.flush(); err != nil {
		return nil, err
	} else {
		for _, delta := range deltas {
			if !seenFirst {
				seenFirst = true
				firstTokenMs = int(time.Since(startTime).Milliseconds())
			}
			output.WriteString(delta)
			if err := writeEvent(grokResponsesTextDeltaEvent(responseID, messageID, delta)); err != nil {
				return nil, err
			}
		}
	}
	content := output.String()
	usage := completeGrokConsoleUsage(adapter.usage, promptTokens, content)
	response := grokResponsesResponse(responseID, messageID, originalModel, content, "", usage, nil, nil)
	if err := writeEvent(grokResponsesTextDoneEvent(responseID, messageID, content)); err != nil {
		return nil, err
	}
	if err := writeEvent(grokResponsesContentPartDoneEvent(responseID, messageID, content, nil)); err != nil {
		return nil, err
	}
	if err := writeEvent(grokResponsesOutputItemDoneEvent(responseID, messageID, content, nil)); err != nil {
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

func (s *OpenAIGatewayService) bufferGrokConsoleResponses(c *gin.Context, resp *http.Response, responseID, messageID, originalModel, billingModel, upstreamModel string, promptTokens int, startTime time.Time) (*OpenAIForwardResult, error) {
	collected, err := readGrokConsoleResponse(resp.Body)
	if err != nil {
		return nil, err
	}
	usage := completeGrokConsoleUsage(collected.Usage, promptTokens, collected.Text)
	c.JSON(http.StatusOK, grokResponsesResponse(responseID, messageID, originalModel, collected.Text, "", usage, nil, nil))
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

func (s *OpenAIGatewayService) streamGrokConsoleAnthropicMessages(c *gin.Context, resp *http.Response, responseID, originalModel, billingModel, upstreamModel string, promptTokens int, startTime time.Time) (*OpenAIForwardResult, error) {
	if s.responseHeaderFilter != nil {
		responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)
	flusher, _ := c.Writer.(http.Flusher)
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
	textIndex := 0
	textBlockOpen := false
	var output strings.Builder
	firstTokenMs := 0
	seenFirst := false
	adapter := &grokConsoleStreamAdapter{}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		deltas, done, err := adapter.FeedLine(scanner.Text())
		if err != nil {
			return nil, err
		}
		for _, delta := range deltas {
			if !seenFirst {
				seenFirst = true
				firstTokenMs = int(time.Since(startTime).Milliseconds())
			}
			if !textBlockOpen {
				if err := writeEvent(grokAnthropicContentBlockStartEvent(textIndex, "text")); err != nil {
					return nil, err
				}
				textBlockOpen = true
			}
			output.WriteString(delta)
			if err := writeEvent(grokAnthropicTextDeltaEvent(textIndex, delta)); err != nil {
				return nil, err
			}
		}
		if done {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read grok console messages stream: %w", err)
	}
	if deltas, _, err := adapter.flush(); err != nil {
		return nil, err
	} else {
		for _, delta := range deltas {
			if !seenFirst {
				seenFirst = true
				firstTokenMs = int(time.Since(startTime).Milliseconds())
			}
			if !textBlockOpen {
				if err := writeEvent(grokAnthropicContentBlockStartEvent(textIndex, "text")); err != nil {
					return nil, err
				}
				textBlockOpen = true
			}
			output.WriteString(delta)
			if err := writeEvent(grokAnthropicTextDeltaEvent(textIndex, delta)); err != nil {
				return nil, err
			}
		}
	}
	if textBlockOpen {
		if err := writeEvent(grokAnthropicContentBlockStopEvent(textIndex)); err != nil {
			return nil, err
		}
	}
	usage := completeGrokConsoleUsage(adapter.usage, promptTokens, output.String())
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

func (s *OpenAIGatewayService) bufferGrokConsoleAnthropicMessages(c *gin.Context, resp *http.Response, responseID, originalModel, billingModel, upstreamModel string, promptTokens int, startTime time.Time) (*OpenAIForwardResult, error) {
	collected, err := readGrokConsoleResponse(resp.Body)
	if err != nil {
		return nil, err
	}
	usage := completeGrokConsoleUsage(collected.Usage, promptTokens, collected.Text)
	c.JSON(http.StatusOK, grokAnthropicMessageResponse(responseID, originalModel, collected.Text, "", usage))
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

package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const openAICompactClientStreamKey = "openai_compact_client_stream"

// MarkOpenAICompactClientStream records the original stream intent for a
// body-signal compact request. The normalized upstream request is unary, but
// the client still expects a Responses SSE stream.
func MarkOpenAICompactClientStream(c *gin.Context) {
	if c == nil {
		return
	}
	c.Set(openAICompactClientStreamKey, true)
}

func OpenAICompactClientStreamKeyForTest() string {
	return openAICompactClientStreamKey
}

func openAICompactClientWantsStream(c *gin.Context) bool {
	if c == nil {
		return false
	}
	value, ok := c.Get(openAICompactClientStreamKey)
	if !ok {
		return false
	}
	wants, _ := value.(bool)
	return wants
}

// writeOpenAICompactSSEBridge converts a successful unary compact response to
// the minimal Responses SSE sequence consumed by Codex remote compact v2.
func writeOpenAICompactSSEBridge(c *gin.Context, statusCode int, finalResponse []byte) bool {
	if c == nil || !openAICompactClientWantsStream(c) {
		return false
	}
	committed := StopOpenAICompactSSEKeepaliveCommitted(c)
	if statusCode < 200 || statusCode >= 300 {
		if committed {
			writeOpenAICompactSSEFailure(c, statusCode, finalResponse)
			return true
		}
		return false
	}
	payload, ok := buildOpenAICompactSSEPayload(finalResponse)
	if !ok {
		if committed {
			writeOpenAICompactSSEFailure(c, http.StatusBadGateway, finalResponse)
			return true
		}
		return false
	}
	if !committed {
		header := c.Writer.Header()
		header.Set("Content-Type", "text/event-stream")
		header.Set("Cache-Control", "no-cache")
		header.Set("Connection", "keep-alive")
		header.Set("X-Accel-Buffering", "no")
		c.Writer.WriteHeader(statusCode)
	}
	_, _ = c.Writer.Write(payload)
	c.Writer.Flush()
	return true
}

func writeOpenAICompactSSEFailure(c *gin.Context, statusCode int, errorBody []byte) {
	message := ""
	if len(errorBody) > 0 {
		message = sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(errorBody)))
	}
	if message == "" {
		message = "Upstream compact request failed with HTTP " + strconv.Itoa(statusCode)
	}
	writeOpenAICompactSSEFailureMessage(c, statusCode, "upstream_error", message)
}

func writeOpenAICompactSSEFailureMessage(c *gin.Context, statusCode int, errType, message string) {
	if c == nil {
		return
	}
	MarkOpsStreamError(c, errType, message, statusCode)
	payload, err := json.Marshal(map[string]any{
		"type": "response.failed",
		"response": map[string]any{
			"id": "resp_" + strings.ReplaceAll(uuid.NewString(), "-", ""), "object": "response", "status": "failed", "output": []any{},
			"error": map[string]any{"code": errType, "message": message},
		},
	})
	if err != nil {
		return
	}
	_, _ = c.Writer.Write([]byte("event: response.failed\ndata: "))
	_, _ = c.Writer.Write(payload)
	_, _ = c.Writer.Write([]byte("\n\n"))
	c.Writer.Flush()
}

func buildOpenAICompactSSEPayload(finalResponse []byte) ([]byte, bool) {
	if len(finalResponse) == 0 || !gjson.ValidBytes(finalResponse) {
		return nil, false
	}
	if !gjson.ParseBytes(finalResponse).IsObject() {
		return nil, false
	}

	var compacted bytes.Buffer
	if err := json.Compact(&compacted, finalResponse); err != nil {
		return nil, false
	}
	response := compacted.Bytes()
	if strings.TrimSpace(gjson.GetBytes(response, "id").String()) == "" {
		next, err := sjson.SetBytes(response, "id", "resp_"+strings.ReplaceAll(uuid.NewString(), "-", ""))
		if err != nil {
			return nil, false
		}
		response = next
	}
	if usage := gjson.GetBytes(response, "usage"); usage.Exists() && !openAICompactUsageParsableByCodex(usage) {
		next, err := sjson.DeleteBytes(response, "usage")
		if err != nil {
			return nil, false
		}
		response = next
	}

	var buf bytes.Buffer
	outputIndex := 0
	appendEvent := func(eventType string, data []byte) {
		_, _ = buf.WriteString("event: ")
		_, _ = buf.WriteString(eventType)
		_, _ = buf.WriteString("\ndata: ")
		_, _ = buf.Write(data)
		_, _ = buf.WriteString("\n\n")
	}
	for _, item := range gjson.GetBytes(response, "output").Array() {
		if !item.IsObject() {
			continue
		}
		event, err := sjson.SetBytes([]byte(`{"type":"response.output_item.done"}`), "output_index", outputIndex)
		if err != nil {
			return nil, false
		}
		event, err = sjson.SetRawBytes(event, "item", []byte(item.Raw))
		if err != nil {
			return nil, false
		}
		appendEvent("response.output_item.done", event)
		outputIndex++
	}

	completed, err := sjson.SetRawBytes([]byte(`{"type":"response.completed"}`), "response", response)
	if err != nil {
		return nil, false
	}
	appendEvent("response.completed", completed)
	return buf.Bytes(), true
}

func openAICompactUsageParsableByCodex(usage gjson.Result) bool {
	if !usage.IsObject() {
		return false
	}
	for _, field := range []string{"input_tokens", "output_tokens", "total_tokens"} {
		if usage.Get(field).Type != gjson.Number {
			return false
		}
	}
	return true
}

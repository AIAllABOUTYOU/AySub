package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func newCompactBridgeTestContext(t *testing.T, markClientStream bool) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	if markClientStream {
		MarkOpenAICompactClientStream(c)
	}
	return c, rec
}

func newCompactBridgeTestService() *OpenAIGatewayService {
	return &OpenAIGatewayService{cfg: &config.Config{}}
}

func parseCompactBridgeSSE(t *testing.T, body string) [][2]string {
	t.Helper()
	var events [][2]string
	for _, block := range strings.Split(strings.TrimSpace(body), "\n\n") {
		var eventType, data string
		for _, line := range strings.Split(block, "\n") {
			switch {
			case strings.HasPrefix(line, "event: "):
				eventType = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				data = strings.TrimPrefix(line, "data: ")
			}
		}
		require.NotEmpty(t, eventType)
		require.True(t, gjson.Valid(data), "invalid SSE JSON: %s", data)
		events = append(events, [2]string{eventType, data})
	}
	return events
}

func TestBuildOpenAICompactSSEPayload_EmitsItemsAndCompleted(t *testing.T) {
	finalResponse := []byte(`{
		"id":"resp_compact_1",
		"object":"response",
		"status":"completed",
		"output":[
			{"type":"compaction","id":"cmp_1","encrypted_content":"enc-1"},
			{"type":"message","id":"msg_1","role":"assistant","content":[]}
		],
		"usage":{"input_tokens":10,"output_tokens":2,"total_tokens":12}
	}`)

	payload, ok := buildOpenAICompactSSEPayload(finalResponse)
	require.True(t, ok)
	events := parseCompactBridgeSSE(t, string(payload))
	require.Len(t, events, 3)
	require.Equal(t, "response.output_item.done", events[0][0])
	require.Equal(t, int64(0), gjson.Get(events[0][1], "output_index").Int())
	require.Equal(t, "compaction", gjson.Get(events[0][1], "item.type").String())
	require.Equal(t, "response.output_item.done", events[1][0])
	require.Equal(t, int64(1), gjson.Get(events[1][1], "output_index").Int())
	require.Equal(t, "response.completed", events[2][0])
	require.Equal(t, "resp_compact_1", gjson.Get(events[2][1], "response.id").String())
	require.Equal(t, int64(12), gjson.Get(events[2][1], "response.usage.total_tokens").Int())
}

func TestBuildOpenAICompactSSEPayload_InjectsMissingResponseID(t *testing.T) {
	payload, ok := buildOpenAICompactSSEPayload([]byte(`{"output":[{"type":"compaction","encrypted_content":"x"}]}`))
	require.True(t, ok)
	events := parseCompactBridgeSSE(t, string(payload))
	id := gjson.Get(events[len(events)-1][1], "response.id").String()
	require.True(t, strings.HasPrefix(id, "resp_"))
}

func TestBuildOpenAICompactSSEPayload_DropsMalformedUsage(t *testing.T) {
	payload, ok := buildOpenAICompactSSEPayload([]byte(`{"id":"resp_1","output":[],"usage":{"input_tokens":10}}`))
	require.True(t, ok)
	events := parseCompactBridgeSSE(t, string(payload))
	require.False(t, gjson.Get(events[len(events)-1][1], "response.usage").Exists())
}

func TestBuildOpenAICompactSSEPayload_KeepsWellFormedUsage(t *testing.T) {
	payload, ok := buildOpenAICompactSSEPayload([]byte(`{"id":"resp_1","output":[],"usage":{"input_tokens":10,"output_tokens":2,"total_tokens":12}}`))
	require.True(t, ok)
	events := parseCompactBridgeSSE(t, string(payload))
	require.Equal(t, int64(12), gjson.Get(events[len(events)-1][1], "response.usage.total_tokens").Int())
}

func TestBuildOpenAICompactSSEPayload_RejectsNonJSONObject(t *testing.T) {
	for _, body := range [][]byte{nil, []byte(`not-json`), []byte(`[]`), []byte(`"text"`)} {
		payload, ok := buildOpenAICompactSSEPayload(body)
		require.False(t, ok)
		require.Nil(t, payload)
	}
}

func TestWriteOpenAICompactSSEBridge_RequiresMarkAndSuccessStatus(t *testing.T) {
	finalResponse := []byte(`{"id":"resp_1","output":[{"type":"compaction","encrypted_content":"x"}]}`)

	c, rec := newCompactBridgeTestContext(t, false)
	require.False(t, writeOpenAICompactSSEBridge(c, http.StatusOK, finalResponse))
	require.Empty(t, rec.Body.String())

	c, rec = newCompactBridgeTestContext(t, true)
	require.False(t, writeOpenAICompactSSEBridge(c, http.StatusBadRequest, finalResponse))
	require.Empty(t, rec.Body.String())

	c, rec = newCompactBridgeTestContext(t, true)
	require.True(t, writeOpenAICompactSSEBridge(c, http.StatusOK, finalResponse))
	require.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
	require.Contains(t, rec.Body.String(), "event: response.completed")
}

func TestHandleNonStreamingResponse_CompactClientStreamBridgesToSSE(t *testing.T) {
	svc := newCompactBridgeTestService()
	c, rec := newCompactBridgeTestContext(t, true)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       ioNopCloser(`{"id":"resp_1","output":[{"type":"compaction","encrypted_content":"x"}],"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}`),
	}

	_, err := svc.handleNonStreamingResponse(c.Request.Context(), resp, c, &Account{Type: AccountTypeOAuth}, "gpt-5.5", "gpt-5.5")
	require.NoError(t, err)
	require.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
	events := parseCompactBridgeSSE(t, rec.Body.String())
	require.Equal(t, "response.completed", events[len(events)-1][0])
}

func TestHandleNonStreamingResponse_PathBasedCompactStaysJSON(t *testing.T) {
	svc := newCompactBridgeTestService()
	c, rec := newCompactBridgeTestContext(t, false)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       ioNopCloser(`{"id":"resp_1","output":[],"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}`),
	}

	_, err := svc.handleNonStreamingResponse(c.Request.Context(), resp, c, &Account{Type: AccountTypeOAuth}, "gpt-5.5", "gpt-5.5")
	require.NoError(t, err)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	require.True(t, json.Valid(rec.Body.Bytes()))
}

func TestHandleSSEToJSON_CompactRawOutputItemDoneRepairsEmptyTerminalOutput(t *testing.T) {
	svc := newCompactBridgeTestService()
	c, rec := newCompactBridgeTestContext(t, true)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)
	upstreamSSE := strings.Join([]string{
		`data: {"type":"response.output_item.done","output_index":0,"item":{"id":"cmp_1","type":"compaction_summary","summary":[{"type":"summary_text","text":"compact summary"}],"encrypted_content":"compact-payload","opaque":{"kept":true}}}`,
		``,
		`data: {"type":"response.completed","response":{"id":"resp_compact","object":"response","status":"completed","output":[],"usage":{"input_tokens":9,"output_tokens":4,"total_tokens":13}}}`,
		``,
	}, "\n")
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       ioNopCloser(upstreamSSE),
	}

	result, err := svc.handleNonStreamingResponse(c.Request.Context(), resp, c, &Account{ID: 1, Type: AccountTypeOAuth}, "gpt-5.5", "gpt-5.5")
	require.NoError(t, err)
	require.NotNil(t, result)

	events := parseCompactBridgeSSE(t, rec.Body.String())
	require.Len(t, events, 2)
	item := gjson.Get(events[0][1], "item")
	require.Equal(t, "compaction_summary", item.Get("type").String())
	require.Equal(t, "compact-payload", item.Get("encrypted_content").String())
	require.True(t, item.Get("opaque.kept").Bool())
	require.Len(t, gjson.Get(events[1][1], "response.output").Array(), 1)
}

func TestReconstructResponseOutputFromSSE_PrefersRawDoneItems(t *testing.T) {
	bodyText := strings.Join([]string{
		`data: {"type":"response.output_text.delta","delta":"hello"}`,
		`data: {"type":"response.output_item.done","item":{"id":"msg_1","type":"message","content":[{"type":"output_text","text":"hello"}]}}`,
	}, "\n")

	outputJSON, ok := reconstructResponseOutputFromSSE(bodyText)
	require.True(t, ok)
	items := gjson.ParseBytes(outputJSON).Array()
	require.Len(t, items, 1)
	require.Equal(t, "msg_1", items[0].Get("id").String())
}

func TestReconstructResponseOutputFromSSE_MixedDoneAndCompactionAdded(t *testing.T) {
	bodyText := strings.Join([]string{
		`data: {"type":"response.output_item.added","item":{"id":"cmp_1","type":"compaction","encrypted_content":"mixed"}}`,
		`data: {"type":"response.output_item.done","item":{"id":"msg_1","type":"message","content":[]}}`,
	}, "\n")

	outputJSON, ok := reconstructResponseOutputFromSSE(bodyText)
	require.True(t, ok)
	items := gjson.ParseBytes(outputJSON).Array()
	require.Len(t, items, 2)
	require.Equal(t, "msg_1", items[0].Get("id").String())
	require.Equal(t, "cmp_1", items[1].Get("id").String())
}

func ioNopCloser(body string) *readCloser {
	return &readCloser{Reader: strings.NewReader(body)}
}

type readCloser struct {
	*strings.Reader
}

func (r *readCloser) Close() error { return nil }

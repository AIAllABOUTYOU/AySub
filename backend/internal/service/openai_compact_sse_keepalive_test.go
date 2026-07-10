package service

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

const keepaliveTestInterval = 10 * time.Millisecond

func waitForKeepaliveBeats() {
	time.Sleep(20 * keepaliveTestInterval)
}

func stripKeepaliveComments(body string) string {
	var blocks []string
	for _, block := range strings.Split(strings.TrimSpace(body), "\n\n") {
		if strings.HasPrefix(strings.TrimSpace(block), ":") {
			continue
		}
		blocks = append(blocks, block)
	}
	return strings.Join(blocks, "\n\n")
}

func TestOpenAICompactSSEKeepalive_CommitsHeadersAndComments(t *testing.T) {
	c, rec := newCompactBridgeTestContext(t, true)
	stop := StartOpenAICompactSSEKeepalive(c, keepaliveTestInterval)
	defer stop()
	waitForKeepaliveBeats()

	require.True(t, StopOpenAICompactSSEKeepaliveCommitted(c))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
	require.Contains(t, rec.Body.String(), ": keepalive\n\n")
}

func TestWriteOpenAICompactSSEBridge_AfterKeepaliveCommitFailureEmitsFailedEvent(t *testing.T) {
	c, rec := newCompactBridgeTestContext(t, true)
	stop := StartOpenAICompactSSEKeepalive(c, keepaliveTestInterval)
	defer stop()
	waitForKeepaliveBeats()

	require.True(t, writeOpenAICompactSSEBridge(c, http.StatusBadGateway, []byte(`{"error":{"message":"upstream exploded"}}`)))

	events := parseCompactBridgeSSE(t, stripKeepaliveComments(rec.Body.String()))
	require.Len(t, events, 1)
	require.Equal(t, "response.failed", events[0][0])
	require.Contains(t, gjson.Get(events[0][1], "response.error.message").String(), "upstream exploded")
	streamErr, ok := GetOpsStreamError(c)
	require.True(t, ok)
	require.Equal(t, http.StatusBadGateway, streamErr.IntendedStatus)
}

func TestOpenAICompactKeepaliveAdjustedWrittenSize_ExcludesHeartbeatBytes(t *testing.T) {
	c, rec := newCompactBridgeTestContext(t, true)
	stop := StartOpenAICompactSSEKeepalive(c, keepaliveTestInterval)
	defer stop()
	before := OpenAICompactKeepaliveAdjustedWrittenSize(c)
	waitForKeepaliveBeats()
	require.Equal(t, before, OpenAICompactKeepaliveAdjustedWrittenSize(c))

	_, err := c.Writer.Write([]byte("real-bytes"))
	require.NoError(t, err)
	require.Equal(t, len("real-bytes"), OpenAICompactKeepaliveAdjustedWrittenSize(c))
	require.Contains(t, rec.Body.String(), ": keepalive\n\n")
}

func TestWriteOpenAIFastPolicyBlockedResponse_AfterKeepaliveCommit(t *testing.T) {
	c, rec := newCompactBridgeTestContext(t, true)
	stop := StartOpenAICompactSSEKeepalive(c, keepaliveTestInterval)
	defer stop()
	waitForKeepaliveBeats()

	writeOpenAIFastPolicyBlockedResponse(c, &OpenAIFastBlockedError{Message: "tier blocked"})

	events := parseCompactBridgeSSE(t, stripKeepaliveComments(rec.Body.String()))
	require.Len(t, events, 1)
	require.Equal(t, "permission_error", gjson.Get(events[0][1], "response.error.code").String())
}

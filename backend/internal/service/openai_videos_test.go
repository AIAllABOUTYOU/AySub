//go:build unit

package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestForwardGrokVideosCreatesCompletedJob(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("DATA_DIR", t.TempDir())

	upstreamBody := strings.Join([]string{
		`data: {"result":{"response":{"streamingVideoGenerationResponse":{"progress":25,"videoPostId":"video-post-1"}}}}`,
		`data: {"result":{"response":{"streamingVideoGenerationResponse":{"progress":100,"videoPostId":"video-post-1","videoUrl":"generated/video.mp4","assetId":"asset-1","thumbnailImageUrl":"generated/thumb.jpg","moderated":false}}}}`,
		`data: [DONE]`,
		``,
	}, "\n")
	upstream := &httpUpstreamRecorder{responses: []*http.Response{
		{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"post":{"id":"parent-post-1"}}`)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"X-Request-Id": []string{"grok-video-1"}},
			Body:       io.NopCloser(strings.NewReader(upstreamBody)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"video/mp4"}},
			Body:       io.NopCloser(strings.NewReader("video-bytes")),
		},
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream}

	body := []byte(`{"model":"grok-imagine-video","prompt":"cinematic skyline","seconds":6,"size":"1280x720","preset":"normal"}`)
	parsed, err := svc.ParseOpenAIVideoRequest(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", strings.NewReader(string(body)))
	c.Request.Header.Set("Content-Type", "application/json")

	account := &Account{
		ID:       99,
		Name:     "grok-cookie",
		Platform: PlatformXAI,
		Type:     AccountTypeCookie,
		Credentials: map[string]any{
			"base_url":  "https://grok.example",
			"sso_token": "tok",
		},
		Concurrency: 1,
	}

	result, err := svc.ForwardVideos(context.Background(), c, account, parsed, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.requests, 3)
	require.Equal(t, "https://grok.example/rest/media/post/create", upstream.requests[0].URL.String())
	require.Equal(t, "https://grok.example/rest/app-chat/conversations/new", upstream.requests[1].URL.String())
	require.Equal(t, "https://assets.grok.com/generated/video.mp4", upstream.requests[2].URL.String())
	require.Equal(t, "MEDIA_POST_TYPE_VIDEO", gjson.GetBytes(upstream.bodies[0], "mediaType").String())
	require.Equal(t, "cinematic skyline", gjson.GetBytes(upstream.bodies[0], "prompt").String())
	require.Equal(t, "imagine-video-gen", gjson.GetBytes(upstream.bodies[1], "modelName").String())
	require.Equal(t, "parent-post-1", gjson.GetBytes(upstream.bodies[1], "responseMetadata.modelConfigOverride.modelMap.videoGenModelConfig.parentPostId").String())
	require.Equal(t, "16:9", gjson.GetBytes(upstream.bodies[1], "responseMetadata.modelConfigOverride.modelMap.videoGenModelConfig.aspectRatio").String())
	require.Equal(t, int64(6), gjson.GetBytes(upstream.bodies[1], "responseMetadata.modelConfigOverride.modelMap.videoGenModelConfig.videoLength").Int())
	require.Contains(t, gjson.GetBytes(upstream.bodies[1], "message").String(), "--mode=normal")

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "video", gjson.Get(rec.Body.String(), "object").String())
	require.Equal(t, "completed", gjson.Get(rec.Body.String(), "status").String())
	require.Equal(t, int64(100), gjson.Get(rec.Body.String(), "progress").Int())
	require.Equal(t, "grok-imagine-video", gjson.Get(rec.Body.String(), "model").String())
	require.Equal(t, result.ResponseID, gjson.Get(rec.Body.String(), "id").String())

	contentURL, found, err := svc.GetVideoContentURL(result.ResponseID)
	require.NoError(t, err)
	require.True(t, found)
	require.Contains(t, contentURL, "/v1/files/video?id=")
}

func TestForwardGrokLiveKitTokenReturnsWebSocketURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Request-Id": []string{"livekit-token-1"}},
		Body:       io.NopCloser(strings.NewReader(`{"access_token":"lk-test-token","room":"room-1"}`)),
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream}

	body := []byte(`{"voice":"rex","speed":1.25,"instructions":"keep answers short"}`)
	parsed, err := svc.ParseGrokLiveKitTokenRequest(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/livekit/tokens", strings.NewReader(string(body)))
	c.Request.Header.Set("Content-Type", "application/json")

	account := &Account{
		ID:       99,
		Name:     "grok-cookie",
		Platform: PlatformXAI,
		Type:     AccountTypeCookie,
		Credentials: map[string]any{
			"base_url":  "https://grok.example",
			"sso_token": "tok",
		},
		Concurrency: 1,
	}

	result, err := svc.ForwardGrokLiveKitToken(context.Background(), c, account, parsed)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "https://grok.example/rest/livekit/tokens", upstream.lastReq.URL.String())
	require.Equal(t, "livekit-token-1", result.RequestID)

	sessionPayload := gjson.GetBytes(upstream.lastBody, "sessionPayload").String()
	require.Equal(t, "rex", gjson.Get(sessionPayload, "voice").String())
	require.Equal(t, 1.25, gjson.Get(sessionPayload, "playback_speed").Float())
	require.Equal(t, "keep answers short", gjson.Get(sessionPayload, "instructions").String())
	require.True(t, gjson.Get(sessionPayload, "is_raw_instructions").Bool())
	require.Equal(t, "wss://livekit.grok.com", gjson.GetBytes(upstream.lastBody, "livekitUrl").String())
	require.Equal(t, "true", gjson.GetBytes(upstream.lastBody, "params.enable_markdown_transcript").String())

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "lk-test-token", gjson.Get(rec.Body.String(), "access_token").String())
	livekitURL := gjson.Get(rec.Body.String(), "livekit_url").String()
	require.Contains(t, livekitURL, "wss://livekit.grok.com/rtc?")
	require.Contains(t, livekitURL, "access_token=lk-test-token")
	require.Contains(t, livekitURL, "sdk=js")
	require.Equal(t, "wss://livekit.grok.com", gjson.Get(rec.Body.String(), "ws_base").String())
}

func TestBuildGrokLiveKitWSHeadersUsesGrokCookie(t *testing.T) {
	account := &Account{
		ID:       99,
		Name:     "grok-cookie",
		Platform: PlatformXAI,
		Type:     AccountTypeCookie,
		Credentials: map[string]any{
			"sso_token":  "tok",
			"user_agent": "test-agent",
			"origin":     "https://grok.com",
			"referer":    "https://grok.com/",
		},
	}

	headers, err := buildGrokLiveKitWSHeaders(account)
	require.NoError(t, err)
	require.Contains(t, headers.Get("Cookie"), "sso")
	require.Contains(t, headers.Get("Cookie"), "tok")
	require.Equal(t, "https://grok.com", headers.Get("Origin"))
	require.Equal(t, "https://grok.com/", headers.Get("Referer"))
	require.Equal(t, "test-agent", headers.Get("User-Agent"))
}

func TestProxyGrokLiveKitRTCRelaysBidirectionalFrames(t *testing.T) {
	upstream := newLiveKitTestFrameConn()
	dialer := &liveKitTestDialer{conn: upstream}
	svc := &OpenAIGatewayService{openaiWSPassthroughDialer: dialer}
	account := &Account{
		ID:       99,
		Name:     "grok-cookie",
		Platform: PlatformXAI,
		Type:     AccountTypeCookie,
		Credentials: map[string]any{
			"sso_token": "tok",
		},
		Concurrency: 1,
	}

	errCh := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := coderws.Accept(w, r, &coderws.AcceptOptions{CompressionMode: coderws.CompressionContextTakeover})
		if err != nil {
			errCh <- err
			return
		}
		defer func() { _ = conn.CloseNow() }()
		_, err = svc.ProxyGrokLiveKitRTC(r.Context(), conn, account, "lk-test-token")
		errCh <- err
	}))
	defer server.Close()

	dialCtx, cancelDial := context.WithTimeout(context.Background(), 3*time.Second)
	client, _, err := coderws.Dial(dialCtx, "ws"+strings.TrimPrefix(server.URL, "http"), nil)
	cancelDial()
	require.NoError(t, err)
	defer func() { _ = client.CloseNow() }()

	writeCtx, cancelWrite := context.WithTimeout(context.Background(), time.Second)
	err = client.Write(writeCtx, coderws.MessageText, []byte("client-frame"))
	cancelWrite()
	require.NoError(t, err)

	require.Equal(t, liveKitTestFrame{msgType: coderws.MessageText, payload: []byte("client-frame")}, upstream.nextWritten(t))
	upstream.queueRead(coderws.MessageBinary, []byte("upstream-frame"))

	readCtx, cancelRead := context.WithTimeout(context.Background(), time.Second)
	msgType, payload, err := client.Read(readCtx)
	cancelRead()
	require.NoError(t, err)
	require.Equal(t, coderws.MessageBinary, msgType)
	require.Equal(t, []byte("upstream-frame"), payload)

	require.Equal(t, buildGrokLiveKitWSURL("lk-test-token"), dialer.wsURL)
	require.Contains(t, dialer.headers.Get("Cookie"), "sso=tok")

	_ = client.Close(coderws.StatusNormalClosure, "done")
	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("waiting for livekit rtc proxy to finish timed out")
	}
}

type liveKitTestFrame struct {
	msgType coderws.MessageType
	payload []byte
}

type liveKitTestFrameConn struct {
	readCh  chan liveKitTestFrame
	writeCh chan liveKitTestFrame
	closed  chan struct{}
	once    sync.Once
}

func newLiveKitTestFrameConn() *liveKitTestFrameConn {
	return &liveKitTestFrameConn{
		readCh:  make(chan liveKitTestFrame, 4),
		writeCh: make(chan liveKitTestFrame, 4),
		closed:  make(chan struct{}),
	}
}

func (c *liveKitTestFrameConn) queueRead(msgType coderws.MessageType, payload []byte) {
	c.readCh <- liveKitTestFrame{msgType: msgType, payload: append([]byte(nil), payload...)}
}

func (c *liveKitTestFrameConn) nextWritten(t *testing.T) liveKitTestFrame {
	t.Helper()
	select {
	case frame := <-c.writeCh:
		return frame
	case <-time.After(time.Second):
		t.Fatal("waiting for upstream write timed out")
		return liveKitTestFrame{}
	}
}

func (c *liveKitTestFrameConn) ReadFrame(ctx context.Context) (coderws.MessageType, []byte, error) {
	select {
	case frame := <-c.readCh:
		return frame.msgType, append([]byte(nil), frame.payload...), nil
	case <-c.closed:
		return coderws.MessageText, nil, coderws.CloseError{Code: coderws.StatusNormalClosure}
	case <-ctx.Done():
		return coderws.MessageText, nil, ctx.Err()
	}
}

func (c *liveKitTestFrameConn) WriteFrame(ctx context.Context, msgType coderws.MessageType, payload []byte) error {
	select {
	case c.writeCh <- liveKitTestFrame{msgType: msgType, payload: append([]byte(nil), payload...)}:
		return nil
	case <-c.closed:
		return coderws.CloseError{Code: coderws.StatusNormalClosure}
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *liveKitTestFrameConn) WriteJSON(context.Context, any) error {
	return nil
}

func (c *liveKitTestFrameConn) ReadMessage(ctx context.Context) ([]byte, error) {
	_, payload, err := c.ReadFrame(ctx)
	return payload, err
}

func (c *liveKitTestFrameConn) Ping(context.Context) error {
	return nil
}

func (c *liveKitTestFrameConn) Close() error {
	c.once.Do(func() {
		close(c.closed)
	})
	return nil
}

type liveKitTestDialer struct {
	conn    *liveKitTestFrameConn
	wsURL   string
	headers http.Header
}

func (d *liveKitTestDialer) Dial(ctx context.Context, wsURL string, headers http.Header, proxyURL string) (openAIWSClientConn, int, http.Header, error) {
	_ = ctx
	_ = proxyURL
	d.wsURL = wsURL
	d.headers = cloneHeader(headers)
	return d.conn, 0, nil, nil
}

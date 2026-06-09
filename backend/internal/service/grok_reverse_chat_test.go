package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"sync"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestBuildGrokWebCookieHeader(t *testing.T) {
	account := &Account{
		Platform: PlatformXAI,
		Type:     AccountTypeCookie,
		Credentials: map[string]any{
			"cookie":       "sso=tok; other=1",
			"cf_cookies":   "foo=bar; cf_clearance=old",
			"cf_clearance": "new",
		},
	}

	got := buildGrokWebCookieHeader(account)

	require.Contains(t, got, "sso=tok")
	require.Contains(t, got, "sso-rw=tok")
	require.Contains(t, got, "foo=bar")
	require.Contains(t, got, "cf_clearance=new")
	require.NotContains(t, got, "cf_clearance=old")
}

func TestGrokDynamicStatsigCompatibilityHeaders(t *testing.T) {
	account := &Account{
		Platform: PlatformXAI,
		Type:     AccountTypeCookie,
		Credentials: map[string]any{
			"sso_token": "tok",
		},
	}
	svc := &OpenAIGatewayService{cfg: &config.Config{Grok: config.GrokConfig{DynamicStatsigEnabled: true}}}

	req, err := svc.buildGrokWebChatRequest(context.Background(), account, []byte(`{"message":"hi"}`))
	require.NoError(t, err)
	decoded, err := base64.StdEncoding.DecodeString(req.Header.Get(grokDynamicStatsigHeader))
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(string(decoded), "x1:TypeError:"), string(decoded))

	account.Extra = map[string]any{"dynamic_statsig_enabled": false}
	req, err = svc.buildGrokWebChatRequest(context.Background(), account, []byte(`{"message":"hi"}`))
	require.NoError(t, err)
	require.Empty(t, req.Header.Get(grokDynamicStatsigHeader))
}

func TestGrokDynamicStatsigAccountOverrideForRateLimits(t *testing.T) {
	account := &Account{
		Platform: PlatformXAI,
		Type:     AccountTypeCookie,
		Credentials: map[string]any{
			"base_url":                "https://grok.example",
			"sso_token":               "tok",
			"dynamic_statsig_enabled": true,
		},
	}
	req, err := buildGrokWebRateLimitsRequest(context.Background(), account, []byte(`{"modelName":"fast"}`), nil)
	require.NoError(t, err)

	decoded, err := base64.StdEncoding.DecodeString(req.Header.Get(grokDynamicStatsigHeader))
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(string(decoded), "x1:TypeError:"), string(decoded))
}

func TestResolveGrokWebModeIDPrefersFastForGrok43Fast(t *testing.T) {
	require.Equal(t, "fast", resolveGrokWebModeID(nil, "grok-4.3-fast"))
	require.Equal(t, "grok-420-computer-use-sa", resolveGrokWebModeID(nil, "grok-4.3-high"))
}

func TestBuildGrokWebChatPayload(t *testing.T) {
	account := &Account{
		Platform: PlatformXAI,
		Type:     AccountTypeCookie,
		Credentials: map[string]any{
			"disable_search":     true,
			"custom_instruction": "be concise",
		},
	}

	payload, err := buildGrokWebChatPayload(account, "grok-4.20-expert", "[user]: hi", []string{"file-1"})
	require.NoError(t, err)

	raw := string(payload)
	require.Equal(t, "expert", gjson.Get(raw, "modeId").String())
	require.True(t, gjson.Get(raw, "disableSearch").Bool())
	require.Equal(t, "[user]: hi", gjson.Get(raw, "message").String())
	require.Equal(t, "file-1", gjson.Get(raw, "fileAttachments.0").String())
	require.Equal(t, "be concise", gjson.Get(raw, "customPersonality").String())
	require.True(t, gjson.Get(raw, "sendFinalMetadata").Bool())
}

func TestBuildGrokConsolePayloadFromChat(t *testing.T) {
	body := []byte(`{"model":"grok-4.3-high","temperature":0.2,"top_p":0.8,"messages":[{"role":"system","content":"be precise"},{"role":"user","content":[{"type":"text","text":"describe this"},{"type":"image_url","image_url":{"url":"https://example.com/image.png"}}]}]}`)

	payload, prompt, err := buildGrokConsolePayloadFromChat(body, "grok-4.3-high", true)
	require.NoError(t, err)

	require.Equal(t, "[system]: be precise\n\n[user]: describe this", prompt)
	require.Equal(t, "grok-4.3", gjson.GetBytes(payload, "model").String())
	require.True(t, gjson.GetBytes(payload, "stream").Bool())
	require.Equal(t, "high", gjson.GetBytes(payload, "reasoning.effort").String())
	require.Equal(t, 0.2, gjson.GetBytes(payload, "temperature").Float())
	require.Equal(t, 0.8, gjson.GetBytes(payload, "top_p").Float())
	require.Equal(t, "system", gjson.GetBytes(payload, "input.0.role").String())
	require.Equal(t, "input_image", gjson.GetBytes(payload, "input.1.content.1.type").String())
	require.Equal(t, "https://example.com/image.png", gjson.GetBytes(payload, "input.1.content.1.image_url").String())
	require.Equal(t, "web_search", gjson.GetBytes(payload, "tools.0.type").String())
	require.Equal(t, "x_search", gjson.GetBytes(payload, "tools.1.type").String())
}

func TestBuildGrokConsolePayloadBuildModelLimits(t *testing.T) {
	payload, _, err := buildGrokConsolePayloadFromResponses([]byte(`{"input":"build it"}`), "grok-build-console", false)
	require.NoError(t, err)

	require.Equal(t, "grok-build-0.1", gjson.GetBytes(payload, "model").String())
	require.Equal(t, int64(256000), gjson.GetBytes(payload, "max_output_tokens").Int())
	require.False(t, gjson.GetBytes(payload, "stream").Bool())
	require.False(t, gjson.GetBytes(payload, "reasoning").Exists())
	require.Equal(t, "auto", gjson.GetBytes(payload, "tool_choice").String())
}

func TestFlattenGrokChatMessagesExtractsDataURIFileInputs(t *testing.T) {
	body := []byte(`{"messages":[{"role":"user","content":[{"type":"text","text":"describe this"},{"type":"image_url","image_url":{"url":"data:image/png;base64,aGVsbG8="}},{"type":"image_url","image_url":{"url":"https://example.com/image.png"}}]}]}`)

	prompt, files, err := flattenGrokChatMessages(body)
	require.NoError(t, err)
	require.Equal(t, "[user]: describe this\n[image] https://example.com/image.png", prompt)
	require.Len(t, files, 1)
	require.Equal(t, "file.png", files[0].Filename)
	require.Equal(t, "image/png", files[0].Mime)
	require.Equal(t, "aGVsbG8=", files[0].Content)
}

func TestForwardGrokConsoleChatCompletionsNonStreaming(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Request-Id": []string{"grok-console-chat-1"}},
		Body:       io.NopCloser(strings.NewReader(`{"output":[{"type":"message","content":[{"type":"output_text","text":"Hello console"}]}],"usage":{"input_tokens":7,"output_tokens":3}}`)),
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{}`))

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
	body := []byte(`{"model":"grok-4.3-console","stream":false,"messages":[{"role":"user","content":"hi"}]}`)

	result, err := svc.ForwardGrokChatCompletions(c.Request.Context(), c, account, body, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "grok-console-chat-1", result.RequestID)
	require.Equal(t, "grok-4.3-console", result.Model)
	require.Equal(t, "grok-4.3-console", result.BillingModel)
	require.Equal(t, "grok-4.3-console", result.UpstreamModel)
	require.False(t, result.Stream)
	require.Equal(t, 7, result.Usage.InputTokens)
	require.Equal(t, 3, result.Usage.OutputTokens)

	require.NotNil(t, upstream.lastReq)
	require.Equal(t, []bool{true}, upstream.tlsFlags)
	require.Equal(t, "https://console.x.ai/v1/responses", upstream.lastReq.URL.String())
	require.Equal(t, "Bearer anonymous", upstream.lastReq.Header.Get("Authorization"))
	require.Contains(t, upstream.lastReq.Header.Get("Cookie"), "sso=tok")
	require.Contains(t, upstream.lastReq.Header.Get("Cookie"), "sso-rw=tok")
	require.NotEmpty(t, upstream.lastReq.Header.Get("x-statsig-id"))
	require.NotEmpty(t, upstream.lastReq.Header.Get("Sec-Ch-Ua"))
	require.Equal(t, "u=1, i", upstream.lastReq.Header.Get("Priority"))
	require.Equal(t, "grok-4.3", gjson.GetBytes(upstream.lastBody, "model").String())
	require.Equal(t, "hi", gjson.GetBytes(upstream.lastBody, "input.0.content.0.text").String())

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "Hello console", gjson.Get(rec.Body.String(), "choices.0.message.content").String())
	require.Equal(t, int64(7), gjson.Get(rec.Body.String(), "usage.prompt_tokens").Int())
}

func TestForwardGrokConsoleResponsesStreaming(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstreamBody := strings.Join([]string{
		"event: response.output_text.delta\ndata: {\"delta\":\"Hello\"}",
		"event: response.output_text.delta\ndata: {\"delta\":\" console\"}",
		"event: response.completed\ndata: {\"response\":{\"usage\":{\"input_tokens\":5,\"output_tokens\":2}}}",
		"",
	}, "\n\n")
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Request-Id": []string{"grok-console-resp-stream-1"}},
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{}`))

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
	body := []byte(`{"model":"grok-4.20-multi-agent-low","stream":true,"input":"hi"}`)

	result, err := svc.Forward(c.Request.Context(), c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Stream)
	require.Equal(t, 5, result.Usage.InputTokens)
	require.Equal(t, 2, result.Usage.OutputTokens)

	require.Equal(t, "grok-4.20-multi-agent-0309", gjson.GetBytes(upstream.lastBody, "model").String())
	require.Equal(t, "low", gjson.GetBytes(upstream.lastBody, "reasoning.effort").String())
	require.Equal(t, int64(2000000), gjson.GetBytes(upstream.lastBody, "max_output_tokens").Int())

	raw := rec.Body.String()
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, raw, `"type":"response.created"`)
	require.Contains(t, raw, `"delta":"Hello"`)
	require.Contains(t, raw, `"delta":" console"`)
	require.Contains(t, raw, `"type":"response.completed"`)
	require.Contains(t, raw, "data: [DONE]")
}

func TestForwardGrokConsoleAnthropicMessagesNonStreaming(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Request-Id": []string{"grok-console-msg-1"}},
		Body:       io.NopCloser(strings.NewReader(`{"output":[{"type":"message","content":[{"type":"output_text","text":"Hello anthropic"}]}],"usage":{"input_tokens":6,"output_tokens":4}}`)),
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{}`))

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
	body := []byte(`{"model":"grok-build-console","max_tokens":64,"stream":false,"messages":[{"role":"user","content":"hi"}]}`)

	result, err := svc.ForwardAsAnthropic(c.Request.Context(), c, account, body, "", "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.Stream)
	require.Equal(t, "grok-console-msg-1", result.RequestID)
	require.Equal(t, "grok-build-0.1", gjson.GetBytes(upstream.lastBody, "model").String())

	raw := rec.Body.String()
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "message", gjson.Get(raw, "type").String())
	require.Equal(t, "Hello anthropic", gjson.Get(raw, "content.0.text").String())
	require.Equal(t, int64(6), gjson.Get(raw, "usage.input_tokens").Int())
	require.Equal(t, int64(4), gjson.Get(raw, "usage.output_tokens").Int())
}

func TestParseGrokWebStreamLine(t *testing.T) {
	line := `data: {"result":{"response":{"token":"Hello <grok:render card_id=\"1\" card_type=\"x\" type=\"y\">drop</grok:render>world","messageTag":"final"}}}`

	ev, ok, done, err := parseGrokWebStreamLine(line)
	require.NoError(t, err)
	require.True(t, ok)
	require.False(t, done)
	require.Equal(t, "text", ev.Kind)
	require.Equal(t, "Hello world", ev.Content)

	_, ok, done, err = parseGrokWebStreamLine(`data: {"result":{"response":{"isSoftStop":true}}}`)
	require.NoError(t, err)
	require.False(t, ok)
	require.True(t, done)
}

func TestGrokWebStreamAdapterSearchAnnotationsAndImages(t *testing.T) {
	adapter := newGrokWebStreamAdapter()
	citationCard := `{"id":"cite-1","url":"https://example.com/source","title":"Example Source"}`
	imageCard := `{"id":"img-1","image_chunk":{"progress":100,"imageUuid":"image-uuid","imageUrl":"generated/image.jpg","moderated":false}}`

	_, done, err := adapter.FeedLine(grokDataFrame(map[string]any{
		"cardAttachment": map[string]any{"jsonData": citationCard},
		"webSearchResults": map[string]any{
			"results": []map[string]any{
				{"url": "https://example.com/source", "title": "Example Source"},
			},
		},
	}))
	require.NoError(t, err)
	require.False(t, done)

	events, done, err := adapter.FeedLine(`data: {"result":{"response":{"token":"Answer<grok:render card_id=\"cite-1\" card_type=\"citation\" type=\"render_inline_citation\">drop</grok:render>","messageTag":"final"}}}`)
	require.NoError(t, err)
	require.False(t, done)
	require.Len(t, events, 2)
	require.Equal(t, "text", events[0].Kind)
	require.Equal(t, "Answer [[1]](https://example.com/source)", events[0].Content)
	require.Equal(t, "annotation", events[1].Kind)
	require.Equal(t, "https://example.com/source", events[1].Annotation.URL)

	events, done, err = adapter.FeedLine(grokDataFrame(map[string]any{
		"cardAttachment": map[string]any{"jsonData": imageCard},
	}))
	require.NoError(t, err)
	require.False(t, done)
	require.Len(t, events, 1)
	require.Equal(t, "image", events[0].Kind)
	require.Equal(t, "https://assets.grok.com/generated/image.jpg", events[0].Content)

	require.Equal(t, "\n\n![image](https://assets.grok.com/generated/image.jpg)", adapter.ImageMarkdownSuffix())
	require.Equal(t, "https://example.com/source", adapter.ChatAnnotations()[0]["url_citation"].(map[string]any)["url"])
	require.Equal(t, "https://example.com/source", adapter.SearchSources()[0]["url"])
}

func TestForwardGrokChatCompletionsNonStreaming(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstreamBody := strings.Join([]string{
		`data: {"result":{"response":{"token":"thinking","isThinking":true}}}`,
		`data: {"result":{"response":{"token":"Hello","messageTag":"final"}}}`,
		`data: {"result":{"response":{"token":" world","messageTag":"final"}}}`,
		`data: {"result":{"response":{"isSoftStop":true}}}`,
		``,
	}, "\n\n")
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Request-Id": []string{"grok-req-1"}},
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{}`))

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
	body := []byte(`{"model":"grok-4.20-auto","stream":false,"messages":[{"role":"user","content":"hi"}]}`)

	result, err := svc.ForwardGrokChatCompletions(c.Request.Context(), c, account, body, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "grok-req-1", result.RequestID)
	require.Equal(t, "grok-4.20-auto", result.Model)
	require.Equal(t, "grok-4.20-auto", result.BillingModel)
	require.False(t, result.Stream)
	require.Positive(t, result.Usage.InputTokens)
	require.Positive(t, result.Usage.OutputTokens)

	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "https://grok.example/rest/app-chat/conversations/new", upstream.lastReq.URL.String())
	require.Contains(t, upstream.lastReq.Header.Get("Cookie"), "sso=tok")
	require.Contains(t, upstream.lastReq.Header.Get("Cookie"), "sso-rw=tok")
	require.Equal(t, "application/json", upstream.lastReq.Header.Get("Content-Type"))
	require.Equal(t, "auto", gjson.GetBytes(upstream.lastBody, "modeId").String())
	require.Equal(t, "[user]: hi", gjson.GetBytes(upstream.lastBody, "message").String())

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "Hello world", gjson.Get(rec.Body.String(), "choices.0.message.content").String())
	require.Equal(t, "thinking", gjson.Get(rec.Body.String(), "choices.0.message.reasoning_content").String())
}

func TestForwardGrokChatCompletionsNormalizesGrok42Alias(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstreamBody := strings.Join([]string{
		`data: {"result":{"response":{"token":"Hello","messageTag":"final"}}}`,
		`data: {"result":{"response":{"isSoftStop":true}}}`,
		``,
	}, "\n\n")
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Request-Id": []string{"grok-alias-1"}},
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{}`))

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
	body := []byte(`{"model":"grok-4.2-fast","stream":false,"messages":[{"role":"user","content":"hi"}]}`)

	result, err := svc.ForwardGrokChatCompletions(c.Request.Context(), c, account, body, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "grok-4.2-fast", result.Model)
	require.Equal(t, "grok-4.2-fast", result.BillingModel)
	require.Equal(t, "grok-4.20-fast", result.UpstreamModel)
	require.Equal(t, "fast", gjson.GetBytes(upstream.lastBody, "modeId").String())
}

func TestForwardGrokChatCompletionsRefreshesCloudflareClearance(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var solverPayload struct {
		Cmd   string `json:"cmd"`
		URL   string `json:"url"`
		Proxy struct {
			URL string `json:"url"`
		} `json:"proxy"`
	}
	solver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&solverPayload))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":"ok","solution":{"userAgent":"Mozilla/5.0 Solver","cookies":[{"name":"cf_clearance","value":"cf-new","domain":".grok.example"},{"name":"other","value":"1","domain":".grok.example"}]}}`)
	}))
	defer solver.Close()

	upstreamBody := strings.Join([]string{
		`data: {"result":{"response":{"token":"Solved","messageTag":"final"}}}`,
		`data: {"result":{"response":{"isSoftStop":true}}}`,
		``,
	}, "\n\n")
	upstream := &httpUpstreamRecorder{responses: []*http.Response{
		{
			StatusCode: http.StatusForbidden,
			Header:     http.Header{"Cf-Mitigated": []string{"challenge"}},
			Body:       io.NopCloser(strings.NewReader(`<!doctype html><title>Just a moment...</title>`)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"X-Request-Id": []string{"grok-cf-2"}},
			Body:       io.NopCloser(strings.NewReader(upstreamBody)),
		},
	}}
	svc := &OpenAIGatewayService{
		httpUpstream: upstream,
		cfg: &config.Config{Grok: config.GrokConfig{
			FlareSolverrURL:            solver.URL,
			FlareSolverrTimeoutSeconds: 2,
			DynamicStatsigEnabled:      true,
		}},
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{}`))

	account := &Account{
		ID:       99,
		Name:     "grok-cookie",
		Platform: PlatformXAI,
		Type:     AccountTypeCookie,
		Credentials: map[string]any{
			"base_url":  "https://grok.example/rest",
			"sso_token": "tok",
		},
		Proxy:       &Proxy{Protocol: "socks5", Host: "warp", Port: 1080, Status: StatusActive},
		Concurrency: 1,
	}
	body := []byte(`{"model":"grok-4.20-auto","stream":false,"messages":[{"role":"user","content":"hi"}]}`)

	result, err := svc.ForwardGrokChatCompletions(c.Request.Context(), c, account, body, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, upstream.requests, 2)
	require.Equal(t, []bool{true, true}, upstream.tlsFlags)
	require.Equal(t, "request.get", solverPayload.Cmd)
	require.Equal(t, "https://grok.example", solverPayload.URL)
	require.Equal(t, "socks5://warp:1080", solverPayload.Proxy.URL)
	require.NotEmpty(t, upstream.requests[0].Header.Get("x-statsig-id"))
	require.NotEmpty(t, upstream.requests[0].Header.Get("Sec-Ch-Ua"))
	require.Equal(t, "u=1, i", upstream.requests[0].Header.Get("Priority"))
	require.Contains(t, upstream.requests[0].Header.Get("Baggage"), "sentry-environment=production")
	require.Contains(t, upstream.requests[1].Header.Get("Cookie"), "cf_clearance=cf-new")
	require.Contains(t, upstream.requests[1].Header.Get("Cookie"), "other=1")
	require.Equal(t, "Mozilla/5.0 Solver", upstream.requests[1].Header.Get("User-Agent"))
	require.Equal(t, "cf-new", account.GetCredential("cf_clearance"))
	require.Equal(t, "Solved", gjson.Get(rec.Body.String(), "choices.0.message.content").String())
}

func TestForwardGrokChatCompletionsRetriesAntiBotJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	solver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":"ok","solution":{"userAgent":"Mozilla/5.0 Solver","cookies":[{"name":"cf_clearance","value":"cf-new","domain":".grok.example"}]}}`)
	}))
	defer solver.Close()

	upstreamBody := strings.Join([]string{
		`data: {"result":{"response":{"token":"Solved","messageTag":"final"}}}`,
		`data: {"result":{"response":{"isSoftStop":true}}}`,
		``,
	}, "\n\n")
	upstream := &httpUpstreamRecorder{responses: []*http.Response{
		{
			StatusCode: http.StatusForbidden,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"error":{"code":7,"message":"Request rejected by anti-bot rules.","details":[]}}`)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"X-Request-Id": []string{"grok-antibot-2"}},
			Body:       io.NopCloser(strings.NewReader(upstreamBody)),
		},
	}}
	svc := &OpenAIGatewayService{
		httpUpstream: upstream,
		cfg: &config.Config{Grok: config.GrokConfig{
			FlareSolverrURL:            solver.URL,
			FlareSolverrTimeoutSeconds: 2,
			DynamicStatsigEnabled:      true,
		}},
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{}`))

	account := &Account{
		ID:       100,
		Name:     "grok-cookie",
		Platform: PlatformXAI,
		Type:     AccountTypeCookie,
		Credentials: map[string]any{
			"base_url":  "https://grok.example/rest",
			"sso_token": "tok",
		},
		Concurrency: 1,
	}
	body := []byte(`{"model":"grok-4.20-fast","stream":false,"messages":[{"role":"user","content":"hi"}]}`)

	result, err := svc.ForwardGrokChatCompletions(c.Request.Context(), c, account, body, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.requests, 2)
	require.Contains(t, upstream.requests[1].Header.Get("Cookie"), "cf_clearance=cf-new")
	require.Equal(t, "Solved", gjson.Get(rec.Body.String(), "choices.0.message.content").String())
}

func TestForwardGrokChatCompletionsUploadsDataURIFileAttachments(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstreamBody := strings.Join([]string{
		`data: {"result":{"response":{"token":"done","messageTag":"final"}}}`,
		`data: {"result":{"response":{"isSoftStop":true}}}`,
		``,
	}, "\n\n")
	upstream := &httpUpstreamRecorder{responses: []*http.Response{
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"X-Request-Id": []string{"grok-upload-1"}},
			Body:       io.NopCloser(strings.NewReader(`{"fileMetadataId":"file-123"}`)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"X-Request-Id": []string{"grok-chat-1"}},
			Body:       io.NopCloser(strings.NewReader(upstreamBody)),
		},
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{}`))

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
	body := []byte(`{"model":"grok-4.20-auto","stream":false,"messages":[{"role":"user","content":[{"type":"text","text":"describe this"},{"type":"image_url","image_url":{"url":"data:image/png;base64,aGVsbG8="}}]}]}`)

	result, err := svc.ForwardGrokChatCompletions(c.Request.Context(), c, account, body, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.requests, 2)
	require.Equal(t, "https://grok.example/rest/app-chat/upload-file", upstream.requests[0].URL.String())
	require.Equal(t, "https://grok.example/rest/app-chat/conversations/new", upstream.requests[1].URL.String())
	require.Equal(t, "file.png", gjson.GetBytes(upstream.bodies[0], "fileName").String())
	require.Equal(t, "image/png", gjson.GetBytes(upstream.bodies[0], "fileMimeType").String())
	require.Equal(t, "aGVsbG8=", gjson.GetBytes(upstream.bodies[0], "content").String())
	require.Equal(t, "file-123", gjson.GetBytes(upstream.bodies[1], "fileAttachments.0").String())
	require.Equal(t, "[user]: describe this", gjson.GetBytes(upstream.bodies[1], "message").String())
	require.Equal(t, "done", gjson.Get(rec.Body.String(), "choices.0.message.content").String())
}

func TestForwardGrokChatCompletionsNonStreamingIncludesSearchAndImages(t *testing.T) {
	gin.SetMode(gin.TestMode)

	citationCard := `{"id":"cite-1","url":"https://example.com/source","title":"Example Source"}`
	imageCard := `{"id":"img-1","image_chunk":{"progress":100,"imageUuid":"image-uuid","imageUrl":"generated/image.jpg","moderated":false}}`
	upstreamBody := strings.Join([]string{
		grokDataFrame(map[string]any{
			"cardAttachment": map[string]any{"jsonData": citationCard},
			"webSearchResults": map[string]any{
				"results": []map[string]any{
					{"url": "https://example.com/source", "title": "Example Source"},
				},
			},
		}),
		`data: {"result":{"response":{"token":"Answer<grok:render card_id=\"cite-1\" card_type=\"citation\" type=\"render_inline_citation\">drop</grok:render>","messageTag":"final"}}}`,
		grokDataFrame(map[string]any{
			"cardAttachment": map[string]any{"jsonData": imageCard},
		}),
		`data: {"result":{"response":{"isSoftStop":true}}}`,
		``,
	}, "\n\n")
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Request-Id": []string{"grok-req-2"}},
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{}`))

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
	body := []byte(`{"model":"grok-4.20-auto","stream":false,"messages":[{"role":"user","content":"hi"}]}`)

	result, err := svc.ForwardGrokChatCompletions(c.Request.Context(), c, account, body, "")
	require.NoError(t, err)
	require.NotNil(t, result)

	raw := rec.Body.String()
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, gjson.Get(raw, "choices.0.message.content").String(), "Answer [[1]](https://example.com/source)")
	require.Contains(t, gjson.Get(raw, "choices.0.message.content").String(), "![image](https://assets.grok.com/generated/image.jpg)")
	require.Equal(t, "https://example.com/source", gjson.Get(raw, "choices.0.message.annotations.0.url_citation.url").String())
	require.Equal(t, "https://example.com/source", gjson.Get(raw, "search_sources.0.url").String())
}

func TestForwardGrokResponsesNonStreaming(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstreamBody := strings.Join([]string{
		`data: {"result":{"response":{"token":"Hello","messageTag":"final"}}}`,
		`data: {"result":{"response":{"token":" world","messageTag":"final"}}}`,
		`data: {"result":{"response":{"isSoftStop":true}}}`,
		``,
	}, "\n\n")
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Request-Id": []string{"grok-resp-1"}},
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{}`))

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
	body := []byte(`{"model":"grok-4.20-auto","stream":false,"input":"hi"}`)

	result, err := svc.Forward(c.Request.Context(), c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "grok-resp-1", result.RequestID)
	require.Equal(t, "grok-4.20-auto", result.Model)
	require.False(t, result.Stream)

	raw := rec.Body.String()
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "response", gjson.Get(raw, "object").String())
	require.Equal(t, "completed", gjson.Get(raw, "status").String())
	require.Equal(t, "message", gjson.Get(raw, "output.0.type").String())
	require.Equal(t, "output_text", gjson.Get(raw, "output.0.content.0.type").String())
	require.Equal(t, "Hello world", gjson.Get(raw, "output.0.content.0.text").String())
	require.Positive(t, gjson.Get(raw, "usage.input_tokens").Int())
	require.Positive(t, gjson.Get(raw, "usage.output_tokens").Int())
}

func TestForwardGrokResponsesStreaming(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstreamBody := strings.Join([]string{
		`data: {"result":{"response":{"token":"Hello","messageTag":"final"}}}`,
		`data: {"result":{"response":{"token":" world","messageTag":"final"}}}`,
		`data: {"result":{"response":{"isSoftStop":true}}}`,
		``,
	}, "\n\n")
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Request-Id": []string{"grok-resp-stream-1"}},
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{}`))

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
	body := []byte(`{"model":"grok-4.20-auto","stream":true,"input":"hi"}`)

	result, err := svc.Forward(c.Request.Context(), c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Stream)

	raw := rec.Body.String()
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, raw, `"type":"response.created"`)
	require.Contains(t, raw, `"type":"response.output_text.delta"`)
	require.Contains(t, raw, `"delta":"Hello"`)
	require.Contains(t, raw, `"delta":" world"`)
	require.Contains(t, raw, `"type":"response.completed"`)
	require.Contains(t, raw, "data: [DONE]")
}

func TestForwardGrokAnthropicMessagesNonStreaming(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstreamBody := strings.Join([]string{
		`data: {"result":{"response":{"token":"thinking","isThinking":true}}}`,
		`data: {"result":{"response":{"token":"Hello","messageTag":"final"}}}`,
		`data: {"result":{"response":{"token":" world","messageTag":"final"}}}`,
		`data: {"result":{"response":{"isSoftStop":true}}}`,
		``,
	}, "\n\n")
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Request-Id": []string{"grok-msg-1"}},
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{}`))

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
	body := []byte(`{"model":"grok-4.20-auto","max_tokens":64,"stream":false,"messages":[{"role":"user","content":"hi"}]}`)

	result, err := svc.ForwardAsAnthropic(c.Request.Context(), c, account, body, "", "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "grok-msg-1", result.RequestID)
	require.Equal(t, "grok-4.20-auto", result.Model)
	require.False(t, result.Stream)
	require.Positive(t, result.Usage.InputTokens)
	require.Positive(t, result.Usage.OutputTokens)

	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "https://grok.example/rest/app-chat/conversations/new", upstream.lastReq.URL.String())
	require.Equal(t, "[user]: hi", gjson.GetBytes(upstream.lastBody, "message").String())

	raw := rec.Body.String()
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "message", gjson.Get(raw, "type").String())
	require.Equal(t, "thinking", gjson.Get(raw, "content.0.type").String())
	require.Equal(t, "thinking", gjson.Get(raw, "content.0.thinking").String())
	require.Equal(t, "text", gjson.Get(raw, "content.1.type").String())
	require.Equal(t, "Hello world", gjson.Get(raw, "content.1.text").String())
	require.Equal(t, "end_turn", gjson.Get(raw, "stop_reason").String())
	require.Positive(t, gjson.Get(raw, "usage.input_tokens").Int())
	require.Positive(t, gjson.Get(raw, "usage.output_tokens").Int())
}

func TestForwardGrokAnthropicMessagesStreaming(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstreamBody := strings.Join([]string{
		`data: {"result":{"response":{"token":"think","isThinking":true}}}`,
		`data: {"result":{"response":{"token":"Hello","messageTag":"final"}}}`,
		`data: {"result":{"response":{"token":" world","messageTag":"final"}}}`,
		`data: {"result":{"response":{"isSoftStop":true}}}`,
		``,
	}, "\n\n")
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Request-Id": []string{"grok-msg-stream-1"}},
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{}`))

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
	body := []byte(`{"model":"grok-4.20-auto","max_tokens":64,"stream":true,"messages":[{"role":"user","content":"hi"}]}`)

	result, err := svc.ForwardAsAnthropic(c.Request.Context(), c, account, body, "", "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Stream)
	require.Positive(t, result.Usage.InputTokens)
	require.Positive(t, result.Usage.OutputTokens)

	raw := rec.Body.String()
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, raw, "event: message_start")
	require.Contains(t, raw, `"type":"thinking_delta"`)
	require.Contains(t, raw, `"thinking":"think"`)
	require.Contains(t, raw, `"type":"text_delta"`)
	require.Contains(t, raw, `"text":"Hello"`)
	require.Contains(t, raw, `"text":" world"`)
	require.Contains(t, raw, `"stop_reason":"end_turn"`)
	require.Contains(t, raw, "event: message_stop")
}

func TestForwardGrokAnthropicMessagesUploadsImageBlock(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstreamBody := strings.Join([]string{
		`data: {"result":{"response":{"token":"done","messageTag":"final"}}}`,
		`data: {"result":{"response":{"isSoftStop":true}}}`,
		``,
	}, "\n\n")
	upstream := &httpUpstreamRecorder{responses: []*http.Response{
		{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"fileMetadataId":"file-anth-image"}`)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"X-Request-Id": []string{"grok-msg-image-1"}},
			Body:       io.NopCloser(strings.NewReader(upstreamBody)),
		},
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{}`))

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
	body := []byte(`{"model":"grok-4.20-auto","max_tokens":64,"stream":false,"messages":[{"role":"user","content":[{"type":"text","text":"describe this"},{"type":"image","source":{"type":"base64","media_type":"image/png","data":"aGVsbG8="}}]}]}`)

	result, err := svc.ForwardAsAnthropic(c.Request.Context(), c, account, body, "", "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.requests, 2)
	require.Equal(t, "https://grok.example/rest/app-chat/upload-file", upstream.requests[0].URL.String())
	require.Equal(t, "https://grok.example/rest/app-chat/conversations/new", upstream.requests[1].URL.String())
	require.Equal(t, "file.png", gjson.GetBytes(upstream.bodies[0], "fileName").String())
	require.Equal(t, "image/png", gjson.GetBytes(upstream.bodies[0], "fileMimeType").String())
	require.Equal(t, "aGVsbG8=", gjson.GetBytes(upstream.bodies[0], "content").String())
	require.Equal(t, "file-anth-image", gjson.GetBytes(upstream.bodies[1], "fileAttachments.0").String())
	require.Equal(t, "[user]: describe this", gjson.GetBytes(upstream.bodies[1], "message").String())
	require.Equal(t, "done", gjson.Get(rec.Body.String(), "content.0.text").String())
}

func TestForwardGrokImagesNonStreamingReturnsURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)

	upstream := newGrokImagineTestFrameConn()
	upstream.queueRead(coderws.MessageText, []byte(`{"type":"json","current_status":"start_stage","image_id":"image-uuid","order":0,"width":1024,"height":1536}`))
	upstream.queueRead(coderws.MessageText, []byte(`{"type":"image","url":"https://assets.grok.com/images/image-uuid.png","blob":"aW1hZ2UtYnl0ZXM=","percentage_complete":100}`))
	upstream.queueRead(coderws.MessageText, []byte(`{"type":"json","current_status":"completed","image_id":"image-uuid","order":0,"width":1024,"height":1536,"moderated":false}`))
	dialer := &grokImagineTestDialer{conn: upstream}
	svc := &OpenAIGatewayService{openaiWSPassthroughDialer: dialer}

	body := []byte(`{"prompt":"draw a cat","n":1}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(string(body)))
	c.Request.Header.Set("Content-Type", "application/json")
	parsed, err := svc.ParseOpenAIImagesRequest(c, body)
	require.NoError(t, err)

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

	result, err := svc.ForwardImages(c.Request.Context(), c, account, body, parsed, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 1, result.ImageCount)
	require.Equal(t, []string{"2K"}, result.ImageOutputSizes)
	require.NotEmpty(t, result.RequestID)
	require.Equal(t, grokImagineWSURL, dialer.wsURL)
	require.Contains(t, dialer.headers.Get("Cookie"), "sso=tok")

	raw := rec.Body.String()
	require.Equal(t, http.StatusOK, rec.Code)
	localURL := gjson.Get(raw, "data.0.url").String()
	require.Contains(t, localURL, "/v1/files/image?id=")
	imageID := strings.TrimPrefix(localURL, "/v1/files/image?id=")
	require.FileExists(t, localMediaDir(localMediaImage)+"/"+imageID+".png")
	require.Equal(t, "draw a cat", gjson.Get(raw, "data.0.revised_prompt").String())
	require.Equal(t, int64(1), gjson.Get(raw, "usage.output_tokens").Int())
}

func TestForwardGrokImagesB64DownloadsAsset(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := newGrokImagineTestFrameConn()
	upstream.queueRead(coderws.MessageText, []byte(`{"type":"json","current_status":"start_stage","image_id":"image-uuid","order":0,"width":1024,"height":1536}`))
	upstream.queueRead(coderws.MessageText, []byte(`{"type":"image","url":"https://assets.grok.com/images/image-uuid.png","blob":"aW1hZ2UtYnl0ZXM=","percentage_complete":100}`))
	upstream.queueRead(coderws.MessageText, []byte(`{"type":"json","current_status":"completed","image_id":"image-uuid","order":0,"width":1024,"height":1536,"moderated":false}`))
	dialer := &grokImagineTestDialer{conn: upstream}
	svc := &OpenAIGatewayService{openaiWSPassthroughDialer: dialer}

	body := []byte(`{"prompt":"draw a cat","response_format":"b64_json"}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(string(body)))
	c.Request.Header.Set("Content-Type", "application/json")
	parsed, err := svc.ParseOpenAIImagesRequest(c, body)
	require.NoError(t, err)

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

	result, err := svc.ForwardImages(c.Request.Context(), c, account, body, parsed, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 1, result.ImageCount)
	require.Equal(t, grokImagineWSURL, dialer.wsURL)
	require.Equal(t, "aW1hZ2UtYnl0ZXM=", gjson.Get(rec.Body.String(), "data.0.b64_json").String())
}

func TestForwardGrokImagesEditMultipartUploadsImageAndMask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("DATA_DIR", t.TempDir())

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gpt-image-2"))
	require.NoError(t, writer.WriteField("prompt", "replace background with aurora"))

	imageHeader := make(textproto.MIMEHeader)
	imageHeader.Set("Content-Disposition", `form-data; name="image"; filename="source.png"`)
	imageHeader.Set("Content-Type", "image/png")
	imagePart, err := writer.CreatePart(imageHeader)
	require.NoError(t, err)
	_, err = imagePart.Write([]byte("png-image-content"))
	require.NoError(t, err)

	maskHeader := make(textproto.MIMEHeader)
	maskHeader.Set("Content-Disposition", `form-data; name="mask"; filename="mask.png"`)
	maskHeader.Set("Content-Type", "image/png")
	maskPart, err := writer.CreatePart(maskHeader)
	require.NoError(t, err)
	_, err = maskPart.Write([]byte("png-mask-content"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	imageCard := `{"id":"img-1","image_chunk":{"progress":100,"imageUuid":"image-uuid","imageUrl":"generated/edit.jpg","moderated":false}}`
	upstreamBody := strings.Join([]string{
		grokDataFrame(map[string]any{
			"cardAttachment": map[string]any{"jsonData": imageCard},
		}),
		`data: {"result":{"response":{"isSoftStop":true}}}`,
		``,
	}, "\n\n")
	upstream := &httpUpstreamRecorder{responses: []*http.Response{
		{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"fileMetadataId":"file-image"}`)),
		},
		{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"fileMetadataId":"file-mask"}`)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"X-Request-Id": []string{"grok-edit-1"}},
			Body:       io.NopCloser(strings.NewReader(upstreamBody)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"image/jpeg"}},
			Body:       io.NopCloser(strings.NewReader("edited-image-bytes")),
		},
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(body.Bytes()))
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	parsed, err := svc.ParseOpenAIImagesRequest(c, body.Bytes())
	require.NoError(t, err)

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

	result, err := svc.ForwardImages(c.Request.Context(), c, account, body.Bytes(), parsed, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 1, result.ImageCount)
	require.Len(t, upstream.requests, 4)
	require.Equal(t, "https://grok.example/rest/app-chat/upload-file", upstream.requests[0].URL.String())
	require.Equal(t, "https://grok.example/rest/app-chat/upload-file", upstream.requests[1].URL.String())
	require.Equal(t, "https://grok.example/rest/app-chat/conversations/new", upstream.requests[2].URL.String())
	require.Equal(t, "https://assets.grok.com/generated/edit.jpg", upstream.requests[3].URL.String())
	require.Equal(t, "source.png", gjson.GetBytes(upstream.bodies[0], "fileName").String())
	require.Equal(t, "cG5nLWltYWdlLWNvbnRlbnQ=", gjson.GetBytes(upstream.bodies[0], "content").String())
	require.Equal(t, "mask.png", gjson.GetBytes(upstream.bodies[1], "fileName").String())
	require.Equal(t, "cG5nLW1hc2stY29udGVudA==", gjson.GetBytes(upstream.bodies[1], "content").String())
	require.Equal(t, "file-image", gjson.GetBytes(upstream.bodies[2], "fileAttachments.0").String())
	require.Equal(t, "file-mask", gjson.GetBytes(upstream.bodies[2], "fileAttachments.1").String())
	require.Contains(t, gjson.GetBytes(upstream.bodies[2], "message").String(), "replace background with aurora")
	require.Contains(t, gjson.GetBytes(upstream.bodies[2], "message").String(), "[mask image attached]")
	require.Contains(t, gjson.Get(rec.Body.String(), "data.0.url").String(), "/v1/files/image?id=")
}

func grokDataFrame(response map[string]any) string {
	raw, err := json.Marshal(map[string]any{
		"result": map[string]any{
			"response": response,
		},
	})
	if err != nil {
		panic(err)
	}
	return "data: " + string(raw)
}

type grokImagineTestFrame struct {
	msgType coderws.MessageType
	payload []byte
}

type grokImagineTestFrameConn struct {
	readCh  chan grokImagineTestFrame
	writeCh chan any
	closed  chan struct{}
	once    sync.Once
}

func newGrokImagineTestFrameConn() *grokImagineTestFrameConn {
	return &grokImagineTestFrameConn{
		readCh:  make(chan grokImagineTestFrame, 8),
		writeCh: make(chan any, 8),
		closed:  make(chan struct{}),
	}
}

func (c *grokImagineTestFrameConn) queueRead(msgType coderws.MessageType, payload []byte) {
	c.readCh <- grokImagineTestFrame{msgType: msgType, payload: append([]byte(nil), payload...)}
}

func (c *grokImagineTestFrameConn) WriteJSON(_ context.Context, value any) error {
	select {
	case c.writeCh <- value:
		return nil
	case <-c.closed:
		return coderws.CloseError{Code: coderws.StatusNormalClosure}
	}
}

func (c *grokImagineTestFrameConn) ReadMessage(ctx context.Context) ([]byte, error) {
	select {
	case frame := <-c.readCh:
		return append([]byte(nil), frame.payload...), nil
	case <-c.closed:
		return nil, coderws.CloseError{Code: coderws.StatusNormalClosure}
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *grokImagineTestFrameConn) Ping(context.Context) error {
	return nil
}

func (c *grokImagineTestFrameConn) Close() error {
	c.once.Do(func() {
		close(c.closed)
	})
	return nil
}

type grokImagineTestDialer struct {
	conn    *grokImagineTestFrameConn
	wsURL   string
	headers http.Header
}

func (d *grokImagineTestDialer) Dial(ctx context.Context, wsURL string, headers http.Header, proxyURL string) (openAIWSClientConn, int, http.Header, error) {
	_ = ctx
	_ = proxyURL
	d.wsURL = wsURL
	d.headers = cloneHeader(headers)
	return d.conn, 0, nil, nil
}

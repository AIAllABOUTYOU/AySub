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
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	openaiwsv2 "github.com/Wei-Shaw/sub2api/internal/service/openai_ws_v2"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

const (
	openAIVideosEndpoint = "/v1/videos"

	grokWebMediaPostCreatePath = "/rest/media/post/create"
	grokWebLiveKitTokensPath   = "/rest/livekit/tokens"
	grokLiveKitWSBase          = "wss://livekit.grok.com"
	grokVideoMediaType         = "MEDIA_POST_TYPE_VIDEO"
	grokVideoModelName         = "imagine-video-gen"
	grokVideoQuality           = "standard"
	grokVideoMaxDownloadBytes  = 512 << 20
)

var grokVideoSizeMap = map[string]struct {
	AspectRatio string
	Resolution  string
}{
	"720x1280":  {AspectRatio: "9:16", Resolution: "720p"},
	"1280x720":  {AspectRatio: "16:9", Resolution: "720p"},
	"1024x1024": {AspectRatio: "1:1", Resolution: "720p"},
	"1024x1792": {AspectRatio: "9:16", Resolution: "720p"},
	"1792x1024": {AspectRatio: "16:9", Resolution: "720p"},
}

var grokVideoPresetFlags = map[string]string{
	"fun":    "--mode=extremely-crazy",
	"normal": "--mode=normal",
	"spicy":  "--mode=extremely-spicy-or-crazy",
	"custom": "--mode=custom",
}

type OpenAIVideoRequest struct {
	Model           string
	Prompt          string
	Image           string
	ImageURL        string
	Seconds         int
	Size            string
	Quality         string
	ResolutionName  string
	Preset          string
	VideoFormat     string
	UpscaleSourceID string
}

type GrokLiveKitTokenRequest struct {
	Voice             string
	Personality       string
	Speed             float64
	CustomInstruction string
}

type GrokLiveKitSession struct {
	Payload     map[string]any
	AccessToken string
	LiveKitURL  string
	Duration    time.Duration
	RequestID   string
}

type GrokLiveKitRTCResult struct {
	Duration                time.Duration
	ClientToUpstreamFrames  int64
	UpstreamToClientFrames  int64
	DroppedDownstreamFrames int64
}

type OpenAIVideoJob struct {
	ID          string         `json:"id"`
	Object      string         `json:"object"`
	CreatedAt   int64          `json:"created_at"`
	Status      string         `json:"status"`
	Model       string         `json:"model"`
	Progress    int            `json:"progress"`
	Prompt      string         `json:"prompt"`
	Seconds     string         `json:"seconds"`
	Size        string         `json:"size"`
	Quality     string         `json:"quality"`
	Resolution  string         `json:"resolution_name"`
	Preset      string         `json:"preset"`
	VideoFormat string         `json:"video_format"`
	CompletedAt *int64         `json:"completed_at,omitempty"`
	Error       map[string]any `json:"error,omitempty"`
	VideoURL    string         `json:"-"`
	LocalURL    string         `json:"-"`
	Thumbnail   string         `json:"-"`
	PostID      string         `json:"-"`
	AssetID     string         `json:"-"`
}

type grokVideoArtifact struct {
	VideoURL  string
	PostID    string
	AssetID   string
	Thumbnail string
}

var grokVideoJobs = struct {
	sync.RWMutex
	items map[string]*OpenAIVideoJob
}{items: make(map[string]*OpenAIVideoJob)}

func (j *OpenAIVideoJob) PublicPayload() map[string]any {
	if j == nil {
		return nil
	}
	payload := map[string]any{
		"id":              j.ID,
		"object":          j.Object,
		"created_at":      j.CreatedAt,
		"status":          j.Status,
		"model":           j.Model,
		"progress":        j.Progress,
		"prompt":          j.Prompt,
		"seconds":         j.Seconds,
		"size":            j.Size,
		"quality":         j.Quality,
		"resolution_name": j.Resolution,
		"preset":          j.Preset,
		"video_format":    j.VideoFormat,
	}
	if j.CompletedAt != nil {
		payload["completed_at"] = *j.CompletedAt
	}
	if j.Error != nil {
		payload["error"] = j.Error
	}
	return payload
}

func (s *OpenAIGatewayService) ParseOpenAIVideoRequest(body []byte) (*OpenAIVideoRequest, error) {
	return s.ParseOpenAIVideoRequestWithContentType("", body)
}

func (s *OpenAIGatewayService) ParseOpenAIVideoRequestWithContentType(contentType string, body []byte) (*OpenAIVideoRequest, error) {
	fields, err := parseOpenAIVideoRequestFields(contentType, body)
	if err != nil {
		return nil, err
	}
	return parseOpenAIVideoFields(fields)
}

func parseOpenAIVideoRequestFields(contentType string, body []byte) (map[string]string, error) {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err == nil && strings.HasPrefix(strings.ToLower(mediaType), "multipart/") {
		boundary := strings.TrimSpace(params["boundary"])
		if boundary == "" {
			return nil, errors.New("multipart boundary is required")
		}
		return parseOpenAIVideoMultipartFields(body, boundary)
	}
	if !gjson.ValidBytes(body) {
		return nil, errors.New("Failed to parse request body")
	}
	fields := map[string]string{}
	for _, name := range []string{
		"model", "prompt", "image", "image_url", "seconds", "size", "quality",
		"resolution_name", "preset", "video_format", "upscale_source_id",
	} {
		if value := strings.TrimSpace(gjson.GetBytes(body, name).String()); value != "" {
			fields[name] = value
		}
	}
	return fields, nil
}

func parseOpenAIVideoMultipartFields(body []byte, boundary string) (map[string]string, error) {
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	fields := map[string]string{}
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, errors.New("Failed to parse multipart request body")
		}
		name := strings.TrimSpace(part.FormName())
		if name == "" {
			_ = part.Close()
			continue
		}
		raw, readErr := io.ReadAll(io.LimitReader(part, 16<<20))
		_ = part.Close()
		if readErr != nil {
			return nil, errors.New("Failed to read multipart field")
		}
		if part.FileName() != "" {
			contentType := responseContentType(&http.Response{Header: http.Header{"Content-Type": []string{part.Header.Get("Content-Type")}}}, "application/octet-stream")
			fields[name] = "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(raw)
			continue
		}
		fields[name] = strings.TrimSpace(string(raw))
	}
	return fields, nil
}

func parseOpenAIVideoFields(fields map[string]string) (*OpenAIVideoRequest, error) {
	model := strings.TrimSpace(fields["model"])
	if model == "" {
		return nil, errors.New("model is required")
	}
	prompt := strings.TrimSpace(fields["prompt"])
	if prompt == "" {
		return nil, errors.New("prompt is required")
	}
	seconds := 0
	if raw := strings.TrimSpace(fields["seconds"]); raw != "" {
		parsedSeconds, err := strconv.Atoi(raw)
		if err != nil {
			return nil, errors.New("seconds must be one of [6, 10, 12, 16, 20]")
		}
		seconds = parsedSeconds
	}
	if seconds == 0 {
		seconds = 6
	}
	if !isSupportedGrokVideoSeconds(seconds) {
		return nil, errors.New("seconds must be one of [6, 10, 12, 16, 20]")
	}
	size := strings.TrimSpace(fields["size"])
	if size == "" {
		size = "720x1280"
	}
	if _, ok := grokVideoSizeMap[size]; !ok {
		return nil, errors.New("size must be one of [720x1280, 1280x720, 1024x1024, 1024x1792, 1792x1024]")
	}
	resolution := strings.ToLower(strings.TrimSpace(fields["resolution_name"]))
	if resolution == "" {
		resolution = grokVideoSizeMap[size].Resolution
	}
	if resolution != "480p" && resolution != "720p" {
		return nil, errors.New("resolution_name must be one of [480p, 720p]")
	}
	preset := strings.ToLower(strings.TrimSpace(fields["preset"]))
	if preset == "" {
		preset = "custom"
	}
	if _, ok := grokVideoPresetFlags[preset]; !ok {
		return nil, errors.New("preset must be one of [custom, fun, normal, spicy]")
	}
	videoFormat := strings.ToLower(strings.TrimSpace(fields["video_format"]))
	if videoFormat == "" {
		videoFormat = "mp4"
	}
	if videoFormat != "mp4" && videoFormat != "webm" && videoFormat != "gif" {
		return nil, errors.New("video_format must be one of [mp4, webm, gif]")
	}
	return &OpenAIVideoRequest{
		Model:           model,
		Prompt:          prompt,
		Image:           strings.TrimSpace(fields["image"]),
		ImageURL:        strings.TrimSpace(fields["image_url"]),
		Seconds:         seconds,
		Size:            size,
		Quality:         grokVideoQuality,
		ResolutionName:  resolution,
		Preset:          preset,
		VideoFormat:     videoFormat,
		UpscaleSourceID: strings.TrimSpace(fields["upscale_source_id"]),
	}, nil
}

func isSupportedGrokVideoSeconds(seconds int) bool {
	switch seconds {
	case 6, 10, 12, 16, 20:
		return true
	default:
		return false
	}
}

func (s *OpenAIGatewayService) ParseGrokLiveKitTokenRequest(body []byte) (*GrokLiveKitTokenRequest, error) {
	if len(body) > 0 && !gjson.ValidBytes(body) {
		return nil, errors.New("Failed to parse request body")
	}
	voice := strings.TrimSpace(gjson.GetBytes(body, "voice").String())
	if voice == "" {
		voice = "ara"
	}
	personality := strings.TrimSpace(gjson.GetBytes(body, "personality").String())
	if personality == "" {
		personality = "assistant"
	}
	speed := gjson.GetBytes(body, "speed").Float()
	if speed == 0 {
		speed = 1
	}
	if speed <= 0 || speed > 4 {
		return nil, errors.New("speed must be greater than 0 and less than or equal to 4")
	}
	return &GrokLiveKitTokenRequest{
		Voice:             voice,
		Personality:       personality,
		Speed:             speed,
		CustomInstruction: strings.TrimSpace(gjson.GetBytes(body, "instructions").String()),
	}, nil
}

func (s *OpenAIGatewayService) ForwardGrokLiveKitToken(ctx context.Context, c *gin.Context, account *Account, parsed *GrokLiveKitTokenRequest) (*OpenAIForwardResult, error) {
	session, err := s.FetchGrokLiveKitSession(ctx, account, parsed)
	if err != nil {
		return nil, err
	}
	if c != nil {
		c.JSON(http.StatusOK, session.Payload)
	}
	return &OpenAIForwardResult{
		RequestID:       session.RequestID,
		Usage:           OpenAIUsage{},
		Model:           "grok-livekit",
		BillingModel:    "grok-livekit",
		UpstreamModel:   "grok-livekit",
		ResponseHeaders: http.Header{"Content-Type": []string{"application/json"}},
		Duration:        session.Duration,
	}, nil
}

func (s *OpenAIGatewayService) FetchGrokLiveKitSession(ctx context.Context, account *Account, parsed *GrokLiveKitTokenRequest) (*GrokLiveKitSession, error) {
	if account == nil || !account.IsXAICookie() {
		return nil, errors.New("grok livekit token forwarding requires xai cookie account")
	}
	if parsed == nil {
		return nil, errors.New("parsed livekit token request is required")
	}
	startTime := time.Now()
	body, err := json.Marshal(buildGrokLiveKitTokenPayload(parsed))
	if err != nil {
		return nil, err
	}
	req, err := s.buildGrokWebJSONRequest(ctx, account, grokWebLiveKitTokensPath, body, grokWebDefaultBaseURL+"/")
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
		return nil, fmt.Errorf("grok livekit token request failed: %s", sanitizeUpstreamErrorMessage(err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		respBody := s.readUpstreamErrorBody(resp)
		msg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		return nil, fmt.Errorf("grok livekit token returned %d: %s", resp.StatusCode, msg)
	}
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, openAIUpstreamErrorBodyReadLimit))
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse grok livekit token response: %w", err)
	}
	accessToken := extractGrokLiveKitAccessToken(respBody)
	if accessToken == "" {
		return nil, errors.New("grok livekit token response did not include access token")
	}
	liveKitURL := buildGrokLiveKitWSURL(accessToken)
	payload["livekit_url"] = liveKitURL
	payload["ws_base"] = grokLiveKitWSBase
	return &GrokLiveKitSession{
		Payload:     payload,
		AccessToken: accessToken,
		LiveKitURL:  liveKitURL,
		Duration:    time.Since(startTime),
		RequestID:   resp.Header.Get("x-request-id"),
	}, nil
}

func buildGrokLiveKitTokenPayload(parsed *GrokLiveKitTokenRequest) map[string]any {
	session := map[string]any{
		"voice":          parsed.Voice,
		"personality":    parsed.Personality,
		"playback_speed": parsed.Speed,
		"enable_vision":  false,
		"turn_detection": map[string]any{"type": "server_vad"},
	}
	if parsed.CustomInstruction != "" {
		session["instructions"] = parsed.CustomInstruction
		session["is_raw_instructions"] = true
		session["personality"] = nil
	}
	sessionRaw, _ := json.Marshal(session)
	return map[string]any{
		"sessionPayload":       string(sessionRaw),
		"requestAgentDispatch": false,
		"livekitUrl":           grokLiveKitWSBase,
		"params":               map[string]any{"enable_markdown_transcript": "true"},
	}
}

func extractGrokLiveKitAccessToken(body []byte) string {
	for _, path := range []string{"access_token", "accessToken", "token", "livekit_token", "livekitToken"} {
		if token := strings.TrimSpace(gjson.GetBytes(body, path).String()); token != "" {
			return token
		}
	}
	return ""
}

func buildGrokLiveKitWSURL(accessToken string) string {
	values := url.Values{}
	values.Set("auto_subscribe", "1")
	values.Set("sdk", "js")
	values.Set("version", "2.11.4")
	values.Set("protocol", "15")
	values.Set("access_token", accessToken)
	return strings.TrimRight(grokLiveKitWSBase, "/") + "/rtc?" + values.Encode()
}

func (s *OpenAIGatewayService) ProxyGrokLiveKitRTC(ctx context.Context, clientConn *coderws.Conn, account *Account, accessToken string) (*GrokLiveKitRTCResult, error) {
	if account == nil || !account.IsXAICookie() {
		return nil, errors.New("grok livekit rtc proxy requires xai cookie account")
	}
	if clientConn == nil {
		return nil, errors.New("client websocket connection is required")
	}
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return nil, errors.New("access_token is required")
	}
	headers, err := buildGrokLiveKitWSHeaders(account)
	if err != nil {
		return nil, err
	}
	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	dialer := s.getOpenAIWSPassthroughDialer()
	if dialer == nil {
		return nil, errors.New("websocket dialer is not configured")
	}
	dialCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	upstreamConn, status, _, err := dialer.Dial(dialCtx, buildGrokLiveKitWSURL(accessToken), headers, proxyURL)
	cancel()
	if err != nil {
		if status > 0 {
			return nil, fmt.Errorf("grok livekit rtc websocket dial returned %d: %s", status, sanitizeUpstreamErrorMessage(err.Error()))
		}
		return nil, fmt.Errorf("grok livekit rtc websocket dial failed: %s", sanitizeUpstreamErrorMessage(err.Error()))
	}
	defer func() { _ = upstreamConn.Close() }()
	upstreamFrameConn, ok := upstreamConn.(openaiwsv2.FrameConn)
	if !ok {
		return nil, errors.New("websocket dialer returned connection without frame support")
	}

	start := time.Now()
	result, exit := openaiwsv2.RunEntry(openaiwsv2.EntryInput{
		Ctx:                ctx,
		ClientConn:         &coderOpenAIWSClientConn{conn: clientConn},
		UpstreamConn:       upstreamFrameConn,
		FirstClientMessage: []byte{},
		Options: openaiwsv2.RelayOptions{
			FirstMessageSent: true,
			IdleTimeout:      5 * time.Minute,
			WriteTimeout:     30 * time.Second,
		},
	})
	if exit != nil && exit.Err != nil && !isOpenAIWSClientDisconnectError(exit.Err) {
		return nil, fmt.Errorf("grok livekit rtc relay failed at %s: %w", exit.Stage, exit.Err)
	}
	duration := result.Duration
	if duration <= 0 {
		duration = time.Since(start)
	}
	return &GrokLiveKitRTCResult{
		Duration:                duration,
		ClientToUpstreamFrames:  result.ClientToUpstreamFrames,
		UpstreamToClientFrames:  result.UpstreamToClientFrames,
		DroppedDownstreamFrames: result.DroppedDownstreamFrames,
	}, nil
}

func buildGrokLiveKitWSHeaders(account *Account) (http.Header, error) {
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
	headers.Set("Referer", grokFirstNonEmpty(account.GetCredential("referer"), grokWebDefaultBaseURL+"/"))
	headers.Set("User-Agent", grokFirstNonEmpty(account.GetCredential("user_agent"), defaultGrokWebUserAgent()))
	applyGrokWebCompatibilityHeaders(headers, account, nil)
	return headers, nil
}

func (s *OpenAIGatewayService) ForwardVideos(ctx context.Context, c *gin.Context, account *Account, parsed *OpenAIVideoRequest, channelMappedModel string) (*OpenAIForwardResult, error) {
	if account == nil || !account.IsXAICookie() {
		return nil, errors.New("grok videos forwarding requires xai cookie account")
	}
	if parsed == nil {
		return nil, errors.New("parsed video request is required")
	}
	startTime := time.Now()
	requestModel := strings.TrimSpace(parsed.Model)
	if mapped := strings.TrimSpace(channelMappedModel); mapped != "" {
		requestModel = mapped
	}
	if requestModel == "" {
		requestModel = "grok-imagine-video"
	}

	job := &OpenAIVideoJob{
		ID:          "video_" + uuid.NewString(),
		Object:      "video",
		CreatedAt:   time.Now().Unix(),
		Status:      "in_progress",
		Model:       requestModel,
		Progress:    1,
		Prompt:      parsed.Prompt,
		Seconds:     fmt.Sprintf("%d", parsed.Seconds),
		Size:        parsed.Size,
		Quality:     parsed.Quality,
		Resolution:  parsed.ResolutionName,
		Preset:      parsed.Preset,
		VideoFormat: parsed.VideoFormat,
	}
	storeGrokVideoJob(job)

	artifact, err := s.generateGrokVideo(ctx, account, parsed, func(progress int) {
		updateGrokVideoJob(job.ID, func(j *OpenAIVideoJob) {
			j.Status = "in_progress"
			if progress > j.Progress {
				j.Progress = maxInt(1, minInt(99, progress))
			}
		})
	})
	if err != nil {
		updateGrokVideoJob(job.ID, func(j *OpenAIVideoJob) {
			j.Status = "failed"
			j.Error = map[string]any{"code": "video_generation_failed", "message": err.Error()}
		})
		c.JSON(http.StatusBadGateway, map[string]any{
			"error": map[string]any{
				"type":    "upstream_error",
				"message": err.Error(),
			},
		})
		return nil, err
	}
	completedAt := time.Now().Unix()
	localURL := ""
	if cachedURL, cacheErr := s.cacheGrokVideoAsLocalURL(ctx, account, artifact, job.ID); cacheErr == nil {
		localURL = cachedURL
	}
	updateGrokVideoJob(job.ID, func(j *OpenAIVideoJob) {
		j.Status = "completed"
		j.Progress = 100
		j.CompletedAt = &completedAt
		j.VideoURL = artifact.VideoURL
		j.LocalURL = localURL
		j.Thumbnail = artifact.Thumbnail
		j.PostID = artifact.PostID
		j.AssetID = artifact.AssetID
	})
	job = getGrokVideoJob(job.ID)
	c.JSON(http.StatusOK, job.PublicPayload())
	return &OpenAIForwardResult{
		RequestID:            job.ID,
		ResponseID:           job.ID,
		Usage:                OpenAIUsage{InputTokens: estimateGrokTextTokens(parsed.Prompt), OutputTokens: 1},
		Model:                requestModel,
		BillingModel:         requestModel,
		UpstreamModel:        grokVideoModelName,
		ResponseHeaders:      http.Header{"Content-Type": []string{"application/json"}},
		Duration:             time.Since(startTime),
		VideoCount:           1,
		VideoResolution:      NormalizeVideoBillingResolutionOrDefault(parsed.ResolutionName),
		VideoDurationSeconds: NormalizeVideoBillingDurationSecondsOrDefault(parsed.Seconds),
	}, nil
}

func storeGrokVideoJob(job *OpenAIVideoJob) {
	grokVideoJobs.Lock()
	defer grokVideoJobs.Unlock()
	grokVideoJobs.items[job.ID] = job
}

func updateGrokVideoJob(id string, fn func(*OpenAIVideoJob)) {
	grokVideoJobs.Lock()
	defer grokVideoJobs.Unlock()
	if job := grokVideoJobs.items[id]; job != nil && fn != nil {
		fn(job)
	}
}

func getGrokVideoJob(id string) *OpenAIVideoJob {
	grokVideoJobs.RLock()
	defer grokVideoJobs.RUnlock()
	if job := grokVideoJobs.items[id]; job != nil {
		copy := *job
		if job.Error != nil {
			copy.Error = make(map[string]any, len(job.Error))
			for k, v := range job.Error {
				copy.Error[k] = v
			}
		}
		return &copy
	}
	return nil
}

func (s *OpenAIGatewayService) GetVideoJob(id string) (*OpenAIVideoJob, bool) {
	job := getGrokVideoJob(strings.TrimSpace(id))
	return job, job != nil
}

func (s *OpenAIGatewayService) GetVideoContentURL(id string) (string, bool, error) {
	job := getGrokVideoJob(strings.TrimSpace(id))
	if job == nil {
		return "", false, nil
	}
	if job.Status != "completed" || strings.TrimSpace(job.VideoURL) == "" {
		return "", true, errors.New("Video content is not ready yet")
	}
	if localURL := strings.TrimSpace(job.LocalURL); localURL != "" {
		return localURL, true, nil
	}
	return job.VideoURL, true, nil
}

func (s *OpenAIGatewayService) cacheGrokVideoAsLocalURL(ctx context.Context, account *Account, artifact *grokVideoArtifact, fallbackID string) (string, error) {
	if artifact == nil || strings.TrimSpace(artifact.VideoURL) == "" {
		return "", errors.New("video URL is required")
	}
	raw, err := s.downloadGrokVideoBytes(ctx, account, artifact.VideoURL)
	if err != nil {
		return "", err
	}
	seed := grokFirstNonEmpty(artifact.AssetID, artifact.PostID, fallbackID, artifact.VideoURL)
	id, err := saveLocalVideo(raw, seed)
	if err != nil {
		return "", err
	}
	return localVideoURL(id), nil
}

func (s *OpenAIGatewayService) downloadGrokVideoBytes(ctx context.Context, account *Account, videoURL string) ([]byte, error) {
	if !isAllowedGrokAssetURL(videoURL) {
		return nil, fmt.Errorf("unsupported Grok video URL: %s", videoURL)
	}
	if s.httpUpstream == nil {
		return nil, errors.New("http upstream is not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, videoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "video/mp4,video/*,*/*;q=0.8")
	req.Header.Set("User-Agent", grokFirstNonEmpty(account.GetCredential("user_agent"), defaultGrokWebUserAgent()))
	req.Header.Set("Cookie", buildGrokWebCookieHeader(account))
	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.doGrokWebRequest(req, account, proxyURL)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.New("grok video download returned empty response")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("grok video download returned %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, grokVideoMaxDownloadBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > grokVideoMaxDownloadBytes {
		return nil, fmt.Errorf("grok video download exceeded %d bytes", grokVideoMaxDownloadBytes)
	}
	return raw, nil
}

func (s *OpenAIGatewayService) generateGrokVideo(ctx context.Context, account *Account, parsed *OpenAIVideoRequest, progress func(int)) (*grokVideoArtifact, error) {
	postID, err := s.createGrokVideoMediaPost(ctx, account, parsed.Prompt)
	if err != nil {
		return nil, err
	}
	payload := buildGrokVideoCreatePayload(parsed, postID)
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := s.buildGrokWebJSONRequest(ctx, account, grokWebChatPath, body, "https://grok.com/imagine")
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
		return nil, fmt.Errorf("grok video upstream request failed: %s", sanitizeUpstreamErrorMessage(err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		respBody := s.readUpstreamErrorBody(resp)
		upstreamMsg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
		if upstreamMsg == "" {
			upstreamMsg = http.StatusText(resp.StatusCode)
		}
		return nil, fmt.Errorf("grok video upstream returned %d: %s", resp.StatusCode, upstreamMsg)
	}
	artifact, err := readGrokVideoArtifact(resp.Body, progress)
	if err != nil {
		return nil, err
	}
	if artifact.PostID == "" {
		artifact.PostID = postID
	}
	return artifact, nil
}

func (s *OpenAIGatewayService) createGrokVideoMediaPost(ctx context.Context, account *Account, prompt string) (string, error) {
	payload := map[string]any{
		"mediaType": grokVideoMediaType,
		"prompt":    prompt,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := s.buildGrokWebJSONRequest(ctx, account, grokWebMediaPostCreatePath, body, "https://grok.com/imagine")
	if err != nil {
		return "", err
	}
	if s.httpUpstream == nil {
		return "", errors.New("http upstream is not configured")
	}
	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.doGrokWebRequest(req, account, proxyURL)
	if err != nil {
		return "", fmt.Errorf("grok video create-post request failed: %s", sanitizeUpstreamErrorMessage(err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		respBody := s.readUpstreamErrorBody(resp)
		msg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		return "", fmt.Errorf("grok video create-post returned %d: %s", resp.StatusCode, msg)
	}
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, openAIUpstreamErrorBodyReadLimit))
	if err != nil {
		return "", err
	}
	postID := strings.TrimSpace(gjson.GetBytes(respBody, "post.id").String())
	if postID == "" {
		postID = strings.TrimSpace(gjson.GetBytes(respBody, "id").String())
	}
	if postID == "" {
		return "", errors.New("grok video create-post returned no post id")
	}
	return postID, nil
}

func (s *OpenAIGatewayService) buildGrokWebJSONRequest(ctx context.Context, account *Account, path string, body []byte, referer string) (*http.Request, error) {
	targetURL, err := grokWebEndpointURL(account, path)
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
	req.Header.Set("Referer", grokFirstNonEmpty(account.GetCredential("referer"), referer))
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

func buildGrokVideoCreatePayload(parsed *OpenAIVideoRequest, parentPostID string) map[string]any {
	size := grokVideoSizeMap[parsed.Size]
	videoConfig := map[string]any{
		"parentPostId":   parentPostID,
		"aspectRatio":    size.AspectRatio,
		"videoLength":    parsed.Seconds,
		"resolutionName": parsed.ResolutionName,
	}
	if parsed.ImageURL != "" {
		videoConfig["imageUrl"] = parsed.ImageURL
	}
	if parsed.Image != "" {
		videoConfig["image"] = parsed.Image
	}
	if parsed.UpscaleSourceID != "" {
		videoConfig["upscaleSourceId"] = parsed.UpscaleSourceID
	}
	if parsed.VideoFormat != "" {
		videoConfig["videoFormat"] = parsed.VideoFormat
	}
	return map[string]any{
		"temporary":        true,
		"modelName":        grokVideoModelName,
		"message":          strings.TrimSpace(parsed.Prompt + " " + grokVideoPresetFlags[parsed.Preset]),
		"enableSideBySide": true,
		"responseMetadata": map[string]any{
			"experiments": []any{},
			"modelConfigOverride": map[string]any{
				"modelMap": map[string]any{
					"videoGenModelConfig": videoConfig,
				},
			},
		},
	}
}

func readGrokVideoArtifact(r io.Reader, progress func(int)) (*grokVideoArtifact, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	var artifact grokVideoArtifact
	var rawItems []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		data := line
		if strings.HasPrefix(data, "data:") {
			data = strings.TrimSpace(strings.TrimPrefix(data, "data:"))
		}
		if data == "[DONE]" {
			break
		}
		if !gjson.Valid(data) {
			continue
		}
		rawItems = append(rawItems, data)
		stream := gjson.Get(data, "result.response.streamingVideoGenerationResponse")
		if !stream.Exists() {
			continue
		}
		p := int(stream.Get("progress").Int())
		if progress != nil && p > 0 {
			progress(p)
		}
		if postID := strings.TrimSpace(grokFirstNonEmpty(stream.Get("videoPostId").String(), stream.Get("videoId").String())); postID != "" {
			artifact.PostID = postID
		}
		if assetID := strings.TrimSpace(stream.Get("assetId").String()); assetID != "" {
			artifact.AssetID = assetID
		}
		if thumbnail := strings.TrimSpace(stream.Get("thumbnailImageUrl").String()); thumbnail != "" {
			artifact.Thumbnail = absolutizeGrokAssetURL(thumbnail)
		}
		if p >= 100 && !stream.Get("moderated").Bool() {
			if videoURL := strings.TrimSpace(stream.Get("videoUrl").String()); videoURL != "" {
				artifact.VideoURL = absolutizeGrokAssetURL(videoURL)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if artifact.VideoURL == "" && artifact.AssetID != "" {
		artifact.VideoURL = absolutizeGrokAssetURL(artifact.AssetID)
	}
	if artifact.VideoURL == "" {
		return nil, fmt.Errorf("grok video generation returned no final video URL: %s", truncateString(strings.Join(rawItems, "\n"), 2048))
	}
	return &artifact, nil
}

func absolutizeGrokAssetURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err == nil && parsed.Scheme != "" && parsed.Host != "" {
		return value
	}
	path := value
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimRight(grokWebAssetsBaseURL, "/") + path
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

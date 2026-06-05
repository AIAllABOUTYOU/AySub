package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/util/httputil"
)

const (
	defaultGrokFlareSolverrTimeoutSeconds = 60
)

type grokFlareSolverrRequest struct {
	Cmd        string                    `json:"cmd"`
	URL        string                    `json:"url"`
	MaxTimeout int                       `json:"maxTimeout"`
	Proxy      *grokFlareSolverrProxyRef `json:"proxy,omitempty"`
}

type grokFlareSolverrProxyRef struct {
	URL string `json:"url"`
}

type grokFlareSolverrResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Solution struct {
		UserAgent string `json:"userAgent"`
		Cookies   []struct {
			Name   string `json:"name"`
			Value  string `json:"value"`
			Domain string `json:"domain"`
		} `json:"cookies"`
	} `json:"solution"`
}

type grokClearanceBundle struct {
	Cookies   string
	UserAgent string
}

func (s *OpenAIGatewayService) retryGrokWebAfterCloudflareChallenge(
	ctx context.Context,
	account *Account,
	resp *http.Response,
	respBody []byte,
	proxyURL string,
	buildRequest func() (*http.Request, error),
) (*http.Response, bool, error) {
	if account == nil || !account.IsXAICookie() || resp == nil {
		return nil, false, nil
	}
	if !httputil.IsCloudflareChallengeResponse(resp.StatusCode, resp.Header, respBody) {
		return nil, false, nil
	}
	if buildRequest == nil {
		return nil, false, errors.New("missing Grok retry request builder")
	}

	if err := s.refreshGrokCloudflareClearance(ctx, account, proxyURL); err != nil {
		return nil, false, err
	}

	req, err := buildRequest()
	if err != nil {
		return nil, false, err
	}
	if s.httpUpstream == nil {
		return nil, false, errors.New("http upstream is not configured")
	}
	retryResp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return nil, false, err
	}
	return retryResp, true, nil
}

func (s *OpenAIGatewayService) retryGrokWebResponseAfterCloudflareChallenge(
	ctx context.Context,
	account *Account,
	resp *http.Response,
	proxyURL string,
	operation string,
	buildRequest func() (*http.Request, error),
) *http.Response {
	if resp == nil || resp.StatusCode < 400 {
		return resp
	}
	respBody := s.readUpstreamErrorBody(resp)
	retryResp, retried, retryErr := s.retryGrokWebAfterCloudflareChallenge(ctx, account, resp, respBody, proxyURL, buildRequest)
	if retried {
		_ = resp.Body.Close()
		return retryResp
	}

	resp.Body = io.NopCloser(bytes.NewReader(respBody))
	if retryErr != nil {
		slog.Warn("grok cloudflare clearance refresh failed",
			"operation", operation,
			"account_id", account.ID,
			"error", retryErr,
		)
	}
	return resp
}

func (s *OpenAIGatewayService) refreshGrokCloudflareClearance(ctx context.Context, account *Account, proxyURL string) error {
	fsURL := strings.TrimSpace(account.GetCredential("flaresolverr_url"))
	if fsURL == "" && s != nil && s.cfg != nil {
		fsURL = strings.TrimSpace(s.cfg.Grok.FlareSolverrURL)
	}
	if fsURL == "" {
		return errors.New("Cloudflare challenge detected, but Grok FlareSolverr is not configured")
	}

	timeoutSeconds := grokFlareSolverrTimeoutSeconds(account, s.cfg)
	targetURL := grokFlareSolverrTargetURL(account)
	bundle, err := requestGrokFlareSolverrClearance(ctx, fsURL, targetURL, proxyURL, timeoutSeconds)
	if err != nil {
		return err
	}
	if bundle.Cookies == "" {
		return errors.New("FlareSolverr returned no Cloudflare cookies")
	}

	if account.Credentials == nil {
		account.Credentials = map[string]any{}
	}
	account.Credentials["cf_cookies"] = bundle.Cookies
	if cfClearance := extractCookieValue(bundle.Cookies, "cf_clearance"); cfClearance != "" {
		account.Credentials["cf_clearance"] = cfClearance
	}
	if strings.TrimSpace(bundle.UserAgent) != "" {
		account.Credentials["user_agent"] = bundle.UserAgent
	}

	slog.Info("grok cloudflare clearance refreshed",
		"account_id", account.ID,
		"target", targetURL,
		"via_proxy", proxyURL != "",
	)
	return nil
}

func grokFlareSolverrTimeoutSeconds(account *Account, cfg *config.Config) int {
	raw := strings.TrimSpace(account.GetCredential("flaresolverr_timeout_seconds"))
	if raw == "" {
		raw = strings.TrimSpace(account.GetCredential("cf_timeout"))
	}
	if raw != "" {
		if n, err := parsePositiveInt(raw); err == nil {
			return n
		}
	}
	if cfg != nil && cfg.Grok.FlareSolverrTimeoutSeconds > 0 {
		return cfg.Grok.FlareSolverrTimeoutSeconds
	}
	return defaultGrokFlareSolverrTimeoutSeconds
}

func parsePositiveInt(raw string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, err
	}
	if n <= 0 {
		return 0, fmt.Errorf("value must be positive")
	}
	return n, nil
}

func grokFlareSolverrTargetURL(account *Account) string {
	base := strings.TrimSpace(account.GetCredential("base_url"))
	if base == "" {
		return grokWebDefaultBaseURL
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return grokWebDefaultBaseURL
	}
	parsed.Path = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/")
}

func requestGrokFlareSolverrClearance(ctx context.Context, fsURL, targetURL, proxyURL string, timeoutSeconds int) (grokClearanceBundle, error) {
	fsURL = strings.TrimSpace(fsURL)
	if fsURL == "" {
		return grokClearanceBundle{}, errors.New("FlareSolverr URL is required")
	}
	if timeoutSeconds <= 0 {
		timeoutSeconds = defaultGrokFlareSolverrTimeoutSeconds
	}

	payload := grokFlareSolverrRequest{
		Cmd:        "request.get",
		URL:        targetURL,
		MaxTimeout: timeoutSeconds * 1000,
	}
	if strings.TrimSpace(proxyURL) != "" {
		payload.Proxy = &grokFlareSolverrProxyRef{URL: strings.TrimSpace(proxyURL)}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return grokClearanceBundle{}, err
	}

	endpoint := strings.TrimRight(fsURL, "/") + "/v1"
	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds+30)*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return grokClearanceBundle{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: time.Duration(timeoutSeconds+30) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return grokClearanceBundle{}, fmt.Errorf("FlareSolverr request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return grokClearanceBundle{}, fmt.Errorf("read FlareSolverr response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return grokClearanceBundle{}, fmt.Errorf("FlareSolverr returned HTTP %d: %s", resp.StatusCode, httputil.TruncateBody(respBody, 300))
	}

	var decoded grokFlareSolverrResponse
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return grokClearanceBundle{}, fmt.Errorf("parse FlareSolverr response: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(decoded.Status), "ok") {
		msg := strings.TrimSpace(decoded.Message)
		if msg == "" {
			msg = "unknown error"
		}
		return grokClearanceBundle{}, fmt.Errorf("FlareSolverr status %q: %s", decoded.Status, msg)
	}

	cookies := make([]string, 0, len(decoded.Solution.Cookies))
	targetHost := ""
	if parsed, err := url.Parse(targetURL); err == nil {
		targetHost = strings.ToLower(parsed.Hostname())
	}
	for _, cookie := range decoded.Solution.Cookies {
		name := strings.TrimSpace(cookie.Name)
		value := strings.TrimSpace(cookie.Value)
		if name == "" || value == "" {
			continue
		}
		domain := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(cookie.Domain)), ".")
		if targetHost != "" && domain != "" && !strings.HasSuffix(targetHost, domain) {
			continue
		}
		cookies = append(cookies, name+"="+value)
	}
	if len(cookies) == 0 {
		return grokClearanceBundle{}, errors.New("FlareSolverr returned no cookies for target host")
	}

	return grokClearanceBundle{
		Cookies:   strings.Join(cookies, "; "),
		UserAgent: strings.TrimSpace(decoded.Solution.UserAgent),
	}, nil
}

func normalizeGrokCookieModelAlias(model string) string {
	trimmed := strings.TrimSpace(model)
	switch strings.ToLower(trimmed) {
	case "grok-4.2-fast":
		return "grok-4.20-fast"
	case "grok-4.2-auto":
		return "grok-4.20-auto"
	case "grok-4.2-expert", "grok-4.2-reasoning":
		return "grok-4.20-expert"
	case "grok-4.2-heavy":
		return "grok-4.20-heavy"
	default:
		return trimmed
	}
}

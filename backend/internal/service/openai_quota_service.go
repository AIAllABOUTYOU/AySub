package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var ErrSparkShadowResetNotSupported = infraerrors.New(http.StatusConflict, "SPARK_SHADOW_RESET_NOT_SUPPORTED", "spark shadow account does not support credit reset; reset the parent account")

var (
	chatGPTUsageURL                 = "https://chatgpt.com/backend-api/wham/usage"
	chatGPTRateLimitResetCreditsURL = "https://chatgpt.com/backend-api/wham/rate-limit-reset-credits"
	chatGPTRateLimitResetURL        = "https://chatgpt.com/backend-api/wham/rate-limit-reset-credits/consume"
)

const (
	openaiQuotaUpstreamTimeout = 20 * time.Second
	openaiQuotaCodexBeta       = "codex-1"
	openaiQuotaCodexOriginator = "Codex Desktop"
	openaiQuotaCodexLanguage   = "zh-CN"
)

type OpenAIRateLimitWindow struct {
	UsedPercent        float64 `json:"used_percent"`
	LimitWindowSeconds int64   `json:"limit_window_seconds"`
	ResetAfterSeconds  int64   `json:"reset_after_seconds"`
	ResetAt            int64   `json:"reset_at"`
}

type OpenAIRateLimit struct {
	Allowed         bool                   `json:"allowed"`
	LimitReached    bool                   `json:"limit_reached"`
	PrimaryWindow   *OpenAIRateLimitWindow `json:"primary_window,omitempty"`
	SecondaryWindow *OpenAIRateLimitWindow `json:"secondary_window,omitempty"`
}

type OpenAIAdditionalRateLimit struct {
	LimitName      string           `json:"limit_name"`
	MeteredFeature string           `json:"metered_feature"`
	RateLimit      *OpenAIRateLimit `json:"rate_limit,omitempty"`
}

type OpenAIRateLimitResetCredits struct {
	AvailableCount int                                `json:"available_count"`
	Credits        []OpenAIRateLimitResetCreditDetail `json:"credits,omitempty"`
}

type OpenAIRateLimitResetCreditDetail struct {
	ExpiresAt string `json:"expires_at,omitempty"`
}

type OpenAIQuotaUsage struct {
	UserID                string                       `json:"user_id,omitempty"`
	AccountID             string                       `json:"account_id,omitempty"`
	Email                 string                       `json:"email,omitempty"`
	PlanType              string                       `json:"plan_type,omitempty"`
	RateLimit             *OpenAIRateLimit             `json:"rate_limit,omitempty"`
	AdditionalRateLimits  []OpenAIAdditionalRateLimit  `json:"additional_rate_limits,omitempty"`
	RateLimitResetCredits *OpenAIRateLimitResetCredits `json:"rate_limit_reset_credits,omitempty"`
	FetchedAt             int64                        `json:"fetched_at"`
}

type OpenAIQuotaResetCredit struct {
	ID              string `json:"id,omitempty"`
	ResetType       string `json:"reset_type,omitempty"`
	Status          string `json:"status,omitempty"`
	GrantedAt       string `json:"granted_at,omitempty"`
	ExpiresAt       string `json:"expires_at,omitempty"`
	RedeemStartedAt string `json:"redeem_started_at,omitempty"`
	RedeemedAt      string `json:"redeemed_at,omitempty"`
	ProfileImageURL string `json:"profile_image_url,omitempty"`
	ProfileUserID   string `json:"profile_user_id,omitempty"`
	Title           string `json:"title,omitempty"`
	Description     string `json:"description,omitempty"`
}

type OpenAIQuotaResetResult struct {
	Code         string                  `json:"code"`
	Credit       *OpenAIQuotaResetCredit `json:"credit,omitempty"`
	WindowsReset int                     `json:"windows_reset"`
}

type OpenAIQuotaResetCreditsDetail struct {
	Credits          []OpenAIQuotaResetCredit `json:"credits,omitempty"`
	AvailableCount   int                      `json:"available_count"`
	TotalEarnedCount int                      `json:"total_earned_count,omitempty"`
	FetchedAt        int64                    `json:"fetched_at"`
}

// OpenAIQuotaService queries and consumes ChatGPT/Codex rate-limit reset credits.
type OpenAIQuotaService struct {
	accountRepo          AccountRepository
	proxyRepo            ProxyRepository
	tokenProvider        *OpenAITokenProvider
	privacyClientFactory PrivacyClientFactory
}

func NewOpenAIQuotaService(
	accountRepo AccountRepository,
	proxyRepo ProxyRepository,
	tokenProvider *OpenAITokenProvider,
	privacyClientFactory PrivacyClientFactory,
) *OpenAIQuotaService {
	return &OpenAIQuotaService{
		accountRepo:          accountRepo,
		proxyRepo:            proxyRepo,
		tokenProvider:        tokenProvider,
		privacyClientFactory: privacyClientFactory,
	}
}

func (s *OpenAIQuotaService) QueryUsage(ctx context.Context, accountID int64) (*OpenAIQuotaUsage, error) {
	accessToken, chatGPTAccountID, proxyURL, err := s.prepareUpstreamCall(ctx, accountID)
	if err != nil {
		return nil, err
	}

	client, err := s.privacyClientFactory(proxyURL)
	if err != nil {
		return nil, infraerrors.Newf(http.StatusBadGateway, "OPENAI_QUOTA_CLIENT_ERROR", "failed to build upstream client: %v", err)
	}

	callCtx, cancel := context.WithTimeout(ctx, openaiQuotaUpstreamTimeout)
	defer cancel()

	var payload OpenAIQuotaUsage
	resp, err := client.R().
		SetContext(callCtx).
		SetHeaders(buildCodexCommonHeaders(accessToken, chatGPTAccountID)).
		SetSuccessResult(&payload).
		Get(chatGPTUsageURL)
	if err != nil {
		return nil, infraerrors.Newf(http.StatusBadGateway, "OPENAI_QUOTA_REQUEST_FAILED", "upstream request failed: %v", err)
	}
	if !resp.IsSuccessState() {
		status := resp.StatusCode
		body := truncate(resp.String(), 240)
		slog.Warn("openai_quota_query_failed", "account_id", accountID, "status", status, "body", body)
		return nil, infraerrors.Newf(mapOpenAIQuotaUpstreamStatus(status), "OPENAI_QUOTA_UPSTREAM_ERROR", "upstream returned %d: %s", status, body)
	}

	payload.FetchedAt = time.Now().Unix()
	if payload.RateLimitResetCredits != nil && payload.RateLimitResetCredits.AvailableCount > 0 {
		credits, detailErr := s.queryResetCreditsWithPreparedAccount(ctx, accessToken, chatGPTAccountID, proxyURL, accountID)
		if detailErr != nil {
			slog.Warn("openai_quota_reset_credit_details_failed", "account_id", accountID, "error", detailErr)
		} else {
			payload.RateLimitResetCredits.Credits = sanitizeResetCreditExpirations(credits.Credits)
		}
	}
	return &payload, nil
}

func sanitizeResetCreditExpirations(credits []OpenAIQuotaResetCredit) []OpenAIRateLimitResetCreditDetail {
	result := make([]OpenAIRateLimitResetCreditDetail, 0, len(credits))
	for _, credit := range credits {
		if expiresAt := strings.TrimSpace(credit.ExpiresAt); expiresAt != "" {
			result = append(result, OpenAIRateLimitResetCreditDetail{ExpiresAt: expiresAt})
		}
	}
	return result
}

func (s *OpenAIQuotaService) QueryResetCredits(ctx context.Context, accountID int64) (*OpenAIQuotaResetCreditsDetail, error) {
	accessToken, chatGPTAccountID, proxyURL, err := s.prepareUpstreamCall(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return s.queryResetCreditsWithPreparedAccount(ctx, accessToken, chatGPTAccountID, proxyURL, accountID)
}

func (s *OpenAIQuotaService) ResetCredit(ctx context.Context, accountID int64) (*OpenAIQuotaResetResult, error) {
	if s.accountRepo != nil {
		account, loadErr := s.accountRepo.GetByID(ctx, accountID)
		if loadErr != nil {
			return nil, infraerrors.Newf(http.StatusNotFound, "OPENAI_QUOTA_ACCOUNT_NOT_FOUND", "account not found: %v", loadErr)
		}
		if account != nil && account.IsShadow() {
			return nil, ErrSparkShadowResetNotSupported
		}
	}
	accessToken, chatGPTAccountID, proxyURL, err := s.prepareUpstreamCall(ctx, accountID)
	if err != nil {
		return nil, err
	}

	credits, err := s.queryResetCreditsWithPreparedAccount(ctx, accessToken, chatGPTAccountID, proxyURL, accountID)
	if err != nil {
		return nil, err
	}
	if credits.AvailableCount <= 0 {
		return nil, infraerrors.New(http.StatusConflict, "OPENAI_QUOTA_NO_RESET_CREDITS", "no available OpenAI/Codex reset credits")
	}

	redeemRequestID, err := generateRedeemRequestID()
	if err != nil {
		return nil, infraerrors.Newf(http.StatusInternalServerError, "OPENAI_QUOTA_REDEEM_ID_FAILED", "failed to generate redeem id: %v", err)
	}

	client, err := s.privacyClientFactory(proxyURL)
	if err != nil {
		return nil, infraerrors.Newf(http.StatusBadGateway, "OPENAI_QUOTA_CLIENT_ERROR", "failed to build upstream client: %v", err)
	}

	callCtx, cancel := context.WithTimeout(ctx, openaiQuotaUpstreamTimeout)
	defer cancel()

	headers := buildCodexCommonHeaders(accessToken, chatGPTAccountID)
	headers["content-type"] = "application/json"

	var payload OpenAIQuotaResetResult
	resp, err := client.R().
		SetContext(callCtx).
		SetHeaders(headers).
		SetBody(map[string]string{"redeem_request_id": redeemRequestID}).
		SetSuccessResult(&payload).
		Post(chatGPTRateLimitResetURL)
	if err != nil {
		return nil, infraerrors.Newf(http.StatusBadGateway, "OPENAI_QUOTA_RESET_REQUEST_FAILED", "upstream request failed: %v", err)
	}
	if !resp.IsSuccessState() {
		status := resp.StatusCode
		body := truncate(resp.String(), 240)
		slog.Warn("openai_quota_reset_failed", "account_id", accountID, "status", status, "body", body)
		return nil, infraerrors.Newf(mapOpenAIQuotaUpstreamStatus(status), "OPENAI_QUOTA_RESET_UPSTREAM_ERROR", "upstream returned %d: %s", status, body)
	}

	slog.Info("openai_quota_reset_success", "account_id", accountID, "code", payload.Code, "windows_reset", payload.WindowsReset)
	return &payload, nil
}

func (s *OpenAIQuotaService) queryResetCreditsWithPreparedAccount(ctx context.Context, accessToken, chatGPTAccountID, proxyURL string, accountID int64) (*OpenAIQuotaResetCreditsDetail, error) {
	client, err := s.privacyClientFactory(proxyURL)
	if err != nil {
		return nil, infraerrors.Newf(http.StatusBadGateway, "OPENAI_QUOTA_CLIENT_ERROR", "failed to build upstream client: %v", err)
	}

	callCtx, cancel := context.WithTimeout(ctx, openaiQuotaUpstreamTimeout)
	defer cancel()

	var payload OpenAIQuotaResetCreditsDetail
	resp, err := client.R().
		SetContext(callCtx).
		SetHeaders(buildCodexCommonHeaders(accessToken, chatGPTAccountID)).
		SetSuccessResult(&payload).
		Get(chatGPTRateLimitResetCreditsURL)
	if err != nil {
		return nil, infraerrors.Newf(http.StatusBadGateway, "OPENAI_QUOTA_RESET_CREDITS_REQUEST_FAILED", "upstream request failed: %v", err)
	}
	if !resp.IsSuccessState() {
		status := resp.StatusCode
		body := truncate(resp.String(), 240)
		slog.Warn("openai_quota_reset_credits_failed", "account_id", accountID, "status", status, "body", body)
		return nil, infraerrors.Newf(mapOpenAIQuotaUpstreamStatus(status), "OPENAI_QUOTA_RESET_CREDITS_UPSTREAM_ERROR", "upstream returned %d: %s", status, body)
	}

	payload.FetchedAt = time.Now().Unix()
	return &payload, nil
}

func (s *OpenAIQuotaService) prepareUpstreamCall(ctx context.Context, accountID int64) (accessToken, chatGPTAccountID, proxyURL string, err error) {
	if s == nil || s.accountRepo == nil || s.tokenProvider == nil || s.privacyClientFactory == nil {
		return "", "", "", infraerrors.New(http.StatusInternalServerError, "OPENAI_QUOTA_NOT_CONFIGURED", "openai quota service is not configured")
	}

	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return "", "", "", infraerrors.Newf(http.StatusNotFound, "OPENAI_QUOTA_ACCOUNT_NOT_FOUND", "account not found: %v", err)
	}
	if account == nil {
		return "", "", "", infraerrors.New(http.StatusNotFound, "OPENAI_QUOTA_ACCOUNT_NOT_FOUND", "account not found")
	}
	if account.Platform != PlatformOpenAI {
		return "", "", "", infraerrors.New(http.StatusBadRequest, "OPENAI_QUOTA_INVALID_PLATFORM", "account is not an OpenAI account")
	}
	if account.Type != AccountTypeOAuth {
		return "", "", "", infraerrors.New(http.StatusBadRequest, "OPENAI_QUOTA_INVALID_TYPE", "account is not an OAuth account")
	}
	if account.IsShadow() {
		resolved, resolveErr := resolveCredentialAccount(ctx, s.accountRepo, account)
		if resolveErr != nil {
			return "", "", "", infraerrors.Newf(http.StatusBadGateway, "OPENAI_QUOTA_SHADOW_RESOLVE_FAILED", "failed to resolve shadow account: %v", resolveErr)
		}
		account = resolved
	}

	chatGPTAccountID = strings.TrimSpace(account.GetCredential("chatgpt_account_id"))
	if chatGPTAccountID == "" {
		chatGPTAccountID = strings.TrimSpace(account.GetCredential("organization_id"))
	}
	if chatGPTAccountID == "" {
		return "", "", "", infraerrors.New(http.StatusBadRequest, "OPENAI_QUOTA_MISSING_ACCOUNT_ID", "chatgpt_account_id is missing; please re-authorize this account")
	}

	accessToken, err = s.tokenProvider.GetAccessToken(ctx, account)
	if err != nil {
		return "", "", "", infraerrors.Newf(http.StatusBadGateway, "OPENAI_QUOTA_TOKEN_UNAVAILABLE", "failed to acquire access token: %v", err)
	}
	if strings.TrimSpace(accessToken) == "" {
		return "", "", "", infraerrors.New(http.StatusBadGateway, "OPENAI_QUOTA_TOKEN_UNAVAILABLE", "access token is empty")
	}

	if account.ProxyID != nil {
		switch {
		case account.Proxy != nil:
			proxyURL = account.Proxy.URL()
		case s.proxyRepo != nil:
			if proxy, perr := s.proxyRepo.GetByID(ctx, *account.ProxyID); perr == nil && proxy != nil {
				proxyURL = proxy.URL()
			}
		}
	}

	return accessToken, chatGPTAccountID, proxyURL, nil
}

func buildCodexCommonHeaders(accessToken, chatGPTAccountID string) map[string]string {
	return map[string]string{
		"authorization":      "Bearer " + accessToken,
		"chatgpt-account-id": chatGPTAccountID,
		"openai-beta":        openaiQuotaCodexBeta,
		"oai-language":       openaiQuotaCodexLanguage,
		"originator":         openaiQuotaCodexOriginator,
		"accept":             "application/json",
		"sec-fetch-site":     "none",
		"sec-fetch-mode":     "no-cors",
		"sec-fetch-dest":     "empty",
		"priority":           "u=4, i",
	}
}

func buildCodexSparkWindowExtraUpdates(usage *OpenAIQuotaUsage, now time.Time) map[string]any {
	if usage == nil {
		return nil
	}
	var spark *OpenAIRateLimit
	for i := range usage.AdditionalRateLimits {
		if usage.AdditionalRateLimits[i].MeteredFeature == "codex_bengalfox" {
			spark = usage.AdditionalRateLimits[i].RateLimit
			break
		}
	}
	if spark == nil {
		return nil
	}
	snapshot := &OpenAICodexUsageSnapshot{}
	if window := spark.PrimaryWindow; window != nil {
		used, reset, minutes := window.UsedPercent, int(window.ResetAfterSeconds), int(window.LimitWindowSeconds/60)
		snapshot.PrimaryUsedPercent, snapshot.PrimaryResetAfterSeconds, snapshot.PrimaryWindowMinutes = &used, &reset, &minutes
	}
	if window := spark.SecondaryWindow; window != nil {
		used, reset, minutes := window.UsedPercent, int(window.ResetAfterSeconds), int(window.LimitWindowSeconds/60)
		snapshot.SecondaryUsedPercent, snapshot.SecondaryResetAfterSeconds, snapshot.SecondaryWindowMinutes = &used, &reset, &minutes
	}
	normalized := snapshot.Normalize()
	if normalized == nil {
		return nil
	}
	updates := make(map[string]any)
	if normalized.Used5hPercent != nil {
		updates["codex_5h_used_percent"] = *normalized.Used5hPercent
	}
	if normalized.Reset5hSeconds != nil {
		updates["codex_5h_reset_after_seconds"] = *normalized.Reset5hSeconds
	}
	if normalized.Window5hMinutes != nil {
		updates["codex_5h_window_minutes"] = *normalized.Window5hMinutes
	}
	if normalized.Used7dPercent != nil {
		updates["codex_7d_used_percent"] = *normalized.Used7dPercent
	}
	if normalized.Reset7dSeconds != nil {
		updates["codex_7d_reset_after_seconds"] = *normalized.Reset7dSeconds
	}
	if normalized.Window7dMinutes != nil {
		updates["codex_7d_window_minutes"] = *normalized.Window7dMinutes
	}
	if resetAt := codexResetAtRFC3339(now, normalized.Reset5hSeconds); resetAt != nil {
		updates["codex_5h_reset_at"] = *resetAt
	}
	if resetAt := codexResetAtRFC3339(now, normalized.Reset7dSeconds); resetAt != nil {
		updates["codex_7d_reset_at"] = *resetAt
	}
	if len(updates) == 0 {
		return nil
	}
	updates["codex_usage_updated_at"] = now.Format(time.RFC3339)
	return updates
}

func generateRedeemRequestID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	hexStr := hex.EncodeToString(b)
	return fmt.Sprintf("%s-%s-%s-%s-%s", hexStr[0:8], hexStr[8:12], hexStr[12:16], hexStr[16:20], hexStr[20:]), nil
}

func mapOpenAIQuotaUpstreamStatus(status int) int {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return status
	case status == http.StatusTooManyRequests:
		return http.StatusTooManyRequests
	case status >= 400:
		return http.StatusBadGateway
	default:
		return http.StatusBadGateway
	}
}

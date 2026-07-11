package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildCodexSparkWindowExtraUpdates(t *testing.T) {
	now := time.Date(2026, 7, 11, 8, 0, 0, 0, time.UTC)
	usage := &OpenAIQuotaUsage{AdditionalRateLimits: []OpenAIAdditionalRateLimit{
		{MeteredFeature: "other", RateLimit: &OpenAIRateLimit{}},
		{MeteredFeature: "codex_bengalfox", RateLimit: &OpenAIRateLimit{
			PrimaryWindow:   &OpenAIRateLimitWindow{UsedPercent: 25, ResetAfterSeconds: 1800, LimitWindowSeconds: 5 * 60 * 60},
			SecondaryWindow: &OpenAIRateLimitWindow{UsedPercent: 60, ResetAfterSeconds: 86400, LimitWindowSeconds: 7 * 24 * 60 * 60},
		}},
	}}
	updates := buildCodexSparkWindowExtraUpdates(usage, now)
	require.Equal(t, float64(25), updates["codex_5h_used_percent"])
	require.Equal(t, float64(60), updates["codex_7d_used_percent"])
	require.Equal(t, 300, updates["codex_5h_window_minutes"])
	require.Equal(t, 10080, updates["codex_7d_window_minutes"])
	require.Equal(t, now.Format(time.RFC3339), updates["codex_usage_updated_at"])
}

func TestBuildCodexSparkWindowExtraUpdatesIgnoresOtherFeatures(t *testing.T) {
	usage := &OpenAIQuotaUsage{AdditionalRateLimits: []OpenAIAdditionalRateLimit{{MeteredFeature: "other", RateLimit: &OpenAIRateLimit{}}}}
	require.Nil(t, buildCodexSparkWindowExtraUpdates(usage, time.Now()))
}

func TestSanitizeResetCreditExpirationsOnlyExposesExpiry(t *testing.T) {
	credits := []OpenAIQuotaResetCredit{
		{ID: "secret", ExpiresAt: " 2026-07-12T04:05:06Z ", Title: "private"},
		{ID: "ignored-without-expiry"},
	}

	require.Equal(t, []OpenAIRateLimitResetCreditDetail{
		{ExpiresAt: "2026-07-12T04:05:06Z"},
	}, sanitizeResetCreditExpirations(credits))
}

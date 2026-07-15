package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseOpenAIRateLimitResetCreditDetails_PreservesAvailableCreditOrder(t *testing.T) {
	body := []byte(`{
		"availableCount":"2",
		"credits":[
			{"reset_type":"codex_rate_limits","status":"redeemed","expires_at":"2026-07-01T04:05:06Z"},
			{"reset_type":"codex_rate_limits","status":"available","expires_at":"2026-07-04T04:05:06Z"},
			{"resetType":"codex_rate_limits","status":"available","expiresAt":"2026-07-03T04:05:06Z"},
			{"reset_type":"other","status":"available","expires_at":"2026-07-02T04:05:06Z"}
		]
	}`)

	details, err := parseOpenAIRateLimitResetCreditDetails(body)
	require.NoError(t, err)
	require.NotNil(t, details.AvailableCount)
	require.Equal(t, 2, *details.AvailableCount)
	require.Equal(t, []OpenAIRateLimitResetCreditDetail{
		{ExpiresAt: "2026-07-04T04:05:06Z"},
		{ExpiresAt: "2026-07-03T04:05:06Z"},
	}, details.Credits)
}

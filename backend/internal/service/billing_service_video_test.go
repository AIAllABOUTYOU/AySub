//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalculateVideoCostUsesSecondsCountAndIndependentPrice(t *testing.T) {
	svc := &BillingService{}
	price := 0.10
	cost := svc.CalculateVideoCost(
		"grok-imagine-video",
		"720p",
		2,
		20,
		&VideoPriceConfig{Price720P: &price},
		0.5,
	)

	require.InDelta(t, 4.0, cost.TotalCost, 1e-12)
	require.InDelta(t, 2.0, cost.ActualCost, 1e-12)
	require.Equal(t, string(BillingModeVideo), cost.BillingMode)
}

func TestCalculateVideoCostUsesAySubDefaultDuration(t *testing.T) {
	svc := &BillingService{}
	cost := svc.CalculateVideoCost("grok-imagine-video", "480p", 1, 0, nil, 1)

	require.InDelta(t, 0.30, cost.TotalCost, 1e-12)
	require.Equal(t, VideoBillingDefaultDurationSeconds, NormalizeVideoBillingDurationSecondsOrDefault(8))
	require.Equal(t, 20, NormalizeVideoBillingDurationSecondsOrDefault(20))
}

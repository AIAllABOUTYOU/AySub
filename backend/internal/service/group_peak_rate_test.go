package service

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/stretchr/testify/require"
)

func TestPeakMultiplierAtBoundaries(t *testing.T) {
	require.NoError(t, timezone.Init("UTC"))
	g := &Group{SubscriptionType: SubscriptionTypeSubscription, PeakRateEnabled: true, PeakStart: "14:00", PeakEnd: "18:00", PeakRateMultiplier: 3}
	at := func(hour, minute int) time.Time { return time.Date(2026, 6, 29, hour, minute, 0, 0, time.UTC) }
	require.Equal(t, 1.0, g.PeakMultiplierAt(at(13, 59)))
	require.Equal(t, 3.0, g.PeakMultiplierAt(at(14, 0)))
	require.Equal(t, 3.0, g.PeakMultiplierAt(at(17, 59)))
	require.Equal(t, 1.0, g.PeakMultiplierAt(at(18, 0)))
}

func TestValidatePeakRateConfig(t *testing.T) {
	require.NoError(t, ValidatePeakRateConfig(SubscriptionTypeSubscription, true, "14:00", "18:00", 0))
	require.Error(t, ValidatePeakRateConfig(SubscriptionTypeStandard, true, "14:00", "18:00", 2))
	require.Error(t, ValidatePeakRateConfig(SubscriptionTypeSubscription, true, "22:00", "02:00", 2))
	require.Error(t, ValidatePeakRateConfig(SubscriptionTypeSubscription, true, "14:00", "18:00", -1))
}

func TestPeakMultiplierDoesNotAffectImageMultiplier(t *testing.T) {
	require.NoError(t, timezone.Init("UTC"))
	g := &Group{SubscriptionType: SubscriptionTypeSubscription, PeakRateEnabled: true, PeakStart: "14:00", PeakEnd: "18:00", PeakRateMultiplier: 3, ImageRateIndependent: true, ImageRateMultiplier: 0.5}
	text, image := computePeakAwareMultipliers(&APIKey{Group: g}, 0.8, time.Date(2026, 6, 29, 15, 0, 0, 0, time.UTC))
	require.InDelta(t, 2.4, text, 1e-12)
	require.InDelta(t, 0.5, image, 1e-12)
}

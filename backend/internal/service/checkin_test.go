package service

import "testing"

func TestCheckinMonthRange(t *testing.T) {
	start, end, err := checkinMonthRange("2026-02")
	if err != nil {
		t.Fatalf("expected valid month: %v", err)
	}
	if start != "2026-02-01" {
		t.Fatalf("start = %q, want %q", start, "2026-02-01")
	}
	if end != "2026-02-28" {
		t.Fatalf("end = %q, want %q", end, "2026-02-28")
	}

	start, end, err = checkinMonthRange("2024-02")
	if err != nil {
		t.Fatalf("expected valid leap month: %v", err)
	}
	if start != "2024-02-01" {
		t.Fatalf("leap start = %q, want %q", start, "2024-02-01")
	}
	if end != "2024-02-29" {
		t.Fatalf("leap end = %q, want %q", end, "2024-02-29")
	}
}

func TestCheckinMonthRangeRejectsInvalidMonth(t *testing.T) {
	if _, _, err := checkinMonthRange("2026-13"); err == nil {
		t.Fatal("expected invalid month error")
	}
	if _, _, err := checkinMonthRange("2026-06-01"); err == nil {
		t.Fatal("expected invalid month format error")
	}
}

func TestResolveCheckinRewardAmountFixed(t *testing.T) {
	got, err := resolveCheckinRewardAmount(&PublicSettings{
		CheckinRewardMode:   "fixed",
		CheckinRewardAmount: 1.235,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 1.24 {
		t.Fatalf("reward = %v, want 1.24", got)
	}
}

func TestResolveCheckinRewardAmountRandom(t *testing.T) {
	for i := 0; i < 100; i++ {
		got, err := resolveCheckinRewardAmount(&PublicSettings{
			CheckinRewardMode:      "random",
			CheckinRewardMinAmount: 0.25,
			CheckinRewardMaxAmount: 0.35,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got < 0.25 || got > 0.35 {
			t.Fatalf("reward = %v, want within [0.25, 0.35]", got)
		}
	}
}

func TestResolveCheckinRewardAmountRandomNormalizesInvertedRange(t *testing.T) {
	got, err := resolveCheckinRewardAmount(&PublicSettings{
		CheckinRewardMode:      "random",
		CheckinRewardMinAmount: 2,
		CheckinRewardMaxAmount: 1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 2 {
		t.Fatalf("reward = %v, want 2", got)
	}
}

func TestNormalizeCheckinRewardMode(t *testing.T) {
	if got := normalizeCheckinRewardMode(" random "); got != "random" {
		t.Fatalf("mode = %q, want random", got)
	}
	if got := normalizeCheckinRewardMode("fixed"); got != "fixed" {
		t.Fatalf("mode = %q, want fixed", got)
	}
	if got := normalizeCheckinRewardMode("bad"); got != "fixed" {
		t.Fatalf("mode = %q, want fixed", got)
	}
}

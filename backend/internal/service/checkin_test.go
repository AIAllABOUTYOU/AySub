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

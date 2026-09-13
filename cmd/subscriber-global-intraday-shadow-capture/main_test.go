package main

import (
	"testing"
	"time"
)

func TestMarketSessionSkipsWeekend(t *testing.T) {
	if _, ok := marketSession(time.Date(2026, 8, 16, 16, 0, 0, 0, time.UTC)); ok {
		t.Fatal("Sunday must not be a capture session")
	}
}
func TestComparisonUsesFifteenMinuteBucket(t *testing.T) {
	base := time.Date(2026, 8, 17, 14, 32, 0, 0, time.UTC)
	if got := comparison(base, legacySnapshot{id: "legacy", asOf: base.Add(5 * time.Minute)}); got != "freshness_match" {
		t.Fatalf("got %s", got)
	}
	if got := comparison(base, legacySnapshot{id: "legacy", asOf: base.Add(16 * time.Minute)}); got != "freshness_mismatch" {
		t.Fatalf("got %s", got)
	}
}

package main

import (
	"testing"
	"time"
)

func TestParseDatasets(t *testing.T) {
	equity, options, err := parseDatasets("equity,options")
	if err != nil {
		t.Fatalf("parse datasets: %v", err)
	}
	if !equity || !options {
		t.Fatalf("equity/options = %v/%v", equity, options)
	}
}

func TestParseDatasetsRejectsUnknown(t *testing.T) {
	_, _, err := parseDatasets("intraday")
	if err == nil {
		t.Fatal("expected dataset error")
	}
}

func TestParseObservationDate(t *testing.T) {
	day, err := parseObservationDate("2026-07-06")
	if err != nil {
		t.Fatalf("parse date: %v", err)
	}
	if !day.Equal(time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("day = %s", day)
	}
}

func TestResolveObservationDateRecomputesPreviousDay(t *testing.T) {
	first, err := resolveObservationDate("", time.Date(2026, 7, 20, 23, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("resolve first date: %v", err)
	}
	second, err := resolveObservationDate("", time.Date(2026, 7, 21, 23, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("resolve second date: %v", err)
	}
	if got, want := first.Format("2006-01-02"), "2026-07-19"; got != want {
		t.Fatalf("first date = %s, want %s", got, want)
	}
	if got, want := second.Format("2006-01-02"), "2026-07-20"; got != want {
		t.Fatalf("second date = %s, want %s", got, want)
	}
}

func TestResolveObservationDateKeepsExplicitDate(t *testing.T) {
	day, err := resolveObservationDate("2026-07-14", time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("resolve explicit date: %v", err)
	}
	if got, want := day.Format("2006-01-02"), "2026-07-14"; got != want {
		t.Fatalf("date = %s, want %s", got, want)
	}
}

func TestEnvDurationOrDefault(t *testing.T) {
	t.Setenv("SIGNALOPS_TEST_DURATION", "750ms")
	if got := envDurationOrDefault("SIGNALOPS_TEST_DURATION", time.Second); got != 750*time.Millisecond {
		t.Fatalf("duration = %s", got)
	}
}

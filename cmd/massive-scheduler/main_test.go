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

func TestEnvDurationOrDefault(t *testing.T) {
	t.Setenv("SIGNALOPS_TEST_DURATION", "750ms")
	if got := envDurationOrDefault("SIGNALOPS_TEST_DURATION", time.Second); got != 750*time.Millisecond {
		t.Fatalf("duration = %s", got)
	}
}

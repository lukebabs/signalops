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

	equity, options, err = parseDatasets("eod")
	if err != nil {
		t.Fatalf("parse eod: %v", err)
	}
	if !equity || options {
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

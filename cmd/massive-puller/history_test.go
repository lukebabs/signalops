package main

import (
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/adapters/marketdata/massive"
)

func TestParseObservationDatesSkipsWeekendsAndEnforcesBound(t *testing.T) {
	days, err := parseObservationDates("", "2026-07-10", "2026-07-14", 3)
	if err != nil {
		t.Fatalf("parse observation dates: %v", err)
	}
	if len(days) != 3 || days[0].Weekday() != time.Friday || days[1].Weekday() != time.Monday || days[2].Weekday() != time.Tuesday {
		t.Fatalf("days = %+v", days)
	}
	if _, err := parseObservationDates("", "2026-07-10", "2026-07-14", 2); err == nil {
		t.Fatal("expected explicit observation-day bound error")
	}
}

func TestParseObservationDatesRequiresCompleteRange(t *testing.T) {
	if _, err := parseObservationDates("", "2026-07-10", "", 10); err == nil {
		t.Fatal("expected incomplete range error")
	}
	if _, err := parseObservationDates("", "2026-07-14", "2026-07-10", 10); err == nil {
		t.Fatal("expected reversed range error")
	}
}

func TestSelectCompaniesUsesExactRequestedOrder(t *testing.T) {
	seed := []massive.MegacapCompanySeed{{Ticker: "MSFT"}, {Ticker: "AAPL"}, {Ticker: "NVDA"}}
	selected, err := selectCompanies(seed, "nvda,AAPL,nvda")
	if err != nil {
		t.Fatalf("select companies: %v", err)
	}
	if len(selected) != 2 || selected[0].Ticker != "NVDA" || selected[1].Ticker != "AAPL" {
		t.Fatalf("selected = %+v", selected)
	}
	if _, err := selectCompanies(seed, "NOTREAL"); err == nil {
		t.Fatal("expected unknown symbol error")
	}
}

func TestRemainingBudget(t *testing.T) {
	if value, err := remainingBudget(0, 100, "request"); err != nil || value != 0 {
		t.Fatalf("unbounded remaining = %d, %v", value, err)
	}
	if value, err := remainingBudget(10, 4, "request"); err != nil || value != 6 {
		t.Fatalf("bounded remaining = %d, %v", value, err)
	}
	if _, err := remainingBudget(10, 10, "request"); err == nil {
		t.Fatal("expected exhausted budget error")
	}
}

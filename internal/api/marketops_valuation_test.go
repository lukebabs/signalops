package api

import (
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestLatestValuationFamilyDoesNotPreferStaleAnnual(t *testing.T) {
	annual := &storage.MarketOpsValuationResultRecord{SessionDate: time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)}
	daily := &storage.MarketOpsValuationResultRecord{SessionDate: time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC)}
	if got := latestValuationFamily(annual, daily, false); got != daily {
		t.Fatalf("latest valuation family selected %v, want daily %v", got.SessionDate, daily.SessionDate)
	}
}

func TestLatestValuationFamilyPrefersAnnualOnTie(t *testing.T) {
	daily := &storage.MarketOpsValuationResultRecord{SessionDate: time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC)}
	annual := &storage.MarketOpsValuationResultRecord{SessionDate: daily.SessionDate}
	if got := latestValuationFamily(daily, annual, true); got != annual {
		t.Fatal("annual valuation should win when session dates tie")
	}
}

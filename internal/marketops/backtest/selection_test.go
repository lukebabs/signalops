package backtest

import (
	"testing"

	"github.com/lukebabs/signalops/internal/subscriber/eodrevisionpolicy"
)

func TestHistoricalEODSelectionDefaultsToInitialCapture(t *testing.T) {
	cfg := Config{}.withDefaults()
	selection, err := cfg.historicalEODSelection()
	if err != nil {
		t.Fatalf("historical selection: %v", err)
	}
	if selection.UsageContext != eodrevisionpolicy.HistoricalAssurance {
		t.Fatalf("usage context = %q, want %q", selection.UsageContext, eodrevisionpolicy.HistoricalAssurance)
	}
	if selection.SelectedObservationRole != eodrevisionpolicy.InitialCapture {
		t.Fatalf("observation role = %q, want %q", selection.SelectedObservationRole, eodrevisionpolicy.InitialCapture)
	}
}

func TestHistoricalEODSelectionRejectsCurrentMarketContext(t *testing.T) {
	_, err := (Config{EODDataSelectionContext: eodrevisionpolicy.CurrentMarketContext}).historicalEODSelection()
	if err == nil {
		t.Fatal("expected current_market_context to be rejected for a backtest")
	}
}

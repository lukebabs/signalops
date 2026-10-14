package eodrevisionpolicy

import "testing"

func TestSelectionIsDeterministicByContext(t *testing.T) {
	historical, err := SelectionFor(HistoricalAssurance)
	if err != nil || historical.SelectedObservationRole != InitialCapture {
		t.Fatalf("historical selection = %#v, %v", historical, err)
	}
	current, err := SelectionFor(CurrentMarketContext)
	if err != nil || current.SelectedObservationRole != LatestVerifiedValue {
		t.Fatalf("current selection = %#v, %v", current, err)
	}
}

func TestSelectionRejectsUnknownContext(t *testing.T) {
	if _, err := SelectionFor(UsageContext("browser_override")); err == nil {
		t.Fatal("expected unknown context to be rejected")
	}
}

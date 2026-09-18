package api

import (
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestCurrentEODContextResponseDeclaresLatestVerifiedContext(t *testing.T) {
	closeValue := 302.3
	response := currentEODContextResponse(storage.SubscriberCurrentEODContextRecord{
		Symbol:                  "AAPL",
		SessionDate:             time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC),
		Close:                   &closeValue,
		Provider:                "massive",
		SelectedObservationRole: "global_reobservation",
		SelectionPolicyVersion:  "s4-as-of-selection-v1",
		QualityState:            "usable",
		AsOfTime:                time.Date(2026, 8, 13, 12, 48, 31, 0, time.UTC),
	})
	if response["usage_context"] != "current_market_context" {
		t.Fatalf("usage_context = %v", response["usage_context"])
	}
	if response["selected_observation_role"] != "global_reobservation" {
		t.Fatalf("selected_observation_role = %v", response["selected_observation_role"])
	}
	if response["close"] != &closeValue {
		t.Fatalf("close = %v", response["close"])
	}
}

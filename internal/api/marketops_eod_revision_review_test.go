package api

import (
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestRevisionReviewResponseRetainsInitialAndRevisedEvidence(t *testing.T) {
	initial := 42819737.0
	revised := 43907632.0
	result := revisionReviewResponse([]storage.SubscriberEODRevisionDeltaRecord{{
		Symbol:            "AAPL",
		SessionDate:       time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC),
		FieldName:         "volume",
		InitialValue:      &initial,
		RevisedValue:      &revised,
		DeltaClass:        "provider_revision",
		Materiality:       "review_required",
		InitialObservedAt: time.Date(2026, 8, 12, 22, 1, 57, 0, time.UTC),
		RevisedObservedAt: time.Date(2026, 8, 13, 12, 48, 31, 0, time.UTC),
	}})
	if result["usage_context"] != "revision_review" || result["review_required_count"] != 1 {
		t.Fatalf("revision review summary = %+v", result)
	}
	deltas := result["deltas"].([]map[string]any)
	if len(deltas) != 1 || deltas[0]["initial_value"] != &initial || deltas[0]["revised_value"] != &revised {
		t.Fatalf("revision deltas = %+v", deltas)
	}
}

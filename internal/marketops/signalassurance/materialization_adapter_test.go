package signalassurance

import (
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestEligibleFromMaterializationUsesPointInTimeBars(t *testing.T) {
	confirmed := time.Date(2026, 8, 7, 20, 0, 0, 0, time.UTC)
	completed := confirmed
	materialization := storage.AlgorithmSignalMaterializationRecord{MaterializationID: "mat-1", TenantID: "tenant-1", AlgorithmID: "algo", AlgorithmVersion: "v1", MaterializationPolicyVersion: "policy-v1", MaterializationStatus: storage.AlgorithmSignalMaterializationStatusSucceeded, CompletedAt: &completed}
	proposal := storage.AlgorithmSignalProposalRecord{Score: .8, ProposalPayloadJSON: []byte(`{"asset_id":"asset:XYZ","symbol":"XYZ","direction":"upside"}`)}
	signal := storage.SignalLedgerRecord{SignalID: "sig-1", TenantID: "tenant-1", SignalType: "marketops.test", SignalTime: confirmed}
	events := []storage.NormalizedEventLedgerRecord{priceEvent("xyz-old", "XYZ", 100, confirmed.Add(-time.Hour), confirmed.Add(-time.Minute)), priceEvent("spy-old", "SPY", 500, confirmed.Add(-time.Hour), confirmed.Add(-time.Minute)), priceEvent("xyz-future", "XYZ", 120, confirmed.Add(time.Hour), confirmed.Add(time.Hour))}
	event, err := EligibleFromMaterialization(materialization, proposal, storage.AlgorithmResultRecord{}, signal, events, "SPY")
	if err != nil {
		t.Fatal(err)
	}
	if event.Direction != "bullish" || event.EvaluationMode != storage.SignalAssuranceModeResearch || event.EvaluationRunID != ResearchMaterializationRunID {
		t.Fatalf("unexpected event %#v", event)
	}
	if string(event.BaselineSnapshot) != `{"asset_price":100,"benchmark_price":500,"benchmark_symbol":"SPY","price_basis":"raw_regular_session_close"}` {
		t.Fatalf("baseline %s", event.BaselineSnapshot)
	}
}
func TestEligibleFromMaterializationRejectsUnresolvedDirection(t *testing.T) {
	now := time.Now().UTC()
	_, err := EligibleFromMaterialization(storage.AlgorithmSignalMaterializationRecord{MaterializationStatus: storage.AlgorithmSignalMaterializationStatusSucceeded}, storage.AlgorithmSignalProposalRecord{ProposalPayloadJSON: []byte(`{"asset_id":"a","symbol":"XYZ"}`)}, storage.AlgorithmResultRecord{}, storage.SignalLedgerRecord{SignalID: "s", SignalTime: now}, nil, "SPY")
	if err == nil {
		t.Fatal("expected unresolved direction rejection")
	}
}
func priceEvent(id, symbol string, close float64, observed, available time.Time) storage.NormalizedEventLedgerRecord {
	return storage.NormalizedEventLedgerRecord{EventID: id, ObservationTime: observed, ProcessingTime: available, NormalizedPayload: []byte(`{"symbol":"` + symbol + `","close":` + formatPrice(close) + `}`)}
}
func formatPrice(value float64) string {
	if value == 100 {
		return "100"
	}
	return "500"
}

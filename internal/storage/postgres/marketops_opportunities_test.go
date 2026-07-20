package postgres

import (
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestValidateMarketOpsOpportunity(t *testing.T) {
	record := validMarketOpsOpportunity()
	if err := validateMarketOpsOpportunity(record); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*storage.MarketOpsOpportunityRecord){
		"direction":   func(record *storage.MarketOpsOpportunityRecord) { record.Direction = "sideways" },
		"lifecycle":   func(record *storage.MarketOpsOpportunityRecord) { record.LifecycleStatus = "submitted" },
		"score":       func(record *storage.MarketOpsOpportunityRecord) { record.OpportunityScore = 1.1 },
		"evaluations": func(record *storage.MarketOpsOpportunityRecord) { record.HypothesisEvaluationIDs = nil },
		"dates": func(record *storage.MarketOpsOpportunityRecord) {
			record.LastEvaluatedDate = record.OpenedSessionDate.AddDate(0, 0, -1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			invalid := validMarketOpsOpportunity()
			mutate(&invalid)
			if validateMarketOpsOpportunity(invalid) == nil {
				t.Fatalf("expected validation failure: %+v", invalid)
			}
		})
	}
}

func validMarketOpsOpportunity() storage.MarketOpsOpportunityRecord {
	now := time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC)
	return storage.MarketOpsOpportunityRecord{
		OpportunityID: "mopp-1", TenantID: "tenant-local", AppID: "marketops", AssetID: "ticker:AAPL",
		Symbol: "AAPL", OpenedSessionDate: now, LastEvaluatedDate: now, Direction: "downside",
		Horizon: "5_to_20_sessions", LifecycleStatus: storage.MarketOpsOpportunityActive,
		OpportunityScore: .8, ConfidenceScore: .75, DomainDiversityScore: .67, ConflictScore: 0,
		HypothesisEvaluationIDs: []string{"eval-1", "eval-2"}, SupportingEvidenceIDs: []string{"evidence-1"},
		Summary: "AAPL downside opportunity.", OpportunityPayloadJSON: []byte(`{"research_only":true}`),
		Version: 1, ResearchOnly: true, BuildRunID: "run-1", DeterministicKey: "key-1",
	}
}

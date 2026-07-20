package postgres

import (
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestValidateMarketOpsHypothesisDefinition(t *testing.T) {
	record := validHypothesisDefinition()
	if err := validateMarketOpsHypothesisDefinition(record); err != nil {
		t.Fatal(err)
	}
	record.LifecycleStatus = "production"
	if validateMarketOpsHypothesisDefinition(record) == nil {
		t.Fatal("expected lifecycle validation")
	}
}
func TestValidateMarketOpsHypothesisEvaluation(t *testing.T) {
	record := validHypothesisEvaluation()
	if err := validateMarketOpsHypothesisEvaluation(record); err != nil {
		t.Fatal(err)
	}
	record.Eligible = false
	record.Triggered = true
	if validateMarketOpsHypothesisEvaluation(record) == nil {
		t.Fatal("expected triggered eligibility validation")
	}
	record = validHypothesisEvaluation()
	bad := 1.1
	record.TriggerScore = &bad
	if validateMarketOpsHypothesisEvaluation(record) == nil {
		t.Fatal("expected score validation")
	}
}

func validHypothesisDefinition() storage.MarketOpsHypothesisDefinitionRecord {
	return storage.MarketOpsHypothesisDefinitionRecord{TenantID: "tenant-1", HypothesisKey: "H004", HypothesisVersion: "v1", Title: "Term shift", Domain: "volatility_surface", Direction: "non_directional", Description: "research", Rationale: "test", RequiredFeaturesJSON: []byte(`[]`), RequiredTransitionsJSON: []byte(`[]`), QualityPolicyJSON: []byte(`{}`), EligibilityExpressionJSON: []byte(`{}`), TriggerExpressionJSON: []byte(`{}`), PersistenceRuleJSON: []byte(`{}`), CorroborationRuleJSON: []byte(`{}`), InvalidationRuleJSON: []byte(`{}`), ExpectedOutcomesJSON: []byte(`[]`), ScoringConfigJSON: []byte(`{}`), CalibrationPolicyJSON: []byte(`{}`), LifecycleStatus: storage.MarketOpsHypothesisLifecycleResearch}
}
func validHypothesisEvaluation() storage.MarketOpsHypothesisEvaluationRecord {
	score := .8
	now := time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC)
	return storage.MarketOpsHypothesisEvaluationRecord{EvaluationID: "mhypeval-1", TenantID: "tenant-1", HypothesisKey: "H004", HypothesisVersion: "v1", MarketStateID: "mstate-1", AssetID: "ticker:AAPL", Symbol: "AAPL", SessionDate: now, AsOfTime: now, Eligible: true, Triggered: true, TriggerScore: &score, EvaluationPayloadJSON: []byte(`{}`), EvaluationRunID: "run-1", DeterministicKey: "key-1"}
}

package postgres

import (
	"strings"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestValidateMarketOpsFeatureDefinition(t *testing.T) {
	record := validMarketOpsFeatureDefinitionRecord()
	if err := validateMarketOpsFeatureDefinition(record); err != nil {
		t.Fatalf("validate feature definition: %v", err)
	}
	record.ValueType = "vector"
	if err := validateMarketOpsFeatureDefinition(record); err == nil {
		t.Fatal("expected value type validation error")
	}
}

func TestValidateMarketOpsFeatureObservation(t *testing.T) {
	record := validMarketOpsFeatureObservationRecord()
	if err := validateMarketOpsFeatureObservation(record); err != nil {
		t.Fatalf("validate feature observation: %v", err)
	}
	textValue := "unexpected"
	record.TextValue = &textValue
	if err := validateMarketOpsFeatureObservation(record); err == nil {
		t.Fatal("expected typed value validation error")
	}
	record = validMarketOpsFeatureObservationRecord()
	record.NumericValue = nil
	if err := validateMarketOpsFeatureObservation(record); err == nil {
		t.Fatal("expected usable value validation error")
	}
}

func TestValidateMarketOpsMarketState(t *testing.T) {
	record := validMarketOpsMarketStateRecord()
	if err := validateMarketOpsMarketState(record); err != nil {
		t.Fatalf("validate market state: %v", err)
	}
	record.CompletenessRatio = 1.01
	if err := validateMarketOpsMarketState(record); err == nil {
		t.Fatal("expected completeness validation error")
	}
}

func TestValidateMarketOpsStateTransition(t *testing.T) {
	record := validMarketOpsStateTransitionRecord()
	if err := validateMarketOpsStateTransition(record); err != nil {
		t.Fatalf("validate state transition: %v", err)
	}
	negative := -1
	record.LookbackSessions = &negative
	if err := validateMarketOpsStateTransition(record); err == nil {
		t.Fatal("expected lookback validation error")
	}
}

func TestValidateMarketOpsEvidence(t *testing.T) {
	record := validMarketOpsEvidenceRecord()
	if err := validateMarketOpsEvidence(record); err != nil {
		t.Fatalf("validate evidence: %v", err)
	}
	invalid := 1.2
	record.RarityScore = &invalid
	err := validateMarketOpsEvidence(record)
	if err == nil || !strings.Contains(err.Error(), "rarity_score") {
		t.Fatalf("expected rarity score validation error, got %v", err)
	}
}

func validMarketOpsFeatureDefinitionRecord() storage.MarketOpsFeatureDefinitionRecord {
	return storage.MarketOpsFeatureDefinitionRecord{TenantID: "tenant-1", FeatureKey: "underlying.return_1d",
		FeatureVersion: "v1", Domain: "underlying_momentum", Title: "One-day return", ValueType: "numeric",
		CalculationSpec: []byte(`{"method":"close_return"}`), RequiredInputs: []byte(`["equity_eod.close"]`),
		QualityPolicy: []byte(`{"minimum_source_count":1}`), Status: storage.MarketOpsFeatureDefinitionStatusActive}
}

func validMarketOpsFeatureObservationRecord() storage.MarketOpsFeatureObservationRecord {
	value, quality := 0.012, 0.98
	return storage.MarketOpsFeatureObservationRecord{FeatureObservationID: "mfo-1", TenantID: "tenant-1", AppID: "marketops",
		AssetID: "asset:AAPL", Symbol: "AAPL", SessionDate: time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC),
		AsOfTime: time.Date(2026, 7, 19, 20, 0, 0, 0, time.UTC), FeatureKey: "underlying.return_1d", FeatureVersion: "v1",
		DimensionsJSON: []byte(`{}`), NumericValue: &value, QualityState: storage.MarketOpsQualityUsable, QualityScore: &quality,
		QualityDetailsJSON: []byte(`{}`), SourceEventIDs: []string{"event-1"}, CalculationRunID: "feature-run-1",
		DeterministicKey: "tenant-1:AAPL:2026-07-19:underlying.return_1d:v1"}
}

func validMarketOpsMarketStateRecord() storage.MarketOpsMarketStateRecord {
	quality := 0.96
	return storage.MarketOpsMarketStateRecord{MarketStateID: "mstate-1", TenantID: "tenant-1", AppID: "marketops",
		AssetID: "asset:AAPL", Symbol: "AAPL", SessionDate: time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC),
		AsOfTime: time.Date(2026, 7, 19, 20, 5, 0, 0, time.UTC), StateSchemaVersion: "marketops.state.v1",
		StatePayloadJSON: []byte(`{"underlying_momentum":{"return_1d":0.012}}`), FeatureObservationIDs: []string{"mfo-1"},
		FeatureCount: 1, RequiredFeatureCount: 1, CompletenessRatio: 1, QualityState: storage.MarketOpsQualityUsable,
		QualityScore: &quality, QualitySummaryJSON: []byte(`{"usable":1}`), BuildRunID: "state-run-1",
		DeterministicKey: "tenant-1:AAPL:2026-07-19:marketops.state.v1"}
}

func validMarketOpsStateTransitionRecord() storage.MarketOpsStateTransitionRecord {
	lookback, percentile := 20, 0.97
	return storage.MarketOpsStateTransitionRecord{TransitionID: "mtrans-1", TenantID: "tenant-1", AppID: "marketops",
		AssetID: "asset:AAPL", Symbol: "AAPL", SessionDate: time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC),
		AsOfTime: time.Date(2026, 7, 19, 20, 6, 0, 0, time.UTC), CurrentStateID: "mstate-1",
		FeatureKey: "underlying.return_1d", FeatureVersion: "v1", DimensionsJSON: []byte(`{}`), TransitionType: "zscore",
		LookbackSessions: &lookback, Percentile: &percentile, QualityState: storage.MarketOpsQualityUsable,
		TransitionPayloadJSON: []byte(`{}`), CalculationRunID: "transition-run-1",
		DeterministicKey: "tenant-1:AAPL:2026-07-19:return-zscore:v1"}
}

func validMarketOpsEvidenceRecord() storage.MarketOpsEvidenceRecord {
	rarity, quality := 0.97, 0.96
	return storage.MarketOpsEvidenceRecord{EvidenceID: "mevidence-1", TenantID: "tenant-1", AppID: "marketops",
		AssetID: "asset:AAPL", Symbol: "AAPL", SessionDate: time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC),
		AsOfTime: time.Date(2026, 7, 19, 20, 7, 0, 0, time.UTC), EvidenceType: "return_expansion",
		EvidenceVersion: "v1", Domain: "underlying_momentum", RarityScore: &rarity, QualityScore: &quality,
		Statement: "AAPL one-day return expanded above its recent baseline.", EvidencePayloadJSON: []byte(`{"observed":true}`),
		SourceFeatureIDs: []string{"mfo-1"}, SourceTransitionIDs: []string{"mtrans-1"},
		DeterministicKey: "tenant-1:AAPL:2026-07-19:return-expansion:v1"}
}

package signalassurance

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestEvaluateLifecyclePrecedenceAndExcursions(t *testing.T) {
	confirmed := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	assertion := testAssertion(confirmed, "bullish")
	contract := testContract(0.10, 20, map[string]any{"invalidation_threshold": -0.15})
	firstPrice := 96.0
	first, err := Evaluate(assertion, contract, nil, EvaluationMarketState{AsOf: confirmed.AddDate(0, 0, 5), AssetPrice: &firstPrice, TradingDaysActive: 5})
	if err != nil {
		t.Fatal(err)
	}
	if first.Persistence.NextState != storage.SignalAssertionActive || math.Abs(*first.Persistence.Evaluation.MFE+0.04) > 1e-9 || math.Abs(*first.Persistence.Evaluation.MAE+0.04) > 1e-9 {
		t.Fatalf("unexpected first result: %#v", first.Persistence)
	}
	materializedPrice := 112.0
	second, err := Evaluate(assertion, contract, &first.Persistence.Evaluation, EvaluationMarketState{AsOf: confirmed.AddDate(0, 0, 20), AssetPrice: &materializedPrice, TradingDaysActive: 20})
	if err != nil {
		t.Fatal(err)
	}
	if second.Persistence.NextState != storage.SignalAssertionMaterialized {
		t.Fatalf("state = %s, want materialized precedence", second.Persistence.NextState)
	}
	if math.Abs(*second.Persistence.Evaluation.MFE-0.12) > 1e-9 || math.Abs(*second.Persistence.Evaluation.MAE+0.04) > 1e-9 {
		t.Fatalf("excursions = %v/%v", *second.Persistence.Evaluation.MFE, *second.Persistence.Evaluation.MAE)
	}
}

func TestEvaluateBearishAndMissingInputs(t *testing.T) {
	confirmed := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	assertion := testAssertion(confirmed, "bearish")
	contract := testContract(0.05, 20, map[string]any{})
	price := 90.0
	result, err := Evaluate(assertion, contract, nil, EvaluationMarketState{AsOf: confirmed.AddDate(0, 0, 10), AssetPrice: &price, TradingDaysActive: 10})
	if err != nil {
		t.Fatal(err)
	}
	if result.Persistence.NextState != storage.SignalAssertionMaterialized || math.Abs(*result.Persistence.Evaluation.AbsoluteReturn+0.1) > 1e-9 {
		t.Fatalf("bearish normalization failed: %#v", result.Persistence)
	}
	missing, err := Evaluate(assertion, contract, nil, EvaluationMarketState{AsOf: confirmed.AddDate(0, 0, 1), TradingDaysActive: 1})
	if err != nil {
		t.Fatal(err)
	}
	if missing.Persistence.Evaluation.InputCompleteness != storage.SignalAssuranceInputIncomplete || missing.Persistence.NextState != storage.SignalAssertionActive {
		t.Fatalf("missing input result: %#v", missing.Persistence)
	}
}

func TestDecodeEligibleEventAndRegistration(t *testing.T) {
	value := []byte(`{"eligible_event_id":"eligible-1","tenant_id":"tenant-1","signal_id":"signal-1","signal_ledger_id":"signal-1","asset_id":"asset-1","symbol":"xyz","signal_type":"TEST","direction":"bullish","status":"confirmed","algorithm":"test","algorithm_version":"v1","confirmed_at":"2026-08-01T00:00:00Z","event_available_at":"2026-08-01T00:01:00Z","confirmation_rule_version":"test:v1","validation_contract_ref":"contract-1","baseline_snapshot":{"asset_price":100},"baseline_provenance":{"price_source":"canonical"},"evaluation_mode":"LIVE"}`)
	event, err := DecodeEligibleEvent(value)
	if err != nil {
		t.Fatal(err)
	}
	registration, err := AssertionRegistration(event, testContract(0.1, 20, map[string]any{}))
	if err != nil {
		t.Fatal(err)
	}
	if registration.Assertion.AssertionID == "" || registration.Assertion.RegistrationIdempotencyKey == "" || registration.Assertion.State != storage.SignalAssertionActive {
		t.Fatalf("invalid registration: %#v", registration.Assertion)
	}
	if _, err := DecodeEligibleEvent(append(value[:len(value)-1], []byte(`,"unexpected":true}`)...)); err == nil {
		t.Fatal("unknown eligible event field accepted")
	}
}

func testAssertion(confirmed time.Time, direction string) storage.SignalAssertionRecord {
	return storage.SignalAssertionRecord{AssertionID: "assertion-1", TenantID: "tenant-1", State: storage.SignalAssertionActive, SignalDirection: direction, ConfirmedAt: confirmed, EvaluationMode: storage.SignalAssuranceModeLive, BaselineSnapshotJSON: []byte(`{"asset_price":100}`), TransitionSequence: 1}
}
func testContract(threshold float64, horizon int, config map[string]any) storage.SignalValidationContractRecord {
	encoded, _ := json.Marshal(config)
	return storage.SignalValidationContractRecord{ContractID: "contract-1", SignalType: "TEST", ContractVersion: "v1", Direction: "bullish", PrimaryMetric: "absolute_return", ComparisonOperator: ">=", Threshold: threshold, EvaluationWindowsJSON: []byte(`[5,10,20]`), MaxHorizonTradingDays: horizon, MaterializationPolicy: "threshold", ConfigJSON: encoded, Active: true, ContractScopeKey: "test|v1"}
}

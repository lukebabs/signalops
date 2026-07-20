package hypotheses

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestResearchDefinitionsAreBoundedAndResearchOnly(t *testing.T) {
	definitions := ResearchDefinitions("tenant-local")
	if len(definitions) != 4 {
		t.Fatalf("definitions=%d", len(definitions))
	}
	want := []string{"H001", "H004", "H006", "H007"}
	for index, definition := range definitions {
		if definition.HypothesisKey != want[index] || definition.LifecycleStatus != storage.MarketOpsHypothesisLifecycleResearch || !strings.Contains(string(definition.CalibrationPolicyJSON), `"production_materialization_allowed":false`) {
			t.Fatalf("definition=%+v", definition)
		}
	}
}

func TestEvaluatePersistsMissingInputReasonsAndStableIdentity(t *testing.T) {
	definition := ResearchDefinitions("tenant-local")[0]
	state := testState()
	first, err := Evaluate("run-one", definition, state, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Evaluate("run-two", definition, state, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.Eligible || first.Triggered || len(first.ReasonCodes) == 0 || first.EvaluationID != second.EvaluationID || first.DeterministicKey != second.DeterministicKey {
		t.Fatalf("unexpected rejected evaluation: first=%+v second=%+v", first, second)
	}
	if first.TriggerScore != nil || first.QualityScore == nil {
		t.Fatalf("rejected score semantics are incorrect: %+v", first)
	}
}

func TestEvaluateResearchHypothesesPositiveFixtures(t *testing.T) {
	definitions := ResearchDefinitions("tenant-local")
	cases := []struct {
		key          string
		observations []storage.MarketOpsFeatureObservationRecord
		transitions  []storage.MarketOpsStateTransitionRecord
	}{
		{"H001", []storage.MarketOpsFeatureObservationRecord{
			feature("rsi_14", 78, nil), feature("surface_coverage_ratio", .95, nil), feature("iv", .36, dims("put", 30, .25)), feature("iv", .39, dims("put", 60, .25)), feature("extrinsic_premium", 4.2, dims("put", 30, .25)), feature("oi_change_1d", 250, dims("put", 30, .25)),
		}, []storage.MarketOpsStateTransitionRecord{
			transition("iv", .03, dims("put", 30, .25), 2, nil, nil), transition("iv", .04, dims("put", 60, .25), 2, nil, nil), transition("extrinsic_premium", .4, dims("put", 30, .25), 2, nil, nil), transition("oi_change_1d", 250, dims("put", 30, .25), 2, nil, nil),
		}},
		{"H004", []storage.MarketOpsFeatureObservationRecord{
			feature("atm_iv_30d", .42, nil), feature("atm_iv_60d", .37, nil), feature("atm_iv_90d", .34, nil), feature("surface_coverage_ratio", 1, nil),
		}, []storage.MarketOpsStateTransitionRecord{
			transition("atm_iv_30d", .03, nil, 2, nil, nil), transition("atm_iv_60d", .02, nil, 2, nil, nil), transition("atm_iv_90d", .01, nil, 2, nil, nil),
		}},
		{"H006", []storage.MarketOpsFeatureObservationRecord{
			feature("return_1d", 2.5, nil), feature("mid_premium", 3.2, dims("put", 30, .25)), feature("iv", .31, dims("put", 30, .25)),
		}, []storage.MarketOpsStateTransitionRecord{
			transition("mid_premium", -.2, dims("put", 30, .25), 1, nil, nil), transition("iv", -.03, dims("put", 30, .25), 1, nil, nil),
		}},
		{"H007", []storage.MarketOpsFeatureObservationRecord{
			feature("oi_change_1d", 500, dims("call", 30, .25)), feature("surface_coverage_ratio", .8, nil),
		}, []storage.MarketOpsStateTransitionRecord{
			transition("oi_change_1d", 500, dims("call", 30, .25), 1, floatPointer(2.5), floatPointer(.98)),
		}},
	}
	for _, testCase := range cases {
		t.Run(testCase.key, func(t *testing.T) {
			definition := definitionByKey(definitions, testCase.key)
			result, err := Evaluate("positive-run", definition, testState(), testCase.observations, testCase.transitions, nil)
			if err != nil {
				t.Fatal(err)
			}
			if !result.Eligible || !result.Triggered || result.TriggerScore == nil || !contains(result.ReasonCodes, "triggered_research_only") {
				t.Fatalf("positive evaluation did not trigger: %+v", result)
			}
		})
	}
}

func TestEvaluateAddsOpportunityCompatibilityProfile(t *testing.T) {
	definitions := ResearchDefinitions("tenant-local")
	cases := []struct {
		key, expectedDirection string
		observations           []storage.MarketOpsFeatureObservationRecord
		transitions            []storage.MarketOpsStateTransitionRecord
	}{
		{"H006", "downside", []storage.MarketOpsFeatureObservationRecord{
			feature("return_1d", 2.5, nil), feature("mid_premium", 3.2, dims("put", 30, .25)), feature("iv", .31, dims("put", 30, .25)),
		}, []storage.MarketOpsStateTransitionRecord{
			transition("mid_premium", -.2, dims("put", 30, .25), 1, nil, nil), transition("iv", -.03, dims("put", 30, .25), 1, nil, nil),
		}},
		{"H007", "upside", []storage.MarketOpsFeatureObservationRecord{
			feature("oi_change_1d", 500, dims("call", 30, .25)), feature("surface_coverage_ratio", .8, nil),
		}, []storage.MarketOpsStateTransitionRecord{
			transition("oi_change_1d", 500, dims("call", 30, .25), 1, floatPointer(2.5), floatPointer(.98)),
		}},
	}
	for _, testCase := range cases {
		result, err := Evaluate("profile-run", definitionByKey(definitions, testCase.key), testState(), testCase.observations, testCase.transitions, nil)
		if err != nil {
			t.Fatal(err)
		}
		var payload map[string]any
		if err := json.Unmarshal(result.EvaluationPayloadJSON, &payload); err != nil {
			t.Fatal(err)
		}
		if payload["resolved_direction"] != testCase.expectedDirection || payload["horizon"] != "5_to_20_sessions" {
			t.Fatalf("key=%s payload=%v", testCase.key, payload)
		}
	}
}

func TestEvaluateEligibleNonTrigger(t *testing.T) {
	definition := definitionByKey(ResearchDefinitions("tenant-local"), "H007")
	result, err := Evaluate("non-trigger-run", definition, testState(), []storage.MarketOpsFeatureObservationRecord{
		feature("oi_change_1d", 50, dims("call", 30, .25)),
		feature("surface_coverage_ratio", .8, nil),
	}, []storage.MarketOpsStateTransitionRecord{
		transition("oi_change_1d", 50, dims("call", 30, .25), 1, floatPointer(1), floatPointer(.7)),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Eligible || result.Triggered || result.TriggerScore == nil || !contains(result.ReasonCodes, "threshold_not_met:oi_change_below_minimum") {
		t.Fatalf("unexpected eligible non-trigger evaluation: %+v", result)
	}
}

func testState() storage.MarketOpsMarketStateRecord {
	quality := 1.0
	return storage.MarketOpsMarketStateRecord{MarketStateID: "mstate-test", TenantID: "tenant-local", AppID: "marketops", AssetID: "ticker:AAPL", Symbol: "AAPL", SessionDate: time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC), AsOfTime: time.Date(2026, 7, 19, 23, 59, 59, 0, time.UTC), StateSchemaVersion: "marketops.market_state.v1", QualityState: storage.MarketOpsQualityUsable, QualityScore: &quality}
}

func feature(key string, value float64, dimensions map[string]any) storage.MarketOpsFeatureObservationRecord {
	encoded, _ := json.Marshal(dimensions)
	if dimensions == nil {
		encoded = []byte(`{}`)
	}
	return storage.MarketOpsFeatureObservationRecord{FeatureObservationID: "feature-" + key + string(encoded), FeatureKey: key, FeatureVersion: "v1", DimensionsJSON: encoded, NumericValue: floatPointer(value), QualityState: storage.MarketOpsQualityUsable}
}

func transition(key string, value float64, dimensions map[string]any, persistence int, zscore, percentile *float64) storage.MarketOpsStateTransitionRecord {
	encoded, _ := json.Marshal(dimensions)
	if dimensions == nil {
		encoded = []byte(`{}`)
	}
	return storage.MarketOpsStateTransitionRecord{TransitionID: "transition-" + key + string(encoded), FeatureKey: key, FeatureVersion: "v1", DimensionsJSON: encoded, TransitionValue: floatPointer(value), PersistenceSessions: &persistence, ZScore: zscore, Percentile: percentile, QualityState: storage.MarketOpsQualityUsable}
}

func definitionByKey(definitions []storage.MarketOpsHypothesisDefinitionRecord, key string) storage.MarketOpsHypothesisDefinitionRecord {
	for _, definition := range definitions {
		if definition.HypothesisKey == key {
			return definition
		}
	}
	return storage.MarketOpsHypothesisDefinitionRecord{}
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func floatPointer(value float64) *float64 { return &value }

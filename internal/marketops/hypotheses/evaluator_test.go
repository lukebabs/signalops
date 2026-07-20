package hypotheses

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/adapters/marketdata/massive"
	marketopsoptions "github.com/lukebabs/signalops/internal/marketops/options"
	marketopsstate "github.com/lukebabs/signalops/internal/marketops/state"
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

func TestG143ProviderShapedEvidenceMakesResearchHypothesesEligible(t *testing.T) {
	start := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	optionSessions := []time.Time{start.AddDate(0, 0, 22), start.AddDate(0, 0, 23), start.AddDate(0, 0, 24)}
	requestIndex := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/snapshot/options/AAPL" || requestIndex >= len(optionSessions) {
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		session := optionSessions[requestIndex]
		spot := 100 + float64(22+requestIndex)*1.25
		results := g143MassiveSurface(session, requestIndex, spot)
		_ = json.NewEncoder(w).Encode(map[string]any{"request_id": fmt.Sprintf("req-g143-%d", requestIndex), "results": results})
		requestIndex++
	}))
	defer server.Close()
	client, err := massive.NewClient(massive.ClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	chain := []storage.MarketOpsOptionsChainRecord{}
	for _, session := range optionSessions {
		batch, err := client.ListOptionChainSnapshotFilteredWithMetadata(context.Background(), "AAPL", massive.OptionChainSnapshotFilter{Limit: 50, MaxPages: 1})
		if err != nil {
			t.Fatal(err)
		}
		converted := make([]storage.MarketOpsOptionsChainRecord, 0, len(batch.Records))
		for _, providerRecord := range batch.Records {
			record, err := marketopsoptions.ChainRecordFromMassiveSnapshot("tenant-local", "src-massive", "g143-provider", providerRecord)
			if err != nil {
				t.Fatal(err)
			}
			converted = append(converted, record)
		}
		selected := marketopsoptions.SelectRequiredSurfaceEvidence(session, converted)
		if len(selected) != marketopsoptions.RequiredSurfaceCellCount || !marketopsoptions.AssessAnalyticsReadiness(session, selected).Ready {
			t.Fatalf("provider surface was not analytics ready: %+v", selected)
		}
		chain = append(chain, selected...)
	}
	equity := make([]storage.NormalizedEventLedgerRecord, 0, 25)
	for index := 0; index < 25; index++ {
		session := start.AddDate(0, 0, index)
		closeValue := 100 + float64(index)*1.25
		payload, _ := json.Marshal(map[string]any{"symbol": "AAPL", "open": closeValue - .5, "high": closeValue + 1, "low": closeValue - 1, "close": closeValue, "volume": 1000000 + index*1000})
		equity = append(equity, storage.NormalizedEventLedgerRecord{EventID: fmt.Sprintf("evt-g143-%02d", index), TenantID: "tenant-local", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", Dataset: "equity_eod_prices", ObservationTime: session, ProcessingTime: session.Add(time.Hour), NormalizedPayload: payload})
	}
	built, err := marketopsstate.Build(marketopsstate.BuildConfig{TenantID: "tenant-local", Symbol: "AAPL", AssetID: "asset-aapl", RunID: "g143-build"}, marketopsstate.BuildInput{EquityEvents: equity, OptionChain: chain})
	if err != nil {
		t.Fatal(err)
	}
	finalSession := optionSessions[len(optionSessions)-1]
	finalState := built.States[len(built.States)-1]
	observations := g143ObservationsForSession(built.Observations, finalSession)
	transitions := g143TransitionsForSession(built.Transitions, finalSession)
	evidence := g143EvidenceForSession(built.Evidence, finalSession)
	for _, definition := range ResearchDefinitions("tenant-local") {
		result, err := Evaluate("g143-evaluate", definition, finalState, observations, transitions, evidence)
		if err != nil {
			t.Fatal(err)
		}
		if !result.Eligible {
			t.Fatalf("%s remained ineligible from provider-shaped evidence: %v", definition.HypothesisKey, result.ReasonCodes)
		}
	}
}

func g143MassiveSurface(session time.Time, index int, spot float64) []map[string]any {
	cells := []struct {
		suffix, optionType string
		dte                int
		delta, iv          float64
	}{
		{"ATM30", "call", 30, .50, .30}, {"ATM60", "call", 60, .50, .32}, {"ATM90", "call", 90, .50, .34},
		{"P30D25", "put", 30, -.25, .36}, {"C30D25", "call", 30, .25, .28},
		{"P60D25", "put", 60, -.25, .38}, {"C60D25", "call", 60, .25, .30},
	}
	results := make([]map[string]any, 0, len(cells))
	for _, cell := range cells {
		strike := spot
		if cell.optionType == "put" && mathAbs(cell.delta) < .4 {
			strike = spot - 10
		}
		if cell.optionType == "call" && mathAbs(cell.delta) < .4 {
			strike = spot + 10
		}
		oi := 100 + index*25
		if cell.suffix == "P30D25" {
			oi = []int{100, 200, 450}[index]
		}
		if cell.suffix == "C30D25" {
			oi = []int{100, 150, 225}[index]
		}
		bid := 2.0 + float64(index)
		lastUpdated := session.Add(20 * time.Hour).UnixNano()
		results = append(results, map[string]any{
			"day":                map[string]any{"close": bid + .1, "volume": 100 + index*10, "last_updated": lastUpdated},
			"details":            map[string]any{"ticker": fmt.Sprintf("O:AAPL%s%s", session.Format("060102"), cell.suffix), "contract_type": cell.optionType, "expiration_date": session.AddDate(0, 0, cell.dte).Format("2006-01-02"), "strike_price": strike, "exercise_style": "american", "shares_per_contract": 100},
			"last_quote":         map[string]any{"bid": bid, "ask": bid + .2, "last_updated": lastUpdated},
			"greeks":             map[string]any{"delta": cell.delta, "gamma": .02, "theta": -.01, "vega": .10},
			"implied_volatility": cell.iv + float64(index)*.03,
			"open_interest":      oi,
			"underlying_asset":   map[string]any{"ticker": "AAPL", "price": spot},
		})
	}
	return results
}

func g143ObservationsForSession(records []storage.MarketOpsFeatureObservationRecord, session time.Time) []storage.MarketOpsFeatureObservationRecord {
	out := []storage.MarketOpsFeatureObservationRecord{}
	for _, record := range records {
		if record.SessionDate.Equal(session) {
			out = append(out, record)
		}
	}
	return out
}

func g143TransitionsForSession(records []storage.MarketOpsStateTransitionRecord, session time.Time) []storage.MarketOpsStateTransitionRecord {
	out := []storage.MarketOpsStateTransitionRecord{}
	for _, record := range records {
		if record.SessionDate.Equal(session) {
			out = append(out, record)
		}
	}
	return out
}

func g143EvidenceForSession(records []storage.MarketOpsEvidenceRecord, session time.Time) []storage.MarketOpsEvidenceRecord {
	out := []storage.MarketOpsEvidenceRecord{}
	for _, record := range records {
		if record.SessionDate.Equal(session) {
			out = append(out, record)
		}
	}
	return out
}

func mathAbs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
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
	return storage.MarketOpsMarketStateRecord{MarketStateID: "mstate-test", TenantID: "tenant-local", AppID: "marketops", AssetID: "ticker:AAPL", Symbol: "AAPL", SessionDate: time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC), AsOfTime: time.Date(2026, 7, 19, 23, 59, 59, 0, time.UTC), StateSchemaVersion: marketopsstate.StateSchemaVersion, QualityState: storage.MarketOpsQualityUsable, QualityScore: &quality}
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

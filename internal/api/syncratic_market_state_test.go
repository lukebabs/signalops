package api

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func marketStateContextFixture() *fakeQueryRepository {
	session := time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC)
	quality := .8
	score := .7
	state := storage.MarketOpsMarketStateRecord{MarketStateID: "mstate-aapl-1", TenantID: "tenant-1", AppID: "marketops", AssetID: "ticker:AAPL", Symbol: "AAPL", SessionDate: session, AsOfTime: session.Add(20 * time.Hour), StateSchemaVersion: "marketops.state.v1", FeatureObservationIDs: []string{"feat-1"}, FeatureCount: 1, RequiredFeatureCount: 39, CompletenessRatio: .14, QualityState: storage.MarketOpsQualityPartial, QualityScore: &quality}
	return &fakeQueryRepository{
		marketOpsMarketStates:          []storage.MarketOpsMarketStateRecord{state},
		marketOpsFeatureObservations:   []storage.MarketOpsFeatureObservationRecord{{FeatureObservationID: "feat-1", TenantID: "tenant-1", AppID: "marketops", AssetID: "ticker:AAPL", Symbol: "AAPL", SessionDate: session, FeatureKey: "atm_iv_30d", FeatureVersion: "marketops.feature.v1", NumericValue: &score, QualityState: storage.MarketOpsQualityUsable}},
		marketOpsHypothesisDefinitions: []storage.MarketOpsHypothesisDefinitionRecord{{TenantID: "tenant-1", HypothesisKey: "H001", HypothesisVersion: "v1", Domain: "volatility", LifecycleStatus: storage.MarketOpsHypothesisLifecycleResearch}},
		marketOpsHypothesisEvaluations: []storage.MarketOpsHypothesisEvaluationRecord{{EvaluationID: "eval-1", TenantID: "tenant-1", AppID: "marketops", HypothesisKey: "H001", HypothesisVersion: "v1", MarketStateID: "mstate-aapl-1", AssetID: "ticker:AAPL", Symbol: "AAPL", SessionDate: session, EvidenceIDs: []string{"evidence-1"}, ReasonCodes: []string{"quality_blocked"}}},
		marketOpsEvidence:              []storage.MarketOpsEvidenceRecord{{EvidenceID: "evidence-1", TenantID: "tenant-1", AppID: "marketops", AssetID: "ticker:AAPL", Symbol: "AAPL", SessionDate: session, EvidenceType: "feature_observation", EvidenceVersion: "v1", Statement: "AAPL observed volatility state", SourceFeatureIDs: []string{"feat-1"}}},
		backtestCalibrationSummaries:   []storage.MarketOpsBacktestCalibrationSummaryRecord{{SummaryID: "cal-1", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", ParametersJSON: []byte(`{"summary_version":"marketops.hypothesis_calibration.v1","hypothesis_key":"H001","hypothesis_versions":["v1"],"minimum_sample_size":30,"warnings":["sample_size_below_minimum:v1:2/30"],"versions":{"v1":{"hypothesis_version":"v1","overall":{"independent_samples":2,"below_minimum_sample_size":true}}}}`)}},
	}
}

func TestMarketStateContextDeterministicAndAskV2PreservesCalibrationWarning(t *testing.T) {
	repo := marketStateContextFixture()
	first, err := buildMarketStateSyncraticContext(context.Background(), repo, "tenant-1", "mstate-aapl-1", "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := buildMarketStateSyncraticContext(context.Background(), repo, "tenant-1", "mstate-aapl-1", "")
	if err != nil {
		t.Fatal(err)
	}
	if first.ContextWindowID != second.ContextWindowID || first.EvidenceDigest != second.EvidenceDigest || string(first.QualityWarningsJSON) != string(second.QualityWarningsJSON) {
		t.Fatalf("identity changed: %#v %#v", first, second)
	}
	if len(first.MarketStateIDs) != 1 || len(first.CalibrationSummaryIDs) != 1 || first.ContextPayloadVersion != marketStateContextPayloadVersion {
		t.Fatalf("context=%+v", first)
	}
	prompt, meta, err := buildMarketStateAskPrompt(first, syncraticAskRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if meta.PromptBuilderVersion != marketStateAskPromptVersion || !strings.Contains(prompt, "sample_size_below_minimum:v1:2/30") || !strings.Contains(prompt, "HISTORICAL_ASSOCIATION") {
		t.Fatalf("prompt contract missing: %s", prompt)
	}
}

func TestMarketStateContextConflictingTickerIsBlockedAndExcluded(t *testing.T) {
	repo := marketStateContextFixture()
	repo.marketOpsEvidence[0].Statement = "MSFT observed volatility state"
	record, err := buildMarketStateSyncraticContext(context.Background(), repo, "tenant-1", "mstate-aapl-1", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(record.MarketOpsEvidenceIDs) != 0 || !strings.Contains(string(record.SummaryMetricsJSON), "data_quality_blocked") || !strings.Contains(string(record.QualityWarningsJSON), "evidence_symbol_conflict") {
		t.Fatalf("purity not blocked: metrics=%s warnings=%s ids=%v", record.SummaryMetricsJSON, record.QualityWarningsJSON, record.MarketOpsEvidenceIDs)
	}
}

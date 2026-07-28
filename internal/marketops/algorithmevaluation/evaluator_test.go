package algorithmevaluation

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type fakeRepository struct {
	events       []storage.NormalizedEventLedgerRecord
	observations []storage.MarketOpsFeatureObservationRecord
	runs         []storage.MarketOpsAlgorithmEvaluationRunRecord
	results      []storage.MarketOpsAlgorithmEvaluationResultRecord
	outcomes     []storage.MarketOpsAlgorithmEvaluationOutcomeRecord
	definitions  []storage.PlatformPrimitiveDefinitionRecord
}

func (f *fakeRepository) UpsertMarketOpsAlgorithmEvaluationRun(_ context.Context, record storage.MarketOpsAlgorithmEvaluationRunRecord) error {
	f.runs = append(f.runs, record)
	return nil
}
func (f *fakeRepository) GetMarketOpsAlgorithmEvaluationRun(context.Context, string, string) (storage.MarketOpsAlgorithmEvaluationRunRecord, error) {
	return storage.MarketOpsAlgorithmEvaluationRunRecord{}, fmt.Errorf("not found")
}
func (f *fakeRepository) ListMarketOpsAlgorithmEvaluationRuns(context.Context, storage.MarketOpsAlgorithmEvaluationRunFilter) ([]storage.MarketOpsAlgorithmEvaluationRunRecord, error) {
	return f.runs, nil
}
func (f *fakeRepository) InsertMarketOpsAlgorithmEvaluationResult(_ context.Context, record storage.MarketOpsAlgorithmEvaluationResultRecord) error {
	f.results = append(f.results, record)
	return nil
}
func (f *fakeRepository) ListMarketOpsAlgorithmEvaluationResults(context.Context, storage.MarketOpsAlgorithmEvaluationResultFilter) ([]storage.MarketOpsAlgorithmEvaluationResultRecord, error) {
	return f.results, nil
}
func (f *fakeRepository) UpsertMarketOpsAlgorithmEvaluationOutcome(_ context.Context, record storage.MarketOpsAlgorithmEvaluationOutcomeRecord) error {
	f.outcomes = append(f.outcomes, record)
	return nil
}
func (f *fakeRepository) ListMarketOpsAlgorithmEvaluationOutcomes(context.Context, storage.MarketOpsAlgorithmEvaluationOutcomeFilter) ([]storage.MarketOpsAlgorithmEvaluationOutcomeRecord, error) {
	return f.outcomes, nil
}
func (f *fakeRepository) UpsertMarketOpsAlgorithmEvaluationBackfillCampaign(context.Context, storage.MarketOpsAlgorithmEvaluationBackfillCampaignRecord) error {
	return nil
}
func (f *fakeRepository) GetMarketOpsAlgorithmEvaluationBackfillCampaign(context.Context, string, string) (storage.MarketOpsAlgorithmEvaluationBackfillCampaignRecord, error) {
	return storage.MarketOpsAlgorithmEvaluationBackfillCampaignRecord{}, fmt.Errorf("not found")
}
func (f *fakeRepository) ListMarketOpsAlgorithmEvaluationBackfillCampaigns(context.Context, storage.MarketOpsAlgorithmEvaluationBackfillCampaignFilter) ([]storage.MarketOpsAlgorithmEvaluationBackfillCampaignRecord, error) {
	return nil, nil
}
func (f *fakeRepository) ListMarketOpsBacktestNormalizedEvents(context.Context, storage.MarketOpsBacktestEventFilter) ([]storage.NormalizedEventLedgerRecord, error) {
	return f.events, nil
}
func (f *fakeRepository) ListMarketOpsFeatureObservations(context.Context, storage.MarketOpsFeatureObservationFilter) ([]storage.MarketOpsFeatureObservationRecord, error) {
	return f.observations, nil
}
func (f *fakeRepository) ListPlatformPrimitiveDefinitions(context.Context, storage.PlatformPrimitiveDefinitionFilter) ([]storage.PlatformPrimitiveDefinitionRecord, error) {
	return f.definitions, nil
}

func TestWalkForwardDoesNotUseFutureSamples(t *testing.T) {
	start := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	repo := &fakeRepository{}
	close := 100.0
	for i := 0; i < 30; i++ {
		value := float64(i)
		if i == 29 {
			value = 10000
		}
		close += 1
		payload, _ := json.Marshal(map[string]any{"symbol": "AAPL", "close": close, "high": close + 1, "low": close - 1, "features": map[string]any{"daily_return_pct": value}})
		metadata, _ := json.Marshal(map[string]any{"platform_definition_versions": map[string]any{"source": "1.0.0"}, "quality": map[string]any{"quality_state": "usable"}})
		repo.events = append(repo.events, storage.NormalizedEventLedgerRecord{EventID: fmt.Sprintf("evt-%02d", i), TenantID: "tenant-local", ObservationTime: start.AddDate(0, 0, i), NormalizedPayload: payload, MetadataJSON: metadata})
	}
	_, err := Run(context.Background(), repo, Config{RunID: "eval-test", TenantID: "tenant-local", UniverseGroup: "top50_megacap", Symbols: []string{"AAPL"}, AlgorithmIDs: []string{"signalops.algorithms.zscore_anomaly_v1"}, Modes: []string{storage.MarketOpsAlgorithmEvaluationModeWalkForward}, WindowStart: start, WindowEnd: start.AddDate(0, 0, 30), AsOf: start.AddDate(0, 0, 30), LookbackSessions: 20, MinSamples: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.results) == 0 {
		t.Fatal("expected walk-forward results")
	}
	first := repo.results[0]
	var payload map[string]any
	if err := json.Unmarshal(first.ResultPayloadJSON, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["mean"].(float64) > 100 {
		t.Fatalf("walk-forward mean included future outlier: %#v", payload)
	}
	if len(repo.outcomes) == 0 {
		t.Fatal("expected forward outcomes")
	}
}

func TestRiskRewardRespectsRequestedModes(t *testing.T) {
	start := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	repo := &fakeRepository{}
	required := []string{"range_position_252d", "rsi_14", "return_5d", "volume_ratio_10d", "distance_sma_50_pct", "distance_sma_200_pct", "sma_50_slope_20d_pct", "atr_14_pct"}
	for _, key := range required {
		value := 1.
		if key == "range_position_252d" {
			value = 5
		}
		repo.observations = append(repo.observations, storage.MarketOpsFeatureObservationRecord{FeatureObservationID: key, TenantID: "tenant-local", AppID: "marketops", Symbol: "AAPL", SessionDate: start, FeatureKey: key, NumericValue: &value, QualityState: "usable", QualityDetailsJSON: []byte(`{"input_provenance":[{"event_id":"evt"}]}`)})
	}
	output, err := evaluateRiskReward(context.Background(), repo, Config{RunID: "risk-test", TenantID: "tenant-local", Symbols: []string{"AAPL"}, AlgorithmIDs: []string{"signalops.algorithms.risk_reward_temporal_v1"}, Modes: []string{storage.MarketOpsAlgorithmEvaluationModeRetrospective}, WindowStart: start, WindowEnd: start.AddDate(0, 0, 1), AsOf: start.AddDate(0, 0, 1)}, "signalops.algorithms.risk_reward_temporal_v1")
	if err != nil {
		t.Fatal(err)
	}
	if len(output) != 1 || output[0].record.EvaluationMode != storage.MarketOpsAlgorithmEvaluationModeRetrospective {
		t.Fatalf("unexpected risk modes: %#v", output)
	}
}

func TestRegistryEnforcementRejectsUnprovenancedEvents(t *testing.T) {
	start := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	payload := []byte(`{"symbol":"AAPL","close":101,"high":102,"low":100}`)
	repo := &fakeRepository{events: []storage.NormalizedEventLedgerRecord{{EventID: "evt-unprovenanced", TenantID: "tenant-local", SourceAdapter: "market_data.massive", SourceID: "source-massive", Dataset: "equity_eod_prices", ObservationTime: start, NormalizedPayload: payload, MetadataJSON: []byte(`{}`)}}}
	_, err := Run(context.Background(), repo, Config{RunID: "eval-registry", TenantID: "tenant-local", UniverseGroup: "top50_megacap", Symbols: []string{"AAPL"}, AlgorithmIDs: []string{"signalops.algorithms.zscore_anomaly_v1"}, WindowStart: start, WindowEnd: start.AddDate(0, 0, 1), AsOf: start, RegistryEnforcement: true})
	if err == nil {
		t.Fatal("expected registry enforcement to reject an event without provenance")
	}
}

package hypothesisbacktest

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type fakeRepository struct {
	evaluations        []storage.MarketOpsHypothesisEvaluationRecord
	outcomes           []storage.MarketOpsSignalOutcomeRecord
	observations       []storage.MarketOpsFeatureObservationRecord
	evaluationFilters  []storage.MarketOpsHypothesisEvaluationFilter
	outcomeFilters     []storage.MarketOpsSignalOutcomeFilter
	observationFilters []storage.MarketOpsFeatureObservationFilter
	created            []storage.MarketOpsBacktestRunRecord
	completed          int
	failed             int
	summaries          []storage.MarketOpsBacktestCalibrationSummaryRecord
}

func (f *fakeRepository) ListMarketOpsHypothesisEvaluations(_ context.Context, filter storage.MarketOpsHypothesisEvaluationFilter) ([]storage.MarketOpsHypothesisEvaluationRecord, error) {
	f.evaluationFilters = append(f.evaluationFilters, filter)
	out := []storage.MarketOpsHypothesisEvaluationRecord{}
	for _, record := range f.evaluations {
		if record.HypothesisVersion == filter.HypothesisVersion && record.Symbol == filter.Symbol {
			out = append(out, record)
		}
	}
	return out, nil
}

func (f *fakeRepository) ListMarketOpsSignalOutcomes(_ context.Context, filter storage.MarketOpsSignalOutcomeFilter) ([]storage.MarketOpsSignalOutcomeRecord, error) {
	f.outcomeFilters = append(f.outcomeFilters, filter)
	out := []storage.MarketOpsSignalOutcomeRecord{}
	for _, record := range f.outcomes {
		if record.HypothesisVersion == filter.HypothesisVersion && record.Symbol == filter.Symbol {
			out = append(out, record)
		}
	}
	return out, nil
}

func (f *fakeRepository) ListMarketOpsFeatureObservations(_ context.Context, filter storage.MarketOpsFeatureObservationFilter) ([]storage.MarketOpsFeatureObservationRecord, error) {
	f.observationFilters = append(f.observationFilters, filter)
	out := []storage.MarketOpsFeatureObservationRecord{}
	for _, record := range f.observations {
		if record.FeatureKey == filter.FeatureKey && record.Symbol == filter.Symbol {
			out = append(out, record)
		}
	}
	return out, nil
}

func (f *fakeRepository) CreateMarketOpsBacktestRun(_ context.Context, record storage.MarketOpsBacktestRunRecord) error {
	f.created = append(f.created, record)
	return nil
}

func (f *fakeRepository) CompleteMarketOpsBacktestRun(_ context.Context, runID string, completedAt time.Time, metrics []byte) (storage.MarketOpsBacktestRunRecord, error) {
	f.completed++
	record := f.created[len(f.created)-1]
	record.RunID, record.Status, record.CompletedAt, record.MetricsJSON = runID, storage.RunStatusSucceeded, &completedAt, metrics
	return record, nil
}

func (f *fakeRepository) FailMarketOpsBacktestRun(_ context.Context, runID string, failedAt time.Time, message string, metrics []byte) (storage.MarketOpsBacktestRunRecord, error) {
	f.failed++
	return storage.MarketOpsBacktestRunRecord{RunID: runID, Status: storage.RunStatusFailed, CompletedAt: &failedAt, ErrorMessage: message, MetricsJSON: metrics}, nil
}

func (f *fakeRepository) UpsertMarketOpsBacktestCalibrationSummary(_ context.Context, record storage.MarketOpsBacktestCalibrationSummaryRecord) error {
	f.summaries = append(f.summaries, record)
	return nil
}

func TestRunDryRunUsesExactVersionQueriesAndWritesNothing(t *testing.T) {
	start := testDay(1)
	v1 := testEvaluation("eval-v1", "v1", start, .7)
	v2 := testEvaluation("eval-v2", "v2", start, .8)
	repo := &fakeRepository{
		evaluations: []storage.MarketOpsHypothesisEvaluationRecord{v1, v2},
		outcomes: []storage.MarketOpsSignalOutcomeRecord{
			testOutcome("out-v1", v1, 1, .01, true, start.AddDate(0, 0, 1)),
			testOutcome("out-v2", v2, 1, .02, true, start.AddDate(0, 0, 1)),
		},
	}
	cfg := testRunConfig(ModeComparison, []string{"v1", "v2"}, start, start)
	cfg.DryRun = true
	result, err := Run(context.Background(), repo, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.created) != 0 || repo.completed != 0 || len(repo.summaries) != 0 {
		t.Fatal("dry run wrote to isolated backtest ledgers")
	}
	if result.Run.Status != "dry_run" {
		t.Fatalf("unexpected dry-run status %s", result.Run.Status)
	}
	if len(repo.evaluationFilters) != 2 || repo.evaluationFilters[0].HypothesisVersion != "v1" || repo.evaluationFilters[1].HypothesisVersion != "v2" {
		t.Fatalf("queries were not version isolated: %+v", repo.evaluationFilters)
	}
	for _, filter := range repo.outcomeFilters {
		if filter.SourceType != storage.MarketOpsOutcomeSourceHypothesisEvaluation || filter.HypothesisKey != "H001" || filter.CalculationVersion != DefaultOutcomeCalculationVersion {
			t.Fatalf("unexpected outcome filter: %+v", filter)
		}
	}
}

func TestRunPersistsExistingBacktestAndCalibrationRecords(t *testing.T) {
	start := testDay(1)
	evaluation := testEvaluation("eval-v1", "v1", start, .8)
	repo := &fakeRepository{
		evaluations: []storage.MarketOpsHypothesisEvaluationRecord{evaluation},
		outcomes: []storage.MarketOpsSignalOutcomeRecord{
			testOutcome("out-v1", evaluation, 1, .03, true, start.AddDate(0, 0, 1)),
		},
	}
	result, err := Run(context.Background(), repo, testRunConfig(ModeSingle, []string{"v1"}, start, start))
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.created) != 1 || repo.completed != 1 || repo.failed != 0 || len(repo.summaries) != 1 {
		t.Fatalf("unexpected persistence counts: created=%d complete=%d failed=%d summaries=%d", len(repo.created), repo.completed, repo.failed, len(repo.summaries))
	}
	if result.Run.Status != storage.RunStatusSucceeded {
		t.Fatalf("unexpected run status %s", result.Run.Status)
	}
	if !repo.created[0].WindowEnd.Equal(start.AddDate(0, 0, 1)) {
		t.Fatalf("isolated ledger did not convert inclusive window to exclusive end: %s", repo.created[0].WindowEnd)
	}
	summary := repo.summaries[0]
	if summary.DetectorID != "marketops.hypothesis.h001" || summary.RunIDs[0] != "run-g145" || summary.Artifacts != 1 {
		t.Fatalf("unexpected adapted summary: %+v", summary)
	}
	var report Report
	if err := json.Unmarshal(summary.ParametersJSON, &report); err != nil {
		t.Fatal(err)
	}
	if report.SummaryVersion != SummaryVersion || report.PromotionAllowed || report.HypothesisVersions[0] != "v1" {
		t.Fatalf("unexpected persisted report: %+v", report)
	}
}

func testRunConfig(mode string, versions []string, start, end time.Time) RunConfig {
	return RunConfig{
		Config: testConfig(mode, versions, start, end),
		RunID:  "run-g145", SummaryID: "summary-g145", RequestedBy: "operator-test", QueryLimit: 100,
	}
}

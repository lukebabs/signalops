package hypothesisbacktest

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestBuildSingleProducesPointInTimeSegmentedCalibration(t *testing.T) {
	start := testDay(1)
	evaluations := []storage.MarketOpsHypothesisEvaluationRecord{
		testEvaluation("eval-1", "v1", start, .8),
		testEvaluation("eval-2", "v1", start.AddDate(0, 0, 1), .6),
	}
	outcomes := []storage.MarketOpsSignalOutcomeRecord{
		testOutcome("out-1", evaluations[0], 1, .10, true, start.AddDate(0, 0, 2)),
		testOutcome("out-2", evaluations[1], 1, -.05, false, start.AddDate(0, 0, 3)),
	}
	pre, outside, contango, backwardation := "pre_earnings", "outside_window", "contango", "backwardation"
	observations := []storage.MarketOpsFeatureObservationRecord{
		testTextObservation("event-1", evaluations[0], "earnings_window_state", pre, evaluations[0].AsOfTime),
		testTextObservation("curve-1", evaluations[0], "term_structure_state", contango, evaluations[0].AsOfTime),
		testTextObservation("event-2", evaluations[1], "earnings_window_state", outside, evaluations[1].AsOfTime),
		testTextObservation("curve-2", evaluations[1], "term_structure_state", backwardation, evaluations[1].AsOfTime),
		// A later revision must not leak into the earlier evaluation's segment.
		testTextObservation("event-future", evaluations[1], "earnings_window_state", pre, evaluations[1].AsOfTime.Add(time.Hour)),
	}
	report, err := Build(testConfig(ModeSingle, []string{"v1"}, start, start.AddDate(0, 0, 1)), Input{
		Evaluations: evaluations, Outcomes: outcomes, Observations: observations,
	})
	if err != nil {
		t.Fatal(err)
	}
	overall := report.Versions["v1"].Overall
	if overall.Evaluations != 2 || overall.EligibleStates != 2 || overall.Triggers != 2 || overall.IndependentSamples != 2 || overall.MaturedOutcomeSamples != 2 {
		t.Fatalf("unexpected overall counts: %+v", overall)
	}
	assertNear(t, overall.DirectionalHitRate, .5)
	assertNear(t, overall.MeanForwardReturn, .025)
	assertNear(t, overall.MedianForwardReturn, .025)
	assertNear(t, overall.CalibrationError, .2)
	if !overall.BelowMinimumSampleSize || len(report.Warnings) != 1 {
		t.Fatalf("expected explicit sample warning: %+v", report.Warnings)
	}
	if report.PromotionAllowed {
		t.Fatal("calibration reports must never promote hypotheses")
	}
	if report.Versions["v1"].ByEarningsWindow["pre_earnings"].MaturedOutcomeSamples != 1 ||
		report.Versions["v1"].ByEarningsWindow["outside_earnings"].MaturedOutcomeSamples != 1 {
		t.Fatalf("unexpected event segmentation: %+v", report.Versions["v1"].ByEarningsWindow)
	}
	if report.Versions["v1"].ByVolatilityRegime["contango"].MaturedOutcomeSamples != 1 ||
		report.Versions["v1"].ByVolatilityRegime["backwardation"].MaturedOutcomeSamples != 1 {
		t.Fatalf("unexpected volatility segmentation: %+v", report.Versions["v1"].ByVolatilityRegime)
	}
}

func TestBuildRejectsCrossVersionContamination(t *testing.T) {
	start := testDay(1)
	input := Input{Evaluations: []storage.MarketOpsHypothesisEvaluationRecord{
		testEvaluation("eval-v2", "v2", start, .8),
	}}
	_, err := Build(testConfig(ModeSingle, []string{"v1"}, start, start), input)
	if err == nil || !strings.Contains(err.Error(), "version isolation") {
		t.Fatalf("expected version isolation error, got %v", err)
	}
}

func TestBuildComparisonKeepsVersionsIndependent(t *testing.T) {
	start := testDay(1)
	v1 := testEvaluation("eval-v1", "v1", start, .8)
	v2 := testEvaluation("eval-v2", "v2", start, .8)
	input := Input{
		Evaluations: []storage.MarketOpsHypothesisEvaluationRecord{v1, v2},
		Outcomes: []storage.MarketOpsSignalOutcomeRecord{
			testOutcome("out-v1", v1, 1, -.02, false, start.AddDate(0, 0, 1)),
			testOutcome("out-v2", v2, 1, .03, true, start.AddDate(0, 0, 1)),
		},
	}
	report, err := Build(testConfig(ModeComparison, []string{"v1", "v2"}, start, start), input)
	if err != nil {
		t.Fatal(err)
	}
	if report.Comparison == nil || !report.Comparison.AdvisoryOnly {
		t.Fatalf("missing advisory comparison: %+v", report.Comparison)
	}
	assertNear(t, report.Comparison.DirectionalHitDelta, 1)
	assertNear(t, report.Comparison.MeanReturnDelta, .05)
	if report.Versions["v1"].Overall.MaturedOutcomeSamples != 1 || report.Versions["v2"].Overall.MaturedOutcomeSamples != 1 {
		t.Fatal("versions were not summarized independently")
	}
}

func TestBuildWalkForwardUsesChronologicalNonOverlappingTests(t *testing.T) {
	start := testDay(1)
	input := Input{}
	for index := 0; index < 6; index++ {
		session := start.AddDate(0, 0, index)
		evaluation := testEvaluation("eval-"+timeKey(session), "v1", session, .7)
		input.Evaluations = append(input.Evaluations, evaluation)
		input.Outcomes = append(input.Outcomes, testOutcome("out-"+timeKey(session), evaluation, 1, .01, true, session.AddDate(0, 0, 1)))
	}
	cfg := testConfig(ModeWalkForward, []string{"v1"}, start, start.AddDate(0, 0, 5))
	cfg.TrainSessions, cfg.TestSessions, cfg.MaximumFolds = 2, 2, 2
	report, err := Build(cfg, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.WalkForward) != 2 {
		t.Fatalf("expected two folds, got %d", len(report.WalkForward))
	}
	first, second := report.WalkForward[0], report.WalkForward[1]
	if first.TrainStart != "2026-01-01" || first.TrainEnd != "2026-01-02" ||
		first.TestStart != "2026-01-03" || first.TestEnd != "2026-01-04" {
		t.Fatalf("unexpected first fold: %+v", first)
	}
	if second.TrainEnd != "2026-01-04" || second.TestStart != "2026-01-05" || second.TestEnd != "2026-01-06" {
		t.Fatalf("unexpected expanding fold: %+v", second)
	}
}

func TestBuildExcludesOutcomesMaturingAfterAsOf(t *testing.T) {
	start := testDay(1)
	evaluation := testEvaluation("eval-1", "v1", start, .8)
	outcome := testOutcome("out-1", evaluation, 20, .2, true, start.AddDate(0, 0, 20))
	cfg := testConfig(ModeSingle, []string{"v1"}, start, start)
	cfg.AsOf = start.AddDate(0, 0, 10)
	report, err := Build(cfg, Input{Evaluations: []storage.MarketOpsHypothesisEvaluationRecord{evaluation}, Outcomes: []storage.MarketOpsSignalOutcomeRecord{outcome}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Versions["v1"].Overall.MaturedOutcomeSamples != 0 {
		t.Fatal("future-maturing outcome leaked into point-in-time report")
	}
	if report.Versions["v1"].ByEarningsWindow["unknown"].Evaluations != 1 {
		t.Fatal("sparse evaluation disappeared from unknown event segment")
	}
}

func TestBuildSampleThresholdDoesNotCountRepeatedHorizonsAsIndependent(t *testing.T) {
	start := testDay(1)
	evaluation := testEvaluation("eval-1", "v1", start, .8)
	outcomes := []storage.MarketOpsSignalOutcomeRecord{}
	for _, horizon := range []int{1, 5, 10, 20} {
		outcomes = append(outcomes, testOutcome("out-"+time.Duration(horizon).String(), evaluation, horizon, .02, true, start.AddDate(0, 0, horizon)))
	}
	cfg := testConfig(ModeSingle, []string{"v1"}, start, start)
	cfg.MinimumSampleSize = 2
	report, err := Build(cfg, Input{Evaluations: []storage.MarketOpsHypothesisEvaluationRecord{evaluation}, Outcomes: outcomes})
	if err != nil {
		t.Fatal(err)
	}
	overall := report.Versions["v1"].Overall
	if overall.IndependentSamples != 1 || overall.MaturedOutcomeSamples != 4 || !overall.BelowMinimumSampleSize {
		t.Fatalf("horizons inflated independent sample size: %+v", overall)
	}
}

func testConfig(mode string, versions []string, start, end time.Time) Config {
	return Config{
		TenantID: "tenant-test", HypothesisKey: "H001", HypothesisVersions: versions,
		Symbols: []string{"AAPL"}, WindowStart: start, WindowEnd: end,
		AsOf: end.AddDate(0, 0, 30), Mode: mode, MinimumSampleSize: 3,
		TrainSessions: 2, TestSessions: 2, MaximumFolds: 3,
	}
}

func testEvaluation(id, version string, session time.Time, confidence float64) storage.MarketOpsHypothesisEvaluationRecord {
	return storage.MarketOpsHypothesisEvaluationRecord{
		EvaluationID: id, TenantID: "tenant-test", AppID: "marketops",
		HypothesisKey: "H001", HypothesisVersion: version, Symbol: "AAPL",
		SessionDate: session, AsOfTime: session.Add(16 * time.Hour), Eligible: true,
		Triggered: true, ConfidenceScore: &confidence,
	}
}

func testOutcome(id string, evaluation storage.MarketOpsHypothesisEvaluationRecord, horizon int, forwardReturn float64, hit bool, matured time.Time) storage.MarketOpsSignalOutcomeRecord {
	mfe, mae, drawdown, vol := .12, -.04, -.03, .02
	return storage.MarketOpsSignalOutcomeRecord{
		OutcomeID: id, TenantID: evaluation.TenantID, AppID: "marketops",
		SourceType: storage.MarketOpsOutcomeSourceHypothesisEvaluation, SourceID: evaluation.EvaluationID,
		HypothesisKey: evaluation.HypothesisKey, HypothesisVersion: evaluation.HypothesisVersion,
		Symbol: evaluation.Symbol, OriginSessionDate: evaluation.SessionDate, HorizonSessions: horizon,
		MaturedSessionDate: &matured, OutcomeStatus: storage.MarketOpsOutcomeMatured,
		ForwardReturn: &forwardReturn, MaxFavorableExcursion: &mfe, MaxAdverseExcursion: &mae,
		MaximumDrawdown: &drawdown, RealizedVolChange: &vol, DirectionalHit: &hit,
		CalculationVersion: DefaultOutcomeCalculationVersion,
	}
}
func testTextObservation(id string, evaluation storage.MarketOpsHypothesisEvaluationRecord, key, value string, asOf time.Time) storage.MarketOpsFeatureObservationRecord {
	return storage.MarketOpsFeatureObservationRecord{
		FeatureObservationID: id, TenantID: evaluation.TenantID, AppID: "marketops",
		Symbol: evaluation.Symbol, SessionDate: evaluation.SessionDate, AsOfTime: asOf,
		FeatureKey: key, TextValue: &value, QualityState: storage.MarketOpsQualityUsable,
	}
}

func assertNear(t *testing.T, actual *float64, expected float64) {
	t.Helper()
	if actual == nil || math.Abs(*actual-expected) > 1e-9 {
		t.Fatalf("got %v, want %v", actual, expected)
	}
}

func testDay(day int) time.Time {
	return time.Date(2026, 1, day, 0, 0, 0, 0, time.UTC)
}

func timeKey(value time.Time) string {
	return value.Format("20060102")
}

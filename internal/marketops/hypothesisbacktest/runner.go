package hypothesisbacktest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type Repository interface {
	ListMarketOpsHypothesisEvaluations(context.Context, storage.MarketOpsHypothesisEvaluationFilter) ([]storage.MarketOpsHypothesisEvaluationRecord, error)
	ListMarketOpsSignalOutcomes(context.Context, storage.MarketOpsSignalOutcomeFilter) ([]storage.MarketOpsSignalOutcomeRecord, error)
	ListMarketOpsFeatureObservations(context.Context, storage.MarketOpsFeatureObservationFilter) ([]storage.MarketOpsFeatureObservationRecord, error)
	CreateMarketOpsBacktestRun(context.Context, storage.MarketOpsBacktestRunRecord) error
	CompleteMarketOpsBacktestRun(context.Context, string, time.Time, []byte) (storage.MarketOpsBacktestRunRecord, error)
	FailMarketOpsBacktestRun(context.Context, string, time.Time, string, []byte) (storage.MarketOpsBacktestRunRecord, error)
	UpsertMarketOpsBacktestCalibrationSummary(context.Context, storage.MarketOpsBacktestCalibrationSummaryRecord) error
}

type RunConfig struct {
	Config
	RunID       string
	SummaryID   string
	RequestedBy string
	QueryLimit  int
	DryRun      bool
}

type RunMetrics struct {
	Scanned              int            `json:"scanned"`
	Signals              int            `json:"signals"`
	Artifacts            int            `json:"artifacts"`
	GraphProposals       int            `json:"graph_proposals"`
	PolicyResults        int            `json:"policy_results"`
	RecommendationCounts map[string]int `json:"recommendation_counts"`
	HypothesisKey        string         `json:"hypothesis_key"`
	HypothesisVersions   []string       `json:"hypothesis_versions"`
	Mode                 string         `json:"mode"`
	MaturedOutcomes      int            `json:"matured_outcomes"`
	Warnings             []string       `json:"warnings"`
	SummaryID            string         `json:"summary_id"`
}

type Result struct {
	Run     storage.MarketOpsBacktestRunRecord                `json:"backtest_run"`
	Summary storage.MarketOpsBacktestCalibrationSummaryRecord `json:"calibration_summary"`
	Report  Report                                            `json:"report"`
	Metrics RunMetrics                                        `json:"metrics"`
	DryRun  bool                                              `json:"dry_run"`
}

func Run(ctx context.Context, repo Repository, cfg RunConfig) (Result, error) {
	cfg = normalizeRunConfig(cfg)
	if err := validateRunConfig(cfg); err != nil {
		return Result{}, err
	}
	if repo == nil {
		return Result{}, errors.New("G145 repository is required")
	}
	runRecord := newRunRecord(cfg)
	result := Result{Run: runRecord, DryRun: cfg.DryRun}
	if !cfg.DryRun {
		if err := repo.CreateMarketOpsBacktestRun(ctx, runRecord); err != nil {
			return result, err
		}
	}
	input, err := loadInput(ctx, repo, cfg)
	if err != nil {
		return failRun(repo, cfg, result, err)
	}
	report, err := Build(cfg.Config, input)
	if err != nil {
		return failRun(repo, cfg, result, err)
	}
	result.Report = report
	result.Metrics = metricsFromReport(report)
	result.Metrics.SummaryID = cfg.SummaryID
	result.Summary = summaryRecord(cfg, report, result.Metrics)
	if cfg.DryRun {
		result.Run.Status = "dry_run"
		return result, nil
	}
	metricsJSON, err := json.Marshal(result.Metrics)
	if err != nil {
		return failRun(repo, cfg, result, err)
	}
	completed, err := repo.CompleteMarketOpsBacktestRun(ctx, cfg.RunID, time.Now().UTC(), metricsJSON)
	if err != nil {
		return result, err
	}
	result.Run = completed
	if err := repo.UpsertMarketOpsBacktestCalibrationSummary(ctx, result.Summary); err != nil {
		return failRun(repo, cfg, result, err)
	}
	return result, nil
}

func normalizeRunConfig(cfg RunConfig) RunConfig {
	cfg.Config = normalize(cfg.Config)
	cfg.RunID = strings.TrimSpace(cfg.RunID)
	cfg.SummaryID = strings.TrimSpace(cfg.SummaryID)
	cfg.RequestedBy = strings.TrimSpace(cfg.RequestedBy)
	if cfg.RequestedBy == "" {
		cfg.RequestedBy = "operator-local"
	}
	if cfg.QueryLimit == 0 {
		cfg.QueryLimit = 1000
	}
	return cfg
}

func validateRunConfig(cfg RunConfig) error {
	if err := validate(cfg.Config); err != nil {
		return err
	}
	if cfg.RunID == "" || cfg.SummaryID == "" {
		return errors.New("G145 run id and summary id are required")
	}
	if cfg.QueryLimit < 1 || cfg.QueryLimit > 1000 {
		return errors.New("G145 query limit must be between 1 and 1000")
	}
	return nil
}

func loadInput(ctx context.Context, repo Repository, cfg RunConfig) (Input, error) {
	input := Input{}
	evaluationIDs := map[string]bool{}
	outcomeIDs := map[string]bool{}
	observationIDs := map[string]bool{}
	for _, version := range cfg.HypothesisVersions {
		for _, symbol := range cfg.Symbols {
			evaluations, err := repo.ListMarketOpsHypothesisEvaluations(ctx, storage.MarketOpsHypothesisEvaluationFilter{
				TenantID: cfg.TenantID, AppID: "marketops", HypothesisKey: cfg.HypothesisKey,
				HypothesisVersion: version, Symbol: symbol, SessionStart: cfg.WindowStart,
				SessionEnd: cfg.WindowEnd, Limit: cfg.QueryLimit,
			})
			if err != nil {
				return input, err
			}
			if len(evaluations) == cfg.QueryLimit {
				return input, fmt.Errorf("G145 evaluation query limit reached for %s %s", version, symbol)
			}
			for _, record := range evaluations {
				if !evaluationIDs[record.EvaluationID] {
					evaluationIDs[record.EvaluationID] = true
					input.Evaluations = append(input.Evaluations, record)
				}
			}
			outcomes, err := repo.ListMarketOpsSignalOutcomes(ctx, storage.MarketOpsSignalOutcomeFilter{
				TenantID: cfg.TenantID, AppID: "marketops",
				SourceType:    storage.MarketOpsOutcomeSourceHypothesisEvaluation,
				HypothesisKey: cfg.HypothesisKey, HypothesisVersion: version, Symbol: symbol,
				OriginStart: cfg.WindowStart, OriginEnd: cfg.WindowEnd, Limit: cfg.QueryLimit,
				CalculationVersion: cfg.OutcomeCalculationVersion,
			})
			if err != nil {
				return input, err
			}
			if len(outcomes) == cfg.QueryLimit {
				return input, fmt.Errorf("G145 outcome query limit reached for %s %s", version, symbol)
			}
			for _, record := range outcomes {
				if !outcomeIDs[record.OutcomeID] {
					outcomeIDs[record.OutcomeID] = true
					input.Outcomes = append(input.Outcomes, record)
				}
			}
		}
	}
	for _, symbol := range cfg.Symbols {
		for _, featureKey := range []string{"earnings_window_state", "term_structure_state"} {
			observations, err := repo.ListMarketOpsFeatureObservations(ctx, storage.MarketOpsFeatureObservationFilter{
				TenantID: cfg.TenantID, AppID: "marketops", Symbol: symbol,
				FeatureKey: featureKey, SessionStart: cfg.WindowStart,
				SessionEnd: cfg.WindowEnd, Limit: cfg.QueryLimit,
			})
			if err != nil {
				return input, err
			}
			if len(observations) == cfg.QueryLimit {
				return input, fmt.Errorf("G145 observation query limit reached for %s %s", featureKey, symbol)
			}
			for _, record := range observations {
				if !observationIDs[record.FeatureObservationID] {
					observationIDs[record.FeatureObservationID] = true
					input.Observations = append(input.Observations, record)
				}
			}
		}
	}
	return input, nil
}

func newRunRecord(cfg RunConfig) storage.MarketOpsBacktestRunRecord {
	filters, _ := json.Marshal(map[string]any{
		"hypothesis_key": cfg.HypothesisKey, "hypothesis_versions": cfg.HypothesisVersions,
		"symbols": cfg.Symbols, "as_of": dateString(cfg.AsOf),
	})
	parameters, _ := json.Marshal(map[string]any{
		"summary_version": SummaryVersion, "mode": cfg.Mode,
		"minimum_sample_size": cfg.MinimumSampleSize, "train_sessions": cfg.TrainSessions,
		"outcome_calculation_version": cfg.OutcomeCalculationVersion,
		"test_sessions":               cfg.TestSessions, "maximum_folds": cfg.MaximumFolds,
		"promotion_allowed": false,
	})
	return storage.MarketOpsBacktestRunRecord{
		RunID: cfg.RunID, TenantID: cfg.TenantID, AppID: "marketops", Domain: "market_data",
		UseCase: "daily_market_surveillance", SourceID: "marketops.research_ledgers",
		SourceAdapter: "marketops.hypothesis_backtest", Dataset: "hypothesis_evaluations",
		DetectorID:      "marketops.hypothesis." + strings.ToLower(cfg.HypothesisKey),
		DetectorVersion: strings.Join(cfg.HypothesisVersions, ","), Status: storage.RunStatusStarted,
		RequestedBy: cfg.RequestedBy, WindowStart: cfg.WindowStart, WindowEnd: cfg.WindowEnd.AddDate(0, 0, 1),
		StartedAt: time.Now().UTC(), FiltersJSON: filters, ParametersJSON: parameters,
		MetricsJSON: []byte(`{}`),
	}
}

func metricsFromReport(report Report) RunMetrics {
	metrics := RunMetrics{
		RecommendationCounts: map[string]int{}, HypothesisKey: report.HypothesisKey,
		HypothesisVersions: append([]string(nil), report.HypothesisVersions...),
		Mode:               report.Mode, Warnings: append([]string(nil), report.Warnings...),
	}
	for _, version := range report.HypothesisVersions {
		item := report.Versions[version].Overall
		metrics.Scanned += item.Evaluations
		metrics.Signals += item.Triggers
		metrics.Artifacts += item.MaturedOutcomeSamples
		metrics.MaturedOutcomes += item.MaturedOutcomeSamples
	}
	return metrics
}

func summaryRecord(cfg RunConfig, report Report, metrics RunMetrics) storage.MarketOpsBacktestCalibrationSummaryRecord {
	parameters, _ := json.Marshal(report)
	filters, _ := json.Marshal(map[string]any{
		"hypothesis_key": cfg.HypothesisKey, "hypothesis_versions": cfg.HypothesisVersions,
		"symbols": cfg.Symbols, "window_start": dateString(cfg.WindowStart),
		"outcome_calculation_version": cfg.OutcomeCalculationVersion,
		"window_end":                  dateString(cfg.WindowEnd), "as_of": dateString(cfg.AsOf),
	})
	signalYield := 0.0
	if metrics.Scanned > 0 {
		signalYield = float64(metrics.Signals) / float64(metrics.Scanned)
	}
	return storage.MarketOpsBacktestCalibrationSummaryRecord{
		SummaryID: cfg.SummaryID, TenantID: cfg.TenantID, AppID: "marketops",
		Domain: "market_data", UseCase: "daily_market_surveillance",
		SourceID: "marketops.research_ledgers", Dataset: "hypothesis_evaluations",
		DetectorID:   "marketops.hypothesis." + strings.ToLower(cfg.HypothesisKey),
		StatusFilter: storage.RunStatusSucceeded, RequestedBy: cfg.RequestedBy,
		RunIDs: []string{cfg.RunID}, RunCount: 1, SucceededCount: 1,
		ZeroInputCount: boolInt(metrics.Scanned == 0), Scanned: metrics.Scanned,
		Signals: metrics.Signals, Artifacts: metrics.Artifacts, SignalYield: signalYield,
		RecommendationCounts: []byte(`{}`), RecommendationShares: []byte(`{}`),
		DominantRecommendation: []byte(`{}`), FiltersJSON: filters, ParametersJSON: parameters,
	}
}

func failRun(repo Repository, cfg RunConfig, result Result, runErr error) (Result, error) {
	if cfg.DryRun {
		return result, runErr
	}
	metricsJSON, _ := json.Marshal(result.Metrics)
	failed, failErr := repo.FailMarketOpsBacktestRun(context.Background(), cfg.RunID, time.Now().UTC(), runErr.Error(), metricsJSON)
	if failErr == nil {
		result.Run = failed
	}
	return result, runErr
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

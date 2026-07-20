package history

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/marketops/hypotheses"
	"github.com/lukebabs/signalops/internal/marketops/opportunities"
	"github.com/lukebabs/signalops/internal/marketops/outcomes"
	marketopsstate "github.com/lukebabs/signalops/internal/marketops/state"
	"github.com/lukebabs/signalops/internal/storage"
)

const (
	DefaultMinimumEquitySessions  = 60
	DefaultMinimumOptionsSessions = 20
	DefaultMaximumSessions        = 120
)

type Repository interface {
	ListMarketOpsBacktestNormalizedEvents(context.Context, storage.MarketOpsBacktestEventFilter) ([]storage.NormalizedEventLedgerRecord, error)
	ListMarketOpsOptionsDistributions(context.Context, storage.MarketOpsOptionsDistributionFilter) ([]storage.MarketOpsOptionsDistributionRecord, error)
	ListMarketOpsOptionsChain(context.Context, storage.MarketOpsOptionsChainFilter) ([]storage.MarketOpsOptionsChainRecord, error)
	UpsertMarketOpsFeatureDefinition(context.Context, storage.MarketOpsFeatureDefinitionRecord) error
	UpsertMarketOpsFeatureObservation(context.Context, storage.MarketOpsFeatureObservationRecord) error
	UpsertMarketOpsMarketState(context.Context, storage.MarketOpsMarketStateRecord) error
	UpsertMarketOpsStateTransition(context.Context, storage.MarketOpsStateTransitionRecord) error
	UpsertMarketOpsEvidence(context.Context, storage.MarketOpsEvidenceRecord) error
	UpsertMarketOpsHypothesisDefinition(context.Context, storage.MarketOpsHypothesisDefinitionRecord) error
	UpsertMarketOpsHypothesisEvaluation(context.Context, storage.MarketOpsHypothesisEvaluationRecord) error
	UpsertMarketOpsOpportunity(context.Context, storage.MarketOpsOpportunityRecord) error
	UpsertMarketOpsSignalOutcome(context.Context, storage.MarketOpsSignalOutcomeRecord) error
}

type Config struct {
	TenantID                  string
	Symbol                    string
	AssetID                   string
	RunID                     string
	SessionStart              time.Time
	SessionEnd                time.Time
	AsOf                      time.Time
	MaxSessions               int
	MinimumEquitySessions     int
	MinimumOptionsSessions    int
	OutcomeThreshold          float64
	AllowInsufficientCoverage bool
	DryRun                    bool
}

type Coverage struct {
	EquityEvents             int      `json:"equity_events"`
	EquitySourceSessions     int      `json:"equity_source_sessions"`
	EquityForwardSessions    int      `json:"equity_forward_sessions"`
	OptionContracts          int      `json:"option_contracts"`
	OptionChainSessions      int      `json:"option_chain_sessions"`
	AnalyticsReadyOptionDays int      `json:"analytics_ready_option_sessions"`
	OptionDistributionDays   int      `json:"option_distribution_sessions"`
	MinimumEquitySessions    int      `json:"minimum_equity_sessions"`
	MinimumOptionsSessions   int      `json:"minimum_options_sessions"`
	Ready                    bool     `json:"ready"`
	BlockingReasons          []string `json:"blocking_reasons"`
	Warnings                 []string `json:"warnings"`
}

type Metrics struct {
	RunID                 string         `json:"run_id"`
	TenantID              string         `json:"tenant_id"`
	Symbol                string         `json:"symbol"`
	SessionStart          string         `json:"session_start"`
	SessionEnd            string         `json:"session_end"`
	AsOf                  string         `json:"as_of"`
	Status                string         `json:"status"`
	Coverage              Coverage       `json:"coverage"`
	FeatureDefinitions    int            `json:"feature_definitions"`
	FeatureObservations   int            `json:"feature_observations"`
	MarketStates          int            `json:"market_states"`
	StateTransitions      int            `json:"state_transitions"`
	Evidence              int            `json:"evidence"`
	HypothesisDefinitions int            `json:"hypothesis_definitions"`
	HypothesisEvaluations int            `json:"hypothesis_evaluations"`
	EligibleEvaluations   int            `json:"eligible_evaluations"`
	TriggeredEvaluations  int            `json:"triggered_evaluations"`
	Opportunities         int            `json:"opportunities"`
	Outcomes              int            `json:"outcomes"`
	MaturedOutcomes       int            `json:"matured_outcomes"`
	PendingOutcomes       int            `json:"pending_outcomes"`
	MissingPriceOutcomes  int            `json:"missing_price_outcomes"`
	EvaluationReasons     map[string]int `json:"evaluation_reason_counts"`
	OutcomeSkipReasons    map[string]int `json:"outcome_skip_reasons"`
	DryRun                bool           `json:"dry_run"`
	StartedAt             string         `json:"started_at"`
	CompletedAt           string         `json:"completed_at"`
}

func Run(ctx context.Context, repo Repository, cfg Config) (Metrics, error) {
	cfg = normalize(cfg)
	if err := validate(cfg); err != nil {
		return Metrics{}, err
	}
	if repo == nil {
		return Metrics{}, errors.New("G141 repository is required")
	}
	started := time.Now().UTC()
	result := Metrics{
		RunID: cfg.RunID, TenantID: cfg.TenantID, Symbol: cfg.Symbol,
		SessionStart: cfg.SessionStart.Format("2006-01-02"), SessionEnd: cfg.SessionEnd.Format("2006-01-02"),
		AsOf: cfg.AsOf.Format("2006-01-02"), Status: "preflight", DryRun: cfg.DryRun,
		EvaluationReasons: map[string]int{}, OutcomeSkipReasons: map[string]int{},
		StartedAt: started.Format(time.RFC3339Nano),
	}

	warmupStart := cfg.SessionStart.AddDate(0, 0, -90)
	events, err := repo.ListMarketOpsBacktestNormalizedEvents(ctx, storage.MarketOpsBacktestEventFilter{
		TenantID: cfg.TenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance",
		SourceAdapter: "market_data.massive", Dataset: "equity_eod_prices", Symbols: []string{cfg.Symbol}, WindowStart: warmupStart,
		WindowEnd: cfg.AsOf.AddDate(0, 0, 1), Limit: 1000,
	})
	if err != nil {
		return result, err
	}
	distributions, err := repo.ListMarketOpsOptionsDistributions(ctx, storage.MarketOpsOptionsDistributionFilter{
		TenantID: cfg.TenantID, Symbol: cfg.Symbol, Limit: 200,
	})
	if err != nil {
		return result, err
	}
	distributions = filterDistributions(distributions, warmupStart, cfg.SessionEnd)
	chain, err := repo.ListMarketOpsOptionsChain(ctx, storage.MarketOpsOptionsChainFilter{
		TenantID: cfg.TenantID, Symbol: cfg.Symbol, WindowStart: warmupStart, WindowEnd: cfg.SessionEnd, Limit: 5000,
	})
	if err != nil {
		return result, err
	}

	result.Coverage = assessCoverage(cfg, events, distributions, chain)
	if !cfg.AllowInsufficientCoverage && !result.Coverage.Ready {
		result.Status = "blocked_insufficient_coverage"
		result.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
		return result, nil
	}

	built, err := marketopsstate.Build(marketopsstate.BuildConfig{
		TenantID: cfg.TenantID, Symbol: cfg.Symbol, AssetID: cfg.AssetID, RunID: cfg.RunID,
		SessionStart: cfg.SessionStart, SessionEnd: cfg.SessionEnd, MaxSessions: cfg.MaxSessions,
	}, marketopsstate.BuildInput{EquityEvents: events, Distributions: distributions, OptionChain: chain})
	if err != nil {
		return result, err
	}
	result.FeatureDefinitions = len(built.Definitions)
	result.FeatureObservations = len(built.Observations)
	result.MarketStates = len(built.States)
	result.StateTransitions = len(built.Transitions)
	result.Evidence = len(built.Evidence)

	definitions := hypotheses.ResearchDefinitions(cfg.TenantID)
	result.HypothesisDefinitions = len(definitions)
	observationsByID := make(map[string]storage.MarketOpsFeatureObservationRecord, len(built.Observations))
	for _, record := range built.Observations {
		observationsByID[record.FeatureObservationID] = record
	}
	transitionsByState := map[string][]storage.MarketOpsStateTransitionRecord{}
	for _, record := range built.Transitions {
		transitionsByState[record.CurrentStateID] = append(transitionsByState[record.CurrentStateID], record)
	}
	evidenceBySession := map[string][]storage.MarketOpsEvidenceRecord{}
	for _, record := range built.Evidence {
		evidenceBySession[dateKey(record.SessionDate)] = append(evidenceBySession[dateKey(record.SessionDate)], record)
	}
	evaluations := make([]storage.MarketOpsHypothesisEvaluationRecord, 0, len(built.States)*len(definitions))
	for _, state := range built.States {
		stateObservations := make([]storage.MarketOpsFeatureObservationRecord, 0, len(state.FeatureObservationIDs))
		for _, id := range state.FeatureObservationIDs {
			if record, ok := observationsByID[id]; ok {
				stateObservations = append(stateObservations, record)
			}
		}
		for _, definition := range definitions {
			record, evaluateErr := hypotheses.Evaluate(cfg.RunID, definition, state, stateObservations, transitionsByState[state.MarketStateID], evidenceBySession[dateKey(state.SessionDate)])
			if evaluateErr != nil {
				return result, evaluateErr
			}
			evaluations = append(evaluations, record)
			if record.Eligible {
				result.EligibleEvaluations++
			}
			if record.Triggered {
				result.TriggeredEvaluations++
			}
			for _, reason := range record.ReasonCodes {
				result.EvaluationReasons[reason]++
			}
		}
	}
	result.HypothesisEvaluations = len(evaluations)

	opportunityResult, err := opportunities.Build(cfg.RunID, definitions, evaluations)
	if err != nil {
		return result, err
	}
	result.Opportunities = len(opportunityResult.Opportunities)

	outcomeResult, err := outcomes.Build(outcomes.BuildConfig{
		TenantID: cfg.TenantID, Symbol: cfg.Symbol, RunID: cfg.RunID, AsOf: cfg.AsOf, Threshold: cfg.OutcomeThreshold,
	}, outcomes.BuildInput{Evaluations: evaluations, Opportunities: opportunityResult.Opportunities, EquityEvents: events})
	if err != nil {
		return result, err
	}
	result.Outcomes = len(outcomeResult.Outcomes)
	result.MaturedOutcomes = outcomeResult.Matured
	result.PendingOutcomes = outcomeResult.Pending
	result.MissingPriceOutcomes = outcomeResult.MissingPrice
	result.OutcomeSkipReasons = outcomeResult.SkippedReasons

	if !cfg.DryRun {
		if err := persist(ctx, repo, built, definitions, evaluations, opportunityResult.Opportunities, outcomeResult.Outcomes); err != nil {
			return result, err
		}
	}
	if result.Coverage.Ready {
		result.Status = "completed"
	} else {
		result.Status = "completed_with_insufficient_coverage"
	}
	result.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
	return result, nil
}

func persist(ctx context.Context, repo Repository, built marketopsstate.BuildResult, definitions []storage.MarketOpsHypothesisDefinitionRecord, evaluations []storage.MarketOpsHypothesisEvaluationRecord, opportunityRows []storage.MarketOpsOpportunityRecord, outcomeRows []storage.MarketOpsSignalOutcomeRecord) error {
	for _, record := range built.Definitions {
		if err := repo.UpsertMarketOpsFeatureDefinition(ctx, record); err != nil {
			return fmt.Errorf("persist feature definition: %w", err)
		}
	}
	for _, record := range built.Observations {
		if err := repo.UpsertMarketOpsFeatureObservation(ctx, record); err != nil {
			return fmt.Errorf("persist feature observation: %w", err)
		}
	}
	for _, record := range built.States {
		if err := repo.UpsertMarketOpsMarketState(ctx, record); err != nil {
			return fmt.Errorf("persist market state: %w", err)
		}
	}
	for _, record := range built.Transitions {
		if err := repo.UpsertMarketOpsStateTransition(ctx, record); err != nil {
			return fmt.Errorf("persist state transition: %w", err)
		}
	}
	for _, record := range built.Evidence {
		if err := repo.UpsertMarketOpsEvidence(ctx, record); err != nil {
			return fmt.Errorf("persist evidence: %w", err)
		}
	}
	for _, record := range definitions {
		if err := repo.UpsertMarketOpsHypothesisDefinition(ctx, record); err != nil {
			return fmt.Errorf("persist hypothesis definition: %w", err)
		}
	}
	for _, record := range evaluations {
		if err := repo.UpsertMarketOpsHypothesisEvaluation(ctx, record); err != nil {
			return fmt.Errorf("persist hypothesis evaluation: %w", err)
		}
	}
	for _, record := range opportunityRows {
		if err := repo.UpsertMarketOpsOpportunity(ctx, record); err != nil {
			return fmt.Errorf("persist opportunity: %w", err)
		}
	}
	for _, record := range outcomeRows {
		if err := repo.UpsertMarketOpsSignalOutcome(ctx, record); err != nil {
			return fmt.Errorf("persist outcome: %w", err)
		}
	}
	return nil
}

func normalize(cfg Config) Config {
	cfg.TenantID = strings.TrimSpace(cfg.TenantID)
	cfg.Symbol = strings.ToUpper(strings.TrimSpace(cfg.Symbol))
	cfg.AssetID = strings.TrimSpace(cfg.AssetID)
	cfg.RunID = strings.TrimSpace(cfg.RunID)
	if cfg.AssetID == "" && cfg.Symbol != "" {
		cfg.AssetID = "ticker:" + cfg.Symbol
	}
	if cfg.MaxSessions == 0 {
		cfg.MaxSessions = DefaultMaximumSessions
	}
	if cfg.MinimumEquitySessions == 0 {
		cfg.MinimumEquitySessions = DefaultMinimumEquitySessions
	}
	if cfg.MinimumOptionsSessions == 0 {
		cfg.MinimumOptionsSessions = DefaultMinimumOptionsSessions
	}
	if cfg.OutcomeThreshold == 0 {
		cfg.OutcomeThreshold = outcomes.DefaultThreshold
	}
	cfg.SessionStart = day(cfg.SessionStart)
	cfg.SessionEnd = day(cfg.SessionEnd)
	cfg.AsOf = day(cfg.AsOf)
	return cfg
}

func validate(cfg Config) error {
	if cfg.TenantID == "" || cfg.Symbol == "" || cfg.AssetID == "" || cfg.RunID == "" {
		return errors.New("G141 tenant-id, symbol, asset-id, and run-id are required")
	}
	if cfg.Symbol != "AAPL" {
		return errors.New("G141 is intentionally bounded to AAPL")
	}
	if cfg.SessionStart.IsZero() || cfg.SessionEnd.IsZero() || !cfg.SessionEnd.After(cfg.SessionStart) {
		return errors.New("G141 session-end must be after session-start")
	}
	if cfg.AsOf.IsZero() || cfg.AsOf.Before(cfg.SessionEnd.AddDate(0, 0, -1)) {
		return errors.New("G141 as-of must be on or after the final source session")
	}
	if cfg.MaxSessions <= 0 || cfg.MaxSessions > 200 {
		return errors.New("G141 max-sessions must be between 1 and 200")
	}
	if cfg.MinimumEquitySessions <= 0 || cfg.MinimumEquitySessions > cfg.MaxSessions {
		return errors.New("G141 minimum-equity-sessions must be between 1 and max-sessions")
	}
	if cfg.MinimumOptionsSessions <= 0 || cfg.MinimumOptionsSessions > cfg.MaxSessions {
		return errors.New("G141 minimum-options-sessions must be between 1 and max-sessions")
	}
	if cfg.OutcomeThreshold <= 0 || cfg.OutcomeThreshold >= 1 {
		return errors.New("G141 outcome-threshold must be between 0 and 1")
	}
	if cfg.AllowInsufficientCoverage && !cfg.DryRun {
		return errors.New("G141 allow-insufficient-coverage is restricted to dry-run diagnostics")
	}
	return nil
}

func assessCoverage(cfg Config, events []storage.NormalizedEventLedgerRecord, distributions []storage.MarketOpsOptionsDistributionRecord, chain []storage.MarketOpsOptionsChainRecord) Coverage {
	sourceEquity := map[string]bool{}
	forwardEquity := map[string]bool{}
	for _, event := range events {
		if event.TenantID != cfg.TenantID || event.Dataset != "equity_eod_prices" {
			continue
		}
		session := day(event.ObservationTime)
		if session.Before(cfg.SessionStart) || session.After(cfg.AsOf) {
			continue
		}
		forwardEquity[dateKey(session)] = true
		if session.Before(cfg.SessionEnd) {
			sourceEquity[dateKey(session)] = true
		}
	}
	chainSessions := map[string]bool{}
	contractsBySession := map[string][]storage.MarketOpsOptionsChainRecord{}
	for _, record := range chain {
		session := day(record.TradeDate)
		if session.Before(cfg.SessionStart) || !session.Before(cfg.SessionEnd) {
			continue
		}
		key := dateKey(session)
		chainSessions[key] = true
		contractsBySession[key] = append(contractsBySession[key], record)
	}
	analyticsDays := 0
	for key, records := range contractsBySession {
		session, _ := time.Parse("2006-01-02", key)
		if hasRequiredSurface(session, records) {
			analyticsDays++
		}
	}
	distributionSessions := map[string]bool{}
	for _, record := range distributions {
		session := day(record.TradeDate)
		if !session.Before(cfg.SessionStart) && session.Before(cfg.SessionEnd) {
			distributionSessions[dateKey(session)] = true
		}
	}
	report := Coverage{
		EquityEvents: len(events), EquitySourceSessions: len(sourceEquity), EquityForwardSessions: len(forwardEquity),
		OptionContracts: len(chain), OptionChainSessions: len(chainSessions), AnalyticsReadyOptionDays: analyticsDays,
		OptionDistributionDays: len(distributionSessions), MinimumEquitySessions: cfg.MinimumEquitySessions,
		MinimumOptionsSessions: cfg.MinimumOptionsSessions,
	}
	if report.EquitySourceSessions < cfg.MinimumEquitySessions {
		report.BlockingReasons = append(report.BlockingReasons, "insufficient_equity_source_sessions")
	}
	if report.AnalyticsReadyOptionDays < cfg.MinimumOptionsSessions {
		report.BlockingReasons = append(report.BlockingReasons, "insufficient_analytics_ready_option_sessions")
	}
	if report.OptionDistributionDays < cfg.MinimumOptionsSessions {
		report.Warnings = append(report.Warnings, "insufficient_option_distribution_sessions_for_positioning_hypotheses")
	}
	if len(chain) == 5000 {
		report.BlockingReasons = append(report.BlockingReasons, "option_chain_query_limit_reached")
	}
	sort.Strings(report.BlockingReasons)
	sort.Strings(report.Warnings)
	report.Ready = len(report.BlockingReasons) == 0
	return report
}

func hasRequiredSurface(session time.Time, records []storage.MarketOpsOptionsChainRecord) bool {
	required := []struct {
		dte        int
		delta      float64
		optionType string
	}{
		{30, .50, ""}, {60, .50, ""}, {90, .50, ""}, {30, .25, "put"}, {30, .25, "call"},
	}
	for _, cell := range required {
		found := false
		for _, record := range records {
			if cell.optionType != "" && strings.ToLower(record.ContractType) != cell.optionType {
				continue
			}
			if record.ImpliedVolatility == nil || *record.ImpliedVolatility <= 0 || record.Delta == nil || record.UnderlyingClose == nil || *record.UnderlyingClose <= 0 {
				continue
			}
			dte := int(day(record.ExpirationDate).Sub(day(session)).Hours() / 24)
			tolerance := 15
			if cell.dte >= 60 {
				tolerance = 20
			}
			if cell.dte >= 90 {
				tolerance = 30
			}
			if dte >= 7 && dte <= 180 && absInt(dte-cell.dte) <= tolerance && math.Abs(math.Abs(*record.Delta)-cell.delta) <= .15 {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func filterDistributions(records []storage.MarketOpsOptionsDistributionRecord, start, end time.Time) []storage.MarketOpsOptionsDistributionRecord {
	out := make([]storage.MarketOpsOptionsDistributionRecord, 0, len(records))
	for _, record := range records {
		if !record.TradeDate.Before(start) && record.TradeDate.Before(end) {
			out = append(out, record)
		}
	}
	return out
}

func day(value time.Time) time.Time {
	if value.IsZero() {
		return value
	}
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func dateKey(value time.Time) string {
	return day(value).Format("2006-01-02")
}

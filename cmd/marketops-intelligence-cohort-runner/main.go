package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/config"
	"github.com/lukebabs/signalops/internal/storage"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
)

var allowedStages = map[string]bool{"preflight": true, "state_materialization": true, "hypothesis_evaluation": true, "opportunity_build": true, "outcome_materialization": true, "hypothesis_proposal_generation": true}

type cliConfig struct {
	TenantID, UniverseGroup, RunID, Actor      string
	Symbols, Stages                            []string
	MaxSymbols                                 int
	SessionStart, SessionEnd                   time.Time
	ContinueOnError, DryRun, AcknowledgeWrites bool
}
type summary struct {
	RunID         string           `json:"run_id"`
	TenantID      string           `json:"tenant_id"`
	UniverseGroup string           `json:"universe_group"`
	Symbols       []string         `json:"symbols"`
	Stages        []string         `json:"stages"`
	DryRun        bool             `json:"dry_run"`
	Status        string           `json:"status"`
	SymbolResults []map[string]any `json:"symbol_results"`
	Errors        []string         `json:"errors"`
}

type repository interface {
	storage.MarketOpsIntelligenceCohortRepository
	ListMarketOpsAssets(context.Context, string, string, bool, int) ([]storage.MarketOpsAssetRecord, error)
	ListMarketOpsMarketStates(context.Context, storage.MarketOpsMarketStateFilter) ([]storage.MarketOpsMarketStateRecord, error)
	ListMarketOpsFeatureObservations(context.Context, storage.MarketOpsFeatureObservationFilter) ([]storage.MarketOpsFeatureObservationRecord, error)
	ListMarketOpsHypothesisEvaluations(context.Context, storage.MarketOpsHypothesisEvaluationFilter) ([]storage.MarketOpsHypothesisEvaluationRecord, error)
	ListMarketOpsOpportunities(context.Context, storage.MarketOpsOpportunityFilter) ([]storage.MarketOpsOpportunityRecord, error)
	ListMarketOpsSignalOutcomes(context.Context, storage.MarketOpsSignalOutcomeFilter) ([]storage.MarketOpsSignalOutcomeRecord, error)
	ListMarketOpsDSMGraphProposals(context.Context, storage.MarketOpsDSMGraphProposalFilter) ([]storage.MarketOpsDSMGraphProposalRecord, error)
	ListMarketOpsBacktestCalibrationSummaries(context.Context, storage.MarketOpsBacktestCalibrationSummaryFilter) ([]storage.MarketOpsBacktestCalibrationSummaryRecord, error)
}

func main() {
	if err := run(); err != nil {
		encoded, _ := json.Marshal(map[string]any{"status": "failed", "error": err.Error()})
		fmt.Println(string(encoded))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	app := config.Load()
	if strings.TrimSpace(app.DatabaseURL) == "" {
		return errors.New("SIGNALOPS_DATABASE_URL is required")
	}
	ctx := context.Background()
	repo, err := postgresstorage.Open(ctx, app.DatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	result, err := execute(ctx, repo, cfg)
	encoded, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(encoded))
	return err
}

func loadConfig() (cliConfig, error) {
	var symbols, stages, start, end string
	cfg := cliConfig{}
	flag.StringVar(&cfg.TenantID, "tenant-id", "tenant-local", "tenant id")
	flag.StringVar(&symbols, "symbols", "", "explicit comma-separated symbols")
	flag.StringVar(&cfg.UniverseGroup, "universe-group", "", "asset universe used only when symbols are omitted")
	flag.IntVar(&cfg.MaxSymbols, "max-symbols", 10, "hard cohort cap; maximum 10")
	flag.StringVar(&start, "session-start", time.Now().UTC().AddDate(0, -1, 0).Format("2006-01-02"), "inclusive session start")
	flag.StringVar(&end, "session-end", time.Now().UTC().Format("2006-01-02"), "inclusive session end")
	flag.StringVar(&stages, "stages", "preflight", "comma-separated explicit stages")
	flag.BoolVar(&cfg.ContinueOnError, "continue-on-error", false, "continue after a symbol stage failure")
	flag.BoolVar(&cfg.DryRun, "dry-run", true, "run child stages without writes and persist no cohort rows")
	flag.BoolVar(&cfg.AcknowledgeWrites, "acknowledge-writes", false, "explicit acknowledgement required for writes")
	flag.StringVar(&cfg.RunID, "run-id", "", "stable operator-supplied run id")
	flag.Parse()
	cfg.TenantID = strings.TrimSpace(cfg.TenantID)
	cfg.UniverseGroup = strings.TrimSpace(cfg.UniverseGroup)
	cfg.RunID = strings.TrimSpace(cfg.RunID)
	cfg.Actor = strings.TrimSpace(os.Getenv("SIGNALOPS_ACTOR"))
	if cfg.Actor == "" {
		cfg.Actor = "operator-local"
	}
	cfg.Symbols = parseList(symbols, true)
	cfg.Stages = parseStages(stages)
	var err error
	if cfg.SessionStart, err = time.Parse("2006-01-02", strings.TrimSpace(start)); err != nil {
		return cfg, err
	}
	if cfg.SessionEnd, err = time.Parse("2006-01-02", strings.TrimSpace(end)); err != nil {
		return cfg, err
	}
	if cfg.RunID == "" {
		cfg.RunID = "intelcohort_" + digest(cfg.TenantID, strings.Join(cfg.Symbols, ","), cfg.UniverseGroup, strings.Join(cfg.Stages, ","), start, end)
	}
	return cfg, validate(cfg)
}
func validate(cfg cliConfig) error {
	if cfg.TenantID == "" || cfg.RunID == "" {
		return errors.New("tenant-id and run-id are required")
	}
	if (len(cfg.Symbols) == 0) == (cfg.UniverseGroup == "") {
		return errors.New("provide either --symbols or --universe-group")
	}
	if cfg.MaxSymbols < 1 || cfg.MaxSymbols > 10 {
		return errors.New("max-symbols must be between 1 and 10")
	}
	if len(cfg.Symbols) > cfg.MaxSymbols {
		return fmt.Errorf("explicit cohort exceeds max-symbols %d", cfg.MaxSymbols)
	}
	if cfg.SessionEnd.Before(cfg.SessionStart) {
		return errors.New("session-end must not precede session-start")
	}
	if len(cfg.Stages) == 0 {
		return errors.New("at least one stage is required")
	}
	for _, stage := range cfg.Stages {
		if !allowedStages[stage] {
			return fmt.Errorf("unsupported stage %s", stage)
		}
	}
	if !cfg.DryRun && !cfg.AcknowledgeWrites {
		return errors.New("--acknowledge-writes is required when dry-run=false")
	}
	return nil
}
func execute(ctx context.Context, repo repository, cfg cliConfig) (summary, error) {
	symbols := append([]string{}, cfg.Symbols...)
	if len(symbols) == 0 {
		assets, err := repo.ListMarketOpsAssets(ctx, cfg.TenantID, cfg.UniverseGroup, true, cfg.MaxSymbols+1)
		if err != nil {
			return summary{}, err
		}
		for _, asset := range assets {
			symbols = append(symbols, strings.ToUpper(asset.Ticker))
		}
	}
	symbols = unique(symbols)
	if len(symbols) == 0 {
		return summary{}, errors.New("cohort resolved no symbols")
	}
	if len(symbols) > cfg.MaxSymbols || len(symbols) > 10 {
		return summary{}, errors.New("resolved cohort exceeds hard 10-symbol cap")
	}
	out := summary{RunID: cfg.RunID, TenantID: cfg.TenantID, UniverseGroup: cfg.UniverseGroup, Symbols: symbols, Stages: cfg.Stages, DryRun: cfg.DryRun, Status: storage.MarketOpsCohortRunRunning}
	started := time.Now().UTC()
	runRecord := storage.MarketOpsIntelligenceCohortRunRecord{RunID: cfg.RunID, TenantID: cfg.TenantID, AppID: "marketops", UniverseGroup: cfg.UniverseGroup, RequestedSymbols: cfg.Symbols, ResolvedSymbols: symbols, Stages: cfg.Stages, MaxSymbols: cfg.MaxSymbols, DryRun: cfg.DryRun, ContinueOnError: cfg.ContinueOnError, Status: storage.MarketOpsCohortRunRunning, AggregateJSON: []byte(`{}`), ErrorsJSON: []byte(`[]`), Actor: cfg.Actor, SessionStart: cfg.SessionStart, SessionEnd: cfg.SessionEnd, StartedAt: started}
	if !cfg.DryRun {
		if _, err := repo.GetMarketOpsIntelligenceCohortRun(ctx, cfg.TenantID, cfg.RunID); err == nil {
			return out, errors.New("cohort run_id already exists")
		}
		if err := repo.UpsertMarketOpsIntelligenceCohortRun(ctx, runRecord); err != nil {
			return out, err
		}
	}
	failed := 0
	for _, symbol := range symbols {
		statuses := map[string]string{}
		stageErrors := map[string]string{}
		for _, stage := range cfg.Stages {
			if stage == "preflight" {
				statuses[stage] = "succeeded"
				continue
			}
			err := runStage(ctx, cfg, symbol, stage)
			if err != nil {
				statuses[stage] = "failed"
				stageErrors[stage] = sanitize(err.Error())
				failed++
				if !cfg.ContinueOnError {
					break
				}
			} else {
				statuses[stage] = "succeeded"
			}
		}
		row, err := inspectReadiness(ctx, repo, cfg, symbol, statuses, stageErrors)
		if err != nil {
			failed++
			out.Errors = append(out.Errors, symbol+": readiness inspection failed")
			row = blockedResult(cfg, symbol, statuses, stageErrors, "readiness inspection failed")
		}
		out.SymbolResults = append(out.SymbolResults, cohortResultOutput(row))
		if !cfg.DryRun {
			if err := repo.UpsertMarketOpsIntelligenceCohortSymbolResult(ctx, row); err != nil {
				return out, err
			}
		}
		if failed > 0 && !cfg.ContinueOnError {
			break
		}
	}
	switch {
	case cfg.DryRun:
		out.Status = storage.MarketOpsCohortRunDryRun
	case failed == 0:
		out.Status = storage.MarketOpsCohortRunSucceeded
	case len(out.SymbolResults) > 0:
		out.Status = storage.MarketOpsCohortRunPartial
	default:
		out.Status = storage.MarketOpsCohortRunFailed
	}
	if !cfg.DryRun {
		completed := time.Now().UTC()
		runRecord.Status = out.Status
		runRecord.CompletedAt = &completed
		runRecord.AggregateJSON = mustJSON(map[string]any{"symbols_requested": len(symbols), "symbols_completed": len(out.SymbolResults), "stage_failures": failed})
		runRecord.ErrorsJSON = mustJSON(out.Errors)
		if err := repo.UpsertMarketOpsIntelligenceCohortRun(ctx, runRecord); err != nil {
			return out, err
		}
	}
	return out, nil
}
func runStage(ctx context.Context, cfg cliConfig, symbol, stage string) error {
	start, end := cfg.SessionStart.Format("2006-01-02"), cfg.SessionEnd.Format("2006-01-02")
	runID := cfg.RunID + "_" + strings.ToLower(symbol) + "_" + stage
	var name string
	var args []string
	switch stage {
	case "state_materialization":
		name = "signalops-marketops-state-materializer"
		args = []string{"--tenant-id", cfg.TenantID, "--symbols", symbol, "--max-symbols", "1", "--window-start", start, "--window-end", cfg.SessionEnd.AddDate(0, 0, 1).Format("2006-01-02"), "--run-id", runID}
	case "hypothesis_evaluation":
		name = "signalops-marketops-hypothesis-evaluator"
		args = []string{"--tenant-id", cfg.TenantID, "--symbol", symbol, "--session-start", start, "--session-end", end, "--run-id", runID, "--max-sessions", "50", "--cohort-run-id", cfg.RunID}
	case "opportunity_build":
		name = "signalops-marketops-opportunity-builder"
		args = []string{"--tenant-id", cfg.TenantID, "--symbol", symbol, "--session-start", start, "--session-end", end, "--run-id", runID, "--max-sessions", "50", "--cohort-run-id", cfg.RunID}
	case "outcome_materialization":
		name = "signalops-marketops-outcome-materializer"
		args = []string{"--tenant-id", cfg.TenantID, "--symbol", symbol, "--session-start", start, "--session-end", end, "--as-of", end, "--run-id", runID, "--max-sessions", "50", "--cohort-run-id", cfg.RunID}
	case "hypothesis_proposal_generation":
		name = "signalops-marketops-hypothesis-proposal-generator"
		args = []string{"--tenant-id", cfg.TenantID, "--symbol", symbol, "--session-start", start, "--session-end", end, "--run-id", runID, "--created-by", cfg.Actor, "--cohort-run-id", cfg.RunID}
	}
	if cfg.DryRun {
		args = append(args, "--dry-run")
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = os.Environ()
	raw, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", stage, sanitize(string(raw)))
	}
	return nil
}
func inspectReadiness(ctx context.Context, repo repository, cfg cliConfig, symbol string, statuses, stageErrors map[string]string) (storage.MarketOpsIntelligenceCohortSymbolResultRecord, error) {
	row := storage.MarketOpsIntelligenceCohortSymbolResultRecord{ResultID: "cohortres_" + digest(cfg.RunID, symbol), RunID: cfg.RunID, TenantID: cfg.TenantID, UniverseGroup: cfg.UniverseGroup, Symbol: symbol, AssetID: "ticker:" + symbol, StageStatusJSON: mustJSON(statuses), StageErrorsJSON: mustJSON(stageErrors), InputCoverageJSON: []byte(`{}`), ProposalStatusCountsJSON: []byte(`{}`), CoverageState: "unavailable", EvaluationState: "not_run", GovernanceState: "research_only", CalibrationState: "unavailable", OutcomeState: "unavailable", RolloutStatus: "not_observed"}
	states, err := repo.ListMarketOpsMarketStates(ctx, storage.MarketOpsMarketStateFilter{TenantID: cfg.TenantID, AppID: "marketops", Symbol: symbol, SessionStart: cfg.SessionStart, SessionEnd: cfg.SessionEnd, Limit: 1})
	if err != nil {
		return row, err
	}
	if len(states) == 0 {
		row.ReadinessReasons = []string{"no persisted market state in cohort session range"}
		return row, nil
	}
	state := states[0]
	day := state.SessionDate
	row.LatestMarketStateID = state.MarketStateID
	row.LatestStateDate = &day
	row.LatestStateSchemaVersion = state.StateSchemaVersion
	row.LatestStateQuality = state.QualityState
	row.LatestStateCompleteness = state.CompletenessRatio
	if state.QualityState == storage.MarketOpsQualityUsable || state.QualityState == storage.MarketOpsQualityUsableWithWarning {
		row.CoverageState = "usable"
	} else {
		row.CoverageState = "incomplete"
		row.ReadinessReasons = append(row.ReadinessReasons, "market state quality is "+state.QualityState)
	}
	row.RequiredFeatureCoverage = state.CompletenessRatio
	obs, _ := repo.ListMarketOpsFeatureObservations(ctx, storage.MarketOpsFeatureObservationFilter{TenantID: cfg.TenantID, Symbol: symbol, FeatureKey: "surface_coverage_ratio", SessionStart: day, SessionEnd: day, Limit: 1})
	if len(obs) > 0 && obs[0].NumericValue != nil {
		row.SurfaceCoverage = *obs[0].NumericValue
	}
	row.InputCoverageJSON = mustJSON(map[string]any{"market_state_count": len(states), "state_feature_count": state.FeatureCount, "required_feature_count": state.RequiredFeatureCount, "surface_coverage_observation_count": len(obs)})
	evals, err := repo.ListMarketOpsHypothesisEvaluations(ctx, storage.MarketOpsHypothesisEvaluationFilter{TenantID: cfg.TenantID, MarketStateID: state.MarketStateID, Symbol: symbol, Limit: 100})
	if err != nil {
		return row, err
	}
	row.EvaluationCount = len(evals)
	pairs := map[string]bool{}
	for _, x := range evals {
		if x.Eligible {
			row.EligibleCount++
		}
		if x.Triggered {
			row.TriggeredCount++
		}
		row.EvaluationRejectionReasons = append(row.EvaluationRejectionReasons, x.ReasonCodes...)
		pairs[x.HypothesisKey+"|"+x.HypothesisVersion] = true
	}
	if row.EvaluationCount == 0 {
		if row.CoverageState != "usable" {
			row.EvaluationState = "blocked"
		}
	} else if row.TriggeredCount > 0 {
		row.EvaluationState = "triggered"
	} else {
		row.EvaluationState = "evaluated_no_trigger"
	}
	opps, err := repo.ListMarketOpsOpportunities(ctx, storage.MarketOpsOpportunityFilter{TenantID: cfg.TenantID, Symbol: symbol, SessionStart: cfg.SessionStart, SessionEnd: cfg.SessionEnd, Limit: 100})
	if err != nil {
		return row, err
	}
	row.OpportunityCount = len(opps)
	proposalCounts := map[string]int{}
	proposals, err := repo.ListMarketOpsDSMGraphProposals(ctx, storage.MarketOpsDSMGraphProposalFilter{TenantID: cfg.TenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SubjectSymbol: symbol, Limit: 500})
	if err != nil {
		return row, err
	}
	for _, proposal := range proposals {
		proposalCounts[proposal.Status]++
	}
	row.ProposalStatusCountsJSON = mustJSON(proposalCounts)
	if proposalCounts[storage.MarketOpsDSMGraphProposalStatusProposed] > 0 {
		row.GovernanceState = "proposal_pending"
	} else if len(proposals) > 0 {
		row.GovernanceState = "reviewed"
	} else if len(opps) > 0 {
		row.GovernanceState = "candidate"
	}
	if len(opps) > 0 {
		row.RolloutStatus = "review_ready"
	} else if row.EvaluationCount > 0 {
		row.RolloutStatus = "research_evaluation_ready"
	} else {
		row.RolloutStatus = "inspection_ready"
	}
	outcomes, err := repo.ListMarketOpsSignalOutcomes(ctx, storage.MarketOpsSignalOutcomeFilter{TenantID: cfg.TenantID, Symbol: symbol, OriginStart: cfg.SessionStart, OriginEnd: cfg.SessionEnd, Limit: 200})
	if err != nil {
		return row, err
	}
	for _, x := range outcomes {
		if x.OutcomeStatus == storage.MarketOpsOutcomeMatured {
			row.MaturedOutcomeCount++
		} else if x.OutcomeStatus == storage.MarketOpsOutcomePending {
			row.PendingOutcomeCount++
		}
	}
	if row.MaturedOutcomeCount > 0 {
		row.OutcomeState = "matured"
	} else if row.PendingOutcomeCount > 0 {
		row.OutcomeState = "pending"
	}
	cals, err := repo.ListMarketOpsBacktestCalibrationSummaries(ctx, storage.MarketOpsBacktestCalibrationSummaryFilter{TenantID: cfg.TenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", Limit: 200})
	if err != nil {
		return row, err
	}
	for _, cal := range cals {
		var report map[string]any
		if json.Unmarshal(cal.ParametersJSON, &report) != nil || fmt.Sprint(report["summary_version"]) != "marketops.hypothesis_calibration.v1" {
			continue
		}
		key := fmt.Sprint(report["hypothesis_key"])
		versions, _ := report["hypothesis_versions"].([]any)
		for _, version := range versions {
			if pairs[key+"|"+fmt.Sprint(version)] {
				row.ExactCalibrationCount++
				if warnings, ok := report["warnings"].([]any); ok && len(warnings) > 0 {
					row.CalibrationBelowMinimum = true
				}
			}
		}
	}
	if row.ExactCalibrationCount > 0 {
		if row.CalibrationBelowMinimum {
			row.CalibrationState = "below_minimum"
		} else {
			row.CalibrationState = "available"
		}
	} else if row.EvaluationCount > 0 {
		row.ReadinessReasons = append(row.ReadinessReasons, "exact-version calibration unavailable")
	}
	if len(stageErrors) > 0 || row.CoverageState == "incomplete" {
		row.RolloutStatus = "blocked"
	}
	row.EvaluationRejectionReasons = unique(row.EvaluationRejectionReasons)
	row.ReadinessReasons = unique(row.ReadinessReasons)
	return row, nil
}
func blockedResult(cfg cliConfig, symbol string, statuses, errs map[string]string, reason string) storage.MarketOpsIntelligenceCohortSymbolResultRecord {
	return storage.MarketOpsIntelligenceCohortSymbolResultRecord{ResultID: "cohortres_" + digest(cfg.RunID, symbol), RunID: cfg.RunID, TenantID: cfg.TenantID, UniverseGroup: cfg.UniverseGroup, Symbol: symbol, AssetID: "ticker:" + symbol, StageStatusJSON: mustJSON(statuses), StageErrorsJSON: mustJSON(errs), InputCoverageJSON: []byte(`{}`), ProposalStatusCountsJSON: []byte(`{}`), CoverageState: "unavailable", EvaluationState: "blocked", GovernanceState: "research_only", CalibrationState: "unavailable", OutcomeState: "unavailable", RolloutStatus: "blocked", ReadinessReasons: []string{reason}}
}
func parseList(value string, upper bool) []string {
	out := []string{}
	for _, x := range strings.Split(value, ",") {
		x = strings.TrimSpace(x)
		if upper {
			x = strings.ToUpper(x)
		} else {
			x = strings.ToLower(x)
		}
		if x != "" {
			out = append(out, x)
		}
	}
	return unique(out)
}
func unique(values []string) []string {
	set := map[string]bool{}
	for _, x := range values {
		if x != "" {
			set[x] = true
		}
	}
	out := make([]string, 0, len(set))
	for x := range set {
		out = append(out, x)
	}
	sort.Strings(out)
	return out
}
func digest(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])[:24]
}
func mustJSON(v any) []byte { raw, _ := json.Marshal(v); return raw }
func sanitize(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) > 300 {
		return value[:300]
	}
	return value
}

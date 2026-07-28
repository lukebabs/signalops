package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/config"
	marketopsstate "github.com/lukebabs/signalops/internal/marketops/state"
	"github.com/lukebabs/signalops/internal/platformregistry"
	"github.com/lukebabs/signalops/internal/storage"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
)

type repository interface {
	platformregistry.DefinitionLister
	ListMarketOpsBacktestNormalizedEvents(context.Context, storage.MarketOpsBacktestEventFilter) ([]storage.NormalizedEventLedgerRecord, error)
	ListMarketOpsOptionsDistributions(context.Context, storage.MarketOpsOptionsDistributionFilter) ([]storage.MarketOpsOptionsDistributionRecord, error)
	ListMarketOpsOptionsChain(context.Context, storage.MarketOpsOptionsChainFilter) ([]storage.MarketOpsOptionsChainRecord, error)
	UpsertMarketOpsFeatureDefinition(context.Context, storage.MarketOpsFeatureDefinitionRecord) error
	UpsertMarketOpsFeatureObservation(context.Context, storage.MarketOpsFeatureObservationRecord) error
	UpsertMarketOpsMarketState(context.Context, storage.MarketOpsMarketStateRecord) error
	UpsertMarketOpsStateTransition(context.Context, storage.MarketOpsStateTransitionRecord) error
	UpsertMarketOpsEvidence(context.Context, storage.MarketOpsEvidenceRecord) error
}

type cliConfig struct {
	TenantID            string
	Symbol              string
	AssetID             string
	WindowStart         time.Time
	WindowEnd           time.Time
	Symbols             []string
	MaxSymbols          int
	MaxSessions         int
	RunID               string
	DryRun              bool
	RegistryEnforcement bool
	FreshWindow         bool
}

type metrics struct {
	RunID               string         `json:"run_id"`
	TenantID            string         `json:"tenant_id"`
	Symbol              string         `json:"symbol"`
	Symbols             []string       `json:"symbols,omitempty"`
	SymbolsRequested    int            `json:"symbols_requested,omitempty"`
	SymbolsProcessed    int            `json:"symbols_processed,omitempty"`
	SymbolResults       []metrics      `json:"symbol_results,omitempty"`
	WindowStart         string         `json:"window_start"`
	WindowEnd           string         `json:"window_end"`
	EquityEvents        int            `json:"equity_events"`
	OptionDistributions int            `json:"option_distributions"`
	OptionContracts     int            `json:"option_contracts"`
	MarketEvents        int            `json:"market_events"`
	Definitions         int            `json:"feature_definitions"`
	Observations        int            `json:"feature_observations"`
	States              int            `json:"market_states"`
	Transitions         int            `json:"state_transitions"`
	Evidence            int            `json:"evidence"`
	StateQualityCounts  map[string]int `json:"state_quality_counts"`
	DryRun              bool           `json:"dry_run"`
	StartedAt           string         `json:"started_at"`
	CompletedAt         string         `json:"completed_at"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("marketops state materializer failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	appConfig := config.Load()
	if strings.TrimSpace(appConfig.DatabaseURL) == "" || strings.TrimSpace(appConfig.TemporalDatabaseURL) == "" {
		return errors.New("SIGNALOPS_DATABASE_URL and SIGNALOPS_TEMPORAL_DATABASE_URL are required")
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	ctx := context.Background()
	repo, err := postgresstorage.OpenWithTemporal(ctx, appConfig.DatabaseURL, appConfig.TemporalDatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	result, err := materializeCohort(ctx, repo, cfg)
	if err != nil {
		return err
	}
	encoded, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(encoded))
	logger.Info("marketops state materializer completed", "run_id", result.RunID, "symbols_processed", result.SymbolsProcessed, "states", result.States, "evidence", result.Evidence, "dry_run", result.DryRun)
	return nil
}

func materializeCohort(ctx context.Context, repo repository, cfg cliConfig) (metrics, error) {
	cfg.Symbols = parseExplicitSymbols(strings.Join(cfg.Symbols, ","))
	if err := cfg.validateCohort(); err != nil {
		return metrics{}, err
	}
	startedAt := time.Now().UTC()
	result := metrics{
		RunID: cfg.RunID, TenantID: cfg.TenantID, Symbols: append([]string{}, cfg.Symbols...),
		SymbolsRequested: len(cfg.Symbols), WindowStart: cfg.WindowStart.Format(time.RFC3339), WindowEnd: cfg.WindowEnd.Format(time.RFC3339),
		StateQualityCounts: map[string]int{}, DryRun: cfg.DryRun, StartedAt: startedAt.Format(time.RFC3339Nano),
	}
	for _, symbol := range cfg.Symbols {
		child := cfg
		child.Symbols = nil
		child.Symbol = symbol
		child.AssetID = "ticker:" + symbol
		if len(cfg.Symbols) == 1 && strings.TrimSpace(cfg.AssetID) != "" {
			child.AssetID = strings.TrimSpace(cfg.AssetID)
		}
		child.RunID = cfg.RunID + "_" + strings.ToLower(symbol)
		childResult, err := materialize(ctx, repo, child)
		if err != nil {
			return result, fmt.Errorf("materialize %s: %w", symbol, err)
		}
		result.SymbolResults = append(result.SymbolResults, childResult)
		result.SymbolsProcessed++
		result.EquityEvents += childResult.EquityEvents
		result.OptionDistributions += childResult.OptionDistributions
		result.OptionContracts += childResult.OptionContracts
		result.MarketEvents += childResult.MarketEvents
		if childResult.Definitions > result.Definitions {
			result.Definitions = childResult.Definitions
		}
		result.Observations += childResult.Observations
		result.States += childResult.States
		result.Transitions += childResult.Transitions
		result.Evidence += childResult.Evidence
		for quality, count := range childResult.StateQualityCounts {
			result.StateQualityCounts[quality] += count
		}
	}
	result.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
	return result, nil
}

func materialize(ctx context.Context, repo repository, cfg cliConfig) (metrics, error) {
	if err := cfg.validate(); err != nil {
		return metrics{}, err
	}
	startedAt := time.Now().UTC()
	warmupStart := cfg.WindowStart.AddDate(0, 0, -90)
	if cfg.FreshWindow {
		warmupStart = cfg.WindowStart
	}
	result := metrics{RunID: cfg.RunID, TenantID: cfg.TenantID, Symbol: cfg.Symbol, WindowStart: cfg.WindowStart.Format(time.RFC3339), WindowEnd: cfg.WindowEnd.Format(time.RFC3339), StateQualityCounts: map[string]int{}, DryRun: cfg.DryRun, StartedAt: startedAt.Format(time.RFC3339Nano)}
	events, err := repo.ListMarketOpsBacktestNormalizedEvents(ctx, storage.MarketOpsBacktestEventFilter{
		TenantID: cfg.TenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance",
		Dataset: "equity_eod_prices", Symbols: []string{cfg.Symbol}, WindowStart: warmupStart, WindowEnd: cfg.WindowEnd,
		Limit: min(cfg.MaxSessions*4, 1000),
	})
	if err != nil {
		return result, err
	}
	eventEvents, err := repo.ListMarketOpsBacktestNormalizedEvents(ctx, storage.MarketOpsBacktestEventFilter{
		TenantID: cfg.TenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance",
		Dataset: "market_event_calendar", Symbols: []string{cfg.Symbol}, WindowStart: cfg.WindowStart.AddDate(-1, 0, 0), WindowEnd: cfg.WindowEnd.AddDate(0, 6, 0),
		Limit: 1000,
	})
	if err != nil {
		return result, err
	}
	if cfg.RegistryEnforcement {
		validator := platformregistry.MassiveNormalizedEventDefinitionValidator{Lister: repo}
		for _, event := range append(append([]storage.NormalizedEventLedgerRecord{}, events...), eventEvents...) {
			if err := validator.ValidateNormalizedEvent(ctx, event); err != nil {
				return result, fmt.Errorf("validate normalized event %s platform definitions: %w", event.EventID, err)
			}
		}
	}
	distributions, err := repo.ListMarketOpsOptionsDistributions(ctx, storage.MarketOpsOptionsDistributionFilter{TenantID: cfg.TenantID, Symbol: cfg.Symbol, Limit: 1000})
	if err != nil {
		return result, err
	}
	distributions = filterDistributions(distributions, warmupStart, cfg.WindowEnd)
	chain, err := repo.ListMarketOpsOptionsChain(ctx, storage.MarketOpsOptionsChainFilter{TenantID: cfg.TenantID, Symbol: cfg.Symbol, WindowStart: warmupStart, WindowEnd: cfg.WindowEnd, Limit: 5000})
	if err != nil {
		return result, err
	}
	result.EquityEvents, result.MarketEvents, result.OptionDistributions, result.OptionContracts = len(events), len(eventEvents), len(distributions), len(chain)
	built, err := marketopsstate.Build(marketopsstate.BuildConfig{TenantID: cfg.TenantID, Symbol: cfg.Symbol, AssetID: cfg.AssetID, RunID: cfg.RunID, SessionStart: cfg.WindowStart, SessionEnd: cfg.WindowEnd, MaxSessions: cfg.MaxSessions}, marketopsstate.BuildInput{EquityEvents: events, EventEvents: eventEvents, Distributions: distributions, OptionChain: chain})
	if err != nil {
		return result, err
	}
	propagateOptionsArtifactProvenance(&built, distributions, chain)
	result.Definitions, result.Observations, result.States, result.Transitions, result.Evidence = len(built.Definitions), len(built.Observations), len(built.States), len(built.Transitions), len(built.Evidence)
	for _, state := range built.States {
		result.StateQualityCounts[state.QualityState]++
	}
	if !cfg.DryRun {
		for _, record := range built.Definitions {
			if err := repo.UpsertMarketOpsFeatureDefinition(ctx, record); err != nil {
				return result, err
			}
		}
		for _, record := range built.Observations {
			if err := repo.UpsertMarketOpsFeatureObservation(ctx, record); err != nil {
				return result, err
			}
		}
		for _, record := range built.States {
			if err := repo.UpsertMarketOpsMarketState(ctx, record); err != nil {
				return result, err
			}
		}
		for _, record := range built.Transitions {
			if err := repo.UpsertMarketOpsStateTransition(ctx, record); err != nil {
				return result, err
			}
		}
		for _, record := range built.Evidence {
			if err := repo.UpsertMarketOpsEvidence(ctx, record); err != nil {
				return result, err
			}
		}
	}
	result.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
	return result, nil
}

func loadConfig() (cliConfig, error) {
	now := time.Now().UTC()
	startDefault := now.AddDate(-1, 0, 0).Format("2006-01-02")
	endDefault := now.AddDate(0, 0, 1).Format("2006-01-02")
	var startValue, endValue, symbolsValue string
	cfg := cliConfig{}
	flag.StringVar(&cfg.TenantID, "tenant-id", "tenant-local", "tenant id")
	flag.StringVar(&cfg.Symbol, "symbol", "", "one explicit asset symbol; compatibility alias for --symbols")
	flag.StringVar(&symbolsValue, "symbols", "", "comma-separated explicit asset symbols; defaults to AAPL")
	flag.IntVar(&cfg.MaxSymbols, "max-symbols", 5, "hard cap for the explicit symbol cohort; maximum 10")
	flag.StringVar(&cfg.AssetID, "asset-id", "", "canonical asset id; permitted only for a single-symbol cohort")
	flag.StringVar(&startValue, "window-start", startDefault, "inclusive session start date")
	flag.StringVar(&endValue, "window-end", endDefault, "exclusive session end date")
	flag.IntVar(&cfg.MaxSessions, "max-sessions", 100, "maximum unioned equity/options sessions")
	flag.StringVar(&cfg.RunID, "run-id", "", "materialization run id")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "build without ledger writes")
	flag.BoolVar(&cfg.FreshWindow, "fresh-window", false, "omit historical warmup inputs for a forward-only provenance-valid window")
	flag.Parse()
	var err error
	if cfg.WindowStart, err = parseDate(startValue); err != nil {
		return cliConfig{}, fmt.Errorf("window-start: %w", err)
	}
	if cfg.WindowEnd, err = parseDate(endValue); err != nil {
		return cliConfig{}, fmt.Errorf("window-end: %w", err)
	}
	cfg.TenantID = strings.TrimSpace(cfg.TenantID)
	cfg.Symbol = strings.ToUpper(strings.TrimSpace(cfg.Symbol))
	cfg.Symbols = parseExplicitSymbols(symbolsValue)
	if len(cfg.Symbols) == 0 && cfg.Symbol != "" {
		cfg.Symbols = []string{cfg.Symbol}
	}
	if len(cfg.Symbols) == 0 {
		cfg.Symbols = []string{"AAPL"}
	}
	cfg.AssetID = strings.TrimSpace(cfg.AssetID)
	if cfg.AssetID == "" && len(cfg.Symbols) == 1 {
		cfg.AssetID = "ticker:" + cfg.Symbols[0]
	}
	if strings.TrimSpace(cfg.RunID) == "" {
		cfg.RunID = "mstatebuild_" + randomHex(12)
	}
	cfg.RegistryEnforcement = envBoolOrDefault("SIGNALOPS_PLATFORM_REGISTRY_ENFORCEMENT", false)
	return cfg, nil
}

func (cfg cliConfig) validateCohort() error {
	if strings.TrimSpace(cfg.TenantID) == "" || strings.TrimSpace(cfg.RunID) == "" {
		return errors.New("tenant-id and run-id are required")
	}
	if cfg.MaxSymbols <= 0 || cfg.MaxSymbols > 10 {
		return errors.New("max-symbols must be between 1 and 10")
	}
	if len(cfg.Symbols) == 0 {
		return errors.New("at least one explicit symbol is required")
	}
	if len(cfg.Symbols) > cfg.MaxSymbols {
		return fmt.Errorf("explicit symbol cohort has %d symbols, exceeds max-symbols %d", len(cfg.Symbols), cfg.MaxSymbols)
	}
	if len(cfg.Symbols) > 1 && strings.TrimSpace(cfg.AssetID) != "" {
		return errors.New("asset-id is only valid for a single-symbol cohort")
	}
	if cfg.WindowStart.IsZero() || cfg.WindowEnd.IsZero() || !cfg.WindowEnd.After(cfg.WindowStart) {
		return errors.New("window-end must be after window-start")
	}
	if cfg.MaxSessions <= 0 || cfg.MaxSessions > 1000 {
		return errors.New("max-sessions must be between 1 and 1000")
	}
	return nil
}

func parseExplicitSymbols(value string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, item := range strings.Split(value, ",") {
		symbol := strings.ToUpper(strings.TrimSpace(item))
		if symbol == "" {
			continue
		}
		if _, exists := seen[symbol]; exists {
			continue
		}
		seen[symbol] = struct{}{}
		out = append(out, symbol)
	}
	return out
}

func (cfg cliConfig) validate() error {
	if cfg.TenantID == "" || cfg.Symbol == "" || cfg.AssetID == "" || cfg.RunID == "" {
		return errors.New("tenant-id, symbol, asset-id, and run-id are required")
	}
	if cfg.WindowStart.IsZero() || cfg.WindowEnd.IsZero() || !cfg.WindowEnd.After(cfg.WindowStart) {
		return errors.New("window-end must be after window-start")
	}
	if cfg.MaxSessions <= 0 || cfg.MaxSessions > 1000 {
		return errors.New("max-sessions must be between 1 and 1000")
	}
	return nil
}

func propagateOptionsArtifactProvenance(built *marketopsstate.BuildResult, distributions []storage.MarketOpsOptionsDistributionRecord, chain []storage.MarketOpsOptionsChainRecord) {
	byArtifact := map[string]any{}
	for _, row := range distributions {
		if len(row.ProvenanceJSON) > 0 && string(row.ProvenanceJSON) != "{}" {
			var value any
			if json.Unmarshal(row.ProvenanceJSON, &value) == nil {
				byArtifact["marketops_options_distribution_daily:"+row.TenantID+":"+strings.ToUpper(row.Symbol)+":"+row.TradeDate.UTC().Format("2006-01-02")+":"+row.WindowName] = value
			}
		}
	}
	for _, row := range chain {
		if len(row.ProvenanceJSON) > 0 && string(row.ProvenanceJSON) != "{}" {
			var value any
			if json.Unmarshal(row.ProvenanceJSON, &value) == nil {
				byArtifact["marketops_options_chain_daily:"+row.TenantID+":"+strings.ToUpper(row.Symbol)+":"+row.TradeDate.UTC().Format("2006-01-02")+":"+row.OptionTicker] = value
			}
		}
	}
	if len(byArtifact) == 0 {
		return
	}
	byObservation := map[string][]any{}
	for index := range built.Observations {
		row := &built.Observations[index]
		entries := []any{}
		for _, artifactID := range row.SourceArtifactIDs {
			if value, ok := byArtifact[artifactID]; ok {
				entries = append(entries, map[string]any{"artifact_id": artifactID, "provenance": value})
			}
		}
		if len(entries) == 0 {
			continue
		}
		details := map[string]any{}
		_ = json.Unmarshal(row.QualityDetailsJSON, &details)
		details["options_artifact_provenance"] = entries
		row.QualityDetailsJSON, _ = json.Marshal(details)
		byObservation[row.FeatureObservationID] = entries
	}
	for index := range built.States {
		row := &built.States[index]
		entries := []any{}
		seen := map[string]bool{}
		for _, id := range row.FeatureObservationIDs {
			for _, entry := range byObservation[id] {
				encoded, _ := json.Marshal(entry)
				if !seen[string(encoded)] {
					seen[string(encoded)] = true
					entries = append(entries, entry)
				}
			}
		}
		if len(entries) == 0 {
			continue
		}
		summary := map[string]any{}
		_ = json.Unmarshal(row.QualitySummaryJSON, &summary)
		summary["options_artifact_provenance"] = entries
		row.QualitySummaryJSON, _ = json.Marshal(summary)
	}
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

func parseDate(value string) (time.Time, error) {
	return time.Parse("2006-01-02", strings.TrimSpace(value))
}
func envBoolOrDefault(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

func randomHex(size int) string {
	value := make([]byte, size)
	_, _ = rand.Read(value)
	return hex.EncodeToString(value)
}

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
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/config"
	marketopsstate "github.com/lukebabs/signalops/internal/marketops/state"
	"github.com/lukebabs/signalops/internal/storage"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
)

type repository interface {
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
	TenantID    string
	Symbol      string
	AssetID     string
	WindowStart time.Time
	WindowEnd   time.Time
	MaxSessions int
	RunID       string
	DryRun      bool
}

type metrics struct {
	RunID               string         `json:"run_id"`
	TenantID            string         `json:"tenant_id"`
	Symbol              string         `json:"symbol"`
	WindowStart         string         `json:"window_start"`
	WindowEnd           string         `json:"window_end"`
	EquityEvents        int            `json:"equity_events"`
	OptionDistributions int            `json:"option_distributions"`
	OptionContracts     int            `json:"option_contracts"`
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
	result, err := materialize(ctx, repo, cfg)
	if err != nil {
		return err
	}
	encoded, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(encoded))
	logger.Info("marketops state materializer completed", "run_id", result.RunID, "states", result.States, "evidence", result.Evidence, "dry_run", result.DryRun)
	return nil
}

func materialize(ctx context.Context, repo repository, cfg cliConfig) (metrics, error) {
	if err := cfg.validate(); err != nil {
		return metrics{}, err
	}
	startedAt := time.Now().UTC()
	warmupStart := cfg.WindowStart.AddDate(0, 0, -90)
	result := metrics{RunID: cfg.RunID, TenantID: cfg.TenantID, Symbol: cfg.Symbol, WindowStart: cfg.WindowStart.Format(time.RFC3339), WindowEnd: cfg.WindowEnd.Format(time.RFC3339), StateQualityCounts: map[string]int{}, DryRun: cfg.DryRun, StartedAt: startedAt.Format(time.RFC3339Nano)}
	events, err := repo.ListMarketOpsBacktestNormalizedEvents(ctx, storage.MarketOpsBacktestEventFilter{
		TenantID: cfg.TenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance",
		Dataset: "equity_eod_prices", Symbols: []string{cfg.Symbol}, WindowStart: warmupStart, WindowEnd: cfg.WindowEnd,
		Limit: min(cfg.MaxSessions*4, 1000),
	})
	if err != nil {
		return result, err
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
	result.EquityEvents, result.OptionDistributions, result.OptionContracts = len(events), len(distributions), len(chain)
	built, err := marketopsstate.Build(marketopsstate.BuildConfig{TenantID: cfg.TenantID, Symbol: cfg.Symbol, AssetID: cfg.AssetID, RunID: cfg.RunID, SessionStart: cfg.WindowStart, SessionEnd: cfg.WindowEnd, MaxSessions: cfg.MaxSessions}, marketopsstate.BuildInput{EquityEvents: events, Distributions: distributions, OptionChain: chain})
	if err != nil {
		return result, err
	}
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
	var startValue, endValue string
	cfg := cliConfig{}
	flag.StringVar(&cfg.TenantID, "tenant-id", "tenant-local", "tenant id")
	flag.StringVar(&cfg.Symbol, "symbol", "AAPL", "asset symbol; G137 permits AAPL only")
	flag.StringVar(&cfg.AssetID, "asset-id", "", "canonical asset id; defaults to ticker:<symbol>")
	flag.StringVar(&startValue, "window-start", startDefault, "inclusive session start date")
	flag.StringVar(&endValue, "window-end", endDefault, "exclusive session end date")
	flag.IntVar(&cfg.MaxSessions, "max-sessions", 100, "maximum unioned equity/options sessions")
	flag.StringVar(&cfg.RunID, "run-id", "", "materialization run id")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "build without ledger writes")
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
	cfg.AssetID = strings.TrimSpace(cfg.AssetID)
	if cfg.AssetID == "" {
		cfg.AssetID = "ticker:" + cfg.Symbol
	}
	if strings.TrimSpace(cfg.RunID) == "" {
		cfg.RunID = "mstatebuild_" + randomHex(12)
	}
	return cfg, nil
}

func (cfg cliConfig) validate() error {
	if cfg.TenantID == "" || cfg.Symbol == "" || cfg.AssetID == "" || cfg.RunID == "" {
		return errors.New("tenant-id, symbol, asset-id, and run-id are required")
	}
	if cfg.Symbol != "AAPL" {
		return errors.New("G137 is intentionally bounded to AAPL")
	}
	if cfg.WindowStart.IsZero() || cfg.WindowEnd.IsZero() || !cfg.WindowEnd.After(cfg.WindowStart) {
		return errors.New("window-end must be after window-start")
	}
	if cfg.MaxSessions <= 0 || cfg.MaxSessions > 1000 {
		return errors.New("max-sessions must be between 1 and 1000")
	}
	return nil
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
func randomHex(size int) string {
	value := make([]byte, size)
	_, _ = rand.Read(value)
	return hex.EncodeToString(value)
}

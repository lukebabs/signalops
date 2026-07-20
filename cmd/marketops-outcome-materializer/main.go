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
	"github.com/lukebabs/signalops/internal/marketops/outcomes"
	"github.com/lukebabs/signalops/internal/storage"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
)

type repository interface {
	ListMarketOpsHypothesisEvaluations(context.Context, storage.MarketOpsHypothesisEvaluationFilter) ([]storage.MarketOpsHypothesisEvaluationRecord, error)
	ListMarketOpsOpportunities(context.Context, storage.MarketOpsOpportunityFilter) ([]storage.MarketOpsOpportunityRecord, error)
	ListMarketOpsBacktestNormalizedEvents(context.Context, storage.MarketOpsBacktestEventFilter) ([]storage.NormalizedEventLedgerRecord, error)
	UpsertMarketOpsSignalOutcome(context.Context, storage.MarketOpsSignalOutcomeRecord) error
}

type cliConfig struct {
	TenantID, Symbol, RunID  string
	SessionStart, SessionEnd time.Time
	AsOf                     time.Time
	MaxSessions              int
	Threshold                float64
	DryRun                   bool
}

type metrics struct {
	RunID          string         `json:"run_id"`
	TenantID       string         `json:"tenant_id"`
	Symbol         string         `json:"symbol"`
	AsOf           string         `json:"as_of"`
	Evaluations    int            `json:"evaluations"`
	Opportunities  int            `json:"opportunities"`
	EquityEvents   int            `json:"equity_events"`
	OutcomeSources int            `json:"outcome_sources"`
	Outcomes       int            `json:"outcomes"`
	Matured        int            `json:"matured"`
	Pending        int            `json:"pending"`
	MissingPrice   int            `json:"missing_price"`
	SkippedReasons map[string]int `json:"skipped_reasons"`
	DryRun         bool           `json:"dry_run"`
	StartedAt      string         `json:"started_at"`
	CompletedAt    string         `json:"completed_at"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("marketops outcome materializer failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	app := config.Load()
	if strings.TrimSpace(app.DatabaseURL) == "" || strings.TrimSpace(app.TemporalDatabaseURL) == "" {
		return errors.New("SIGNALOPS_DATABASE_URL and SIGNALOPS_TEMPORAL_DATABASE_URL are required")
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	ctx := context.Background()
	repo, err := postgresstorage.OpenWithTemporal(ctx, app.DatabaseURL, app.TemporalDatabaseURL)
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
	logger.Info("marketops outcome materializer completed", "run_id", result.RunID, "outcomes", result.Outcomes, "matured", result.Matured, "dry_run", result.DryRun)
	return nil
}

func materialize(ctx context.Context, repo repository, cfg cliConfig) (metrics, error) {
	if err := cfg.validate(); err != nil {
		return metrics{}, err
	}
	started := time.Now().UTC()
	result := metrics{RunID: cfg.RunID, TenantID: cfg.TenantID, Symbol: cfg.Symbol, AsOf: cfg.AsOf.Format("2006-01-02"), SkippedReasons: map[string]int{}, DryRun: cfg.DryRun, StartedAt: started.Format(time.RFC3339Nano)}
	evaluations, err := repo.ListMarketOpsHypothesisEvaluations(ctx, storage.MarketOpsHypothesisEvaluationFilter{
		TenantID: cfg.TenantID, AppID: "marketops", Symbol: cfg.Symbol,
		SessionStart: cfg.SessionStart, SessionEnd: cfg.SessionEnd, Limit: cfg.MaxSessions * 4,
	})
	if err != nil {
		return result, err
	}
	opportunities, err := repo.ListMarketOpsOpportunities(ctx, storage.MarketOpsOpportunityFilter{
		TenantID: cfg.TenantID, AppID: "marketops", Symbol: cfg.Symbol,
		SessionStart: cfg.SessionStart, SessionEnd: cfg.SessionEnd, Limit: cfg.MaxSessions,
	})
	if err != nil {
		return result, err
	}
	events, err := repo.ListMarketOpsBacktestNormalizedEvents(ctx, storage.MarketOpsBacktestEventFilter{
		TenantID: cfg.TenantID, AppID: "marketops", Domain: "market_data",
		UseCase: "daily_market_surveillance", Dataset: "equity_eod_prices", Symbols: []string{cfg.Symbol},
		WindowStart: cfg.SessionStart.AddDate(0, 0, -60), WindowEnd: cfg.AsOf.AddDate(0, 0, 1), Limit: 200,
	})
	if err != nil {
		return result, err
	}
	result.Evaluations, result.Opportunities, result.EquityEvents = len(evaluations), len(opportunities), len(events)
	built, err := outcomes.Build(outcomes.BuildConfig{
		TenantID: cfg.TenantID, Symbol: cfg.Symbol, RunID: cfg.RunID, AsOf: cfg.AsOf, Threshold: cfg.Threshold,
	}, outcomes.BuildInput{Evaluations: evaluations, Opportunities: opportunities, EquityEvents: events})
	if err != nil {
		return result, err
	}
	result.OutcomeSources, result.Outcomes = built.Sources, len(built.Outcomes)
	result.Matured, result.Pending, result.MissingPrice = built.Matured, built.Pending, built.MissingPrice
	result.SkippedReasons = built.SkippedReasons
	for _, outcome := range built.Outcomes {
		if !cfg.DryRun {
			if err := repo.UpsertMarketOpsSignalOutcome(ctx, outcome); err != nil {
				return result, err
			}
		}
	}
	result.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
	return result, nil
}

func loadConfig() (cliConfig, error) {
	now := time.Now().UTC()
	var startValue, endValue, asOfValue string
	cfg := cliConfig{}
	flag.StringVar(&cfg.TenantID, "tenant-id", "tenant-local", "tenant id")
	flag.StringVar(&cfg.Symbol, "symbol", "AAPL", "G140 asset symbol; AAPL only")
	flag.StringVar(&startValue, "session-start", now.AddDate(-1, 0, 0).Format("2006-01-02"), "inclusive source session start")
	flag.StringVar(&endValue, "session-end", now.Format("2006-01-02"), "inclusive source session end")
	flag.StringVar(&asOfValue, "as-of", now.Format("2006-01-02"), "point-in-time outcome cutoff")
	flag.IntVar(&cfg.MaxSessions, "max-sessions", 50, "maximum source sessions (1-50)")
	flag.Float64Var(&cfg.Threshold, "threshold", outcomes.DefaultThreshold, "absolute threshold return (0-1)")
	flag.StringVar(&cfg.RunID, "run-id", "", "calculation run id")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "calculate without writes")
	flag.Parse()
	var err error
	if cfg.SessionStart, err = time.Parse("2006-01-02", strings.TrimSpace(startValue)); err != nil {
		return cfg, err
	}
	if cfg.SessionEnd, err = time.Parse("2006-01-02", strings.TrimSpace(endValue)); err != nil {
		return cfg, err
	}
	if cfg.AsOf, err = time.Parse("2006-01-02", strings.TrimSpace(asOfValue)); err != nil {
		return cfg, err
	}
	cfg.TenantID, cfg.Symbol, cfg.RunID = strings.TrimSpace(cfg.TenantID), strings.ToUpper(strings.TrimSpace(cfg.Symbol)), strings.TrimSpace(cfg.RunID)
	if cfg.RunID == "" {
		cfg.RunID = "outcome_" + randomHex(12)
	}
	return cfg, nil
}

func (cfg cliConfig) validate() error {
	if cfg.TenantID == "" || cfg.RunID == "" {
		return errors.New("tenant-id and run-id are required")
	}
	if cfg.Symbol != "AAPL" {
		return errors.New("G140 is intentionally bounded to AAPL")
	}
	if cfg.SessionStart.IsZero() || cfg.SessionEnd.IsZero() || cfg.AsOf.IsZero() || cfg.SessionEnd.Before(cfg.SessionStart) {
		return errors.New("source session dates and as-of are invalid")
	}
	if cfg.SessionEnd.After(cfg.AsOf) {
		return errors.New("session-end must not be after as-of")
	}
	if cfg.MaxSessions <= 0 || cfg.MaxSessions > 50 {
		return errors.New("max-sessions must be between 1 and 50")
	}
	if cfg.Threshold <= 0 || cfg.Threshold >= 1 {
		return errors.New("threshold must be between 0 and 1")
	}
	return nil
}

func randomHex(length int) string {
	value := make([]byte, length)
	_, _ = rand.Read(value)
	return hex.EncodeToString(value)
}

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
	"github.com/lukebabs/signalops/internal/marketops/history"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
)

type cliConfig struct {
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

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("marketops historical research runner failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	app := config.Load()
	if strings.TrimSpace(app.DatabaseURL) == "" || strings.TrimSpace(app.TemporalDatabaseURL) == "" {
		return errors.New("SIGNALOPS_DATABASE_URL and SIGNALOPS_TEMPORAL_DATABASE_URL are required")
	}
	cfg, err := loadConfig(os.Args[1:])
	if err != nil {
		return err
	}
	ctx := context.Background()
	repo, err := postgresstorage.OpenWithTemporal(ctx, app.DatabaseURL, app.TemporalDatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()

	result, err := history.Run(ctx, repo, history.Config{
		TenantID: cfg.TenantID, Symbol: cfg.Symbol, AssetID: cfg.AssetID, RunID: cfg.RunID,
		SessionStart: cfg.SessionStart, SessionEnd: cfg.SessionEnd, AsOf: cfg.AsOf,
		MaxSessions: cfg.MaxSessions, MinimumEquitySessions: cfg.MinimumEquitySessions,
		MinimumOptionsSessions: cfg.MinimumOptionsSessions, OutcomeThreshold: cfg.OutcomeThreshold,
		AllowInsufficientCoverage: cfg.AllowInsufficientCoverage, DryRun: cfg.DryRun,
	})
	if err != nil {
		return err
	}
	encoded, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(encoded))
	logger.Info("marketops historical research runner completed",
		"run_id", result.RunID, "status", result.Status, "states", result.MarketStates,
		"triggered", result.TriggeredEvaluations, "opportunities", result.Opportunities,
		"outcomes", result.Outcomes, "dry_run", result.DryRun)
	return nil
}

func loadConfig(args []string) (cliConfig, error) {
	now := time.Now().UTC()
	var startValue, endValue, asOfValue string
	cfg := cliConfig{}
	flags := flag.NewFlagSet("marketops-history-runner", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.StringVar(&cfg.TenantID, "tenant-id", "tenant-local", "tenant id")
	flags.StringVar(&cfg.Symbol, "symbol", "AAPL", "G141 asset symbol; AAPL only")
	flags.StringVar(&cfg.AssetID, "asset-id", "", "canonical asset id; defaults to ticker:<symbol>")
	flags.StringVar(&startValue, "session-start", now.AddDate(-1, 0, 0).Format("2006-01-02"), "inclusive source-session start")
	flags.StringVar(&endValue, "session-end", now.AddDate(0, 0, 1).Format("2006-01-02"), "exclusive source-session end")
	flags.StringVar(&asOfValue, "as-of", now.Format("2006-01-02"), "point-in-time outcome cutoff")
	flags.IntVar(&cfg.MaxSessions, "max-sessions", history.DefaultMaximumSessions, "maximum source sessions (1-200)")
	flags.IntVar(&cfg.MinimumEquitySessions, "minimum-equity-sessions", history.DefaultMinimumEquitySessions, "minimum persisted equity sessions")
	flags.IntVar(&cfg.MinimumOptionsSessions, "minimum-options-sessions", history.DefaultMinimumOptionsSessions, "minimum analytics-ready option sessions")
	flags.Float64Var(&cfg.OutcomeThreshold, "outcome-threshold", 0.02, "absolute forward-return threshold (0-1)")
	flags.StringVar(&cfg.RunID, "run-id", "", "historical research run id")
	flags.BoolVar(&cfg.AllowInsufficientCoverage, "allow-insufficient-coverage", false, "run a diagnostic partial build despite failed coverage")
	flags.BoolVar(&cfg.DryRun, "dry-run", false, "calculate without ledger writes")
	if err := flags.Parse(args); err != nil {
		return cliConfig{}, err
	}
	var err error
	if cfg.SessionStart, err = time.Parse("2006-01-02", strings.TrimSpace(startValue)); err != nil {
		return cliConfig{}, fmt.Errorf("session-start: %w", err)
	}
	if cfg.SessionEnd, err = time.Parse("2006-01-02", strings.TrimSpace(endValue)); err != nil {
		return cliConfig{}, fmt.Errorf("session-end: %w", err)
	}
	if cfg.AsOf, err = time.Parse("2006-01-02", strings.TrimSpace(asOfValue)); err != nil {
		return cliConfig{}, fmt.Errorf("as-of: %w", err)
	}
	cfg.TenantID = strings.TrimSpace(cfg.TenantID)
	cfg.Symbol = strings.ToUpper(strings.TrimSpace(cfg.Symbol))
	cfg.AssetID = strings.TrimSpace(cfg.AssetID)
	cfg.RunID = strings.TrimSpace(cfg.RunID)
	if cfg.AssetID == "" {
		cfg.AssetID = "ticker:" + cfg.Symbol
	}
	if cfg.RunID == "" {
		cfg.RunID = "history_" + randomHex(12)
	}
	return cfg, nil
}

func randomHex(length int) string {
	value := make([]byte, length)
	_, _ = rand.Read(value)
	return hex.EncodeToString(value)
}

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
	"github.com/lukebabs/signalops/internal/marketops/hypothesisbacktest"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("marketops hypothesis backtest failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	app := config.Load()
	if strings.TrimSpace(app.DatabaseURL) == "" || strings.TrimSpace(app.TemporalDatabaseURL) == "" {
		return errors.New("SIGNALOPS_DATABASE_URL and SIGNALOPS_TEMPORAL_DATABASE_URL are required")
	}
	cfg, err := loadCLIConfig()
	if err != nil {
		return err
	}
	ctx := context.Background()
	repo, err := postgresstorage.OpenWithTemporal(ctx, app.DatabaseURL, app.TemporalDatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	result, err := hypothesisbacktest.Run(ctx, repo, cfg)
	if err != nil {
		return err
	}
	encoded, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(encoded))
	logger.Info("marketops hypothesis backtest completed",
		"run_id", cfg.RunID, "summary_id", cfg.SummaryID, "mode", result.Report.Mode,
		"warnings", len(result.Report.Warnings), "dry_run", cfg.DryRun)
	return nil
}

func loadCLIConfig() (hypothesisbacktest.RunConfig, error) {
	now := time.Now().UTC()
	var versions, symbols, startValue, endValue, asOfValue string
	cfg := hypothesisbacktest.RunConfig{}
	flag.StringVar(&cfg.RunID, "run-id", "", "isolated backtest run id")
	flag.StringVar(&cfg.SummaryID, "summary-id", "", "versioned calibration summary id")
	flag.StringVar(&cfg.TenantID, "tenant-id", "tenant-local", "tenant id")
	flag.StringVar(&cfg.HypothesisKey, "hypothesis-key", "", "exact hypothesis key, for example H001")
	flag.StringVar(&versions, "hypothesis-versions", "", "comma-separated exact versions; comparison requires baseline,candidate")
	flag.StringVar(&cfg.OutcomeCalculationVersion, "outcome-calculation-version", hypothesisbacktest.DefaultOutcomeCalculationVersion, "exact forward-outcome calculation version")
	flag.StringVar(&symbols, "symbols", "AAPL", "explicit comma-separated symbols, maximum 10")
	flag.StringVar(&cfg.Mode, "mode", hypothesisbacktest.ModeSingle, "single, comparison, or walk_forward")
	flag.StringVar(&startValue, "window-start", now.AddDate(-1, 0, 0).Format("2006-01-02"), "inclusive evaluation start date")
	flag.StringVar(&endValue, "window-end", now.Format("2006-01-02"), "inclusive evaluation end date")
	flag.StringVar(&asOfValue, "as-of", now.Format("2006-01-02"), "point-in-time outcome maturity cutoff")
	flag.IntVar(&cfg.MinimumSampleSize, "minimum-sample-size", 100, "warning threshold for independent triggered evaluations with matured outcomes")
	flag.IntVar(&cfg.TrainSessions, "train-sessions", 60, "initial chronological walk-forward train sessions")
	flag.IntVar(&cfg.TestSessions, "test-sessions", 20, "walk-forward test sessions per fold")
	flag.IntVar(&cfg.MaximumFolds, "max-folds", 6, "maximum walk-forward folds, up to 20")
	flag.IntVar(&cfg.QueryLimit, "query-limit", 1000, "hard per-version, per-symbol ledger read limit")
	flag.StringVar(&cfg.RequestedBy, "requested-by", "operator-local", "requesting operator")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "calculate without backtest or summary writes")
	flag.Parse()

	var err error
	if cfg.WindowStart, err = time.Parse("2006-01-02", strings.TrimSpace(startValue)); err != nil {
		return cfg, errors.New("window-start must be YYYY-MM-DD")
	}
	if cfg.WindowEnd, err = time.Parse("2006-01-02", strings.TrimSpace(endValue)); err != nil {
		return cfg, errors.New("window-end must be YYYY-MM-DD")
	}
	if cfg.AsOf, err = time.Parse("2006-01-02", strings.TrimSpace(asOfValue)); err != nil {
		return cfg, errors.New("as-of must be YYYY-MM-DD")
	}
	cfg.HypothesisVersions = splitValues(versions, false)
	cfg.Symbols = splitValues(symbols, true)
	cfg.RunID = strings.TrimSpace(cfg.RunID)
	cfg.SummaryID = strings.TrimSpace(cfg.SummaryID)
	if cfg.RunID == "" {
		cfg.RunID = "bt_hypothesis_" + randomHex(12)
	}
	if cfg.SummaryID == "" {
		cfg.SummaryID = "btcal_hypothesis_" + randomHex(12)
	}
	return cfg, nil
}

func splitValues(value string, uppercase bool) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if uppercase {
			item = strings.ToUpper(item)
		}
		if item != "" && !seen[item] {
			seen[item] = true
			out = append(out, item)
		}
	}
	return out
}

func randomHex(length int) string {
	value := make([]byte, length)
	_, _ = rand.Read(value)
	return hex.EncodeToString(value)
}

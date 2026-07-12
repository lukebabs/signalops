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
	marketopsbacktest "github.com/lukebabs/signalops/internal/marketops/backtest"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("marketops backtest failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg := config.Load()
	if strings.TrimSpace(cfg.DatabaseURL) == "" || strings.TrimSpace(cfg.TemporalDatabaseURL) == "" {
		return errors.New("SIGNALOPS_DATABASE_URL and SIGNALOPS_TEMPORAL_DATABASE_URL are required")
	}
	backtestCfg, err := loadCLIConfig()
	if err != nil {
		return err
	}
	ctx := context.Background()
	repo, err := postgresstorage.OpenWithTemporal(ctx, cfg.DatabaseURL, cfg.TemporalDatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()

	logger.Info("marketops backtest run started", "run_id", backtestCfg.RunID, "detector_id", backtestCfg.DetectorID, "window_start", backtestCfg.WindowStart, "window_end", backtestCfg.WindowEnd)
	result, err := marketopsbacktest.Run(ctx, repo, backtestCfg)
	if err != nil {
		return err
	}
	out, _ := json.MarshalIndent(map[string]any{"backtest_run": result.Run, "metrics": result.Metrics}, "", "  ")
	fmt.Println(string(out))
	logger.Info("marketops backtest run completed", "run_id", result.Run.RunID, "signals", result.Metrics.Signals, "graph_proposals", result.Metrics.GraphProposals)
	return nil
}

func loadCLIConfig() (marketopsbacktest.Config, error) {
	var start, end, symbols string
	cfg := marketopsbacktest.Config{}
	flag.StringVar(&cfg.RunID, "run-id", "", "backtest run id")
	flag.StringVar(&cfg.TenantID, "tenant-id", "tenant-local", "tenant id")
	flag.StringVar(&cfg.SourceID, "source-id", "", "optional source id filter")
	flag.StringVar(&cfg.SourceAdapter, "source-adapter", "market_data.massive", "source adapter filter")
	flag.StringVar(&cfg.Dataset, "dataset", "equity_eod_prices", "dataset filter")
	flag.StringVar(&cfg.DetectorID, "detector-id", "marketops.dsm.taxonomy_v1", "detector id")
	flag.StringVar(&cfg.DetectorVersion, "detector-version", "", "detector version label")
	flag.StringVar(&cfg.RequestedBy, "requested-by", "operator-local", "requesting operator")
	flag.StringVar(&start, "window-start", "", "inclusive RFC3339 observation start")
	flag.StringVar(&end, "window-end", "", "exclusive RFC3339 observation end")
	flag.StringVar(&symbols, "symbols", "", "comma-separated symbol filter")
	flag.IntVar(&cfg.MaxRecords, "max-records", 50, "maximum normalized events to scan")
	flag.IntVar(&cfg.BatchSize, "batch-size", 50, "normalized events per detector batch")
	flag.Float64Var(&cfg.AutoAcceptConfidence, "auto-accept-confidence", 0.75, "policy auto-accept confidence threshold")
	flag.StringVar(&cfg.PythonBin, "python-bin", "python3", "python executable")
	flag.Parse()
	if strings.TrimSpace(cfg.RunID) == "" {
		cfg.RunID = "bt_marketops_" + randomHex(12)
	}
	if strings.TrimSpace(start) == "" || strings.TrimSpace(end) == "" {
		return marketopsbacktest.Config{}, errors.New("window-start and window-end are required")
	}
	var err error
	cfg.WindowStart, err = time.Parse(time.RFC3339Nano, strings.TrimSpace(start))
	if err != nil {
		return marketopsbacktest.Config{}, errors.New("window-start must be RFC3339")
	}
	cfg.WindowEnd, err = time.Parse(time.RFC3339Nano, strings.TrimSpace(end))
	if err != nil {
		return marketopsbacktest.Config{}, errors.New("window-end must be RFC3339")
	}
	if !cfg.WindowEnd.After(cfg.WindowStart) {
		return marketopsbacktest.Config{}, errors.New("window-end must be after window-start")
	}
	for _, symbol := range strings.Split(symbols, ",") {
		if strings.TrimSpace(symbol) != "" {
			cfg.Symbols = append(cfg.Symbols, strings.ToUpper(strings.TrimSpace(symbol)))
		}
	}
	return cfg, nil
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

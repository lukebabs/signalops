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

	"github.com/lukebabs/signalops/internal/algorithms"
	"github.com/lukebabs/signalops/internal/config"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("algorithm runner failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	appCfg := config.Load()
	if strings.TrimSpace(appCfg.DatabaseURL) == "" || strings.TrimSpace(appCfg.TemporalDatabaseURL) == "" {
		return errors.New("SIGNALOPS_DATABASE_URL and SIGNALOPS_TEMPORAL_DATABASE_URL are required")
	}
	runnerCfg, err := loadCLIConfig()
	if err != nil {
		return err
	}
	ctx := context.Background()
	repo, err := postgresstorage.OpenWithTemporal(ctx, appCfg.DatabaseURL, appCfg.TemporalDatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()

	logger.Info("algorithm execution started", "execution_request_id", runnerCfg.ExecutionRequestID, "algorithm_id", runnerCfg.AlgorithmID, "window_start", runnerCfg.WindowStart, "window_end", runnerCfg.WindowEnd)
	result, err := algorithms.Run(ctx, repo, runnerCfg)
	if err != nil {
		return err
	}
	out, _ := json.MarshalIndent(map[string]any{"algorithm_execution_request": result.ExecutionRequest, "metrics": result.Metrics}, "", "  ")
	fmt.Println(string(out))
	logger.Info("algorithm execution completed", "execution_request_id", result.ExecutionRequest.ExecutionRequestID, "results", result.Metrics.Results)
	return nil
}

func loadCLIConfig() (algorithms.Config, error) {
	var start, end, symbols string
	cfg := algorithms.Config{}
	flag.StringVar(&cfg.ExecutionRequestID, "execution-request-id", "", "algorithm execution request id")
	flag.StringVar(&cfg.TenantID, "tenant-id", "tenant-local", "tenant id")
	flag.StringVar(&cfg.AlgorithmID, "algorithm-id", algorithms.ZScoreAnomalyAlgorithmID, "algorithm id")
	flag.StringVar(&cfg.AlgorithmVersion, "algorithm-version", algorithms.DefaultAlgorithmVersion, "algorithm version")
	flag.StringVar(&cfg.RequestedBy, "requested-by", "operator-local", "requesting operator")
	flag.StringVar(&cfg.CorrelationID, "correlation-id", "", "correlation id")
	flag.StringVar(&cfg.AppID, "app-id", "marketops", "app id filter")
	flag.StringVar(&cfg.Domain, "domain", "market_data", "domain filter")
	flag.StringVar(&cfg.UseCase, "use-case", "daily_market_surveillance", "use case filter")
	flag.StringVar(&cfg.SourceID, "source-id", "", "optional source id filter")
	flag.StringVar(&cfg.SourceAdapter, "source-adapter", "market_data.massive", "source adapter filter")
	flag.StringVar(&cfg.Dataset, "dataset", "equity_eod_prices", "dataset filter")
	flag.StringVar(&cfg.Feature, "feature", algorithms.DefaultZScoreFeature, "numeric feature name")
	flag.StringVar(&start, "window-start", "", "inclusive RFC3339 observation start")
	flag.StringVar(&end, "window-end", "", "exclusive RFC3339 observation end")
	flag.StringVar(&symbols, "symbols", "", "comma-separated symbol filter")
	flag.IntVar(&cfg.MaxRecords, "max-records", 50, "maximum normalized events to scan")
	flag.IntVar(&cfg.BatchSize, "batch-size", 50, "normalized events per scan batch")
	flag.IntVar(&cfg.MinSamples, "min-samples", 3, "minimum usable samples required before writing z-score results")
	flag.Float64Var(&cfg.ZThreshold, "z-threshold", 3.0, "z-score anomaly threshold")
	flag.Parse()
	if strings.TrimSpace(cfg.ExecutionRequestID) == "" {
		cfg.ExecutionRequestID = "algexec_" + randomHex(12)
	}
	if strings.TrimSpace(start) == "" || strings.TrimSpace(end) == "" {
		return algorithms.Config{}, errors.New("window-start and window-end are required")
	}
	var err error
	cfg.WindowStart, err = time.Parse(time.RFC3339Nano, strings.TrimSpace(start))
	if err != nil {
		return algorithms.Config{}, errors.New("window-start must be RFC3339")
	}
	cfg.WindowEnd, err = time.Parse(time.RFC3339Nano, strings.TrimSpace(end))
	if err != nil {
		return algorithms.Config{}, errors.New("window-end must be RFC3339")
	}
	if !cfg.WindowEnd.After(cfg.WindowStart) {
		return algorithms.Config{}, errors.New("window-end must be after window-start")
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

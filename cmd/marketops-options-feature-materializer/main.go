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
	marketopsoptions "github.com/lukebabs/signalops/internal/marketops/options"
	"github.com/lukebabs/signalops/internal/storage"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
)

type repository interface {
	ListMarketOpsOptionsDistributions(context.Context, storage.MarketOpsOptionsDistributionFilter) ([]storage.MarketOpsOptionsDistributionRecord, error)
	UpsertNormalizedEventLedger(context.Context, storage.NormalizedEventLedgerRecord) error
}

type cliConfig struct {
	TenantID   string
	Symbol     string
	WindowName string
	Limit      int
	RunID      string
	DryRun     bool
}

type metrics struct {
	RunID     string `json:"run_id"`
	TenantID  string `json:"tenant_id"`
	Symbol    string `json:"symbol"`
	Window    string `json:"window_name"`
	Scanned   int    `json:"scanned"`
	Upserted  int    `json:"upserted"`
	DryRun    bool   `json:"dry_run"`
	StartedAt string `json:"started_at"`
	EndedAt   string `json:"ended_at"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("marketops options feature materializer failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	appCfg := config.Load()
	if strings.TrimSpace(appCfg.DatabaseURL) == "" {
		return errors.New("SIGNALOPS_DATABASE_URL is required")
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	ctx := context.Background()
	repo, err := postgresstorage.OpenWithTemporal(ctx, appCfg.DatabaseURL, appCfg.TemporalDatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	result, err := materialize(ctx, repo, cfg)
	if err != nil {
		return err
	}
	out, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(out))
	logger.Info("marketops options feature materializer completed", "run_id", result.RunID, "scanned", result.Scanned, "upserted", result.Upserted, "dry_run", result.DryRun)
	return nil
}

func materialize(ctx context.Context, repo repository, cfg cliConfig) (metrics, error) {
	cfg = cfg.withDefaults()
	if err := cfg.validate(); err != nil {
		return metrics{}, err
	}
	startedAt := time.Now().UTC()
	result := metrics{RunID: cfg.RunID, TenantID: cfg.TenantID, Symbol: cfg.Symbol, Window: cfg.WindowName, DryRun: cfg.DryRun, StartedAt: startedAt.Format(time.RFC3339Nano)}
	records, err := repo.ListMarketOpsOptionsDistributions(ctx, storage.MarketOpsOptionsDistributionFilter{TenantID: cfg.TenantID, Symbol: cfg.Symbol, WindowName: cfg.WindowName, Limit: cfg.Limit})
	if err != nil {
		return result, err
	}
	result.Scanned = len(records)
	for _, record := range records {
		event, err := marketopsoptions.NormalizedEventFromDistribution(record, time.Now().UTC())
		if err != nil {
			return result, err
		}
		if cfg.DryRun {
			continue
		}
		if err := repo.UpsertNormalizedEventLedger(ctx, event); err != nil {
			return result, err
		}
		result.Upserted++
	}
	result.EndedAt = time.Now().UTC().Format(time.RFC3339Nano)
	return result, nil
}

func loadConfig() (cliConfig, error) {
	cfg := cliConfig{}
	flag.StringVar(&cfg.TenantID, "tenant-id", "tenant-local", "tenant id")
	flag.StringVar(&cfg.Symbol, "symbol", "NVDA", "asset symbol")
	flag.StringVar(&cfg.WindowName, "window", marketopsoptions.DefaultWindowName, "distribution window")
	flag.IntVar(&cfg.Limit, "limit", 10, "maximum distribution snapshots to materialize")
	flag.StringVar(&cfg.RunID, "run-id", "", "materialization run id")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "build feature events without writing normalized_event_ledger")
	flag.Parse()
	return cfg, nil
}

func (cfg cliConfig) withDefaults() cliConfig {
	cfg.TenantID = strings.TrimSpace(cfg.TenantID)
	cfg.Symbol = strings.ToUpper(strings.TrimSpace(cfg.Symbol))
	cfg.WindowName = strings.TrimSpace(cfg.WindowName)
	if cfg.WindowName == "" {
		cfg.WindowName = marketopsoptions.DefaultWindowName
	}
	if cfg.Limit <= 0 || cfg.Limit > 1000 {
		cfg.Limit = 10
	}
	if strings.TrimSpace(cfg.RunID) == "" {
		cfg.RunID = "optfeat_" + randomHex(12)
	}
	return cfg
}

func (cfg cliConfig) validate() error {
	if cfg.TenantID == "" {
		return errors.New("tenant-id is required")
	}
	if cfg.Symbol == "" {
		return errors.New("symbol is required")
	}
	return nil
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

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

	"github.com/lukebabs/signalops/internal/adapters/marketdata/massive"
	"github.com/lukebabs/signalops/internal/config"
	marketopsoptions "github.com/lukebabs/signalops/internal/marketops/options"
	"github.com/lukebabs/signalops/internal/storage"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
)

type snapshotProvider interface {
	ListOptionChainSnapshot(ctx context.Context, underlying string, limit int, maxPages int) ([]massive.OptionContractDailyRecord, error)
}

type repository interface {
	UpsertMarketOpsOptionsChain(context.Context, storage.MarketOpsOptionsChainRecord) error
	ListMarketOpsOptionsChain(context.Context, storage.MarketOpsOptionsChainFilter) ([]storage.MarketOpsOptionsChainRecord, error)
	UpsertMarketOpsOptionsDistribution(context.Context, storage.MarketOpsOptionsDistributionRecord) error
}

type cliConfig struct {
	TenantID   string
	Symbol     string
	SourceID   string
	RunID      string
	Limit      int
	MaxPages   int
	WindowDays int
	DryRun     bool
}

type metrics struct {
	RunID                     string `json:"run_id"`
	TenantID                  string `json:"tenant_id"`
	Symbol                    string `json:"symbol"`
	SourceID                  string `json:"source_id"`
	DryRun                    bool   `json:"dry_run"`
	Limit                     int    `json:"limit"`
	MaxPages                  int    `json:"max_pages"`
	WindowDays                int    `json:"window_days"`
	Fetched                   int    `json:"fetched"`
	Converted                 int    `json:"converted"`
	Skipped                   int    `json:"skipped"`
	ChainUpserted             int    `json:"chain_upserted"`
	DistributionWritten       bool   `json:"distribution_written"`
	DistributionContractCount int    `json:"distribution_contract_count"`
	DistributionTradeDays     int    `json:"distribution_trade_days"`
	TradeDate                 string `json:"trade_date,omitempty"`
	StartedAt                 string `json:"started_at"`
	EndedAt                   string `json:"ended_at"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("marketops options chain ingestor failed", "error", err)
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
	massiveClient, err := massive.NewClient(massive.LoadClientConfigFromEnv())
	if err != nil {
		return err
	}
	ctx := context.Background()
	repo, err := postgresstorage.OpenWithTemporal(ctx, appCfg.DatabaseURL, appCfg.TemporalDatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	result, err := ingest(ctx, massiveClient, repo, cfg)
	if err != nil {
		return err
	}
	out, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(out))
	logger.Info("marketops options chain ingestor completed", "run_id", result.RunID, "symbol", result.Symbol, "fetched", result.Fetched, "chain_upserted", result.ChainUpserted, "distribution_written", result.DistributionWritten, "dry_run", result.DryRun)
	return nil
}

func ingest(ctx context.Context, provider snapshotProvider, repo repository, cfg cliConfig) (metrics, error) {
	cfg = cfg.withDefaults()
	if err := cfg.validate(); err != nil {
		return metrics{}, err
	}
	startedAt := time.Now().UTC()
	result := metrics{RunID: cfg.RunID, TenantID: cfg.TenantID, Symbol: cfg.Symbol, SourceID: cfg.SourceID, DryRun: cfg.DryRun, Limit: cfg.Limit, MaxPages: cfg.MaxPages, WindowDays: cfg.WindowDays, StartedAt: startedAt.Format(time.RFC3339Nano)}
	providerRecords, err := provider.ListOptionChainSnapshot(ctx, cfg.Symbol, cfg.Limit, cfg.MaxPages)
	if err != nil {
		return result, err
	}
	result.Fetched = len(providerRecords)
	chainRecords := []storage.MarketOpsOptionsChainRecord{}
	for _, providerRecord := range providerRecords {
		chainRecord, err := marketopsoptions.ChainRecordFromMassiveSnapshot(cfg.TenantID, cfg.SourceID, cfg.RunID, providerRecord)
		if err != nil {
			result.Skipped++
			continue
		}
		chainRecords = append(chainRecords, chainRecord)
		result.Converted++
		if cfg.DryRun {
			continue
		}
		if err := repo.UpsertMarketOpsOptionsChain(ctx, chainRecord); err != nil {
			return result, err
		}
		result.ChainUpserted++
	}
	tradeDate := marketopsoptions.LatestTradeDate(chainRecords)
	if tradeDate.IsZero() {
		result.EndedAt = time.Now().UTC().Format(time.RFC3339Nano)
		return result, nil
	}
	result.TradeDate = tradeDate.Format("2006-01-02")
	windowStart := tradeDate.AddDate(0, 0, -cfg.WindowDays+1)
	windowEnd := tradeDate.AddDate(0, 0, 1)
	windowRows, err := repo.ListMarketOpsOptionsChain(ctx, storage.MarketOpsOptionsChainFilter{TenantID: cfg.TenantID, Symbol: cfg.Symbol, WindowStart: windowStart, WindowEnd: windowEnd, Limit: 100000})
	if err != nil {
		return result, err
	}
	if cfg.DryRun {
		windowRows = append(windowRows, chainRecords...)
	}
	distribution := marketopsoptions.BuildDistribution(cfg.TenantID, cfg.Symbol, tradeDate, windowRows)
	result.DistributionContractCount = distribution.ContractCount
	result.DistributionTradeDays = distribution.TradeDays
	if !cfg.DryRun {
		if err := repo.UpsertMarketOpsOptionsDistribution(ctx, distribution); err != nil {
			return result, err
		}
		result.DistributionWritten = true
	}
	result.EndedAt = time.Now().UTC().Format(time.RFC3339Nano)
	return result, nil
}

func loadConfig() (cliConfig, error) {
	cfg := cliConfig{}
	flag.StringVar(&cfg.TenantID, "tenant-id", "tenant-local", "tenant id")
	flag.StringVar(&cfg.Symbol, "symbol", "NVDA", "asset symbol")
	flag.StringVar(&cfg.SourceID, "source-id", "src-massive", "source id")
	flag.StringVar(&cfg.RunID, "run-id", "", "ingestion run id")
	flag.IntVar(&cfg.Limit, "limit", 250, "Massive option-chain page size")
	flag.IntVar(&cfg.MaxPages, "max-pages", 1, "maximum Massive option-chain pages to fetch")
	flag.IntVar(&cfg.WindowDays, "window-days", 10, "calendar-day lookback used to derive the rolling distribution")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "fetch and derive distribution without writing storage")
	flag.Parse()
	return cfg, nil
}

func (cfg cliConfig) withDefaults() cliConfig {
	cfg.TenantID = strings.TrimSpace(cfg.TenantID)
	cfg.Symbol = strings.ToUpper(strings.TrimSpace(cfg.Symbol))
	cfg.SourceID = strings.TrimSpace(cfg.SourceID)
	if cfg.SourceID == "" {
		cfg.SourceID = "src-massive"
	}
	if cfg.Limit <= 0 || cfg.Limit > 250 {
		cfg.Limit = 250
	}
	if cfg.MaxPages <= 0 || cfg.MaxPages > 20 {
		cfg.MaxPages = 1
	}
	if cfg.WindowDays <= 0 || cfg.WindowDays > 60 {
		cfg.WindowDays = 10
	}
	if strings.TrimSpace(cfg.RunID) == "" {
		cfg.RunID = "optchain_" + randomHex(12)
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

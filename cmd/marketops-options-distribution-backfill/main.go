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
	"sort"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/config"
	marketopsoptions "github.com/lukebabs/signalops/internal/marketops/options"
	"github.com/lukebabs/signalops/internal/storage"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
)

type repository interface {
	ListMarketOpsOptionsChain(context.Context, storage.MarketOpsOptionsChainFilter) ([]storage.MarketOpsOptionsChainRecord, error)
	UpsertMarketOpsOptionsDistribution(context.Context, storage.MarketOpsOptionsDistributionRecord) error
}

type cliConfig struct {
	TenantID   string
	Symbol     string
	RunID      string
	WindowDays int
	Limit      int
	DryRun     bool
}

type metrics struct {
	RunID                 string `json:"run_id"`
	TenantID              string `json:"tenant_id"`
	Symbol                string `json:"symbol"`
	WindowName            string `json:"window_name"`
	WindowDays            int    `json:"window_days"`
	DryRun                bool   `json:"dry_run"`
	ChainRowsScanned      int    `json:"chain_rows_scanned"`
	TradeDatesScanned     int    `json:"trade_dates_scanned"`
	DistributionsBuilt    int    `json:"distributions_built"`
	DistributionsUpserted int    `json:"distributions_upserted"`
	FirstTradeDate        string `json:"first_trade_date,omitempty"`
	LastTradeDate         string `json:"last_trade_date,omitempty"`
	StartedAt             string `json:"started_at"`
	EndedAt               string `json:"ended_at"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("marketops options distribution backfill failed", "error", err)
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
	result, err := backfill(ctx, repo, cfg)
	if err != nil {
		return err
	}
	out, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(out))
	logger.Info("marketops options distribution backfill completed", "run_id", result.RunID, "symbol", result.Symbol, "built", result.DistributionsBuilt, "upserted", result.DistributionsUpserted, "dry_run", result.DryRun)
	return nil
}

func backfill(ctx context.Context, repo repository, cfg cliConfig) (metrics, error) {
	cfg = cfg.withDefaults()
	if err := cfg.validate(); err != nil {
		return metrics{}, err
	}
	startedAt := time.Now().UTC()
	result := metrics{RunID: cfg.RunID, TenantID: cfg.TenantID, Symbol: cfg.Symbol, WindowName: marketopsoptions.DefaultWindowName, WindowDays: cfg.WindowDays, DryRun: cfg.DryRun, StartedAt: startedAt.Format(time.RFC3339Nano)}
	rows, err := repo.ListMarketOpsOptionsChain(ctx, storage.MarketOpsOptionsChainFilter{TenantID: cfg.TenantID, Symbol: cfg.Symbol, Limit: cfg.Limit})
	if err != nil {
		return result, err
	}
	result.ChainRowsScanned = len(rows)
	dates := tradeDates(rows)
	result.TradeDatesScanned = len(dates)
	if len(dates) > 0 {
		result.FirstTradeDate = dates[0].Format("2006-01-02")
		result.LastTradeDate = dates[len(dates)-1].Format("2006-01-02")
	}
	for _, tradeDate := range dates {
		windowRows := rowsForWindow(rows, tradeDate, cfg.WindowDays)
		if len(windowRows) == 0 {
			continue
		}
		distribution := marketopsoptions.BuildDistribution(cfg.TenantID, cfg.Symbol, tradeDate, windowRows)
		if distribution.ContractCount == 0 {
			continue
		}
		result.DistributionsBuilt++
		if cfg.DryRun {
			continue
		}
		if err := repo.UpsertMarketOpsOptionsDistribution(ctx, distribution); err != nil {
			return result, err
		}
		result.DistributionsUpserted++
	}
	result.EndedAt = time.Now().UTC().Format(time.RFC3339Nano)
	return result, nil
}

func loadConfig() (cliConfig, error) {
	cfg := cliConfig{}
	flag.StringVar(&cfg.TenantID, "tenant-id", "tenant-local", "tenant id")
	flag.StringVar(&cfg.Symbol, "symbol", "NVDA", "asset symbol")
	flag.StringVar(&cfg.RunID, "run-id", "", "backfill run id")
	flag.IntVar(&cfg.WindowDays, "window-days", 10, "calendar-day lookback used for each distribution snapshot")
	flag.IntVar(&cfg.Limit, "limit", 5000, "maximum persisted chain rows to scan")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "derive distributions without writing storage")
	flag.Parse()
	return cfg, nil
}

func (cfg cliConfig) withDefaults() cliConfig {
	cfg.TenantID = strings.TrimSpace(cfg.TenantID)
	cfg.Symbol = strings.ToUpper(strings.TrimSpace(cfg.Symbol))
	if cfg.WindowDays <= 0 || cfg.WindowDays > 60 {
		cfg.WindowDays = 10
	}
	if cfg.Limit <= 0 || cfg.Limit > 5000 {
		cfg.Limit = 5000
	}
	if strings.TrimSpace(cfg.RunID) == "" {
		cfg.RunID = "optdist_" + randomHex(12)
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

func tradeDates(rows []storage.MarketOpsOptionsChainRecord) []time.Time {
	seen := map[time.Time]struct{}{}
	for _, row := range rows {
		if row.TradeDate.IsZero() {
			continue
		}
		seen[dayOnly(row.TradeDate)] = struct{}{}
	}
	dates := make([]time.Time, 0, len(seen))
	for date := range seen {
		dates = append(dates, date)
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
	return dates
}

func rowsForWindow(rows []storage.MarketOpsOptionsChainRecord, tradeDate time.Time, windowDays int) []storage.MarketOpsOptionsChainRecord {
	tradeDate = dayOnly(tradeDate)
	start := tradeDate.AddDate(0, 0, -windowDays+1)
	out := []storage.MarketOpsOptionsChainRecord{}
	for _, row := range rows {
		date := dayOnly(row.TradeDate)
		if (date.Equal(start) || date.After(start)) && (date.Equal(tradeDate) || date.Before(tradeDate)) {
			out = append(out, row)
		}
	}
	return out
}

func dayOnly(value time.Time) time.Time {
	utc := value.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

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
	ListMarketOpsAssets(ctx context.Context, tenantID string, universeGroup string, activeOnly bool, limit int) ([]storage.MarketOpsAssetRecord, error)
	UpsertMarketOpsOptionsChain(context.Context, storage.MarketOpsOptionsChainRecord) error
	ListMarketOpsOptionsChain(context.Context, storage.MarketOpsOptionsChainFilter) ([]storage.MarketOpsOptionsChainRecord, error)
	UpsertMarketOpsOptionsDistribution(context.Context, storage.MarketOpsOptionsDistributionRecord) error
	ListMarketOpsOptionsDistributions(context.Context, storage.MarketOpsOptionsDistributionFilter) ([]storage.MarketOpsOptionsDistributionRecord, error)
	UpsertNormalizedEventLedger(context.Context, storage.NormalizedEventLedgerRecord) error
}

type cliConfig struct {
	TenantID          string
	Symbols           []string
	UniverseGroup     string
	SourceID          string
	RunID             string
	Limit             int
	MaxPages          int
	MaxSymbols        int
	WindowDays        int
	ChainScanLimit    int
	DistributionLimit int
	DryRun            bool
}

type symbolMetrics struct {
	Symbol                      string         `json:"symbol"`
	Fetched                     int            `json:"fetched"`
	Converted                   int            `json:"converted"`
	Skipped                     int            `json:"skipped"`
	ChainUpserted               int            `json:"chain_upserted"`
	ChainRowsScanned            int            `json:"chain_rows_scanned"`
	TradeDatesScanned           int            `json:"trade_dates_scanned"`
	DistributionsBuilt          int            `json:"distributions_built"`
	DistributionsUpserted       int            `json:"distributions_upserted"`
	FeatureRowsScanned          int            `json:"feature_rows_scanned"`
	FeatureRowsUpserted         int            `json:"feature_rows_upserted"`
	FirstTradeDate              string         `json:"first_trade_date,omitempty"`
	LastTradeDate               string         `json:"last_trade_date,omitempty"`
	LatestTradeDate             string         `json:"latest_trade_date,omitempty"`
	QualityCounts               map[string]int `json:"quality_counts"`
	OpenInterestQualityCounts   map[string]int `json:"open_interest_quality_counts"`
	DistributionContractCount   int            `json:"distribution_contract_count"`
	DistributionSourceTradeDays int            `json:"distribution_source_trade_days"`
}

type metrics struct {
	RunID                 string          `json:"run_id"`
	TenantID              string          `json:"tenant_id"`
	UniverseGroup         string          `json:"universe_group"`
	SourceID              string          `json:"source_id"`
	DryRun                bool            `json:"dry_run"`
	Limit                 int             `json:"limit"`
	MaxPages              int             `json:"max_pages"`
	MaxSymbols            int             `json:"max_symbols"`
	WindowDays            int             `json:"window_days"`
	ChainScanLimit        int             `json:"chain_scan_limit"`
	DistributionLimit     int             `json:"distribution_limit"`
	SymbolsRequested      int             `json:"symbols_requested"`
	SymbolsProcessed      int             `json:"symbols_processed"`
	Fetched               int             `json:"fetched"`
	Converted             int             `json:"converted"`
	Skipped               int             `json:"skipped"`
	ChainUpserted         int             `json:"chain_upserted"`
	DistributionsBuilt    int             `json:"distributions_built"`
	DistributionsUpserted int             `json:"distributions_upserted"`
	FeatureRowsUpserted   int             `json:"feature_rows_upserted"`
	QualityCounts         map[string]int  `json:"quality_counts"`
	Symbols               []symbolMetrics `json:"symbols"`
	StartedAt             string          `json:"started_at"`
	EndedAt               string          `json:"ended_at"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("marketops options coverage runner failed", "error", err)
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
	result, err := runCoverage(ctx, massiveClient, repo, cfg)
	if err != nil {
		return err
	}
	out, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(out))
	logger.Info("marketops options coverage runner completed", "run_id", result.RunID, "symbols_processed", result.SymbolsProcessed, "chain_upserted", result.ChainUpserted, "distributions_upserted", result.DistributionsUpserted, "feature_rows_upserted", result.FeatureRowsUpserted, "dry_run", result.DryRun)
	return nil
}

func runCoverage(ctx context.Context, provider snapshotProvider, repo repository, cfg cliConfig) (metrics, error) {
	cfg = cfg.withDefaults()
	if err := cfg.validate(); err != nil {
		return metrics{}, err
	}
	startedAt := time.Now().UTC()
	out := metrics{RunID: cfg.RunID, TenantID: cfg.TenantID, UniverseGroup: cfg.UniverseGroup, SourceID: cfg.SourceID, DryRun: cfg.DryRun, Limit: cfg.Limit, MaxPages: cfg.MaxPages, MaxSymbols: cfg.MaxSymbols, WindowDays: cfg.WindowDays, ChainScanLimit: cfg.ChainScanLimit, DistributionLimit: cfg.DistributionLimit, QualityCounts: map[string]int{}, StartedAt: startedAt.Format(time.RFC3339Nano)}
	symbols, err := resolveSymbols(ctx, repo, cfg)
	if err != nil {
		return out, err
	}
	out.SymbolsRequested = len(symbols)
	for _, symbol := range symbols {
		item, err := processSymbol(ctx, provider, repo, cfg, symbol)
		if err != nil {
			return out, err
		}
		out.Symbols = append(out.Symbols, item)
		out.SymbolsProcessed++
		out.Fetched += item.Fetched
		out.Converted += item.Converted
		out.Skipped += item.Skipped
		out.ChainUpserted += item.ChainUpserted
		out.DistributionsBuilt += item.DistributionsBuilt
		out.DistributionsUpserted += item.DistributionsUpserted
		out.FeatureRowsUpserted += item.FeatureRowsUpserted
		for key, count := range item.QualityCounts {
			out.QualityCounts[key] += count
		}
	}
	out.EndedAt = time.Now().UTC().Format(time.RFC3339Nano)
	return out, nil
}

func processSymbol(ctx context.Context, provider snapshotProvider, repo repository, cfg cliConfig, symbol string) (symbolMetrics, error) {
	item := symbolMetrics{Symbol: symbol, QualityCounts: map[string]int{}, OpenInterestQualityCounts: map[string]int{}}
	providerRecords, err := provider.ListOptionChainSnapshot(ctx, symbol, cfg.Limit, cfg.MaxPages)
	if err != nil {
		return item, fmt.Errorf("fetch option chain for %s: %w", symbol, err)
	}
	item.Fetched = len(providerRecords)
	chainRecords := []storage.MarketOpsOptionsChainRecord{}
	for _, providerRecord := range providerRecords {
		chainRecord, err := marketopsoptions.ChainRecordFromMassiveSnapshot(cfg.TenantID, cfg.SourceID, cfg.RunID, providerRecord)
		if err != nil {
			item.Skipped++
			continue
		}
		chainRecords = append(chainRecords, chainRecord)
		item.Converted++
		if cfg.DryRun {
			continue
		}
		if err := repo.UpsertMarketOpsOptionsChain(ctx, chainRecord); err != nil {
			return item, err
		}
		item.ChainUpserted++
	}
	rows, err := repo.ListMarketOpsOptionsChain(ctx, storage.MarketOpsOptionsChainFilter{TenantID: cfg.TenantID, Symbol: symbol, Limit: cfg.ChainScanLimit})
	if err != nil {
		return item, err
	}
	if cfg.DryRun {
		rows = append(rows, chainRecords...)
	}
	item.ChainRowsScanned = len(rows)
	dates := tradeDates(rows)
	item.TradeDatesScanned = len(dates)
	if len(dates) > 0 {
		item.FirstTradeDate = dates[0].Format("2006-01-02")
		item.LastTradeDate = dates[len(dates)-1].Format("2006-01-02")
	}
	builtDistributions := []storage.MarketOpsOptionsDistributionRecord{}
	for _, tradeDate := range dates {
		windowRows := rowsForWindow(rows, tradeDate, cfg.WindowDays)
		if len(windowRows) == 0 {
			continue
		}
		distribution := marketopsoptions.BuildDistribution(cfg.TenantID, symbol, tradeDate, windowRows)
		if distribution.ContractCount == 0 {
			continue
		}
		item.DistributionsBuilt++
		item.DistributionContractCount = distribution.ContractCount
		item.DistributionSourceTradeDays = distribution.TradeDays
		item.LatestTradeDate = distribution.TradeDate.Format("2006-01-02")
		quality, oiQuality := qualityFromMetrics(distribution.MetricsJSON)
		item.QualityCounts[quality]++
		item.OpenInterestQualityCounts[oiQuality]++
		builtDistributions = append(builtDistributions, distribution)
		if cfg.DryRun {
			continue
		}
		if err := repo.UpsertMarketOpsOptionsDistribution(ctx, distribution); err != nil {
			return item, err
		}
		item.DistributionsUpserted++
	}
	materializedRows := builtDistributions
	if !cfg.DryRun {
		materializedRows, err = repo.ListMarketOpsOptionsDistributions(ctx, storage.MarketOpsOptionsDistributionFilter{TenantID: cfg.TenantID, Symbol: symbol, WindowName: marketopsoptions.DefaultWindowName, Limit: cfg.DistributionLimit})
		if err != nil {
			return item, err
		}
	}
	if cfg.DistributionLimit > 0 && len(materializedRows) > cfg.DistributionLimit {
		materializedRows = materializedRows[:cfg.DistributionLimit]
	}
	item.FeatureRowsScanned = len(materializedRows)
	for _, distribution := range materializedRows {
		event, err := marketopsoptions.NormalizedEventFromDistribution(distribution, time.Now().UTC())
		if err != nil {
			return item, err
		}
		if cfg.DryRun {
			continue
		}
		if err := repo.UpsertNormalizedEventLedger(ctx, event); err != nil {
			return item, err
		}
		item.FeatureRowsUpserted++
	}
	return item, nil
}

func resolveSymbols(ctx context.Context, repo repository, cfg cliConfig) ([]string, error) {
	if len(cfg.Symbols) > 0 {
		return limitSymbols(cfg.Symbols, cfg.MaxSymbols), nil
	}
	assets, err := repo.ListMarketOpsAssets(ctx, cfg.TenantID, cfg.UniverseGroup, true, cfg.MaxSymbols)
	if err != nil {
		return nil, err
	}
	symbols := []string{}
	for _, asset := range assets {
		symbol := strings.ToUpper(strings.TrimSpace(asset.Ticker))
		if symbol != "" {
			symbols = append(symbols, symbol)
		}
	}
	return limitSymbols(symbols, cfg.MaxSymbols), nil
}

func loadConfig() (cliConfig, error) {
	cfg := cliConfig{}
	symbols := ""
	flag.StringVar(&cfg.TenantID, "tenant-id", "tenant-local", "tenant id")
	flag.StringVar(&symbols, "symbols", "", "comma-separated symbols; when empty, resolve from the asset universe")
	flag.StringVar(&cfg.UniverseGroup, "universe-group", "top50_megacap", "asset universe group used when symbols are omitted")
	flag.StringVar(&cfg.SourceID, "source-id", "src-massive", "source id")
	flag.StringVar(&cfg.RunID, "run-id", "", "coverage run id")
	flag.IntVar(&cfg.Limit, "limit", 250, "Massive option-chain page size per symbol")
	flag.IntVar(&cfg.MaxPages, "max-pages", 1, "maximum Massive option-chain pages to fetch per symbol")
	flag.IntVar(&cfg.MaxSymbols, "max-symbols", 3, "maximum symbols to process")
	flag.IntVar(&cfg.WindowDays, "window-days", 10, "calendar-day lookback used for distribution snapshots")
	flag.IntVar(&cfg.ChainScanLimit, "chain-scan-limit", 5000, "maximum persisted chain rows to scan per symbol")
	flag.IntVar(&cfg.DistributionLimit, "distribution-limit", 100, "maximum distribution snapshots to materialize per symbol")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "fetch and derive without writing storage")
	flag.Parse()
	cfg.Symbols = parseSymbols(symbols)
	return cfg, nil
}

func (cfg cliConfig) withDefaults() cliConfig {
	cfg.TenantID = strings.TrimSpace(cfg.TenantID)
	cfg.UniverseGroup = strings.TrimSpace(cfg.UniverseGroup)
	if cfg.UniverseGroup == "" {
		cfg.UniverseGroup = "top50_megacap"
	}
	cfg.SourceID = strings.TrimSpace(cfg.SourceID)
	if cfg.SourceID == "" {
		cfg.SourceID = "src-massive"
	}
	cfg.Symbols = normalizeSymbols(cfg.Symbols)
	if cfg.Limit <= 0 || cfg.Limit > 250 {
		cfg.Limit = 250
	}
	if cfg.MaxPages <= 0 || cfg.MaxPages > 20 {
		cfg.MaxPages = 1
	}
	if cfg.MaxSymbols <= 0 || cfg.MaxSymbols > 50 {
		cfg.MaxSymbols = 3
	}
	if cfg.WindowDays <= 0 || cfg.WindowDays > 60 {
		cfg.WindowDays = 10
	}
	if cfg.ChainScanLimit <= 0 || cfg.ChainScanLimit > 5000 {
		cfg.ChainScanLimit = 5000
	}
	if cfg.DistributionLimit <= 0 || cfg.DistributionLimit > 1000 {
		cfg.DistributionLimit = 100
	}
	if strings.TrimSpace(cfg.RunID) == "" {
		cfg.RunID = "optcov_" + randomHex(12)
	}
	return cfg
}

func (cfg cliConfig) validate() error {
	if cfg.TenantID == "" {
		return errors.New("tenant-id is required")
	}
	if cfg.MaxSymbols <= 0 {
		return errors.New("max-symbols must be positive")
	}
	return nil
}

func parseSymbols(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return normalizeSymbols(strings.Split(value, ","))
}

func normalizeSymbols(values []string) []string {
	out := []string{}
	seen := map[string]struct{}{}
	for _, value := range values {
		symbol := strings.ToUpper(strings.TrimSpace(value))
		if symbol == "" {
			continue
		}
		if _, ok := seen[symbol]; ok {
			continue
		}
		seen[symbol] = struct{}{}
		out = append(out, symbol)
	}
	return out
}

func limitSymbols(values []string, max int) []string {
	values = normalizeSymbols(values)
	if max > 0 && len(values) > max {
		return values[:max]
	}
	return values
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

func qualityFromMetrics(raw []byte) (string, string) {
	metrics := map[string]any{}
	_ = json.Unmarshal(raw, &metrics)
	ratioQuality, _ := metrics["call_put_oi_ratio_quality"].(string)
	openInterestQuality, _ := metrics["open_interest_quality"].(string)
	if strings.TrimSpace(ratioQuality) == "" {
		ratioQuality = "unknown"
	}
	if strings.TrimSpace(openInterestQuality) == "" {
		openInterestQuality = "unknown"
	}
	return ratioQuality, openInterestQuality
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

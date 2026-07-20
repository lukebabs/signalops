package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
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
	ListOptionChainSnapshotFilteredWithMetadata(ctx context.Context, underlying string, filter massive.OptionChainSnapshotFilter) (massive.OptionChainSnapshotBatch, error)
}

type repository interface {
	ListMarketOpsAssets(ctx context.Context, tenantID string, universeGroup string, activeOnly bool, limit int) ([]storage.MarketOpsAssetRecord, error)
	UpsertMarketOpsOptionsChain(context.Context, storage.MarketOpsOptionsChainRecord) error
	ListMarketOpsOptionsChain(context.Context, storage.MarketOpsOptionsChainFilter) ([]storage.MarketOpsOptionsChainRecord, error)
	UpsertMarketOpsOptionsDistribution(context.Context, storage.MarketOpsOptionsDistributionRecord) error
	ListMarketOpsOptionsDistributions(context.Context, storage.MarketOpsOptionsDistributionFilter) ([]storage.MarketOpsOptionsDistributionRecord, error)
	UpsertNormalizedEventLedger(context.Context, storage.NormalizedEventLedgerRecord) error
	UpsertMarketOpsOptionsCapture(context.Context, storage.MarketOpsOptionsCaptureRecord) error
	GetMarketOpsOptionsCapture(context.Context, string, string) (storage.MarketOpsOptionsCaptureRecord, error)
	ListMarketOpsBacktestNormalizedEvents(context.Context, storage.MarketOpsBacktestEventFilter) ([]storage.NormalizedEventLedgerRecord, error)
}

type cliConfig struct {
	TenantID          string
	Symbols           []string
	UniverseGroup     string
	SourceID          string
	RunID             string
	Limit             int
	MaxPages          int
	MaxCandidates     int
	MinDTE            int
	MaxDTE            int
	MinMoneyness      float64
	MaxMoneyness      float64
	MaxSymbols        int
	WindowDays        int
	ChainScanLimit    int
	DistributionLimit int
	SessionDate       time.Time
	SkipComplete      bool
	ContinueOnError   bool
	MaxRetries        int
	RetryBackoff      time.Duration
	DryRun            bool
}

type symbolMetrics struct {
	Symbol                       string         `json:"symbol"`
	CaptureID                    string         `json:"capture_id,omitempty"`
	CaptureStatus                string         `json:"capture_status,omitempty"`
	AnalyticsReady               bool           `json:"analytics_ready"`
	RequiredSurfaceCells         int            `json:"required_surface_cells"`
	UsableIVCount                int            `json:"usable_iv_count"`
	UsableGreeksCount            int            `json:"usable_greeks_count"`
	OpenInterestCount            int            `json:"open_interest_count"`
	UnderlyingPriceCount         int            `json:"underlying_price_count"`
	UnderlyingPriceSourceEventID string         `json:"underlying_price_source_event_id,omitempty"`
	QualityReasons               []string       `json:"quality_reasons,omitempty"`
	Error                        string         `json:"error,omitempty"`
	Attempts                     int            `json:"attempts"`
	Fetched                      int            `json:"fetched"`
	ProviderRequestIDs           []string       `json:"provider_request_ids,omitempty"`
	ProviderPagesFetched         int            `json:"provider_pages_fetched"`
	ProviderPaginationComplete   bool           `json:"provider_pagination_complete"`
	SelectedEvidence             int            `json:"selected_evidence"`
	DiscardedCandidates          int            `json:"discarded_candidates"`
	AcquisitionExpirationStart   string         `json:"acquisition_expiration_start,omitempty"`
	AcquisitionExpirationEnd     string         `json:"acquisition_expiration_end,omitempty"`
	AcquisitionStrikeMinimum     float64        `json:"acquisition_strike_minimum,omitempty"`
	AcquisitionStrikeMaximum     float64        `json:"acquisition_strike_maximum,omitempty"`
	Converted                    int            `json:"converted"`
	Skipped                      int            `json:"skipped"`
	ChainUpserted                int            `json:"chain_upserted"`
	ChainRowsScanned             int            `json:"chain_rows_scanned"`
	TradeDatesScanned            int            `json:"trade_dates_scanned"`
	DistributionsBuilt           int            `json:"distributions_built"`
	DistributionsUpserted        int            `json:"distributions_upserted"`
	FeatureRowsScanned           int            `json:"feature_rows_scanned"`
	FeatureRowsUpserted          int            `json:"feature_rows_upserted"`
	FirstTradeDate               string         `json:"first_trade_date,omitempty"`
	LastTradeDate                string         `json:"last_trade_date,omitempty"`
	LatestTradeDate              string         `json:"latest_trade_date,omitempty"`
	QualityCounts                map[string]int `json:"quality_counts"`
	OpenInterestQualityCounts    map[string]int `json:"open_interest_quality_counts"`
	DistributionContractCount    int            `json:"distribution_contract_count"`
	DistributionSourceTradeDays  int            `json:"distribution_source_trade_days"`
}

type metrics struct {
	RunID                 string          `json:"run_id"`
	TenantID              string          `json:"tenant_id"`
	UniverseGroup         string          `json:"universe_group"`
	SourceID              string          `json:"source_id"`
	DryRun                bool            `json:"dry_run"`
	Limit                 int             `json:"limit"`
	MaxPages              int             `json:"max_pages"`
	MaxCandidates         int             `json:"max_candidates"`
	MinDTE                int             `json:"min_dte"`
	MaxDTE                int             `json:"max_dte"`
	MinMoneyness          float64         `json:"min_moneyness"`
	MaxMoneyness          float64         `json:"max_moneyness"`
	MaxSymbols            int             `json:"max_symbols"`
	WindowDays            int             `json:"window_days"`
	ChainScanLimit        int             `json:"chain_scan_limit"`
	DistributionLimit     int             `json:"distribution_limit"`
	SessionDate           string          `json:"session_date,omitempty"`
	CaptureEnabled        bool            `json:"capture_enabled"`
	SkippedComplete       int             `json:"skipped_complete"`
	Failed                int             `json:"failed"`
	AnalyticsReady        int             `json:"analytics_ready"`
	Partial               int             `json:"partial"`
	NoData                int             `json:"no_data"`
	SymbolsRequested      int             `json:"symbols_requested"`
	SymbolsProcessed      int             `json:"symbols_processed"`
	Fetched               int             `json:"fetched"`
	SelectedEvidence      int             `json:"selected_evidence"`
	DiscardedCandidates   int             `json:"discarded_candidates"`
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
	out := metrics{RunID: cfg.RunID, TenantID: cfg.TenantID, UniverseGroup: cfg.UniverseGroup, SourceID: cfg.SourceID, DryRun: cfg.DryRun, Limit: cfg.Limit, MaxPages: cfg.MaxPages, MaxCandidates: cfg.MaxCandidates, MinDTE: cfg.MinDTE, MaxDTE: cfg.MaxDTE, MinMoneyness: cfg.MinMoneyness, MaxMoneyness: cfg.MaxMoneyness, MaxSymbols: cfg.MaxSymbols, WindowDays: cfg.WindowDays, ChainScanLimit: cfg.ChainScanLimit, DistributionLimit: cfg.DistributionLimit, QualityCounts: map[string]int{}, StartedAt: startedAt.Format(time.RFC3339Nano), CaptureEnabled: !cfg.SessionDate.IsZero()}
	if !cfg.SessionDate.IsZero() {
		out.SessionDate = dayOnly(cfg.SessionDate).Format("2006-01-02")
		if cfg.SessionDate.Weekday() == time.Saturday || cfg.SessionDate.Weekday() == time.Sunday {
			return out, errors.New("session-date must be a weekday")
		}
	}
	symbols, err := resolveSymbols(ctx, repo, cfg)
	if err != nil {
		return out, err
	}
	out.SymbolsRequested = len(symbols)
	for _, symbol := range symbols {
		captureID := ""
		if !cfg.SessionDate.IsZero() {
			captureID = optionsCaptureID(cfg.TenantID, cfg.SourceID, symbol, cfg.SessionDate)
			if cfg.SkipComplete {
				existing, getErr := repo.GetMarketOpsOptionsCapture(ctx, cfg.TenantID, captureID)
				if getErr == nil && existing.AnalyticsReady {
					out.SkippedComplete++
					continue
				}
				if getErr != nil && !errors.Is(getErr, storage.ErrNotFound) {
					return out, getErr
				}
			}
		}
		item, err := processSymbolWithRetry(ctx, provider, repo, cfg, symbol)
		item.CaptureID = captureID
		if err != nil {
			item.CaptureStatus = storage.MarketOpsOptionsCaptureFailed
			item.Error = err.Error()
			out.Failed++
			if persistErr := persistCapture(ctx, repo, cfg, item, startedAt); persistErr != nil {
				return out, persistErr
			}
			out.Symbols = append(out.Symbols, item)
			if cfg.ContinueOnError {
				continue
			}
			return out, err
		}
		if item.Fetched == 0 {
			item.CaptureStatus = storage.MarketOpsOptionsCaptureNoData
			out.NoData++
		} else if item.AnalyticsReady {
			item.CaptureStatus = storage.MarketOpsOptionsCaptureAnalyticsReady
			out.AnalyticsReady++
		} else {
			item.CaptureStatus = storage.MarketOpsOptionsCapturePartial
			out.Partial++
		}
		if err := persistCapture(ctx, repo, cfg, item, startedAt); err != nil {
			return out, err
		}
		out.Symbols = append(out.Symbols, item)
		out.SymbolsProcessed++
		out.Fetched += item.Fetched
		out.SelectedEvidence += item.SelectedEvidence
		out.DiscardedCandidates += item.DiscardedCandidates
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

func processSymbolWithRetry(ctx context.Context, provider snapshotProvider, repo repository, cfg cliConfig, symbol string) (symbolMetrics, error) {
	var item symbolMetrics
	var err error
	for attempt := 1; attempt <= cfg.MaxRetries+1; attempt++ {
		item, err = processSymbol(ctx, provider, repo, cfg, symbol)
		item.Attempts = attempt
		if err == nil {
			return item, nil
		}
		if attempt <= cfg.MaxRetries {
			timer := time.NewTimer(cfg.RetryBackoff * time.Duration(attempt))
			select {
			case <-ctx.Done():
				timer.Stop()
				return item, ctx.Err()
			case <-timer.C:
			}
		}
	}
	return item, err
}
func processSymbol(ctx context.Context, provider snapshotProvider, repo repository, cfg cliConfig, symbol string) (symbolMetrics, error) {
	item := symbolMetrics{Symbol: symbol, Attempts: 1, QualityCounts: map[string]int{}, OpenInterestQualityCounts: map[string]int{}}
	filter := massive.OptionChainSnapshotFilter{Limit: cfg.Limit, MaxPages: cfg.MaxPages}
	canonicalClose := 0.0
	if !cfg.SessionDate.IsZero() {
		var err error
		canonicalClose, item.UnderlyingPriceSourceEventID, err = resolveSameSessionUnderlyingPrice(ctx, repo, cfg, symbol)
		if err != nil {
			return item, err
		}
		if canonicalClose <= 0 {
			return item, fmt.Errorf("canonical same-session underlying close is required before bounded options acquisition for %s", symbol)
		}
		strikeMinimum, strikeMaximum := canonicalClose*cfg.MinMoneyness, canonicalClose*cfg.MaxMoneyness
		filter.ExpirationDateGTE = dayOnly(cfg.SessionDate).AddDate(0, 0, cfg.MinDTE)
		filter.ExpirationDateLTE = dayOnly(cfg.SessionDate).AddDate(0, 0, cfg.MaxDTE)
		filter.StrikePriceGTE, filter.StrikePriceLTE = &strikeMinimum, &strikeMaximum
		filter.Limit = minInt(filter.Limit, cfg.MaxCandidates)
		filter.MaxPages = minInt(filter.MaxPages, (cfg.MaxCandidates+filter.Limit-1)/filter.Limit)
		item.AcquisitionExpirationStart = filter.ExpirationDateGTE.Format("2006-01-02")
		item.AcquisitionExpirationEnd = filter.ExpirationDateLTE.Format("2006-01-02")
		item.AcquisitionStrikeMinimum = strikeMinimum
		item.AcquisitionStrikeMaximum = strikeMaximum
	}
	providerBatch, err := provider.ListOptionChainSnapshotFilteredWithMetadata(ctx, symbol, filter)
	if err != nil {
		return item, fmt.Errorf("fetch option chain for %s: %w", symbol, err)
	}
	providerRecords := providerBatch.Records
	item.ProviderRequestIDs = providerBatch.ProviderRequestIDs
	item.ProviderPagesFetched = providerBatch.PagesFetched
	item.ProviderPaginationComplete = providerBatch.PaginationComplete
	if !cfg.SessionDate.IsZero() && len(providerRecords) > cfg.MaxCandidates {
		providerRecords = providerRecords[:cfg.MaxCandidates]
	}
	item.Fetched = len(providerRecords)
	chainRecords := []storage.MarketOpsOptionsChainRecord{}
	sawRequestedSessionActivity := cfg.SessionDate.IsZero()
	for _, providerRecord := range providerRecords {
		if !cfg.SessionDate.IsZero() {
			if dayOnly(providerRecord.ObservationDate).After(dayOnly(cfg.SessionDate)) {
				return item, fmt.Errorf("provider contract activity date %s is after requested capture session %s", dayOnly(providerRecord.ObservationDate).Format("2006-01-02"), dayOnly(cfg.SessionDate).Format("2006-01-02"))
			}
			if dayOnly(providerRecord.ObservationDate).Equal(dayOnly(cfg.SessionDate)) {
				sawRequestedSessionActivity = true
			}
			providerRecord.ObservationDate = dayOnly(cfg.SessionDate)
			value := canonicalClose
			providerRecord.UnderlyingClose = &value
		}
		chainRecord, err := marketopsoptions.ChainRecordFromMassiveSnapshot(cfg.TenantID, cfg.SourceID, cfg.RunID, providerRecord)
		if err != nil {
			item.Skipped++
			continue
		}
		chainRecords = append(chainRecords, chainRecord)
		item.Converted++
	}
	if len(providerRecords) > 0 && !sawRequestedSessionActivity {
		return item, fmt.Errorf("provider snapshot has no contract activity on requested capture session %s", dayOnly(cfg.SessionDate).Format("2006-01-02"))
	}
	if !cfg.SessionDate.IsZero() {
		for _, record := range chainRecords {
			if !dayOnly(record.TradeDate).Equal(dayOnly(cfg.SessionDate)) {
				return item, fmt.Errorf("provider session %s does not match requested session-date %s", dayOnly(record.TradeDate).Format("2006-01-02"), dayOnly(cfg.SessionDate).Format("2006-01-02"))
			}
		}
	}
	readinessSession := cfg.SessionDate
	if readinessSession.IsZero() && len(chainRecords) > 0 {
		readinessSession = chainRecords[0].TradeDate
	}
	retainedEvidence := chainRecords
	if !cfg.SessionDate.IsZero() {
		retainedEvidence = marketopsoptions.SelectRequiredSurfaceEvidence(readinessSession, chainRecords)
	}
	item.SelectedEvidence = len(retainedEvidence)
	item.DiscardedCandidates = len(chainRecords) - len(retainedEvidence)
	readiness := marketopsoptions.AssessAnalyticsReadiness(readinessSession, retainedEvidence)
	item.AnalyticsReady = readiness.Ready
	item.RequiredSurfaceCells = readiness.RequiredSurfaceCells
	item.UsableIVCount = readiness.UsableIVCount
	item.UsableGreeksCount = readiness.UsableGreeksCount
	item.OpenInterestCount = readiness.OpenInterestCount
	item.UnderlyingPriceCount = readiness.UnderlyingPriceCount
	item.QualityReasons = append([]string{}, readiness.QualityReasons...)
	if !providerBatch.PaginationComplete {
		item.QualityReasons = append(item.QualityReasons, "provider_candidate_window_incomplete")
		sort.Strings(item.QualityReasons)
	}
	if !cfg.DryRun {
		for _, chainRecord := range retainedEvidence {
			if err := repo.UpsertMarketOpsOptionsChain(ctx, chainRecord); err != nil {
				return item, err
			}
			item.ChainUpserted++
		}
	}
	rows := chainRecords
	if cfg.SessionDate.IsZero() {
		rows, err = repo.ListMarketOpsOptionsChain(ctx, storage.MarketOpsOptionsChainFilter{TenantID: cfg.TenantID, Symbol: symbol, Limit: cfg.ChainScanLimit})
		if err != nil {
			return item, err
		}
		if cfg.DryRun {
			rows = append(rows, chainRecords...)
		}
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

func resolveSameSessionUnderlyingPrice(ctx context.Context, repo repository, cfg cliConfig, symbol string) (float64, string, error) {
	events, err := repo.ListMarketOpsBacktestNormalizedEvents(ctx, storage.MarketOpsBacktestEventFilter{
		TenantID: cfg.TenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance",
		SourceAdapter: "market_data.massive", Dataset: "equity_eod_prices", Symbols: []string{symbol},
		WindowStart: dayOnly(cfg.SessionDate), WindowEnd: dayOnly(cfg.SessionDate).AddDate(0, 0, 1), Limit: 10,
	})
	if err != nil {
		return 0, "", fmt.Errorf("resolve same-session underlying price for %s: %w", symbol, err)
	}
	var closeValue float64
	eventID := ""
	for _, event := range events {
		payload := map[string]any{}
		if json.Unmarshal(event.NormalizedPayload, &payload) != nil {
			continue
		}
		value, ok := payload["close"].(float64)
		if ok && value > 0 {
			closeValue = value
			eventID = event.EventID
		}
	}
	return closeValue, eventID, nil
}

func persistCapture(ctx context.Context, repo repository, cfg cliConfig, item symbolMetrics, startedAt time.Time) error {
	if cfg.SessionDate.IsZero() || cfg.DryRun {
		return nil
	}
	reasonsJSON, _ := json.Marshal(item.QualityReasons)
	metricsJSON, _ := json.Marshal(item)
	return repo.UpsertMarketOpsOptionsCapture(ctx, storage.MarketOpsOptionsCaptureRecord{
		CaptureID:            item.CaptureID,
		TenantID:             cfg.TenantID,
		Symbol:               item.Symbol,
		SessionDate:          dayOnly(cfg.SessionDate),
		Provider:             "massive",
		SourceID:             cfg.SourceID,
		RunID:                cfg.RunID,
		Status:               item.CaptureStatus,
		AnalyticsReady:       item.AnalyticsReady,
		ContractCount:        item.Converted,
		UsableIVCount:        item.UsableIVCount,
		UsableGreeksCount:    item.UsableGreeksCount,
		OpenInterestCount:    item.OpenInterestCount,
		RequiredSurfaceCells: item.RequiredSurfaceCells,
		QualityReasonsJSON:   reasonsJSON,
		MetricsJSON:          metricsJSON,
		ErrorMessage:         item.Error,
		AttemptCount:         1,
		StartedAt:            startedAt,
		CompletedAt:          time.Now().UTC(),
	})
}

func optionsCaptureID(tenantID, sourceID, symbol string, sessionDate time.Time) string {
	identity := strings.Join([]string{
		strings.TrimSpace(tenantID),
		strings.TrimSpace(sourceID),
		strings.ToUpper(strings.TrimSpace(symbol)),
		dayOnly(sessionDate).Format("2006-01-02"),
	}, "|")
	sum := sha256.Sum256([]byte(identity))
	return "optcap_" + hex.EncodeToString(sum[:12])
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
	sessionDate := ""
	flag.StringVar(&cfg.TenantID, "tenant-id", "tenant-local", "tenant id")
	flag.StringVar(&symbols, "symbols", "", "comma-separated symbols; when empty, resolve from the asset universe")
	flag.StringVar(&cfg.UniverseGroup, "universe-group", "top50_megacap", "asset universe group used when symbols are omitted")
	flag.StringVar(&cfg.SourceID, "source-id", "src-massive", "source id")
	flag.StringVar(&cfg.RunID, "run-id", "", "coverage run id")
	flag.IntVar(&cfg.Limit, "limit", 250, "Massive option-chain page size per symbol")
	flag.IntVar(&cfg.MaxPages, "max-pages", 1, "maximum Massive option-chain pages to fetch per symbol")
	flag.IntVar(&cfg.MaxCandidates, "max-candidates", 500, "hard maximum provider candidates per session and symbol")
	flag.IntVar(&cfg.MinDTE, "min-dte", 14, "minimum expiration DTE for session acquisition")
	flag.IntVar(&cfg.MaxDTE, "max-dte", 120, "maximum expiration DTE for session acquisition")
	flag.Float64Var(&cfg.MinMoneyness, "min-moneyness", 0.70, "minimum strike to canonical-close ratio for session acquisition")
	flag.Float64Var(&cfg.MaxMoneyness, "max-moneyness", 1.30, "maximum strike to canonical-close ratio for session acquisition")
	flag.IntVar(&cfg.MaxSymbols, "max-symbols", 3, "maximum symbols to process")
	flag.IntVar(&cfg.WindowDays, "window-days", 10, "calendar-day lookback used for distribution snapshots")
	flag.IntVar(&cfg.ChainScanLimit, "chain-scan-limit", 5000, "maximum persisted chain rows to scan per symbol")
	flag.IntVar(&cfg.DistributionLimit, "distribution-limit", 100, "maximum distribution snapshots to materialize per symbol")
	flag.StringVar(&sessionDate, "session-date", "", "expected provider session date in YYYY-MM-DD; enables the G142 capture ledger")
	flag.BoolVar(&cfg.SkipComplete, "skip-complete", true, "skip an existing analytics-ready capture for the same symbol and session")
	flag.BoolVar(&cfg.ContinueOnError, "continue-on-error", true, "continue with remaining symbols after a failed capture")
	flag.IntVar(&cfg.MaxRetries, "max-retries", 1, "maximum retries per failed symbol")
	flag.DurationVar(&cfg.RetryBackoff, "retry-backoff", time.Second, "base retry backoff")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "fetch and derive without writing storage")
	flag.Parse()
	cfg.Symbols = parseSymbols(symbols)
	if strings.TrimSpace(sessionDate) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(sessionDate))
		if err != nil {
			return cliConfig{}, fmt.Errorf("invalid session-date: %w", err)
		}
		cfg.SessionDate = parsed.UTC()
	}
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
	if cfg.MaxCandidates <= 0 || cfg.MaxCandidates > 1000 {
		cfg.MaxCandidates = 500
	}
	if cfg.MinDTE <= 0 {
		cfg.MinDTE = 14
	}
	if cfg.MaxDTE <= 0 {
		cfg.MaxDTE = 120
	}
	if cfg.MinMoneyness <= 0 {
		cfg.MinMoneyness = 0.70
	}
	if cfg.MaxMoneyness <= 0 {
		cfg.MaxMoneyness = 1.30
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
	if cfg.MaxRetries < 0 || cfg.MaxRetries > 5 {
		cfg.MaxRetries = 1
	}
	if cfg.RetryBackoff <= 0 || cfg.RetryBackoff > time.Minute {
		cfg.RetryBackoff = time.Second
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
	if cfg.MinDTE < 7 || cfg.MaxDTE > 180 || cfg.MinDTE >= cfg.MaxDTE {
		return errors.New("session DTE bounds must satisfy 7 <= min-dte < max-dte <= 180")
	}
	if cfg.MinMoneyness <= 0 || cfg.MinMoneyness >= cfg.MaxMoneyness {
		return errors.New("session moneyness bounds must be positive and increasing")
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

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
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

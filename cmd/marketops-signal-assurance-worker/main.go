package main

import (
	"context"
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
	"github.com/lukebabs/signalops/internal/marketops/signalassurance"
	"github.com/lukebabs/signalops/internal/storage"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
)

type repository interface {
	storage.SignalAssuranceQueryRepository
	storage.SignalAssuranceWriteRepository
	storage.MarketOpsOutcomeWriteRepository
	storage.SignalAssuranceAggregateRepository
	ListMarketOpsBacktestNormalizedEvents(context.Context, storage.MarketOpsBacktestEventFilter) ([]storage.NormalizedEventLedgerRecord, error)
}
type workerConfig struct {
	TenantID, Mode, RunID, Benchmark string
	AsOf                             time.Time
	Limit                            int
	DryRun                           bool
}
type workerReport struct {
	TenantID    string `json:"tenant_id"`
	AsOf        string `json:"as_of"`
	Evaluated   int    `json:"evaluated"`
	Incomplete  int    `json:"incomplete"`
	Transitions int    `json:"transitions"`
	DryRun      bool   `json:"dry_run"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("signal assurance worker failed", "error", err)
		os.Exit(1)
	}
}
func run(logger *slog.Logger) error {
	app := config.Load()
	databaseURL := app.DatabaseURL
	temporalDatabaseURL := app.TemporalDatabaseURL
	if strings.TrimSpace(app.MarketOpsDatabaseURL) != "" {
		databaseURL = app.MarketOpsDatabaseURL
		temporalDatabaseURL = app.MarketOpsTemporalDatabaseURL
		logger.Info("SAF evaluation writes are routed to the dedicated MarketOps data boundary")
	}
	if strings.TrimSpace(databaseURL) == "" || strings.TrimSpace(temporalDatabaseURL) == "" {
		return errors.New("SIGNALOPS_DATABASE_URL and SIGNALOPS_TEMPORAL_DATABASE_URL are required")
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	repo, err := postgresstorage.OpenWithTemporal(context.Background(), databaseURL, temporalDatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	report, err := evaluate(context.Background(), repo, cfg)
	if err != nil {
		return err
	}
	encoded, _ := json.Marshal(report)
	fmt.Println(string(encoded))
	logger.Info("signal assurance evaluation complete", "evaluated", report.Evaluated, "transitions", report.Transitions)
	return nil
}
func loadConfig() (workerConfig, error) {
	var asOf string
	cfg := workerConfig{}
	flag.StringVar(&cfg.TenantID, "tenant-id", "tenant-local", "tenant id")
	flag.StringVar(&cfg.Mode, "mode", storage.SignalAssuranceModeLive, "LIVE, BACKTEST, or RESEARCH")
	flag.StringVar(&cfg.RunID, "run-id", "", "required for BACKTEST and RESEARCH")
	flag.StringVar(&asOf, "as-of", time.Now().UTC().Format("2006-01-02"), "evaluation session date")
	flag.StringVar(&cfg.Benchmark, "benchmark", "SPY", "canonical benchmark symbol")
	flag.IntVar(&cfg.Limit, "limit", 500, "maximum active assertions")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "calculate without persistence")
	flag.Parse()
	var err error
	cfg.AsOf, err = time.Parse("2006-01-02", strings.TrimSpace(asOf))
	cfg.TenantID = strings.TrimSpace(cfg.TenantID)
	cfg.Mode = strings.ToUpper(strings.TrimSpace(cfg.Mode))
	cfg.RunID = strings.TrimSpace(cfg.RunID)
	cfg.Benchmark = strings.ToUpper(strings.TrimSpace(cfg.Benchmark))
	if err != nil {
		return cfg, err
	}
	if cfg.TenantID == "" || cfg.Benchmark == "" || cfg.Limit < 1 || cfg.Limit > 1000 {
		return cfg, errors.New("invalid worker configuration")
	}
	if cfg.Mode != storage.SignalAssuranceModeLive && cfg.Mode != storage.SignalAssuranceModeBacktest && cfg.Mode != storage.SignalAssuranceModeResearch {
		return cfg, errors.New("invalid evaluation mode")
	}
	if (cfg.Mode == storage.SignalAssuranceModeLive && cfg.RunID != "") || (cfg.Mode != storage.SignalAssuranceModeLive && cfg.RunID == "") {
		return cfg, errors.New("invalid mode/run namespace")
	}
	return cfg, nil
}
func evaluate(ctx context.Context, repo repository, cfg workerConfig) (workerReport, error) {
	report := workerReport{TenantID: cfg.TenantID, AsOf: cfg.AsOf.Format("2006-01-02"), DryRun: cfg.DryRun}
	assertions, err := repo.ListSignalAssuranceAssertions(ctx, storage.SignalAssuranceAssertionFilter{TenantID: cfg.TenantID, State: storage.SignalAssertionActive, EvaluationMode: cfg.Mode, Limit: cfg.Limit})
	if err != nil {
		return report, err
	}
	for _, assertion := range assertions {
		if assertion.EvaluationRunID != cfg.RunID {
			continue
		}
		contract, err := repo.GetSignalValidationContract(ctx, assertion.ValidationContractID)
		if err != nil {
			return report, err
		}
		events, err := repo.ListMarketOpsBacktestNormalizedEvents(ctx, storage.MarketOpsBacktestEventFilter{TenantID: cfg.TenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", Dataset: "equity_eod_prices", Symbols: []string{assertion.Symbol, cfg.Benchmark}, WindowStart: assertion.ConfirmedAt.AddDate(0, 0, -5), WindowEnd: cfg.AsOf.AddDate(0, 0, 1), Limit: 1000})
		if err != nil {
			return report, err
		}
		asset, benchmark, tradingDays, snapshot := marketInput(events, assertion.Symbol, cfg.Benchmark, assertion.ConfirmedAt, cfg.AsOf)
		previous, err := repo.ListSignalAssuranceEvaluations(ctx, storage.SignalAssuranceEvaluationFilter{AssertionID: assertion.AssertionID, Limit: 1})
		if err != nil {
			return report, err
		}
		var last *storage.SignalAssertionEvaluationRecord
		if len(previous) > 0 {
			last = &previous[0]
		}
		result, err := signalassurance.Evaluate(assertion, contract, last, signalassurance.EvaluationMarketState{AsOf: cfg.AsOf, AssetPrice: asset, BenchmarkPrice: benchmark, TradingDaysActive: tradingDays, InputSnapshotJSON: snapshot})
		if err != nil {
			return report, err
		}
		report.Evaluated++
		if result.Persistence.Evaluation.InputCompleteness == storage.SignalAssuranceInputIncomplete {
			report.Incomplete++
		}
		if result.Persistence.NextState != result.Persistence.PreviousState {
			report.Transitions++
		}
		if !cfg.DryRun {
			persisted, inserted, err := repo.PersistSignalAssuranceEvaluation(ctx, result.Persistence)
			if err != nil {
				return report, err
			}
			if inserted {
				if projection := signalassurance.CompatibleOutcomeProjection(assertion, persisted, result.Persistence.NextState); projection != nil {
					if err := repo.UpsertMarketOpsSignalOutcome(ctx, *projection); err != nil {
						return report, err
					}
				}
			}
		}
	}
	if !cfg.DryRun {
		if err := repo.RefreshSignalAssuranceEffectiveness(ctx, cfg.TenantID, cfg.Mode, cfg.AsOf, signalassurance.EvaluationEngineVersion); err != nil {
			return report, err
		}
	}
	return report, nil
}

type closePoint struct {
	Session time.Time
	Price   float64
	EventID string
}

func marketInput(events []storage.NormalizedEventLedgerRecord, symbol, benchmark string, confirmed, asOf time.Time) (*float64, *float64, int, []byte) {
	points := map[string][]closePoint{}
	for _, event := range events {
		if event.ObservationTime.After(asOf) || event.ProcessingTime.After(asOf) {
			continue
		}
		var payload struct {
			Symbol string  `json:"symbol"`
			Ticker string  `json:"ticker"`
			Close  float64 `json:"close"`
		}
		if json.Unmarshal(event.NormalizedPayload, &payload) != nil || payload.Close <= 0 {
			continue
		}
		ticker := strings.ToUpper(strings.TrimSpace(payload.Symbol))
		if ticker == "" {
			ticker = strings.ToUpper(strings.TrimSpace(payload.Ticker))
		}
		if ticker == "" {
			continue
		}
		points[ticker] = append(points[ticker], closePoint{Session: date(event.ObservationTime), Price: payload.Close, EventID: event.EventID})
	}
	latest := func(ticker string) *float64 {
		values := points[ticker]
		sort.Slice(values, func(i, j int) bool { return values[i].Session.Before(values[j].Session) })
		var out *float64
		for _, value := range values {
			if !value.Session.After(asOf) {
				v := value.Price
				out = &v
			}
		}
		return out
	}
	sessions := map[string]bool{}
	for _, value := range points[strings.ToUpper(symbol)] {
		if !value.Session.Before(date(confirmed)) && !value.Session.After(asOf) {
			sessions[value.Session.Format("2006-01-02")] = true
		}
	}
	snapshot, _ := json.Marshal(map[string]any{"price_source": "normalized_event_ledger", "asset_symbol": strings.ToUpper(symbol), "benchmark_symbol": strings.ToUpper(benchmark), "as_of": asOf.Format("2006-01-02"), "asset_session_count": len(sessions)})
	return latest(strings.ToUpper(symbol)), latest(strings.ToUpper(benchmark)), len(sessions), snapshot
}
func date(value time.Time) time.Time {
	return time.Date(value.UTC().Year(), value.UTC().Month(), value.UTC().Day(), 0, 0, 0, 0, time.UTC)
}

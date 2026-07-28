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
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/config"
	"github.com/lukebabs/signalops/internal/storage"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
)

type cliConfig struct {
	CampaignID, TenantID, UniverseGroup, RequestedBy              string
	WindowStart, WindowEnd                                        time.Time
	MaxSymbols, ChunkCalendarDays, MaxProviderRequests, MaxEvents int
	ExecutePull, AcknowledgeWrites                                bool
}

type campaignResult struct {
	Campaign         storage.MarketOpsAlgorithmEvaluationBackfillCampaignRecord `json:"campaign"`
	PlannedChildren  int                                                        `json:"planned_children"`
	ExecutedChildren int                                                        `json:"executed_children"`
}

type child struct {
	id         string
	symbols    []string
	start, end time.Time
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("algorithm evaluation backfill campaign failed", "error", err)
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
	assets, err := repo.ListMarketOpsAssets(ctx, cfg.TenantID, cfg.UniverseGroup, true, cfg.MaxSymbols+1)
	if err != nil {
		return err
	}
	if len(assets) > cfg.MaxSymbols {
		return fmt.Errorf("active universe exceeds max-symbols %d", cfg.MaxSymbols)
	}
	symbols := []string{}
	for _, asset := range assets {
		symbols = append(symbols, strings.ToUpper(asset.Ticker))
	}
	if len(symbols) == 0 {
		return errors.New("backfill campaign resolved no active symbols")
	}
	children := planChildren(cfg, symbols)
	ids := make([]string, 0, len(children))
	for _, item := range children {
		ids = append(ids, item.id)
	}
	now := time.Now().UTC()
	status := storage.MarketOpsAlgorithmBackfillStatusPlanned
	if cfg.ExecutePull {
		status = storage.MarketOpsAlgorithmBackfillStatusRunning
	}
	completed := map[string]bool{}
	if existing, err := repo.GetMarketOpsAlgorithmEvaluationBackfillCampaign(ctx, cfg.TenantID, cfg.CampaignID); err == nil {
		completed = completedChildRunIDs(existing.CoverageJSON)
		if existing.Status == storage.MarketOpsAlgorithmBackfillStatusSucceeded {
			encoded, _ := json.MarshalIndent(campaignResult{Campaign: existing, PlannedChildren: len(children), ExecutedChildren: len(completed)}, "", "  ")
			fmt.Println(string(encoded))
			return nil
		}
	} else if !errors.Is(err, storage.ErrNotFound) {
		return err
	}
	record := storage.MarketOpsAlgorithmEvaluationBackfillCampaignRecord{CampaignID: cfg.CampaignID, TenantID: cfg.TenantID, UniverseGroup: cfg.UniverseGroup, WindowStart: cfg.WindowStart, WindowEnd: cfg.WindowEnd, Status: status, ParametersJSON: mustJSON(map[string]any{"dataset": "equity", "max_symbols": cfg.MaxSymbols, "chunk_calendar_days": cfg.ChunkCalendarDays, "max_provider_requests": cfg.MaxProviderRequests, "max_events": cfg.MaxEvents, "execute_pull": cfg.ExecutePull}), CoverageJSON: campaignCoverage(symbols, len(children), completed), ChildRunIDs: ids, RequestedBy: cfg.RequestedBy}
	if cfg.ExecutePull {
		record.StartedAt = &now
	}
	if err := repo.UpsertMarketOpsAlgorithmEvaluationBackfillCampaign(ctx, record); err != nil {
		return err
	}
	result := campaignResult{Campaign: record, PlannedChildren: len(children), ExecutedChildren: len(completed)}
	if !cfg.ExecutePull {
		encoded, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(encoded))
		logger.Info("algorithm evaluation backfill campaign planned", "campaign_id", cfg.CampaignID, "children", len(children))
		return nil
	}
	for _, item := range children {
		if completed[item.id] {
			continue
		}
		if err := executePull(ctx, cfg, item); err != nil {
			finished := time.Now().UTC()
			record.Status = storage.MarketOpsAlgorithmBackfillStatusFailed
			record.ErrorMessage = err.Error()
			record.CompletedAt = &finished
			_ = repo.UpsertMarketOpsAlgorithmEvaluationBackfillCampaign(ctx, record)
			return err
		}
		if err := awaitNormalizedCoverage(ctx, repo, cfg, item); err != nil {
			finished := time.Now().UTC()
			record.Status = storage.MarketOpsAlgorithmBackfillStatusFailed
			record.ErrorMessage = err.Error()
			record.CompletedAt = &finished
			_ = repo.UpsertMarketOpsAlgorithmEvaluationBackfillCampaign(ctx, record)
			return err
		}
		completed[item.id] = true
		result.ExecutedChildren = len(completed)
		record.CoverageJSON = campaignCoverage(symbols, len(children), completed)
		if err := repo.UpsertMarketOpsAlgorithmEvaluationBackfillCampaign(ctx, record); err != nil {
			return err
		}
	}
	finished := time.Now().UTC()
	record.Status = storage.MarketOpsAlgorithmBackfillStatusSucceeded
	record.CompletedAt = &finished
	if err := repo.UpsertMarketOpsAlgorithmEvaluationBackfillCampaign(ctx, record); err != nil {
		return err
	}
	result.Campaign = record
	encoded, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(encoded))
	logger.Info("algorithm evaluation backfill campaign completed", "campaign_id", cfg.CampaignID, "children", result.ExecutedChildren)
	return nil
}
func loadConfig(args []string) (cliConfig, error) {
	now := time.Now().UTC()
	cfg := cliConfig{}
	var start, end string
	flags := flag.NewFlagSet("marketops-algorithm-evaluation-backfill", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.StringVar(&cfg.CampaignID, "campaign-id", "", "stable campaign id")
	flags.StringVar(&cfg.TenantID, "tenant-id", "tenant-local", "tenant id")
	flags.StringVar(&cfg.UniverseGroup, "universe-group", "top50_megacap", "active asset universe")
	flags.StringVar(&start, "window-start", now.AddDate(0, 0, -400).Format("2006-01-02"), "inclusive historical start")
	flags.StringVar(&end, "window-end", now.AddDate(0, 0, 1).Format("2006-01-02"), "exclusive historical end")
	flags.IntVar(&cfg.MaxSymbols, "max-symbols", 50, "maximum active symbols (1-50)")
	flags.IntVar(&cfg.ChunkCalendarDays, "chunk-calendar-days", 20, "calendar days per pull child (1-20)")
	flags.IntVar(&cfg.MaxProviderRequests, "max-provider-requests", 250, "hard provider request budget per child")
	flags.IntVar(&cfg.MaxEvents, "max-events", 250, "hard built/published event budget per child")
	flags.BoolVar(&cfg.ExecutePull, "execute-pull", false, "invoke the canonical massive puller for planned children")
	flags.BoolVar(&cfg.AcknowledgeWrites, "acknowledge-writes", false, "required with execute-pull")
	flags.StringVar(&cfg.RequestedBy, "requested-by", "operator-local", "requesting operator")
	if err := flags.Parse(args); err != nil {
		return cfg, err
	}
	var err error
	cfg.WindowStart, err = time.Parse("2006-01-02", strings.TrimSpace(start))
	if err != nil {
		return cfg, fmt.Errorf("window-start: %w", err)
	}
	cfg.WindowEnd, err = time.Parse("2006-01-02", strings.TrimSpace(end))
	if err != nil {
		return cfg, fmt.Errorf("window-end: %w", err)
	}
	cfg.CampaignID = strings.TrimSpace(cfg.CampaignID)
	cfg.TenantID = strings.TrimSpace(cfg.TenantID)
	cfg.UniverseGroup = strings.TrimSpace(cfg.UniverseGroup)
	cfg.RequestedBy = strings.TrimSpace(cfg.RequestedBy)
	if cfg.CampaignID == "" {
		cfg.CampaignID = "algeval_backfill_" + randomHex(12)
	}
	if cfg.TenantID == "" || cfg.UniverseGroup == "" || !cfg.WindowEnd.After(cfg.WindowStart) {
		return cfg, errors.New("tenant, universe, and date range are required")
	}
	if cfg.MaxSymbols < 1 || cfg.MaxSymbols > 50 || cfg.ChunkCalendarDays < 1 || cfg.ChunkCalendarDays > 20 || cfg.MaxProviderRequests < 1 || cfg.MaxEvents < 1 {
		return cfg, errors.New("backfill campaign caps are invalid")
	}
	if cfg.ExecutePull && !cfg.AcknowledgeWrites {
		return cfg, errors.New("--acknowledge-writes is required with --execute-pull")
	}
	return cfg, nil
}
func planChildren(cfg cliConfig, symbols []string) []child {
	out := []child{}
	for start := cfg.WindowStart; start.Before(cfg.WindowEnd); start = start.AddDate(0, 0, cfg.ChunkCalendarDays) {
		end := start.AddDate(0, 0, cfg.ChunkCalendarDays)
		if end.After(cfg.WindowEnd) {
			end = cfg.WindowEnd
		}
		for index := 0; index < len(symbols); index += 10 {
			stop := index + 10
			if stop > len(symbols) {
				stop = len(symbols)
			}
			id := fmt.Sprintf("%s_%s_%s_%02d", cfg.CampaignID, start.Format("20060102"), end.AddDate(0, 0, -1).Format("20060102"), index/10+1)
			out = append(out, child{id: id, symbols: append([]string(nil), symbols[index:stop]...), start: start, end: end})
		}
	}
	return out
}
func executePull(ctx context.Context, cfg cliConfig, item child) error {
	args := []string{"--start-date", item.start.Format("2006-01-02"), "--end-date", item.end.AddDate(0, 0, -1).Format("2006-01-02"), "--symbols", strings.Join(item.symbols, ","), "--datasets", "equity", "--max-observation-days", fmt.Sprint(cfg.ChunkCalendarDays), "--max-companies", fmt.Sprint(len(item.symbols)), "--max-provider-requests", fmt.Sprint(cfg.MaxProviderRequests), "--max-events-built", fmt.Sprint(cfg.MaxEvents), "--max-events-published", fmt.Sprint(cfg.MaxEvents), "--acknowledge-writes"}
	pullerBin := strings.TrimSpace(os.Getenv("SIGNALOPS_MASSIVE_PULLER_BIN"))
	if pullerBin == "" {
		pullerBin = "signalops-massive-puller"
	}
	command := exec.CommandContext(ctx, pullerBin, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("backfill child %s: %w", item.id, err)
	}
	return nil
}

func awaitNormalizedCoverage(ctx context.Context, repo interface {
	ListMarketOpsBacktestNormalizedEvents(context.Context, storage.MarketOpsBacktestEventFilter) ([]storage.NormalizedEventLedgerRecord, error)
}, cfg cliConfig, item child) error {
	deadline := time.Now().UTC().Add(90 * time.Second)
	expected := weekdayCount(item.start, item.end) * len(item.symbols)
	required := (expected*80 + 99) / 100
	if required < len(item.symbols) {
		required = len(item.symbols)
	}
	for {
		records, err := repo.ListMarketOpsBacktestNormalizedEvents(ctx, storage.MarketOpsBacktestEventFilter{
			TenantID: cfg.TenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance",
			Dataset: "equity_eod_prices", Symbols: item.symbols, WindowStart: item.start, WindowEnd: item.end, Limit: 5000,
		})
		if err != nil {
			return fmt.Errorf("query normalized coverage for child %s: %w", item.id, err)
		}
		covered := map[string]bool{}
		for _, record := range records {
			var payload map[string]any
			if json.Unmarshal(record.NormalizedPayload, &payload) != nil {
				continue
			}
			symbol, _ := payload["symbol"].(string)
			if strings.TrimSpace(symbol) != "" {
				covered[strings.ToUpper(strings.TrimSpace(symbol))+"|"+record.ObservationTime.UTC().Format("2006-01-02")] = true
			}
		}
		if len(covered) >= required {
			return nil
		}
		if time.Now().UTC().After(deadline) {
			return fmt.Errorf("normalized coverage for child %s is incomplete: got %d, require at least %d distinct symbol/date rows", item.id, len(covered), required)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

func weekdayCount(start, end time.Time) int {
	count := 0
	for day := start.UTC(); day.Before(end.UTC()); day = day.AddDate(0, 0, 1) {
		if day.Weekday() != time.Saturday && day.Weekday() != time.Sunday {
			count++
		}
	}
	return count
}

func completedChildRunIDs(raw []byte) map[string]bool {
	var coverage struct {
		CompletedChildRunIDs []string `json:"completed_child_run_ids"`
	}
	if json.Unmarshal(raw, &coverage) != nil {
		return map[string]bool{}
	}
	out := map[string]bool{}
	for _, id := range coverage.CompletedChildRunIDs {
		if strings.TrimSpace(id) != "" {
			out[id] = true
		}
	}
	return out
}

func campaignCoverage(symbols []string, planned int, completed map[string]bool) []byte {
	ids := make([]string, 0, len(completed))
	for id := range completed {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return mustJSON(map[string]any{"symbols": symbols, "planned_children": planned, "completed_children": len(ids), "completed_child_run_ids": ids, "historical_options": "not_requested"})
}

func mustJSON(value any) []byte { encoded, _ := json.Marshal(value); return encoded }
func randomHex(length int) string {
	value := make([]byte, length)
	_, _ = rand.Read(value)
	return hex.EncodeToString(value)
}

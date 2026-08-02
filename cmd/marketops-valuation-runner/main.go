// marketops-valuation-runner writes weekly, research-only VC and DOSM results.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/adapters/marketdata/fmp"
	"github.com/lukebabs/signalops/internal/adapters/marketdata/massive"
	"github.com/lukebabs/signalops/internal/config"
	"github.com/lukebabs/signalops/internal/marketops/valuation"
	"github.com/lukebabs/signalops/internal/storage"
	postgres "github.com/lukebabs/signalops/internal/storage/postgres"
)

const (
	vcID   = "signalops.algorithms.valuation_composite_v3"
	dosmID = "signalops.algorithms.distressed_opportunity_scoring_v3"
)

type candidate struct {
	asset        storage.MarketOpsAssetRecord
	fundamentals fmp.FundamentalSnapshot
	price        float64
	input        valuation.Input
	provider     string
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(context.Background(), logger); err != nil {
		logger.Error("valuation runner failed", "error", err)
		os.Exit(1)
	}
}
func run(ctx context.Context, logger *slog.Logger) error {
	app := config.Load()
	tenant := flag.String("tenant-id", "tenant-local", "tenant")
	group := flag.String("universe-group", "top50_megacap", "universe")
	symbols := flag.String("symbols", "", "comma-separated symbols")
	dateValue := flag.String("session-date", "", "completed YYYY-MM-DD session")
	dry := flag.Bool("dry-run", false, "calculate only")
	fmpBudget := flag.Int("fmp-max-requests", 240, "maximum FMP requests per daily allowance")
	flag.Parse()
	if app.DatabaseURL == "" {
		return fmt.Errorf("SIGNALOPS_DATABASE_URL is required")
	}
	session := lastSession()
	if strings.TrimSpace(*dateValue) != "" {
		var err error
		session, err = time.Parse("2006-01-02", *dateValue)
		if err != nil {
			return err
		}
	}
	repo, err := postgres.Open(ctx, app.DatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	priceClient, err := massive.NewClient(massive.LoadClientConfigFromEnv())
	if err != nil {
		return err
	}
	fundamentalsClient, err := fmp.NewClient(fmp.LoadClientConfigFromEnv())
	if err != nil {
		return err
	}
	if *fmpBudget < 3 {
		return fmt.Errorf("fmp-max-requests must be at least 3")
	}
	assets, err := repo.ListMarketOpsAssets(ctx, *tenant, *group, true, 200)
	if err != nil {
		return err
	}
	if strings.TrimSpace(*symbols) != "" {
		wanted := map[string]bool{}
		for _, symbol := range strings.Split(*symbols, ",") {
			wanted[strings.ToUpper(strings.TrimSpace(symbol))] = true
		}
		filtered := assets[:0]
		for _, asset := range assets {
			if wanted[strings.ToUpper(asset.Ticker)] {
				filtered = append(filtered, asset)
			}
		}
		assets = filtered
	}
	priorities := map[string]int{}
	for _, asset := range assets {
		if state, stateErr := repo.GetMarketOpsFMPPollState(ctx, *tenant, asset.Ticker); stateErr == nil && state.Status == "deferred_budget" {
			priorities[strings.ToUpper(asset.Ticker)] = -1
		}
	}
	sort.SliceStable(assets, func(i, j int) bool {
		return priorities[strings.ToUpper(assets[i].Ticker)] < priorities[strings.ToUpper(assets[j].Ticker)]
	})
	candidates := []candidate{}
	cacheHits, refreshed, deferred, fmpFailures := 0, 0, 0, 0
	for _, asset := range assets {
		bar, err := priceClient.GetEquityDailyBar(ctx, asset.Ticker, session)
		if err != nil || bar.Close == nil {
			logger.Warn("close unavailable", "symbol", asset.Ticker, "error", err)
			continue
		}
		provider := "fmp"
		if cached, ok, _ := cachedFundamentals(ctx, repo, *tenant, asset.Ticker); ok {
			cacheHits++
			candidates = append(candidates, candidate{asset: asset, fundamentals: cached, price: *bar.Close, input: valuation.Input{Ticker: asset.Ticker, Price: *bar.Close, MarketCap: cached.MarketCap, EnterpriseValue: cached.EnterpriseValue, RevenueTTM: cached.RevenueTTM, Revenue3YAgo: cached.Revenue3YAgo, NetIncomeGAAPTTM: cached.NetIncomeTTM, EBITDAProviderTTM: cached.EBITDATTM, OperatingIncomeTTM: cached.OperatingIncomeTTM, OperatingCashFlowTTM: cached.OperatingCashFlowTTM, CapitalExpendituresTTM: cached.CapitalExpendituresTTM, TotalDebt: cached.TotalDebt, CashAndEquivalents: cached.Cash, ShareholdersEquity: cached.Equity, InvestedCapital: cached.InvestedCapital, FinancialAgeDays: int(session.Sub(cached.FilingDate).Hours() / 24), EffectiveTaxRate: taxRatePtr(cached.EffectiveTaxRate)}, provider: "fmp_cache"})
			continue
		}
		reserved := fundamentalsClient.Calls()+3 <= *fmpBudget
		if !*dry {
			reserved, err = repo.ReserveMarketOpsFMPCalls(ctx, *tenant, time.Now().UTC(), *fmpBudget, 3)
			if err != nil {
				return err
			}
		}
		if !reserved {
			deferred++
			if !*dry {
				_ = repo.UpsertMarketOpsFMPPollState(ctx, storage.MarketOpsFMPPollState{TenantID: *tenant, Symbol: asset.Ticker, Status: "deferred_budget", NextEligibleAt: ptrTime(time.Now().UTC().AddDate(0, 0, 1))})
			}
			continue
		}
		if !*dry {
			_ = repo.UpsertMarketOpsFMPPollState(ctx, storage.MarketOpsFMPPollState{TenantID: *tenant, Symbol: asset.Ticker, Status: "in_progress"})
		}
		f, err := loadDerivedFundamentals(ctx, repo, fundamentalsClient, priceClient, *tenant, asset.Ticker, session, *bar.Close, !*dry)
		if !*dry {
			_ = repo.CompleteMarketOpsFMPCalls(ctx, *tenant, time.Now().UTC(), 3)
		}
		if err != nil {
			fmpFailures++
			logger.Warn("financial derivation unavailable", "symbol", asset.Ticker, "error", err)
			if !*dry {
				_ = repo.UpsertMarketOpsFMPPollState(ctx, storage.MarketOpsFMPPollState{TenantID: *tenant, Symbol: asset.Ticker, Status: "failed", AttemptCount: 1, LastError: err.Error(), NextEligibleAt: ptrTime(time.Now().UTC().AddDate(0, 0, 1))})
			}
			f = fmp.FundamentalSnapshot{Ticker: asset.Ticker, FilingDate: time.Now().UTC(), ProviderRequestIDs: []string{"fmp:error"}}
			provider = "fmp_error"
		} else {
			refreshed++
			if !*dry {
				now := time.Now().UTC()
				_ = repo.UpsertMarketOpsFMPPollState(ctx, storage.MarketOpsFMPPollState{TenantID: *tenant, Symbol: asset.Ticker, Status: "succeeded", LastSuccessAt: &now, FinancialSnapshotID: financialLinkID("fmp", *tenant, asset.Ticker, session, f.FilingDate)})
			}
		}
		in := valuation.Input{Ticker: asset.Ticker, Price: *bar.Close, MarketCap: f.MarketCap, EnterpriseValue: f.EnterpriseValue, RevenueTTM: f.RevenueTTM, Revenue3YAgo: f.Revenue3YAgo, NetIncomeGAAPTTM: f.NetIncomeTTM, EBITDAProviderTTM: f.EBITDATTM, OperatingIncomeTTM: f.OperatingIncomeTTM, OperatingCashFlowTTM: f.OperatingCashFlowTTM, CapitalExpendituresTTM: f.CapitalExpendituresTTM, TotalDebt: f.TotalDebt, CashAndEquivalents: f.Cash, ShareholdersEquity: f.Equity, InvestedCapital: f.InvestedCapital, FinancialAgeDays: int(session.Sub(f.FilingDate).Hours() / 24), EffectiveTaxRate: taxRatePtr(f.EffectiveTaxRate)}
		candidates = append(candidates, candidate{asset: asset, fundamentals: f, price: *bar.Close, input: in, provider: provider})
	}
	for i := range candidates {
		applyPeers(candidates, i)
		result := valuation.Evaluate(candidates[i].input)
		payload, _ := json.Marshal(result)
		available := candidates[i].fundamentals.FilingDate
		if available.IsZero() {
			available = time.Now().UTC()
		}
		snapshotID := stable("valsnap", *tenant, candidates[i].asset.Ticker, session.Format("2006-01-02"), available.Format(time.RFC3339Nano))
		inputJSON, _ := json.Marshal(candidates[i].input)
		if !*dry {
			if err := repo.UpsertMarketOpsValuationSnapshot(ctx, storage.MarketOpsValuationSnapshotRecord{SnapshotID: snapshotID, FinancialSnapshotID: financialLinkID(candidates[i].provider, *tenant, candidates[i].asset.Ticker, session, available), TenantID: *tenant, Symbol: candidates[i].asset.Ticker, SessionDate: session, AvailableAt: available, Sector: first(candidates[i].asset.SectorKey, candidates[i].asset.Sector), Industry: first(candidates[i].asset.IndustryKey, candidates[i].asset.Industry), Provider: candidates[i].provider, ProviderRequestIDs: candidates[i].fundamentals.ProviderRequestIDs, InputJSON: inputJSON}); err != nil {
				return err
			}
			for _, output := range []struct {
				id             string
				score, fair    float64
				classification string
			}{{vcID, result.VCScore, result.VCFairValue, result.VCClassification}, {dosmID, result.DOSMScore, result.DOSMFairValue, result.DOSMClassification}} {
				if err := repo.UpsertMarketOpsValuationResult(ctx, storage.MarketOpsValuationResultRecord{ResultID: stable("valresult", snapshotID, output.id), SnapshotID: snapshotID, TenantID: *tenant, Symbol: candidates[i].asset.Ticker, SessionDate: session, AlgorithmID: output.id, ModelVersion: valuation.ModelVersion, Score: output.score, FairValue: output.fair, Classification: output.classification, Confidence: result.Confidence, ConfidenceLabel: result.ConfidenceLabel, EvaluationStatus: result.Status, Eligible: result.Eligible, ResultJSON: payload}); err != nil {
					return err
				}
			}
		}
		logger.Info("valuation evaluated", "symbol", candidates[i].asset.Ticker, "vc", result.VCScore, "dosm", result.DOSMScore, "eligible", result.Eligible)
	}
	logger.Info("valuation runner complete", "assets", len(assets), "evaluated", len(candidates), "fmp_calls", fundamentalsClient.Calls(), "fmp_budget", *fmpBudget, "cache_hits", cacheHits, "refreshed", refreshed, "deferred", deferred, "fmp_failures", fmpFailures, "dry_run", *dry)
	return nil
}
func cachedFundamentals(ctx context.Context, repo storage.MarketOpsValuationRepository, tenant, ticker string) (fmp.FundamentalSnapshot, bool, error) {
	snapshot, err := repo.LatestMarketOpsValuationSnapshot(ctx, tenant, ticker, "fmp")
	if err != nil {
		return fmp.FundamentalSnapshot{}, false, nil
	}
	if snapshot.CreatedAt.Before(time.Now().UTC().AddDate(0, 0, -7)) {
		return fmp.FundamentalSnapshot{}, false, nil
	}
	var in valuation.Input
	if err := json.Unmarshal(snapshot.InputJSON, &in); err != nil {
		return fmp.FundamentalSnapshot{}, false, nil
	}
	return fmp.FundamentalSnapshot{Ticker: ticker, FilingDate: snapshot.AvailableAt, RevenueTTM: in.RevenueTTM, Revenue3YAgo: in.Revenue3YAgo, NetIncomeTTM: in.NetIncomeGAAPTTM, EBITDATTM: in.EBITDAProviderTTM, OperatingIncomeTTM: in.OperatingIncomeTTM, OperatingCashFlowTTM: in.OperatingCashFlowTTM, CapitalExpendituresTTM: in.CapitalExpendituresTTM, TotalDebt: in.TotalDebt, Cash: in.CashAndEquivalents, Equity: in.ShareholdersEquity, InvestedCapital: in.InvestedCapital, MarketCap: in.MarketCap, EnterpriseValue: in.EnterpriseValue, ProviderRequestIDs: snapshot.ProviderRequestIDs}, true, nil
}

func financialLinkID(provider, tenant, symbol string, session, available time.Time) string {
	if provider != "fmp" {
		return ""
	}
	return stable("financial", tenant, symbol, session.Format("2006-01-02"), available.Format(time.RFC3339Nano))
}
func ptrTime(value time.Time) *time.Time { return &value }
func taxRatePtr(value float64) *float64  { return &value }

func applyPeers(items []candidate, index int) {
	subject := items[index]
	ps, pe, ev := []float64{}, []float64{}, []float64{}
	for j, item := range items {
		if j == index || first(item.asset.SectorKey, item.asset.Sector) != first(subject.asset.SectorKey, subject.asset.Sector) || first(item.asset.IndustryKey, item.asset.Industry) != first(subject.asset.IndustryKey, subject.asset.Industry) {
			continue
		}
		if item.input.RevenueTTM > 0 {
			ps = append(ps, item.input.MarketCap/item.input.RevenueTTM)
		}
		if item.input.NetIncomeGAAPTTM > 0 {
			pe = append(pe, item.input.MarketCap/item.input.NetIncomeGAAPTTM)
		}
		if item.input.EnterpriseValue > 0 && item.input.EBITDAProviderTTM > 0 {
			ev = append(ev, item.input.EnterpriseValue/item.input.EBITDAProviderTTM)
		}
	}
	items[index].input.PeerCount = len(ps)
	if len(ps) >= 3 {
		items[index].input.PeerPSMedian = ptr(median(ps))
	}
	if len(pe) >= 3 {
		items[index].input.PeerPEMedian = ptr(median(pe))
	}
	if len(ev) >= 3 {
		items[index].input.PeerEVEBITDAMedian = ptr(median(ev))
	}
}
func median(values []float64) float64 { sort.Float64s(values); return values[len(values)/2] }
func ptr(v float64) *float64          { return &v }
func first(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
func stable(prefix string, parts ...string) string {
	h := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return prefix + "_" + hex.EncodeToString(h[:])[:24]
}
func lastSession() time.Time {
	loc, _ := time.LoadLocation("America/New_York")
	d := time.Now().In(loc).AddDate(0, 0, -1)
	for d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
		d = d.AddDate(0, 0, -1)
	}
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
}

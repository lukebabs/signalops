package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/config"
	"github.com/lukebabs/signalops/internal/marketops/eroc"
	"github.com/lukebabs/signalops/internal/storage"
	postgres "github.com/lukebabs/signalops/internal/storage/postgres"
)

const algorithmID = "signalops.algorithms.eroc_v6"

type bar struct {
	Date   string  `json:"observation_date"`
	Symbol string  `json:"symbol"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
}

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(ctx context.Context) error {
	app := config.Load()
	tenant := flag.String("tenant-id", "tenant-local", "tenant")
	group := flag.String("universe-group", "all_active", "universe")
	date := flag.String("session-date", "", "completed YYYY-MM-DD")
	backfill := flag.Int("backfill-trading-days", 0, "latest completed sessions to score")
	dry := flag.Bool("dry-run", false, "calculate only")
	flag.Parse()
	if app.DatabaseURL == "" || app.TemporalDatabaseURL == "" {
		return fmt.Errorf("both database URLs are required")
	}
	repo, err := postgres.OpenWithTemporal(ctx, app.DatabaseURL, app.TemporalDatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	assets, err := repo.ListMarketOpsAssets(ctx, *tenant, *group, true, 5000)
	if err != nil {
		return err
	}
	symbols := make([]string, 0, len(assets))
	for _, asset := range assets {
		symbols = append(symbols, asset.Ticker)
	}
	events, err := repo.ListMarketOpsEODEvents(ctx, *tenant, symbols)
	if err != nil {
		return err
	}
	bars := map[string][]bar{}
	dates := map[string]struct{}{}
	for _, event := range events {
		var x bar
		if json.Unmarshal(event.NormalizedPayload, &x) == nil && x.Symbol != "" && x.Close > 0 && x.Volume >= 0 {
			symbol := strings.ToUpper(x.Symbol)
			bars[symbol] = append(bars[symbol], x)
			dates[x.Date] = struct{}{}
		}
	}
	for symbol := range bars {
		sort.Slice(bars[symbol], func(i, j int) bool { return bars[symbol][i].Date < bars[symbol][j].Date })
	}
	if *backfill > 0 {
		return fmt.Errorf("historical EROC output is not supported; use a completed session date")
	}
	sessions, err := requestedSessions(*date, 0, dates)
	if err != nil {
		return err
	}
	written := 0
	for _, session := range sessions {
		for _, asset := range assets {
			symbol := strings.ToUpper(asset.Ticker)
			history := asOfBars(bars[symbol], session)
			if len(history) > 21 {
				history = history[len(history)-21:]
			}
			closes, volumes := make([]float64, 0, len(history)), make([]float64, 0, len(history))
			for _, x := range history {
				closes = append(closes, x.Close)
				volumes = append(volumes, x.Volume)
			}
			distributions, err := repo.ListMarketOpsOptionsDistributions(ctx, storage.MarketOpsOptionsDistributionFilter{TenantID: *tenant, Symbol: symbol, Limit: 1000})
			if err != nil {
				return err
			}
			var calls, puts int64
			for _, d := range distributions {
				if d.TradeDate.UTC().Format("2006-01-02") == session.Format("2006-01-02") {
					calls, puts = d.TotalCallVolume, d.TotalPutVolume
					break
				}
			}
			ivRegime, err := ivRegimeForSession(ctx, repo, *tenant, symbol, session)
			if err != nil {
				return err
			}
			result := eroc.Evaluate(eroc.Input{Closes: closes, Volumes: volumes, CallVolume: calls, PutVolume: puts, IVRegime: ivRegime})
			payload, _ := json.Marshal(result)
			input, _ := json.Marshal(map[string]any{"close_count": len(closes), "call_volume": calls, "put_volume": puts, "session_date": session.Format("2006-01-02")})
			id := stable(*tenant, symbol, session.Format("2006-01-02"), eroc.ModelVersion)
			if !*dry {
				snapshot := storage.MarketOpsValuationSnapshotRecord{SnapshotID: "erocsnap_" + id, TenantID: *tenant, Symbol: symbol, SessionDate: session, AvailableAt: session.Add(20*time.Hour + 5*time.Minute), Provider: "marketops_eroc", InputJSON: input}
				if err := repo.UpsertMarketOpsValuationSnapshot(ctx, snapshot); err != nil {
					return err
				}
				if err := repo.UpsertMarketOpsValuationResult(ctx, storage.MarketOpsValuationResultRecord{ResultID: "erocres_" + id, SnapshotID: snapshot.SnapshotID, TenantID: *tenant, Symbol: symbol, SessionDate: session, AlgorithmID: algorithmID, ModelVersion: eroc.ModelVersion, Score: result.Score, FairValue: 0, Classification: string(result.State), Confidence: 100, ConfidenceLabel: "complete", EvaluationStatus: "complete", Eligible: result.State == eroc.StateConfirmed, ResultJSON: payload}); err != nil {
					return err
				}
			}
			written++
		}
	}
	fmt.Printf("eroc completed assets=%d sessions=%d dry_run=%t\n", written, len(sessions), *dry)
	return nil
}
func ivRegimeForSession(ctx context.Context, repo *postgres.Repository, tenant, symbol string, session time.Time) (string, error) {
	rows, err := repo.ListMarketOpsFeatureObservations(ctx, storage.MarketOpsFeatureObservationFilter{TenantID: tenant, AppID: "marketops", Symbol: symbol, SessionStart: session, SessionEnd: session, Limit: 100})
	if err != nil {
		return "", err
	}
	for _, row := range rows {
		if row.FeatureKey == "medium_term_iv_regime" && (row.QualityState == storage.MarketOpsQualityUsable || row.QualityState == storage.MarketOpsQualityUsableWithWarning) && row.TextValue != nil {
			return *row.TextValue, nil
		}
	}
	return "", nil
}

func requestedSessions(date string, backfill int, available map[string]struct{}) ([]time.Time, error) {
	if backfill <= 0 {
		if date == "" {
			return nil, fmt.Errorf("session-date is required unless backfill-trading-days is set")
		}
		d, err := time.Parse("2006-01-02", date)
		return []time.Time{d}, err
	}
	keys := make([]string, 0, len(available))
	for key := range available {
		keys = append(keys, key)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	if len(keys) > backfill {
		keys = keys[:backfill]
	}
	out := make([]time.Time, 0, len(keys))
	for _, key := range keys {
		d, err := time.Parse("2006-01-02", key)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}
func asOfBars(bars []bar, session time.Time) []bar {
	cutoff := session.Format("2006-01-02")
	out := make([]bar, 0, len(bars))
	for _, b := range bars {
		if b.Date <= cutoff {
			out = append(out, b)
		}
	}
	return out
}
func stable(parts ...string) string {
	h := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(h[:])[:32]
}

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

	"github.com/lukebabs/signalops/internal/adapters/marketdata/massive"
	"github.com/lukebabs/signalops/internal/config"
	"github.com/lukebabs/signalops/internal/storage"
	postgres "github.com/lukebabs/signalops/internal/storage/postgres"
)

const strategicVC = "signalops.algorithms.valuation_composite_v3"
const strategicDOSM = "signalops.algorithms.distressed_opportunity_scoring_v3"
const tacticalPosture = "signalops.algorithms.tactical_market_posture_v1"

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(ctx context.Context) error {
	tenant := flag.String("tenant-id", "tenant-local", "tenant")
	group := flag.String("universe-group", "all_active", "universe")
	date := flag.String("session-date", "", "completed session YYYY-MM-DD")
	flag.Parse()
	app := config.Load()
	if app.DatabaseURL == "" {
		return fmt.Errorf("SIGNALOPS_DATABASE_URL is required")
	}
	session := lastSession()
	if *date != "" {
		var err error
		session, err = time.Parse("2006-01-02", *date)
		if err != nil {
			return err
		}
	}
	repo, err := postgres.Open(ctx, app.DatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	market, err := massive.NewClient(massive.LoadClientConfigFromEnv())
	if err != nil {
		return fmt.Errorf("Massive SMA client: %w", err)
	}
	assets, err := repo.ListMarketOpsAssets(ctx, *tenant, *group, true, 200)
	if err != nil {
		return err
	}
	for _, asset := range assets {
		results, err := repo.ListMarketOpsValuationResults(ctx, storage.MarketOpsValuationFilter{TenantID: *tenant, Symbol: asset.Ticker, Limit: 500})
		if err != nil {
			continue
		}
		var vc, dosm *storage.MarketOpsValuationResultRecord
		for i := range results {
			if results[i].AlgorithmID == strategicVC && (vc == nil || results[i].SessionDate.After(vc.SessionDate)) {
				vc = &results[i]
			}
			if results[i].AlgorithmID == strategicDOSM && (dosm == nil || results[i].SessionDate.After(dosm.SessionDate)) {
				dosm = &results[i]
			}
		}
		if vc == nil || dosm == nil {
			continue
		}
		observations, err := repo.ListMarketOpsFeatureObservations(ctx, storage.MarketOpsFeatureObservationFilter{TenantID: *tenant, AppID: "marketops", Symbol: asset.Ticker, SessionStart: session, SessionEnd: session, Limit: 100})
		if err != nil {
			continue
		}
		values := map[string]float64{}
		ids := []string{}
		for _, o := range observations {
			if (o.QualityState == "usable" || o.QualityState == "usable_with_warning") && o.NumericValue != nil {
				values[o.FeatureKey] = *o.NumericValue
				ids = append(ids, o.FeatureObservationID)
			}
		}
		sma50, err := market.ListSimpleMovingAverage(ctx, asset.Ticker, 50, 21)
		if err != nil {
			return fmt.Errorf("%s SMA-50: %w", asset.Ticker, err)
		}
		sma200, err := market.ListSimpleMovingAverage(ctx, asset.Ticker, 200, 1)
		if err != nil {
			return fmt.Errorf("%s SMA-200: %w", asset.Ticker, err)
		}
		rsi14, err := market.ListRelativeStrengthIndex(ctx, asset.Ticker, 14, 1)
		if err != nil {
			return fmt.Errorf("%s RSI-14: %w", asset.Ticker, err)
		}
		overlay, components, providerTechnicals, ok := technical(values, sma50, sma200, rsi14, session)
		if !ok {
			continue
		}
		status := posture(overlay)
		payload := map[string]any{"data_profile": "tactical_eod_v3_provider_technicals", "strategic_snapshot_id": vc.SnapshotID, "strategic_vc": vc.Score, "strategic_dosm": dosm.Score, "technical_overlay": overlay, "posture": status, "technical_components": components, "provider_technicals": providerTechnicals, "feature_observation_ids": ids, "session_date": session.Format("2006-01-02")}
		raw, _ := json.Marshal(payload)
		available := session.Add(20*time.Hour + time.Minute) // separate technical publication slot; snapshot uniqueness is per availability time
		snapshot := stable("tactical-posture", *tenant, asset.Ticker, session.Format("2006-01-02"))
		if err = repo.UpsertMarketOpsValuationSnapshot(ctx, storage.MarketOpsValuationSnapshotRecord{SnapshotID: snapshot, FinancialSnapshotID: "", TenantID: *tenant, Symbol: asset.Ticker, SessionDate: session, AvailableAt: available, Sector: asset.Sector, Industry: asset.Industry, Provider: "massive_technical_indicators", ProviderRequestIDs: ids, InputJSON: raw}); err != nil {
			return err
		}
		payload["score"] = overlay
		b, _ := json.Marshal(payload)
		if err = repo.UpsertMarketOpsValuationResult(ctx, storage.MarketOpsValuationResultRecord{ResultID: stable("tacticalposture", snapshot, tacticalPosture), SnapshotID: snapshot, TenantID: *tenant, Symbol: asset.Ticker, SessionDate: session, AlgorithmID: tacticalPosture, ModelVersion: "tactical-posture-v3", Score: overlay, FairValue: 0, Classification: status, Confidence: 100, ConfidenceLabel: "complete", EvaluationStatus: "complete", Eligible: true, ResultJSON: b}); err != nil {
			return err
		}
	}
	return nil
}
func technical(v map[string]float64, sma50, sma200 []massive.SimpleMovingAverage, rsi14 []massive.RelativeStrengthIndex, session time.Time) (float64, map[string]float64, map[string]float64, bool) {
	for _, k := range []string{"return_5d", "distance_sma_50_pct"} {
		if _, ok := v[k]; !ok {
			return 0, nil, nil, false
		}
	}
	if len(sma50) < 21 || len(sma200) < 1 || len(rsi14) < 1 || sma50[0].Value <= 0 || sma50[20].Value <= 0 || sma200[0].Value <= 0 || rsi14[0].Value <= 0 || sma50[0].Timestamp.Format("2006-01-02") != session.Format("2006-01-02") || sma200[0].Timestamp.Format("2006-01-02") != session.Format("2006-01-02") || rsi14[0].Timestamp.Format("2006-01-02") != session.Format("2006-01-02") {
		return 0, nil, nil, false
	}
	price := sma50[0].Value * (1 + v["distance_sma_50_pct"]/100)
	distance200 := (price/sma200[0].Value - 1) * 100
	slope50 := (sma50[0].Value/sma50[20].Value - 1) * 100
	rsi, trend, extension := 0.0, 0.0, 0.0
	if rsi14[0].Value < 30 {
		rsi = .5
	} else if rsi14[0].Value > 70 {
		rsi = -.5
	}
	if v["distance_sma_50_pct"] > 0 && distance200 > 0 && slope50 > 0 {
		trend = .5
	} else if v["distance_sma_50_pct"] < 0 && distance200 < 0 && slope50 < 0 {
		trend = -.5
	}
	if v["return_5d"] <= -5 {
		extension = .5
	} else if v["return_5d"] >= 5 {
		extension = -.5
	}
	return clamp(rsi+trend+extension, -1.5, 1.5), map[string]float64{"rsi_reversal": rsi, "sma_trend": trend, "price_extension": extension}, map[string]float64{"rsi_14": rsi14[0].Value, "sma_50": sma50[0].Value, "sma_200": sma200[0].Value, "distance_sma_200_pct": distance200, "sma_50_slope_20d_pct": slope50}, true
}
func posture(overlay float64) string {
	if overlay >= .5 {
		return "constructive"
	}
	if overlay <= -.5 {
		return "caution"
	}
	return "neutral"
}
func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
func classify(v float64) string {
	if v >= 8 {
		return "exceptional"
	}
	if v >= 6 {
		return "opportunity"
	}
	if v >= 4 {
		return "neutral"
	}
	if v >= 2 {
		return "weak"
	}
	return "avoid"
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

var _ = sort.Strings

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
	assets, err := repo.ListMarketOpsAssets(ctx, *tenant, *group, true, 200)
	if err != nil {
		return err
	}
	for _, asset := range assets {
		results, err := repo.ListMarketOpsValuationResults(ctx, storage.MarketOpsValuationFilter{TenantID: *tenant, Symbol: asset.Ticker, Limit: 30})
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
		overlay, components, ok := technical(values)
		if !ok {
			continue
		}
		status := posture(overlay)
		payload := map[string]any{"data_profile": "tactical_eod_v1", "strategic_snapshot_id": vc.SnapshotID, "strategic_vc": vc.Score, "strategic_dosm": dosm.Score, "technical_overlay": overlay, "posture": status, "technical_components": components, "feature_observation_ids": ids, "session_date": session.Format("2006-01-02")}
		raw, _ := json.Marshal(payload)
		available := session.Add(20 * time.Hour)
		snapshot := stable("tactical-posture", *tenant, asset.Ticker, session.Format("2006-01-02"))
		if err = repo.UpsertMarketOpsValuationSnapshot(ctx, storage.MarketOpsValuationSnapshotRecord{SnapshotID: snapshot, FinancialSnapshotID: vc.SnapshotID, TenantID: *tenant, Symbol: asset.Ticker, SessionDate: session, AvailableAt: available, Sector: asset.Sector, Industry: asset.Industry, Provider: "massive_eod_technical", ProviderRequestIDs: ids, InputJSON: raw}); err != nil {
			return err
		}
		payload["score"] = overlay
		b, _ := json.Marshal(payload)
		if err = repo.UpsertMarketOpsValuationResult(ctx, storage.MarketOpsValuationResultRecord{ResultID: stable("tacticalposture", snapshot, tacticalPosture), SnapshotID: snapshot, TenantID: *tenant, Symbol: asset.Ticker, SessionDate: session, AlgorithmID: tacticalPosture, ModelVersion: "tactical-posture-v1", Score: overlay, FairValue: 0, Classification: status, Confidence: 100, ConfidenceLabel: "complete", EvaluationStatus: "complete", Eligible: true, ResultJSON: b}); err != nil {
			return err
		}
	}
	return nil
}
func technical(v map[string]float64) (float64, map[string]float64, bool) {
	keys := []string{"rsi_14", "return_5d", "distance_sma_50_pct", "distance_sma_200_pct", "sma_50_slope_20d_pct"}
	for _, k := range keys {
		if _, ok := v[k]; !ok {
			return 0, nil, false
		}
	}
	rsi, trend, extension := 0.0, 0.0, 0.0
	if v["rsi_14"] < 30 {
		rsi = .5
	} else if v["rsi_14"] > 70 {
		rsi = -.5
	}
	if v["distance_sma_50_pct"] > 0 && v["distance_sma_200_pct"] > 0 && v["sma_50_slope_20d_pct"] > 0 {
		trend = .5
	} else if v["distance_sma_50_pct"] < 0 && v["distance_sma_200_pct"] < 0 && v["sma_50_slope_20d_pct"] < 0 {
		trend = -.5
	}
	if v["return_5d"] <= -5 {
		extension = .5
	} else if v["return_5d"] >= 5 {
		extension = -.5
	}
	return clamp(rsi+trend+extension, -1.5, 1.5), map[string]float64{"rsi_reversal": rsi, "sma_trend": trend, "price_extension": extension}, true
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

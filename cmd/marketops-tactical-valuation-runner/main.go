package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
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
const tacticalTaskType = "tactical_posture"

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	tenant := flag.String("tenant-id", "tenant-local", "tenant")
	group := flag.String("universe-group", "all_workflow_ready", "universe")
	date := flag.String("session-date", "", "completed session YYYY-MM-DD")
	symbols := flag.String("symbols", "", "optional comma-separated asset scope")
	maxRetries := flag.Int("max-retries", 2, "bounded retries per transient provider call")
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
		return fmt.Errorf("Massive technical client: %w", err)
	}
	assets, err := repo.ListMarketOpsAssets(ctx, *tenant, *group, true, 5000)
	if err != nil {
		return err
	}
	wanted := symbolSet(*symbols)
	if len(wanted) > 0 {
		filtered := assets[:0]
		for _, asset := range assets {
			if wanted[strings.ToUpper(asset.Ticker)] {
				filtered = append(filtered, asset)
			}
		}
		assets = filtered
	}
	workflowID := stable("marketops-workflow", *tenant, "postclose", session.Format("2006-01-02"))
	now := time.Now().UTC()
	if err := repo.UpsertMarketOpsTaskWorkflow(ctx, storage.MarketOpsTaskWorkflowRecord{WorkflowID: workflowID, TenantID: *tenant, SessionDate: session, WorkflowType: "marketops_postclose", Status: "running", ScheduleJobID: "marketops-daily-postclose", StartedAt: &now}); err != nil {
		return err
	}
	counts := map[string]int{}
	for _, asset := range assets {
		status, class, message, retryAt := runAsset(ctx, repo, market, *tenant, session, workflowID, asset, *maxRetries)
		counts[status]++
		if err := recordTacticalTask(ctx, repo, workflowID, *tenant, session, asset.Ticker, status, class, message, retryAt, *maxRetries); err != nil {
			return err
		}
	}
	// A retry worker may process only a subset of assets. Always derive the
	// parent workflow from the full durable ledger, never from this invocation's
	// local counts, so prior skipped, deferred, or blocked work stays visible.
	items, err := repo.ListMarketOpsTaskItems(ctx, storage.MarketOpsTaskItemFilter{TenantID: *tenant, SessionDate: session, TaskType: tacticalTaskType, Limit: 500})
	if err != nil {
		return err
	}
	ledgerCounts := map[string]int{}
	workflowStatus := "succeeded"
	for _, item := range items {
		ledgerCounts[item.Status]++
		if item.Status != "succeeded" {
			workflowStatus = "degraded"
		}
	}
	complete := time.Now().UTC()
	coverage, _ := json.Marshal(ledgerCounts)
	return repo.UpsertMarketOpsTaskWorkflow(ctx, storage.MarketOpsTaskWorkflowRecord{WorkflowID: workflowID, TenantID: *tenant, SessionDate: session, WorkflowType: "marketops_postclose", Status: workflowStatus, ScheduleJobID: "marketops-daily-postclose", CoverageJSON: coverage, CompletedAt: &complete})
}

func runAsset(ctx context.Context, repo *postgres.Repository, market *massive.Client, tenant string, session time.Time, workflowID string, asset storage.MarketOpsAssetRecord, maxRetries int) (string, string, string, time.Time) {
	results, err := repo.ListMarketOpsValuationResults(ctx, storage.MarketOpsValuationFilter{TenantID: tenant, Symbol: asset.Ticker, Limit: 500})
	if err != nil {
		return "retry_scheduled", "storage_transient", safeError(err), time.Now().UTC().Add(2 * time.Minute)
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
		return "skipped_no_data", "missing_strategic_valuation", "VC and DOSM results are unavailable", time.Time{}
	}
	observations, err := repo.ListMarketOpsFeatureObservations(ctx, storage.MarketOpsFeatureObservationFilter{TenantID: tenant, AppID: "marketops", Symbol: asset.Ticker, SessionStart: session, SessionEnd: session, Limit: 100})
	if err != nil {
		return "retry_scheduled", "storage_transient", safeError(err), time.Now().UTC().Add(2 * time.Minute)
	}
	values := map[string]float64{}
	ids := []string{}
	for _, o := range observations {
		if (o.QualityState == "usable" || o.QualityState == "usable_with_warning") && o.NumericValue != nil {
			values[o.FeatureKey] = *o.NumericValue
			ids = append(ids, o.FeatureObservationID)
		}
	}
	if _, ok := values["return_5d"]; !ok {
		return "skipped_no_data", "missing_market_state_feature", "return_5d is unavailable for the completed session", time.Time{}
	}
	if _, ok := values["distance_sma_50_pct"]; !ok {
		return "skipped_no_data", "missing_market_state_feature", "distance_sma_50_pct is unavailable for the completed session", time.Time{}
	}
	sma50, sma200, rsi14, err := providerTechnicalWithRetry(ctx, market, asset.Ticker, maxRetries)
	if err != nil {
		class, status := classifyProviderFailure(err)
		if status == "retry_scheduled" {
			return status, class, safeError(err), time.Now().UTC().Add(5 * time.Minute)
		}
		return status, class, safeError(err), time.Time{}
	}
	overlay, components, providerTechnicals, ok := technical(values, sma50, sma200, rsi14, session)
	if !ok {
		return "skipped_no_data", "missing_provider_technical", "same-session RSI-14, SMA-50, or SMA-200 is unavailable", time.Time{}
	}
	status := posture(overlay)
	payload := map[string]any{"data_profile": "tactical_eod_v3_provider_technicals", "strategic_snapshot_id": vc.SnapshotID, "strategic_vc": vc.Score, "strategic_dosm": dosm.Score, "technical_overlay": overlay, "posture": status, "technical_components": components, "provider_technicals": providerTechnicals, "feature_observation_ids": ids, "session_date": session.Format("2006-01-02")}
	raw, _ := json.Marshal(payload)
	available := session.Add(20*time.Hour + time.Minute)
	snapshot := stable("tactical-posture", tenant, asset.Ticker, session.Format("2006-01-02"))
	if err := repo.UpsertMarketOpsValuationSnapshot(ctx, storage.MarketOpsValuationSnapshotRecord{SnapshotID: snapshot, FinancialSnapshotID: "", TenantID: tenant, Symbol: asset.Ticker, SessionDate: session, AvailableAt: available, Sector: asset.Sector, Industry: asset.Industry, Provider: "massive_technical_indicators", ProviderRequestIDs: ids, InputJSON: raw}); err != nil {
		return "retry_scheduled", "storage_transient", safeError(err), time.Now().UTC().Add(2 * time.Minute)
	}
	// The valuation snapshot uniqueness key is tenant/symbol/session/availability.
	// Resolve its authoritative ID after an upsert so a retry never references a
	// newly computed ID when the immutable snapshot already exists.
	persisted, err := repo.LatestMarketOpsValuationSnapshot(ctx, tenant, asset.Ticker, "massive_technical_indicators")
	if err != nil || !persisted.SessionDate.Equal(session) {
		if err == nil {
			err = fmt.Errorf("persisted tactical snapshot session mismatch")
		}
		return "retry_scheduled", "storage_transient", safeError(err), time.Now().UTC().Add(2 * time.Minute)
	}
	snapshot = persisted.SnapshotID
	payload["score"] = overlay
	b, _ := json.Marshal(payload)
	if err := repo.UpsertMarketOpsValuationResult(ctx, storage.MarketOpsValuationResultRecord{ResultID: stable("tacticalposture", snapshot, tacticalPosture), SnapshotID: snapshot, TenantID: tenant, Symbol: asset.Ticker, SessionDate: session, AlgorithmID: tacticalPosture, ModelVersion: "tactical-posture-v3", Score: overlay, FairValue: 0, Classification: status, Confidence: 100, ConfidenceLabel: "complete", EvaluationStatus: "complete", Eligible: true, ResultJSON: b}); err != nil {
		return "retry_scheduled", "storage_transient", safeError(err), time.Now().UTC().Add(2 * time.Minute)
	}
	return "succeeded", "", "", time.Time{}
}

func providerTechnicalWithRetry(ctx context.Context, market *massive.Client, symbol string, maxRetries int) ([]massive.SimpleMovingAverage, []massive.SimpleMovingAverage, []massive.RelativeStrengthIndex, error) {
	var sma50, sma200 []massive.SimpleMovingAverage
	var rsi []massive.RelativeStrengthIndex
	var err error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		sma50, err = market.ListSimpleMovingAverage(ctx, symbol, 50, 21)
		if err == nil {
			sma200, err = market.ListSimpleMovingAverage(ctx, symbol, 200, 1)
		}
		if err == nil {
			rsi, err = market.ListRelativeStrengthIndex(ctx, symbol, 14, 1)
		}
		if err == nil {
			return sma50, sma200, rsi, nil
		}
		class, status := classifyProviderFailure(err)
		if status != "retry_scheduled" || attempt == maxRetries {
			return nil, nil, nil, fmt.Errorf("%s (%s): %w", class, symbol, err)
		}
		select {
		case <-ctx.Done():
			return nil, nil, nil, ctx.Err()
		case <-time.After(time.Duration(attempt+1) * time.Second):
		}
	}
	return nil, nil, nil, err
}
func classifyProviderFailure(err error) (string, string) {
	s := strings.ToLower(err.Error())
	code := httpStatus(err)
	switch {
	case code == 401 || code == 403:
		return "provider_entitlement", "blocked_entitlement"
	case code == 402:
		return "provider_quota", "deferred_quota"
	case code == 429 || code >= 500 || errors.Is(err, context.DeadlineExceeded) || isNetwork(err):
		return "provider_transient", "retry_scheduled"
	case strings.Contains(s, "no bars") || strings.Contains(s, "not found") || strings.Contains(s, "response contained no values") || code == 404:
		return "provider_no_data", "skipped_no_data"
	default:
		return "provider_terminal", "failed_terminal"
	}
}
func httpStatus(err error) int {
	var coded interface{ StatusCode() int }
	if errors.As(err, &coded) {
		return coded.StatusCode()
	}
	for _, part := range strings.Fields(err.Error()) {
		if n, e := strconv.Atoi(part); e == nil && n >= 100 && n <= 599 {
			return n
		}
	}
	return 0
}
func isNetwork(err error) bool { var n net.Error; return errors.As(err, &n) }
func safeError(err error) string {
	s := err.Error()
	if len(s) > 500 {
		return s[:500]
	}
	return s
}
func recordTacticalTask(ctx context.Context, repo *postgres.Repository, workflow, tenant string, session time.Time, symbol, status, class, message string, next time.Time, maxRetries int) error {
	now := time.Now().UTC()
	attempt := 1
	existing, err := repo.ListMarketOpsTaskItems(ctx, storage.MarketOpsTaskItemFilter{TenantID: tenant, SessionDate: session, TaskType: tacticalTaskType, Symbol: symbol, Limit: 1})
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		attempt = existing[0].AttemptCount + 1
	}
	maxAttempts := maxRetries + 1
	if status == "retry_scheduled" && attempt >= maxAttempts {
		status, class = "failed_terminal", "retry_exhausted"
		message = "bounded task retries exhausted: " + message
		next = time.Time{}
	}
	done := &now
	if status == "retry_scheduled" {
		done = nil
	}
	result, _ := json.Marshal(map[string]any{"reason": message})
	return repo.UpsertMarketOpsTaskItem(ctx, storage.MarketOpsTaskItemRecord{TaskID: stable("marketops-task", tenant, session.Format("2006-01-02"), tacticalTaskType, symbol), WorkflowID: workflow, TenantID: tenant, SessionDate: session, TaskType: tacticalTaskType, Symbol: symbol, Status: status, AttemptCount: attempt, MaxAttempts: maxAttempts, NextAttemptAt: next, Provider: "massive", FailureClass: class, ErrorMessage: message, ResultJSON: result, CompletedAt: done})
}
func symbolSet(value string) map[string]bool {
	out := map[string]bool{}
	for _, x := range strings.Split(value, ",") {
		if x = strings.ToUpper(strings.TrimSpace(x)); x != "" {
			out[x] = true
		}
	}
	return out
}
func technical(v map[string]float64, sma50, sma200 []massive.SimpleMovingAverage, rsi14 []massive.RelativeStrengthIndex, session time.Time) (float64, map[string]float64, map[string]float64, bool) {
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
func stable(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}
func lastSession() time.Time {
	now := time.Now().UTC()
	for now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		now = now.AddDate(0, 0, -1)
	}
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

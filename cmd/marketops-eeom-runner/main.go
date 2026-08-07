package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/lukebabs/signalops/internal/adapters/marketdata/fmp"
	"github.com/lukebabs/signalops/internal/config"
	"github.com/lukebabs/signalops/internal/marketops/eeom"
	"github.com/lukebabs/signalops/internal/storage"
	postgres "github.com/lukebabs/signalops/internal/storage/postgres"
	"os"
	"strings"
	"time"
)

const eeomAlgorithmID = "signalops.algorithms.earnings_event_opportunity_v1"

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(ctx context.Context) error {
	app := config.Load()
	tenant := flag.String("tenant-id", "tenant-local", "tenant")
	session := flag.String("session-date", "", "completed date")
	days := flag.Int("window-days", 30, "maximum earnings horizon")
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
	asof := time.Now().UTC()
	if *session != "" {
		asof, err = time.Parse("2006-01-02", *session)
		if err != nil {
			return err
		}
	}
	calendarClient, err := fmp.NewClient(fmp.LoadClientConfigFromEnv())
	if err != nil {
		return err
	}
	assets, err := repo.ListMarketOpsAssets(ctx, *tenant, "all_active", true, 5000)
	if err != nil {
		return err
	}
	active := map[string]bool{}
	for _, asset := range assets {
		active[strings.ToUpper(asset.Ticker)] = true
	}
	// One bounded FMP request is shared by the canonical event projection and
	// EEOM. The two-day lookback preserves post-earnings awareness without
	// consuming an additional provider call.
	retrievedAt := time.Now().UTC()
	events, err := calendarClient.GetEarningsCalendar(ctx, asof.AddDate(0, 0, -2), asof.AddDate(0, 0, *days))
	if err != nil {
		return fmt.Errorf("fetch FMP earnings calendar: %w", err)
	}
	written, projected := 0, 0
	for _, event := range events {
		symbol := strings.ToUpper(event.Symbol)
		if !active[symbol] {
			continue
		}
		date, parseErr := time.Parse("2006-01-02", event.Date)
		if parseErr != nil {
			continue
		}
		d := int(date.Sub(asof.Truncate(24*time.Hour)).Hours() / 24)
		if d < -2 || d > *days {
			continue
		}
		eventID := "fmp_earnings_" + stable(*tenant, symbol, event.Date)
		if !*dry {
			if err := repo.UpsertNormalizedEventLedger(ctx, normalizedEarningsEvent(*tenant, symbol, eventID, date, retrievedAt, event)); err != nil {
				return fmt.Errorf("persist FMP earnings event %s: %w", symbol, err)
			}
			projected++
		}
		if d < 0 {
			continue
		}
		result, calcErr := calculate(ctx, repo, *tenant, symbol, asof, date, d, nil)
		if calcErr != nil {
			return calcErr
		}
		payload, _ := json.Marshal(map[string]any{"algorithm_id": eeomAlgorithmID, "calendar_provider": "fmp", "event_id": eventID, "calendar_retrieved_at": retrievedAt.Format(time.RFC3339), "event": earningsEventPayload(symbol, date, retrievedAt, event), "calendar_record": event, "result": result})
		if !*dry {
			id := stable(*tenant, symbol, eventID, asof.Format("2006-01-02"), eeom.ModelVersion)
			if err := repo.UpsertMarketOpsEEOMResult(ctx, storage.MarketOpsEEOMResultRecord{ResultID: "eeom_" + id, TenantID: *tenant, Symbol: symbol, EarningsEventID: eventID, EarningsDate: date, SessionDate: asof, ModelVersion: eeom.ModelVersion, Score: result.Score, Posture: result.Posture, Classification: result.Classification, EvidenceQuality: result.EvidenceQuality, Eligible: result.Eligible, ResultJSON: payload}); err != nil {
				return err
			}
		}
		written++
	}
	fmt.Printf("eeom completed provider=fmp calendar_events=%d projected_events=%d results=%d fmp_calls=%d dry_run=%t\n", len(events), projected, written, calendarClient.Calls(), *dry)
	return nil
}
func earningsEventPayload(symbol string, date, retrievedAt time.Time, event fmp.EarningsCalendarRecord) map[string]any {
	lastVerified := retrievedAt.UTC()
	if value := strings.TrimSpace(event.LastUpdated); value != "" {
		if parsed, err := time.Parse(time.RFC3339, value); err == nil {
			lastVerified = parsed.UTC()
		}
	}
	return map[string]any{
		"symbol": symbol, "event_type": "earnings", "event_date": date.UTC().Format("2006-01-02"),
		"event_time": nil, "status": "date_reported", "confidence": nil,
		"source": "FinancialModelingPrep", "last_verified": lastVerified.Format(time.RFC3339),
		"known_at": retrievedAt.UTC().Format(time.RFC3339),
	}
}

func normalizedEarningsEvent(tenant, symbol, eventID string, date, retrievedAt time.Time, event fmp.EarningsCalendarRecord) storage.NormalizedEventLedgerRecord {
	payload, _ := json.Marshal(earningsEventPayload(symbol, date, retrievedAt, event))
	eventJSON, _ := json.Marshal(map[string]any{"event_id": eventID, "event_type": "market_data.fmp.earnings_calendar", "payload": json.RawMessage(payload)})
	entities, _ := json.Marshal([]map[string]string{{"type": "ticker", "external_id": symbol}})
	metadata, _ := json.Marshal(map[string]any{"provider": "fmp", "calendar_retrieved_at": retrievedAt.UTC().Format(time.RFC3339)})
	return storage.NormalizedEventLedgerRecord{
		EventID: eventID, TenantID: tenant, SourceID: "src-fmp", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance",
		SourceAdapter: "market_data.fmp", Dataset: "market_event_calendar", IdempotencyKey: "idem_" + stable(tenant, eventID), SchemaID: "signalops.market_event_calendar.fmp.v1", SchemaVersion: "1.0.0",
		ObservationTime: date.UTC(), ProcessingTime: retrievedAt.UTC(), Confidence: 0, RawTopic: "signalops.fmp.calendar", NormalizedTopic: "signalops.normalized.marketops.market_event_calendar",
		NormalizedPayload: payload, EntitiesJSON: entities, MetadataJSON: metadata, EventJSON: eventJSON,
	}
}

func calculate(ctx context.Context, repo *postgres.Repository, tenant, symbol string, session, event time.Time, days int, importance *float64) (eeom.Result, error) {
	vals, err := repo.ListMarketOpsValuationResults(ctx, storage.MarketOpsValuationFilter{TenantID: tenant, Symbol: symbol, Limit: 20})
	if err != nil {
		return eeom.Result{}, err
	}
	var technical, vc, dosm eeom.Component
	for _, x := range vals {
		var trace map[string]any
		_ = json.Unmarshal(x.ResultJSON, &trace)
		switch x.AlgorithmID {
		case "signalops.algorithms.tactical_market_posture_v1":
			technical = eeom.Component{Score: 50 + x.Score/3*50, Available: true, Direction: direction(x.Classification)}
		case "signalops.algorithms.valuation_composite_v3":
			vc = eeom.Component{Score: x.Score * 10, Available: x.Eligible}
		case "signalops.algorithms.distressed_opportunity_scoring_v3":
			dosm = eeom.Component{Score: x.Score * 10, Available: x.Eligible}
		}
	}
	options := eeom.Component{}
	observations, err := repo.ListMarketOpsFeatureObservations(ctx, storage.MarketOpsFeatureObservationFilter{TenantID: tenant, AppID: "marketops", Symbol: symbol, Limit: 500})
	if err != nil {
		return eeom.Result{}, err
	}
	var ivRV *float64
	regime := ""
	for _, observation := range observations {
		if observation.QualityState != storage.MarketOpsQualityUsable && observation.QualityState != storage.MarketOpsQualityUsableWithWarning {
			continue
		}
		if observation.FeatureKey == "iv_rv_ratio_20d" && observation.NumericValue != nil && ivRV == nil {
			ivRV = observation.NumericValue
		}
		if observation.FeatureKey == "medium_term_iv_regime" && observation.TextValue != nil && regime == "" {
			regime = *observation.TextValue
		}
	}
	if ivRV != nil && regime != "" {
		options = eeom.Component{Score: 50, Available: true, Reason: "30-90 DTE IV regime: " + regime}
		if regime == "elevated_premium" && technical.Direction != "" && technical.Direction != "neutral" {
			options.Score, options.Direction = 60, technical.Direction
			options.Reason = "30-90 DTE IV premium corroborates the independently observed " + technical.Direction + " technical posture"
		}
	}
	rrs, err := repo.ListMarketOpsRiskRewardSnapshots(ctx, storage.MarketOpsRiskRewardSnapshotFilter{TenantID: tenant, Symbol: symbol, Limit: 10})
	if err != nil {
		return eeom.Result{}, err
	}
	rr := eeom.Component{}
	if len(rrs) > 0 {
		x := rrs[0]
		rr = eeom.Component{Score: clamp50(x.TechnicalScore), Available: x.Eligible, Direction: direction(x.TechnicalDirection)}
	}
	material := 50.0 + float64(30-days)*50/30
	if importance != nil {
		material = clamp50(*importance * 10)
	}
	return eeom.Evaluate(eeom.Input{DaysToEarnings: days, Technical: technical, Options: options, RiskReward: rr, VC: vc, DOSM: dosm, Materiality: eeom.Component{Score: material, Available: true}}), nil
}
func direction(x string) string {
	x = strings.ToLower(x)
	if strings.Contains(x, "bull") || strings.Contains(x, "constructive") {
		return "bullish"
	}
	if strings.Contains(x, "bear") || strings.Contains(x, "caution") {
		return "bearish"
	}
	return "neutral"
}
func clamp50(x float64) float64 {
	if x >= 0 && x <= 1 {
		return x * 100
	}
	if x >= -100 && x <= 100 {
		return 50 + x/2
	}
	if x < 0 {
		return 0
	}
	if x > 100 {
		return 100
	}
	return x
}
func stable(p ...string) string {
	h := sha256.Sum256([]byte(strings.Join(p, "|")))
	return hex.EncodeToString(h[:])[:32]
}

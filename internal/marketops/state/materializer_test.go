package state

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestBuildG137PositiveAAPLVerticalSlice(t *testing.T) {
	start := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	events := equityFixtures(start, 25)
	session := start.AddDate(0, 0, 24)
	chain := usableSurfaceFixtures(session)
	distribution := storage.MarketOpsOptionsDistributionRecord{
		TenantID: "tenant-local", Symbol: "AAPL", TradeDate: session, WindowName: "10_trade_days",
		TotalCallOpenInterest: 100, TotalPutOpenInterest: 125, TotalCallVolume: 80, TotalPutVolume: 120,
		MetricsJSON: []byte(`{"open_interest_quality":"usable","call_put_oi_ratio_quality":"usable"}`),
	}
	result, err := Build(BuildConfig{TenantID: "tenant-local", Symbol: "AAPL", RunID: "g137-positive", MaxSessions: 100}, BuildInput{EquityEvents: events, Distributions: []storage.MarketOpsOptionsDistributionRecord{distribution}, OptionChain: chain})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Definitions) != 44 || len(result.States) != 25 || len(result.Observations) != 25*totalFeatureSlots {
		t.Fatalf("unexpected build counts: definitions=%d states=%d observations=%d", len(result.Definitions), len(result.States), len(result.Observations))
	}
	finalState := result.States[len(result.States)-1]
	if len(finalState.FeatureObservationIDs) != totalFeatureSlots || finalState.CompletenessRatio < .8 || finalState.QualityState != storage.MarketOpsQualityUsable || finalState.StateSchemaVersion != StateSchemaVersion {
		t.Fatalf("unexpected final state quality: %+v", finalState)
	}
	final := observationsForSession(result.Observations, session)
	assertNumericFeature(t, final, "atm_iv_30d", `{}`, .30, storage.MarketOpsQualityUsable)
	assertNumericFeature(t, final, "atm_iv_60d", `{}`, .32, storage.MarketOpsQualityUsable)
	assertNumericFeature(t, final, "atm_iv_90d", `{}`, .35, storage.MarketOpsQualityUsable)
	assertNumericFeature(t, final, "iv", `{"option_type":"put","target_delta":0.25,"target_dte":30}`, .38, storage.MarketOpsQualityUsable)
	assertNumericFeature(t, final, "put_call_oi_ratio", `{}`, 1.25, storage.MarketOpsQualityUsable)
	assertNumericFeature(t, final, "surface_coverage_ratio", `{}`, 1, storage.MarketOpsQualityUsable)
	assertNumericFeature(t, final, "iv_term_slope", `{"far_dte":60,"near_dte":30}`, .02, storage.MarketOpsQualityUsable)
	assertNumericFeature(t, final, "risk_reversal", `{"target_delta":0.25,"target_dte":30}`, -.10, storage.MarketOpsQualityUsable)
	assertQualityFeature(t, final, "surface_selection_confidence", storage.MarketOpsQualityUsableWithWarning)
	assertNumericFeature(t, final, "iv", `{"option_type":"put","target_delta":0.25,"target_dte":60}`, .36, storage.MarketOpsQualityUsable)
	assertNumericFeature(t, final, "mid_premium", `{"option_type":"put","target_delta":0.25,"target_dte":30}`, 2.1, storage.MarketOpsQualityUsable)
	if !hasEvidenceType(result.Evidence, "put_call_oi_ratio_observed") || !hasEvidenceType(result.Evidence, "underlying_return_observed") {
		t.Fatalf("expected quality-gated evidence, got %+v", result.Evidence)
	}
	if len(result.Transitions) == 0 {
		t.Fatal("expected one-session transitions")
	}
}

func TestPremiumFeaturesPreserveMissingInvalidAndStaleQuality(t *testing.T) {
	session := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	record := usableSurfaceFixtures(session)[3]
	dimensions := optionDimensions("put", 30, .25)
	record.Ask = nil
	missing := premiumFeatures(session, []storage.MarketOpsOptionsChainRecord{record}, 30, .25, "put", dimensions, nil)
	if missing[0].Quality != storage.MarketOpsQualityMissing {
		t.Fatalf("missing quote quality = %+v", missing)
	}
	bid, ask := 3.0, 2.0
	record.Bid, record.Ask = &bid, &ask
	invalid := premiumFeatures(session, []storage.MarketOpsOptionsChainRecord{record}, 30, .25, "put", dimensions, nil)
	if invalid[0].Quality != storage.MarketOpsQualityInvalid || invalid[0].Numeric != nil {
		t.Fatalf("crossed quote quality = %+v", invalid)
	}
	bid, ask = 2.0, 2.2
	staleTimestamp := session.AddDate(0, 0, -1)
	record.Bid, record.Ask, record.QuoteTimestamp = &bid, &ask, &staleTimestamp
	stale := premiumFeatures(session, []storage.MarketOpsOptionsChainRecord{record}, 30, .25, "put", dimensions, nil)
	if stale[0].Quality != storage.MarketOpsQualityStale || stale[0].Numeric == nil {
		t.Fatalf("stale quote quality = %+v", stale)
	}
}

func TestBuildEmitsDimensionedSurfaceOIChange(t *testing.T) {
	firstSession := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	secondSession := firstSession.AddDate(0, 0, 1)
	first, second := usableSurfaceFixtures(firstSession), usableSurfaceFixtures(secondSession)
	for index := range first {
		if first[index].OptionTicker == "O:AAPL-PUT25" {
			value := int64(100)
			first[index].OpenInterest = &value
		}
	}
	for index := range second {
		if second[index].OptionTicker == "O:AAPL-PUT25" {
			value := int64(275)
			second[index].OpenInterest = &value
		}
	}
	result, err := Build(BuildConfig{TenantID: "tenant-local", Symbol: "AAPL", RunID: "g143-oi"}, BuildInput{OptionChain: append(first, second...)})
	if err != nil {
		t.Fatal(err)
	}
	assertNumericFeature(t, observationsForSession(result.Observations, secondSession), "oi_change_1d", `{"option_type":"put","target_delta":0.25,"target_dte":30}`, 175, storage.MarketOpsQualityUsable)
}

func TestBuildBlocksUnusableOpenInterestEvidence(t *testing.T) {
	session := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	distribution := storage.MarketOpsOptionsDistributionRecord{
		TenantID: "tenant-local", Symbol: "AAPL", TradeDate: session, WindowName: "10_trade_days",
		TotalCallOpenInterest: 2, TotalPutOpenInterest: 0, TotalCallVolume: 6, TotalPutVolume: 1,
		MetricsJSON: []byte(`{"open_interest_quality":"partial_zero","call_put_oi_ratio_quality":"denominator_zero"}`),
	}
	chain := []storage.MarketOpsOptionsChainRecord{{TenantID: "tenant-local", Symbol: "AAPL", TradeDate: session, OptionTicker: "O:AAPL260720C00225000", ContractType: "call", ExpirationDate: session.AddDate(0, 0, 6), OpenInterest: int64Ptr(2)}}
	result, err := Build(BuildConfig{TenantID: "tenant-local", Symbol: "AAPL", RunID: "g137-blocked"}, BuildInput{Distributions: []storage.MarketOpsOptionsDistributionRecord{distribution}, OptionChain: chain})
	if err != nil {
		t.Fatal(err)
	}
	observations := observationsForSession(result.Observations, session)
	assertQualityFeature(t, observations, "put_call_oi_ratio", storage.MarketOpsQualityInvalid)
	if hasEvidenceType(result.Evidence, "put_call_oi_ratio_observed") {
		t.Fatal("unusable OI produced analytical evidence")
	}
	if len(result.States) != 1 || len(result.States[0].FeatureObservationIDs) != totalFeatureSlots {
		t.Fatalf("blocked state lost lineage: %+v", result.States)
	}
}

func TestBuildIsIdempotentAcrossRunIDs(t *testing.T) {
	input := BuildInput{EquityEvents: equityFixtures(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), 3)}
	first, err := Build(BuildConfig{TenantID: "tenant-local", Symbol: "AAPL", RunID: "run-one"}, input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Build(BuildConfig{TenantID: "tenant-local", Symbol: "AAPL", RunID: "run-two"}, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Observations) != len(second.Observations) || len(first.States) != len(second.States) || len(first.Transitions) != len(second.Transitions) || len(first.Evidence) != len(second.Evidence) {
		t.Fatal("rerun record counts changed")
	}
	for i := range first.Observations {
		if first.Observations[i].FeatureObservationID != second.Observations[i].FeatureObservationID || first.Observations[i].DeterministicKey != second.Observations[i].DeterministicKey {
			t.Fatalf("observation identity changed at %d", i)
		}
	}
	for i := range first.States {
		if first.States[i].MarketStateID != second.States[i].MarketStateID || first.States[i].DeterministicKey != second.States[i].DeterministicKey {
			t.Fatalf("state identity changed at %d", i)
		}
	}
	for i := range first.Evidence {
		if first.Evidence[i].EvidenceID != second.Evidence[i].EvidenceID {
			t.Fatalf("evidence identity changed at %d", i)
		}
	}
}

func TestBuildUsesPriorDistributionForFirstTargetOIChange(t *testing.T) {
	start := time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC)
	quality := []byte(`{"open_interest_quality":"usable","call_put_oi_ratio_quality":"usable"}`)
	input := BuildInput{Distributions: []storage.MarketOpsOptionsDistributionRecord{
		{TenantID: "tenant-local", Symbol: "AAPL", TradeDate: start, WindowName: "10_trade_days", TotalCallOpenInterest: 100, TotalPutOpenInterest: 100, MetricsJSON: quality},
		{TenantID: "tenant-local", Symbol: "AAPL", TradeDate: start.AddDate(0, 0, 1), WindowName: "10_trade_days", TotalCallOpenInterest: 100, TotalPutOpenInterest: 150, MetricsJSON: quality},
	}}
	result, err := Build(BuildConfig{TenantID: "tenant-local", Symbol: "AAPL", RunID: "run-oi-warmup", SessionStart: start.AddDate(0, 0, 1), SessionEnd: start.AddDate(0, 0, 2)}, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.States) != 1 {
		t.Fatalf("prior distribution leaked into states: %d", len(result.States))
	}
	assertNumericFeature(t, result.Observations, "put_call_oi_change_1d", `{}`, .5, storage.MarketOpsQualityUsable)
}

func TestBuildUsesPriorEquityAsWarmupWithoutMaterializingIt(t *testing.T) {
	start := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	input := BuildInput{EquityEvents: equityFixtures(start, 30)}
	result, err := Build(BuildConfig{TenantID: "tenant-local", Symbol: "AAPL", RunID: "run-warmup", SessionStart: start.AddDate(0, 0, 20), SessionEnd: start.AddDate(0, 0, 30), MaxSessions: 20}, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.States) != 10 || dateKey(result.States[0].SessionDate) != dateKey(start.AddDate(0, 0, 20)) {
		t.Fatalf("warmup sessions leaked into output: %+v", result.States)
	}
	first := observationsForSession(result.Observations, result.States[0].SessionDate)
	for _, observation := range first {
		if observation.FeatureKey == "return_20d" {
			if observation.NumericValue == nil || observation.QualityState != storage.MarketOpsQualityUsable {
				t.Fatalf("warmup history was not used: %+v", observation)
			}
			return
		}
	}
	t.Fatal("return_20d observation not found")
}

func TestBuildSupportsSixtyEligibleHistoricalSessions(t *testing.T) {
	input := BuildInput{EquityEvents: equityFixtures(time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), 65)}
	result, err := Build(BuildConfig{TenantID: "tenant-local", Symbol: "AAPL", RunID: "run-sixty", MaxSessions: 60}, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.States) != 60 || len(result.Observations) != 60*totalFeatureSlots {
		t.Fatalf("unexpected bounded replay counts: states=%d observations=%d", len(result.States), len(result.Observations))
	}
	for _, state := range result.States {
		if len(state.FeatureObservationIDs) != totalFeatureSlots {
			t.Fatalf("state %s does not preserve full lineage", state.MarketStateID)
		}
	}
}

func TestBuildSupportsExplicitNonAAPLSymbol(t *testing.T) {
	events := equityFixtures(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), 2)
	for index := range events {
		payload := map[string]any{}
		_ = json.Unmarshal(events[index].NormalizedPayload, &payload)
		payload["symbol"] = "MSFT"
		events[index].NormalizedPayload, _ = json.Marshal(payload)
	}
	result, err := Build(BuildConfig{TenantID: "tenant-local", Symbol: "msft", RunID: "g144-msft"}, BuildInput{EquityEvents: events})
	if err != nil || len(result.States) != 2 || result.States[0].Symbol != "MSFT" || result.States[0].AssetID != "ticker:MSFT" {
		t.Fatalf("explicit-symbol build failed: result=%+v err=%v", result, err)
	}
}

func equityFixtures(start time.Time, count int) []storage.NormalizedEventLedgerRecord {
	out := make([]storage.NormalizedEventLedgerRecord, 0, count)
	for i := 0; i < count; i++ {
		session := start.AddDate(0, 0, i)
		closeValue := 100 + float64(i)*1.25
		payload, _ := json.Marshal(map[string]any{"symbol": "AAPL", "open": closeValue - .5, "high": closeValue + 1, "low": closeValue - 1, "close": closeValue, "volume": 1_000_000 + i*10_000})
		out = append(out, storage.NormalizedEventLedgerRecord{EventID: fmt.Sprintf("evt-aapl-%02d", i), TenantID: "tenant-local", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", Dataset: "equity_eod_prices", ObservationTime: session, ProcessingTime: session.Add(time.Hour), NormalizedPayload: payload})
	}
	return out
}

func usableSurfaceFixtures(session time.Time) []storage.MarketOpsOptionsChainRecord {
	values := []struct {
		ticker, optionType string
		dte                int
		delta, iv          float64
	}{
		{"O:AAPL-ATM30C", "call", 30, .50, .30}, {"O:AAPL-ATM60C", "call", 60, .50, .32},
		{"O:AAPL-ATM90P", "put", 90, -.50, .35}, {"O:AAPL-PUT25", "put", 30, -.25, .38},
		{"O:AAPL-CALL25", "call", 30, .25, .28},
		{"O:AAPL-PUT25-60", "put", 60, -.25, .36}, {"O:AAPL-CALL25-60", "call", 60, .25, .29},
	}
	out := make([]storage.MarketOpsOptionsChainRecord, 0, len(values))
	for _, value := range values {
		openInterest := int64(20)
		underlying, bid, ask := 130.0, 2.0, 2.2
		quoteTimestamp := session.Add(20 * time.Hour)
		out = append(out, storage.MarketOpsOptionsChainRecord{TenantID: "tenant-local", Symbol: "AAPL", TradeDate: session, OptionTicker: value.ticker, ContractType: value.optionType, ExpirationDate: session.AddDate(0, 0, value.dte), StrikePrice: 130, UnderlyingClose: &underlying, OpenInterest: &openInterest, ImpliedVolatility: floatPtr(value.iv), Delta: floatPtr(value.delta), Bid: &bid, Ask: &ask, QuoteTimestamp: &quoteTimestamp})
	}
	return out
}

func observationsForSession(records []storage.MarketOpsFeatureObservationRecord, session time.Time) []storage.MarketOpsFeatureObservationRecord {
	out := []storage.MarketOpsFeatureObservationRecord{}
	for _, record := range records {
		if dateKey(record.SessionDate) == dateKey(session) {
			out = append(out, record)
		}
	}
	return out
}

func assertNumericFeature(t *testing.T, records []storage.MarketOpsFeatureObservationRecord, key, dimensions string, expected float64, quality string) {
	t.Helper()
	for _, record := range records {
		canonical, _ := CanonicalDimensions(record.DimensionsJSON)
		if record.FeatureKey == key && canonical == dimensions {
			if record.NumericValue == nil || mathAbs(*record.NumericValue-expected) > 0.000001 || record.QualityState != quality {
				t.Fatalf("unexpected %s observation: %+v", key, record)
			}
			return
		}
	}
	t.Fatalf("feature %s %s not found", key, dimensions)
}

func assertQualityFeature(t *testing.T, records []storage.MarketOpsFeatureObservationRecord, key, quality string) {
	t.Helper()
	for _, record := range records {
		if record.FeatureKey == key {
			if record.QualityState != quality {
				t.Fatalf("unexpected %s quality: %s", key, record.QualityState)
			}
			return
		}
	}
	t.Fatalf("feature %s not found", key)
}

func hasEvidenceType(records []storage.MarketOpsEvidenceRecord, evidenceType string) bool {
	for _, record := range records {
		if record.EvidenceType == evidenceType {
			return true
		}
	}
	return false
}

func int64Ptr(value int64) *int64 { return &value }
func mathAbs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

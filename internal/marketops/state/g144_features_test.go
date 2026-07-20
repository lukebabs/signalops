package state

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestG144RealizedVolatilityMaturesWithoutChangingRequiredCompleteness(t *testing.T) {
	start := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	result, err := Build(BuildConfig{TenantID: "tenant-local", Symbol: "AAPL", RunID: "g144-rv", MaxSessions: 65}, BuildInput{EquityEvents: equityFixtures(start, 65)})
	if err != nil {
		t.Fatal(err)
	}
	final := observationsForSession(result.Observations, start.AddDate(0, 0, 64))
	for _, key := range []string{"rv_10d", "rv_20d", "rv_60d", "rv_acceleration_5d"} {
		observation := findG144Observation(final, key, "{}")
		if observation.NumericValue == nil || observation.QualityState != storage.MarketOpsQualityUsable {
			t.Fatalf("%s did not mature: %+v", key, observation)
		}
	}
	finalState := result.States[len(result.States)-1]
	if finalState.RequiredFeatureCount != requiredFeatureSlots || finalState.FeatureCount != totalFeatureSlots {
		t.Fatalf("state completeness boundary changed: %+v", finalState)
	}
}

func TestG144SurfaceChangesUseEligibleSessionLookbacks(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	chain := []storage.MarketOpsOptionsChainRecord{}
	for day := 0; day < 6; day++ {
		session := start.AddDate(0, 0, day)
		records := usableSurfaceFixtures(session)
		for index := range records {
			*records[index].ImpliedVolatility += float64(day) * .01
			*records[index].Bid += float64(day) * .1
			*records[index].Ask += float64(day) * .1
			value := int64(100 + day*10)
			records[index].OpenInterest = &value
		}
		chain = append(chain, records...)
	}
	result, err := Build(BuildConfig{TenantID: "tenant-local", Symbol: "AAPL", RunID: "g144-options"}, BuildInput{OptionChain: chain})
	if err != nil {
		t.Fatal(err)
	}
	final := observationsForSession(result.Observations, start.AddDate(0, 0, 5))
	dims := `{"option_type":"put","target_delta":0.25,"target_dte":30}`
	assertNumericFeature(t, final, "iv_change_1d", dims, .01, storage.MarketOpsQualityUsable)
	assertNumericFeature(t, final, "iv_change_5d", dims, .05, storage.MarketOpsQualityUsable)
	assertNumericFeature(t, final, "premium_change_1d", dims, .1, storage.MarketOpsQualityUsable)
	assertNumericFeature(t, final, "premium_change_5d", dims, .5, storage.MarketOpsQualityUsable)
	assertNumericFeature(t, final, "oi_change_5d", dims, 50, storage.MarketOpsQualityUsable)
	term := findG144Observation(final, "term_structure_state", "{}")
	if term.TextValue == nil || *term.TextValue != "contango" || len(term.SourceArtifactIDs) != 3 {
		t.Fatalf("unexpected term structure state: %+v", term)
	}
}

func TestG144EarningsContextEnforcesPointInTimeKnowledge(t *testing.T) {
	session := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	events := []storage.NormalizedEventLedgerRecord{
		g144EarningsEvent("known-next", "2026-07-23", time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)),
		g144EarningsEvent("known-next-old", "2026-07-23", time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)),
		g144EarningsEvent("known-prior", "2026-07-18", time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)),
		g144EarningsEvent("learned-late", "2026-07-21", time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)),
	}
	result, err := Build(BuildConfig{TenantID: "tenant-local", Symbol: "AAPL", RunID: "g144-event"}, BuildInput{EquityEvents: equityFixtures(session, 1), EventEvents: events})
	if err != nil {
		t.Fatal(err)
	}
	observations := observationsForSession(result.Observations, session)
	assertNumericFeature(t, observations, "days_to_earnings", "{}", 3, storage.MarketOpsQualityUsable)
	assertNumericFeature(t, observations, "days_since_earnings", "{}", 2, storage.MarketOpsQualityUsable)
	state := findG144Observation(observations, "earnings_window_state", "{}")
	if state.TextValue == nil || *state.TextValue != "pre_earnings" || len(state.SourceEventIDs) != 2 {
		t.Fatalf("unexpected earnings state: %+v", state)
	}
	for _, eventID := range state.SourceEventIDs {
		if eventID == "learned-late" || eventID == "known-next-old" {
			t.Fatal("ineligible or superseded event leaked into point-in-time state")
		}
	}
}

func TestG144UnknownEarningsRemainExplicitlyMissing(t *testing.T) {
	session := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	values := earningsContextFeatures(BuildConfig{TenantID: "tenant-local", Symbol: "AAPL"}, session, []storage.NormalizedEventLedgerRecord{g144EarningsEvent("late", "2026-07-21", session.AddDate(0, 0, 2))})
	for _, value := range values {
		if value.Quality != storage.MarketOpsQualityMissing || value.Numeric != nil || value.Text != nil {
			t.Fatalf("unknown event was not missing: %+v", value)
		}
	}
}

func g144EarningsEvent(id, date string, knownAt time.Time) storage.NormalizedEventLedgerRecord {
	payload, _ := json.Marshal(map[string]any{"symbol": "AAPL", "event_type": "earnings", "event_date": date, "known_at": knownAt.Format(time.RFC3339)})
	return storage.NormalizedEventLedgerRecord{EventID: id, TenantID: "tenant-local", Dataset: "market_event_calendar", ProcessingTime: knownAt, NormalizedPayload: payload}
}

func findG144Observation(records []storage.MarketOpsFeatureObservationRecord, key, dimensions string) storage.MarketOpsFeatureObservationRecord {
	for _, record := range records {
		canonical, _ := CanonicalDimensions(record.DimensionsJSON)
		if record.FeatureKey == key && canonical == dimensions {
			return record
		}
	}
	return storage.MarketOpsFeatureObservationRecord{}
}

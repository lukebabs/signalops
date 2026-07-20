package state

import (
	"encoding/json"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const totalFeatureSlots = 69

func g144UnderlyingFeatures(history []equityPoint, index int) []featureValue {
	return []featureValue{
		realizedVolatilityFeature(history, index, 10),
		realizedVolatilityFeature(history, index, 20),
		realizedVolatilityFeature(history, index, 60),
		realizedVolatilityAccelerationFeature(history, index, 20, 5),
	}
}

func missingG144UnderlyingFeatures(reason string) []featureValue {
	return []featureValue{
		missingFeature("rv_10d", reason, nil, nil),
		missingFeature("rv_20d", reason, nil, nil),
		missingFeature("rv_60d", reason, nil, nil),
		missingFeature("rv_acceleration_5d", reason, nil, nil),
	}
}

func realizedVolatilityFeature(history []equityPoint, index, lookback int) featureValue {
	value, refs, ok := realizedVolatility(history, index, lookback)
	key := "rv_" + stringInt(lookback) + "d"
	if !ok {
		return missingFeature(key, "insufficient_or_invalid_equity_history", map[string]any{"available_sessions": index + 1, "required_sessions": lookback + 1}, refs)
	}
	return usableNumeric(key, value, refs, nil, map[string]any{"lookback_sessions": lookback, "annualization_sessions": 252, "return_method": "log", "variance_method": "population"})
}

func realizedVolatilityAccelerationFeature(history []equityPoint, index, rvLookback, changeLookback int) featureValue {
	current, currentRefs, currentOK := realizedVolatility(history, index, rvLookback)
	prior, priorRefs, priorOK := realizedVolatility(history, index-changeLookback, rvLookback)
	refs := uniqueStrings(append(currentRefs, priorRefs...))
	if !currentOK || !priorOK {
		return missingFeature("rv_acceleration_5d", "insufficient_or_invalid_equity_history", map[string]any{"rv_lookback_sessions": rvLookback, "change_lookback_sessions": changeLookback, "required_sessions": rvLookback + changeLookback + 1}, refs)
	}
	return usableNumeric("rv_acceleration_5d", round(current-prior, 8), refs, nil, map[string]any{"rv_lookback_sessions": rvLookback, "change_lookback_sessions": changeLookback})
}

func realizedVolatility(history []equityPoint, index, lookback int) (float64, []string, bool) {
	if index < lookback || index >= len(history) || index < 0 {
		return 0, eventIDs(history, 0, index), false
	}
	returns := make([]float64, 0, lookback)
	for i := index - lookback + 1; i <= index; i++ {
		if i <= 0 || !history[i-1].Valid || !history[i].Valid || history[i-1].Close <= 0 || history[i].Close <= 0 {
			return 0, eventIDs(history, index-lookback, index), false
		}
		returns = append(returns, math.Log(history[i].Close/history[i-1].Close))
	}
	mean := 0.0
	for _, value := range returns {
		mean += value
	}
	mean /= float64(len(returns))
	variance := 0.0
	for _, value := range returns {
		delta := value - mean
		variance += delta * delta
	}
	variance /= float64(len(returns))
	return round(math.Sqrt(variance)*math.Sqrt(252), 8), eventIDs(history, index-lookback, index), true
}

type surfaceTarget struct {
	dte        int
	delta      float64
	optionType string
	dimensions map[string]any
}

func g144OptionFeatures(session time.Time, chain []storage.MarketOpsOptionsChainRecord, allChain map[string][]storage.MarketOpsOptionsChainRecord) []featureValue {
	targets := []surfaceTarget{
		{30, .50, "", map[string]any{"surface_cell": "atm", "target_dte": 30, "target_delta": .50}},
		{60, .50, "", map[string]any{"surface_cell": "atm", "target_dte": 60, "target_delta": .50}},
		{90, .50, "", map[string]any{"surface_cell": "atm", "target_dte": 90, "target_delta": .50}},
		{30, .25, "put", optionDimensions("put", 30, .25)},
		{30, .25, "call", optionDimensions("call", 30, .25)},
		{60, .25, "put", optionDimensions("put", 60, .25)},
		{60, .25, "call", optionDimensions("call", 60, .25)},
	}
	out := make([]featureValue, 0, 21)
	for _, target := range targets {
		out = append(out, surfaceIVChange(session, chain, allChain, target, 1), surfaceIVChange(session, chain, allChain, target, 5))
	}
	out = append(out, termStructureState(session, chain))
	for _, optionType := range []string{"put", "call"} {
		target := surfaceTarget{30, .25, optionType, optionDimensions(optionType, 30, .25)}
		out = append(out, surfacePremiumChange(session, chain, allChain, target, 1), surfacePremiumChange(session, chain, allChain, target, 5))
		out = append(out, surfaceOIChange(session, chain, allChain, target, 5))
	}
	return out
}

func surfaceIVChange(session time.Time, chain []storage.MarketOpsOptionsChainRecord, allChain map[string][]storage.MarketOpsOptionsChainRecord, target surfaceTarget, lookback int) featureValue {
	key := "iv_change_" + stringInt(lookback) + "d"
	current, ok := selectContract(session, chain, target.dte, target.delta, target.optionType)
	if !ok {
		return missingFeatureWithDimensions(key, target.dimensions, "no_current_eligible_surface_contract", nil, nil, chainArtifactIDsForRecords(chain))
	}
	priorSession, prior, ok := nthPreviousSurfaceContract(session, allChain, target, lookback, nil)
	if !ok {
		return missingFeatureWithDimensions(key, target.dimensions, "insufficient_prior_eligible_surface_sessions", map[string]any{"required_prior_sessions": lookback}, nil, []string{optionArtifactID(current.Record)})
	}
	value := usableNumeric(key, round(*current.Record.ImpliedVolatility-*prior.Record.ImpliedVolatility, 8), nil, []string{optionArtifactID(current.Record), optionArtifactID(prior.Record)}, map[string]any{"lookback_sessions": lookback, "prior_session": dateKey(priorSession), "current_option_ticker": current.Record.OptionTicker, "prior_option_ticker": prior.Record.OptionTicker})
	value.Dimensions = target.dimensions
	return value
}

func surfacePremiumChange(session time.Time, chain []storage.MarketOpsOptionsChainRecord, allChain map[string][]storage.MarketOpsOptionsChainRecord, target surfaceTarget, lookback int) featureValue {
	key := "premium_change_" + stringInt(lookback) + "d"
	current, ok := selectContract(session, chain, target.dte, target.delta, target.optionType)
	if !ok {
		return missingFeatureWithDimensions(key, target.dimensions, "no_current_eligible_surface_contract", nil, nil, chainArtifactIDsForRecords(chain))
	}
	if _, currentOK := validSessionMidpoint(session, current.Record); !currentOK {
		return missingFeatureWithDimensions(key, target.dimensions, "missing_or_invalid_current_point_in_time_quote", nil, nil, []string{optionArtifactID(current.Record)})
	}
	priorSession, prior, ok := nthPreviousSurfaceContract(session, allChain, target, lookback, func(priorSession time.Time, record storage.MarketOpsOptionsChainRecord) bool {
		_, valid := validSessionMidpoint(priorSession, record)
		return valid
	})
	if !ok {
		return missingFeatureWithDimensions(key, target.dimensions, "insufficient_prior_eligible_quote_sessions", map[string]any{"required_prior_sessions": lookback}, nil, []string{optionArtifactID(current.Record)})
	}
	currentMid, currentOK := validSessionMidpoint(session, current.Record)
	priorMid, priorOK := validSessionMidpoint(priorSession, prior.Record)
	if !currentOK || !priorOK {
		return missingFeatureWithDimensions(key, target.dimensions, "missing_or_invalid_point_in_time_quote", map[string]any{"prior_session": dateKey(priorSession)}, nil, []string{optionArtifactID(current.Record), optionArtifactID(prior.Record)})
	}
	value := usableNumeric(key, round(currentMid-priorMid, 8), nil, []string{optionArtifactID(current.Record), optionArtifactID(prior.Record)}, map[string]any{"lookback_sessions": lookback, "prior_session": dateKey(priorSession), "current_option_ticker": current.Record.OptionTicker, "prior_option_ticker": prior.Record.OptionTicker})
	value.Dimensions = target.dimensions
	return value
}

func surfaceOIChange(session time.Time, chain []storage.MarketOpsOptionsChainRecord, allChain map[string][]storage.MarketOpsOptionsChainRecord, target surfaceTarget, lookback int) featureValue {
	key := "oi_change_" + stringInt(lookback) + "d"
	current, ok := selectContract(session, chain, target.dte, target.delta, target.optionType)
	if !ok || current.Record.OpenInterest == nil {
		return missingFeatureWithDimensions(key, target.dimensions, "missing_current_usable_surface_open_interest", nil, nil, chainArtifactIDsForRecords(chain))
	}
	priorSession, prior, ok := nthPreviousSurfaceContract(session, allChain, target, lookback, func(_ time.Time, record storage.MarketOpsOptionsChainRecord) bool { return record.OpenInterest != nil })
	if !ok {
		return missingFeatureWithDimensions(key, target.dimensions, "insufficient_prior_eligible_open_interest_sessions", map[string]any{"required_prior_sessions": lookback}, nil, []string{optionArtifactID(current.Record)})
	}
	value := usableNumeric(key, float64(*current.Record.OpenInterest-*prior.Record.OpenInterest), nil, []string{optionArtifactID(current.Record), optionArtifactID(prior.Record)}, map[string]any{"lookback_sessions": lookback, "prior_session": dateKey(priorSession), "current_option_ticker": current.Record.OptionTicker, "prior_option_ticker": prior.Record.OptionTicker})
	value.Dimensions = target.dimensions
	return value
}

func termStructureState(session time.Time, chain []storage.MarketOpsOptionsChainRecord) featureValue {
	values := make([]selectedContract, 0, 3)
	for _, dte := range []int{30, 60, 90} {
		selected, ok := selectContract(session, chain, dte, .50, "")
		if !ok {
			return missingFeature("term_structure_state", "missing_usable_atm_curve_cell", map[string]any{"target_dte": dte}, nil)
		}
		values = append(values, selected)
	}
	iv30, iv60, iv90 := *values[0].Record.ImpliedVolatility, *values[1].Record.ImpliedVolatility, *values[2].Record.ImpliedVolatility
	nearSlope, farSlope, tolerance := iv60-iv30, iv90-iv60, .0025
	state := "mixed"
	if math.Abs(nearSlope) <= tolerance && math.Abs(farSlope) <= tolerance {
		state = "flat"
	} else if nearSlope > tolerance && farSlope > tolerance {
		state = "contango"
	} else if nearSlope < -tolerance && farSlope < -tolerance {
		state = "backwardation"
	}
	refs := []string{optionArtifactID(values[0].Record), optionArtifactID(values[1].Record), optionArtifactID(values[2].Record)}
	return featureValue{Key: "term_structure_state", Text: &state, Quality: storage.MarketOpsQualityUsable, QualityScore: 1, Details: map[string]any{"near_slope": round(nearSlope, 8), "far_slope": round(farSlope, 8), "flat_tolerance": tolerance}, ArtifactIDs: refs}
}

func crossDomainG144Features(values []featureValue) []featureValue {
	atm, atmOK := findUsableNumericValue(values, "atm_iv_30d", nil)
	rv, rvOK := findUsableNumericValue(values, "rv_20d", nil)
	refs := uniqueStrings(append(append([]string{}, atm.ArtifactIDs...), rv.EventIDs...))
	if !atmOK || !rvOK {
		return []featureValue{
			missingFeatureWithDimensions("iv_minus_rv_20d", nil, "missing_usable_iv_or_realized_volatility", nil, rv.EventIDs, atm.ArtifactIDs),
			missingFeatureWithDimensions("iv_rv_ratio_20d", nil, "missing_usable_iv_or_realized_volatility", nil, rv.EventIDs, atm.ArtifactIDs),
		}
	}
	spread := usableNumeric("iv_minus_rv_20d", round(*atm.Numeric-*rv.Numeric, 8), rv.EventIDs, atm.ArtifactIDs, map[string]any{"iv_feature": "atm_iv_30d", "rv_feature": "rv_20d"})
	if *rv.Numeric <= 0 {
		return []featureValue{spread, missingFeatureWithDimensions("iv_rv_ratio_20d", nil, "non_positive_realized_volatility", nil, rv.EventIDs, atm.ArtifactIDs)}
	}
	ratio := usableNumeric("iv_rv_ratio_20d", round(*atm.Numeric / *rv.Numeric, 8), rv.EventIDs, atm.ArtifactIDs, map[string]any{"iv_feature": "atm_iv_30d", "rv_feature": "rv_20d", "source_refs": refs})
	return []featureValue{spread, ratio}
}

func findUsableNumericValue(values []featureValue, key string, dimensions map[string]any) (featureValue, bool) {
	for _, value := range values {
		if value.Key != key || value.Numeric == nil || !isUsable(value.Quality) {
			continue
		}
		if dimensions == nil || mapsEqual(value.Dimensions, dimensions) {
			return value, true
		}
	}
	return featureValue{}, false
}

func mapsEqual(left, right map[string]any) bool {
	leftJSON, _ := json.Marshal(left)
	rightJSON, _ := json.Marshal(right)
	leftCanonical, _ := CanonicalDimensions(leftJSON)
	rightCanonical, _ := CanonicalDimensions(rightJSON)
	return leftCanonical == rightCanonical
}

func nthPreviousSurfaceContract(session time.Time, allChain map[string][]storage.MarketOpsOptionsChainRecord, target surfaceTarget, lookback int, eligible func(time.Time, storage.MarketOpsOptionsChainRecord) bool) (time.Time, selectedContract, bool) {
	dates := make([]time.Time, 0, len(allChain))
	for key := range allChain {
		date, err := time.Parse("2006-01-02", key)
		if err == nil && date.Before(dayOnly(session)) {
			dates = append(dates, date)
		}
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].After(dates[j]) })
	if lookback <= 0 {
		return time.Time{}, selectedContract{}, false
	}
	matched := 0
	for _, date := range dates {
		selected, ok := selectContract(date, allChain[dateKey(date)], target.dte, target.delta, target.optionType)
		if !ok || (eligible != nil && !eligible(date, selected.Record)) {
			continue
		}
		matched++
		if matched == lookback {
			return date, selected, true
		}
	}
	return time.Time{}, selectedContract{}, false
}

func validSessionMidpoint(session time.Time, record storage.MarketOpsOptionsChainRecord) (float64, bool) {
	if record.Bid == nil || record.Ask == nil || *record.Bid <= 0 || *record.Ask <= 0 || *record.Ask < *record.Bid || record.QuoteTimestamp == nil || !dayOnly(*record.QuoteTimestamp).Equal(dayOnly(session)) {
		return 0, false
	}
	return (*record.Bid + *record.Ask) / 2, true
}

func chainArtifactIDsForRecords(records []storage.MarketOpsOptionsChainRecord) []string {
	out := make([]string, 0, len(records))
	for _, record := range records {
		out = append(out, optionArtifactID(record))
	}
	return uniqueStrings(out)
}

type knownEarningsEvent struct {
	date    time.Time
	knownAt time.Time
	eventID string
}

func earningsContextFeatures(config BuildConfig, session time.Time, events []storage.NormalizedEventLedgerRecord) []featureValue {
	eligible := make([]knownEarningsEvent, 0, len(events))
	for _, event := range events {
		if event.TenantID != config.TenantID || event.Dataset != "market_event_calendar" {
			continue
		}
		payload := map[string]any{}
		if json.Unmarshal(event.NormalizedPayload, &payload) != nil || strings.ToUpper(stringValue(payload["symbol"])) != config.Symbol || strings.ToLower(stringValue(payload["event_type"])) != "earnings" {
			continue
		}
		eventDate, err := time.Parse("2006-01-02", stringValue(payload["event_date"]))
		if err != nil {
			continue
		}
		knownAt := event.ProcessingTime
		if value := stringValue(payload["known_at"]); value != "" {
			if parsed, parseErr := time.Parse(time.RFC3339, value); parseErr == nil {
				knownAt = parsed
			}
		}
		if knownAt.IsZero() || knownAt.After(sessionEnd(session)) {
			continue
		}
		eligible = append(eligible, knownEarningsEvent{date: dayOnly(eventDate), knownAt: knownAt.UTC(), eventID: event.EventID})
	}
	byDate := map[string]knownEarningsEvent{}
	for _, candidate := range eligible {
		key := dateKey(candidate.date)
		current, exists := byDate[key]
		if !exists || candidate.knownAt.After(current.knownAt) || (candidate.knownAt.Equal(current.knownAt) && candidate.eventID > current.eventID) {
			byDate[key] = candidate
		}
	}
	eligible = eligible[:0]
	for _, candidate := range byDate {
		eligible = append(eligible, candidate)
	}
	if len(eligible) == 0 {
		return []featureValue{
			missingFeature("days_to_earnings", "no_point_in_time_known_earnings_event", nil, nil),
			missingFeature("days_since_earnings", "no_point_in_time_known_earnings_event", nil, nil),
			{Key: "earnings_window_state", Quality: storage.MarketOpsQualityMissing, Details: map[string]any{"reason": "no_point_in_time_known_earnings_event"}},
		}
	}
	sort.Slice(eligible, func(i, j int) bool { return eligible[i].date.Before(eligible[j].date) })
	var next, prior *knownEarningsEvent
	for index := range eligible {
		candidate := &eligible[index]
		if !candidate.date.Before(dayOnly(session)) && next == nil {
			next = candidate
		}
		if !candidate.date.After(dayOnly(session)) {
			prior = candidate
		}
	}
	daysTo := missingFeature("days_to_earnings", "no_next_point_in_time_known_earnings_event", nil, nil)
	daysSince := missingFeature("days_since_earnings", "no_prior_point_in_time_known_earnings_event", nil, nil)
	windowState, refs := "outside_window", []string{}
	if next != nil {
		days := int(next.date.Sub(dayOnly(session)).Hours() / 24)
		daysTo = usableNumeric("days_to_earnings", float64(days), []string{next.eventID}, nil, map[string]any{"event_date": dateKey(next.date), "known_at": next.knownAt.Format(time.RFC3339Nano)})
		refs = append(refs, next.eventID)
		if days == 0 {
			windowState = "earnings_day"
		} else if days <= 5 {
			windowState = "pre_earnings"
		}
	}
	if prior != nil {
		days := int(dayOnly(session).Sub(prior.date).Hours() / 24)
		daysSince = usableNumeric("days_since_earnings", float64(days), []string{prior.eventID}, nil, map[string]any{"event_date": dateKey(prior.date), "known_at": prior.knownAt.Format(time.RFC3339Nano)})
		refs = append(refs, prior.eventID)
		if days == 0 {
			windowState = "earnings_day"
		} else if days <= 2 && windowState == "outside_window" {
			windowState = "post_earnings"
		}
	}
	refs = uniqueStrings(refs)
	state := featureValue{Key: "earnings_window_state", Text: &windowState, Quality: storage.MarketOpsQualityUsable, QualityScore: 1, Details: map[string]any{"pre_event_days": 5, "post_event_days": 2, "point_in_time_safe": true}, EventIDs: refs}
	return []featureValue{daysTo, daysSince, state}
}

func stringInt(value int) string {
	if value == 1 {
		return "1"
	}
	if value == 5 {
		return "5"
	}
	if value == 10 {
		return "10"
	}
	if value == 20 {
		return "20"
	}
	if value == 60 {
		return "60"
	}
	return "0"
}

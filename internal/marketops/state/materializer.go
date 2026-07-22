package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const requiredFeatureSlots = 39 // G137/G143 hypothesis-critical slots; G144 longitudinal slots mature independently.

type BuildConfig struct {
	TenantID     string
	Symbol       string
	AssetID      string
	RunID        string
	SessionStart time.Time
	SessionEnd   time.Time
	MaxSessions  int
}

type BuildInput struct {
	EquityEvents  []storage.NormalizedEventLedgerRecord
	EventEvents   []storage.NormalizedEventLedgerRecord
	Distributions []storage.MarketOpsOptionsDistributionRecord
	OptionChain   []storage.MarketOpsOptionsChainRecord
}

type BuildResult struct {
	Definitions  []storage.MarketOpsFeatureDefinitionRecord
	Observations []storage.MarketOpsFeatureObservationRecord
	States       []storage.MarketOpsMarketStateRecord
	Transitions  []storage.MarketOpsStateTransitionRecord
	Evidence     []storage.MarketOpsEvidenceRecord
}

type equityPoint struct {
	Session time.Time
	EventID string
	Open    float64
	High    float64
	Low     float64
	Close   float64
	Volume  float64
	Valid   bool
}

type featureValue struct {
	Key          string
	Dimensions   map[string]any
	Numeric      *float64
	Text         *string
	Quality      string
	QualityScore float64
	Details      map[string]any
	EventIDs     []string
	ArtifactIDs  []string
}

type selectedContract struct {
	Record storage.MarketOpsOptionsChainRecord
	DTE    int
	Score  float64
}

func Build(config BuildConfig, input BuildInput) (BuildResult, error) {
	config = normalizeBuildConfig(config)
	if err := validateBuildConfig(config); err != nil {
		return BuildResult{}, err
	}

	equity, err := equityHistory(config, input.EquityEvents)
	if err != nil {
		return BuildResult{}, err
	}
	distributions := distributionsBySession(config, input.Distributions)
	chain := chainBySession(config, input.OptionChain)
	sessions := buildSessions(equity, distributions, chain, config.SessionStart, config.SessionEnd, config.MaxSessions)
	result := BuildResult{Definitions: FeatureDefinitions(config.TenantID)}
	if len(sessions) == 0 {
		return result, nil
	}

	equityIndex := map[string]int{}
	for i, point := range equity {
		equityIndex[dateKey(point.Session)] = i
	}
	observationsBySession := map[string][]storage.MarketOpsFeatureObservationRecord{}
	for _, session := range sessions {
		key := dateKey(session)
		values := make([]featureValue, 0, totalFeatureSlots)
		if index, ok := equityIndex[key]; ok {
			values = append(values, underlyingFeatures(equity, index)...)
		} else {
			values = append(values, missingUnderlyingFeatures()...)
		}
		// Preserve the original 39 slots first so required completeness remains stable.
		values = append(values, optionFeatures(config, session, distributions[key], distributions, chain[key], chain)...)
		if index, ok := equityIndex[key]; ok {
			values = append(values, g144UnderlyingFeatures(equity, index)...)
		} else {
			values = append(values, missingG144UnderlyingFeatures("no_equity_event_for_session")...)
		}
		values = append(values, g144OptionFeatures(session, chain[key], chain)...)
		values = append(values, crossDomainG144Features(values)...)
		values = append(values, earningsContextFeatures(config, session, input.EventEvents)...)
		if len(values) != totalFeatureSlots {
			return BuildResult{}, fmt.Errorf("G144 feature slot count for %s is %d, expected %d", key, len(values), totalFeatureSlots)
		}

		observations := make([]storage.MarketOpsFeatureObservationRecord, 0, len(values))
		for _, value := range values {
			observation, err := observationRecord(config, session, value)
			if err != nil {
				return BuildResult{}, err
			}
			observations = append(observations, observation)
		}
		observationsBySession[key] = observations
		result.Observations = append(result.Observations, observations...)
		stateRecord, err := marketStateRecord(config, session, observations)
		if err != nil {
			return BuildResult{}, err
		}
		result.States = append(result.States, stateRecord)
	}

	transitions, err := buildTransitions(config, result.States, observationsBySession)
	if err != nil {
		return BuildResult{}, err
	}
	g144Transitions, err := buildG144Transitions(config, result.States, observationsBySession)
	if err != nil {
		return BuildResult{}, err
	}
	transitions = append(transitions, g144Transitions...)
	result.Transitions = transitions
	evidence, err := buildEvidence(config, result.States, observationsBySession, transitions)
	if err != nil {
		return BuildResult{}, err
	}
	result.Evidence = evidence
	return result, nil
}

func normalizeBuildConfig(config BuildConfig) BuildConfig {
	config.TenantID = strings.TrimSpace(config.TenantID)
	config.Symbol = strings.ToUpper(strings.TrimSpace(config.Symbol))
	config.AssetID = strings.TrimSpace(config.AssetID)
	config.RunID = strings.TrimSpace(config.RunID)
	if config.AssetID == "" && config.Symbol != "" {
		config.AssetID = "ticker:" + config.Symbol
	}
	if !config.SessionStart.IsZero() {
		config.SessionStart = dayOnly(config.SessionStart)
	}
	if !config.SessionEnd.IsZero() {
		config.SessionEnd = dayOnly(config.SessionEnd)
	}
	if config.MaxSessions <= 0 || config.MaxSessions > 1000 {
		config.MaxSessions = 100
	}
	return config
}

func validateBuildConfig(config BuildConfig) error {
	if config.TenantID == "" || config.Symbol == "" || config.AssetID == "" || config.RunID == "" {
		return errors.New("G144 tenant_id, symbol, asset_id, and run_id are required")
	}
	if (!config.SessionStart.IsZero() || !config.SessionEnd.IsZero()) && (config.SessionStart.IsZero() || config.SessionEnd.IsZero() || !config.SessionEnd.After(config.SessionStart)) {
		return errors.New("G144 session_end must be after session_start")
	}
	return nil
}

func equityHistory(config BuildConfig, events []storage.NormalizedEventLedgerRecord) ([]equityPoint, error) {
	bySession := map[string]storage.NormalizedEventLedgerRecord{}
	for _, event := range events {
		if event.TenantID != config.TenantID || event.Dataset != "equity_eod_prices" {
			continue
		}
		payload := map[string]any{}
		if err := json.Unmarshal(event.NormalizedPayload, &payload); err != nil {
			return nil, fmt.Errorf("decode equity event %s: %w", event.EventID, err)
		}
		if strings.ToUpper(stringValue(payload["symbol"])) != config.Symbol {
			continue
		}
		key := dateKey(event.ObservationTime)
		current, exists := bySession[key]
		if !exists || event.ProcessingTime.After(current.ProcessingTime) || (event.ProcessingTime.Equal(current.ProcessingTime) && event.EventID > current.EventID) {
			bySession[key] = event
		}
	}
	points := make([]equityPoint, 0, len(bySession))
	for _, event := range bySession {
		payload := map[string]any{}
		_ = json.Unmarshal(event.NormalizedPayload, &payload)
		open, okOpen := numberValue(payload["open"])
		high, okHigh := numberValue(payload["high"])
		low, okLow := numberValue(payload["low"])
		closeValue, okClose := numberValue(payload["close"])
		volume, okVolume := numberValue(payload["volume"])
		points = append(points, equityPoint{Session: dayOnly(event.ObservationTime), EventID: event.EventID, Open: open, High: high, Low: low, Close: closeValue, Volume: volume, Valid: okOpen && okHigh && okLow && okClose && okVolume && open > 0 && high > 0 && low > 0 && closeValue > 0 && volume >= 0})
	}
	sort.Slice(points, func(i, j int) bool { return points[i].Session.Before(points[j].Session) })
	return points, nil
}

func distributionsBySession(config BuildConfig, records []storage.MarketOpsOptionsDistributionRecord) map[string]*storage.MarketOpsOptionsDistributionRecord {
	out := map[string]*storage.MarketOpsOptionsDistributionRecord{}
	for i := range records {
		record := records[i]
		if record.TenantID != config.TenantID || strings.ToUpper(record.Symbol) != config.Symbol {
			continue
		}
		key := dateKey(record.TradeDate)
		current, exists := out[key]
		if !exists || record.UpdatedAt.After(current.UpdatedAt) || (record.UpdatedAt.Equal(current.UpdatedAt) && record.WindowName < current.WindowName) {
			copyRecord := record
			out[key] = &copyRecord
		}
	}
	return out
}

func chainBySession(config BuildConfig, records []storage.MarketOpsOptionsChainRecord) map[string][]storage.MarketOpsOptionsChainRecord {
	out := map[string][]storage.MarketOpsOptionsChainRecord{}
	for _, record := range records {
		if record.TenantID == config.TenantID && strings.ToUpper(record.Symbol) == config.Symbol {
			key := dateKey(record.TradeDate)
			out[key] = append(out[key], record)
		}
	}
	for key := range out {
		sort.Slice(out[key], func(i, j int) bool { return out[key][i].OptionTicker < out[key][j].OptionTicker })
	}
	return out
}

func buildSessions(equity []equityPoint, distributions map[string]*storage.MarketOpsOptionsDistributionRecord, chain map[string][]storage.MarketOpsOptionsChainRecord, sessionStart, sessionEnd time.Time, maxSessions int) []time.Time {
	values := map[string]time.Time{}
	for _, point := range equity {
		values[dateKey(point.Session)] = point.Session
	}
	for key, record := range distributions {
		values[key] = dayOnly(record.TradeDate)
	}
	for key, records := range chain {
		if len(records) > 0 {
			values[key] = dayOnly(records[0].TradeDate)
		}
	}
	out := make([]time.Time, 0, len(values))
	for _, value := range values {
		if !sessionStart.IsZero() && value.Before(sessionStart) {
			continue
		}
		if !sessionEnd.IsZero() && !value.Before(sessionEnd) {
			continue
		}
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Before(out[j]) })
	if len(out) > maxSessions {
		out = out[:maxSessions]
	}
	return out
}

func underlyingFeatures(history []equityPoint, index int) []featureValue {
	point := history[index]
	if !point.Valid {
		return invalidUnderlyingFeatures(point.EventID)
	}
	return []featureValue{
		returnFeature(history, index, 1), returnFeature(history, index, 5), returnFeature(history, index, 10), returnFeature(history, index, 20),
		rsiFeature(history, index, 14), smaDistanceFeature(history, index, 20), volumeRatioFeature(history, index, 20), gapFeature(history, index), atrFeature(history, index, 14),
		rangePositionFeature(history, index, 252), volumeRatioFeature(history, index, 10), smaDistanceFeature(history, index, 50), smaDistanceFeature(history, index, 200), smaSlopeFeature(history, index, 50, 20),
	}
}

func missingUnderlyingFeatures() []featureValue {
	keys := []string{"return_1d", "return_5d", "return_10d", "return_20d", "rsi_14", "distance_sma_20_pct", "volume_ratio_20d", "gap_pct", "atr_14_pct", "range_position_252d", "volume_ratio_10d", "distance_sma_50_pct", "distance_sma_200_pct", "sma_50_slope_20d_pct"}
	out := make([]featureValue, 0, len(keys))
	for _, key := range keys {
		out = append(out, missingFeature(key, "no_equity_event_for_session", nil, nil))
	}
	return out
}

func invalidUnderlyingFeatures(eventID string) []featureValue {
	values := missingUnderlyingFeatures()
	for i := range values {
		values[i].Quality = storage.MarketOpsQualityInvalid
		values[i].Details = map[string]any{"reason": "invalid_equity_price_fields"}
		values[i].EventIDs = []string{eventID}
	}
	return values
}

func returnFeature(history []equityPoint, index, lookback int) featureValue {
	key := fmt.Sprintf("return_%dd", lookback)
	if index < lookback {
		return missingFeature(key, "insufficient_equity_history", map[string]any{"available_sessions": index + 1, "required_sessions": lookback + 1}, eventIDs(history, 0, index))
	}
	baseline := history[index-lookback]
	current := history[index]
	if !baseline.Valid || baseline.Close <= 0 {
		return invalidFeature(key, "invalid_baseline_close", eventIDs(history, index-lookback, index), nil)
	}
	value := round((current.Close/baseline.Close-1)*100, 6)
	return usableNumeric(key, value, eventIDs(history, index-lookback, index), nil, map[string]any{"lookback_sessions": lookback})
}

func rsiFeature(history []equityPoint, index, period int) featureValue {
	if index < period {
		return missingFeature("rsi_14", "insufficient_equity_history", map[string]any{"available_sessions": index + 1, "required_sessions": period + 1}, eventIDs(history, 0, index))
	}
	gain, loss := 0.0, 0.0
	for i := 1; i <= period; i++ {
		delta := history[i].Close - history[i-1].Close
		if delta >= 0 {
			gain += delta
		} else {
			loss -= delta
		}
	}
	avgGain, avgLoss := gain/float64(period), loss/float64(period)
	for i := period + 1; i <= index; i++ {
		delta := history[i].Close - history[i-1].Close
		currentGain, currentLoss := 0.0, 0.0
		if delta >= 0 {
			currentGain = delta
		} else {
			currentLoss = -delta
		}
		avgGain = (avgGain*float64(period-1) + currentGain) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + currentLoss) / float64(period)
	}
	value := 100.0
	if avgLoss > 0 {
		value = 100 - 100/(1+avgGain/avgLoss)
	}
	return usableNumeric("rsi_14", round(value, 6), eventIDs(history, 0, index), nil, map[string]any{"lookback_sessions": period, "method": "wilder"})
}

func smaDistanceFeature(history []equityPoint, index, period int) featureValue {
	key := fmt.Sprintf("distance_sma_%d_pct", period)
	if index+1 < period {
		return missingFeature(key, "insufficient_equity_history", map[string]any{"available_sessions": index + 1, "required_sessions": period}, eventIDs(history, 0, index))
	}
	sum := 0.0
	for i := index - period + 1; i <= index; i++ {
		sum += history[i].Close
	}
	mean := sum / float64(period)
	value := (history[index].Close/mean - 1) * 100
	return usableNumeric(key, round(value, 6), eventIDs(history, index-period+1, index), nil, map[string]any{"lookback_sessions": period})
}

func rangePositionFeature(history []equityPoint, index, period int) featureValue {
	if index+1 < period {
		return missingFeature("range_position_252d", "insufficient_equity_history", map[string]any{"available_sessions": index + 1, "required_sessions": period}, eventIDs(history, 0, index))
	}
	low, high := history[index-period+1].Low, history[index-period+1].High
	for i := index - period + 1; i <= index; i++ {
		if !history[i].Valid { return invalidFeature("range_position_252d", "invalid_equity_history", eventIDs(history, index-period+1, index), nil) }
		low, high = math.Min(low, history[i].Low), math.Max(high, history[i].High)
	}
	if high <= low { return invalidFeature("range_position_252d", "invalid_252_session_range", eventIDs(history, index-period+1, index), nil) }
	return usableNumeric("range_position_252d", round((history[index].Close-low)/(high-low)*100, 6), eventIDs(history, index-period+1, index), nil, map[string]any{"lookback_sessions": period, "range_low": low, "range_high": high})
}

func smaSlopeFeature(history []equityPoint, index, period, slopeLookback int) featureValue {
	key := fmt.Sprintf("sma_%d_slope_%dd_pct", period, slopeLookback)
	if index+1 < period+slopeLookback {
		return missingFeature(key, "insufficient_equity_history", map[string]any{"available_sessions": index + 1, "required_sessions": period + slopeLookback}, eventIDs(history, 0, index))
	}
	mean := func(end int) float64 { sum := 0.0; for i := end-period+1; i <= end; i++ { sum += history[i].Close }; return sum / float64(period) }
	prior, current := mean(index-slopeLookback), mean(index)
	if prior <= 0 { return invalidFeature(key, "non_positive_prior_sma", eventIDs(history, index-period-slopeLookback+1, index), nil) }
	return usableNumeric(key, round((current/prior-1)*100, 6), eventIDs(history, index-period-slopeLookback+1, index), nil, map[string]any{"sma_sessions": period, "slope_lookback_sessions": slopeLookback})
}

func volumeRatioFeature(history []equityPoint, index, period int) featureValue {
	key := fmt.Sprintf("volume_ratio_%dd", period)
	if index < period {
		return missingFeature(key, "insufficient_equity_history", map[string]any{"available_prior_sessions": index, "required_prior_sessions": period}, eventIDs(history, 0, index))
	}
	sum := 0.0
	for i := index - period; i < index; i++ {
		sum += history[i].Volume
	}
	mean := sum / float64(period)
	if mean <= 0 {
		return invalidFeature(key, "non_positive_trailing_volume", eventIDs(history, index-period, index), nil)
	}
	return usableNumeric(key, round(history[index].Volume/mean, 6), eventIDs(history, index-period, index), nil, map[string]any{"lookback_sessions": period, "excludes_current_session": true})
}

func gapFeature(history []equityPoint, index int) featureValue {
	if index < 1 {
		return missingFeature("gap_pct", "missing_prior_equity_session", map[string]any{"available_sessions": 1}, eventIDs(history, 0, index))
	}
	prior := history[index-1]
	if prior.Close <= 0 {
		return invalidFeature("gap_pct", "invalid_prior_close", eventIDs(history, index-1, index), nil)
	}
	return usableNumeric("gap_pct", round((history[index].Open/prior.Close-1)*100, 6), eventIDs(history, index-1, index), nil, nil)
}

func atrFeature(history []equityPoint, index, period int) featureValue {
	if index < period {
		return missingFeature("atr_14_pct", "insufficient_equity_history", map[string]any{"available_sessions": index + 1, "required_sessions": period + 1}, eventIDs(history, 0, index))
	}
	trueRange := func(i int) float64 {
		return math.Max(history[i].High-history[i].Low, math.Max(math.Abs(history[i].High-history[i-1].Close), math.Abs(history[i].Low-history[i-1].Close)))
	}
	sum := 0.0
	for i := 1; i <= period; i++ {
		sum += trueRange(i)
	}
	atr := sum / float64(period)
	for i := period + 1; i <= index; i++ {
		atr = (atr*float64(period-1) + trueRange(i)) / float64(period)
	}
	return usableNumeric("atr_14_pct", round(atr/history[index].Close*100, 6), eventIDs(history, 0, index), nil, map[string]any{"lookback_sessions": period, "method": "wilder"})
}

func optionFeatures(config BuildConfig, session time.Time, distribution *storage.MarketOpsOptionsDistributionRecord, distributions map[string]*storage.MarketOpsOptionsDistributionRecord, chain []storage.MarketOpsOptionsChainRecord, allChain map[string][]storage.MarketOpsOptionsChainRecord) []featureValue {
	chainRefs := chainArtifactIDs(config, session, chain)
	atm30 := ivCell("atm_iv_30d", session, chain, 30, 0.50, "", nil, chainRefs)
	atm60 := ivCell("atm_iv_60d", session, chain, 60, 0.50, "", nil, chainRefs)
	atm90 := ivCell("atm_iv_90d", session, chain, 90, 0.50, "", nil, chainRefs)
	put30Dims := optionDimensions("put", 30, .25)
	call30Dims := optionDimensions("call", 30, .25)
	put60Dims := optionDimensions("put", 60, .25)
	call60Dims := optionDimensions("call", 60, .25)
	put30 := ivCell("iv", session, chain, 30, .25, "put", put30Dims, chainRefs)
	call30 := ivCell("iv", session, chain, 30, .25, "call", call30Dims, chainRefs)
	put60 := ivCell("iv", session, chain, 60, .25, "put", put60Dims, chainRefs)
	call60 := ivCell("iv", session, chain, 60, .25, "call", call60Dims, chainRefs)

	cells := []featureValue{atm30, atm60, atm90, put30, call30, put60, call60}
	values := append([]featureValue{}, cells...)
	values = append(values, surfaceShapeFeatures(session, chain, cells, chainRefs)...)
	values = append(values, premiumFeatures(session, chain, 30, .25, "put", put30Dims, chainRefs)...)
	values = append(values, premiumFeatures(session, chain, 30, .25, "call", call30Dims, chainRefs)...)
	values = append(values,
		oiChangeCell(session, chain, allChain, 30, .25, "put", put30Dims, chainRefs),
		oiChangeCell(session, chain, allChain, 30, .25, "call", call30Dims, chainRefs),
	)
	values = append(values, positioningFeatures(config, session, distribution, distributions)...)
	values = append(values, optionQualityFeatures(chain, cells, chainRefs)...)
	return values
}

func ivCell(key string, session time.Time, chain []storage.MarketOpsOptionsChainRecord, targetDTE int, targetDelta float64, optionType string, dimensions map[string]any, allRefs []string) featureValue {
	if len(chain) == 0 {
		return missingFeatureWithDimensions(key, dimensions, "no_option_chain_for_session", nil, nil, nil)
	}
	selected, ok := selectContract(session, chain, targetDTE, targetDelta, optionType)
	if !ok {
		return missingFeatureWithDimensions(key, dimensions, "no_eligible_surface_contract", map[string]any{"target_dte": targetDTE, "target_abs_delta": targetDelta, "option_type": optionType, "eligible_dte_min": 7, "eligible_dte_max": 180}, nil, allRefs)
	}
	details := selectedOptionDetails(selected, targetDTE, targetDelta)
	ref := optionArtifactID(selected.Record)
	return featureValue{Key: key, Dimensions: dimensions, Numeric: floatPtr(round(*selected.Record.ImpliedVolatility, 8)), Quality: storage.MarketOpsQualityUsable, QualityScore: 1, Details: details, ArtifactIDs: []string{ref}}
}

func selectContract(session time.Time, chain []storage.MarketOpsOptionsChainRecord, targetDTE int, targetDelta float64, optionType string) (selectedContract, bool) {
	best := selectedContract{Score: math.MaxFloat64}
	found := false
	for _, record := range chain {
		if optionType != "" && strings.ToLower(record.ContractType) != optionType {
			continue
		}
		dte := int(dayOnly(record.ExpirationDate).Sub(dayOnly(session)).Hours() / 24)
		if dte < 7 || dte > 180 || record.ImpliedVolatility == nil || *record.ImpliedVolatility <= 0 || record.Delta == nil {
			continue
		}
		dteTolerance := 15
		if targetDTE >= 60 {
			dteTolerance = 20
		}
		if targetDTE >= 90 {
			dteTolerance = 30
		}
		if absInt(dte-targetDTE) > dteTolerance || math.Abs(math.Abs(*record.Delta)-targetDelta) > 0.15 {
			continue
		}
		score := float64(absInt(dte-targetDTE))/float64(targetDTE) + math.Abs(math.Abs(*record.Delta)-targetDelta)
		if !found || score < best.Score || (score == best.Score && record.OptionTicker < best.Record.OptionTicker) {
			best, found = selectedContract{Record: record, DTE: dte, Score: score}, true
		}
	}
	return best, found
}

func surfaceShapeFeatures(session time.Time, chain []storage.MarketOpsOptionsChainRecord, cells []featureValue, refs []string) []featureValue {
	derivedDifference := func(key string, left, right featureValue, dimensions map[string]any, formula string) featureValue {
		artifacts := uniqueStrings(append(append([]string{}, left.ArtifactIDs...), right.ArtifactIDs...))
		if left.Numeric == nil || right.Numeric == nil || !isUsable(left.Quality) || !isUsable(right.Quality) {
			return missingFeatureWithDimensions(key, dimensions, "missing_usable_surface_inputs", map[string]any{"formula": formula}, nil, artifacts)
		}
		value := usableNumeric(key, round(*left.Numeric-*right.Numeric, 8), nil, artifacts, map[string]any{"formula": formula, "left_feature": left.Key, "right_feature": right.Key})
		value.Dimensions = dimensions
		return value
	}
	shape := []featureValue{
		derivedDifference("iv_term_slope", cells[1], cells[0], map[string]any{"near_dte": 30, "far_dte": 60}, "far_iv_minus_near_iv"),
		derivedDifference("iv_term_slope", cells[2], cells[1], map[string]any{"near_dte": 60, "far_dte": 90}, "far_iv_minus_near_iv"),
		derivedDifference("risk_reversal", cells[4], cells[3], map[string]any{"target_dte": 30, "target_delta": .25}, "call_iv_minus_put_iv"),
		derivedDifference("risk_reversal", cells[6], cells[5], map[string]any{"target_dte": 60, "target_delta": .25}, "call_iv_minus_put_iv"),
	}
	targets := []struct {
		dte        int
		delta      float64
		optionType string
	}{{30, .50, ""}, {60, .50, ""}, {90, .50, ""}, {30, .25, "put"}, {30, .25, "call"}, {60, .25, "put"}, {60, .25, "call"}}
	total, selectedCount, versionedCount := 0.0, 0, 0
	for _, target := range targets {
		selected, ok := selectContract(session, chain, target.dte, target.delta, target.optionType)
		if !ok {
			continue
		}
		score := selected.Score
		if selected.Record.SelectionScore != nil {
			score = *selected.Record.SelectionScore
		}
		total += 1 - math.Min(math.Max(score, 0), 1)
		selectedCount++
		if selected.Record.SelectionPolicyVersion != "" {
			versionedCount++
		}
	}
	confidence := missingFeature("surface_selection_confidence", "incomplete_surface_selection", map[string]any{"selected_cells": selectedCount, "required_cells": len(targets)}, nil)
	confidence.ArtifactIDs = refs
	if selectedCount == len(targets) {
		quality, score := storage.MarketOpsQualityUsable, 1.0
		if versionedCount != len(targets) {
			quality, score = storage.MarketOpsQualityUsableWithWarning, .75
		}
		value := round(total/float64(selectedCount), 8)
		confidence = featureValue{Key: "surface_selection_confidence", Numeric: &value, Quality: quality, QualityScore: score, Details: map[string]any{"selected_cells": selectedCount, "versioned_selection_cells": versionedCount, "policy_version": "marketops.options.surface_selection.v1"}, ArtifactIDs: refs}
	}
	return append(shape, confidence)
}

func optionDimensions(optionType string, targetDTE int, targetDelta float64) map[string]any {
	return map[string]any{"option_type": optionType, "target_dte": targetDTE, "target_delta": targetDelta}
}

func premiumFeatures(session time.Time, chain []storage.MarketOpsOptionsChainRecord, targetDTE int, targetDelta float64, optionType string, dimensions map[string]any, allRefs []string) []featureValue {
	keys := []string{"mid_premium", "extrinsic_premium", "premium_pct_spot", "spread_pct"}
	missing := func(reason string, details map[string]any, refs []string) []featureValue {
		out := make([]featureValue, 0, len(keys))
		for _, key := range keys {
			out = append(out, missingFeatureWithDimensions(key, dimensions, reason, details, nil, refs))
		}
		return out
	}
	selected, ok := selectContract(session, chain, targetDTE, targetDelta, optionType)
	if !ok {
		return missing("no_eligible_surface_contract", map[string]any{"target_dte": targetDTE, "target_abs_delta": targetDelta, "option_type": optionType}, allRefs)
	}
	ref := optionArtifactID(selected.Record)
	details := selectedOptionDetails(selected, targetDTE, targetDelta)
	if selected.Record.Bid == nil || selected.Record.Ask == nil {
		return missing("missing_bid_or_ask", details, []string{ref})
	}
	bid, ask := *selected.Record.Bid, *selected.Record.Ask
	if bid <= 0 || ask <= 0 || ask < bid {
		out := make([]featureValue, 0, len(keys))
		details["bid"], details["ask"] = bid, ask
		for _, key := range keys {
			out = append(out, featureValue{Key: key, Dimensions: dimensions, Quality: storage.MarketOpsQualityInvalid, Details: details, ArtifactIDs: []string{ref}})
		}
		return out
	}
	mid := (bid + ask) / 2
	if selected.Record.UnderlyingClose == nil || *selected.Record.UnderlyingClose <= 0 {
		return missing("missing_underlying_close", details, []string{ref})
	}
	spot := *selected.Record.UnderlyingClose
	intrinsic := math.Max(spot-selected.Record.StrikePrice, 0)
	if optionType == "put" {
		intrinsic = math.Max(selected.Record.StrikePrice-spot, 0)
	}
	extrinsic := math.Max(mid-intrinsic, 0)
	quality, qualityScore := storage.MarketOpsQualityUsable, 1.0
	if selected.Record.QuoteTimestamp == nil {
		quality, qualityScore = storage.MarketOpsQualityUsableWithWarning, .75
		details["quality_warning"] = "missing_quote_timestamp"
	} else {
		details["quote_timestamp"] = selected.Record.QuoteTimestamp.UTC().Format(time.RFC3339Nano)
		if !dayOnly(*selected.Record.QuoteTimestamp).Equal(dayOnly(session)) {
			quality, qualityScore = storage.MarketOpsQualityStale, 0
			details["reason"] = "quote_not_from_session"
		}
	}
	details["bid"], details["ask"], details["midpoint"], details["intrinsic"] = bid, ask, mid, intrinsic
	value := func(key string, numeric float64) featureValue {
		return featureValue{Key: key, Dimensions: dimensions, Numeric: floatPtr(round(numeric, 8)), Quality: quality, QualityScore: qualityScore, Details: details, ArtifactIDs: []string{ref}}
	}
	return []featureValue{
		value("mid_premium", mid),
		value("extrinsic_premium", extrinsic),
		value("premium_pct_spot", mid/spot*100),
		value("spread_pct", (ask-bid)/mid*100),
	}
}

func oiChangeCell(session time.Time, chain []storage.MarketOpsOptionsChainRecord, allChain map[string][]storage.MarketOpsOptionsChainRecord, targetDTE int, targetDelta float64, optionType string, dimensions map[string]any, allRefs []string) featureValue {
	current, ok := selectContract(session, chain, targetDTE, targetDelta, optionType)
	if !ok {
		return missingFeatureWithDimensions("oi_change_1d", dimensions, "no_eligible_surface_contract", nil, nil, allRefs)
	}
	currentRef := optionArtifactID(current.Record)
	if current.Record.OpenInterest == nil {
		return missingFeatureWithDimensions("oi_change_1d", dimensions, "missing_current_open_interest", selectedOptionDetails(current, targetDTE, targetDelta), nil, []string{currentRef})
	}
	priorSession, priorChain, ok := previousOptionChain(session, allChain)
	if !ok {
		return missingFeatureWithDimensions("oi_change_1d", dimensions, "no_prior_option_session", nil, nil, []string{currentRef})
	}
	prior, ok := selectContract(priorSession, priorChain, targetDTE, targetDelta, optionType)
	if !ok || prior.Record.OpenInterest == nil {
		return missingFeatureWithDimensions("oi_change_1d", dimensions, "no_prior_usable_surface_open_interest", map[string]any{"prior_session": dateKey(priorSession)}, nil, []string{currentRef})
	}
	priorRef := optionArtifactID(prior.Record)
	change := float64(*current.Record.OpenInterest - *prior.Record.OpenInterest)
	details := selectedOptionDetails(current, targetDTE, targetDelta)
	details["current_open_interest"] = *current.Record.OpenInterest
	details["prior_open_interest"] = *prior.Record.OpenInterest
	details["prior_session"] = dateKey(priorSession)
	details["prior_option_ticker"] = prior.Record.OptionTicker
	value := usableNumeric("oi_change_1d", round(change, 8), nil, []string{currentRef, priorRef}, details)
	value.Dimensions = dimensions
	return value
}

func previousOptionChain(session time.Time, allChain map[string][]storage.MarketOpsOptionsChainRecord) (time.Time, []storage.MarketOpsOptionsChainRecord, bool) {
	var latest time.Time
	var records []storage.MarketOpsOptionsChainRecord
	for key, candidate := range allChain {
		date, err := time.Parse("2006-01-02", key)
		if err != nil || !date.Before(dayOnly(session)) || !date.After(latest) {
			continue
		}
		latest, records = date, candidate
	}
	return latest, records, !latest.IsZero()
}

func selectedOptionDetails(selected selectedContract, targetDTE int, targetDelta float64) map[string]any {
	return map[string]any{
		"selected_option_ticker":   selected.Record.OptionTicker,
		"actual_dte":               selected.DTE,
		"actual_delta":             *selected.Record.Delta,
		"target_dte":               targetDTE,
		"target_abs_delta":         targetDelta,
		"selection_score":          round(selected.Score, 6),
		"selection_cell":           selected.Record.SelectionCell,
		"selection_policy_version": selected.Record.SelectionPolicyVersion,
		"provider_request_id":      selected.Record.ProviderRequestID,
	}
}

func positioningFeatures(config BuildConfig, session time.Time, distribution *storage.MarketOpsOptionsDistributionRecord, distributions map[string]*storage.MarketOpsOptionsDistributionRecord) []featureValue {
	if distribution == nil {
		return []featureValue{
			missingFeature("put_call_oi_ratio", "no_options_distribution_for_session", nil, nil),
			missingFeature("put_call_oi_change_1d", "no_options_distribution_for_session", nil, nil),
			missingFeature("put_call_volume_ratio", "no_options_distribution_for_session", nil, nil),
			missingFeature("put_call_volume_ratio_10d_deviation_pct", "no_options_distribution_for_session", nil, nil),
		}
	}
	ref := distributionArtifactID(config, *distribution)
	metrics := map[string]any{}
	_ = json.Unmarshal(distribution.MetricsJSON, &metrics)
	ratioQuality := stringValue(metrics["call_put_oi_ratio_quality"])
	oiRatio := featureValue{Key: "put_call_oi_ratio", Quality: storage.MarketOpsQualityInvalid, Details: map[string]any{"reason": "unusable_open_interest", "source_ratio_quality": ratioQuality, "open_interest_quality": stringValue(metrics["open_interest_quality"])}, ArtifactIDs: []string{ref}}
	if ratioQuality == "usable" && distribution.TotalCallOpenInterest > 0 {
		value := float64(distribution.TotalPutOpenInterest) / float64(distribution.TotalCallOpenInterest)
		oiRatio.Numeric, oiRatio.Quality, oiRatio.QualityScore = floatPtr(round(value, 8)), storage.MarketOpsQualityUsable, 1
		oiRatio.Details = map[string]any{"source_ratio_quality": ratioQuality, "put_open_interest": distribution.TotalPutOpenInterest, "call_open_interest": distribution.TotalCallOpenInterest}
	}
	oiChange := missingFeature("put_call_oi_change_1d", "no_prior_usable_oi_ratio", nil, nil)
	oiChange.ArtifactIDs = []string{ref}
	if oiRatio.Numeric != nil {
		if previousValue, previousRef, ok := previousUsableOIRatio(config, session, distributions); ok {
			value := *oiRatio.Numeric - previousValue
			oiChange = usableNumeric("put_call_oi_change_1d", round(value, 8), nil, []string{ref, previousRef}, map[string]any{"baseline_source_artifact_id": previousRef})
		}
	}
	volumeRatio := featureValue{Key: "put_call_volume_ratio", Quality: storage.MarketOpsQualityInvalid, Details: map[string]any{"reason": "zero_call_volume"}, ArtifactIDs: []string{ref}}
	if distribution.TotalCallVolume > 0 {
		value := float64(distribution.TotalPutVolume) / float64(distribution.TotalCallVolume)
		volumeRatio.Numeric, volumeRatio.Quality, volumeRatio.QualityScore = floatPtr(round(value, 8)), storage.MarketOpsQualityUsableWithWarning, .75
		volumeRatio.Details = map[string]any{"quality_warning": "distribution_volume_is_not_open_interest", "put_volume": distribution.TotalPutVolume, "call_volume": distribution.TotalCallVolume}
	}
	deviation := putCallVolumeDeviationFeature(config, session, distributions, volumeRatio)
	return []featureValue{oiRatio, oiChange, volumeRatio, deviation}
}

func putCallVolumeDeviationFeature(config BuildConfig, session time.Time, distributions map[string]*storage.MarketOpsOptionsDistributionRecord, current featureValue) featureValue {
	const key = "put_call_volume_ratio_10d_deviation_pct"
	if current.Numeric == nil || !isUsable(current.Quality) { return missingFeature(key, "current_put_call_volume_ratio_unusable", nil, current.ArtifactIDs) }
	values, refs := []float64{}, append([]string{}, current.ArtifactIDs...)
	dates := make([]time.Time, 0, len(distributions))
	for date := range distributions { if parsed, err := time.Parse("2006-01-02", date); err == nil && parsed.Before(dayOnly(session)) { dates = append(dates, parsed) } }
	sort.Slice(dates, func(i, j int) bool { return dates[i].After(dates[j]) })
	for _, date := range dates { record := distributions[dateKey(date)]; if record == nil || record.TotalCallVolume <= 0 { continue }; values = append(values, float64(record.TotalPutVolume)/float64(record.TotalCallVolume)); refs = append(refs, distributionArtifactID(config, *record)); if len(values) == 10 { break } }
	if len(values) < 10 { return missingFeature(key, "insufficient_prior_usable_put_call_volume_sessions", map[string]any{"available_sessions": len(values), "required_sessions": 10}, refs) }
	mean := 0.0; for _, value := range values { mean += value }; mean /= float64(len(values))
	if mean <= 0 { return invalidFeature(key, "non_positive_put_call_volume_baseline", nil, refs) }
	return usableNumeric(key, round((*current.Numeric/mean-1)*100, 6), nil, refs, map[string]any{"lookback_sessions": 10, "trailing_mean": round(mean, 8), "canonical_ratio": "put_call_volume"})
}

func optionQualityFeatures(chain []storage.MarketOpsOptionsChainRecord, cells []featureValue, refs []string) []featureValue {
	if len(chain) == 0 {
		return []featureValue{
			missingFeature("usable_contract_ratio", "no_option_chain_for_session", nil, nil), missingFeature("missing_iv_ratio", "no_option_chain_for_session", nil, nil),
			missingFeature("missing_greeks_ratio", "no_option_chain_for_session", nil, nil), missingFeature("surface_coverage_ratio", "no_option_chain_for_session", nil, nil),
			{Key: "oi_quality_state", Quality: storage.MarketOpsQualityMissing, Details: map[string]any{"reason": "no_option_chain_for_session"}},
		}
	}
	usable, missingIV, missingDelta := 0, 0, 0
	for _, record := range chain {
		dte := int(dayOnly(record.ExpirationDate).Sub(dayOnly(record.TradeDate)).Hours() / 24)
		if record.ImpliedVolatility == nil || *record.ImpliedVolatility <= 0 {
			missingIV++
		}
		if record.Delta == nil {
			missingDelta++
		}
		if dte >= 7 && dte <= 180 && record.ImpliedVolatility != nil && *record.ImpliedVolatility > 0 && record.Delta != nil {
			usable++
		}
	}
	covered := 0
	for _, cell := range cells {
		if cell.Numeric != nil && isUsable(cell.Quality) {
			covered++
		}
	}
	openInterestQuality := "missing"
	positiveOI := 0
	for _, record := range chain {
		if record.OpenInterest != nil {
			if *record.OpenInterest > 0 {
				positiveOI++
			}
			if openInterestQuality == "missing" {
				openInterestQuality = "all_zero"
			}
		}
	}
	if positiveOI == len(chain) {
		openInterestQuality = "usable"
	} else if positiveOI > 0 {
		openInterestQuality = "partial_zero"
	}
	qualityText := openInterestQuality
	return []featureValue{
		usableNumeric("usable_contract_ratio", round(float64(usable)/float64(len(chain)), 8), nil, refs, map[string]any{"contract_count": len(chain), "usable_contract_count": usable}),
		usableNumeric("missing_iv_ratio", round(float64(missingIV)/float64(len(chain)), 8), nil, refs, map[string]any{"contract_count": len(chain), "missing_count": missingIV}),
		usableNumeric("missing_greeks_ratio", round(float64(missingDelta)/float64(len(chain)), 8), nil, refs, map[string]any{"contract_count": len(chain), "missing_count": missingDelta, "greek": "delta"}),
		usableNumeric("surface_coverage_ratio", round(float64(covered)/7.0, 8), nil, refs, map[string]any{"required_cells": 7, "covered_cells": covered}),
		{Key: "oi_quality_state", Text: &qualityText, Quality: storage.MarketOpsQualityUsable, QualityScore: 1, Details: map[string]any{"positive_open_interest_contract_count": positiveOI, "contract_count": len(chain)}, ArtifactIDs: refs},
	}
}

func observationRecord(config BuildConfig, session time.Time, value featureValue) (storage.MarketOpsFeatureObservationRecord, error) {
	if value.Dimensions == nil {
		value.Dimensions = map[string]any{}
	}
	dimensions, err := json.Marshal(value.Dimensions)
	if err != nil {
		return storage.MarketOpsFeatureObservationRecord{}, err
	}
	canonical, err := CanonicalDimensions(dimensions)
	if err != nil {
		return storage.MarketOpsFeatureObservationRecord{}, err
	}
	identity, err := NewIdentity(IdentityFeatureObservation, config.TenantID, config.AssetID, dateKey(session), value.Key, FeatureVersion, canonical)
	if err != nil {
		return storage.MarketOpsFeatureObservationRecord{}, err
	}
	details, _ := json.Marshal(value.Details)
	var score *float64
	if value.QualityScore > 0 || isUsable(value.Quality) {
		score = floatPtr(value.QualityScore)
	}
	return storage.MarketOpsFeatureObservationRecord{
		FeatureObservationID: identity.ID, TenantID: config.TenantID, AppID: "marketops", AssetID: config.AssetID, Symbol: config.Symbol,
		SessionDate: dayOnly(session), AsOfTime: sessionEnd(session), FeatureKey: value.Key, FeatureVersion: FeatureVersion,
		DimensionsJSON: []byte(canonical), NumericValue: value.Numeric, TextValue: value.Text, QualityState: value.Quality, QualityScore: score,
		QualityDetailsJSON: jsonOrObject(details), SourceEventIDs: uniqueStrings(value.EventIDs), SourceArtifactIDs: uniqueStrings(value.ArtifactIDs),
		CalculationRunID: config.RunID, DeterministicKey: identity.DeterministicKey,
	}, nil
}

func marketStateRecord(config BuildConfig, session time.Time, observations []storage.MarketOpsFeatureObservationRecord) (storage.MarketOpsMarketStateRecord, error) {
	identity, err := NewIdentity(IdentityMarketState, config.TenantID, config.AssetID, dateKey(session), StateSchemaVersion)
	if err != nil {
		return storage.MarketOpsMarketStateRecord{}, err
	}
	ids := make([]string, 0, len(observations))
	features := make([]map[string]any, 0, len(observations))
	qualityCounts := map[string]int{}
	usable, requiredUsable := 0, 0
	for index, observation := range observations {
		ids = append(ids, observation.FeatureObservationID)
		qualityCounts[observation.QualityState]++
		if isUsable(observation.QualityState) {
			usable++
			if index < requiredFeatureSlots {
				requiredUsable++
			}
		}
		feature := map[string]any{"feature_observation_id": observation.FeatureObservationID, "feature_key": observation.FeatureKey, "feature_version": observation.FeatureVersion, "quality_state": observation.QualityState, "dimensions": json.RawMessage(observation.DimensionsJSON)}
		if observation.NumericValue != nil {
			feature["numeric_value"] = *observation.NumericValue
		}
		if observation.TextValue != nil {
			feature["text_value"] = *observation.TextValue
		}
		features = append(features, feature)
	}
	completeness := float64(requiredUsable) / float64(requiredFeatureSlots)
	quality := storage.MarketOpsQualityMissing
	if requiredUsable > 0 {
		quality = storage.MarketOpsQualityPartial
	}
	if completeness >= .6 {
		quality = storage.MarketOpsQualityUsableWithWarning
	}
	if completeness >= .8 {
		quality = storage.MarketOpsQualityUsable
	}
	payload, _ := json.Marshal(map[string]any{"schema_version": StateSchemaVersion, "subject": map[string]any{"asset_id": config.AssetID, "symbol": config.Symbol}, "session_date": dateKey(session), "features": features})
	summary, _ := json.Marshal(map[string]any{"required_feature_slots": requiredFeatureSlots, "usable_required_feature_slots": requiredUsable, "blocked_required_feature_slots": requiredFeatureSlots - requiredUsable, "total_feature_slots": len(observations), "usable_total_feature_slots": usable, "quality_counts": qualityCounts})
	return storage.MarketOpsMarketStateRecord{
		MarketStateID: identity.ID, TenantID: config.TenantID, AppID: "marketops", AssetID: config.AssetID, Symbol: config.Symbol,
		SessionDate: dayOnly(session), AsOfTime: sessionEnd(session), StateSchemaVersion: StateSchemaVersion, StatePayloadJSON: payload,
		FeatureObservationIDs: ids, FeatureCount: len(observations), RequiredFeatureCount: requiredFeatureSlots, CompletenessRatio: completeness,
		QualityState: quality, QualityScore: floatPtr(completeness), QualitySummaryJSON: summary, EligibleHypotheses: []string{},
		BuildRunID: config.RunID, DeterministicKey: identity.DeterministicKey,
	}, nil
}

func buildTransitions(config BuildConfig, states []storage.MarketOpsMarketStateRecord, observations map[string][]storage.MarketOpsFeatureObservationRecord) ([]storage.MarketOpsStateTransitionRecord, error) {
	out := []storage.MarketOpsStateTransitionRecord{}
	history := map[string][]float64{}
	directions := map[string][]string{}
	lastTransitionState := map[string]string{}
	for index := 1; index < len(states); index++ {
		current, baseline := states[index], states[index-1]
		baselineByKey := observationMap(observations[dateKey(baseline.SessionDate)])
		for _, currentObservation := range observations[dateKey(current.SessionDate)] {
			if currentObservation.NumericValue == nil || !isUsable(currentObservation.QualityState) {
				continue
			}
			canonical, _ := CanonicalDimensions(currentObservation.DimensionsJSON)
			prior, ok := baselineByKey[observationKey(currentObservation.FeatureKey, canonical)]
			if !ok || prior.NumericValue == nil || !isUsable(prior.QualityState) {
				continue
			}
			identity, err := NewIdentity(IdentityStateTransition, config.TenantID, config.AssetID, dateKey(current.SessionDate), currentObservation.FeatureKey, FeatureVersion, canonical, "absolute_difference", dateKey(baseline.SessionDate))
			if err != nil {
				return nil, err
			}
			change := round(*currentObservation.NumericValue-*prior.NumericValue, 8)
			direction := directionFor(change)
			historyKey := observationKey(currentObservation.FeatureKey, canonical)
			priorChanges := history[historyKey]
			zscore := trailingZScore(change, priorChanges)
			percentile := trailingPercentile(change, priorChanges)
			priorDirections := directions[historyKey]
			if lastTransitionState[historyKey] != baseline.MarketStateID {
				priorDirections = nil
			}
			persistence := trailingPersistence(direction, priorDirections)
			lookback := 1
			payload, _ := json.Marshal(map[string]any{"current_feature_observation_id": currentObservation.FeatureObservationID, "baseline_feature_observation_id": prior.FeatureObservationID, "baseline_session_date": dateKey(baseline.SessionDate), "operator": "absolute_difference", "trailing_sample_count": len(priorChanges), "point_in_time_statistics": true})
			out = append(out, storage.MarketOpsStateTransitionRecord{
				TransitionID: identity.ID, TenantID: config.TenantID, AppID: "marketops", AssetID: config.AssetID, Symbol: config.Symbol,
				SessionDate: current.SessionDate, AsOfTime: current.AsOfTime, CurrentStateID: current.MarketStateID, BaselineStateID: baseline.MarketStateID,
				FeatureKey: currentObservation.FeatureKey, FeatureVersion: FeatureVersion, DimensionsJSON: currentObservation.DimensionsJSON,
				TransitionType: "absolute_difference", LookbackSessions: &lookback, CurrentValue: currentObservation.NumericValue, BaselineValue: prior.NumericValue,
				TransitionValue: &change, ZScore: zscore, Percentile: percentile, PersistenceSessions: &persistence,
				Direction: direction, QualityState: storage.MarketOpsQualityUsable, TransitionPayloadJSON: payload,
				CalculationRunID: config.RunID, DeterministicKey: identity.DeterministicKey,
			})
			history[historyKey] = append(priorChanges, change)
			directions[historyKey] = append(priorDirections, direction)
			lastTransitionState[historyKey] = current.MarketStateID
		}
	}
	return out, nil
}

func trailingZScore(current float64, prior []float64) *float64 {
	if len(prior) < 20 {
		return nil
	}
	window := prior
	if len(window) > 60 {
		window = window[len(window)-60:]
	}
	mean := 0.0
	for _, value := range window {
		mean += value
	}
	mean /= float64(len(window))
	variance := 0.0
	for _, value := range window {
		delta := value - mean
		variance += delta * delta
	}
	variance /= float64(len(window))
	if variance <= 0 {
		return nil
	}
	value := round((current-mean)/math.Sqrt(variance), 8)
	return &value
}

func trailingPercentile(current float64, prior []float64) *float64 {
	if len(prior) < 20 {
		return nil
	}
	window := prior
	if len(window) > 60 {
		window = window[len(window)-60:]
	}
	lessOrEqual := 0
	for _, value := range window {
		if value <= current {
			lessOrEqual++
		}
	}
	value := round(float64(lessOrEqual)/float64(len(window)), 8)
	return &value
}

func trailingPersistence(current string, prior []string) int {
	if current == "flat" || current == "" {
		return 1
	}
	count := 1
	for index := len(prior) - 1; index >= 0 && prior[index] == current; index-- {
		count++
	}
	return count
}

func buildEvidence(config BuildConfig, states []storage.MarketOpsMarketStateRecord, observations map[string][]storage.MarketOpsFeatureObservationRecord, transitions []storage.MarketOpsStateTransitionRecord) ([]storage.MarketOpsEvidenceRecord, error) {
	transitionByState := map[string][]storage.MarketOpsStateTransitionRecord{}
	for _, transition := range transitions {
		transitionByState[transition.CurrentStateID] = append(transitionByState[transition.CurrentStateID], transition)
	}
	out := []storage.MarketOpsEvidenceRecord{}
	for _, marketState := range states {
		byKey := observationMap(observations[dateKey(marketState.SessionDate)])
		if observation, ok := byKey[observationKey("return_1d", "{}")]; ok && observation.NumericValue != nil && isUsable(observation.QualityState) {
			value := *observation.NumericValue
			statement := fmt.Sprintf("%s closed %s %.2f%% over one eligible session.", config.Symbol, directionVerb(value), math.Abs(value))
			record, err := evidenceRecord(config, marketState, "underlying_return_observed", "underlying_momentum", directionFor(value), value, statement, []string{observation.FeatureObservationID}, nil, map[string]any{"return_1d_pct": value})
			if err != nil {
				return nil, err
			}
			out = append(out, record)
		}
		if observation, ok := byKey[observationKey("put_call_oi_ratio", "{}")]; ok && observation.NumericValue != nil && isUsable(observation.QualityState) {
			value := *observation.NumericValue
			statement := fmt.Sprintf("%s usable put/call open-interest ratio was %.3f.", config.Symbol, value)
			record, err := evidenceRecord(config, marketState, "put_call_oi_ratio_observed", "option_positioning", ratioDirection(value), value, statement, []string{observation.FeatureObservationID}, nil, map[string]any{"put_call_oi_ratio": value})
			if err != nil {
				return nil, err
			}
			out = append(out, record)
		}
		for _, transition := range transitionByState[marketState.MarketStateID] {
			if !strings.HasPrefix(transition.FeatureKey, "atm_iv_") || transition.TransitionValue == nil || math.Abs(*transition.TransitionValue) < .01 {
				continue
			}
			value := *transition.TransitionValue
			statement := fmt.Sprintf("%s %s changed %s by %.4f implied-volatility points over one eligible state session.", config.Symbol, transition.FeatureKey, directionVerb(value), math.Abs(value))
			record, err := evidenceRecord(config, marketState, "atm_iv_changed", "implied_volatility", directionFor(value), value, statement, nil, []string{transition.TransitionID}, map[string]any{"feature_key": transition.FeatureKey, "absolute_change": value})
			if err != nil {
				return nil, err
			}
			out = append(out, record)
		}
	}
	return out, nil
}

func evidenceRecord(config BuildConfig, marketState storage.MarketOpsMarketStateRecord, evidenceType, domain, direction string, magnitude float64, statement string, featureIDs, transitionIDs []string, payload map[string]any) (storage.MarketOpsEvidenceRecord, error) {
	featureComponent := strings.Join(uniqueStrings(featureIDs), ",")
	if featureComponent == "" {
		featureComponent = "no_features"
	}
	transitionComponent := strings.Join(uniqueStrings(transitionIDs), ",")
	if transitionComponent == "" {
		transitionComponent = "no_transitions"
	}
	identity, err := NewIdentity(IdentityEvidence, config.TenantID, config.AssetID, dateKey(marketState.SessionDate), evidenceType, EvidenceVersion, featureComponent, transitionComponent)
	if err != nil {
		return storage.MarketOpsEvidenceRecord{}, err
	}
	payload["market_state_id"] = marketState.MarketStateID
	payloadJSON, _ := json.Marshal(payload)
	quality := marketState.CompletenessRatio
	return storage.MarketOpsEvidenceRecord{EvidenceID: identity.ID, TenantID: config.TenantID, AppID: "marketops", AssetID: config.AssetID, Symbol: config.Symbol, SessionDate: marketState.SessionDate, AsOfTime: marketState.AsOfTime, EvidenceType: evidenceType, EvidenceVersion: EvidenceVersion, Domain: domain, Direction: direction, Magnitude: floatPtr(magnitude), QualityScore: &quality, Statement: statement, EvidencePayloadJSON: payloadJSON, SourceFeatureIDs: uniqueStrings(featureIDs), SourceTransitionIDs: uniqueStrings(transitionIDs), DeterministicKey: identity.DeterministicKey}, nil
}

func previousUsableOIRatio(config BuildConfig, session time.Time, distributions map[string]*storage.MarketOpsOptionsDistributionRecord) (float64, string, bool) {
	keys := make([]string, 0, len(distributions))
	for date := range distributions {
		if date < dateKey(session) {
			keys = append(keys, date)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	for _, date := range keys {
		record := distributions[date]
		metrics := map[string]any{}
		_ = json.Unmarshal(record.MetricsJSON, &metrics)
		if stringValue(metrics["call_put_oi_ratio_quality"]) == "usable" && record.TotalCallOpenInterest > 0 {
			return round(float64(record.TotalPutOpenInterest)/float64(record.TotalCallOpenInterest), 8), distributionArtifactID(config, *record), true
		}
	}
	return 0, "", false
}

func observationMap(records []storage.MarketOpsFeatureObservationRecord) map[string]storage.MarketOpsFeatureObservationRecord {
	out := map[string]storage.MarketOpsFeatureObservationRecord{}
	for _, record := range records {
		dimensions, _ := CanonicalDimensions(record.DimensionsJSON)
		out[observationKey(record.FeatureKey, dimensions)] = record
	}
	return out
}

func observationKey(key, dimensions string) string { return key + "|" + dimensions }

func missingFeature(key, reason string, details map[string]any, eventIDs []string) featureValue {
	return missingFeatureWithDimensions(key, nil, reason, details, eventIDs, nil)
}

func missingFeatureWithDimensions(key string, dimensions map[string]any, reason string, details map[string]any, eventIDs, artifacts []string) featureValue {
	if details == nil {
		details = map[string]any{}
	}
	details["reason"] = reason
	return featureValue{Key: key, Dimensions: dimensions, Quality: storage.MarketOpsQualityMissing, Details: details, EventIDs: eventIDs, ArtifactIDs: artifacts}
}

func invalidFeature(key, reason string, eventIDs, artifacts []string) featureValue {
	return featureValue{Key: key, Quality: storage.MarketOpsQualityInvalid, Details: map[string]any{"reason": reason}, EventIDs: eventIDs, ArtifactIDs: artifacts}
}

func usableNumeric(key string, value float64, eventIDs, artifacts []string, details map[string]any) featureValue {
	return featureValue{Key: key, Numeric: floatPtr(value), Quality: storage.MarketOpsQualityUsable, QualityScore: 1, Details: details, EventIDs: eventIDs, ArtifactIDs: artifacts}
}

func eventIDs(history []equityPoint, start, end int) []string {
	if start < 0 {
		start = 0
	}
	if end >= len(history) {
		end = len(history) - 1
	}
	out := []string{}
	for i := start; i <= end; i++ {
		if history[i].EventID != "" {
			out = append(out, history[i].EventID)
		}
	}
	return uniqueStrings(out)
}

func chainArtifactIDs(config BuildConfig, session time.Time, records []storage.MarketOpsOptionsChainRecord) []string {
	out := make([]string, 0, len(records))
	for _, record := range records {
		out = append(out, optionArtifactID(record))
	}
	return uniqueStrings(out)
}

func optionArtifactID(record storage.MarketOpsOptionsChainRecord) string {
	return strings.Join([]string{"marketops_options_chain_daily", record.TenantID, strings.ToUpper(record.Symbol), dateKey(record.TradeDate), record.OptionTicker}, ":")
}

func distributionArtifactID(config BuildConfig, record storage.MarketOpsOptionsDistributionRecord) string {
	return strings.Join([]string{"marketops_options_distribution_daily", config.TenantID, config.Symbol, dateKey(record.TradeDate), record.WindowName}, ":")
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func isUsable(quality string) bool {
	return quality == storage.MarketOpsQualityUsable || quality == storage.MarketOpsQualityUsableWithWarning
}
func floatPtr(value float64) *float64 { return &value }
func dayOnly(value time.Time) time.Time {
	utc := value.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}
func sessionEnd(value time.Time) time.Time { return dayOnly(value).Add(24*time.Hour - time.Nanosecond) }
func dateKey(value time.Time) string       { return dayOnly(value).Format("2006-01-02") }
func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
func round(value float64, places int) float64 {
	factor := math.Pow10(places)
	return math.Round(value*factor) / factor
}
func jsonOrObject(value []byte) []byte {
	if len(value) == 0 || string(value) == "null" {
		return []byte(`{}`)
	}
	return value
}
func directionFor(value float64) string {
	if value > 0 {
		return "up"
	}
	if value < 0 {
		return "down"
	}
	return "flat"
}
func directionVerb(value float64) string {
	if value > 0 {
		return "up"
	}
	if value < 0 {
		return "down"
	}
	return "unchanged"
}
func ratioDirection(value float64) string {
	if value > 1 {
		return "put_heavy"
	}
	if value < 1 {
		return "call_heavy"
	}
	return "balanced"
}

func stringValue(value any) string {
	if typed, ok := value.(string); ok {
		return strings.TrimSpace(typed)
	}
	return ""
}

func numberValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

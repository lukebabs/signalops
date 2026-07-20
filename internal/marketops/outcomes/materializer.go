package outcomes

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	marketopsstate "github.com/lukebabs/signalops/internal/marketops/state"
	"github.com/lukebabs/signalops/internal/storage"
)

const (
	CalculationVersion = "marketops.forward_outcome.v1"
	DefaultThreshold   = 0.02
)

var DefaultHorizons = []int{1, 5, 10, 20}

type BuildConfig struct {
	TenantID      string
	Symbol        string
	RunID         string
	AsOf          time.Time
	Threshold     float64
	BoundedCohort bool
	Horizons      []int
}

type BuildInput struct {
	Evaluations   []storage.MarketOpsHypothesisEvaluationRecord
	Opportunities []storage.MarketOpsOpportunityRecord
	EquityEvents  []storage.NormalizedEventLedgerRecord
}

type BuildResult struct {
	Outcomes       []storage.MarketOpsSignalOutcomeRecord
	Sources        int
	Matured        int
	Pending        int
	MissingPrice   int
	SkippedReasons map[string]int
}

type source struct {
	sourceType        string
	sourceID          string
	tenantID          string
	appID             string
	hypothesisKey     string
	hypothesisVersion string
	assetID           string
	symbol            string
	direction         string
	origin            time.Time
}

type pricePoint struct {
	session time.Time
	eventID string
	open    float64
	high    float64
	low     float64
	close   float64
	valid   bool
}

func Build(config BuildConfig, input BuildInput) (BuildResult, error) {
	config = defaults(config)
	if err := validateConfig(config); err != nil {
		return BuildResult{}, err
	}
	result := BuildResult{SkippedReasons: map[string]int{}}
	sources := outcomeSources(config, input, result.SkippedReasons)
	result.Sources = len(sources)
	prices, err := equityHistory(config, input.EquityEvents)
	if err != nil {
		return result, err
	}
	for _, item := range sources {
		for _, horizon := range config.Horizons {
			record, err := buildOutcome(config, item, prices, horizon)
			if err != nil {
				return result, err
			}
			switch record.OutcomeStatus {
			case storage.MarketOpsOutcomeMatured:
				result.Matured++
			case storage.MarketOpsOutcomePending:
				result.Pending++
			case storage.MarketOpsOutcomeMissingPrice:
				result.MissingPrice++
			}
			result.Outcomes = append(result.Outcomes, record)
		}
	}
	sort.Slice(result.Outcomes, func(i, j int) bool {
		left, right := result.Outcomes[i], result.Outcomes[j]
		if left.OriginSessionDate.Equal(right.OriginSessionDate) {
			if left.SourceID == right.SourceID {
				return left.HorizonSessions < right.HorizonSessions
			}
			return left.SourceID < right.SourceID
		}
		return left.OriginSessionDate.Before(right.OriginSessionDate)
	})
	return result, nil
}

func defaults(config BuildConfig) BuildConfig {
	config.TenantID = strings.TrimSpace(config.TenantID)
	config.Symbol = strings.ToUpper(strings.TrimSpace(config.Symbol))
	config.RunID = strings.TrimSpace(config.RunID)
	config.AsOf = day(config.AsOf)
	if config.Threshold == 0 {
		config.Threshold = DefaultThreshold
	}
	if len(config.Horizons) == 0 {
		config.Horizons = append([]int(nil), DefaultHorizons...)
	}
	return config
}

func validateConfig(config BuildConfig) error {
	if config.TenantID == "" || config.Symbol == "" || config.RunID == "" || config.AsOf.IsZero() {
		return errors.New("G140 tenant_id, symbol, run_id, and as_of are required")
	}
	if config.Symbol != "AAPL" && !config.BoundedCohort {
		return errors.New("G140 is intentionally bounded to AAPL")
	}
	if config.Threshold <= 0 || config.Threshold >= 1 {
		return errors.New("G140 threshold must be between 0 and 1")
	}
	seen := map[int]bool{}
	for _, horizon := range config.Horizons {
		if horizon != 1 && horizon != 5 && horizon != 10 && horizon != 20 {
			return fmt.Errorf("unsupported G140 horizon %d", horizon)
		}
		if seen[horizon] {
			return fmt.Errorf("duplicate G140 horizon %d", horizon)
		}
		seen[horizon] = true
	}
	return nil
}

func outcomeSources(config BuildConfig, input BuildInput, skipped map[string]int) []source {
	out := []source{}
	seen := map[string]bool{}
	for _, evaluation := range input.Evaluations {
		if evaluation.TenantID != config.TenantID || strings.ToUpper(evaluation.Symbol) != config.Symbol {
			continue
		}
		if !evaluation.Eligible || !evaluation.Triggered || evaluation.Invalidated {
			skipped["evaluation_not_triggered"]++
			continue
		}
		direction := evaluationDirection(evaluation)
		if !validDirection(direction) {
			skipped["evaluation_direction_unresolved"]++
			continue
		}
		if day(evaluation.SessionDate).After(config.AsOf) {
			skipped["source_after_as_of"]++
			continue
		}
		item := source{storage.MarketOpsOutcomeSourceHypothesisEvaluation, evaluation.EvaluationID, evaluation.TenantID, appID(evaluation.AppID), evaluation.HypothesisKey, evaluation.HypothesisVersion, evaluation.AssetID, strings.ToUpper(evaluation.Symbol), direction, day(evaluation.SessionDate)}
		if !seen[item.sourceType+"\x00"+item.sourceID] {
			seen[item.sourceType+"\x00"+item.sourceID] = true
			out = append(out, item)
		}
	}
	for _, opportunity := range input.Opportunities {
		if opportunity.TenantID != config.TenantID || strings.ToUpper(opportunity.Symbol) != config.Symbol {
			continue
		}
		if !validDirection(opportunity.Direction) {
			skipped["opportunity_direction_unresolved"]++
			continue
		}
		if day(opportunity.LastEvaluatedDate).After(config.AsOf) {
			skipped["source_after_as_of"]++
			continue
		}
		item := source{storage.MarketOpsOutcomeSourceOpportunity, opportunity.OpportunityID, opportunity.TenantID, appID(opportunity.AppID), "", "", opportunity.AssetID, strings.ToUpper(opportunity.Symbol), opportunity.Direction, day(opportunity.LastEvaluatedDate)}
		if !seen[item.sourceType+"\x00"+item.sourceID] {
			seen[item.sourceType+"\x00"+item.sourceID] = true
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].origin.Equal(out[j].origin) {
			if out[i].sourceType == out[j].sourceType {
				return out[i].sourceID < out[j].sourceID
			}
			return out[i].sourceType < out[j].sourceType
		}
		return out[i].origin.Before(out[j].origin)
	})
	return out
}

func evaluationDirection(record storage.MarketOpsHypothesisEvaluationRecord) string {
	var payload struct {
		ResolvedDirection string `json:"resolved_direction"`
	}
	_ = json.Unmarshal(record.EvaluationPayloadJSON, &payload)
	return strings.TrimSpace(payload.ResolvedDirection)
}

func equityHistory(config BuildConfig, events []storage.NormalizedEventLedgerRecord) ([]pricePoint, error) {
	latest := map[string]storage.NormalizedEventLedgerRecord{}
	for _, event := range events {
		if event.TenantID != config.TenantID || event.Dataset != "equity_eod_prices" || day(event.ObservationTime).After(config.AsOf) {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal(event.NormalizedPayload, &payload); err != nil {
			return nil, fmt.Errorf("decode equity event %s: %w", event.EventID, err)
		}
		symbol := strings.ToUpper(stringValue(payload["symbol"]))
		if symbol == "" {
			symbol = strings.ToUpper(stringValue(payload["ticker"]))
		}
		if symbol != config.Symbol {
			continue
		}
		key := day(event.ObservationTime).Format("2006-01-02")
		current, ok := latest[key]
		if !ok || event.ProcessingTime.After(current.ProcessingTime) || (event.ProcessingTime.Equal(current.ProcessingTime) && event.EventID > current.EventID) {
			latest[key] = event
		}
	}
	out := make([]pricePoint, 0, len(latest))
	for _, event := range latest {
		var payload map[string]any
		_ = json.Unmarshal(event.NormalizedPayload, &payload)
		open, okOpen := numberValue(payload["open"])
		high, okHigh := numberValue(payload["high"])
		low, okLow := numberValue(payload["low"])
		closeValue, okClose := numberValue(payload["close"])
		valid := okOpen && okHigh && okLow && okClose && open > 0 && high > 0 && low > 0 && closeValue > 0 && high >= low
		out = append(out, pricePoint{day(event.ObservationTime), event.EventID, open, high, low, closeValue, valid})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].session.Before(out[j].session) })
	return out, nil
}

func buildOutcome(config BuildConfig, item source, prices []pricePoint, horizon int) (storage.MarketOpsSignalOutcomeRecord, error) {
	identity, err := marketopsstate.NewIdentity(marketopsstate.IdentityOutcome, item.tenantID, item.sourceType, item.sourceID, fmt.Sprint(horizon), CalculationVersion)
	if err != nil {
		return storage.MarketOpsSignalOutcomeRecord{}, err
	}
	record := storage.MarketOpsSignalOutcomeRecord{
		OutcomeID: identity.ID, TenantID: item.tenantID, AppID: item.appID, SourceType: item.sourceType, SourceID: item.sourceID,
		HypothesisKey: item.hypothesisKey, HypothesisVersion: item.hypothesisVersion, AssetID: item.assetID, Symbol: item.symbol,
		Direction: item.direction, OriginSessionDate: item.origin, HorizonSessions: horizon, OutcomeStatus: storage.MarketOpsOutcomePending,
		OutcomePayloadJSON: []byte(`{}`), CalculationVersion: CalculationVersion, CalculationRunID: config.RunID, DeterministicKey: identity.DeterministicKey,
	}
	originIndex := -1
	for i := range prices {
		if prices[i].session.Equal(item.origin) {
			originIndex = i
			break
		}
	}
	if originIndex < 0 || !prices[originIndex].valid {
		record.OutcomeStatus = priceGapStatus(config.AsOf, item.origin, horizon)
		record.OutcomePayloadJSON = outcomePayload(config, item, nil, nil, "origin_price_unavailable")
		return record, nil
	}
	record.OriginEventID = prices[originIndex].eventID
	if originIndex+horizon >= len(prices) {
		record.OutcomeStatus = priceGapStatus(config.AsOf, item.origin, horizon)
		record.OutcomePayloadJSON = outcomePayload(config, item, &prices[originIndex], nil, "forward_horizon_unavailable")
		return record, nil
	}
	window := prices[originIndex+1 : originIndex+horizon+1]
	for _, point := range window {
		if !point.valid {
			record.OutcomeStatus = storage.MarketOpsOutcomeMissingPrice
			record.OutcomePayloadJSON = outcomePayload(config, item, &prices[originIndex], window, "invalid_forward_price")
			return record, nil
		}
	}
	origin := prices[originIndex]
	final := window[len(window)-1]
	forwardReturn := final.close/origin.close - 1
	upExcursion, downExcursion := -math.MaxFloat64, math.MaxFloat64
	eventIDs := make([]string, 0, len(window))
	for _, point := range window {
		upExcursion = math.Max(upExcursion, point.high/origin.close-1)
		downExcursion = math.Min(downExcursion, point.low/origin.close-1)
		eventIDs = append(eventIDs, point.eventID)
	}
	mfe, mae := directionalExcursions(item.direction, upExcursion, downExcursion)
	drawdown := maximumDrawdown(append([]pricePoint{origin}, window...))
	directionalHit := directionHit(item.direction, forwardReturn)
	thresholdHit, daysToThreshold := thresholdOutcome(item.direction, origin.close, window, config.Threshold)
	record.OutcomeStatus = storage.MarketOpsOutcomeMatured
	record.MaturedSessionDate = timePtr(final.session)
	record.ForwardReturn = floatPtr(forwardReturn)
	record.MaxFavorableExcursion = mfe
	record.MaxAdverseExcursion = mae
	record.MaximumDrawdown = floatPtr(drawdown)
	record.RealizedVolChange = realizedVolChange(prices, originIndex, horizon)
	record.DirectionalHit = directionalHit
	record.ThresholdHit = boolPtr(thresholdHit)
	record.DaysToThreshold = daysToThreshold
	record.OutcomeEventIDs = eventIDs
	record.OutcomePayloadJSON = outcomePayload(config, item, &origin, window, "")
	return record, nil
}

func outcomePayload(config BuildConfig, item source, origin *pricePoint, window []pricePoint, reason string) []byte {
	payload := map[string]any{
		"as_of_date": config.AsOf.Format("2006-01-02"), "threshold": config.Threshold,
		"source_type": item.sourceType, "source_id": item.sourceID, "direction": item.direction,
		"price_basis": "normalized_equity_eod", "excursion_basis": "direction_adjusted_intraday_high_low",
	}
	if origin != nil {
		payload["origin_event_id"] = origin.eventID
		payload["origin_close"] = origin.close
	}
	if len(window) > 0 {
		ids := make([]string, 0, len(window))
		for _, point := range window {
			ids = append(ids, point.eventID)
		}
		payload["forward_event_ids"] = ids
	}
	if reason != "" {
		payload["status_reason"] = reason
	}
	encoded, _ := json.Marshal(payload)
	return encoded
}

func priceGapStatus(asOf, origin time.Time, horizon int) string {
	if !asOf.Before(addBusinessDays(origin, horizon)) {
		return storage.MarketOpsOutcomeMissingPrice
	}
	return storage.MarketOpsOutcomePending
}

func addBusinessDays(value time.Time, sessions int) time.Time {
	result := day(value)
	for sessions > 0 {
		result = result.AddDate(0, 0, 1)
		if result.Weekday() != time.Saturday && result.Weekday() != time.Sunday {
			sessions--
		}
	}
	return result
}

func directionalExcursions(direction string, up, down float64) (*float64, *float64) {
	switch direction {
	case "upside":
		return floatPtr(up), floatPtr(down)
	case "downside":
		return floatPtr(-down), floatPtr(-up)
	default:
		favorable := math.Max(math.Abs(up), math.Abs(down))
		return floatPtr(favorable), nil
	}
}

func directionHit(direction string, value float64) *bool {
	switch direction {
	case "upside":
		return boolPtr(value > 0)
	case "downside":
		return boolPtr(value < 0)
	default:
		return nil
	}
}

func thresholdOutcome(direction string, origin float64, window []pricePoint, threshold float64) (bool, *int) {
	for i, point := range window {
		up := point.high/origin - 1
		down := point.low/origin - 1
		hit := (direction == "upside" && up >= threshold) ||
			(direction == "downside" && down <= -threshold) ||
			(direction == "non_directional" && (up >= threshold || down <= -threshold))
		if hit {
			days := i + 1
			return true, &days
		}
	}
	return false, nil
}

func maximumDrawdown(points []pricePoint) float64 {
	peak, drawdown := points[0].close, 0.0
	for _, point := range points[1:] {
		if point.close > peak {
			peak = point.close
		}
		drawdown = math.Min(drawdown, point.close/peak-1)
	}
	return drawdown
}

func realizedVolChange(prices []pricePoint, originIndex, horizon int) *float64 {
	if horizon < 2 || originIndex < horizon || originIndex+horizon >= len(prices) {
		return nil
	}
	before := prices[originIndex-horizon : originIndex+1]
	after := prices[originIndex : originIndex+horizon+1]
	baseline, okBaseline := annualizedVol(before)
	forward, okForward := annualizedVol(after)
	if !okBaseline || !okForward {
		return nil
	}
	return floatPtr(forward - baseline)
}

func annualizedVol(points []pricePoint) (float64, bool) {
	if len(points) < 3 {
		return 0, false
	}
	returns := make([]float64, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		if !points[i-1].valid || !points[i].valid || points[i-1].close <= 0 {
			return 0, false
		}
		returns = append(returns, points[i].close/points[i-1].close-1)
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
	variance /= float64(len(returns) - 1)
	return math.Sqrt(variance) * math.Sqrt(252), true
}

func validDirection(value string) bool {
	return value == "upside" || value == "downside" || value == "non_directional"
}

func appID(value string) string {
	if strings.TrimSpace(value) == "" {
		return "marketops"
	}
	return strings.TrimSpace(value)
}

func day(value time.Time) time.Time {
	if value.IsZero() {
		return value
	}
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return ""
	}
}

func numberValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	case json.Number:
		result, err := typed.Float64()
		return result, err == nil
	default:
		return 0, false
	}
}

func floatPtr(value float64) *float64 { return &value }
func boolPtr(value bool) *bool        { return &value }
func timePtr(value time.Time) *time.Time {
	value = day(value)
	return &value
}

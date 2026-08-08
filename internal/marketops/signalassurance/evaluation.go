package signalassurance

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

// EvaluationMarketState is the complete point-in-time market input for one
// assertion session. A nil price deliberately produces an append-only
// INCOMPLETE evaluation; it never invents a lifecycle outcome.
type EvaluationMarketState struct {
	AsOf               time.Time
	AssetPrice         *float64
	BenchmarkPrice     *float64
	TradingDaysActive  int
	InputSnapshotJSON  []byte
	Superseded         bool
	SupersessionReason string
}

type EvaluationResult struct {
	Persistence storage.SignalAssuranceEvaluationPersistence
}

func Evaluate(assertion storage.SignalAssertionRecord, contract storage.SignalValidationContractRecord, previous *storage.SignalAssertionEvaluationRecord, market EvaluationMarketState) (EvaluationResult, error) {
	if assertion.State != storage.SignalAssertionActive {
		return EvaluationResult{}, errors.New("only ACTIVE assertions can be evaluated")
	}
	if err := ValidateContract(contract); err != nil {
		return EvaluationResult{}, err
	}
	market.AsOf = day(market.AsOf)
	if market.AsOf.IsZero() || market.TradingDaysActive < 0 {
		return EvaluationResult{}, errors.New("evaluation date and trading days are required")
	}
	base, err := baselinePrices(assertion.BaselineSnapshotJSON)
	if err != nil {
		return EvaluationResult{}, err
	}
	evaluation := storage.SignalAssertionEvaluationRecord{
		EvaluationID: deterministicID("saf_eval", assertion.AssertionID, market.AsOf.Format("2006-01-02"), EvaluationEngineVersion),
		AssertionID:  assertion.AssertionID, EvaluatedAt: market.AsOf, EvaluationSessionDate: market.AsOf,
		EvaluationMode: assertion.EvaluationMode, EvaluationRunID: assertion.EvaluationRunID,
		InputSnapshotJSON: copyJSONOrEmpty(market.InputSnapshotJSON), InputCompleteness: storage.SignalAssuranceInputComplete,
		TransitionSequence: assertion.TransitionSequence, TradingDaysActive: market.TradingDaysActive,
		CalendarDaysActive: calendarDays(assertion.ConfirmedAt, market.AsOf), EvaluationVersion: EvaluationEngineVersion,
	}
	if market.AssetPrice == nil || *market.AssetPrice <= 0 || (contract.PrimaryMetric == "benchmark_relative_return" && (market.BenchmarkPrice == nil || *market.BenchmarkPrice <= 0 || base.BenchmarkPrice <= 0)) {
		evaluation.InputCompleteness = storage.SignalAssuranceInputIncomplete
		return EvaluationResult{Persistence: storage.SignalAssuranceEvaluationPersistence{Evaluation: evaluation, PreviousState: assertion.State, NextState: assertion.State, ReasonCode: "required_input_missing"}}, nil
	}
	assetReturn := *market.AssetPrice/base.AssetPrice - 1
	evaluation.AssetPrice, evaluation.AbsoluteReturn = floatPtr(*market.AssetPrice), floatPtr(assetReturn)
	metric := assetReturn
	if market.BenchmarkPrice != nil && base.BenchmarkPrice > 0 {
		benchmarkReturn := *market.BenchmarkPrice/base.BenchmarkPrice - 1
		relative := assetReturn - benchmarkReturn
		evaluation.BenchmarkPrice, evaluation.BenchmarkReturn, evaluation.BenchmarkRelativeReturn = floatPtr(*market.BenchmarkPrice), floatPtr(benchmarkReturn), floatPtr(relative)
		if contract.PrimaryMetric == "benchmark_relative_return" {
			metric = relative
		}
	}
	normalized := metric
	if assertion.SignalDirection == "bearish" {
		normalized = -normalized
	}
	previousMFE, previousMAE := 0.0, 0.0
	if previous != nil {
		if previous.MFE != nil {
			previousMFE = *previous.MFE
		}
		if previous.MAE != nil {
			previousMAE = *previous.MAE
		}
	}
	if previous == nil {
		previousMFE, previousMAE = normalized, normalized
	}
	mfe, mae := math.Max(previousMFE, normalized), math.Min(previousMAE, normalized)
	evaluation.MFE, evaluation.MAE = floatPtr(mfe), floatPtr(mae)
	materialized := compare(normalized, contract.ComparisonOperator, contract.Threshold)
	invalidated := invalidationMet(contract.ConfigJSON, normalized)
	evaluation.MaterializationConditionMet, evaluation.InvalidationConditionMet = materialized, invalidated
	next, reason := assertion.State, "evaluated"
	// Precedence is normative: materialization, invalidation, supersession, expiry.
	switch {
	case materialized:
		next, reason = storage.SignalAssertionMaterialized, "materialization_condition_met"
	case invalidated:
		next, reason = storage.SignalAssertionInvalidated, "invalidation_condition_met"
	case market.Superseded:
		next, reason = storage.SignalAssertionSuperseded, firstNonEmpty(market.SupersessionReason, "superseded")
	case market.TradingDaysActive >= contract.MaxHorizonTradingDays:
		next, reason = storage.SignalAssertionExpired, "max_horizon_reached"
	}
	return EvaluationResult{Persistence: storage.SignalAssuranceEvaluationPersistence{Evaluation: evaluation, PreviousState: assertion.State, NextState: next, ReasonCode: reason, EventDetailsJSON: evaluationDetails(metric, normalized, market)}}, nil
}

type baseline struct {
	AssetPrice     float64 `json:"asset_price"`
	BenchmarkPrice float64 `json:"benchmark_price"`
}

func baselinePrices(value []byte) (baseline, error) {
	var out baseline
	if json.Unmarshal(value, &out) != nil || out.AssetPrice <= 0 {
		return baseline{}, errors.New("assertion baseline asset price is invalid")
	}
	return out, nil
}
func compare(value float64, op string, threshold float64) bool {
	switch op {
	case ">=":
		return value >= threshold
	case ">":
		return value > threshold
	case "<=":
		return value <= threshold
	case "<":
		return value < threshold
	}
	return false
}
func invalidationMet(config []byte, normalized float64) bool {
	var c struct {
		InvalidationThreshold *float64 `json:"invalidation_threshold"`
	}
	if json.Unmarshal(config, &c) != nil || c.InvalidationThreshold == nil {
		return false
	}
	return normalized <= *c.InvalidationThreshold
}
func calendarDays(from, to time.Time) int {
	if to.Before(from) {
		return 0
	}
	return int(to.Sub(from).Hours() / 24)
}
func day(value time.Time) time.Time {
	if value.IsZero() {
		return value
	}
	return time.Date(value.UTC().Year(), value.UTC().Month(), value.UTC().Day(), 0, 0, 0, 0, time.UTC)
}
func floatPtr(value float64) *float64 { return &value }
func copyJSONOrEmpty(value []byte) []byte {
	if len(value) == 0 {
		return []byte(`{}`)
	}
	return append([]byte(nil), value...)
}
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
func evaluationDetails(metric, normalized float64, market EvaluationMarketState) []byte {
	value, _ := json.Marshal(map[string]any{"primary_metric_value": metric, "normalized_metric_value": normalized, "as_of": day(market.AsOf).Format("2006-01-02")})
	return value
}

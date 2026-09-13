package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/lukebabs/signalops/internal/storage"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

const signalAssuranceMetricDefinitionVersion = "saf_effectiveness.v1"
const signalAssuranceMinimumRankedSample = 30

type effectivenessObservation struct {
	source, algorithm, version, signalType, direction, state                   string
	confidence, absoluteReturn, relativeReturn, sectorRelativeReturn, mfe, mae *float64
	broadMarketBenchmarkState, sectorBenchmarkState                            string
	horizon                                                                    int
	complete                                                                   bool
	legacyHit                                                                  *bool
	id, referenceID, symbol, evaluationMode                                    string
	score                                                                      *float64
	originAt, outcomeAt                                                        *time.Time
	calculationVersion, calculationRunID                                       string
}
type effectivenessAccumulator struct {
	sample, hits, materialized, invalidated, expired, censored, excluded int
	absoluteSum, relativeSum, sectorRelativeSum, mfeSum, maeSum          float64
	absoluteN, relativeN, sectorRelativeN, mfeN, maeN                    int
}

func (r *Repository) ListSignalAssuranceEffectiveness(ctx context.Context, f storage.SignalAssuranceEffectivenessFilter) ([]storage.SignalAssuranceEffectivenessRecord, error) {
	sources := effectivenessSources(f.EvidenceSource)
	observations := make([]effectivenessObservation, 0)
	for _, source := range sources {
		var values []effectivenessObservation
		var err error
		if source == "SAF" {
			values, err = r.listSAFEffectivenessObservations(ctx, f)
		} else {
			values, err = r.listLegacyEffectivenessObservations(ctx, f)
		}
		if err != nil {
			return nil, err
		}
		observations = append(observations, values...)
	}
	observations = filterEffectivenessObservationsByOutcomeStart(observations, f.OutcomeNotBefore)
	return aggregateEffectiveness(observations, normalizedEffectivenessDimension(f.Dimension)), nil
}
func (r *Repository) ListSignalAssuranceEffectivenessObservations(ctx context.Context, f storage.SignalAssuranceEffectivenessFilter) ([]storage.SignalAssuranceEffectivenessObservationRecord, error) {
	dimension := normalizedEffectivenessDimension(f.Dimension)
	cohort := strings.TrimSpace(f.DimensionValue)
	if cohort == "" {
		return []storage.SignalAssuranceEffectivenessObservationRecord{}, nil
	}
	values := make([]effectivenessObservation, 0)
	for _, source := range effectivenessSources(f.EvidenceSource) {
		var sourceValues []effectivenessObservation
		var err error
		if source == "SAF" {
			sourceValues, err = r.listSAFEffectivenessObservations(ctx, f)
		} else {
			sourceValues, err = r.listLegacyEffectivenessObservations(ctx, f)
		}
		if err != nil {
			return nil, err
		}
		for _, value := range sourceValues {
			if dimensionValue(value, dimension) == cohort && effectivenessObservationIncluded(value) {
				values = append(values, value)
			}
		}
	}
	values = filterEffectivenessObservationsByOutcomeStart(values, f.OutcomeNotBefore)
	sort.Slice(values, func(i, j int) bool {
		if values[i].outcomeAt == nil {
			return false
		}
		if values[j].outcomeAt == nil {
			return true
		}
		return values[i].outcomeAt.After(*values[j].outcomeAt)
	})
	limit := clampLimit(f.Limit)
	if limit <= 0 {
		limit = 200
	}
	if len(values) > limit {
		values = values[:limit]
	}
	out := make([]storage.SignalAssuranceEffectivenessObservationRecord, 0, len(values))
	for _, value := range values {
		var hit *bool
		if value.legacyHit != nil {
			hit = value.legacyHit
		} else if value.absoluteReturn != nil {
			v := *value.absoluteReturn > 0
			if value.direction == "bearish" {
				v = *value.absoluteReturn < 0
			}
			hit = &v
		}
		var directionalReturn *float64
		if value.absoluteReturn != nil {
			v := *value.absoluteReturn
			if value.direction == "bearish" || value.direction == "downside" {
				v = -v
			}
			directionalReturn = &v
		}
		out = append(out, storage.SignalAssuranceEffectivenessObservationRecord{EvidenceSource: value.source, ObservationID: value.id, ReferenceID: value.referenceID, Symbol: value.symbol, SignalType: value.signalType, Direction: value.direction, Algorithm: value.algorithm, AlgorithmVersion: value.version, State: value.state, EvaluationMode: value.evaluationMode, HorizonSessions: value.horizon, SignalScore: value.score, Confidence: value.confidence, DirectionalHit: hit, AbsoluteReturn: value.absoluteReturn, DirectionalReturn: directionalReturn, RelativeReturn: value.relativeReturn, SectorRelativeReturn: value.sectorRelativeReturn, MFE: value.mfe, MAE: value.mae, OriginAt: value.originAt, OutcomeAt: value.outcomeAt, CalculationVersion: value.calculationVersion, CalculationRunID: value.calculationRunID})
	}
	return out, nil
}

func (r *Repository) listSAFEffectivenessObservations(ctx context.Context, f storage.SignalAssuranceEffectivenessFilter) ([]effectivenessObservation, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT a.assertion_id,a.signal_id,a.symbol,a.algorithm,a.algorithm_version,a.signal_type,a.signal_direction,a.state,a.evaluation_mode,a.signal_score,a.confidence,a.confirmed_at,COALESCE(e.trading_days_active,0),e.evaluation_session_date,e.absolute_return,e.benchmark_relative_return,e.mfe,e.mae,COALESCE(e.input_completeness,'INCOMPLETE'),COALESCE(e.evaluation_version,''),COALESCE(e.evaluation_run_id,'') FROM signal_assertions a LEFT JOIN LATERAL (SELECT trading_days_active,evaluation_session_date,absolute_return,benchmark_relative_return,mfe,mae,input_completeness,evaluation_version,evaluation_run_id FROM signal_assertion_evaluations WHERE assertion_id=a.assertion_id ORDER BY evaluation_session_date DESC LIMIT 1) e ON true WHERE a.tenant_id=$1 AND ($2='' OR a.evaluation_mode=$2)`, strings.TrimSpace(f.TenantID), strings.TrimSpace(f.EvaluationMode))
	if err != nil {
		return nil, fmt.Errorf("list SAF effectiveness observations: %w", err)
	}
	defer rows.Close()
	out := []effectivenessObservation{}
	for rows.Next() {
		var x effectivenessObservation
		var confidence, score, absolute, relative, mfe, mae sql.NullFloat64
		var origin time.Time
		var outcome sql.NullTime
		var completeness string
		if err := rows.Scan(&x.id, &x.referenceID, &x.symbol, &x.algorithm, &x.version, &x.signalType, &x.direction, &x.state, &x.evaluationMode, &score, &confidence, &origin, &x.horizon, &outcome, &absolute, &relative, &mfe, &mae, &completeness, &x.calculationVersion, &x.calculationRunID); err != nil {
			return nil, err
		}
		x.source = "SAF"
		x.referenceID = x.id
		x.complete = completeness == storage.SignalAssuranceInputComplete && absolute.Valid
		x.confidence = nullableFloat(confidence)
		x.score = nullableFloat(score)
		x.originAt = &origin
		if outcome.Valid {
			value := outcome.Time
			x.outcomeAt = &value
		}
		x.absoluteReturn = nullableFloat(absolute)
		x.relativeReturn = nullableFloat(relative)
		x.mfe = nullableFloat(mfe)
		x.mae = nullableFloat(mae)
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *Repository) listLegacyEffectivenessObservations(ctx context.Context, f storage.SignalAssuranceEffectivenessFilter) ([]effectivenessObservation, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT o.outcome_id,o.source_id,o.symbol,o.direction,o.horizon_sessions,o.origin_session_date,o.matured_session_date,o.directional_hit,o.forward_return,o.max_favorable_excursion,o.max_adverse_excursion,o.calculation_version,o.calculation_run_id,p.opportunity_score,p.confidence_score FROM marketops_signal_outcomes o JOIN marketops_opportunities p ON p.tenant_id=o.tenant_id AND p.opportunity_id=o.source_id WHERE o.tenant_id=$1 AND o.source_type='opportunity' AND o.outcome_status='matured' AND o.direction IN ('upside','downside')`, strings.TrimSpace(f.TenantID))
	if err != nil {
		return nil, fmt.Errorf("list legacy effectiveness observations: %w", err)
	}
	defer rows.Close()
	out := []effectivenessObservation{}
	for rows.Next() {
		var x effectivenessObservation
		var hit sql.NullBool
		var forward, confidence, score, mfe, mae sql.NullFloat64
		var origin, outcome time.Time
		if err := rows.Scan(&x.id, &x.referenceID, &x.symbol, &x.direction, &x.horizon, &origin, &outcome, &hit, &forward, &mfe, &mae, &x.calculationVersion, &x.calculationRunID, &score, &confidence); err != nil {
			return nil, err
		}
		x.source = "LEGACY"
		x.algorithm = "legacy_opportunity"
		x.version = "unattributable"
		x.signalType = "opportunity"
		x.state = "MATURED"
		x.originAt = &origin
		x.outcomeAt = &outcome
		x.score = nullableFloat(score)
		x.complete = hit.Valid
		if hit.Valid {
			value := hit.Bool
			x.legacyHit = &value
		}
		x.absoluteReturn = nullableFloat(forward)
		x.confidence = nullableFloat(confidence)
		x.mfe = nullableFloat(mfe)
		x.mae = nullableFloat(mae)
		out = append(out, x)
	}
	return out, rows.Err()
}

func filterEffectivenessObservationsByOutcomeStart(values []effectivenessObservation, cutoff *time.Time) []effectivenessObservation {
	if cutoff == nil {
		return values
	}
	out := make([]effectivenessObservation, 0, len(values))
	for _, value := range values {
		effectiveAt := value.outcomeAt
		if effectiveAt == nil {
			effectiveAt = value.originAt
		}
		if effectiveAt == nil {
			continue
		}
		if !effectiveAt.Before(*cutoff) {
			out = append(out, value)
		}
	}
	return out
}

func effectivenessObservationIncluded(x effectivenessObservation) bool {
	terminal := x.source == "LEGACY" || x.state == storage.SignalAssertionMaterialized || x.state == storage.SignalAssertionInvalidated || x.state == storage.SignalAssertionExpired || x.state == storage.SignalAssertionSuperseded || x.state == storage.SignalAssertionClosed
	return terminal && x.complete
}
func aggregateEffectiveness(values []effectivenessObservation, dimension string) []storage.SignalAssuranceEffectivenessRecord {
	grouped := map[string]*effectivenessAccumulator{}
	for _, x := range values {
		key := x.source + "\x00" + dimensionValue(x, dimension)
		a := grouped[key]
		if a == nil {
			a = &effectivenessAccumulator{}
			grouped[key] = a
		}
		terminal := x.source == "LEGACY" || x.state == storage.SignalAssertionMaterialized || x.state == storage.SignalAssertionInvalidated || x.state == storage.SignalAssertionExpired || x.state == storage.SignalAssertionSuperseded || x.state == storage.SignalAssertionClosed
		if x.source == "SAF" {
			switch x.state {
			case storage.SignalAssertionMaterialized:
				a.materialized++
			case storage.SignalAssertionInvalidated:
				a.invalidated++
			case storage.SignalAssertionExpired, storage.SignalAssertionSuperseded, storage.SignalAssertionClosed:
				a.expired++
			case storage.SignalAssertionActive:
				a.censored++
			}
		}
		if !terminal {
			continue
		}
		if !x.complete {
			a.excluded++
			continue
		}
		a.sample++
		hit := false
		if x.legacyHit != nil {
			hit = *x.legacyHit
		} else if x.absoluteReturn != nil {
			hit = *x.absoluteReturn > 0
			if x.direction == "bearish" {
				hit = *x.absoluteReturn < 0
			}
		}
		if hit {
			a.hits++
		}
		addMetric(&a.absoluteSum, &a.absoluteN, x.absoluteReturn)
		addMetric(&a.relativeSum, &a.relativeN, x.relativeReturn)
		addMetric(&a.sectorRelativeSum, &a.sectorRelativeN, x.sectorRelativeReturn)
		addMetric(&a.mfeSum, &a.mfeN, x.mfe)
		addMetric(&a.maeSum, &a.maeN, x.mae)
	}
	now := time.Now().UTC()
	out := make([]storage.SignalAssuranceEffectivenessRecord, 0, len(grouped))
	for key, a := range grouped {
		parts := strings.SplitN(key, "\x00", 2)
		record := storage.SignalAssuranceEffectivenessRecord{EvidenceSource: parts[0], Dimension: dimension, DimensionValue: parts[1], SampleSize: a.sample, DirectionalHits: a.hits, MaterializedCount: a.materialized, InvalidatedCount: a.invalidated, ExpiredCount: a.expired, CensoredCount: a.censored, ExcludedCount: a.excluded, Exploratory: a.sample < signalAssuranceMinimumRankedSample, AsOf: now, MetricDefinitionVersion: signalAssuranceMetricDefinitionVersion}
		if a.sample > 0 {
			accuracy := float64(a.hits) / float64(a.sample)
			lower, upper := wilsonInterval(a.hits, a.sample)
			record.DirectionalAccuracy = &accuracy
			record.AccuracyLowerBound = &lower
			record.AccuracyUpperBound = &upper
		}
		terminalCount := a.materialized + a.invalidated + a.expired
		if terminalCount > 0 {
			rate := float64(a.materialized) / float64(terminalCount)
			record.MaterializationRate = &rate
		}
		record.AverageReturn = averageMetric(a.absoluteSum, a.absoluteN)
		record.AverageRelativeReturn = averageMetric(a.relativeSum, a.relativeN)
		record.AverageSectorRelativeReturn = averageMetric(a.sectorRelativeSum, a.sectorRelativeN)
		record.BroadMarketBenchmarkSampleSize = a.relativeN
		record.SectorBenchmarkSampleSize = a.sectorRelativeN
		record.AverageMFE = averageMetric(a.mfeSum, a.mfeN)
		record.AverageMAE = averageMetric(a.maeSum, a.maeN)
		out = append(out, record)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].EvidenceSource < out[j].EvidenceSource || out[i].EvidenceSource == out[j].EvidenceSource && out[i].DimensionValue < out[j].DimensionValue
	})
	return out
}
func (r *Repository) ListSignalAssuranceRecommendations(ctx context.Context, f storage.SignalAssuranceEffectivenessFilter) ([]storage.SignalAssuranceRecommendationRecord, error) {
	return signalAssuranceRecommendations(f, func(filter storage.SignalAssuranceEffectivenessFilter) ([]storage.SignalAssuranceEffectivenessRecord, error) {
		return r.ListSignalAssuranceEffectiveness(ctx, filter)
	})
}

func signalAssuranceRecommendations(f storage.SignalAssuranceEffectivenessFilter, list func(storage.SignalAssuranceEffectivenessFilter) ([]storage.SignalAssuranceEffectivenessRecord, error)) ([]storage.SignalAssuranceRecommendationRecord, error) {
	dimensions := []string{"algorithm_version", "signal_type", "confidence_band"}
	out := []storage.SignalAssuranceRecommendationRecord{}
	for _, dimension := range dimensions {
		rows, err := list(storage.SignalAssuranceEffectivenessFilter{TenantID: f.TenantID, EvidenceSource: f.EvidenceSource, EvaluationMode: f.EvaluationMode, Dimension: dimension, OutcomeNotBefore: f.OutcomeNotBefore})
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			if row.Exploratory || row.DirectionalAccuracy == nil || row.AccuracyUpperBound == nil {
				continue
			}
			if *row.AccuracyUpperBound < .5 {
				out = append(out, storage.SignalAssuranceRecommendationRecord{RecommendationID: "saf_recommendation_" + row.EvidenceSource + "_" + dimension + "_" + strings.ReplaceAll(row.DimensionValue, " ", "_"), EvidenceSource: row.EvidenceSource, Dimension: dimension, DimensionValue: row.DimensionValue, Priority: "high", Kind: "directional_accuracy", Summary: "Directional accuracy remains below the 50% reference even at the upper 95% confidence bound; inspect feature inputs, confirmation logic, and contract thresholds.", SampleSize: row.SampleSize, DirectionalAccuracy: row.DirectionalAccuracy, AccuracyUpperBound: row.AccuracyUpperBound, MetricDefinitionVersion: signalAssuranceMetricDefinitionVersion, AsOf: row.AsOf})
			} else if dimension == "confidence_band" && confidenceCalibrationGap(row) > .15 {
				out = append(out, storage.SignalAssuranceRecommendationRecord{RecommendationID: "saf_recommendation_calibration_" + row.EvidenceSource + "_" + strings.ReplaceAll(row.DimensionValue, " ", "_"), EvidenceSource: row.EvidenceSource, Dimension: dimension, DimensionValue: row.DimensionValue, Priority: "medium", Kind: "confidence_calibration", Summary: "Observed directional accuracy differs materially from the declared confidence band; review confidence calibration before increasing reliance on this cohort.", SampleSize: row.SampleSize, DirectionalAccuracy: row.DirectionalAccuracy, AccuracyUpperBound: row.AccuracyUpperBound, MetricDefinitionVersion: signalAssuranceMetricDefinitionVersion, AsOf: row.AsOf})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Priority < out[j].Priority || out[i].Priority == out[j].Priority && out[i].SampleSize > out[j].SampleSize
	})
	return out, nil
}
func effectivenessSources(value string) []string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "SAF":
		return []string{"SAF"}
	case "LEGACY":
		return []string{"LEGACY"}
	default:
		return []string{"SAF", "LEGACY"}
	}
}
func normalizedEffectivenessDimension(value string) string {
	switch strings.TrimSpace(value) {
	case "algorithm_version", "signal_type", "direction", "confidence_band", "horizon", "benchmark_coverage":
		return strings.TrimSpace(value)
	default:
		return "overall"
	}
}
func dimensionValue(x effectivenessObservation, dimension string) string {
	switch dimension {
	case "algorithm_version":
		if x.source == "LEGACY" {
			return "legacy_opportunity@unattributable"
		}
		return x.algorithm + "@" + x.version
	case "signal_type":
		return x.signalType
	case "direction":
		return x.direction
	case "confidence_band":
		return confidenceBand(x.confidence)
	case "horizon":
		return strconv.Itoa(x.horizon) + " sessions"
	case "benchmark_coverage":
		return benchmarkCoverageBand(x)
	default:
		return "all"
	}
}
func benchmarkCoverageBand(x effectivenessObservation) string {
	broad := strings.TrimSpace(x.broadMarketBenchmarkState)
	sector := strings.TrimSpace(x.sectorBenchmarkState)
	if broad == "" {
		broad = "not_recorded"
	}
	if sector == "" {
		sector = "not_recorded"
	}
	return "broad=" + broad + "; sector=" + sector
}

func confidenceBand(value *float64) string {
	if value == nil {
		return "unknown"
	}
	v := *value
	switch {
	case v < .2:
		return "0-20%"
	case v < .4:
		return "20-40%"
	case v < .6:
		return "40-60%"
	case v < .8:
		return "60-80%"
	default:
		return "80-100%"
	}
}
func nullableFloat(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	out := value.Float64
	return &out
}
func addMetric(sum *float64, n *int, value *float64) {
	if value != nil {
		*sum += *value
		*n++
	}
}
func averageMetric(sum float64, n int) *float64 {
	if n == 0 {
		return nil
	}
	value := sum / float64(n)
	return &value
}
func wilsonInterval(hits, n int) (float64, float64) {
	if n == 0 {
		return 0, 0
	}
	z := 1.959963984540054
	p := float64(hits) / float64(n)
	d := 1 + z*z/float64(n)
	center := (p + z*z/(2*float64(n))) / d
	spread := z * math.Sqrt((p*(1-p)+z*z/(4*float64(n)))/float64(n)) / d
	return math.Max(0, center-spread), math.Min(1, center+spread)
}
func confidenceCalibrationGap(row storage.SignalAssuranceEffectivenessRecord) float64 {
	if row.DirectionalAccuracy == nil {
		return 0
	}
	parts := strings.Split(strings.TrimSuffix(row.DimensionValue, "%"), "-")
	if len(parts) != 2 {
		return 0
	}
	low, _ := strconv.ParseFloat(parts[0], 64)
	high, _ := strconv.ParseFloat(strings.TrimSuffix(parts[1], "%"), 64)
	return math.Abs(*row.DirectionalAccuracy - (low+high)/200)
}

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

// ListSubscriberGlobalSignalAssuranceEffectiveness calculates effectiveness
// from the platform-owned outcome projection. SAF assertions are intentionally
// absent until a real confirmed assertion is recorded globally.
func (r *Repository) ListSubscriberGlobalSignalAssuranceEffectiveness(ctx context.Context, symbols []string, f storage.SignalAssuranceEffectivenessFilter) ([]storage.SignalAssuranceEffectivenessRecord, error) {
	values, err := r.listSubscriberGlobalHistoricalAssuranceObservations(ctx, symbols, f)
	if err != nil {
		return nil, err
	}
	return aggregateEffectiveness(values, normalizedEffectivenessDimension(f.Dimension)), nil
}

func (r *Repository) ListSubscriberGlobalSignalAssuranceEffectivenessObservations(ctx context.Context, symbols []string, f storage.SignalAssuranceEffectivenessFilter) ([]storage.SignalAssuranceEffectivenessObservationRecord, error) {
	cohort := strings.TrimSpace(f.DimensionValue)
	if cohort == "" {
		return []storage.SignalAssuranceEffectivenessObservationRecord{}, nil
	}
	values, err := r.listSubscriberGlobalHistoricalAssuranceObservations(ctx, symbols, f)
	if err != nil {
		return nil, err
	}
	dimension := normalizedEffectivenessDimension(f.Dimension)
	out := make([]storage.SignalAssuranceEffectivenessObservationRecord, 0, len(values))
	for _, value := range values {
		if dimensionValue(value, dimension) != cohort {
			continue
		}
		directionalReturn := value.absoluteReturn
		if directionalReturn != nil && value.direction == "downside" {
			inverted := -*directionalReturn
			directionalReturn = &inverted
		}
		out = append(out, storage.SignalAssuranceEffectivenessObservationRecord{
			EvidenceSource: value.source, ObservationID: value.id, ReferenceID: value.referenceID,
			Symbol: value.symbol, SignalType: value.signalType, Direction: value.direction,
			Algorithm: value.algorithm, AlgorithmVersion: value.version, State: value.state,
			EvaluationMode: value.evaluationMode, HorizonSessions: value.horizon,
			DirectionalHit: value.legacyHit, AbsoluteReturn: value.absoluteReturn,
			DirectionalReturn: directionalReturn, MFE: value.mfe, MAE: value.mae,
			OriginAt: value.originAt, OutcomeAt: value.outcomeAt,
			CalculationVersion: value.calculationVersion, CalculationRunID: value.calculationRunID,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].OutcomeAt == nil {
			return false
		}
		if out[j].OutcomeAt == nil {
			return true
		}
		return out[i].OutcomeAt.After(*out[j].OutcomeAt)
	})
	limit := clampLimit(f.Limit)
	if limit <= 0 {
		limit = 200
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *Repository) ListSubscriberGlobalSignalAssuranceRecommendations(ctx context.Context, symbols []string, f storage.SignalAssuranceEffectivenessFilter) ([]storage.SignalAssuranceRecommendationRecord, error) {
	return signalAssuranceRecommendations(f, func(filter storage.SignalAssuranceEffectivenessFilter) ([]storage.SignalAssuranceEffectivenessRecord, error) {
		return r.ListSubscriberGlobalSignalAssuranceEffectiveness(ctx, symbols, filter)
	})
}

func (r *Repository) listSubscriberGlobalHistoricalAssuranceObservations(ctx context.Context, symbols []string, f storage.SignalAssuranceEffectivenessFilter) ([]effectivenessObservation, error) {
	if len(symbols) == 0 || strings.EqualFold(strings.TrimSpace(f.EvidenceSource), "SAF") {
		return []effectivenessObservation{}, nil
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT observation_id, source_id, symbol, direction, horizon_sessions,
  origin_session_date, matured_session_date, directional_hit, forward_return,
  mfe, mae, calculation_version, calculation_run_id
FROM subscriber_gateway_global_signal_assurance_observations
WHERE upper(symbol) = ANY($1)
ORDER BY matured_session_date DESC NULLS LAST, origin_session_date DESC, observation_id
LIMIT $2`, pqArray(symbols), clampLimit(f.Limit))
	if err != nil {
		return nil, fmt.Errorf("list subscriber global historical assurance observations: %w", err)
	}
	defer rows.Close()
	out := []effectivenessObservation{}
	for rows.Next() {
		var x effectivenessObservation
		var origin time.Time
		var matured sql.NullTime
		var hit sql.NullBool
		var forward, mfe, mae sql.NullFloat64
		if err := rows.Scan(&x.id, &x.referenceID, &x.symbol, &x.direction, &x.horizon, &origin, &matured, &hit, &forward, &mfe, &mae, &x.calculationVersion, &x.calculationRunID); err != nil {
			return nil, fmt.Errorf("scan subscriber global historical assurance observation: %w", err)
		}
		x.source, x.algorithm, x.version, x.signalType, x.state = "LEGACY", "legacy_opportunity", "unattributable", "opportunity", "MATURED"
		x.complete, x.originAt = hit.Valid, &origin
		if matured.Valid {
			value := matured.Time
			x.outcomeAt = &value
		}
		if hit.Valid {
			value := hit.Bool
			x.legacyHit = &value
		}
		x.absoluteReturn, x.mfe, x.mae = nullableFloat(forward), nullableFloat(mfe), nullableFloat(mae)
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate subscriber global historical assurance observations: %w", err)
	}
	return out, nil
}

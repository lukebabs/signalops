package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) UpsertMarketOpsSignalOutcome(ctx context.Context, record storage.MarketOpsSignalOutcomeRecord) error {
	if err := validateMarketOpsSignalOutcome(record); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO marketops_signal_outcomes (
 outcome_id, tenant_id, app_id, source_type, source_id, hypothesis_key, hypothesis_version,
 asset_id, symbol, direction, origin_session_date, horizon_sessions, matured_session_date,
 outcome_status, forward_return, max_favorable_excursion, max_adverse_excursion,
 maximum_drawdown, realized_vol_change, directional_hit, threshold_hit, days_to_threshold,
 origin_event_id, outcome_event_ids, outcome_payload, calculation_version,
 calculation_run_id, deterministic_key
) VALUES ($1,$2,$3,$4,$5,NULLIF($6,''),NULLIF($7,''),$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,NULLIF($23,''),$24,$25,$26,$27,$28)
ON CONFLICT (tenant_id, deterministic_key) DO UPDATE SET
 outcome_status=EXCLUDED.outcome_status, matured_session_date=EXCLUDED.matured_session_date,
 forward_return=EXCLUDED.forward_return, max_favorable_excursion=EXCLUDED.max_favorable_excursion,
 max_adverse_excursion=EXCLUDED.max_adverse_excursion, maximum_drawdown=EXCLUDED.maximum_drawdown,
 realized_vol_change=EXCLUDED.realized_vol_change, directional_hit=EXCLUDED.directional_hit,
 threshold_hit=EXCLUDED.threshold_hit, days_to_threshold=EXCLUDED.days_to_threshold,
 origin_event_id=EXCLUDED.origin_event_id, outcome_event_ids=EXCLUDED.outcome_event_ids,
 outcome_payload=EXCLUDED.outcome_payload, calculation_run_id=EXCLUDED.calculation_run_id,
 updated_at=now()
WHERE marketops_signal_outcomes.outcome_status <> 'matured'
 AND NOT (marketops_signal_outcomes.outcome_status = 'missing_price' AND EXCLUDED.outcome_status = 'pending')`,
		record.OutcomeID, strings.TrimSpace(record.TenantID), recordAppID(record.AppID),
		strings.TrimSpace(record.SourceType), strings.TrimSpace(record.SourceID),
		strings.TrimSpace(record.HypothesisKey), strings.TrimSpace(record.HypothesisVersion),
		strings.TrimSpace(record.AssetID), strings.ToUpper(strings.TrimSpace(record.Symbol)),
		strings.TrimSpace(record.Direction), record.OriginSessionDate.UTC(), record.HorizonSessions,
		record.MaturedSessionDate, strings.TrimSpace(record.OutcomeStatus), record.ForwardReturn,
		record.MaxFavorableExcursion, record.MaxAdverseExcursion, record.MaximumDrawdown,
		record.RealizedVolChange, record.DirectionalHit, record.ThresholdHit, record.DaysToThreshold,
		strings.TrimSpace(record.OriginEventID), pqArray(record.OutcomeEventIDs),
		jsonOrEmpty(record.OutcomePayloadJSON), strings.TrimSpace(record.CalculationVersion),
		strings.TrimSpace(record.CalculationRunID), strings.TrimSpace(record.DeterministicKey))
	if err != nil {
		return fmt.Errorf("upsert marketops signal outcome: %w", err)
	}
	return nil
}

func (r *Repository) ListMarketOpsSignalOutcomes(ctx context.Context, filter storage.MarketOpsSignalOutcomeFilter) ([]storage.MarketOpsSignalOutcomeRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsSignalOutcomeSelect+`
WHERE ($1='' OR tenant_id=$1) AND ($2='' OR app_id=$2)
 AND ($3='' OR source_type=$3) AND ($4='' OR source_id=$4)
 AND ($5='' OR hypothesis_key=$5) AND ($6='' OR hypothesis_version=$6)
 AND ($7='' OR symbol=$7) AND ($8='' OR direction=$8)
 AND ($9='' OR outcome_status=$9) AND ($10=0 OR horizon_sessions=$10)
 AND ($11::timestamptz IS NULL OR origin_session_date >= $11::date)
 AND ($12::timestamptz IS NULL OR origin_session_date <= $12::date)
ORDER BY origin_session_date DESC, source_type, source_id, horizon_sessions LIMIT $13`,
		strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AppID),
		strings.TrimSpace(filter.SourceType), strings.TrimSpace(filter.SourceID),
		strings.TrimSpace(filter.HypothesisKey), strings.TrimSpace(filter.HypothesisVersion),
		strings.ToUpper(strings.TrimSpace(filter.Symbol)), strings.TrimSpace(filter.Direction),
		strings.TrimSpace(filter.OutcomeStatus), filter.HorizonSessions,
		nullTime(filter.OriginStart), nullTime(filter.OriginEnd), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops signal outcomes: %w", err)
	}
	defer rows.Close()
	out := []storage.MarketOpsSignalOutcomeRecord{}
	for rows.Next() {
		record, err := scanMarketOpsSignalOutcome(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

func (r *Repository) GetMarketOpsSignalOutcome(ctx context.Context, tenantID, outcomeID string) (storage.MarketOpsSignalOutcomeRecord, error) {
	return scanMarketOpsSignalOutcome(r.db.QueryRowContext(ctx, marketOpsSignalOutcomeSelect+` WHERE tenant_id=$1 AND outcome_id=$2`,
		strings.TrimSpace(tenantID), strings.TrimSpace(outcomeID)))
}

const marketOpsSignalOutcomeSelect = `
SELECT outcome_id, tenant_id, app_id, source_type, source_id,
 COALESCE(hypothesis_key,''), COALESCE(hypothesis_version,''), asset_id, symbol, direction,
 origin_session_date, horizon_sessions, matured_session_date, outcome_status, forward_return,
 max_favorable_excursion, max_adverse_excursion, maximum_drawdown, realized_vol_change,
 directional_hit, threshold_hit, days_to_threshold, COALESCE(origin_event_id,''),
 COALESCE(array_to_json(outcome_event_ids),'[]'::json)::text, outcome_payload,
 calculation_version, calculation_run_id, deterministic_key, created_at, updated_at
FROM marketops_signal_outcomes`

func scanMarketOpsSignalOutcome(scanner interface{ Scan(...any) error }) (storage.MarketOpsSignalOutcomeRecord, error) {
	var record storage.MarketOpsSignalOutcomeRecord
	var eventIDsJSON string
	err := scanner.Scan(&record.OutcomeID, &record.TenantID, &record.AppID, &record.SourceType,
		&record.SourceID, &record.HypothesisKey, &record.HypothesisVersion, &record.AssetID,
		&record.Symbol, &record.Direction, &record.OriginSessionDate, &record.HorizonSessions,
		&record.MaturedSessionDate, &record.OutcomeStatus, &record.ForwardReturn,
		&record.MaxFavorableExcursion, &record.MaxAdverseExcursion, &record.MaximumDrawdown,
		&record.RealizedVolChange, &record.DirectionalHit, &record.ThresholdHit,
		&record.DaysToThreshold, &record.OriginEventID, &eventIDsJSON,
		&record.OutcomePayloadJSON, &record.CalculationVersion, &record.CalculationRunID,
		&record.DeterministicKey, &record.CreatedAt, &record.UpdatedAt)
	if err != nil {
		return storage.MarketOpsSignalOutcomeRecord{}, mapScanError("scan marketops signal outcome", err)
	}
	if err := json.Unmarshal([]byte(eventIDsJSON), &record.OutcomeEventIDs); err != nil {
		return storage.MarketOpsSignalOutcomeRecord{}, err
	}
	return record, nil
}

func validateMarketOpsSignalOutcome(record storage.MarketOpsSignalOutcomeRecord) error {
	if err := requireMarketOpsFields("signal outcome", map[string]string{
		"outcome_id": record.OutcomeID, "tenant_id": record.TenantID, "source_type": record.SourceType,
		"source_id": record.SourceID, "asset_id": record.AssetID, "symbol": record.Symbol,
		"direction": record.Direction, "outcome_status": record.OutcomeStatus,
		"calculation_version": record.CalculationVersion, "calculation_run_id": record.CalculationRunID,
		"deterministic_key": record.DeterministicKey,
	}); err != nil {
		return err
	}
	if record.OriginSessionDate.IsZero() {
		return fmt.Errorf("marketops signal outcome origin session is required")
	}
	if !oneOf(record.SourceType, storage.MarketOpsOutcomeSourceHypothesisEvaluation, storage.MarketOpsOutcomeSourceOpportunity, storage.MarketOpsOutcomeSourceSignal) {
		return fmt.Errorf("marketops signal outcome source_type is invalid")
	}
	if record.SourceType == storage.MarketOpsOutcomeSourceHypothesisEvaluation &&
		(strings.TrimSpace(record.HypothesisKey) == "" || strings.TrimSpace(record.HypothesisVersion) == "") {
		return fmt.Errorf("marketops hypothesis outcome requires hypothesis key and version")
	}
	if !oneOf(record.Direction, "upside", "downside", "non_directional") {
		return fmt.Errorf("marketops signal outcome direction is invalid")
	}
	if record.HorizonSessions != 1 && record.HorizonSessions != 5 && record.HorizonSessions != 10 && record.HorizonSessions != 20 {
		return fmt.Errorf("marketops signal outcome horizon is invalid")
	}
	if !oneOf(record.OutcomeStatus, storage.MarketOpsOutcomePending, storage.MarketOpsOutcomeMatured, storage.MarketOpsOutcomeMissingPrice) {
		return fmt.Errorf("marketops signal outcome status is invalid")
	}
	if record.OutcomeStatus == storage.MarketOpsOutcomeMatured {
		if record.MaturedSessionDate == nil || record.ForwardReturn == nil || record.MaturedSessionDate.Before(record.OriginSessionDate) {
			return fmt.Errorf("matured marketops signal outcome requires valid maturity and return")
		}
	} else if record.MaturedSessionDate != nil {
		return fmt.Errorf("unmatured marketops signal outcome cannot have a maturity date")
	}
	if record.OutcomeStatus != storage.MarketOpsOutcomeMatured && (record.ForwardReturn != nil || record.MaxFavorableExcursion != nil || record.MaxAdverseExcursion != nil || record.MaximumDrawdown != nil || record.RealizedVolChange != nil || record.DirectionalHit != nil || record.ThresholdHit != nil || record.DaysToThreshold != nil) {
		return fmt.Errorf("unmatured marketops signal outcome cannot have realized metrics")
	}
	if record.DaysToThreshold != nil && (*record.DaysToThreshold <= 0 || *record.DaysToThreshold > record.HorizonSessions) {
		return fmt.Errorf("marketops signal outcome days_to_threshold is invalid")
	}
	for name, value := range map[string]*float64{
		"forward_return": record.ForwardReturn, "max_favorable_excursion": record.MaxFavorableExcursion,
		"max_adverse_excursion": record.MaxAdverseExcursion, "maximum_drawdown": record.MaximumDrawdown,
		"realized_vol_change": record.RealizedVolChange,
	} {
		if value != nil && (math.IsNaN(*value) || math.IsInf(*value, 0)) {
			return fmt.Errorf("marketops signal outcome %s must be finite", name)
		}
	}
	return validateJSONObject("marketops signal outcome payload", jsonOrEmpty(record.OutcomePayloadJSON))
}

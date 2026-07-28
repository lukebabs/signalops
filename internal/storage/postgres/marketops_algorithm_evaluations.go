package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) UpsertMarketOpsAlgorithmEvaluationRun(ctx context.Context, record storage.MarketOpsAlgorithmEvaluationRunRecord) error {
	if err := validateMarketOpsAlgorithmEvaluationRun(record); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO marketops_algorithm_evaluation_runs (
 run_id, tenant_id, app_id, universe_group, algorithm_ids, modes, window_start, window_end, as_of_date,
 status, parameters, coverage, metrics, error_message, requested_by, started_at, completed_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
ON CONFLICT (run_id) DO UPDATE SET
 status=EXCLUDED.status, parameters=EXCLUDED.parameters, coverage=EXCLUDED.coverage, metrics=EXCLUDED.metrics,
 error_message=EXCLUDED.error_message, completed_at=EXCLUDED.completed_at, updated_at=now()`,
		strings.TrimSpace(record.RunID), strings.TrimSpace(record.TenantID), recordAppID(record.AppID), strings.TrimSpace(record.UniverseGroup),
		pqArray(record.AlgorithmIDs), pqArray(record.Modes), record.WindowStart.UTC(), record.WindowEnd.UTC(), record.AsOfDate.UTC(),
		strings.TrimSpace(record.Status), jsonOrEmpty(record.ParametersJSON), jsonOrEmpty(record.CoverageJSON), jsonOrEmpty(record.MetricsJSON),
		strings.TrimSpace(record.ErrorMessage), firstNonEmptyString(record.RequestedBy, "operator-local"), record.StartedAt.UTC(), record.CompletedAt)
	if err != nil {
		return fmt.Errorf("upsert marketops algorithm evaluation run: %w", err)
	}
	return nil
}

func (r *Repository) GetMarketOpsAlgorithmEvaluationRun(ctx context.Context, tenantID, runID string) (storage.MarketOpsAlgorithmEvaluationRunRecord, error) {
	return scanMarketOpsAlgorithmEvaluationRun(r.db.QueryRowContext(ctx, marketOpsAlgorithmEvaluationRunSelect+` WHERE tenant_id=$1 AND run_id=$2`, strings.TrimSpace(tenantID), strings.TrimSpace(runID)))
}

func (r *Repository) ListMarketOpsAlgorithmEvaluationRuns(ctx context.Context, filter storage.MarketOpsAlgorithmEvaluationRunFilter) ([]storage.MarketOpsAlgorithmEvaluationRunRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsAlgorithmEvaluationRunSelect+`
WHERE tenant_id=$1 AND ($2='' OR $2 = ANY(algorithm_ids)) AND ($3='' OR status=$3)
ORDER BY started_at DESC LIMIT $4`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AlgorithmID), strings.TrimSpace(filter.Status), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops algorithm evaluation runs: %w", err)
	}
	defer rows.Close()
	out := []storage.MarketOpsAlgorithmEvaluationRunRecord{}
	for rows.Next() {
		record, err := scanMarketOpsAlgorithmEvaluationRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

const marketOpsAlgorithmEvaluationRunSelect = `
SELECT run_id, tenant_id, app_id, universe_group, COALESCE(array_to_json(algorithm_ids),'[]'::json)::text,
 COALESCE(array_to_json(modes),'[]'::json)::text, window_start, window_end, as_of_date, status,
 parameters, coverage, metrics, error_message, requested_by, started_at, completed_at, created_at, updated_at
FROM marketops_algorithm_evaluation_runs`

type marketOpsAlgorithmEvaluationRunScanner interface{ Scan(...any) error }

func scanMarketOpsAlgorithmEvaluationRun(scanner marketOpsAlgorithmEvaluationRunScanner) (storage.MarketOpsAlgorithmEvaluationRunRecord, error) {
	var record storage.MarketOpsAlgorithmEvaluationRunRecord
	var algorithmIDs, modes string
	if err := scanner.Scan(&record.RunID, &record.TenantID, &record.AppID, &record.UniverseGroup, &algorithmIDs, &modes,
		&record.WindowStart, &record.WindowEnd, &record.AsOfDate, &record.Status, &record.ParametersJSON, &record.CoverageJSON,
		&record.MetricsJSON, &record.ErrorMessage, &record.RequestedBy, &record.StartedAt, &record.CompletedAt, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return storage.MarketOpsAlgorithmEvaluationRunRecord{}, mapScanError("scan marketops algorithm evaluation run", err)
	}
	if err := json.Unmarshal([]byte(algorithmIDs), &record.AlgorithmIDs); err != nil {
		return record, err
	}
	if err := json.Unmarshal([]byte(modes), &record.Modes); err != nil {
		return record, err
	}
	return record, nil
}

func (r *Repository) InsertMarketOpsAlgorithmEvaluationResult(ctx context.Context, record storage.MarketOpsAlgorithmEvaluationResultRecord) error {
	if err := validateMarketOpsAlgorithmEvaluationResult(record); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO marketops_algorithm_evaluation_results (
 evaluation_result_id, run_id, tenant_id, algorithm_id, algorithm_version, evaluation_mode, evaluation_profile,
 result_type, symbol, observation_session_date, score, confidence, severity, direction, result_payload,
 input_provenance, source_event_ids, feature_value_ids, deterministic_key
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
ON CONFLICT (run_id, deterministic_key) DO NOTHING`, strings.TrimSpace(record.EvaluationResultID), strings.TrimSpace(record.RunID), strings.TrimSpace(record.TenantID),
		strings.TrimSpace(record.AlgorithmID), strings.TrimSpace(record.AlgorithmVersion), strings.TrimSpace(record.EvaluationMode),
		strings.TrimSpace(record.EvaluationProfile), strings.TrimSpace(record.ResultType), strings.ToUpper(strings.TrimSpace(record.Symbol)),
		record.ObservationSessionDate.UTC(), record.Score, record.Confidence, strings.TrimSpace(record.Severity), strings.TrimSpace(record.Direction),
		jsonOrEmpty(record.ResultPayloadJSON), jsonOrEmpty(record.InputProvenanceJSON), pqArray(record.SourceEventIDs), pqArray(record.FeatureValueIDs), strings.TrimSpace(record.DeterministicKey))
	if err != nil {
		return fmt.Errorf("insert marketops algorithm evaluation result: %w", err)
	}
	return nil
}

func (r *Repository) ListMarketOpsAlgorithmEvaluationResults(ctx context.Context, filter storage.MarketOpsAlgorithmEvaluationResultFilter) ([]storage.MarketOpsAlgorithmEvaluationResultRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsAlgorithmEvaluationResultSelect+`
WHERE tenant_id=$1 AND ($2='' OR run_id=$2) AND ($3='' OR algorithm_id=$3) AND ($4='' OR symbol=$4) AND ($5='' OR evaluation_mode=$5)
ORDER BY observation_session_date DESC, algorithm_id LIMIT $6`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.RunID), strings.TrimSpace(filter.AlgorithmID), strings.ToUpper(strings.TrimSpace(filter.Symbol)), strings.TrimSpace(filter.EvaluationMode), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops algorithm evaluation results: %w", err)
	}
	defer rows.Close()
	out := []storage.MarketOpsAlgorithmEvaluationResultRecord{}
	for rows.Next() {
		record, err := scanMarketOpsAlgorithmEvaluationResult(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

const marketOpsAlgorithmEvaluationResultSelect = `
SELECT evaluation_result_id, run_id, tenant_id, algorithm_id, algorithm_version, evaluation_mode, evaluation_profile,
 result_type, symbol, observation_session_date, score, confidence, severity, direction, result_payload, input_provenance,
 COALESCE(array_to_json(source_event_ids),'[]'::json)::text, COALESCE(array_to_json(feature_value_ids),'[]'::json)::text,
 deterministic_key, created_at
FROM marketops_algorithm_evaluation_results`

type marketOpsAlgorithmEvaluationResultScanner interface{ Scan(...any) error }

func scanMarketOpsAlgorithmEvaluationResult(scanner marketOpsAlgorithmEvaluationResultScanner) (storage.MarketOpsAlgorithmEvaluationResultRecord, error) {
	var record storage.MarketOpsAlgorithmEvaluationResultRecord
	var events, features string
	if err := scanner.Scan(&record.EvaluationResultID, &record.RunID, &record.TenantID, &record.AlgorithmID, &record.AlgorithmVersion,
		&record.EvaluationMode, &record.EvaluationProfile, &record.ResultType, &record.Symbol, &record.ObservationSessionDate,
		&record.Score, &record.Confidence, &record.Severity, &record.Direction, &record.ResultPayloadJSON, &record.InputProvenanceJSON,
		&events, &features, &record.DeterministicKey, &record.CreatedAt); err != nil {
		return record, mapScanError("scan marketops algorithm evaluation result", err)
	}
	if err := json.Unmarshal([]byte(events), &record.SourceEventIDs); err != nil {
		return record, err
	}
	if err := json.Unmarshal([]byte(features), &record.FeatureValueIDs); err != nil {
		return record, err
	}
	return record, nil
}

func (r *Repository) UpsertMarketOpsAlgorithmEvaluationOutcome(ctx context.Context, record storage.MarketOpsAlgorithmEvaluationOutcomeRecord) error {
	if err := validateMarketOpsAlgorithmEvaluationOutcome(record); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO marketops_algorithm_evaluation_outcomes (
 evaluation_outcome_id, run_id, evaluation_result_id, tenant_id, horizon_sessions, outcome_status, matured_session_date,
 forward_return, absolute_forward_return, max_favorable_excursion, max_adverse_excursion, maximum_drawdown,
 realized_vol_change, directional_hit, threshold_hit, outcome_event_ids, outcome_payload, deterministic_key
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
ON CONFLICT (run_id, deterministic_key) DO UPDATE SET
 outcome_status=EXCLUDED.outcome_status, matured_session_date=EXCLUDED.matured_session_date, forward_return=EXCLUDED.forward_return,
 absolute_forward_return=EXCLUDED.absolute_forward_return, max_favorable_excursion=EXCLUDED.max_favorable_excursion,
 max_adverse_excursion=EXCLUDED.max_adverse_excursion, maximum_drawdown=EXCLUDED.maximum_drawdown,
 realized_vol_change=EXCLUDED.realized_vol_change, directional_hit=EXCLUDED.directional_hit, threshold_hit=EXCLUDED.threshold_hit,
 outcome_event_ids=EXCLUDED.outcome_event_ids, outcome_payload=EXCLUDED.outcome_payload, updated_at=now()
WHERE marketops_algorithm_evaluation_outcomes.outcome_status <> 'matured'`,
		strings.TrimSpace(record.EvaluationOutcomeID), strings.TrimSpace(record.RunID), strings.TrimSpace(record.EvaluationResultID), strings.TrimSpace(record.TenantID),
		record.HorizonSessions, strings.TrimSpace(record.OutcomeStatus), record.MaturedSessionDate, record.ForwardReturn, record.AbsoluteForwardReturn,
		record.MaxFavorableExcursion, record.MaxAdverseExcursion, record.MaximumDrawdown, record.RealizedVolChange, record.DirectionalHit,
		record.ThresholdHit, pqArray(record.OutcomeEventIDs), jsonOrEmpty(record.OutcomePayloadJSON), strings.TrimSpace(record.DeterministicKey))
	if err != nil {
		return fmt.Errorf("upsert marketops algorithm evaluation outcome: %w", err)
	}
	return nil
}

func (r *Repository) ListMarketOpsAlgorithmEvaluationOutcomes(ctx context.Context, filter storage.MarketOpsAlgorithmEvaluationOutcomeFilter) ([]storage.MarketOpsAlgorithmEvaluationOutcomeRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsAlgorithmEvaluationOutcomeSelect+`
WHERE tenant_id=$1 AND ($2='' OR run_id=$2) AND ($3='' OR evaluation_result_id=$3) AND ($4='' OR outcome_status=$4) AND ($5=0 OR horizon_sessions=$5)
ORDER BY created_at DESC LIMIT $6`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.RunID), strings.TrimSpace(filter.EvaluationResultID), strings.TrimSpace(filter.OutcomeStatus), filter.HorizonSessions, clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops algorithm evaluation outcomes: %w", err)
	}
	defer rows.Close()
	out := []storage.MarketOpsAlgorithmEvaluationOutcomeRecord{}
	for rows.Next() {
		record, err := scanMarketOpsAlgorithmEvaluationOutcome(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

const marketOpsAlgorithmEvaluationOutcomeSelect = `
SELECT evaluation_outcome_id, run_id, evaluation_result_id, tenant_id, horizon_sessions, outcome_status, matured_session_date,
 forward_return, absolute_forward_return, max_favorable_excursion, max_adverse_excursion, maximum_drawdown, realized_vol_change,
 directional_hit, threshold_hit, COALESCE(array_to_json(outcome_event_ids),'[]'::json)::text, outcome_payload, deterministic_key, created_at, updated_at
FROM marketops_algorithm_evaluation_outcomes`

type marketOpsAlgorithmEvaluationOutcomeScanner interface{ Scan(...any) error }

func scanMarketOpsAlgorithmEvaluationOutcome(scanner marketOpsAlgorithmEvaluationOutcomeScanner) (storage.MarketOpsAlgorithmEvaluationOutcomeRecord, error) {
	var record storage.MarketOpsAlgorithmEvaluationOutcomeRecord
	var eventIDs string
	if err := scanner.Scan(&record.EvaluationOutcomeID, &record.RunID, &record.EvaluationResultID, &record.TenantID, &record.HorizonSessions,
		&record.OutcomeStatus, &record.MaturedSessionDate, &record.ForwardReturn, &record.AbsoluteForwardReturn, &record.MaxFavorableExcursion,
		&record.MaxAdverseExcursion, &record.MaximumDrawdown, &record.RealizedVolChange, &record.DirectionalHit, &record.ThresholdHit,
		&eventIDs, &record.OutcomePayloadJSON, &record.DeterministicKey, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return record, mapScanError("scan marketops algorithm evaluation outcome", err)
	}
	if err := json.Unmarshal([]byte(eventIDs), &record.OutcomeEventIDs); err != nil {
		return record, err
	}
	return record, nil
}

func (r *Repository) UpsertMarketOpsAlgorithmEvaluationBackfillCampaign(ctx context.Context, record storage.MarketOpsAlgorithmEvaluationBackfillCampaignRecord) error {
	if err := validateMarketOpsAlgorithmEvaluationBackfillCampaign(record); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO marketops_algorithm_evaluation_backfill_campaigns (
 campaign_id, tenant_id, universe_group, window_start, window_end, status, parameters, coverage, child_run_ids,
 error_message, requested_by, started_at, completed_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
ON CONFLICT (campaign_id) DO UPDATE SET status=EXCLUDED.status, parameters=EXCLUDED.parameters, coverage=EXCLUDED.coverage,
 child_run_ids=EXCLUDED.child_run_ids, error_message=EXCLUDED.error_message, started_at=EXCLUDED.started_at,
 completed_at=EXCLUDED.completed_at, updated_at=now()`, strings.TrimSpace(record.CampaignID), strings.TrimSpace(record.TenantID),
		strings.TrimSpace(record.UniverseGroup), record.WindowStart.UTC(), record.WindowEnd.UTC(), strings.TrimSpace(record.Status),
		jsonOrEmpty(record.ParametersJSON), jsonOrEmpty(record.CoverageJSON), pqArray(record.ChildRunIDs), strings.TrimSpace(record.ErrorMessage),
		firstNonEmptyString(record.RequestedBy, "operator-local"), record.StartedAt, record.CompletedAt)
	if err != nil {
		return fmt.Errorf("upsert marketops algorithm evaluation backfill campaign: %w", err)
	}
	return nil
}

func (r *Repository) GetMarketOpsAlgorithmEvaluationBackfillCampaign(ctx context.Context, tenantID, campaignID string) (storage.MarketOpsAlgorithmEvaluationBackfillCampaignRecord, error) {
	return scanMarketOpsAlgorithmEvaluationBackfillCampaign(r.db.QueryRowContext(ctx, marketOpsAlgorithmEvaluationBackfillCampaignSelect+` WHERE tenant_id=$1 AND campaign_id=$2`, strings.TrimSpace(tenantID), strings.TrimSpace(campaignID)))
}

func (r *Repository) ListMarketOpsAlgorithmEvaluationBackfillCampaigns(ctx context.Context, filter storage.MarketOpsAlgorithmEvaluationBackfillCampaignFilter) ([]storage.MarketOpsAlgorithmEvaluationBackfillCampaignRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsAlgorithmEvaluationBackfillCampaignSelect+` WHERE tenant_id=$1 AND ($2='' OR status=$2) ORDER BY created_at DESC LIMIT $3`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.Status), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops algorithm evaluation backfill campaigns: %w", err)
	}
	defer rows.Close()
	out := []storage.MarketOpsAlgorithmEvaluationBackfillCampaignRecord{}
	for rows.Next() {
		record, err := scanMarketOpsAlgorithmEvaluationBackfillCampaign(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

const marketOpsAlgorithmEvaluationBackfillCampaignSelect = `
SELECT campaign_id, tenant_id, universe_group, window_start, window_end, status, parameters, coverage,
 COALESCE(array_to_json(child_run_ids),'[]'::json)::text, error_message, requested_by, started_at, completed_at, created_at, updated_at
FROM marketops_algorithm_evaluation_backfill_campaigns`

type marketOpsAlgorithmEvaluationBackfillCampaignScanner interface{ Scan(...any) error }

func scanMarketOpsAlgorithmEvaluationBackfillCampaign(scanner marketOpsAlgorithmEvaluationBackfillCampaignScanner) (storage.MarketOpsAlgorithmEvaluationBackfillCampaignRecord, error) {
	var record storage.MarketOpsAlgorithmEvaluationBackfillCampaignRecord
	var children string
	if err := scanner.Scan(&record.CampaignID, &record.TenantID, &record.UniverseGroup, &record.WindowStart, &record.WindowEnd,
		&record.Status, &record.ParametersJSON, &record.CoverageJSON, &children, &record.ErrorMessage, &record.RequestedBy,
		&record.StartedAt, &record.CompletedAt, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return record, mapScanError("scan marketops algorithm evaluation backfill campaign", err)
	}
	if err := json.Unmarshal([]byte(children), &record.ChildRunIDs); err != nil {
		return record, err
	}
	return record, nil
}

func validateMarketOpsAlgorithmEvaluationRun(record storage.MarketOpsAlgorithmEvaluationRunRecord) error {
	if err := requireMarketOpsFields("algorithm evaluation run", map[string]string{"run_id": record.RunID, "tenant_id": record.TenantID, "universe_group": record.UniverseGroup, "status": record.Status}); err != nil {
		return err
	}
	if len(record.AlgorithmIDs) == 0 || len(record.Modes) == 0 || record.WindowStart.IsZero() || record.WindowEnd.IsZero() || record.AsOfDate.IsZero() || !record.WindowEnd.After(record.WindowStart) || record.AsOfDate.Before(record.WindowStart) {
		return fmt.Errorf("marketops algorithm evaluation run window or algorithms are invalid")
	}
	if !oneOf(record.Status, storage.MarketOpsAlgorithmEvaluationStatusRunning, storage.MarketOpsAlgorithmEvaluationStatusSucceeded, storage.MarketOpsAlgorithmEvaluationStatusPartial, storage.MarketOpsAlgorithmEvaluationStatusFailed) {
		return fmt.Errorf("marketops algorithm evaluation run status is invalid")
	}
	for _, mode := range record.Modes {
		if !oneOf(mode, storage.MarketOpsAlgorithmEvaluationModeRetrospective, storage.MarketOpsAlgorithmEvaluationModeWalkForward) {
			return fmt.Errorf("marketops algorithm evaluation mode is invalid")
		}
	}
	return validateJSONObject("marketops algorithm evaluation run parameters", jsonOrEmpty(record.ParametersJSON))
}

func validateMarketOpsAlgorithmEvaluationResult(record storage.MarketOpsAlgorithmEvaluationResultRecord) error {
	if err := requireMarketOpsFields("algorithm evaluation result", map[string]string{"evaluation_result_id": record.EvaluationResultID, "run_id": record.RunID, "tenant_id": record.TenantID, "algorithm_id": record.AlgorithmID, "algorithm_version": record.AlgorithmVersion, "evaluation_mode": record.EvaluationMode, "evaluation_profile": record.EvaluationProfile, "result_type": record.ResultType, "symbol": record.Symbol, "severity": record.Severity, "direction": record.Direction, "deterministic_key": record.DeterministicKey}); err != nil {
		return err
	}
	if record.ObservationSessionDate.IsZero() || math.IsNaN(record.Score) || math.IsInf(record.Score, 0) || record.Score < 0 || math.IsNaN(record.Confidence) || math.IsInf(record.Confidence, 0) || record.Confidence < 0 || record.Confidence > 1 {
		return fmt.Errorf("marketops algorithm evaluation result metrics are invalid")
	}
	if !oneOf(record.EvaluationMode, storage.MarketOpsAlgorithmEvaluationModeRetrospective, storage.MarketOpsAlgorithmEvaluationModeWalkForward) || !oneOf(record.EvaluationProfile, storage.MarketOpsAlgorithmEvaluationProfileDirectional, storage.MarketOpsAlgorithmEvaluationProfileEventStudy, storage.MarketOpsAlgorithmEvaluationProfileForecast, storage.MarketOpsAlgorithmEvaluationProfileClassification) || !oneOf(record.Severity, "info", "low", "medium", "high", "critical") || !oneOf(record.Direction, "upside", "downside", "non_directional") {
		return fmt.Errorf("marketops algorithm evaluation result category is invalid")
	}
	return validateJSONObject("marketops algorithm evaluation result payload", jsonOrEmpty(record.ResultPayloadJSON))
}

func validateMarketOpsAlgorithmEvaluationOutcome(record storage.MarketOpsAlgorithmEvaluationOutcomeRecord) error {
	if err := requireMarketOpsFields("algorithm evaluation outcome", map[string]string{"evaluation_outcome_id": record.EvaluationOutcomeID, "run_id": record.RunID, "evaluation_result_id": record.EvaluationResultID, "tenant_id": record.TenantID, "outcome_status": record.OutcomeStatus, "deterministic_key": record.DeterministicKey}); err != nil {
		return err
	}
	if record.HorizonSessions != 1 && record.HorizonSessions != 5 && record.HorizonSessions != 10 && record.HorizonSessions != 20 {
		return fmt.Errorf("marketops algorithm evaluation horizon is invalid")
	}
	if !oneOf(record.OutcomeStatus, storage.MarketOpsOutcomePending, storage.MarketOpsOutcomeMatured, storage.MarketOpsOutcomeMissingPrice) {
		return fmt.Errorf("marketops algorithm evaluation outcome status is invalid")
	}
	if record.OutcomeStatus == storage.MarketOpsOutcomeMatured && (record.MaturedSessionDate == nil || record.ForwardReturn == nil) {
		return fmt.Errorf("matured algorithm evaluation outcome requires maturity and return")
	}
	if record.OutcomeStatus != storage.MarketOpsOutcomeMatured && (record.MaturedSessionDate != nil || record.ForwardReturn != nil) {
		return fmt.Errorf("unmatured algorithm evaluation outcome cannot include maturity or return")
	}
	return validateJSONObject("marketops algorithm evaluation outcome payload", jsonOrEmpty(record.OutcomePayloadJSON))
}

func validateMarketOpsAlgorithmEvaluationBackfillCampaign(record storage.MarketOpsAlgorithmEvaluationBackfillCampaignRecord) error {
	if err := requireMarketOpsFields("algorithm evaluation backfill campaign", map[string]string{"campaign_id": record.CampaignID, "tenant_id": record.TenantID, "universe_group": record.UniverseGroup, "status": record.Status}); err != nil {
		return err
	}
	if record.WindowStart.IsZero() || record.WindowEnd.IsZero() || !record.WindowEnd.After(record.WindowStart) {
		return fmt.Errorf("marketops algorithm evaluation backfill campaign window is invalid")
	}
	if !oneOf(record.Status, storage.MarketOpsAlgorithmBackfillStatusPlanned, storage.MarketOpsAlgorithmBackfillStatusRunning, storage.MarketOpsAlgorithmBackfillStatusSucceeded, storage.MarketOpsAlgorithmBackfillStatusPartial, storage.MarketOpsAlgorithmBackfillStatusFailed) {
		return fmt.Errorf("marketops algorithm evaluation backfill campaign status is invalid")
	}
	return validateJSONObject("marketops algorithm evaluation backfill campaign parameters", jsonOrEmpty(record.ParametersJSON))
}

package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) UpsertMarketOpsIntelligenceCohortRun(ctx context.Context, x storage.MarketOpsIntelligenceCohortRunRecord) error {
	if strings.TrimSpace(x.RunID) == "" || strings.TrimSpace(x.TenantID) == "" || strings.TrimSpace(x.Actor) == "" || x.MaxSymbols < 1 || x.MaxSymbols > 10 || x.SessionStart.IsZero() || x.SessionEnd.Before(x.SessionStart) {
		return fmt.Errorf("cohort run requires run_id, tenant_id, actor, valid sessions, and max_symbols 1..10")
	}
	result, err := r.db.ExecContext(ctx, `
INSERT INTO marketops_intelligence_cohort_runs
(run_id,tenant_id,app_id,universe_group,requested_symbols,resolved_symbols,stages,max_symbols,dry_run,continue_on_error,status,aggregate_metrics,errors,actor,session_start,session_end,started_at,completed_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
ON CONFLICT (run_id) DO UPDATE SET
 status=EXCLUDED.status, aggregate_metrics=EXCLUDED.aggregate_metrics, errors=EXCLUDED.errors,
 completed_at=EXCLUDED.completed_at, updated_at=now()
WHERE marketops_intelligence_cohort_runs.tenant_id=EXCLUDED.tenant_id
 AND marketops_intelligence_cohort_runs.requested_symbols=EXCLUDED.requested_symbols
 AND marketops_intelligence_cohort_runs.resolved_symbols=EXCLUDED.resolved_symbols
 AND marketops_intelligence_cohort_runs.stages=EXCLUDED.stages
 AND marketops_intelligence_cohort_runs.session_start=EXCLUDED.session_start
 AND marketops_intelligence_cohort_runs.session_end=EXCLUDED.session_end
`, x.RunID, x.TenantID, recordAppID(x.AppID), x.UniverseGroup, pqArray(x.RequestedSymbols), pqArray(x.ResolvedSymbols), pqArray(x.Stages), x.MaxSymbols, x.DryRun, x.ContinueOnError, x.Status, jsonOrEmpty(x.AggregateJSON), jsonArrayOrEmpty(x.ErrorsJSON), x.Actor, x.SessionStart, x.SessionEnd, x.StartedAt, x.CompletedAt)
	if err != nil {
		return fmt.Errorf("upsert intelligence cohort run: %w", err)
	}
	if affected, rowsErr := result.RowsAffected(); rowsErr != nil || affected != 1 {
		return fmt.Errorf("cohort run_id conflicts with a different immutable scope")
	}
	return nil
}

func (r *Repository) UpsertMarketOpsIntelligenceCohortSymbolResult(ctx context.Context, x storage.MarketOpsIntelligenceCohortSymbolResultRecord) error {
	if x.ResultID == "" || x.RunID == "" || x.TenantID == "" || x.Symbol == "" {
		return fmt.Errorf("cohort symbol result requires result_id, run_id, tenant_id, and symbol")
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO marketops_intelligence_cohort_symbol_results
(result_id,run_id,tenant_id,universe_group,symbol,asset_id,stage_status,stage_errors,input_coverage,
 latest_market_state_id,latest_state_date,latest_state_schema_version,latest_state_quality,latest_state_completeness,
 required_feature_coverage,surface_coverage,evaluation_count,eligible_count,triggered_count,evaluation_rejection_reasons,
 opportunity_count,pending_outcome_count,matured_outcome_count,proposal_status_counts,exact_calibration_count,
 calibration_below_minimum,coverage_state,evaluation_state,governance_state,calibration_state,outcome_state,rollout_status,readiness_reasons)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33)
ON CONFLICT (run_id,symbol) DO UPDATE SET
 stage_status=EXCLUDED.stage_status,stage_errors=EXCLUDED.stage_errors,input_coverage=EXCLUDED.input_coverage,
 latest_market_state_id=EXCLUDED.latest_market_state_id,latest_state_date=EXCLUDED.latest_state_date,
 latest_state_schema_version=EXCLUDED.latest_state_schema_version,latest_state_quality=EXCLUDED.latest_state_quality,
 latest_state_completeness=EXCLUDED.latest_state_completeness,required_feature_coverage=EXCLUDED.required_feature_coverage,
 surface_coverage=EXCLUDED.surface_coverage,evaluation_count=EXCLUDED.evaluation_count,eligible_count=EXCLUDED.eligible_count,
 triggered_count=EXCLUDED.triggered_count,evaluation_rejection_reasons=EXCLUDED.evaluation_rejection_reasons,
 opportunity_count=EXCLUDED.opportunity_count,pending_outcome_count=EXCLUDED.pending_outcome_count,
 matured_outcome_count=EXCLUDED.matured_outcome_count,proposal_status_counts=EXCLUDED.proposal_status_counts,
 exact_calibration_count=EXCLUDED.exact_calibration_count,calibration_below_minimum=EXCLUDED.calibration_below_minimum,
 coverage_state=EXCLUDED.coverage_state,evaluation_state=EXCLUDED.evaluation_state,governance_state=EXCLUDED.governance_state,
 calibration_state=EXCLUDED.calibration_state,outcome_state=EXCLUDED.outcome_state,rollout_status=EXCLUDED.rollout_status,
 readiness_reasons=EXCLUDED.readiness_reasons,updated_at=now()
`, x.ResultID, x.RunID, x.TenantID, x.UniverseGroup, strings.ToUpper(x.Symbol), x.AssetID, jsonOrEmpty(x.StageStatusJSON), jsonOrEmpty(x.StageErrorsJSON), jsonOrEmpty(x.InputCoverageJSON), x.LatestMarketStateID, x.LatestStateDate, x.LatestStateSchemaVersion, x.LatestStateQuality, x.LatestStateCompleteness, x.RequiredFeatureCoverage, x.SurfaceCoverage, x.EvaluationCount, x.EligibleCount, x.TriggeredCount, pqArray(x.EvaluationRejectionReasons), x.OpportunityCount, x.PendingOutcomeCount, x.MaturedOutcomeCount, jsonOrEmpty(x.ProposalStatusCountsJSON), x.ExactCalibrationCount, x.CalibrationBelowMinimum, x.CoverageState, x.EvaluationState, x.GovernanceState, x.CalibrationState, x.OutcomeState, x.RolloutStatus, pqArray(x.ReadinessReasons))
	if err != nil {
		return fmt.Errorf("upsert intelligence cohort symbol result: %w", err)
	}
	return nil
}

const cohortRunSelect = `SELECT run_id,tenant_id,app_id,universe_group,
 COALESCE(array_to_json(requested_symbols),'[]'::json)::text,COALESCE(array_to_json(resolved_symbols),'[]'::json)::text,
 COALESCE(array_to_json(stages),'[]'::json)::text,max_symbols,dry_run,continue_on_error,status,aggregate_metrics,errors,
 actor,session_start,session_end,started_at,completed_at,created_at,updated_at FROM marketops_intelligence_cohort_runs`

func (r *Repository) ListMarketOpsIntelligenceCohortRuns(ctx context.Context, f storage.MarketOpsIntelligenceCohortRunFilter) ([]storage.MarketOpsIntelligenceCohortRunRecord, error) {
	rows, err := r.db.QueryContext(ctx, cohortRunSelect+` WHERE ($1='' OR tenant_id=$1) AND ($2='' OR universe_group=$2) AND ($3='' OR status=$3) ORDER BY started_at DESC LIMIT $4`, f.TenantID, f.UniverseGroup, f.Status, clampLimit(f.Limit))
	if err != nil {
		return nil, fmt.Errorf("list intelligence cohort runs: %w", err)
	}
	defer rows.Close()
	out := []storage.MarketOpsIntelligenceCohortRunRecord{}
	for rows.Next() {
		x, e := scanCohortRun(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *Repository) GetMarketOpsIntelligenceCohortRun(ctx context.Context, tenantID, runID string) (storage.MarketOpsIntelligenceCohortRunRecord, error) {
	return scanCohortRun(r.db.QueryRowContext(ctx, cohortRunSelect+` WHERE tenant_id=$1 AND run_id=$2`, tenantID, runID))
}

func scanCohortRun(s interface{ Scan(...any) error }) (storage.MarketOpsIntelligenceCohortRunRecord, error) {
	var x storage.MarketOpsIntelligenceCohortRunRecord
	var requested, resolved, stages string
	if err := s.Scan(&x.RunID, &x.TenantID, &x.AppID, &x.UniverseGroup, &requested, &resolved, &stages, &x.MaxSymbols, &x.DryRun, &x.ContinueOnError, &x.Status, &x.AggregateJSON, &x.ErrorsJSON, &x.Actor, &x.SessionStart, &x.SessionEnd, &x.StartedAt, &x.CompletedAt, &x.CreatedAt, &x.UpdatedAt); err != nil {
		return x, mapScanError("scan intelligence cohort run", err)
	}
	for _, p := range []struct {
		raw string
		dst *[]string
	}{{requested, &x.RequestedSymbols}, {resolved, &x.ResolvedSymbols}, {stages, &x.Stages}} {
		if err := json.Unmarshal([]byte(p.raw), p.dst); err != nil {
			return x, err
		}
	}
	return x, nil
}

const cohortResultSelect = `SELECT result_id,run_id,tenant_id,universe_group,symbol,asset_id,stage_status,stage_errors,input_coverage,
 latest_market_state_id,latest_state_date,latest_state_schema_version,latest_state_quality,latest_state_completeness,
 required_feature_coverage,surface_coverage,evaluation_count,eligible_count,triggered_count,
 COALESCE(array_to_json(evaluation_rejection_reasons),'[]'::json)::text,opportunity_count,pending_outcome_count,
 matured_outcome_count,proposal_status_counts,exact_calibration_count,calibration_below_minimum,coverage_state,
 evaluation_state,governance_state,calibration_state,outcome_state,rollout_status,
 COALESCE(array_to_json(readiness_reasons),'[]'::json)::text,created_at,updated_at FROM marketops_intelligence_cohort_symbol_results`

func (r *Repository) ListMarketOpsIntelligenceCohortSymbolResults(ctx context.Context, tenantID, runID string) ([]storage.MarketOpsIntelligenceCohortSymbolResultRecord, error) {
	return r.listCohortResults(ctx, cohortResultSelect+` WHERE tenant_id=$1 AND run_id=$2 ORDER BY symbol LIMIT 10`, tenantID, runID)
}

func (r *Repository) ListMarketOpsIntelligenceReadiness(ctx context.Context, f storage.MarketOpsIntelligenceReadinessFilter) ([]storage.MarketOpsIntelligenceCohortSymbolResultRecord, error) {
	query := `SELECT * FROM (` + strings.Replace(cohortResultSelect, "SELECT ", "SELECT DISTINCT ON (symbol) ", 1) + ` WHERE ($1='' OR tenant_id=$1) AND ($2='' OR universe_group=$2)
 AND (cardinality($3::text[])=0 OR symbol=ANY($3::text[])) AND ($4::timestamptz IS NULL OR latest_state_date=$4::date)
 AND ($5='' OR rollout_status=$5) ORDER BY symbol,updated_at DESC) latest ORDER BY symbol LIMIT $6`
	return r.listCohortResults(ctx, query, f.TenantID, f.UniverseGroup, pqArray(f.Symbols), nullTime(f.LatestSession), f.RolloutStatus, clampLimit(f.Limit))
}

func (r *Repository) listCohortResults(ctx context.Context, query string, args ...any) ([]storage.MarketOpsIntelligenceCohortSymbolResultRecord, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list intelligence cohort results: %w", err)
	}
	defer rows.Close()
	out := []storage.MarketOpsIntelligenceCohortSymbolResultRecord{}
	for rows.Next() {
		x, e := scanCohortResult(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func scanCohortResult(s interface{ Scan(...any) error }) (storage.MarketOpsIntelligenceCohortSymbolResultRecord, error) {
	var x storage.MarketOpsIntelligenceCohortSymbolResultRecord
	var rejections, reasons string
	if err := s.Scan(&x.ResultID, &x.RunID, &x.TenantID, &x.UniverseGroup, &x.Symbol, &x.AssetID, &x.StageStatusJSON, &x.StageErrorsJSON, &x.InputCoverageJSON, &x.LatestMarketStateID, &x.LatestStateDate, &x.LatestStateSchemaVersion, &x.LatestStateQuality, &x.LatestStateCompleteness, &x.RequiredFeatureCoverage, &x.SurfaceCoverage, &x.EvaluationCount, &x.EligibleCount, &x.TriggeredCount, &rejections, &x.OpportunityCount, &x.PendingOutcomeCount, &x.MaturedOutcomeCount, &x.ProposalStatusCountsJSON, &x.ExactCalibrationCount, &x.CalibrationBelowMinimum, &x.CoverageState, &x.EvaluationState, &x.GovernanceState, &x.CalibrationState, &x.OutcomeState, &x.RolloutStatus, &reasons, &x.CreatedAt, &x.UpdatedAt); err != nil {
		return x, mapScanError("scan intelligence cohort result", err)
	}
	if err := json.Unmarshal([]byte(rejections), &x.EvaluationRejectionReasons); err != nil {
		return x, err
	}
	if err := json.Unmarshal([]byte(reasons), &x.ReadinessReasons); err != nil {
		return x, err
	}
	return x, nil
}

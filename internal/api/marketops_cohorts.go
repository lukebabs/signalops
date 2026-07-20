package api

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func marketOpsCohortRunResponse(x storage.MarketOpsIntelligenceCohortRunRecord) map[string]any {
	return map[string]any{"run_id": x.RunID, "tenant_id": x.TenantID, "app_id": x.AppID, "universe_group": x.UniverseGroup,
		"requested_symbols": x.RequestedSymbols, "resolved_symbols": x.ResolvedSymbols, "stages": x.Stages, "max_symbols": x.MaxSymbols,
		"dry_run": x.DryRun, "continue_on_error": x.ContinueOnError, "status": x.Status,
		"aggregate_metrics": json.RawMessage(jsonOrDefault(x.AggregateJSON, `{}`)), "errors": json.RawMessage(jsonOrDefault(x.ErrorsJSON, `[]`)),
		"actor": x.Actor, "session_start": x.SessionStart, "session_end": x.SessionEnd, "started_at": x.StartedAt,
		"completed_at": x.CompletedAt, "created_at": x.CreatedAt, "updated_at": x.UpdatedAt}
}

func marketOpsCohortResultResponse(x storage.MarketOpsIntelligenceCohortSymbolResultRecord) map[string]any {
	return map[string]any{"result_id": x.ResultID, "run_id": x.RunID, "tenant_id": x.TenantID, "universe_group": x.UniverseGroup,
		"symbol": x.Symbol, "asset_id": x.AssetID, "stage_status": json.RawMessage(jsonOrDefault(x.StageStatusJSON, `{}`)),
		"stage_errors": json.RawMessage(jsonOrDefault(x.StageErrorsJSON, `{}`)), "input_coverage": json.RawMessage(jsonOrDefault(x.InputCoverageJSON, `{}`)),
		"latest_market_state_id": x.LatestMarketStateID, "latest_state_date": x.LatestStateDate,
		"latest_state_schema_version": x.LatestStateSchemaVersion, "latest_state_quality": x.LatestStateQuality,
		"latest_state_completeness": x.LatestStateCompleteness, "required_feature_coverage": x.RequiredFeatureCoverage,
		"surface_coverage": x.SurfaceCoverage, "evaluation_count": x.EvaluationCount, "eligible_count": x.EligibleCount,
		"triggered_count": x.TriggeredCount, "evaluation_rejection_reasons": x.EvaluationRejectionReasons,
		"opportunity_count": x.OpportunityCount, "pending_outcome_count": x.PendingOutcomeCount, "matured_outcome_count": x.MaturedOutcomeCount,
		"proposal_status_counts":  json.RawMessage(jsonOrDefault(x.ProposalStatusCountsJSON, `{}`)),
		"exact_calibration_count": x.ExactCalibrationCount, "calibration_below_minimum": x.CalibrationBelowMinimum,
		"coverage_state": x.CoverageState, "evaluation_state": x.EvaluationState, "governance_state": x.GovernanceState,
		"calibration_state": x.CalibrationState, "outcome_state": x.OutcomeState, "rollout_status": x.RolloutStatus,
		"readiness_reasons": x.ReadinessReasons, "created_at": x.CreatedAt, "updated_at": x.UpdatedAt}
}

func marketOpsReadinessResponse(rows []storage.MarketOpsIntelligenceCohortSymbolResultRecord) map[string]any {
	dimensions := map[string]map[string]int{"coverage_state": {}, "evaluation_state": {}, "governance_state": {}, "calibration_state": {}, "outcome_state": {}, "rollout_status": {}}
	items := make([]map[string]any, 0, len(rows))
	var latest time.Time
	for _, x := range rows {
		dimensions["coverage_state"][x.CoverageState]++
		dimensions["evaluation_state"][x.EvaluationState]++
		dimensions["governance_state"][x.GovernanceState]++
		dimensions["calibration_state"][x.CalibrationState]++
		dimensions["outcome_state"][x.OutcomeState]++
		dimensions["rollout_status"][x.RolloutStatus]++
		if x.LatestStateDate != nil && x.LatestStateDate.After(latest) {
			latest = *x.LatestStateDate
		}
		items = append(items, marketOpsCohortResultResponse(x))
	}
	aggregate := map[string]any{"symbol_count": len(rows), "dimension_counts": dimensions, "production_ready_supported": false}
	if !latest.IsZero() {
		aggregate["latest_session_date"] = latest.Format("2006-01-02")
	}
	return map[string]any{"aggregate": aggregate, "symbols": items}
}

func parseCommaSymbols(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if symbol := strings.ToUpper(strings.TrimSpace(part)); symbol != "" {
			out = append(out, symbol)
		}
	}
	return uniqueSorted(out)
}

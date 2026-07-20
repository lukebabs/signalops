package main

import (
	"encoding/json"

	"github.com/lukebabs/signalops/internal/storage"
)

func cohortResultOutput(x storage.MarketOpsIntelligenceCohortSymbolResultRecord) map[string]any {
	return map[string]any{
		"result_id": x.ResultID, "run_id": x.RunID, "tenant_id": x.TenantID, "universe_group": x.UniverseGroup,
		"symbol": x.Symbol, "asset_id": x.AssetID, "stage_status": json.RawMessage(defaultJSON(x.StageStatusJSON, `{}`)),
		"stage_errors": json.RawMessage(defaultJSON(x.StageErrorsJSON, `{}`)), "input_coverage": json.RawMessage(defaultJSON(x.InputCoverageJSON, `{}`)),
		"latest_market_state_id": x.LatestMarketStateID, "latest_state_date": x.LatestStateDate, "latest_state_schema_version": x.LatestStateSchemaVersion,
		"latest_state_quality": x.LatestStateQuality, "latest_state_completeness": x.LatestStateCompleteness,
		"required_feature_coverage": x.RequiredFeatureCoverage, "surface_coverage": x.SurfaceCoverage,
		"evaluation_count": x.EvaluationCount, "eligible_count": x.EligibleCount, "triggered_count": x.TriggeredCount,
		"evaluation_rejection_reasons": x.EvaluationRejectionReasons, "opportunity_count": x.OpportunityCount,
		"pending_outcome_count": x.PendingOutcomeCount, "matured_outcome_count": x.MaturedOutcomeCount,
		"proposal_status_counts":  json.RawMessage(defaultJSON(x.ProposalStatusCountsJSON, `{}`)),
		"exact_calibration_count": x.ExactCalibrationCount, "calibration_below_minimum": x.CalibrationBelowMinimum,
		"coverage_state": x.CoverageState, "evaluation_state": x.EvaluationState, "governance_state": x.GovernanceState,
		"calibration_state": x.CalibrationState, "outcome_state": x.OutcomeState, "rollout_status": x.RolloutStatus,
		"readiness_reasons": x.ReadinessReasons,
	}
}
func defaultJSON(raw []byte, fallback string) []byte {
	if len(raw) == 0 || !json.Valid(raw) {
		return []byte(fallback)
	}
	return raw
}

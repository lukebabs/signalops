package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func registerMarketOpsAlgorithmEvaluationRoutes(mux *http.ServeMux, queryRepository storage.QueryRepository) {
	mux.HandleFunc("GET /v1/marketops/algorithm-evaluations", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireMarketOpsAlgorithmEvaluationRepository(w, queryRepository)
		if !ok {
			return
		}
		records, err := repo.ListMarketOpsAlgorithmEvaluationRuns(r.Context(), storage.MarketOpsAlgorithmEvaluationRunFilter{TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), AlgorithmID: strings.TrimSpace(r.URL.Query().Get("algorithm_id")), Status: strings.TrimSpace(r.URL.Query().Get("status")), Limit: queryLimit(r, 50)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list algorithm evaluations")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"algorithm_evaluations": algorithmEvaluationRunResponses(records)})
	})
	mux.HandleFunc("GET /v1/marketops/algorithm-evaluations/{run_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireMarketOpsAlgorithmEvaluationRepository(w, queryRepository)
		if !ok {
			return
		}
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "missing_query", "tenant_id is required")
			return
		}
		record, err := repo.GetMarketOpsAlgorithmEvaluationRun(r.Context(), tenantID, r.PathValue("run_id"))
		if err != nil {
			writeQueryError(w, err, "algorithm_evaluation_not_found", "algorithm evaluation not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"algorithm_evaluation": algorithmEvaluationRunResponse(record)})
	})
	mux.HandleFunc("GET /v1/marketops/algorithm-evaluations/{run_id}/results", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireMarketOpsAlgorithmEvaluationRepository(w, queryRepository)
		if !ok {
			return
		}
		writeAlgorithmEvaluationResults(w, r, repo, r.PathValue("run_id"))
	})
	mux.HandleFunc("GET /v1/marketops/algorithm-evaluations/{run_id}/outcomes", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireMarketOpsAlgorithmEvaluationRepository(w, queryRepository)
		if !ok {
			return
		}
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "missing_query", "tenant_id is required")
			return
		}
		horizon, ok := algorithmEvaluationHorizon(w, r)
		if !ok {
			return
		}
		records, err := repo.ListMarketOpsAlgorithmEvaluationOutcomes(r.Context(), storage.MarketOpsAlgorithmEvaluationOutcomeFilter{TenantID: tenantID, RunID: r.PathValue("run_id"), OutcomeStatus: strings.TrimSpace(r.URL.Query().Get("outcome_status")), HorizonSessions: horizon, Limit: queryLimit(r, 50)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list algorithm evaluation outcomes")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"algorithm_evaluation_outcomes": algorithmEvaluationOutcomeResponses(records)})
	})
	mux.HandleFunc("GET /v1/marketops/algorithm-evaluation-backfills", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireMarketOpsAlgorithmEvaluationRepository(w, queryRepository)
		if !ok {
			return
		}
		records, err := repo.ListMarketOpsAlgorithmEvaluationBackfillCampaigns(r.Context(), storage.MarketOpsAlgorithmEvaluationBackfillCampaignFilter{TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), Status: strings.TrimSpace(r.URL.Query().Get("status")), Limit: queryLimit(r, 50)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list algorithm evaluation backfill campaigns")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"algorithm_evaluation_backfill_campaigns": algorithmEvaluationBackfillCampaignResponses(records)})
	})
	mux.HandleFunc("GET /v1/marketops/algorithm-evaluation-backfills/{campaign_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireMarketOpsAlgorithmEvaluationRepository(w, queryRepository)
		if !ok {
			return
		}
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "missing_query", "tenant_id is required")
			return
		}
		record, err := repo.GetMarketOpsAlgorithmEvaluationBackfillCampaign(r.Context(), tenantID, r.PathValue("campaign_id"))
		if err != nil {
			writeQueryError(w, err, "algorithm_evaluation_backfill_not_found", "algorithm evaluation backfill campaign not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"algorithm_evaluation_backfill_campaign": algorithmEvaluationBackfillCampaignResponse(record)})
	})
}

func requireMarketOpsAlgorithmEvaluationRepository(w http.ResponseWriter, queryRepository storage.QueryRepository) (storage.MarketOpsAlgorithmEvaluationRepository, bool) {
	if queryRepository == nil {
		writeError(w, http.StatusServiceUnavailable, "storage_unavailable", "storage is not configured")
		return nil, false
	}
	repo, ok := any(queryRepository).(storage.MarketOpsAlgorithmEvaluationRepository)
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "storage_unavailable", "algorithm evaluation storage is not configured")
		return nil, false
	}
	return repo, true
}
func writeAlgorithmEvaluationResults(w http.ResponseWriter, r *http.Request, repo storage.MarketOpsAlgorithmEvaluationRepository, runID string) {
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "missing_query", "tenant_id is required")
		return
	}
	records, err := repo.ListMarketOpsAlgorithmEvaluationResults(r.Context(), storage.MarketOpsAlgorithmEvaluationResultFilter{TenantID: tenantID, RunID: runID, AlgorithmID: strings.TrimSpace(r.URL.Query().Get("algorithm_id")), Symbol: strings.TrimSpace(r.URL.Query().Get("symbol")), EvaluationMode: strings.TrimSpace(r.URL.Query().Get("evaluation_mode")), Limit: queryLimit(r, 50)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query_failed", "failed to list algorithm evaluation results")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"algorithm_evaluation_results": algorithmEvaluationResultResponses(records)})
}
func algorithmEvaluationHorizon(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("horizon_sessions"))
	if raw == "" {
		return 0, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || (value != 1 && value != 5 && value != 10 && value != 20) {
		writeError(w, http.StatusBadRequest, "invalid_horizon", "horizon_sessions must be one of 1, 5, 10, or 20")
		return 0, false
	}
	return value, true
}
func jsonValue(raw []byte) any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return map[string]any{}
	}
	return value
}
func algorithmEvaluationRunResponses(records []storage.MarketOpsAlgorithmEvaluationRunRecord) []map[string]any {
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		out = append(out, algorithmEvaluationRunResponse(record))
	}
	return out
}
func algorithmEvaluationRunResponse(record storage.MarketOpsAlgorithmEvaluationRunRecord) map[string]any {
	return map[string]any{"run_id": record.RunID, "tenant_id": record.TenantID, "app_id": record.AppID, "universe_group": record.UniverseGroup, "algorithm_ids": record.AlgorithmIDs, "modes": record.Modes, "window_start": record.WindowStart, "window_end": record.WindowEnd, "as_of_date": record.AsOfDate, "status": record.Status, "parameters": jsonValue(record.ParametersJSON), "coverage": jsonValue(record.CoverageJSON), "metrics": jsonValue(record.MetricsJSON), "error_message": record.ErrorMessage, "requested_by": record.RequestedBy, "started_at": record.StartedAt, "completed_at": record.CompletedAt, "created_at": record.CreatedAt, "updated_at": record.UpdatedAt}
}
func algorithmEvaluationResultResponses(records []storage.MarketOpsAlgorithmEvaluationResultRecord) []map[string]any {
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		out = append(out, map[string]any{"evaluation_result_id": record.EvaluationResultID, "run_id": record.RunID, "tenant_id": record.TenantID, "algorithm_id": record.AlgorithmID, "algorithm_version": record.AlgorithmVersion, "evaluation_mode": record.EvaluationMode, "evaluation_profile": record.EvaluationProfile, "result_type": record.ResultType, "symbol": record.Symbol, "observation_session_date": record.ObservationSessionDate, "score": record.Score, "confidence": record.Confidence, "severity": record.Severity, "direction": record.Direction, "result_payload": jsonValue(record.ResultPayloadJSON), "input_provenance": jsonValue(record.InputProvenanceJSON), "source_event_ids": record.SourceEventIDs, "feature_value_ids": record.FeatureValueIDs, "created_at": record.CreatedAt})
	}
	return out
}
func algorithmEvaluationOutcomeResponses(records []storage.MarketOpsAlgorithmEvaluationOutcomeRecord) []map[string]any {
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		out = append(out, map[string]any{"evaluation_outcome_id": record.EvaluationOutcomeID, "run_id": record.RunID, "evaluation_result_id": record.EvaluationResultID, "tenant_id": record.TenantID, "horizon_sessions": record.HorizonSessions, "outcome_status": record.OutcomeStatus, "matured_session_date": record.MaturedSessionDate, "forward_return": record.ForwardReturn, "absolute_forward_return": record.AbsoluteForwardReturn, "max_favorable_excursion": record.MaxFavorableExcursion, "max_adverse_excursion": record.MaxAdverseExcursion, "maximum_drawdown": record.MaximumDrawdown, "realized_vol_change": record.RealizedVolChange, "directional_hit": record.DirectionalHit, "threshold_hit": record.ThresholdHit, "outcome_event_ids": record.OutcomeEventIDs, "outcome_payload": jsonValue(record.OutcomePayloadJSON), "created_at": record.CreatedAt, "updated_at": record.UpdatedAt})
	}
	return out
}
func algorithmEvaluationBackfillCampaignResponses(records []storage.MarketOpsAlgorithmEvaluationBackfillCampaignRecord) []map[string]any {
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		out = append(out, algorithmEvaluationBackfillCampaignResponse(record))
	}
	return out
}
func algorithmEvaluationBackfillCampaignResponse(record storage.MarketOpsAlgorithmEvaluationBackfillCampaignRecord) map[string]any {
	return map[string]any{"campaign_id": record.CampaignID, "tenant_id": record.TenantID, "universe_group": record.UniverseGroup, "window_start": record.WindowStart, "window_end": record.WindowEnd, "status": record.Status, "parameters": jsonValue(record.ParametersJSON), "coverage": jsonValue(record.CoverageJSON), "child_run_ids": record.ChildRunIDs, "error_message": record.ErrorMessage, "requested_by": record.RequestedBy, "started_at": record.StartedAt, "completed_at": record.CompletedAt, "created_at": record.CreatedAt, "updated_at": record.UpdatedAt}
}

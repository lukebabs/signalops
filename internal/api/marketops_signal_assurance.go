package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func registerMarketOpsSignalAssuranceRoutes(mux *http.ServeMux, cfg RouterConfig) {
	query, ok := cfg.QueryRepository.(storage.SignalAssuranceQueryRepository)
	if !ok {
		return
	}
	mux.HandleFunc("GET /v1/marketops/signal-assurance/readiness", func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := requireRequestTenant(w, r, r.URL.Query().Get("tenant_id"))
		if !ok {
			return
		}
		record, err := query.GetSignalAssuranceOperationalReadiness(r.Context(), tenantID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to load signal assurance readiness")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"readiness": signalAssuranceReadinessResponse(record)})
	})
	mux.HandleFunc("GET /v1/marketops/signal-assurance/assertions", func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := requireRequestTenant(w, r, r.URL.Query().Get("tenant_id"))
		if !ok {
			return
		}
		records, err := query.ListSignalAssuranceAssertions(r.Context(), storage.SignalAssuranceAssertionFilter{TenantID: tenantID, State: strings.TrimSpace(r.URL.Query().Get("state")), EvaluationMode: strings.TrimSpace(r.URL.Query().Get("evaluation_mode")), Symbol: strings.TrimSpace(r.URL.Query().Get("symbol")), Limit: queryLimit(r, 50)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list signal assurance assertions")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"assertions": assertionResponses(records)})
	})
	mux.HandleFunc("GET /v1/marketops/signal-assurance/assertions/{assertion_id}", func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := requireRequestTenant(w, r, r.URL.Query().Get("tenant_id"))
		if !ok {
			return
		}
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "missing_query", "tenant_id is required")
			return
		}
		record, err := query.GetSignalAssuranceAssertion(r.Context(), tenantID, r.PathValue("assertion_id"))
		if err != nil {
			writeQueryError(w, err, "assertion_not_found", "signal assurance assertion not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"assertion": assertionResponse(record)})
	})
	mux.HandleFunc("GET /v1/marketops/signal-assurance/assertions/{assertion_id}/evaluations", func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := requireRequestTenant(w, r, r.URL.Query().Get("tenant_id"))
		if !ok {
			return
		}
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "missing_query", "tenant_id is required")
			return
		}
		if _, err := query.GetSignalAssuranceAssertion(r.Context(), tenantID, r.PathValue("assertion_id")); err != nil {
			writeQueryError(w, err, "assertion_not_found", "signal assurance assertion not found")
			return
		}
		records, err := query.ListSignalAssuranceEvaluations(r.Context(), storage.SignalAssuranceEvaluationFilter{AssertionID: r.PathValue("assertion_id"), Limit: queryLimit(r, 50)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list signal assurance evaluations")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"evaluations": evaluationResponses(records)})
	})
}

type signalAssuranceReadinessDTO struct {
	TenantID                            string   `json:"tenant_id"`
	ReadinessState                      string   `json:"readiness_state"`
	ReadinessReasons                    []string `json:"readiness_reasons"`
	ActiveContractCount                 int      `json:"active_contract_count"`
	LiveContractCount                   int      `json:"live_contract_count"`
	ResearchContractCount               int      `json:"research_contract_count"`
	AssertionCount                      int      `json:"assertion_count"`
	LiveAssertionCount                  int      `json:"live_assertion_count"`
	ResearchAssertionCount              int      `json:"research_assertion_count"`
	SucceededMaterializationCount       int      `json:"succeeded_materialization_count"`
	DirectionalMaterializationCount     int      `json:"directional_materialization_count"`
	NonDirectionalMaterializationCount  int      `json:"non_directional_materialization_count"`
	ContractCoveredMaterializationCount int      `json:"contract_covered_materialization_count"`
	ContractBlockedMaterializationCount int      `json:"contract_blocked_materialization_count"`
	LatestMaterializationAt             string   `json:"latest_materialization_at,omitempty"`
	LatestAssertionAt                   string   `json:"latest_assertion_at,omitempty"`
}

func signalAssuranceReadinessResponse(x storage.SignalAssuranceOperationalReadinessRecord) signalAssuranceReadinessDTO {
	response := signalAssuranceReadinessDTO{TenantID: x.TenantID, ReadinessState: x.ReadinessState, ReadinessReasons: x.ReadinessReasons, ActiveContractCount: x.ActiveContractCount, LiveContractCount: x.LiveContractCount, ResearchContractCount: x.ResearchContractCount, AssertionCount: x.AssertionCount, LiveAssertionCount: x.LiveAssertionCount, ResearchAssertionCount: x.ResearchAssertionCount, SucceededMaterializationCount: x.SucceededMaterializationCount, DirectionalMaterializationCount: x.DirectionalMaterializationCount, NonDirectionalMaterializationCount: x.NonDirectionalMaterializationCount, ContractCoveredMaterializationCount: x.ContractCoveredMaterializationCount, ContractBlockedMaterializationCount: x.ContractBlockedMaterializationCount}
	if x.LatestMaterializationAt != nil {
		response.LatestMaterializationAt = x.LatestMaterializationAt.UTC().Format("2006-01-02T15:04:05Z")
	}
	if x.LatestAssertionAt != nil {
		response.LatestAssertionAt = x.LatestAssertionAt.UTC().Format("2006-01-02T15:04:05Z")
	}
	return response
}

type signalAssuranceAssertionDTO struct {
	AssertionID        string          `json:"assertion_id"`
	TenantID           string          `json:"tenant_id"`
	AssetID            string          `json:"asset_id"`
	Symbol             string          `json:"symbol"`
	SignalID           string          `json:"signal_id"`
	SignalType         string          `json:"signal_type"`
	Direction          string          `json:"direction"`
	State              string          `json:"state"`
	EvaluationMode     string          `json:"evaluation_mode"`
	EvaluationRunID    string          `json:"evaluation_run_id,omitempty"`
	ContractID         string          `json:"validation_contract_id"`
	ContractVersion    string          `json:"validation_contract_version"`
	Confidence         *float64        `json:"confidence,omitempty"`
	BaselineSnapshot   json.RawMessage `json:"baseline_snapshot"`
	BaselineProvenance json.RawMessage `json:"baseline_provenance"`
	ConfirmedAt        string          `json:"confirmed_at"`
	TransitionSequence int             `json:"transition_sequence"`
}

func assertionResponse(x storage.SignalAssertionRecord) signalAssuranceAssertionDTO {
	return signalAssuranceAssertionDTO{AssertionID: x.AssertionID, TenantID: x.TenantID, AssetID: x.AssetID, Symbol: x.Symbol, SignalID: x.SignalID, SignalType: x.SignalType, Direction: x.SignalDirection, State: x.State, EvaluationMode: x.EvaluationMode, EvaluationRunID: x.EvaluationRunID, ContractID: x.ValidationContractID, ContractVersion: x.ValidationContractVersion, Confidence: x.Confidence, BaselineSnapshot: json.RawMessage(jsonOrDefault(x.BaselineSnapshotJSON, "{}")), BaselineProvenance: json.RawMessage(jsonOrDefault(x.BaselineProvenanceJSON, "{}")), ConfirmedAt: x.ConfirmedAt.UTC().Format("2006-01-02T15:04:05Z"), TransitionSequence: x.TransitionSequence}
}
func assertionResponses(records []storage.SignalAssertionRecord) []signalAssuranceAssertionDTO {
	out := make([]signalAssuranceAssertionDTO, 0, len(records))
	for _, x := range records {
		out = append(out, assertionResponse(x))
	}
	return out
}

type signalAssuranceEvaluationDTO struct {
	EvaluationID                string   `json:"evaluation_id"`
	AssertionID                 string   `json:"assertion_id"`
	SessionDate                 string   `json:"evaluation_session_date"`
	InputCompleteness           string   `json:"input_completeness"`
	TradingDaysActive           int      `json:"trading_days_active"`
	AbsoluteReturn              *float64 `json:"absolute_return,omitempty"`
	BenchmarkRelativeReturn     *float64 `json:"benchmark_relative_return,omitempty"`
	MFE                         *float64 `json:"mfe,omitempty"`
	MAE                         *float64 `json:"mae,omitempty"`
	MaterializationConditionMet bool     `json:"materialization_condition_met"`
	InvalidationConditionMet    bool     `json:"invalidation_condition_met"`
	EvaluationVersion           string   `json:"evaluation_version"`
}

func evaluationResponses(records []storage.SignalAssertionEvaluationRecord) []signalAssuranceEvaluationDTO {
	out := make([]signalAssuranceEvaluationDTO, 0, len(records))
	for _, x := range records {
		out = append(out, signalAssuranceEvaluationDTO{EvaluationID: x.EvaluationID, AssertionID: x.AssertionID, SessionDate: x.EvaluationSessionDate.Format("2006-01-02"), InputCompleteness: x.InputCompleteness, TradingDaysActive: x.TradingDaysActive, AbsoluteReturn: x.AbsoluteReturn, BenchmarkRelativeReturn: x.BenchmarkRelativeReturn, MFE: x.MFE, MAE: x.MAE, MaterializationConditionMet: x.MaterializationConditionMet, InvalidationConditionMet: x.InvalidationConditionMet, EvaluationVersion: x.EvaluationVersion})
	}
	return out
}

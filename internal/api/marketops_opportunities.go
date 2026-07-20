package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func registerMarketOpsOpportunityRoutes(mux *http.ServeMux, queryRepository storage.QueryRepository) {
	mux.HandleFunc("GET /v1/marketops/opportunities", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, queryRepository)
		if !ok {
			return
		}
		start, end, ok := marketOpsSessionRange(w, r)
		if !ok {
			return
		}
		researchOnly, ok := optionalBoolQuery(w, r, "research_only")
		if !ok {
			return
		}
		filter := storage.MarketOpsOpportunityFilter{
			TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), AppID: strings.TrimSpace(r.URL.Query().Get("app_id")),
			OpportunityID: strings.TrimSpace(r.URL.Query().Get("opportunity_id")), AssetID: strings.TrimSpace(r.URL.Query().Get("asset_id")),
			Symbol: strings.TrimSpace(r.URL.Query().Get("symbol")), Direction: strings.TrimSpace(r.URL.Query().Get("direction")),
			Horizon: strings.TrimSpace(r.URL.Query().Get("horizon")), LifecycleStatus: strings.TrimSpace(r.URL.Query().Get("lifecycle_status")),
			ResearchOnly: researchOnly, SessionStart: start, SessionEnd: end, Limit: queryLimit(r, 50),
		}
		records, err := repo.ListMarketOpsOpportunities(r.Context(), filter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps opportunities")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"opportunities": opportunityResponses(records)})
	})
	mux.HandleFunc("GET /v1/marketops/opportunities/{opportunity_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, queryRepository)
		if !ok {
			return
		}
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "missing_query", "tenant_id is required")
			return
		}
		record, err := repo.GetMarketOpsOpportunity(r.Context(), tenantID, r.PathValue("opportunity_id"))
		if err != nil {
			writeQueryError(w, err, "opportunity_not_found", "MarketOps opportunity not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"opportunity": opportunityResponse(record)})
	})
}

type marketOpsOpportunityDTO struct {
	OpportunityID            string          `json:"opportunity_id"`
	TenantID                 string          `json:"tenant_id"`
	AppID                    string          `json:"app_id"`
	AssetID                  string          `json:"asset_id"`
	Symbol                   string          `json:"symbol"`
	OpenedSessionDate        time.Time       `json:"opened_session_date"`
	LastEvaluatedDate        time.Time       `json:"last_evaluated_date"`
	Direction                string          `json:"direction"`
	Horizon                  string          `json:"horizon"`
	LifecycleStatus          string          `json:"lifecycle_status"`
	OpportunityScore         float64         `json:"opportunity_score"`
	ConfidenceScore          float64         `json:"confidence_score"`
	DomainDiversityScore     float64         `json:"domain_diversity_score"`
	ConflictScore            float64         `json:"conflict_score"`
	HypothesisEvaluationIDs  []string        `json:"hypothesis_evaluation_ids"`
	ConflictingEvaluationIDs []string        `json:"conflicting_evaluation_ids"`
	SignalIDs                []string        `json:"signal_ids"`
	SupportingEvidenceIDs    []string        `json:"supporting_evidence_ids"`
	InvalidatingEvidenceIDs  []string        `json:"invalidating_evidence_ids"`
	Summary                  string          `json:"summary"`
	OpportunityPayload       json.RawMessage `json:"opportunity_payload"`
	Version                  int             `json:"version"`
	ResearchOnly             bool            `json:"research_only"`
	BuildRunID               string          `json:"build_run_id"`
	DeterministicKey         string          `json:"deterministic_key"`
	CreatedAt                time.Time       `json:"created_at"`
	UpdatedAt                time.Time       `json:"updated_at"`
}

func opportunityResponse(record storage.MarketOpsOpportunityRecord) marketOpsOpportunityDTO {
	return marketOpsOpportunityDTO{
		OpportunityID: record.OpportunityID, TenantID: record.TenantID, AppID: record.AppID,
		AssetID: record.AssetID, Symbol: record.Symbol, OpenedSessionDate: record.OpenedSessionDate,
		LastEvaluatedDate: record.LastEvaluatedDate, Direction: record.Direction, Horizon: record.Horizon,
		LifecycleStatus: record.LifecycleStatus, OpportunityScore: record.OpportunityScore,
		ConfidenceScore: record.ConfidenceScore, DomainDiversityScore: record.DomainDiversityScore,
		ConflictScore: record.ConflictScore, HypothesisEvaluationIDs: nonNilStrings(record.HypothesisEvaluationIDs),
		ConflictingEvaluationIDs: nonNilStrings(record.ConflictingEvaluationIDs), SignalIDs: nonNilStrings(record.SignalIDs),
		SupportingEvidenceIDs: nonNilStrings(record.SupportingEvidenceIDs), InvalidatingEvidenceIDs: nonNilStrings(record.InvalidatingEvidenceIDs),
		Summary: record.Summary, OpportunityPayload: json.RawMessage(jsonOrDefault(record.OpportunityPayloadJSON, `{}`)),
		Version: record.Version, ResearchOnly: record.ResearchOnly, BuildRunID: record.BuildRunID,
		DeterministicKey: record.DeterministicKey, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt,
	}
}

func opportunityResponses(records []storage.MarketOpsOpportunityRecord) []marketOpsOpportunityDTO {
	out := make([]marketOpsOpportunityDTO, 0, len(records))
	for _, record := range records {
		out = append(out, opportunityResponse(record))
	}
	return out
}

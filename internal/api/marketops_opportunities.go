package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	marketopsstate "github.com/lukebabs/signalops/internal/marketops/state"
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
	mux.HandleFunc("GET /v1/marketops/opportunities/{opportunity_id}/dispositions", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, queryRepository)
		if !ok {
			return
		}
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "missing_query", "tenant_id is required")
			return
		}
		records, err := repo.ListMarketOpsOpportunityDispositions(r.Context(), storage.MarketOpsOpportunityDispositionFilter{TenantID: tenantID, OpportunityID: r.PathValue("opportunity_id"), Disposition: strings.TrimSpace(r.URL.Query().Get("disposition")), Limit: queryLimit(r, 50)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list opportunity dispositions")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"opportunity_dispositions": opportunityDispositionResponses(records)})
	})
	mux.HandleFunc("POST /v1/marketops/opportunities/{opportunity_id}/dispositions", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, queryRepository)
		if !ok {
			return
		}
		var req marketOpsOpportunityDispositionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		tenantID := strings.TrimSpace(req.TenantID)
		if tenantID == "" {
			tenantID = strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		}
		disposition := strings.TrimSpace(req.Disposition)
		if !validOpportunityDisposition(disposition) {
			writeError(w, http.StatusBadRequest, "invalid_disposition", "opportunity disposition is invalid")
			return
		}
		metadata := []byte(req.Metadata)
		if len(metadata) == 0 {
			metadata = []byte(`{}`)
		}
		var metadataObject map[string]any
		if err := json.Unmarshal(metadata, &metadataObject); err != nil || metadataObject == nil {
			writeError(w, http.StatusBadRequest, "invalid_metadata", "metadata must be a JSON object")
			return
		}
		actor := replayActor(r, req.Actor)
		createdAt := time.Now().UTC()
		identity, err := marketopsstate.NewIdentity(marketopsstate.IdentityOpportunityDisposition, tenantID, r.PathValue("opportunity_id"), disposition, actor, createdAt.Format(time.RFC3339Nano))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_disposition", err.Error())
			return
		}
		record := storage.MarketOpsOpportunityDispositionRecord{DispositionID: identity.ID, TenantID: tenantID, OpportunityID: r.PathValue("opportunity_id"), Disposition: disposition, Actor: actor, Note: strings.TrimSpace(req.Note), MetadataJSON: metadata, CreatedAt: createdAt}
		if err := repo.InsertMarketOpsOpportunityDisposition(r.Context(), record); err != nil {
			writeQueryError(w, err, "opportunity_not_found", "MarketOps opportunity not found")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"opportunity_disposition": opportunityDispositionResponse(record)})
	})
}

type marketOpsOpportunityDispositionRequest struct {
	TenantID    string          `json:"tenant_id"`
	Disposition string          `json:"disposition"`
	Actor       string          `json:"actor"`
	Note        string          `json:"note"`
	Metadata    json.RawMessage `json:"metadata"`
}

type marketOpsOpportunityDispositionDTO struct {
	DispositionID string          `json:"disposition_id"`
	TenantID      string          `json:"tenant_id"`
	OpportunityID string          `json:"opportunity_id"`
	Disposition   string          `json:"disposition"`
	Actor         string          `json:"actor"`
	Note          string          `json:"note"`
	Metadata      json.RawMessage `json:"metadata"`
	CreatedAt     time.Time       `json:"created_at"`
}

func validOpportunityDisposition(value string) bool {
	switch value {
	case storage.MarketOpsOpportunityDispositionWatch, storage.MarketOpsOpportunityDispositionAdvance, storage.MarketOpsOpportunityDispositionNeedsMoreEvidence, storage.MarketOpsOpportunityDispositionDismiss, storage.MarketOpsOpportunityDispositionResolved:
		return true
	default:
		return false
	}
}

func opportunityDispositionResponse(record storage.MarketOpsOpportunityDispositionRecord) marketOpsOpportunityDispositionDTO {
	return marketOpsOpportunityDispositionDTO{DispositionID: record.DispositionID, TenantID: record.TenantID, OpportunityID: record.OpportunityID, Disposition: record.Disposition, Actor: record.Actor, Note: record.Note, Metadata: json.RawMessage(jsonOrDefault(record.MetadataJSON, `{}`)), CreatedAt: record.CreatedAt}
}

func opportunityDispositionResponses(records []storage.MarketOpsOpportunityDispositionRecord) []marketOpsOpportunityDispositionDTO {
	out := make([]marketOpsOpportunityDispositionDTO, 0, len(records))
	for _, record := range records {
		out = append(out, opportunityDispositionResponse(record))
	}
	return out
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

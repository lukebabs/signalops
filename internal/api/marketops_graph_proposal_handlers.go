package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func marketOpsGraphProposalListHandler(queryRepository storage.QueryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, queryRepository)
		if !ok {
			return
		}
		records, err := repo.ListMarketOpsDSMGraphProposals(r.Context(), storage.MarketOpsDSMGraphProposalFilter{
			TenantID:         strings.TrimSpace(r.URL.Query().Get("tenant_id")),
			AppID:            strings.TrimSpace(r.URL.Query().Get("app_id")),
			Domain:           strings.TrimSpace(r.URL.Query().Get("domain")),
			UseCase:          strings.TrimSpace(r.URL.Query().Get("use_case")),
			SubjectSymbol:    strings.TrimSpace(r.URL.Query().Get("subject_symbol")),
			CandidateType:    strings.TrimSpace(r.URL.Query().Get("candidate_type")),
			Status:           strings.TrimSpace(r.URL.Query().Get("status")),
			ProposalSource:   strings.TrimSpace(r.URL.Query().Get("proposal_source")),
			SourceRecordType: strings.TrimSpace(r.URL.Query().Get("source_record_type")),
			SourceRecordID:   strings.TrimSpace(r.URL.Query().Get("source_record_id")),
			Limit:            queryLimit(r, 50),
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps graph proposals")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"graph_proposals": marketOpsDSMGraphProposalResponses(records)})
	}
}

func marketOpsGraphProposalGetHandler(queryRepository storage.QueryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, queryRepository)
		if !ok {
			return
		}
		record, err := repo.GetMarketOpsDSMGraphProposal(r.Context(), r.PathValue("proposal_id"))
		if err != nil {
			writeQueryError(w, err, "graph_proposal_not_found", "MarketOps graph proposal not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"graph_proposal": marketOpsDSMGraphProposalResponse(record)})
	}
}

func marketOpsGraphProposalDecisionHandler(queryRepository storage.QueryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, queryRepository)
		if !ok {
			return
		}
		var req graphProposalDecisionRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, defaultMaxRawEventBytes))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		status := strings.TrimSpace(req.Status)
		if status == "" {
			writeError(w, http.StatusBadRequest, "invalid_status", "status is required")
			return
		}
		record, err := repo.MutateMarketOpsDSMGraphProposal(r.Context(), storage.MarketOpsDSMGraphProposalMutation{
			ProposalID:   r.PathValue("proposal_id"),
			Status:       status,
			ReviewedBy:   lifecycleActor(r, req.Actor),
			DecisionNote: strings.TrimSpace(req.Note),
			DecidedAt:    time.Now().UTC(),
		})
		if err != nil {
			if strings.Contains(err.Error(), "status") {
				writeError(w, http.StatusBadRequest, "invalid_status", "graph proposal status is invalid")
				return
			}
			writeQueryError(w, err, "graph_proposal_not_found", "MarketOps graph proposal not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"graph_proposal": marketOpsDSMGraphProposalResponse(record)})
	}
}

package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func registerCyberOpsIntegrityRoutes(mux *http.ServeMux, cfg RouterConfig, repository storage.CyberOpsConnectRepository) {
	mux.HandleFunc("GET /v1/cyberops/integrity-failures", func(w http.ResponseWriter, r *http.Request) {
		if repository == nil {
			writeError(w, http.StatusServiceUnavailable, "storage_unavailable", "cyberops storage is unavailable")
			return
		}
		tenant, ok := cyberOpsTenant(r, cfg.Auth.Enabled)
		if !ok {
			writeError(w, http.StatusForbidden, "missing_tenant_claim", "tenant scope is required")
			return
		}
		filter := storage.CyberOpsIntegrityFailureFilter{TenantID: tenant, ResolutionStatus: strings.TrimSpace(r.URL.Query().Get("resolution_status")), Limit: queryLimit(r, 50)}
		records, err := repository.ListCyberOpsIntegrityFailures(r.Context(), filter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list cyberops integrity failures")
			return
		}
		items := make([]map[string]any, 0, len(records))
		for _, record := range records {
			items = append(items, cyberOpsIntegrityFailureResponse(record))
		}
		writeJSON(w, http.StatusOK, map[string]any{"integrity_failures": items})
	})

	mux.HandleFunc("POST /v1/cyberops/integrity-failures/{failure_id}/resolve", func(w http.ResponseWriter, r *http.Request) {
		if repository == nil {
			writeError(w, http.StatusServiceUnavailable, "storage_unavailable", "cyberops storage is unavailable")
			return
		}
		tenant, ok := cyberOpsTenant(r, cfg.Auth.Enabled)
		if !ok {
			writeError(w, http.StatusForbidden, "missing_tenant_claim", "tenant scope is required")
			return
		}
		status, reason, err := readCyberOpsIntegrityResolutionRequest(w, r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_integrity_resolution", err.Error())
			return
		}
		record, err := repository.ResolveCyberOpsIntegrityFailure(r.Context(), tenant, strings.TrimSpace(r.PathValue("failure_id")), status, lifecycleActor(r, ""), reason, time.Now().UTC())
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "cyberops_integrity_failure_not_found", "cyberops integrity failure not found")
			return
		}
		if errors.Is(err, storage.ErrConflict) {
			writeError(w, http.StatusConflict, "integrity_failure_already_resolved", err.Error())
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "resolution_failed", "failed to resolve cyberops integrity failure")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"integrity_failure": cyberOpsIntegrityFailureResponse(record)})
	})
}

func readCyberOpsIntegrityResolutionRequest(w http.ResponseWriter, r *http.Request) (string, string, error) {
	body, _, err := readJSONObject(w, r, 64<<10)
	if err != nil {
		return "", "", err
	}
	var request struct {
		ResolutionStatus string `json:"resolution_status"`
		Reason           string `json:"reason"`
	}
	if err := json.Unmarshal(body, &request); err != nil {
		return "", "", err
	}
	status := strings.TrimSpace(request.ResolutionStatus)
	switch status {
	case "resolved_false_positive", "resolved_test_fixture", "resolved_confirmed_conflict":
	default:
		return "", "", errors.New("resolution_status is not supported")
	}
	reason := strings.TrimSpace(request.Reason)
	if len(reason) < 3 || len(reason) > 1024 {
		return "", "", errors.New("reason must be between 3 and 1024 characters")
	}
	return status, reason, nil
}

func cyberOpsIntegrityFailureResponse(record storage.CyberOpsIntegrityFailureRecord) map[string]any {
	return map[string]any{"failure_id": record.FailureID, "tenant_id": record.TenantID, "connect_ingress_event_id": record.ConnectIngressEventID, "existing_event_id": record.ExistingEventID, "existing_payload_hash": record.ExistingPayloadHash, "incoming_payload_hash": record.IncomingPayloadHash, "first_seen_at": record.FirstSeenAt, "last_seen_at": record.LastSeenAt, "occurrence_count": record.OccurrenceCount, "resolution_status": record.ResolutionStatus, "resolved_at": record.ResolvedAt, "resolution_actor": record.ResolutionActor, "resolution_reason": record.ResolutionReason}
}

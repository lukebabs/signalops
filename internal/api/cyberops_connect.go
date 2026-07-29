package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func registerCyberOpsConnectRoutes(mux *http.ServeMux, cfg RouterConfig) {
	repository := cfg.CyberOpsConnectRepository
	mux.HandleFunc("GET /v1/cyberops/events", func(w http.ResponseWriter, r *http.Request) {
		if repository == nil {
			writeError(w, http.StatusServiceUnavailable, "storage_unavailable", "cyberops storage is unavailable")
			return
		}
		tenant, ok := cyberOpsTenant(r, cfg.Auth.Enabled)
		if !ok {
			writeError(w, http.StatusForbidden, "missing_tenant_claim", "tenant scope is required")
			return
		}
		from, to, err := cyberOpsRange(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_range", err.Error())
			return
		}
		search := strings.TrimSpace(r.URL.Query().Get("search"))
		if len(search) > 256 {
			writeError(w, http.StatusBadRequest, "search_too_long", "search must be at most 256 characters")
			return
		}
		if search != "" && (from.IsZero() || to.IsZero() || to.Sub(from) > 31*24*time.Hour) {
			writeError(w, http.StatusBadRequest, "search_range_required", "search requires a range of 31 days or less")
			return
		}
		filter := storage.CyberOpsConnectRawFilter{TenantID: tenant, From: from, To: to, Hostname: strings.TrimSpace(r.URL.Query().Get("hostname")), Application: strings.TrimSpace(r.URL.Query().Get("application")), ProducerID: strings.TrimSpace(r.URL.Query().Get("producer")), ConnectorID: strings.TrimSpace(r.URL.Query().Get("connector")), EventType: strings.TrimSpace(r.URL.Query().Get("event_type")), Search: search, Limit: queryLimit(r, 50)}
		filter.Severity = optionalInt(r.URL.Query().Get("severity"))
		filter.Facility = optionalInt(r.URL.Query().Get("facility"))
		records, err := repository.ListCyberOpsConnectRaw(r.Context(), filter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list cyberops events")
			return
		}
		items := make([]map[string]any, 0, len(records))
		for _, record := range records {
			items = append(items, cyberOpsEventResponse(record, false))
		}
		writeJSON(w, http.StatusOK, map[string]any{"events": items})
	})
	mux.HandleFunc("GET /v1/cyberops/events/{connect_ingress_event_id}", func(w http.ResponseWriter, r *http.Request) {
		if repository == nil {
			writeError(w, http.StatusServiceUnavailable, "storage_unavailable", "cyberops storage is unavailable")
			return
		}
		tenant, ok := cyberOpsTenant(r, cfg.Auth.Enabled)
		if !ok {
			writeError(w, http.StatusForbidden, "missing_tenant_claim", "tenant scope is required")
			return
		}
		record, err := repository.GetCyberOpsConnectRaw(r.Context(), tenant, strings.TrimSpace(r.PathValue("connect_ingress_event_id")))
		if err == storage.ErrNotFound {
			writeError(w, http.StatusNotFound, "cyberops_event_not_found", "cyberops event not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to get cyberops event")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"event": cyberOpsEventResponse(record, true), "derived_artifact_count": 0})
	})
	registerCyberOpsIntegrityRoutes(mux, cfg, repository)
}
func cyberOpsTenant(r *http.Request, authEnabled bool) (string, bool) {
	if principal, ok := principalFromContext(r.Context()); ok && strings.TrimSpace(principal.TenantID) != "" {
		return principal.TenantID, true
	}
	if !authEnabled {
		tenant := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		return tenant, tenant != ""
	}
	return "", false
}
func cyberOpsRange(r *http.Request) (time.Time, time.Time, error) {
	parse := func(key string) (time.Time, error) {
		value := strings.TrimSpace(r.URL.Query().Get(key))
		if value == "" {
			return time.Time{}, nil
		}
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return time.Time{}, err
		}
		return parsed.UTC(), nil
	}
	from, err := parse("from")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	to, err := parse("to")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if !from.IsZero() && !to.IsZero() && to.Before(from) {
		return time.Time{}, time.Time{}, &rangeError{}
	}
	return from, to, nil
}

type rangeError struct{}

func (*rangeError) Error() string { return "to must not precede from" }
func optionalInt(value string) *int {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return nil
	}
	return &parsed
}
func cyberOpsEventResponse(record storage.CyberOpsConnectRawRecord, detail bool) map[string]any {
	item := map[string]any{"connect_ingress_event_id": record.ConnectIngressEventID, "event_id": record.EventID, "event_type": record.EventType, "source_id": record.SourceID, "occurred_at": record.OccurredAt, "ingested_at": record.IngestedAt, "hostname": record.Hostname, "application": record.Application, "severity": record.Severity, "facility": record.Facility, "payload_hash": record.PayloadHash}
	if detail {
		item["message"] = record.Message
		item["raw_event"] = json.RawMessage(record.RawEventJSON)
		item["connect_metadata"] = json.RawMessage(record.ConnectMetadataJSON)
	}
	return item
}

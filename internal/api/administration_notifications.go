package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func registerAdministrationNotificationRoutes(mux *http.ServeMux, cfg RouterConfig) {
	repo, ok := cfg.QueryRepository.(storage.AdministrationNotificationRepository)
	if !ok {
		return
	}
	mux.HandleFunc("GET /v1/administration/notifications", func(w http.ResponseWriter, r *http.Request) {
		p, _ := principalFromContext(r.Context())
		if cfg.Auth.Enabled && !p.SuperAdmin {
			writeError(w, http.StatusForbidden, "super_admin_required", "super-admin access is required")
			return
		}
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if tenantID == "" {
			writeError(w, 400, "tenant_id_required", "tenant_id is required")
			return
		}
		xs, err := repo.ListAdministrationNotifications(r.Context(), storage.AdministrationNotificationFilter{
			TenantID: tenantID, Subject: p.Subject, Severity: r.URL.Query().Get("severity"), State: r.URL.Query().Get("state"),
			IncludeArchived: r.URL.Query().Get("include_archived") == "true", Limit: queryLimit(r, 100),
		})
		if err != nil {
			writeError(w, 500, "query_failed", err.Error())
			return
		}
		unread := 0
		for _, x := range xs {
			if x.ReadAt == nil {
				unread++
			}
		}
		writeJSON(w, 200, map[string]any{"notifications": xs, "unread_count": unread})
	})
	mux.HandleFunc("POST /v1/administration/notifications/{id}/state", func(w http.ResponseWriter, r *http.Request) {
		p, _ := principalFromContext(r.Context())
		if cfg.Auth.Enabled && !p.SuperAdmin {
			writeError(w, http.StatusForbidden, "super_admin_required", "super-admin access is required")
			return
		}
		var b struct {
			TenantID string `json:"tenant_id"`
			Read     bool   `json:"read"`
			Archived bool   `json:"archived"`
		}
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			writeError(w, 400, "invalid_request", "invalid request")
			return
		}
		if strings.TrimSpace(b.TenantID) == "" {
			writeError(w, 400, "tenant_id_required", "tenant_id is required")
			return
		}
		known, err := repo.ListAdministrationNotifications(r.Context(), storage.AdministrationNotificationFilter{TenantID: b.TenantID, Subject: p.Subject, IncludeArchived: true, Limit: 200})
		if err != nil {
			writeError(w, 500, "query_failed", err.Error())
			return
		}
		found := false
		for _, x := range known {
			if x.NotificationID == r.PathValue("id") {
				found = true
				break
			}
		}
		if !found {
			writeError(w, 404, "notification_not_found", "notification was not found")
			return
		}
		now := time.Now().UTC()
		var read, archived *time.Time
		if b.Read {
			read = &now
		}
		if b.Archived {
			archived = &now
		}
		if err := repo.SetAdministrationNotificationInboxState(r.Context(), storage.AdministrationNotificationInboxState{NotificationID: r.PathValue("id"), Subject: p.Subject, ReadAt: read, ArchivedAt: archived}); err != nil {
			writeError(w, 500, "persistence_failed", err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

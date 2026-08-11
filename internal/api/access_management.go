package api

import (
	"encoding/json"
	"github.com/lukebabs/signalops/internal/storage"
	"net/http"
	"strings"
)

func registerAccessManagementRoutes(mux *http.ServeMux, cfg RouterConfig) {
	registerIDPDirectoryRoute(mux, cfg)
	mux.HandleFunc("GET /v1/administration/access", func(w http.ResponseWriter, r *http.Request) {
		repo := cfg.AccessRepository
		if repo == nil {
			writeError(w, 503, "access_unavailable", "access repository is unavailable")
			return
		}
		tenant, ok := requireRequestTenant(w, r, r.URL.Query().Get("tenant_id"))
		if !ok {
			return
		}
		records, err := repo.ListTenantUserAccess(r.Context(), tenant)
		if err != nil {
			writeError(w, 500, "query_failed", "failed to list tenant access")
			return
		}
		writeJSON(w, 200, map[string]any{"access": records})
	})
	mux.HandleFunc("GET /v1/administration/access/audit", func(w http.ResponseWriter, r *http.Request) {
		repo := cfg.AccessRepository
		if repo == nil {
			writeError(w, 503, "access_unavailable", "access repository is unavailable")
			return
		}
		tenant, ok := requireRequestTenant(w, r, r.URL.Query().Get("tenant_id"))
		if !ok {
			return
		}
		records, err := repo.ListTenantUserAccessAudit(r.Context(), tenant, strings.TrimSpace(r.URL.Query().Get("subject")), queryLimit(r, 100))
		if err != nil {
			writeError(w, 500, "query_failed", "failed to list access audit")
			return
		}
		writeJSON(w, 200, map[string]any{"audit": records})
	})
	mux.HandleFunc("PUT /v1/administration/access", func(w http.ResponseWriter, r *http.Request) {
		repo := cfg.AccessRepository
		if repo == nil {
			writeError(w, 503, "access_unavailable", "access repository is unavailable")
			return
		}
		var body struct {
			TenantID    string `json:"tenant_id"`
			Subject     string `json:"subject"`
			DisplayName string `json:"display_name"`
			Email       string `json:"email"`
			AppID       string `json:"app_id"`
			Permission  string `json:"permission"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, 400, "invalid_request", "invalid access request")
			return
		}
		tenant, ok := requireRequestTenant(w, r, body.TenantID)
		if !ok {
			return
		}
		p, _ := principalFromContext(r.Context())
		record, err := repo.UpsertTenantUserAccess(r.Context(), storage.TenantUserAccessRecord{TenantID: tenant, Subject: body.Subject, DisplayName: body.DisplayName, Email: body.Email, AppID: body.AppID, Permission: body.Permission, GrantedBy: p.Subject}, p.Subject, p.Actor)
		if err != nil {
			writeError(w, 400, "invalid_access_grant", err.Error())
			return
		}
		writeJSON(w, 200, map[string]any{"access": record})
	})
	mux.HandleFunc("DELETE /v1/administration/access/{subject}/{app_id}", func(w http.ResponseWriter, r *http.Request) {
		repo := cfg.AccessRepository
		if repo == nil {
			writeError(w, 503, "access_unavailable", "access repository is unavailable")
			return
		}
		tenant, ok := requireRequestTenant(w, r, r.URL.Query().Get("tenant_id"))
		if !ok {
			return
		}
		p, _ := principalFromContext(r.Context())
		err := repo.DeleteTenantUserAccess(r.Context(), tenant, r.PathValue("subject"), r.PathValue("app_id"), p.Subject, p.Actor)
		if err != nil {
			writeQueryError(w, err, "access_grant_not_found", "access grant not found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

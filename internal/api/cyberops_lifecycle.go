package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

type cyberOpsApprovedServiceRequest struct {
	DestinationIP   string `json:"destination_ip"`
	Protocol        string `json:"protocol"`
	DestinationPort int    `json:"destination_port"`
	Reason          string `json:"reason"`
}

func registerCyberOpsLifecycleRoutes(mux *http.ServeMux, repo storage.QueryRepository) {
	lifecycle, ok := any(repo).(storage.CyberOpsLifecycleRepository)
	authorize := func(w http.ResponseWriter, r *http.Request) (string, bool) {
		tenant := strings.TrimSpace(r.PathValue("tenant_id"))
		p, exists := principalFromContext(r.Context())
		if !exists || p.TenantID != tenant {
			writeError(w, http.StatusForbidden, "tenant_scope_mismatch", "tenant scope mismatch")
			return "", false
		}
		return tenant, true
	}
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/cyberops/lifecycle/policies", func(w http.ResponseWriter, r *http.Request) {
		tenant, valid := authorize(w, r)
		if !valid {
			return
		}
		if !ok {
			writeError(w, 501, "lifecycle_unavailable", "CyberOps lifecycle is unavailable")
			return
		}
		items, err := lifecycle.ListCyberOpsLifecyclePolicies(r.Context(), tenant)
		if err != nil {
			writeError(w, 500, "query_failed", "failed to list lifecycle policies")
			return
		}
		writeJSON(w, 200, map[string]any{"policies": items})
	})
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/cyberops/lifecycle/episodes", func(w http.ResponseWriter, r *http.Request) {
		tenant, valid := authorize(w, r)
		if !valid {
			return
		}
		if !ok {
			writeError(w, 501, "lifecycle_unavailable", "CyberOps lifecycle is unavailable")
			return
		}
		items, err := lifecycle.ListCyberOpsLifecycleEpisodes(r.Context(), tenant, queryLimit(r, 50))
		if err != nil {
			writeError(w, 500, "query_failed", "failed to list lifecycle episodes")
			return
		}
		writeJSON(w, 200, map[string]any{"episodes": items})
	})
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/cyberops/lifecycle/decisions", func(w http.ResponseWriter, r *http.Request) {
		tenant, valid := authorize(w, r)
		if !valid {
			return
		}
		if !ok {
			writeError(w, 501, "lifecycle_unavailable", "CyberOps lifecycle is unavailable")
			return
		}
		items, err := lifecycle.ListCyberOpsLifecycleDecisions(r.Context(), tenant, queryLimit(r, 50))
		if err != nil {
			writeError(w, 500, "query_failed", "failed to list lifecycle decisions")
			return
		}
		writeJSON(w, 200, map[string]any{"decisions": items})
	})
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/cyberops/lifecycle/approved-services", func(w http.ResponseWriter, r *http.Request) {
		tenant, valid := authorize(w, r)
		if !valid {
			return
		}
		if !ok {
			writeError(w, 501, "lifecycle_unavailable", "CyberOps lifecycle is unavailable")
			return
		}
		items, err := lifecycle.ListCyberOpsApprovedServices(r.Context(), tenant)
		if err != nil {
			writeError(w, 500, "query_failed", "failed to list approved services")
			return
		}
		writeJSON(w, 200, map[string]any{"approved_services": items})
	})
	mux.HandleFunc("PUT /v1/tenants/{tenant_id}/cyberops/lifecycle/approved-services", func(w http.ResponseWriter, r *http.Request) {
		tenant, valid := authorize(w, r)
		if !valid {
			return
		}
		if !ok {
			writeError(w, 501, "lifecycle_unavailable", "CyberOps lifecycle is unavailable")
			return
		}
		var req cyberOpsApprovedServiceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, 400, "invalid_json", "invalid approved service")
			return
		}
		ip, err := netip.ParseAddr(strings.TrimSpace(req.DestinationIP))
		if err != nil || req.DestinationPort < 1 || req.DestinationPort > 65535 || (strings.ToLower(req.Protocol) != "tcp" && strings.ToLower(req.Protocol) != "udp") {
			writeError(w, 400, "invalid_service", "destination_ip, tcp/udp protocol, and destination_port are required")
			return
		}
		p, _ := principalFromContext(r.Context())
		service := storage.CyberOpsApprovedService{TenantID: tenant, DestinationIP: ip.String(), Protocol: strings.ToLower(req.Protocol), DestinationPort: req.DestinationPort, ApprovedBy: p.Actor, Reason: strings.TrimSpace(req.Reason)}
		if err := lifecycle.UpsertCyberOpsApprovedService(r.Context(), service, p.Actor); err != nil {
			writeError(w, 500, "persistence_failed", "failed to approve service")
			return
		}
		writeJSON(w, 200, map[string]any{"approved_service": service})
	})
	mux.HandleFunc("DELETE /v1/tenants/{tenant_id}/cyberops/lifecycle/approved-services/{destination_ip}/{protocol}/{destination_port}", func(w http.ResponseWriter, r *http.Request) {
		tenant, valid := authorize(w, r)
		if !valid {
			return
		}
		if !ok {
			writeError(w, 501, "lifecycle_unavailable", "CyberOps lifecycle is unavailable")
			return
		}
		ip, err := netip.ParseAddr(strings.TrimSpace(r.PathValue("destination_ip")))
		port, portErr := parsePositiveInt(r.PathValue("destination_port"))
		protocol := strings.ToLower(strings.TrimSpace(r.PathValue("protocol")))
		if err != nil || portErr != nil || (protocol != "tcp" && protocol != "udp") {
			writeError(w, 400, "invalid_service", "invalid approved service identity")
			return
		}
		p, _ := principalFromContext(r.Context())
		if err := lifecycle.DeleteCyberOpsApprovedService(r.Context(), tenant, ip.String(), protocol, port, p.Actor); err != nil {
			writeQueryError(w, err, "approved_service_not_found", "approved service not found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
func parsePositiveInt(value string) (int, error) {
	var n int
	_, err := fmt.Sscan(value, &n)
	if n < 1 {
		return 0, fmt.Errorf("positive integer required")
	}
	return n, err
}

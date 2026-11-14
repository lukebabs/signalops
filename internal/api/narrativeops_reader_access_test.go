package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNarrativeReaderGrantIsExactAndIdempotent(t *testing.T) {
	grants := 0
	assigned := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer operator" {
			t.Error("missing operator bearer")
			w.WriteHeader(401)
			return
		}
		switch r.URL.Path {
		case "/admin/realms/syncratic/clients":
			if r.URL.Query().Get("clientId") != narrativeReaderClient {
				t.Error("wrong client lookup")
			}
			writeJSON(w, 200, []keycloakReaderClient{{ID: "client-1", ClientID: narrativeReaderClient, Enabled: true, ServiceAccountsEnabled: true}})
		case "/admin/realms/syncratic/clients/client-1/protocol-mappers/models":
			writeJSON(w, 200, []keycloakReaderMapper{
				{Name: "tenant_id", Config: map[string]string{"claim.name": "tenant_id", "claim.value": "tenant-local", "access.token.claim": "true"}},
				{Name: "signalops-api-audience", Config: map[string]string{"included.custom.audience": "signalops-api", "access.token.claim": "true"}},
			})
		case "/admin/realms/syncratic/clients/client-1/service-account-user":
			writeJSON(w, 200, map[string]string{"id": "account-1", "username": "service-account-" + narrativeReaderClient})
		case "/admin/realms/syncratic/users/account-1/role-mappings/realm":
			if r.Method == http.MethodPost {
				var roles []keycloakReaderRole
				if json.NewDecoder(r.Body).Decode(&roles) != nil || len(roles) != 1 || roles[0].Name != narrativeReaderRole || roles[0].ID != "role-1" {
					t.Error("unexpected grant payload")
				}
				assigned = true
				grants++
				w.WriteHeader(204)
				return
			}
			if assigned {
				writeJSON(w, 200, []keycloakReaderRole{{ID: "role-1", Name: narrativeReaderRole}})
			} else {
				writeJSON(w, 200, []keycloakReaderRole{})
			}
		case "/admin/realms/syncratic/roles/narrativeops:reader":
			writeJSON(w, 200, keycloakReaderRole{ID: "role-1", Name: narrativeReaderRole})
		default:
			t.Errorf("unexpected Keycloak path: %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	mux := http.NewServeMux()
	registerNarrativeOpsReaderAccessRoute(mux, RouterConfig{Auth: AuthConfig{Enabled: true, Issuer: server.URL + "/realms/syncratic", HTTPClient: server.Client()}})
	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodPut} {
		r := httptest.NewRequest(method, "/v1/administration/access/narrativeops-reader", nil)
		r.Header.Set("Authorization", "Bearer operator")
		r = r.WithContext(context.WithValue(r.Context(), authContextKey{}, Principal{SuperAdmin: true, TenantID: "tenant-local"}))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatalf("%s: status=%d body=%s", method, w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), "secret") {
			t.Fatal("response exposed credential field")
		}
		if method == http.MethodGet && !strings.Contains(w.Body.String(), `"granted":false`) {
			t.Fatal("expected initial ungranted status")
		}
	}
	if grants != 1 {
		t.Fatalf("grants=%d, want 1", grants)
	}
}

func TestNarrativeReaderGrantRequiresTenantLocalSuperAdmin(t *testing.T) {
	mux := http.NewServeMux()
	registerNarrativeOpsReaderAccessRoute(mux, RouterConfig{Auth: AuthConfig{Enabled: true, Issuer: "https://auth.example.test/realms/syncratic"}})
	for _, principal := range []Principal{{TenantID: "tenant-local"}, {TenantID: "other", SuperAdmin: true}} {
		r := httptest.NewRequest(http.MethodPut, "/v1/administration/access/narrativeops-reader", nil)
		r = r.WithContext(context.WithValue(r.Context(), authContextKey{}, principal))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != http.StatusForbidden {
			t.Fatalf("status=%d", w.Code)
		}
	}
}

func TestNarrativeReaderGrantRejectsTenantMapperDriftBeforeMutation(t *testing.T) {
	writes := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writes++
			w.WriteHeader(500)
			return
		}
		switch r.URL.Path {
		case "/admin/realms/syncratic/clients":
			writeJSON(w, 200, []keycloakReaderClient{{ID: "client-1", ClientID: narrativeReaderClient, Enabled: true, ServiceAccountsEnabled: true}})
		case "/admin/realms/syncratic/clients/client-1/protocol-mappers/models":
			writeJSON(w, 200, []keycloakReaderMapper{{Name: "tenant_id", Config: map[string]string{"claim.name": "tenant_id", "claim.value": "wrong-tenant", "access.token.claim": "true"}}, {Name: "signalops-api-audience", Config: map[string]string{"included.custom.audience": "signalops-api", "access.token.claim": "true"}}})
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	mux := http.NewServeMux()
	registerNarrativeOpsReaderAccessRoute(mux, RouterConfig{Auth: AuthConfig{Enabled: true, Issuer: server.URL + "/realms/syncratic", HTTPClient: server.Client()}})
	r := httptest.NewRequest(http.MethodPut, "/v1/administration/access/narrativeops-reader", nil)
	r = r.WithContext(context.WithValue(r.Context(), authContextKey{}, Principal{SuperAdmin: true, TenantID: "tenant-local"}))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusConflict || writes != 0 {
		t.Fatalf("status=%d writes=%d", w.Code, writes)
	}
}

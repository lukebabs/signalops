package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

const narrativeReaderClient = "syncratic-narrativeops-reader"
const narrativeReaderRole = "narrativeops:reader"

type keycloakReaderClient struct {
	ID                        string `json:"id"`
	ClientID                  string `json:"clientId"`
	Enabled                   bool   `json:"enabled"`
	ServiceAccountsEnabled    bool   `json:"serviceAccountsEnabled"`
	StandardFlowEnabled       bool   `json:"standardFlowEnabled"`
	DirectAccessGrantsEnabled bool   `json:"directAccessGrantsEnabled"`
	ImplicitFlowEnabled       bool   `json:"implicitFlowEnabled"`
}

type keycloakReaderRole struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type keycloakReaderMapper struct {
	Name   string            `json:"name"`
	Config map[string]string `json:"config"`
}

// This route deliberately has no variable client, role, or tenant parameters.
// It forwards the operator's bearer token to Keycloak; SignalOps stores no
// realm-admin credential and never reads or returns the service client secret.
func registerNarrativeOpsReaderAccessRoute(mux *http.ServeMux, cfg RouterConfig) {
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		mux.HandleFunc(method+" /v1/administration/access/narrativeops-reader", func(w http.ResponseWriter, r *http.Request) {
			if !cfg.Auth.Enabled {
				writeError(w, http.StatusServiceUnavailable, "reader_grant_unavailable", "authenticated mode is required")
				return
			}
			principal, ok := principalFromContext(r.Context())
			if !ok || !principal.SuperAdmin || principal.TenantID != "tenant-local" {
				writeError(w, http.StatusForbidden, "insufficient_role", "tenant-local super-admin access is required")
				return
			}
			base, err := keycloakRealmAdminBase(cfg.Auth.Issuer)
			if err != nil {
				writeError(w, http.StatusServiceUnavailable, "reader_grant_unavailable", "IdP issuer is not configured")
				return
			}
			client := cfg.Auth.HTTPClient
			if client == nil {
				client = http.DefaultClient
			}
			call := func(method, path string, payload any, target any) int {
				var body *bytes.Reader
				if payload == nil {
					body = bytes.NewReader(nil)
				} else {
					encoded, marshalErr := json.Marshal(payload)
					if marshalErr != nil {
						return http.StatusInternalServerError
					}
					body = bytes.NewReader(encoded)
				}
				request, requestErr := http.NewRequestWithContext(r.Context(), method, base+path, body)
				if requestErr != nil {
					return http.StatusInternalServerError
				}
				request.Header.Set("Authorization", r.Header.Get("Authorization"))
				request.Header.Set("Accept", "application/json")
				if payload != nil {
					request.Header.Set("Content-Type", "application/json")
				}
				response, callErr := client.Do(request)
				if callErr != nil {
					return http.StatusBadGateway
				}
				defer response.Body.Close()
				if response.StatusCode < 200 || response.StatusCode >= 300 {
					return response.StatusCode
				}
				if target != nil && json.NewDecoder(response.Body).Decode(target) != nil {
					return http.StatusBadGateway
				}
				return response.StatusCode
			}
			var clients []keycloakReaderClient
			status := call(http.MethodGet, "/clients?clientId="+url.QueryEscape(narrativeReaderClient), nil, &clients)
			if status == http.StatusForbidden || status == http.StatusUnauthorized {
				writeError(w, http.StatusForbidden, "idp_admin_denied", "Keycloak realm-management permission is required")
				return
			}
			if status != http.StatusOK || len(clients) != 1 || clients[0].ID == "" || clients[0].ClientID != narrativeReaderClient {
				writeError(w, http.StatusBadGateway, "reader_client_unavailable", "exact Keycloak reader client was not found")
				return
			}
			bound := clients[0]
			if !bound.Enabled || !bound.ServiceAccountsEnabled || bound.StandardFlowEnabled || bound.DirectAccessGrantsEnabled || bound.ImplicitFlowEnabled {
				writeError(w, http.StatusConflict, "reader_client_contract_mismatch", "reader client does not match the service-only contract")
				return
			}
			var mappers []keycloakReaderMapper
			if call(http.MethodGet, "/clients/"+url.PathEscape(bound.ID)+"/protocol-mappers/models", nil, &mappers) != http.StatusOK {
				writeError(w, http.StatusBadGateway, "reader_mappers_unavailable", "reader audience and tenant could not be verified")
				return
			}
			audienceOK, tenantOK := false, false
			for _, mapper := range mappers {
				if mapper.Name == "signalops-api-audience" && mapper.Config["included.custom.audience"] == "signalops-api" && mapper.Config["access.token.claim"] == "true" {
					audienceOK = true
				}
				if mapper.Name == "tenant_id" && mapper.Config["claim.name"] == "tenant_id" && mapper.Config["claim.value"] == "tenant-local" && mapper.Config["access.token.claim"] == "true" {
					tenantOK = true
				}
			}
			if !audienceOK || !tenantOK {
				writeError(w, http.StatusConflict, "reader_mapper_contract_mismatch", "reader audience or tenant claim does not match the approved contract")
				return
			}
			var account struct {
				ID       string `json:"id"`
				Username string `json:"username"`
			}
			if call(http.MethodGet, "/clients/"+url.PathEscape(bound.ID)+"/service-account-user", nil, &account) != http.StatusOK || account.ID == "" || account.Username != "service-account-"+narrativeReaderClient {
				writeError(w, http.StatusBadGateway, "reader_account_unavailable", "exact service account was not found")
				return
			}
			var assigned []keycloakReaderRole
			if call(http.MethodGet, "/users/"+url.PathEscape(account.ID)+"/role-mappings/realm", nil, &assigned) != http.StatusOK {
				writeError(w, http.StatusBadGateway, "reader_roles_unavailable", "service-account roles could not be verified")
				return
			}
			granted := false
			for _, role := range assigned {
				if role.Name == narrativeReaderRole {
					granted = true
				}
				if role.Name == "super_admin" || role.Name == "signalops:admin" {
					writeError(w, http.StatusConflict, "reader_admin_role_present", "reader has an incompatible administrator role")
					return
				}
			}
			if r.Method == http.MethodPut && !granted {
				var role keycloakReaderRole
				if call(http.MethodGet, "/roles/"+url.PathEscape(narrativeReaderRole), nil, &role) != http.StatusOK || role.ID == "" || role.Name != narrativeReaderRole {
					writeError(w, http.StatusBadGateway, "reader_role_unavailable", "exact Keycloak reader role was not found")
					return
				}
				if call(http.MethodPost, "/users/"+url.PathEscape(account.ID)+"/role-mappings/realm", []keycloakReaderRole{role}, nil) != http.StatusNoContent {
					writeError(w, http.StatusBadGateway, "reader_grant_failed", "Keycloak rejected the exact reader-role grant")
					return
				}
				assigned = nil
				if call(http.MethodGet, "/users/"+url.PathEscape(account.ID)+"/role-mappings/realm", nil, &assigned) != http.StatusOK {
					writeError(w, http.StatusBadGateway, "reader_grant_unverified", "role grant could not be verified")
					return
				}
				granted = false
				for _, role := range assigned {
					if role.Name == narrativeReaderRole {
						granted = true
					}
				}
				if !granted {
					writeError(w, http.StatusBadGateway, "reader_grant_unverified", "reader role is absent after grant")
					return
				}
			}
			writeJSON(w, http.StatusOK, map[string]any{"client_id": narrativeReaderClient, "tenant_id": "tenant-local", "audience": "signalops-api", "role": narrativeReaderRole, "granted": granted})
		})
	}
}

func keycloakRealmAdminBase(issuer string) (string, error) {
	endpoint, err := keycloakUserSearchURL(issuer, "placeholder")
	if err != nil {
		return "", err
	}
	return strings.Split(endpoint, "/users?")[0], nil
}

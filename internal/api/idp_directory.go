package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type idpDirectoryUser struct {
	Subject     string `json:"subject"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Enabled     bool   `json:"enabled"`
}

// registerIDPDirectoryRoute exposes a narrowly-scoped Keycloak directory lookup
// to SignalOps super-admins. The caller's bearer token is forwarded; no IdP
// administration secret is stored in or exposed by SignalOps.
func registerIDPDirectoryRoute(mux *http.ServeMux, cfg RouterConfig) {
	mux.HandleFunc("GET /v1/administration/idp-users", func(w http.ResponseWriter, r *http.Request) {
		if !cfg.Auth.Enabled {
			writeError(w, http.StatusServiceUnavailable, "idp_search_unavailable", "IdP user search requires authenticated mode")
			return
		}
		principal, ok := principalFromContext(r.Context())
		if !ok || !principal.SuperAdmin {
			writeError(w, http.StatusForbidden, "insufficient_role", "super-admin access is required")
			return
		}
		query := strings.TrimSpace(r.URL.Query().Get("query"))
		if len(query) < 2 {
			writeError(w, http.StatusBadRequest, "invalid_query", "search query must contain at least two characters")
			return
		}
		endpoint, err := keycloakUserSearchURL(cfg.Auth.Issuer, query)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "idp_search_unavailable", err.Error())
			return
		}
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, endpoint, nil)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "idp_search_failed", "failed to create IdP search request")
			return
		}
		req.Header.Set("Authorization", r.Header.Get("Authorization"))
		req.Header.Set("Accept", "application/json")
		client := cfg.Auth.HTTPClient
		if client == nil {
			client = http.DefaultClient
		}
		response, err := client.Do(req)
		if err != nil {
			writeError(w, http.StatusBadGateway, "idp_search_failed", "IdP user search is unavailable")
			return
		}
		defer response.Body.Close()
		if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
			writeError(w, http.StatusBadGateway, "idp_search_not_authorized", "Keycloak denied directory search; grant super_admin realm-management query-users and view-users")
			return
		}
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			writeError(w, http.StatusBadGateway, "idp_search_failed", "IdP user search failed")
			return
		}
		var upstream []struct {
			ID        string `json:"id"`
			Username  string `json:"username"`
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
			Email     string `json:"email"`
			Enabled   bool   `json:"enabled"`
		}
		if err := json.NewDecoder(response.Body).Decode(&upstream); err != nil {
			writeError(w, http.StatusBadGateway, "idp_search_failed", "IdP user search returned invalid JSON")
			return
		}
		users := make([]idpDirectoryUser, 0, len(upstream))
		for _, user := range upstream {
			displayName := strings.TrimSpace(strings.TrimSpace(user.FirstName) + " " + strings.TrimSpace(user.LastName))
			if displayName == "" {
				displayName = user.Username
			}
			users = append(users, idpDirectoryUser{Subject: user.ID, Username: user.Username, DisplayName: displayName, Email: user.Email, Enabled: user.Enabled})
		}
		writeJSON(w, http.StatusOK, map[string]any{"users": users})
	})
}

func keycloakUserSearchURL(issuer, query string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(issuer))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("IdP issuer is not configured")
	}
	path := strings.TrimSuffix(parsed.Path, "/")
	marker := "/realms/"
	idx := strings.LastIndex(path, marker)
	if idx < 0 || strings.TrimSpace(path[idx+len(marker):]) == "" {
		return "", fmt.Errorf("IdP issuer does not identify a realm")
	}
	realm := strings.TrimSpace(path[idx+len(marker):])
	basePath := strings.TrimSuffix(path[:idx], "/")
	endpoint := &url.URL{Scheme: parsed.Scheme, Host: parsed.Host, Path: basePath + "/admin/realms/" + url.PathEscape(realm) + "/users"}
	parameters := endpoint.Query()
	parameters.Set("search", query)
	parameters.Set("max", "20")
	parameters.Set("briefRepresentation", "true")
	endpoint.RawQuery = parameters.Encode()
	return endpoint.String(), nil
}

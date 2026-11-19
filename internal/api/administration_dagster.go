package api

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

const (
	dagsterAdminPrefix   = "/admin/orchestration/dagster"
	dagsterAdminPath     = "/admin/orchestration/dagster/"
	dagsterSessionPath   = "/v1/administration/orchestration/dagster/session"
	dagsterSessionCookie = "signalops_dagster_admin"
)

// registerDagsterAdministrationRoutes exposes only the read-only Dagster UI.
// The browser first calls the session endpoint with its normal SignalOps bearer
// token. The short-lived, path-scoped cookie then allows top-level navigation to
// the proxied UI without placing a JWT in a URL.
func registerDagsterAdministrationRoutes(mux *http.ServeMux, cfg RouterConfig) {
	mux.HandleFunc("POST "+dagsterSessionPath, func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromContext(r.Context())
		if !ok || !principal.SuperAdmin {
			writeError(w, http.StatusForbidden, "insufficient_role", "Dagster observability requires a platform administrator")
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if token == "" {
			writeError(w, http.StatusUnauthorized, "missing_authorization", "an authenticated SignalOps session is required")
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name: dagsterSessionCookie, Value: token, Path: dagsterAdminPath,
			MaxAge: 300, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode,
		})
		writeJSON(w, http.StatusOK, map[string]string{"path": dagsterAdminPath})
	})

	mux.Handle(dagsterAdminPath, dagsterProxyHandler(cfg))
}

func dagsterProxyHandler(cfg RouterConfig) http.Handler {
	target := strings.TrimSpace(cfg.DagsterReadonlyURL)
	parsed, err := url.Parse(target)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeError(w, http.StatusServiceUnavailable, "dagster_unavailable", "Dagster read-only endpoint is not configured")
		})
	}
	proxy := httputil.NewSingleHostReverseProxy(parsed)
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		writeError(w, http.StatusBadGateway, "dagster_proxy_failed", "Dagster read-only endpoint is unavailable")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !cfg.Auth.Enabled {
			writeError(w, http.StatusForbidden, "admin_auth_required", "Dagster observability requires gateway authentication")
			return
		}
		principal, ok := principalFromContext(r.Context())
		if !ok || !principal.SuperAdmin {
			writeError(w, http.StatusForbidden, "insufficient_role", "Dagster observability requires a platform administrator")
			return
		}
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodOptions:
		default:
			writeError(w, http.StatusMethodNotAllowed, "dagster_read_only", "Dagster administration is read-only")
			return
		}
		// Do not forward the SignalOps JWT or the bootstrap cookie to Dagster.
		r.Header.Del("Authorization")
		r.Header.Del("Cookie")
		proxy.ServeHTTP(w, r)
	})
}

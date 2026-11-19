package api

import (
	"net/http"

	"github.com/lukebabs/signalops/internal/appmeta"
)

type sessionExperienceProfile struct {
	appmeta.Profile
	Permission string `json:"permission"`
}

func registerSessionExperienceRoute(mux *http.ServeMux, cfg RouterConfig) {
	mux.HandleFunc("GET /v1/session/experience", func(w http.ResponseWriter, r *http.Request) {
		principal, authenticated := principalFromContext(r.Context())
		if cfg.Auth.Enabled && !authenticated {
			writeError(w, http.StatusUnauthorized, "unauthorized", "authenticated session is required")
			return
		}
		profiles := make([]sessionExperienceProfile, 0)
		for _, profile := range appmeta.Profiles {
			if profile.AppID == appmeta.AppConsole {
				continue
			}
			permission := sessionProfilePermission(cfg, principal, profile)
			if permission != "" {
				profiles = append(profiles, sessionExperienceProfile{Profile: profile, Permission: permission})
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"tenant_id":    principal.TenantID,
			"super_admin":  principal.SuperAdmin || !cfg.Auth.Enabled,
			"app_profiles": profiles,
		})
	})
}

func sessionProfilePermission(cfg RouterConfig, principal Principal, profile appmeta.Profile) string {
	// NarrativeOps is an administrative intelligence workspace. Keep it out of
	// landing cards for ordinary users even if a stale or overly broad access
	// grant exists.
	if cfg.Auth.Enabled && profile.AppID == appmeta.AppNarrativeOps && !principal.SuperAdmin {
		return ""
	}
	switch {
	case !cfg.Auth.Enabled, principal.SuperAdmin:
		return "write"
	case principal.Access != nil:
		return principal.Access[profile.AppID]
	default:
		return ""
	}
}

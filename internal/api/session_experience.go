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
			permission := ""
			switch {
			case !cfg.Auth.Enabled, principal.SuperAdmin:
				permission = "write"
			case principal.Access != nil:
				permission = principal.Access[profile.AppID]
			}
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

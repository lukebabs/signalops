package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSessionExperienceReturnsRegisteredUseCasesInLocalMode(t *testing.T) {
	router := NewRouter(RouterConfig{})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/session/experience", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var body struct {
		SuperAdmin  bool `json:"super_admin"`
		AppProfiles []struct {
			AppID          string `json:"app_id"`
			Permission     string `json:"permission"`
			LandingSummary string `json:"landing_summary"`
		} `json:"app_profiles"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.SuperAdmin || len(body.AppProfiles) < 2 {
		t.Fatalf("unexpected experience: %+v", body)
	}
	for _, profile := range body.AppProfiles {
		if profile.AppID == "console" || profile.Permission != "write" || profile.LandingSummary == "" {
			t.Fatalf("invalid profile: %+v", profile)
		}
	}
}

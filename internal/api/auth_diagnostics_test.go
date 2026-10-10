package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAuthDiagnosticRouteAcceptsBoundedJourneyEvent(t *testing.T) {
	authDiagnosticLimiter.Lock()
	authDiagnosticLimiter.seen = make(map[string][]time.Time)
	authDiagnosticLimiter.Unlock()
	mux := http.NewServeMux()
	registerAuthDiagnosticRoute(mux)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/diagnostics", strings.NewReader(`{"event_type":"redirect_started","correlation_id":"auth-test","destination":"/","oidc_issuer":"https://auth.example/realms/syncratic","redirect_uri":"https://signalops.example/auth/callback"}`))
	req.RemoteAddr = "192.0.2.10:1234"
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, req)
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected accepted auth diagnostic, got %d: %s", response.Code, response.Body.String())
	}
}

func TestAuthDiagnosticRouteRateLimitsClient(t *testing.T) {
	authDiagnosticLimiter.Lock()
	authDiagnosticLimiter.seen = make(map[string][]time.Time)
	authDiagnosticLimiter.Unlock()
	mux := http.NewServeMux()
	registerAuthDiagnosticRoute(mux)
	for index := 0; index < authDiagnosticLimit; index++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/diagnostics", strings.NewReader(`{"event_type":"redirect_started","correlation_id":"auth-test"}`))
		req.RemoteAddr = "192.0.2.11:1234"
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		if response.Code != http.StatusAccepted {
			t.Fatalf("request %d unexpectedly rejected: %d", index, response.Code)
		}
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/diagnostics", strings.NewReader(`{"event_type":"redirect_started","correlation_id":"auth-test"}`))
	req.RemoteAddr = "192.0.2.11:1234"
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, req)
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("expected rate limit, got %d", response.Code)
	}
}

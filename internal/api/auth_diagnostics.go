package api

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	authDiagnosticMaxBody = 4096
	authDiagnosticWindow  = time.Minute
	authDiagnosticLimit   = 20
)

type authDiagnosticRequest struct {
	EventType   string `json:"event_type"`
	Correlation string `json:"correlation_id"`
	Destination string `json:"destination"`
	OIDCIssuer  string `json:"oidc_issuer"`
	RedirectURI string `json:"redirect_uri"`
	Error       string `json:"error"`
}

var authDiagnosticLimiter = struct {
	sync.Mutex
	seen map[string][]time.Time
}{seen: make(map[string][]time.Time)}

func registerAuthDiagnosticRoute(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/auth/diagnostics", func(w http.ResponseWriter, r *http.Request) {
		if !allowAuthDiagnostic(clientAddress(r), time.Now()) {
			writeError(w, http.StatusTooManyRequests, "auth_diagnostics_rate_limited", "auth diagnostics rate limit exceeded")
			return
		}
		var request authDiagnosticRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, authDiagnosticMaxBody))
		if err := decoder.Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_auth_diagnostic", "invalid auth diagnostic payload")
			return
		}
		request.EventType = strings.TrimSpace(request.EventType)
		request.Correlation = strings.TrimSpace(request.Correlation)
		request.Destination = strings.TrimSpace(request.Destination)
		request.OIDCIssuer = strings.TrimSpace(request.OIDCIssuer)
		request.RedirectURI = strings.TrimSpace(request.RedirectURI)
		request.Error = strings.TrimSpace(request.Error)
		if !validAuthDiagnosticEvent(request.EventType) || request.Correlation == "" || len(request.Correlation) > 96 {
			writeError(w, http.StatusBadRequest, "invalid_auth_diagnostic", "unsupported auth diagnostic event")
			return
		}
		if len(request.Destination) > 256 || len(request.OIDCIssuer) > 256 || len(request.RedirectURI) > 256 || len(request.Error) > 256 {
			writeError(w, http.StatusBadRequest, "invalid_auth_diagnostic", "auth diagnostic field is too long")
			return
		}
		slog.Default().Info("signalops auth journey", "event_type", request.EventType, "correlation_id", request.Correlation,
			"destination", request.Destination, "oidc_issuer", request.OIDCIssuer, "redirect_uri", request.RedirectURI,
			"error", request.Error,
			"host", r.Host, "origin", r.Header.Get("Origin"), "referer", r.Header.Get("Referer"),
			"user_agent", r.UserAgent(), "remote_addr", clientAddress(r))
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "recorded"})
	})
}

func validAuthDiagnosticEvent(event string) bool {
	switch event {
	case "redirect_started", "callback_succeeded", "callback_failed":
		return true
	default:
		return false
	}
}

func clientAddress(r *http.Request) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func allowAuthDiagnostic(client string, now time.Time) bool {
	authDiagnosticLimiter.Lock()
	defer authDiagnosticLimiter.Unlock()
	cutoff := now.Add(-authDiagnosticWindow)
	entries := authDiagnosticLimiter.seen[client]
	kept := entries[:0]
	for _, at := range entries {
		if at.After(cutoff) {
			kept = append(kept, at)
		}
	}
	if len(kept) >= authDiagnosticLimit {
		authDiagnosticLimiter.seen[client] = kept
		return false
	}
	authDiagnosticLimiter.seen[client] = append(kept, now)
	return true
}

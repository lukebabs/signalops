package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type subscriberSessionActivityRequest struct {
	EventType     string          `json:"event_type"`
	AppID         string          `json:"app_id"`
	FeatureKey    string          `json:"feature_key"`
	RoutePath     string          `json:"route_path"`
	CorrelationID string          `json:"correlation_id"`
	Metadata      json.RawMessage `json:"metadata"`
}

func registerSubscriberSessionActivityRoutes(mux *http.ServeMux, cfg RouterConfig) {
	repository := cfg.SubscriberSubscriptionAdministrationRepository
	if repository == nil {
		return
	}
	mux.HandleFunc("POST /v1/session/activity", func(w http.ResponseWriter, r *http.Request) {
		principal, authenticated := principalFromContext(r.Context())
		if !authenticated || strings.TrimSpace(principal.TenantID) == "" || strings.TrimSpace(principal.Subject) == "" {
			writeError(w, http.StatusForbidden, "session_activity_identity_required", "authenticated tenant-scoped identity is required")
			return
		}
		var request subscriberSessionActivityRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		eventType := strings.TrimSpace(request.EventType)
		if eventType != "login" && eventType != "logout" && eventType != "feature_view" {
			writeError(w, http.StatusBadRequest, "invalid_activity_event", "session activity event_type must be login, logout, or feature_view")
			return
		}
		metadata := request.Metadata
		if len(metadata) == 0 {
			metadata = json.RawMessage(`{}`)
		}
		if err := repository.RecordSubscriberUserActivity(r.Context(), storage.SubscriberUserActivityRecordInput{
			TenantID: strings.TrimSpace(principal.TenantID), Subject: strings.TrimSpace(principal.Subject),
			SubjectDisplayName: strings.TrimSpace(principal.PreferredName), SubjectEmail: strings.TrimSpace(principal.Email),
			AppID: firstNonEmpty(strings.TrimSpace(request.AppID), "marketops"), EventType: eventType,
			FeatureKey: strings.TrimSpace(request.FeatureKey), RoutePath: normalizeActivityRoutePath(strings.TrimSpace(request.RoutePath)),
			CorrelationID: subscriptionCorrelationID(r, request.CorrelationID), MetadataJSON: []byte(metadata),
		}); err != nil {
			writeSubscriberSubscriptionAdministrationError(w, err)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "recorded"})
	})
}

func subscriberUserActivityMiddleware(next http.Handler, cfg RouterConfig) http.Handler {
	repository := cfg.SubscriberSubscriptionAdministrationRepository
	if repository == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		shouldCapture := isMarketOpsActivityMutationRequest(r)
		recorder := &activityResponseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		if shouldCapture {
			next.ServeHTTP(recorder, r)
			recordMarketOpsActivityMutation(r, repository, recorder.statusCode)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type activityResponseRecorder struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (r *activityResponseRecorder) WriteHeader(statusCode int) {
	if !r.wroteHeader {
		r.statusCode = statusCode
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(statusCode)
}

func isMarketOpsActivityMutationRequest(r *http.Request) bool {
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodDelete:
	default:
		return false
	}
	path := strings.TrimSpace(r.URL.Path)
	return strings.HasPrefix(path, "/v1/tenants/") && strings.Contains(path, "/marketops/")
}

func recordMarketOpsActivityMutation(r *http.Request, repository storage.SubscriberSubscriptionAdministrationRepository, statusCode int) {
	principal, authenticated := principalFromContext(r.Context())
	if !authenticated || strings.TrimSpace(principal.TenantID) == "" || strings.TrimSpace(principal.Subject) == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = repository.RecordSubscriberUserActivity(ctx, storage.SubscriberUserActivityRecordInput{
		TenantID: strings.TrimSpace(principal.TenantID), Subject: strings.TrimSpace(principal.Subject),
		SubjectDisplayName: strings.TrimSpace(principal.PreferredName), SubjectEmail: strings.TrimSpace(principal.Email), AppID: "marketops",
		EventType: "api_mutation", FeatureKey: activityFeatureKey(r.URL.Path), HTTPMethod: r.Method,
		RoutePath: normalizeActivityRoutePath(r.URL.Path), StatusCode: statusCode,
		CorrelationID: firstNonEmpty(headerValue(r, "X-Correlation-ID"), headerValue(r, "X-Request-ID")),
		MetadataJSON:  []byte(`{"capture":"gateway_marketops_mutation"}`),
	})
}

func activityFeatureKey(path string) string {
	normalized := normalizeActivityRoutePath(path)
	parts := strings.Split(strings.Trim(normalized, "/"), "/")
	for idx, part := range parts {
		if part == "marketops" && idx+1 < len(parts) {
			return parts[idx+1]
		}
	}
	return ""
}

func normalizeActivityRoutePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 3 && parts[0] == "v1" && parts[1] == "tenants" {
		parts[2] = "{tenant}"
		return "/" + strings.Join(parts, "/")
	}
	return path
}

func subscriberUserActivityResponse(snapshot storage.SubscriberUserActivitySnapshot) map[string]any {
	summaries := make([]map[string]any, 0, len(snapshot.Summaries))
	for _, record := range snapshot.Summaries {
		summaries = append(summaries, map[string]any{
			"subject": record.Subject, "subject_display_name": record.SubjectDisplayName, "subject_email": record.SubjectEmail,
			"last_activity_at": record.LastActivityAt, "last_login_at": record.LastLoginAt, "last_logout_at": record.LastLogoutAt,
			"login_count": record.LoginCount, "feature_view_count": record.FeatureViewCount, "mutation_count": record.MutationCount,
			"failed_mutation_count": record.FailedMutationCount, "top_feature_key": record.TopFeatureKey,
		})
	}
	events := make([]map[string]any, 0, len(snapshot.Events))
	for _, record := range snapshot.Events {
		events = append(events, map[string]any{
			"activity_id": record.ActivityID, "tenant_id": record.TenantID, "subject": record.Subject,
			"subject_display_name": record.SubjectDisplayName, "subject_email": record.SubjectEmail,
			"app_id": record.AppID, "event_type": record.EventType, "feature_key": record.FeatureKey,
			"http_method": record.HTTPMethod, "route_path": record.RoutePath, "status_code": record.StatusCode,
			"correlation_id": record.CorrelationID, "metadata": subscriptionJSON(record.MetadataJSON), "occurred_at": record.OccurredAt,
		})
	}
	return map[string]any{"tenant_id": snapshot.TenantID, "summaries": summaries, "events": events}
}

func subscriberActivityLimit(r *http.Request) int {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return 100
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 100
	}
	if value < 1 {
		return 1
	}
	if value > 500 {
		return 500
	}
	return value
}

func parseOptionalActivityTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}
	return &parsed
}

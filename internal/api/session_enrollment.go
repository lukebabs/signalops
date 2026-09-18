package api

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const (
	sessionEnrollmentStateReady               = "marketops_ready"
	sessionEnrollmentStateEmailVerification   = "email_verification_required"
	sessionEnrollmentStateAccessMissing       = "tenant_access_missing"
	sessionEnrollmentStateSubscriptionMissing = "subscription_missing"
	sessionEnrollmentStateWatchlistMissing    = "watchlist_context_missing"
)

var defaultSessionEnrollmentLimiter = newSessionEnrollmentLimiter(12, 10*time.Minute)

type sessionEnrollmentLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	hits   map[string][]time.Time
}

func newSessionEnrollmentLimiter(limit int, window time.Duration) *sessionEnrollmentLimiter {
	return &sessionEnrollmentLimiter{limit: limit, window: window, hits: map[string][]time.Time{}}
}

func (l *sessionEnrollmentLimiter) allow(key string, now time.Time) bool {
	if l == nil || l.limit <= 0 || l.window <= 0 {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := now.Add(-l.window)
	current := l.hits[key]
	kept := current[:0]
	for _, seen := range current {
		if seen.After(cutoff) {
			kept = append(kept, seen)
		}
	}
	if len(kept) >= l.limit {
		l.hits[key] = kept
		return false
	}
	kept = append(kept, now)
	l.hits[key] = kept
	return true
}

func registerSessionEnrollmentRoute(mux *http.ServeMux, cfg RouterConfig) {
	mux.HandleFunc("GET /v1/session/enrollment", func(w http.ResponseWriter, r *http.Request) {
		principal, authenticated := principalFromContext(r.Context())
		if cfg.Auth.Enabled && !authenticated {
			writeError(w, http.StatusUnauthorized, "unauthorized", "authenticated session is required")
			return
		}
		if strings.TrimSpace(principal.TenantID) == "" || strings.TrimSpace(principal.Subject) == "" {
			writeError(w, http.StatusForbidden, "enrollment_identity_required", "authenticated tenant-scoped identity is required")
			return
		}
		tenantID := strings.TrimSpace(principal.TenantID)
		subject := strings.TrimSpace(principal.Subject)
		b2cTenantID := strings.TrimSpace(cfg.SubscriberB2CTenantID)
		canSelfEnroll := b2cTenantID != "" && tenantID == b2cTenantID
		created := []string{}

		accessRecords := []storage.TenantUserAccessRecord{}
		if cfg.AccessRepository != nil {
			records, err := cfg.AccessRepository.ListTenantUserAccessForSubject(r.Context(), tenantID, subject)
			if err != nil {
				writeError(w, http.StatusServiceUnavailable, "enrollment_access_unavailable", "tenant access could not be resolved")
				return
			}
			accessRecords = records
			if !hasMarketOpsAccess(records) && canSelfEnroll && principal.EmailVerified {
				if !allowSessionEnrollmentMutation(w, r, principal) {
					return
				}
				record, err := cfg.AccessRepository.UpsertTenantUserAccess(r.Context(), storage.TenantUserAccessRecord{
					TenantID: tenantID, Subject: subject, DisplayName: firstNonEmpty(principal.PreferredName, principal.Actor, principal.Email),
					Email: principal.Email, AppID: "marketops", Permission: "read", GrantedBy: "self-enrollment",
				}, "self-enrollment", "SignalOps Enrollment")
				if err != nil {
					writeError(w, http.StatusServiceUnavailable, "enrollment_access_failed", "MarketOps access could not be provisioned")
					return
				}
				accessRecords = append(accessRecords, record)
				created = append(created, "marketops_access")
			}
		}

		if !principal.EmailVerified {
			writeJSON(w, http.StatusOK, sessionEnrollmentResponse(principal, sessionEnrollmentStateEmailVerification, created, accessRecords, nil, nil, canSelfEnroll))
			return
		}
		if !hasMarketOpsAccess(accessRecords) {
			writeJSON(w, http.StatusOK, sessionEnrollmentResponse(principal, sessionEnrollmentStateAccessMissing, created, accessRecords, nil, nil, canSelfEnroll))
			return
		}

		requiresB2CSubscription := requiresB2CSubscriptionActivation(cfg, canSelfEnroll, accessRecords)
		if requiresB2CSubscription && cfg.SubscriberSubscriptionRepository == nil {
			writeError(w, http.StatusServiceUnavailable, "enrollment_subscription_unavailable", "subscription state could not be resolved")
			return
		}

		var subscription *storage.SubscriberEffectiveSubscriptionRecord
		if cfg.SubscriberSubscriptionRepository != nil {
			record, err := cfg.SubscriberSubscriptionRepository.GetSubscriberEffectiveSubscription(r.Context(), tenantID, subject)
			if errors.Is(err, storage.ErrNotFound) && canSelfEnroll && cfg.SubscriberB2CAutoActivateExplorer && cfg.SubscriberSubscriptionAdministrationRepository != nil {
				if !allowSessionEnrollmentMutation(w, r, principal) {
					return
				}
				if err := cfg.SubscriberSubscriptionAdministrationRepository.UpsertSubscriberSubjectSubscription(r.Context(), storage.SubscriberSubjectSubscriptionMutation{
					TenantID: tenantID, Subject: subject, ProductKey: "explorer", Status: "active", ActorSubject: "self-enrollment", CorrelationID: "session-enrollment",
				}); err != nil {
					writeError(w, http.StatusServiceUnavailable, "enrollment_subscription_failed", "Explorer subscription could not be provisioned")
					return
				}
				created = append(created, "explorer_subscription")
				record, err = cfg.SubscriberSubscriptionRepository.GetSubscriberEffectiveSubscription(r.Context(), tenantID, subject)
			}
			if errors.Is(err, storage.ErrNotFound) {
				if cfg.SubscriberSubscriptionsEnabled || requiresB2CSubscription {
					writeJSON(w, http.StatusOK, sessionEnrollmentResponse(principal, sessionEnrollmentStateSubscriptionMissing, created, accessRecords, nil, nil, canSelfEnroll))
					return
				}
			} else if err != nil {
				writeError(w, http.StatusServiceUnavailable, "enrollment_subscription_unavailable", "subscription state could not be resolved")
				return
			} else {
				subscription = &record
			}
		}

		var watchlistContext *subscriberWatchlistContext
		if subscriberWatchlistContextEnabled(cfg, tenantID) {
			context, err := resolveSubscriberWatchlistContext(r, cfg, tenantID, subject)
			if errors.Is(err, storage.ErrNotFound) && canSelfEnroll && cfg.SubscriberWatchlistRepository != nil {
				if !allowSessionEnrollmentMutation(w, r, principal) {
					return
				}
				_, createErr := cfg.SubscriberWatchlistRepository.CreateSubscriberTenantDefaultWatchlist(r.Context(), storage.SubscriberWatchlistCreateRequest{
					TenantID: tenantID, ListName: "MarketOps Starter List", ActorSubject: "self-enrollment", CorrelationID: "session-enrollment", ProvenanceJSON: []byte(`{"source":"session_enrollment"}`),
				})
				if createErr != nil && !errors.Is(createErr, storage.ErrConflict) {
					writeError(w, http.StatusServiceUnavailable, "enrollment_watchlist_failed", "default watchlist could not be provisioned")
					return
				}
				if createErr == nil {
					created = append(created, "tenant_default_watchlist")
				}
				context, err = resolveSubscriberWatchlistContext(r, cfg, tenantID, subject)
			}
			if errors.Is(err, storage.ErrNotFound) {
				writeJSON(w, http.StatusOK, sessionEnrollmentResponse(principal, sessionEnrollmentStateWatchlistMissing, created, accessRecords, subscription, nil, canSelfEnroll))
				return
			}
			if err != nil {
				writeError(w, http.StatusServiceUnavailable, "enrollment_watchlist_unavailable", "watchlist context could not be resolved")
				return
			}
			watchlistContext = &context
		}

		if cfg.SubscriberSubscriptionAdministrationRepository != nil {
			_ = cfg.SubscriberSubscriptionAdministrationRepository.RecordSubscriberUserActivity(r.Context(), storage.SubscriberUserActivityRecordInput{
				TenantID: tenantID, Subject: subject, SubjectDisplayName: strings.TrimSpace(principal.PreferredName), SubjectEmail: strings.TrimSpace(principal.Email),
				AppID: "marketops", EventType: "feature_view", FeatureKey: "enrollment", RoutePath: "/v1/session/enrollment", CorrelationID: "session-enrollment", MetadataJSON: []byte(`{"milestone":"resolved"}`),
			})
		}
		writeJSON(w, http.StatusOK, sessionEnrollmentResponse(principal, sessionEnrollmentStateReady, created, accessRecords, subscription, watchlistContext, canSelfEnroll))
	})
}

func allowSessionEnrollmentMutation(w http.ResponseWriter, r *http.Request, principal Principal) bool {
	limiterKey := sessionEnrollmentRateLimitKey(r, principal)
	if !defaultSessionEnrollmentLimiter.allow(limiterKey, time.Now().UTC()) {
		writeError(w, http.StatusTooManyRequests, "enrollment_rate_limited", "too many enrollment attempts; retry later")
		return false
	}
	return true
}

func hasMarketOpsAccess(records []storage.TenantUserAccessRecord) bool {
	for _, record := range records {
		if record.AppID == "marketops" && (record.Permission == "read" || record.Permission == "write") {
			return true
		}
	}
	return false
}

func requiresB2CSubscriptionActivation(cfg RouterConfig, canSelfEnroll bool, records []storage.TenantUserAccessRecord) bool {
	if !cfg.SubscriberB2CRequireSubscription || !canSelfEnroll {
		return false
	}
	for _, record := range records {
		if record.AppID == "marketops" && (record.Permission == "read" || record.Permission == "write") && strings.TrimSpace(record.GrantedBy) == "self-enrollment" {
			return true
		}
	}
	return false
}

func sessionEnrollmentResponse(principal Principal, state string, created []string, access []storage.TenantUserAccessRecord, subscription *storage.SubscriberEffectiveSubscriptionRecord, context *subscriberWatchlistContext, canSelfEnroll bool) map[string]any {
	response := map[string]any{
		"state":          state,
		"tenant_id":      principal.TenantID,
		"subject":        principal.Subject,
		"email":          principal.Email,
		"display_name":   firstNonEmpty(principal.PreferredName, principal.Actor, principal.Email, principal.Subject),
		"email_verified": principal.EmailVerified,
		"self_enrollment": map[string]any{
			"eligible": canSelfEnroll,
			"created":  created,
		},
		"access": map[string]any{"marketops": ""},
	}
	for _, record := range access {
		if record.AppID == "marketops" {
			response["access"] = map[string]any{"marketops": record.Permission}
			break
		}
	}
	if subscription != nil {
		response["subscription"] = effectiveSubscriptionResponse(*subscription)
	} else {
		response["subscription"] = nil
	}
	if context != nil {
		response["watchlist_context"] = subscriberWatchlistContextResponse(*context)
	} else {
		response["watchlist_context"] = nil
	}
	return response
}

func sessionEnrollmentRateLimitKey(r *http.Request, principal Principal) string {
	ip := requestIP(r)
	h := sha256.Sum256([]byte(strings.TrimSpace(principal.TenantID) + "|" + strings.TrimSpace(principal.Subject) + "|" + ip))
	return hex.EncodeToString(h[:])
}

func requestIP(r *http.Request) string {
	for _, header := range []string{"CF-Connecting-IP", "X-Real-IP", "X-Forwarded-For"} {
		value := strings.TrimSpace(r.Header.Get(header))
		if value == "" {
			continue
		}
		if header == "X-Forwarded-For" {
			value = strings.TrimSpace(strings.Split(value, ",")[0])
		}
		if parsed := net.ParseIP(value); parsed != nil {
			return parsed.String()
		}
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

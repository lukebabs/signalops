package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
	"github.com/lukebabs/signalops/internal/subscriber/subscription"
)

// subscriptionFeatureMiddleware is the server-side enforcement boundary for
// commercial capabilities. It runs after JWT authentication, so tenant and
// subject scope come from the signed token rather than browser input. It is
// intentionally feature-specific: ordinary dashboard and public-signal routes
// remain available to Explorer subscribers.
//
// The feature flag is default-off. Enabling it before every affected user has
// an effective subscription fails closed with a clear 402 response, rather than
// silently falling back to a role-only decision.
func subscriptionFeatureMiddleware(next http.Handler, cfg RouterConfig) http.Handler {
	if !cfg.SubscriberSubscriptionsEnabled {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		feature, protected := subscriptionFeatureForRequest(r)
		if !protected {
			next.ServeHTTP(w, r)
			return
		}
		principal, authenticated := principalFromContext(r.Context())
		if !authenticated || strings.TrimSpace(principal.TenantID) == "" || strings.TrimSpace(principal.Subject) == "" {
			writeError(w, http.StatusForbidden, "subscription_identity_required", "an authenticated tenant-scoped subscriber identity is required")
			return
		}
		repository := cfg.SubscriberSubscriptionRepository
		if repository == nil {
			writeError(w, http.StatusServiceUnavailable, "subscription_unavailable", "subscription access is unavailable")
			return
		}
		effective, err := repository.GetSubscriberEffectiveSubscription(r.Context(), principal.TenantID, principal.Subject)
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusPaymentRequired, "subscription_required", "this capability requires an active MarketOps subscription")
			return
		}
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "subscription_unavailable", "subscription access is unavailable")
			return
		}
		if !subscriberProductAllows(effective.Product, feature) {
			writeJSON(w, http.StatusPaymentRequired, map[string]any{
				"error":             "subscription_feature_required",
				"message":           "this capability is not included in the active MarketOps subscription",
				"required_feature":  feature,
				"subscription_tier": effective.Product.ProductKey,
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// subscriptionFeatureForRequest assigns only routes that expose paid or
// Institutional analytical depth. New routes must be mapped here as part of
// their API review; the default is intentionally not an implicit paid grant.
func subscriptionFeatureForRequest(r *http.Request) (subscription.Feature, bool) {
	path := strings.TrimSuffix(strings.TrimSpace(r.URL.Path), "/")
	switch {
	case r.Method == http.MethodPost && (path == "/v1/syncratic/materialize" || path == "/v1/syncratic/daily-narratives/materialize" || strings.HasPrefix(path, "/v1/syncratic/context-windows/") && strings.HasSuffix(path, "/ask")):
		return subscription.FeatureSyncraticExplainability, true
	case strings.HasPrefix(path, "/v1/tenants/") && strings.Contains(path, "/marketops/valuation"):
		return subscription.FeatureValueIntelligence, true
	case strings.HasPrefix(path, "/v1/tenants/") && strings.Contains(path, "/marketops/eroc"):
		return subscription.FeatureDistressedOpportunityIntelligence, true
	case strings.HasPrefix(path, "/v1/tenants/") && strings.Contains(path, "/marketops/earnings-opportunities"):
		return subscription.FeatureEarningsOpportunityIntelligence, true
	case strings.HasPrefix(path, "/v1/tenants/") && strings.Contains(path, "/marketops/") && strings.Contains(path, "/options"):
		return subscription.FeatureOptionsSignals, true
	case strings.HasPrefix(path, "/v1/marketops/signal-assurance"):
		return subscription.FeatureSignalAssuranceAnalytics, true
	case strings.HasPrefix(path, "/v1/marketops/backtest"):
		return subscription.FeatureHistoricalReplay, true
	case strings.HasPrefix(path, "/v1/marketops/sectors/") &&
		!strings.HasPrefix(path, "/v1/marketops/sectors/rankings"):
		return subscription.FeatureSectorRotationDetail, true
	default:
		return "", false
	}
}

func subscriberProductAllows(product storage.SubscriberSubscriptionProductRecord, feature subscription.Feature) bool {
	if _, valid := subscription.ParseTier(product.ProductKey); !valid || !product.Active {
		return false
	}
	features := map[string]bool{}
	if err := json.Unmarshal(product.FeaturePolicyJSON, &features); err != nil {
		return false
	}
	return features[string(feature)]
}

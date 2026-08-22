package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type subscriberSubjectSubscriptionRequest struct {
	TenantID      string `json:"tenant_id"`
	Subject       string `json:"subject"`
	ProductKey    string `json:"product_key"`
	Status        string `json:"status"`
	CorrelationID string `json:"correlation_id"`
}
type subscriberTenantSubscriptionRequest struct {
	TenantID      string `json:"tenant_id"`
	ProductKey    string `json:"product_key"`
	Status        string `json:"status"`
	CorrelationID string `json:"correlation_id"`
}
type subscriberSubscriptionSeatRequest struct {
	TenantID      string `json:"tenant_id"`
	Subject       string `json:"subject"`
	SeatRole      string `json:"seat_role"`
	Status        string `json:"status"`
	CorrelationID string `json:"correlation_id"`
}
type subscriberSubscriptionProductPolicyRequest struct {
	DisplayName   string          `json:"display_name"`
	IsFree        bool            `json:"is_free"`
	TrialDays     int             `json:"trial_days"`
	FeaturePolicy json.RawMessage `json:"feature_policy"`
	LimitPolicy   json.RawMessage `json:"limit_policy"`
	Active        bool            `json:"active"`
	CorrelationID string          `json:"correlation_id"`
}

type subscriberSubscriptionProductBillingRequest struct {
	StripeProductID      string `json:"stripe_product_id"`
	StripeMonthlyPriceID string `json:"stripe_monthly_price_id"`
	StripeAnnualPriceID  string `json:"stripe_annual_price_id"`
	CorrelationID        string `json:"correlation_id"`
}
type subscriberSubjectSubscriptionBillingRequest struct {
	TenantID             string `json:"tenant_id"`
	Subject              string `json:"subject"`
	StripeCustomerID     string `json:"stripe_customer_id"`
	StripeSubscriptionID string `json:"stripe_subscription_id"`
	Status               string `json:"status"`
	CurrentPeriodEndsAt  string `json:"current_period_ends_at"`
	GraceEndsAt          string `json:"grace_ends_at"`
	CanceledAt           string `json:"canceled_at"`
	CorrelationID        string `json:"correlation_id"`
}
type subscriberTenantSubscriptionBillingRequest struct {
	TenantID             string `json:"tenant_id"`
	StripeCustomerID     string `json:"stripe_customer_id"`
	StripeSubscriptionID string `json:"stripe_subscription_id"`
	Status               string `json:"status"`
	CurrentPeriodEndsAt  string `json:"current_period_ends_at"`
	GraceEndsAt          string `json:"grace_ends_at"`
	CanceledAt           string `json:"canceled_at"`
	CorrelationID        string `json:"correlation_id"`
}

// registerSubscriberSubscriptionAdministrationRoutes provides the controlled
// provisioning boundary for launch operations. It deliberately has no browser
// self-service upgrade operation: a caller needs the dedicated platform role
// signalops:subscription_admin (or an existing platform-super-admin role).
func registerSubscriberSubscriptionAdministrationRoutes(mux *http.ServeMux, cfg RouterConfig) {
	repository := cfg.SubscriberSubscriptionAdministrationRepository
	if repository == nil {
		return
	}

	mux.HandleFunc("GET /v1/administration/subscriptions", func(w http.ResponseWriter, r *http.Request) {
		_, ok := requireSubscriptionAdministrator(w, r)
		if !ok {
			return
		}
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if tenantID == "" {
			if principal, authenticated := principalFromContext(r.Context()); authenticated {
				tenantID = strings.TrimSpace(principal.TenantID)
			}
		}
		snapshot, err := repository.ListSubscriberSubscriptionAdministration(r.Context(), storage.SubscriberSubscriptionAdministrationFilter{TenantID: tenantID})
		if err != nil {
			writeSubscriberSubscriptionAdministrationError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, subscriberSubscriptionAdministrationResponse(snapshot))
	})
	mux.HandleFunc("GET /v1/administration/subscriptions/products", func(w http.ResponseWriter, r *http.Request) {
		_, ok := requireSubscriptionAdministrator(w, r)
		if !ok {
			return
		}
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if tenantID == "" {
			if principal, authenticated := principalFromContext(r.Context()); authenticated {
				tenantID = strings.TrimSpace(principal.TenantID)
			}
		}
		snapshot, err := repository.ListSubscriberSubscriptionAdministration(r.Context(), storage.SubscriberSubscriptionAdministrationFilter{TenantID: tenantID})
		if err != nil {
			writeSubscriberSubscriptionAdministrationError(w, err)
			return
		}
		products := make([]map[string]any, 0, len(snapshot.Products))
		for _, product := range snapshot.Products {
			products = append(products, subscriptionAdministrationProductResponse(product))
		}
		writeJSON(w, http.StatusOK, map[string]any{"products": products})
	})
	mux.HandleFunc("PUT /v1/administration/subscriptions/products/{product_key}", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := requireSubscriptionAdministrator(w, r)
		if !ok {
			return
		}
		principal, _ := principalFromContext(r.Context())
		var request subscriberSubscriptionProductPolicyRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		featurePolicy := request.FeaturePolicy
		if len(featurePolicy) == 0 {
			featurePolicy = json.RawMessage(`{}`)
		}
		limitPolicy := request.LimitPolicy
		if len(limitPolicy) == 0 {
			limitPolicy = json.RawMessage(`{}`)
		}
		if err := repository.UpdateSubscriberSubscriptionProduct(r.Context(), storage.SubscriberSubscriptionProductMutation{
			TenantID: strings.TrimSpace(principal.TenantID), ProductKey: strings.TrimSpace(r.PathValue("product_key")), DisplayName: strings.TrimSpace(request.DisplayName), IsFree: request.IsFree,
			TrialDays: request.TrialDays, FeaturePolicyJSON: []byte(featurePolicy), LimitPolicyJSON: []byte(limitPolicy), Active: request.Active,
			ActorSubject: actor, CorrelationID: subscriptionCorrelationID(r, request.CorrelationID),
		}); err != nil {
			writeSubscriberSubscriptionAdministrationError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	})
	mux.HandleFunc("PUT /v1/administration/subscriptions/products/{product_key}/billing", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := requireSubscriptionAdministrator(w, r)
		if !ok {
			return
		}
		principal, _ := principalFromContext(r.Context())
		var request subscriberSubscriptionProductBillingRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		if err := repository.UpdateSubscriberSubscriptionProductBilling(r.Context(), storage.SubscriberSubscriptionProductBillingMutation{
			TenantID: strings.TrimSpace(principal.TenantID), ProductKey: strings.TrimSpace(r.PathValue("product_key")),
			StripeProductID: strings.TrimSpace(request.StripeProductID), StripeMonthlyPriceID: strings.TrimSpace(request.StripeMonthlyPriceID), StripeAnnualPriceID: strings.TrimSpace(request.StripeAnnualPriceID),
			ActorSubject: actor, CorrelationID: subscriptionCorrelationID(r, request.CorrelationID),
		}); err != nil {
			writeSubscriberSubscriptionAdministrationError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	})
	mux.HandleFunc("POST /v1/administration/subscriptions/subject", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := requireSubscriptionAdministrator(w, r)
		if !ok {
			return
		}
		var request subscriberSubjectSubscriptionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		if err := repository.UpsertSubscriberSubjectSubscription(r.Context(), storage.SubscriberSubjectSubscriptionMutation{
			TenantID: strings.TrimSpace(request.TenantID), Subject: strings.TrimSpace(request.Subject), ProductKey: strings.TrimSpace(request.ProductKey), Status: strings.TrimSpace(request.Status), ActorSubject: actor, CorrelationID: subscriptionCorrelationID(r, request.CorrelationID),
		}); err != nil {
			writeSubscriberSubscriptionAdministrationError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "provisioned"})
	})
	mux.HandleFunc("PUT /v1/administration/subscriptions/subject/billing", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := requireSubscriptionAdministrator(w, r)
		if !ok {
			return
		}
		var request subscriberSubjectSubscriptionBillingRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		currentPeriodEndsAt, graceEndsAt, canceledAt, parseErr := parseSubscriptionBillingTimes(request.CurrentPeriodEndsAt, request.GraceEndsAt, request.CanceledAt)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "invalid_subscription_request", parseErr.Error())
			return
		}
		if err := repository.UpdateSubscriberSubjectSubscriptionBilling(r.Context(), storage.SubscriberSubjectSubscriptionBillingMutation{
			TenantID: strings.TrimSpace(request.TenantID), Subject: strings.TrimSpace(request.Subject), StripeCustomerID: strings.TrimSpace(request.StripeCustomerID), StripeSubscriptionID: strings.TrimSpace(request.StripeSubscriptionID), Status: strings.TrimSpace(request.Status),
			CurrentPeriodEndsAt: currentPeriodEndsAt, GraceEndsAt: graceEndsAt, CanceledAt: canceledAt,
			ActorSubject: actor, CorrelationID: subscriptionCorrelationID(r, request.CorrelationID),
		}); err != nil {
			writeSubscriberSubscriptionAdministrationError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	})
	mux.HandleFunc("POST /v1/administration/subscriptions/tenant", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := requireSubscriptionAdministrator(w, r)
		if !ok {
			return
		}
		var request subscriberTenantSubscriptionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		if err := repository.UpsertSubscriberTenantSubscription(r.Context(), storage.SubscriberTenantSubscriptionMutation{
			TenantID: strings.TrimSpace(request.TenantID), ProductKey: strings.TrimSpace(request.ProductKey), Status: strings.TrimSpace(request.Status), ActorSubject: actor, CorrelationID: subscriptionCorrelationID(r, request.CorrelationID),
		}); err != nil {
			writeSubscriberSubscriptionAdministrationError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "provisioned"})
	})
	mux.HandleFunc("PUT /v1/administration/subscriptions/tenant/billing", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := requireSubscriptionAdministrator(w, r)
		if !ok {
			return
		}
		var request subscriberTenantSubscriptionBillingRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		currentPeriodEndsAt, graceEndsAt, canceledAt, parseErr := parseSubscriptionBillingTimes(request.CurrentPeriodEndsAt, request.GraceEndsAt, request.CanceledAt)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "invalid_subscription_request", parseErr.Error())
			return
		}
		if err := repository.UpdateSubscriberTenantSubscriptionBilling(r.Context(), storage.SubscriberTenantSubscriptionBillingMutation{
			TenantID: strings.TrimSpace(request.TenantID), StripeCustomerID: strings.TrimSpace(request.StripeCustomerID), StripeSubscriptionID: strings.TrimSpace(request.StripeSubscriptionID), Status: strings.TrimSpace(request.Status),
			CurrentPeriodEndsAt: currentPeriodEndsAt, GraceEndsAt: graceEndsAt, CanceledAt: canceledAt,
			ActorSubject: actor, CorrelationID: subscriptionCorrelationID(r, request.CorrelationID),
		}); err != nil {
			writeSubscriberSubscriptionAdministrationError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	})
	mux.HandleFunc("PUT /v1/administration/subscriptions/seats", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := requireSubscriptionAdministrator(w, r)
		if !ok {
			return
		}
		var request subscriberSubscriptionSeatRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		if err := repository.UpsertSubscriberSubscriptionSeat(r.Context(), storage.SubscriberSubscriptionSeatMutation{
			TenantID: strings.TrimSpace(request.TenantID), Subject: strings.TrimSpace(request.Subject), SeatRole: strings.TrimSpace(request.SeatRole), Status: strings.TrimSpace(request.Status), ActorSubject: actor, CorrelationID: subscriptionCorrelationID(r, request.CorrelationID),
		}); err != nil {
			writeSubscriberSubscriptionAdministrationError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "provisioned"})
	})
}

func requireSubscriptionAdministrator(w http.ResponseWriter, r *http.Request) (string, bool) {
	principal, authenticated := principalFromContext(r.Context())
	if !authenticated || !isSubscriptionAdministrator(principal) || strings.TrimSpace(principal.Subject) == "" {
		writeError(w, http.StatusForbidden, "subscription_admin_required", "platform subscription administrator role is required")
		return "", false
	}
	return strings.TrimSpace(principal.Subject), true
}
func subscriptionCorrelationID(r *http.Request, requested string) string {
	return firstNonEmpty(strings.TrimSpace(requested), headerValue(r, "X-Correlation-ID"), newID("subcorr"))
}
func parseSubscriptionBillingTimes(currentPeriodEnd, graceEnd, canceled string) (*time.Time, *time.Time, *time.Time, error) {
	current, err := parseOptionalSubscriptionTime("current_period_ends_at", currentPeriodEnd)
	if err != nil {
		return nil, nil, nil, err
	}
	grace, err := parseOptionalSubscriptionTime("grace_ends_at", graceEnd)
	if err != nil {
		return nil, nil, nil, err
	}
	canceledAt, err := parseOptionalSubscriptionTime("canceled_at", canceled)
	if err != nil {
		return nil, nil, nil, err
	}
	return current, grace, canceledAt, nil
}
func parseOptionalSubscriptionTime(field, raw string) (*time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, fmt.Errorf("%s must be RFC3339", field)
	}
	return &parsed, nil
}
func writeSubscriberSubscriptionAdministrationError(w http.ResponseWriter, err error) {
	if err == storage.ErrNotFound {
		writeError(w, http.StatusNotFound, "subscription_target_not_found", "subscription product or tenant contract was not found")
		return
	}
	if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "must use") {
		writeError(w, http.StatusBadRequest, "invalid_subscription_request", err.Error())
		return
	}
	writeError(w, http.StatusServiceUnavailable, "subscription_provisioning_failed", "subscription provisioning could not be completed")
}

func subscriberSubscriptionAdministrationResponse(snapshot storage.SubscriberSubscriptionAdministrationSnapshot) map[string]any {
	products := make([]map[string]any, 0, len(snapshot.Products))
	for _, product := range snapshot.Products {
		products = append(products, subscriptionAdministrationProductResponse(product))
	}
	subjects := make([]map[string]any, 0, len(snapshot.SubjectSubscriptions))
	for _, record := range snapshot.SubjectSubscriptions {
		subjects = append(subjects, map[string]any{
			"tenant_id": record.TenantID, "subject": record.Subject, "subscription_id": record.SubscriptionID,
			"product_key": record.ProductKey, "display_name": record.DisplayName, "status": record.Status,
			"trial_ends_at": record.TrialEndsAt, "current_period_ends_at": record.CurrentPeriodEndsAt,
			"grace_ends_at": record.GraceEndsAt, "canceled_at": record.CanceledAt,
			"stripe_customer_id": record.StripeCustomerID, "stripe_subscription_id": record.StripeSubscriptionID,
			"provisioned_by": record.ProvisionedBy, "correlation_id": record.CorrelationID,
			"created_at": record.CreatedAt, "updated_at": record.UpdatedAt,
		})
	}
	contracts := make([]map[string]any, 0, len(snapshot.TenantSubscriptions))
	for _, record := range snapshot.TenantSubscriptions {
		contracts = append(contracts, map[string]any{
			"tenant_id": record.TenantID, "subscription_id": record.SubscriptionID,
			"product_key": record.ProductKey, "display_name": record.DisplayName, "status": record.Status,
			"current_period_ends_at": record.CurrentPeriodEndsAt, "grace_ends_at": record.GraceEndsAt,
			"canceled_at": record.CanceledAt, "stripe_customer_id": record.StripeCustomerID, "stripe_subscription_id": record.StripeSubscriptionID, "provisioned_by": record.ProvisionedBy,
			"correlation_id": record.CorrelationID, "created_at": record.CreatedAt, "updated_at": record.UpdatedAt,
		})
	}
	seats := make([]map[string]any, 0, len(snapshot.Seats))
	for _, record := range snapshot.Seats {
		seats = append(seats, map[string]any{
			"tenant_id": record.TenantID, "subject": record.Subject, "tenant_subscription_id": record.TenantSubscriptionID,
			"seat_role": record.SeatRole, "status": record.Status, "assigned_by": record.AssignedBy,
			"correlation_id": record.CorrelationID, "assigned_at": record.AssignedAt, "revoked_at": record.RevokedAt,
		})
	}
	audit := make([]map[string]any, 0, len(snapshot.AuditEvents))
	for _, record := range snapshot.AuditEvents {
		audit = append(audit, map[string]any{
			"audit_id": record.AuditID, "tenant_id": record.TenantID, "subject": record.Subject,
			"subscription_id": record.SubscriptionID, "actor_subject": record.ActorSubject,
			"event_type": record.EventType, "before_state": subscriptionJSON(record.BeforeStateJSON),
			"after_state": subscriptionJSON(record.AfterStateJSON), "correlation_id": record.CorrelationID,
			"occurred_at": record.OccurredAt,
		})
	}
	webhooks := make([]map[string]any, 0, len(snapshot.BillingWebhookEvents))
	for _, record := range snapshot.BillingWebhookEvents {
		webhooks = append(webhooks, map[string]any{
			"provider_event_id": record.ProviderEventID, "event_type": record.EventType, "processing_status": record.ProcessingStatus,
			"error_message": record.ErrorMessage, "received_at": record.ReceivedAt, "processed_at": record.ProcessedAt,
		})
	}
	return map[string]any{"tenant_id": snapshot.TenantID, "products": products, "subject_subscriptions": subjects, "tenant_subscriptions": contracts, "seats": seats, "audit_events": audit, "billing_webhook_events": webhooks}
}

func subscriptionAdministrationProductResponse(product storage.SubscriberSubscriptionProductRecord) map[string]any {
	response := subscriptionProductResponse(product)
	response["active"] = product.Active
	response["changed_by"] = product.ChangedBy
	response["created_at"] = product.CreatedAt
	response["updated_at"] = product.UpdatedAt
	response["stripe_product_id"] = product.StripeProductID
	response["stripe_monthly_price_id"] = product.StripeMonthlyPriceID
	response["stripe_annual_price_id"] = product.StripeAnnualPriceID
	return response
}

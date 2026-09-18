package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const stripeWebhookTolerance = 5 * time.Minute

type stripeWebhookEvent struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		Object json.RawMessage `json:"object"`
	} `json:"data"`
}

type stripeSubscriptionObject struct {
	ID               string            `json:"id"`
	Metadata         map[string]string `json:"metadata"`
	Customer         any               `json:"customer"`
	Status           string            `json:"status"`
	CurrentPeriodEnd int64             `json:"current_period_end"`
	TrialEnd         int64             `json:"trial_end"`
	CanceledAt       int64             `json:"canceled_at"`
	CancelAt         int64             `json:"cancel_at"`
}

type stripeInvoiceObject struct {
	ID           string            `json:"id"`
	Metadata     map[string]string `json:"metadata"`
	Customer     any               `json:"customer"`
	Subscription any               `json:"subscription"`
	PeriodEnd    int64             `json:"period_end"`
}

type stripeCheckoutSessionWebhookObject struct {
	ID            string            `json:"id"`
	Metadata      map[string]string `json:"metadata"`
	Customer      any               `json:"customer"`
	Subscription  any               `json:"subscription"`
	Status        string            `json:"status"`
	PaymentStatus string            `json:"payment_status"`
}

func registerSubscriberStripeWebhookRoutes(mux *http.ServeMux, cfg RouterConfig) {
	repository := cfg.SubscriberSubscriptionAdministrationRepository
	if repository == nil {
		return
	}
	mux.HandleFunc("POST /v1/billing/stripe/webhook", func(w http.ResponseWriter, r *http.Request) {
		secret := strings.TrimSpace(cfg.StripeWebhookSecret)
		if secret == "" {
			writeError(w, http.StatusServiceUnavailable, "stripe_webhook_disabled", "stripe webhook secret is not configured")
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_webhook_body", "failed to read webhook body")
			return
		}
		if err := verifyStripeSignature(r.Header.Get("Stripe-Signature"), body, secret, time.Now().UTC()); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_stripe_signature", err.Error())
			return
		}
		mutation, err := stripeWebhookMutation(body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "unsupported_stripe_event", err.Error())
			return
		}
		record, err := repository.ProcessSubscriberStripeWebhook(r.Context(), mutation)
		if err != nil {
			slog.Warn("stripe webhook reconciliation failed", "provider_event_id", mutation.ProviderEventID, "event_type", mutation.EventType, "stripe_subscription_id", mutation.StripeSubscriptionID, "error", err)
			writeError(w, http.StatusServiceUnavailable, "stripe_webhook_processing_failed", "stripe webhook could not be reconciled")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": record.ProcessingStatus, "provider_event_id": record.ProviderEventID, "event_type": record.EventType})
	})
}

func verifyStripeSignature(header string, payload []byte, secret string, now time.Time) error {
	parts := strings.Split(header, ",")
	var timestamp string
	signatures := []string{}
	for _, part := range parts {
		keyValue := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(keyValue) != 2 {
			continue
		}
		switch keyValue[0] {
		case "t":
			timestamp = keyValue[1]
		case "v1":
			signatures = append(signatures, keyValue[1])
		}
	}
	if timestamp == "" || len(signatures) == 0 {
		return errors.New("missing Stripe timestamp or v1 signature")
	}
	seconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return errors.New("invalid Stripe timestamp")
	}
	eventTime := time.Unix(seconds, 0).UTC()
	if now.Sub(eventTime) > stripeWebhookTolerance || eventTime.Sub(now) > stripeWebhookTolerance {
		return errors.New("Stripe signature timestamp is outside tolerance")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp + "."))
	_, _ = mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	for _, signature := range signatures {
		if hmac.Equal([]byte(expected), []byte(signature)) {
			return nil
		}
	}
	return errors.New("Stripe signature mismatch")
}

func stripeWebhookMutation(payload []byte) (storage.SubscriberStripeWebhookMutation, error) {
	var event stripeWebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return storage.SubscriberStripeWebhookMutation{}, fmt.Errorf("invalid Stripe event JSON: %w", err)
	}
	if strings.TrimSpace(event.ID) == "" || strings.TrimSpace(event.Type) == "" {
		return storage.SubscriberStripeWebhookMutation{}, errors.New("Stripe event id and type are required")
	}
	mutation := storage.SubscriberStripeWebhookMutation{ProviderEventID: strings.TrimSpace(event.ID), EventType: strings.TrimSpace(event.Type), PayloadJSON: payload}
	switch event.Type {
	case "checkout.session.completed":
		var object stripeCheckoutSessionWebhookObject
		if err := json.Unmarshal(event.Data.Object, &object); err != nil {
			return mutation, fmt.Errorf("invalid Stripe checkout session object: %w", err)
		}
		mutation.StripeSubscriptionID = stripeStringID(object.Subscription)
		mutation.StripeCustomerID = stripeStringID(object.Customer)
		mutation.CheckoutRef = strings.TrimSpace(object.Metadata["checkout_ref"])
		if strings.TrimSpace(object.Status) == "complete" && strings.TrimSpace(object.PaymentStatus) == "paid" {
			mutation.Status = storage.SubscriberSubscriptionActive
		} else {
			mutation.Status = storage.SubscriberSubscriptionPastDue
			grace := time.Now().UTC().Add(7 * 24 * time.Hour)
			mutation.GraceEndsAt = &grace
		}
	case "customer.subscription.created", "customer.subscription.updated", "customer.subscription.deleted":
		var object stripeSubscriptionObject
		if err := json.Unmarshal(event.Data.Object, &object); err != nil {
			return mutation, fmt.Errorf("invalid Stripe subscription object: %w", err)
		}
		mutation.StripeSubscriptionID = strings.TrimSpace(object.ID)
		mutation.StripeCustomerID = stripeStringID(object.Customer)
		mutation.Status = mapStripeSubscriptionStatus(object.Status)
		mutation.CurrentPeriodEndsAt = unixPtr(object.CurrentPeriodEnd)
		if mutation.Status == storage.SubscriberSubscriptionPastDue {
			grace := time.Now().UTC().Add(7 * 24 * time.Hour)
			mutation.GraceEndsAt = &grace
		}
		mutation.CanceledAt = unixPtr(firstPositiveInt64(object.CanceledAt, object.CancelAt))
		mutation.CheckoutRef = strings.TrimSpace(object.Metadata["checkout_ref"])
		if event.Type == "customer.subscription.deleted" {
			mutation.Status = storage.SubscriberSubscriptionCanceled
			if mutation.CanceledAt == nil {
				now := time.Now().UTC()
				mutation.CanceledAt = &now
			}
		}
	case "invoice.payment_succeeded", "invoice.payment_failed":
		var object stripeInvoiceObject
		if err := json.Unmarshal(event.Data.Object, &object); err != nil {
			return mutation, fmt.Errorf("invalid Stripe invoice object: %w", err)
		}
		mutation.StripeSubscriptionID = stripeStringID(object.Subscription)
		mutation.StripeCustomerID = stripeStringID(object.Customer)
		mutation.CurrentPeriodEndsAt = unixPtr(object.PeriodEnd)
		mutation.CheckoutRef = strings.TrimSpace(object.Metadata["checkout_ref"])
		if event.Type == "invoice.payment_failed" {
			mutation.Status = storage.SubscriberSubscriptionPastDue
			grace := time.Now().UTC().Add(7 * 24 * time.Hour)
			mutation.GraceEndsAt = &grace
		} else {
			mutation.Status = storage.SubscriberSubscriptionActive
		}
	default:
		return mutation, fmt.Errorf("unsupported Stripe event type %q", event.Type)
	}
	return mutation, nil
}

func mapStripeSubscriptionStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "trialing":
		return storage.SubscriberSubscriptionTrialing
	case "active":
		return storage.SubscriberSubscriptionActive
	case "past_due":
		return storage.SubscriberSubscriptionPastDue
	case "canceled":
		return storage.SubscriberSubscriptionCanceled
	default:
		return storage.SubscriberSubscriptionSuspended
	}
}
func stripeStringID(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		if id, ok := typed["id"].(string); ok {
			return strings.TrimSpace(id)
		}
	}
	return ""
}
func unixPtr(seconds int64) *time.Time {
	if seconds <= 0 {
		return nil
	}
	value := time.Unix(seconds, 0).UTC()
	return &value
}
func firstPositiveInt64(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

package storage

import (
	"context"
	"time"
)

const (
	SubscriberSubscriptionTrialing  = "trialing"
	SubscriberSubscriptionActive    = "active"
	SubscriberSubscriptionPastDue   = "past_due"
	SubscriberSubscriptionSuspended = "suspended"
	SubscriberSubscriptionCanceled  = "canceled"
)

type SubscriberSubscriptionProductRecord struct {
	ProductKey           string
	BillingScope         string
	DisplayName          string
	IsFree               bool
	TrialDays            int
	StripeProductID      string
	StripeMonthlyPriceID string
	StripeAnnualPriceID  string
	FeaturePolicyJSON    []byte
	LimitPolicyJSON      []byte
	Revision             int
	Active               bool
	ChangedBy            string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type SubscriberEffectiveSubscriptionRecord struct {
	TenantID            string
	Subject             string
	SubscriptionID      string
	Product             SubscriberSubscriptionProductRecord
	Status              string
	Source              string // subject or tenant_seat
	SeatRole            string
	TrialEndsAt         *time.Time
	CurrentPeriodEndsAt *time.Time
	GraceEndsAt         *time.Time
	CanceledAt          *time.Time
}

// SubscriberSubscriptionRepository deliberately carries commercial access only.
// It is separate from SubscriberEntitlementRepository, whose capabilities govern
// central collection and provider-cost decisions.
type SubscriberSubscriptionRepository interface {
	ListSubscriberSubscriptionProducts(context.Context) ([]SubscriberSubscriptionProductRecord, error)
	GetSubscriberEffectiveSubscription(ctx context.Context, tenantID, subject string) (SubscriberEffectiveSubscriptionRecord, error)
}

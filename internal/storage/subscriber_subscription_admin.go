package storage

import (
	"context"
	"time"
)

// SubscriberSubscriptionAdministrationRepository is deliberately distinct from
// read-side subscription resolution. Only a platform subscription administrator
// may invoke these operations; browser subscribers and tenant administrators
// cannot self-upgrade a product or create a tenant contract.
type SubscriberSubscriptionAdministrationRepository interface {
	ListSubscriberSubscriptionAdministration(context.Context, SubscriberSubscriptionAdministrationFilter) (SubscriberSubscriptionAdministrationSnapshot, error)
	UpdateSubscriberSubscriptionProduct(context.Context, SubscriberSubscriptionProductMutation) error
	UpdateSubscriberSubscriptionProductBilling(context.Context, SubscriberSubscriptionProductBillingMutation) error
	UpsertSubscriberSubjectSubscription(context.Context, SubscriberSubjectSubscriptionMutation) error
	UpdateSubscriberSubjectSubscriptionBilling(context.Context, SubscriberSubjectSubscriptionBillingMutation) error
	UpsertSubscriberTenantSubscription(context.Context, SubscriberTenantSubscriptionMutation) error
	UpdateSubscriberTenantSubscriptionBilling(context.Context, SubscriberTenantSubscriptionBillingMutation) error
	UpsertSubscriberSubscriptionSeat(context.Context, SubscriberSubscriptionSeatMutation) error
	ProcessSubscriberStripeWebhook(context.Context, SubscriberStripeWebhookMutation) (SubscriberBillingWebhookEventRecord, error)
}

type SubscriberSubjectSubscriptionMutation struct {
	TenantID      string
	Subject       string
	ProductKey    string
	Status        string
	ActorSubject  string
	CorrelationID string
}

type SubscriberSubjectSubscriptionBillingMutation struct {
	TenantID             string
	Subject              string
	StripeCustomerID     string
	StripeSubscriptionID string
	Status               string
	CurrentPeriodEndsAt  *time.Time
	GraceEndsAt          *time.Time
	CanceledAt           *time.Time
	ActorSubject         string
	CorrelationID        string
}

type SubscriberTenantSubscriptionMutation struct {
	TenantID      string
	ProductKey    string
	Status        string
	ActorSubject  string
	CorrelationID string
}

type SubscriberTenantSubscriptionBillingMutation struct {
	TenantID             string
	StripeCustomerID     string
	StripeSubscriptionID string
	Status               string
	CurrentPeriodEndsAt  *time.Time
	GraceEndsAt          *time.Time
	CanceledAt           *time.Time
	ActorSubject         string
	CorrelationID        string
}

type SubscriberSubscriptionSeatMutation struct {
	TenantID      string
	Subject       string
	SeatRole      string
	Status        string
	ActorSubject  string
	CorrelationID string
}

type SubscriberSubscriptionAdministrationFilter struct {
	TenantID string
}

type SubscriberSubscriptionAdministrationSnapshot struct {
	TenantID             string
	Products             []SubscriberSubscriptionProductRecord
	SubjectSubscriptions []SubscriberSubjectSubscriptionRecord
	TenantSubscriptions  []SubscriberTenantSubscriptionRecord
	Seats                []SubscriberSubscriptionSeatRecord
	AuditEvents          []SubscriberSubscriptionAuditEventRecord
	BillingWebhookEvents []SubscriberBillingWebhookEventRecord
}

type SubscriberSubjectSubscriptionRecord struct {
	TenantID             string
	Subject              string
	SubjectDisplayName   string
	SubjectEmail         string
	SubscriptionID       string
	ProductKey           string
	DisplayName          string
	Status               string
	TrialEndsAt          *time.Time
	CurrentPeriodEndsAt  *time.Time
	GraceEndsAt          *time.Time
	CanceledAt           *time.Time
	StripeCustomerID     string
	StripeSubscriptionID string
	ProvisionedBy        string
	CorrelationID        string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type SubscriberTenantSubscriptionRecord struct {
	TenantID             string
	SubscriptionID       string
	ProductKey           string
	DisplayName          string
	Status               string
	CurrentPeriodEndsAt  *time.Time
	GraceEndsAt          *time.Time
	CanceledAt           *time.Time
	StripeCustomerID     string
	StripeSubscriptionID string
	ProvisionedBy        string
	CorrelationID        string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type SubscriberSubscriptionSeatRecord struct {
	TenantID              string
	Subject               string
	SubjectDisplayName    string
	SubjectEmail          string
	TenantSubscriptionID  string
	SeatRole              string
	Status                string
	AssignedBy            string
	AssignedByDisplayName string
	AssignedByEmail       string
	CorrelationID         string
	AssignedAt            time.Time
	RevokedAt             *time.Time
}

type SubscriberSubscriptionAuditEventRecord struct {
	AuditID            string
	TenantID           string
	Subject            string
	SubjectDisplayName string
	SubjectEmail       string
	SubscriptionID     string
	ActorSubject       string
	ActorDisplayName   string
	ActorEmail         string
	EventType          string
	BeforeStateJSON    []byte
	AfterStateJSON     []byte
	CorrelationID      string
	OccurredAt         time.Time
}

type SubscriberSubscriptionProductMutation struct {
	TenantID          string
	ProductKey        string
	DisplayName       string
	IsFree            bool
	TrialDays         int
	FeaturePolicyJSON []byte
	LimitPolicyJSON   []byte
	Active            bool
	ActorSubject      string
	CorrelationID     string
}

type SubscriberSubscriptionProductBillingMutation struct {
	TenantID             string
	ProductKey           string
	StripeProductID      string
	StripeMonthlyPriceID string
	StripeAnnualPriceID  string
	ActorSubject         string
	CorrelationID        string
}

type SubscriberStripeWebhookMutation struct {
	ProviderEventID      string
	EventType            string
	PayloadJSON          []byte
	StripeCustomerID     string
	StripeSubscriptionID string
	Status               string
	CurrentPeriodEndsAt  *time.Time
	GraceEndsAt          *time.Time
	CanceledAt           *time.Time
}

type SubscriberBillingWebhookEventRecord struct {
	ProviderEventID  string
	EventType        string
	ProcessingStatus string
	ErrorMessage     string
	ReceivedAt       time.Time
	ProcessedAt      *time.Time
}

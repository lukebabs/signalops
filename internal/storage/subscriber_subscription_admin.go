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
	UpsertSubscriberSubjectSubscription(context.Context, SubscriberSubjectSubscriptionMutation) error
	UpsertSubscriberTenantSubscription(context.Context, SubscriberTenantSubscriptionMutation) error
	UpsertSubscriberSubscriptionSeat(context.Context, SubscriberSubscriptionSeatMutation) error
}

type SubscriberSubjectSubscriptionMutation struct {
	TenantID      string
	Subject       string
	ProductKey    string
	Status        string
	ActorSubject  string
	CorrelationID string
}

type SubscriberTenantSubscriptionMutation struct {
	TenantID      string
	ProductKey    string
	Status        string
	ActorSubject  string
	CorrelationID string
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
}

type SubscriberSubjectSubscriptionRecord struct {
	TenantID            string
	Subject             string
	SubscriptionID      string
	ProductKey          string
	DisplayName         string
	Status              string
	TrialEndsAt         *time.Time
	CurrentPeriodEndsAt *time.Time
	GraceEndsAt         *time.Time
	CanceledAt          *time.Time
	ProvisionedBy       string
	CorrelationID       string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type SubscriberTenantSubscriptionRecord struct {
	TenantID            string
	SubscriptionID      string
	ProductKey          string
	DisplayName         string
	Status              string
	CurrentPeriodEndsAt *time.Time
	GraceEndsAt         *time.Time
	CanceledAt          *time.Time
	ProvisionedBy       string
	CorrelationID       string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type SubscriberSubscriptionSeatRecord struct {
	TenantID             string
	Subject              string
	TenantSubscriptionID string
	SeatRole             string
	Status               string
	AssignedBy           string
	CorrelationID        string
	AssignedAt           time.Time
	RevokedAt            *time.Time
}

type SubscriberSubscriptionAuditEventRecord struct {
	AuditID         string
	TenantID        string
	Subject         string
	SubscriptionID  string
	ActorSubject    string
	EventType       string
	BeforeStateJSON []byte
	AfterStateJSON  []byte
	CorrelationID   string
	OccurredAt      time.Time
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

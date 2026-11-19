package storage

import (
	"context"
	"time"
)

const (
	SubscriberEntitlementActive    = "active"
	SubscriberEntitlementSuspended = "suspended"

	SubscriberQuotaReserved = "reserved"
	SubscriberQuotaConsumed = "consumed"
	SubscriberQuotaReleased = "released"
)

type SubscriberEntitlementCapabilityRecord struct {
	Capability string
	Enabled    bool
	QuotaLimit int
}

type SubscriberEntitlementRecord struct {
	TenantID            string
	ProvisioningVersion string
	ProductTier         string
	Status              string
	ProvisionedBy       string
	CorrelationID       string
	Capabilities        []SubscriberEntitlementCapabilityRecord
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type SubscriberQuotaReservationRequest struct {
	TenantID       string
	Subject        string
	Capability     string
	RequestedUnits int
	IdempotencyKey string
	CorrelationID  string
	RequestedAt    time.Time
}

type SubscriberQuotaReservationRecord struct {
	ReservationID       string
	TenantID            string
	Capability          string
	ProvisioningVersion string
	IdempotencyKey      string
	Subject             string
	RequestedUnits      int
	Status              string
	PolicyVersion       string
	CorrelationID       string
	ReservedAt          time.Time
	ReleasedAt          *time.Time
}

type SubscriberEntitlementDecisionRecord struct {
	DecisionID         string
	TenantID           string
	Subject            string
	Capability         string
	DecisionReason     string
	RequestedUnits     int
	ConsumedUnits      int
	QuotaLimit         int
	EntitlementVersion string
	PolicyVersion      string
	CorrelationID      string
	DecisionAt         time.Time
	ProvenanceJSON     []byte
}

type SubscriberQuotaReservationLifecycleRequest struct {
	TenantID      string
	ReservationID string
	ActorSubject  string
	Transition    string
	CorrelationID string
	OccurredAt    time.Time
}

type SubscriberEntitlementRepository interface {
	UpsertSubscriberEntitlement(context.Context, SubscriberEntitlementRecord) (SubscriberEntitlementRecord, error)
	GetSubscriberEntitlement(context.Context, string) (SubscriberEntitlementRecord, error)
	ReserveSubscriberQuota(context.Context, SubscriberQuotaReservationRequest) (SubscriberQuotaReservationRecord, SubscriberEntitlementDecisionRecord, error)
	FinalizeSubscriberQuotaReservation(context.Context, SubscriberQuotaReservationLifecycleRequest) (SubscriberQuotaReservationRecord, error)
}

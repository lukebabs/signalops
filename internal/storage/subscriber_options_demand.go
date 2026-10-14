package storage

import (
	"context"
	"time"
)

const SubscriberOptionsDemandShadowVersion = "s6-options-demand-shadow-v1"

// SubscriberOptionsDemandAggregate is deliberately non-identifying output
// from the restricted database projection.
type SubscriberOptionsDemandAggregate struct {
	GlobalAssetID        string
	HighestTierRank      int
	EligibleTenantCount  int
	EligibleWatcherCount int
	DeferredSessions     int
}

type SubscriberOptionsDemandSnapshotMember struct {
	SubscriberOptionsDemandAggregate
	Priority       int
	SelectionState string
}

type SubscriberOptionsDemandSnapshot struct {
	SnapshotRunID     string
	PlannerVersion    string
	SessionDate       time.Time
	MaxSymbols        int
	SourceDemandCount int
	Members           []SubscriberOptionsDemandSnapshotMember
	PlannedBy         string
	CorrelationID     string
	PlannedAt         time.Time
}

type SubscriberOptionsDemandPlannerRepository interface {
	ListSubscriberOptionsDemandAggregates(context.Context) ([]SubscriberOptionsDemandAggregate, error)
	RecordSubscriberOptionsDemandShadowSnapshot(context.Context, SubscriberOptionsDemandSnapshot) (SubscriberOptionsDemandSnapshot, error)
}

package storage

import (
	"context"
	"time"
)

const SubscriberGlobalEODPlannerShadowVersion = "s2-eod-hot-set-shadow-v1"

type SubscriberGlobalEODHotSetCandidate struct {
	GlobalAssetID     string
	EligibilityStatus string
	ActiveSourceRows  int
	BestSourceRank    int
}

type SubscriberGlobalEODHotSetMember struct {
	GlobalAssetID string
	Priority      int
	SourceRank    int
}

type SubscriberGlobalEODHotSetPlan struct {
	PlanRunID        string
	PlannerVersion   string
	Capacity         int
	CandidateCount   int
	EligibleCount    int
	ExcludedCount    int
	ExcludedByReason map[string]int
	Members          []SubscriberGlobalEODHotSetMember
	PlannedBy        string
	CorrelationID    string
	PlannedAt        time.Time
}

type SubscriberGlobalEODPlannerRepository interface {
	ListSubscriberGlobalEODHotSetCandidates(context.Context, int) ([]SubscriberGlobalEODHotSetCandidate, error)
	RecordSubscriberGlobalEODHotSetShadowPlan(context.Context, SubscriberGlobalEODHotSetPlan) (SubscriberGlobalEODHotSetPlan, error)
}

package storage

import (
	"context"
	"time"
)

const SubscriberGlobalEODCanaryVersion = "s4-shared-eod-canary-v1"

type SubscriberGlobalEODCanaryMember struct {
	GlobalAssetID string
	Priority      int
	SourceRank    int
}

type SubscriberGlobalEODCanaryPreparation struct {
	CanaryRunID   string
	PlanRunID     string
	SessionDate   time.Time
	MaxSymbols    int
	Members       []SubscriberGlobalEODCanaryMember
	PreparedBy    string
	CorrelationID string
	PreparedAt    time.Time
}

type SubscriberGlobalEODCanaryRepository interface {
	PrepareSubscriberGlobalEODCanary(context.Context, SubscriberGlobalEODCanaryPreparation) (SubscriberGlobalEODCanaryPreparation, error)
}

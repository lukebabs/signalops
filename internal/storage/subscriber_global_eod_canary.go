package storage

import (
	"context"
	"time"
)

const SubscriberGlobalEODCanaryVersion = "s4-shared-eod-canary-v1"
const SubscriberGlobalEODCanaryExecutionGateVersion = "s4-canary-execution-gate-v1"

type SubscriberGlobalEODCanaryMember struct {
	GlobalAssetID string
	Priority      int
	SourceRank    int
}

type SubscriberGlobalEODCanaryPreparation struct {
	CanaryRunID   string
	PlanRunID     string
	SessionDate   time.Time
	StartPriority int
	MaxSymbols    int
	Members       []SubscriberGlobalEODCanaryMember
	PreparedBy    string
	CorrelationID string
	PreparedAt    time.Time
}

// SubscriberGlobalEODCanaryExecutionGate is a disabled, append-only execution
// control. It reserves no provider capacity and cannot enable collection.
type SubscriberGlobalEODCanaryExecutionGate struct {
	ExecutionPlanID        string
	CanaryRunID            string
	ExpectedWorkerIdentity string
	MaxProviderRequests    int
	Members                []SubscriberGlobalEODCanaryExecutionMember
	PlannedBy              string
	CorrelationID          string
	PlannedAt              time.Time
}

type SubscriberGlobalEODCanaryExecutionMember struct {
	GlobalAssetID  string
	Ticker         string
	RequestOrdinal int
}

type SubscriberGlobalEODCanaryRepository interface {
	PrepareSubscriberGlobalEODCanary(context.Context, SubscriberGlobalEODCanaryPreparation) (SubscriberGlobalEODCanaryPreparation, error)
	PrepareSubscriberGlobalEODCanaryExecutionGate(context.Context, SubscriberGlobalEODCanaryExecutionGate) (SubscriberGlobalEODCanaryExecutionGate, error)
}

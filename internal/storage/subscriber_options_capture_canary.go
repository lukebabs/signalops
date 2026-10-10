package storage

import (
	"context"
	"time"
)

const SubscriberOptionsCaptureCanaryGateVersion = "s6-options-capture-gate-v1"

type SubscriberOptionsCaptureCanaryGate struct {
	CapturePlanID string
	SnapshotRunID string
	GlobalAssetID string
	Ticker        string
	PlannedBy     string
	CorrelationID string
	PlannedAt     time.Time
}

type SubscriberOptionsCaptureCanaryRepository interface {
	PrepareSubscriberOptionsCaptureCanaryGate(context.Context, SubscriberOptionsCaptureCanaryGate) (SubscriberOptionsCaptureCanaryGate, error)
}

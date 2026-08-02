package storage

import (
	"context"
	"time"
)

// MarketOpsRiskRewardSnapshotRecord is an immutable calculation revision for a
// symbol/session. Consumers select the best-evidenced revision instead of
// allowing a degraded rerun to turn historical analysis into neutral output.
type MarketOpsRiskRewardSnapshotRecord struct {
	SnapshotID         string
	TenantID           string
	AlgorithmResultID  string
	ExecutionRequestID string
	Symbol             string
	SessionDate        time.Time
	ObservedAt         time.Time
	TechnicalScore     float64
	TechnicalDirection string
	RiskLevel          string
	Confidence         float64
	UsableInputCount   int
	RequiredInputCount int
	Eligible           bool
	ResultPayloadJSON  []byte
	InputSnapshotJSON  []byte
	CreatedAt          time.Time
}

type MarketOpsRiskRewardSnapshotFilter struct {
	TenantID     string
	Symbol       string
	Symbols      []string
	SessionStart time.Time
	SessionEnd   time.Time
	EligibleOnly bool
	Limit        int
}

// MarketOpsRiskRewardSnapshotRepository is deliberately separate from the
// generic algorithm repository so existing algorithm consumers do not need to
// understand this MarketOps-specific historical projection.
type MarketOpsRiskRewardSnapshotRepository interface {
	UpsertMarketOpsRiskRewardSnapshot(ctx context.Context, record MarketOpsRiskRewardSnapshotRecord) error
	ListMarketOpsRiskRewardSnapshots(ctx context.Context, filter MarketOpsRiskRewardSnapshotFilter) ([]MarketOpsRiskRewardSnapshotRecord, error)
}

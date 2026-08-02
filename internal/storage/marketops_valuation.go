package storage

import (
	"context"
	"time"
)

// MarketOpsValuationSnapshotRecord is the immutable provider-backed input used
// by both VC and DOSM for one symbol and completed market session.
type MarketOpsValuationSnapshotRecord struct {
	SnapshotID          string
	FinancialSnapshotID string
	TenantID            string
	Symbol              string
	SessionDate         time.Time
	AvailableAt         time.Time
	Sector              string
	Industry            string
	Provider            string
	ProviderRequestIDs  []string
	InputJSON           []byte
	CreatedAt           time.Time
}

type MarketOpsValuationResultRecord struct {
	ResultID         string
	SnapshotID       string
	TenantID         string
	Symbol           string
	SessionDate      time.Time
	AlgorithmID      string
	ModelVersion     string
	Score            float64
	FairValue        float64
	Classification   string
	Confidence       int
	ConfidenceLabel  string
	EvaluationStatus string
	Eligible         bool
	ResultJSON       []byte
	CreatedAt        time.Time
}

type MarketOpsValuationFilter struct {
	TenantID      string
	Symbol        string
	UniverseGroup string
	SessionDate   time.Time
	AlgorithmID   string
	EligibleOnly  bool
	Limit         int
}

type MarketOpsValuationRepository interface {
	UpsertMarketOpsValuationSnapshot(context.Context, MarketOpsValuationSnapshotRecord) error
	UpsertMarketOpsValuationResult(context.Context, MarketOpsValuationResultRecord) error
	LatestMarketOpsValuationSnapshot(ctx context.Context, tenantID, symbol, provider string) (MarketOpsValuationSnapshotRecord, error)
	ListMarketOpsValuationResults(context.Context, MarketOpsValuationFilter) ([]MarketOpsValuationResultRecord, error)
}

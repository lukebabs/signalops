package storage

import (
	"context"
	"time"
)

type MarketOpsEEOMResultRecord struct {
	ResultID        string
	TenantID        string
	Symbol          string
	EarningsEventID string
	EarningsDate    time.Time
	SessionDate     time.Time
	ModelVersion    string
	Score           float64
	Posture         string
	Classification  string
	EvidenceQuality string
	Eligible        bool
	ResultJSON      []byte
	CreatedAt       time.Time
}
type MarketOpsEEOMFilter struct {
	TenantID, Symbol   string
	StartDate, EndDate time.Time
	EligibleOnly       bool
	Limit              int
}
type MarketOpsEEOMRepository interface {
	UpsertMarketOpsEEOMResult(context.Context, MarketOpsEEOMResultRecord) error
	ListMarketOpsEEOMResults(context.Context, MarketOpsEEOMFilter) ([]MarketOpsEEOMResultRecord, error)
}

package storage

import (
	"context"
	"time"
)

type MarketOpsFinancialStatementRecord struct {
	StatementID, TenantID, Symbol, StatementType, Period string
	PeriodEnd, AcceptedAt                                time.Time
	NormalizedJSON, RawJSON                              []byte
}
type MarketOpsFinancialSnapshotRecord struct {
	FinancialSnapshotID, TenantID, Symbol, SnapshotVersion string
	EvaluationDate, AvailableAt                            time.Time
	StatementIDs                                           []string
	InputJSON, DerivedJSON                                 []byte
}
type MarketOpsFinancialRepository interface {
	UpsertMarketOpsFinancialStatement(context.Context, MarketOpsFinancialStatementRecord) error
	UpsertMarketOpsFinancialSnapshot(context.Context, MarketOpsFinancialSnapshotRecord) error
	LatestMarketOpsFinancialSnapshot(ctx context.Context, tenantID, symbol string) (MarketOpsFinancialSnapshotRecord, error)
}

package storage

import (
	"context"
	"time"
)

type MarketOpsFMPPollState struct {
	TenantID, Symbol, Status, LastError, FinancialSnapshotID string
	LastSuccessAt, NextEligibleAt                            *time.Time
	AttemptCount                                             int
	LastProviderStatus                                       *int
	UpdatedAt                                                time.Time
}
type MarketOpsFMPPollingRepository interface {
	ReserveMarketOpsFMPCalls(context.Context, string, time.Time, int, int) (bool, error)
	CompleteMarketOpsFMPCalls(context.Context, string, time.Time, int) error
	UpsertMarketOpsFMPPollState(context.Context, MarketOpsFMPPollState) error
	GetMarketOpsFMPPollState(context.Context, string, string) (MarketOpsFMPPollState, error)
}

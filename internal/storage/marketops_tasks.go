package storage

import (
	"context"
	"time"
)

type MarketOpsTaskWorkflowRecord struct {
	WorkflowID, TenantID, WorkflowType, Status, ScheduleJobID, FailureClass, ErrorMessage string
	SessionDate                                                                           time.Time
	CoverageJSON                                                                          []byte
	StartedAt, CompletedAt                                                                *time.Time
	CreatedAt, UpdatedAt                                                                  time.Time
}

type MarketOpsTaskItemRecord struct {
	TaskID, WorkflowID, TenantID, TaskType, Symbol, Status, Provider, FailureClass, ErrorMessage string
	SessionDate                                                                                  time.Time
	AttemptCount, MaxAttempts                                                                    int
	NextAttemptAt                                                                                time.Time
	LeaseExpiresAt, CompletedAt                                                                  *time.Time
	ProviderStatus                                                                               *int
	ResultJSON                                                                                   []byte
	CreatedAt, UpdatedAt                                                                         time.Time
}

type MarketOpsTaskItemFilter struct {
	TenantID, WorkflowID, TaskType, Symbol, Status string
	SessionDate                                    time.Time
	Limit                                          int
}

type MarketOpsTaskRepository interface {
	UpsertMarketOpsTaskWorkflow(context.Context, MarketOpsTaskWorkflowRecord) error
	UpsertMarketOpsTaskItem(context.Context, MarketOpsTaskItemRecord) error
	ListMarketOpsTaskItems(context.Context, MarketOpsTaskItemFilter) ([]MarketOpsTaskItemRecord, error)
}

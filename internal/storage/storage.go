package storage

import (
	"context"
	"time"
)

const (
	RunStatusStarted   = "started"
	RunStatusSucceeded = "succeeded"
	RunStatusFailed    = "failed"
	RunStatusCanceled  = "canceled"
)

const (
	IdempotencyStatusAccepted  = "accepted"
	IdempotencyStatusPublished = "published"
	IdempotencyStatusProcessed = "processed"
	IdempotencyStatusFailed    = "failed"
	IdempotencyStatusDuplicate = "duplicate"
)

type SchedulerRunRecord struct {
	RunID            string
	TenantID         string
	SourceID         string
	SourceAdapter    string
	Datasets         []string
	ObservationDate  time.Time
	DryRun           bool
	Status           string
	StartedAt        time.Time
	CompletedAt      *time.Time
	EventsBuilt      int
	EventsPublished  int
	ProviderRequests int
	ProviderRetries  int
	Failures         int
	ConfigJSON       []byte
	ReportJSON       []byte
	ErrorMessage     string
}

type ProviderUsageRecord struct {
	UsageID      string
	RunID        string
	Provider     string
	Dataset      string
	RequestCount int
	RetryCount   int
	EventCount   int
	BudgetJSON   []byte
}

type IdempotencyRecord struct {
	TenantID       string
	SourceID       string
	IdempotencyKey string
	EventID        string
	SourceAdapter  string
	Dataset        string
	Topic          string
	Partition      *int32
	Offset         *int64
	PayloadHash    string
	Status         string
	MetadataJSON   []byte
}

type RawEventLedgerRecord struct {
	EventID         string
	TenantID        string
	SourceID        string
	SourceAdapter   string
	Dataset         string
	IdempotencyKey  string
	ObservationTime time.Time
	ProcessingTime  time.Time
	BrokerTopic     string
	BrokerPartition *int32
	BrokerOffset    *int64
	PayloadJSON     []byte
	EntityHintsJSON []byte
}

type SchedulerRunRepository interface {
	UpsertSchedulerRun(ctx context.Context, record SchedulerRunRecord) error
	InsertProviderUsage(ctx context.Context, record ProviderUsageRecord) error
}

type IdempotencyRepository interface {
	UpsertIdempotencyRecord(ctx context.Context, record IdempotencyRecord) error
}

type RawEventLedgerRepository interface {
	UpsertRawEventLedger(ctx context.Context, record RawEventLedgerRecord) error
}

type PublishRepository interface {
	IdempotencyRepository
	RawEventLedgerRepository
}

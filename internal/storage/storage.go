package storage

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("storage record not found")

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

const (
	CatalogSourceStatusActive     = "active"
	CatalogSourceStatusInactive   = "inactive"
	CatalogSourceStatusDeprecated = "deprecated"
)

const (
	CatalogPipelineStatusActive     = "active"
	CatalogPipelineStatusInactive   = "inactive"
	CatalogPipelineStatusDeprecated = "deprecated"
)

const (
	CatalogRuleStatusActive     = "active"
	CatalogRuleStatusInactive   = "inactive"
	CatalogRuleStatusDeprecated = "deprecated"
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
	CreatedAt        time.Time
	UpdatedAt        time.Time
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
	CreatedAt    time.Time
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
	FirstSeenAt    time.Time
	LastSeenAt     time.Time
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
	CreatedAt       time.Time
}

type CatalogSourceRecord struct {
	TenantID       string
	SourceID       string
	SourceDomain   string
	SourceAdapter  string
	DisplayName    string
	Description    string
	Status         string
	IngestionModes []string
	Datasets       []string
	MetadataJSON   []byte
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CatalogPipelineRecord struct {
	TenantID      string
	PipelineID    string
	SourceID      string
	SourceDomain  string
	PipelineName  string
	Description   string
	Status        string
	Stages        []string
	InputDatasets []string
	OutputTopics  []string
	MetadataJSON  []byte
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type CatalogRuleRecord struct {
	TenantID       string
	RuleID         string
	RuleName       string
	Description    string
	RuleType       string
	Severity       string
	Status         string
	Version        int
	SourceID       string
	PipelineID     string
	DatasetScope   []string
	EntityScope    []string
	ExpressionJSON []byte
	Actions        []string
	MetadataJSON   []byte
	CreatedAt      time.Time
	UpdatedAt      time.Time
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

type CatalogRepository interface {
	UpsertCatalogSource(ctx context.Context, record CatalogSourceRecord) error
	UpsertCatalogPipeline(ctx context.Context, record CatalogPipelineRecord) error
	UpsertCatalogRule(ctx context.Context, record CatalogRuleRecord) error
}

type PublishRepository interface {
	IdempotencyRepository
	RawEventLedgerRepository
}

type RawEventLedgerFilter struct {
	TenantID string
	SourceID string
	Dataset  string
	Limit    int
}

type QueryRepository interface {
	ListSchedulerRuns(ctx context.Context, limit int) ([]SchedulerRunRecord, error)
	GetSchedulerRun(ctx context.Context, runID string) (SchedulerRunRecord, error)
	ListProviderUsage(ctx context.Context, runID string, limit int) ([]ProviderUsageRecord, error)
	ListRawEventLedger(ctx context.Context, filter RawEventLedgerFilter) ([]RawEventLedgerRecord, error)
	GetRawEventLedger(ctx context.Context, eventID string) (RawEventLedgerRecord, error)
	GetIdempotencyRecord(ctx context.Context, tenantID string, sourceID string, idempotencyKey string) (IdempotencyRecord, error)
	ListCatalogSources(ctx context.Context, tenantID string, limit int) ([]CatalogSourceRecord, error)
	ListCatalogPipelines(ctx context.Context, tenantID string, limit int) ([]CatalogPipelineRecord, error)
	ListCatalogRules(ctx context.Context, tenantID string, limit int) ([]CatalogRuleRecord, error)
}

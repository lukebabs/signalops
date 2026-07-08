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

const (
	AlertStatusOpen         = "open"
	AlertStatusAcknowledged = "acknowledged"
	AlertStatusResolved     = "resolved"
	AlertStatusSuppressed   = "suppressed"
)

const (
	InsightStatusActive    = "active"
	InsightStatusReviewed  = "reviewed"
	InsightStatusDismissed = "dismissed"
	InsightStatusArchived  = "archived"
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

type NormalizedEventLedgerRecord struct {
	EventID             string
	TenantID            string
	SourceID            string
	SourceAdapter       string
	Dataset             string
	IdempotencyKey      string
	SchemaID            string
	SchemaVersion       string
	ObservationTime     time.Time
	ProcessingTime      time.Time
	Confidence          float64
	RawTopic            string
	RawPartition        int32
	RawOffset           int64
	NormalizedTopic     string
	NormalizedPartition int32
	NormalizedOffset    int64
	NormalizedPayload   []byte
	EntitiesJSON        []byte
	EvidenceJSON        []byte
	MetadataJSON        []byte
	EventJSON           []byte
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type SignalLedgerRecord struct {
	SignalID             string
	TenantID             string
	SourceID             string
	SourceDomain         string
	SourceAdapter        string
	IngestionMode        string
	Dataset              string
	EventIDs             []string
	ArtifactIDs          []string
	SignalType           string
	DetectorID           string
	DetectorVersion      string
	ModelVersion         string
	SignalTime           time.Time
	ObservationTime      time.Time
	EffectiveTime        time.Time
	ProcessingTime       time.Time
	WindowStart          time.Time
	WindowEnd            time.Time
	Confidence           float64
	Severity             string
	EntitiesJSON         []byte
	SupportingMetrics    []byte
	GraphTargetsJSON     []byte
	SemanticEvidenceJSON []byte
	EvidenceJSON         []byte
	RecommendationJSON   []byte
	CorrelationID        string
	TraceID              string
	CausationID          string
	ReplayJobID          string
	BrokerTopic          string
	BrokerPartition      int32
	BrokerOffset         int64
	EventJSON            []byte
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type AlertLedgerRecord struct {
	AlertID            string
	TenantID           string
	SourceID           string
	SourceDomain       string
	SourceAdapter      string
	Dataset            string
	SignalID           string
	DetectorID         string
	AlertType          string
	Severity           string
	Status             string
	Title              string
	Summary            string
	Confidence         float64
	EventIDs           []string
	EntitiesJSON       []byte
	EvidenceJSON       []byte
	RecommendationJSON []byte
	CorrelationID      string
	FirstObservedAt    time.Time
	LastObservedAt     time.Time
	AcknowledgedAt     *time.Time
	AcknowledgedBy     string
	ResolvedAt         *time.Time
	ResolvedBy         string
	MetadataJSON       []byte
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type InsightLedgerRecord struct {
	InsightID            string
	TenantID             string
	SourceID             string
	SourceDomain         string
	SourceAdapter        string
	Dataset              string
	SignalID             string
	DetectorID           string
	InsightType          string
	Status               string
	Title                string
	Summary              string
	Confidence           float64
	Severity             string
	EventIDs             []string
	EntitiesJSON         []byte
	SupportingMetrics    []byte
	SemanticEvidenceJSON []byte
	RecommendationJSON   []byte
	CorrelationID        string
	ObservedAt           time.Time
	ReviewedAt           *time.Time
	ReviewedBy           string
	MetadataJSON         []byte
	CreatedAt            time.Time
	UpdatedAt            time.Time
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

type NormalizedEventLedgerRepository interface {
	UpsertNormalizedEventLedger(ctx context.Context, record NormalizedEventLedgerRecord) error
}

type SignalLedgerRepository interface {
	UpsertSignalLedger(ctx context.Context, record SignalLedgerRecord) error
}

type AlertLedgerRepository interface {
	UpsertAlertLedger(ctx context.Context, record AlertLedgerRecord) error
}

type InsightLedgerRepository interface {
	UpsertInsightLedger(ctx context.Context, record InsightLedgerRecord) error
}

type SignalLifecycleRepository interface {
	SignalLedgerRepository
	AlertLedgerRepository
	InsightLedgerRepository
	PersistSignalLifecycle(ctx context.Context, signal SignalLedgerRecord, alerts []AlertLedgerRecord, insights []InsightLedgerRecord) error
}

type CatalogRepository interface {
	UpsertCatalogSource(ctx context.Context, record CatalogSourceRecord) error
	UpsertCatalogPipeline(ctx context.Context, record CatalogPipelineRecord) error
	UpsertCatalogRule(ctx context.Context, record CatalogRuleRecord) error
}

type PublishRepository interface {
	IdempotencyRepository
	RawEventLedgerRepository
	PersistPublishedRawEvent(ctx context.Context, ledger RawEventLedgerRecord, idempotency IdempotencyRecord) error
}

type RawEventLedgerFilter struct {
	TenantID string
	SourceID string
	Dataset  string
	Limit    int
}

type SignalLedgerFilter struct {
	TenantID   string
	SourceID   string
	Dataset    string
	DetectorID string
	Severity   string
	Limit      int
}

type AlertLedgerFilter struct {
	TenantID string
	SourceID string
	Dataset  string
	Severity string
	Status   string
	Limit    int
}

type InsightLedgerFilter struct {
	TenantID    string
	SourceID    string
	Dataset     string
	InsightType string
	Status      string
	Limit       int
}

type QueryRepository interface {
	ListSchedulerRuns(ctx context.Context, limit int) ([]SchedulerRunRecord, error)
	GetSchedulerRun(ctx context.Context, runID string) (SchedulerRunRecord, error)
	ListProviderUsage(ctx context.Context, runID string, limit int) ([]ProviderUsageRecord, error)
	ListRawEventLedger(ctx context.Context, filter RawEventLedgerFilter) ([]RawEventLedgerRecord, error)
	GetRawEventLedger(ctx context.Context, eventID string) (RawEventLedgerRecord, error)
	ListNormalizedEventLedger(ctx context.Context, filter RawEventLedgerFilter) ([]NormalizedEventLedgerRecord, error)
	GetNormalizedEventLedger(ctx context.Context, eventID string) (NormalizedEventLedgerRecord, error)
	ListSignalLedger(ctx context.Context, filter SignalLedgerFilter) ([]SignalLedgerRecord, error)
	GetSignalLedger(ctx context.Context, signalID string) (SignalLedgerRecord, error)
	ListAlertLedger(ctx context.Context, filter AlertLedgerFilter) ([]AlertLedgerRecord, error)
	GetAlertLedger(ctx context.Context, alertID string) (AlertLedgerRecord, error)
	ListInsightLedger(ctx context.Context, filter InsightLedgerFilter) ([]InsightLedgerRecord, error)
	GetInsightLedger(ctx context.Context, insightID string) (InsightLedgerRecord, error)
	GetIdempotencyRecord(ctx context.Context, tenantID string, sourceID string, idempotencyKey string) (IdempotencyRecord, error)
	ListCatalogSources(ctx context.Context, tenantID string, limit int) ([]CatalogSourceRecord, error)
	ListCatalogPipelines(ctx context.Context, tenantID string, limit int) ([]CatalogPipelineRecord, error)
	ListCatalogRules(ctx context.Context, tenantID string, limit int) ([]CatalogRuleRecord, error)
}

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
	ReplayJobStatusQueued    = "queued"
	ReplayJobStatusRunning   = "running"
	ReplayJobStatusSucceeded = "succeeded"
	ReplayJobStatusFailed    = "failed"
	ReplayJobStatusCanceled  = "canceled"
)

const (
	ReplaySourceRaw        = "raw_events"
	ReplaySourceNormalized = "normalized_events"
	ReplaySourceSignals    = "signals"
)

const (
	ReplayModeOriginal         = "original"
	ReplayModeLatestCompatible = "latest_compatible"
	ReplayModeExplicit         = "explicit"
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

const (
	MarketOpsDSMGraphProposalStatusProposed   = "proposed"
	MarketOpsDSMGraphProposalStatusAccepted   = "accepted"
	MarketOpsDSMGraphProposalStatusRejected   = "rejected"
	MarketOpsDSMGraphProposalStatusSuperseded = "superseded"
)

const (
	MarketOpsBacktestPolicyAutoAcceptCandidate  = "auto_accept_candidate"
	MarketOpsBacktestPolicyAutoRejectCandidate  = "auto_reject_candidate"
	MarketOpsBacktestPolicyManualReviewRequired = "manual_review_required"
	MarketOpsBacktestPolicySupersedeCandidate   = "supersede_candidate"
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

// ReplayJobRecord is control-plane state for replaying temporal ledgers.
// Execution is owned by a later worker gate; this record captures the request,
// filters, lifecycle status, and eventual result metadata.
type ReplayJobRecord struct {
	ReplayJobID  string
	TenantID     string
	SourceID     string
	Dataset      string
	SourceKind   string
	ReplayMode   string
	Status       string
	RequestedBy  string
	WindowStart  time.Time
	WindowEnd    time.Time
	StartedAt    *time.Time
	CompletedAt  *time.Time
	FiltersJSON  []byte
	OptionsJSON  []byte
	ResultJSON   []byte
	ErrorMessage string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ReplayWorkerHeartbeatRecord struct {
	WorkerID                 string
	Status                   string
	ProcessStartedAt         time.Time
	LastSeenAt               time.Time
	LastClaimedAt            *time.Time
	LastClaimedReplayJobID   string
	LastCompletedAt          *time.Time
	LastCompletedReplayJobID string
	LastErrorAt              *time.Time
	LastErrorMessage         string
	MetadataJSON             []byte
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

type ReplayJobStatusCount struct {
	Status string
	Count  int
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
	AppID           string
	Domain          string
	UseCase         string
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
	AppID               string
	Domain              string
	UseCase             string
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
	AppID                string
	Domain               string
	UseCase              string
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
	AppID              string
	Domain             string
	UseCase            string
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
	AppID                string
	Domain               string
	UseCase              string
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

type AlertLifecycleMutation struct {
	AlertID      string
	Status       string
	Actor        string
	MutatedAt    time.Time
	MetadataJSON []byte
}

type InsightLifecycleMutation struct {
	InsightID    string
	Status       string
	Actor        string
	MutatedAt    time.Time
	MetadataJSON []byte
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

type MarketOpsAssetRecord struct {
	TenantID      string
	AppID         string
	Domain        string
	UseCase       string
	SourceID      string
	UniverseGroup string
	Rank          int
	Ticker        string
	TickerKey     string
	Company       string
	CompanyKey    string
	AssetType     string
	Exchange      string
	Sector        string
	SectorKey     string
	Industry      string
	IndustryKey   string
	IsActive      bool
	MetadataJSON  []byte
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type MarketOpsDSMArtifactRecord struct {
	ArtifactID           string
	TenantID             string
	AppID                string
	Domain               string
	UseCase              string
	SourceID             string
	SourceAdapter        string
	Dataset              string
	SignalID             string
	SignalType           string
	DetectorID           string
	Severity             string
	Confidence           float64
	EventIDs             []string
	SubjectSymbol        string
	ArtifactType         string
	ArtifactJSON         []byte
	SemanticEvidenceJSON []byte
	GraphTargetsJSON     []byte
	SupportingMetrics    []byte
	QualityIssues        []string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type MarketOpsDSMGraphProposalRecord struct {
	ProposalID     string
	TenantID       string
	AppID          string
	Domain         string
	UseCase        string
	SourceID       string
	SourceAdapter  string
	Dataset        string
	ArtifactID     string
	SignalID       string
	SignalType     string
	DetectorID     string
	Severity       string
	Confidence     float64
	EventIDs       []string
	SubjectSymbol  string
	CandidateType  string
	NodeID         string
	FromNode       string
	Relationship   string
	ToNode         string
	Labels         []string
	PropertiesJSON []byte
	RawCandidate   []byte
	Status         string
	ReviewedBy     string
	DecisionNote   string
	DecidedAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type MarketOpsBacktestRunRecord struct {
	RunID           string
	TenantID        string
	AppID           string
	Domain          string
	UseCase         string
	SourceID        string
	SourceAdapter   string
	Dataset         string
	DetectorID      string
	DetectorVersion string
	Status          string
	RequestedBy     string
	WindowStart     time.Time
	WindowEnd       time.Time
	StartedAt       time.Time
	CompletedAt     *time.Time
	FiltersJSON     []byte
	ParametersJSON  []byte
	MetricsJSON     []byte
	ErrorMessage    string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type MarketOpsBacktestSignalRecord struct {
	RunID string
	SignalLedgerRecord
}

type MarketOpsBacktestArtifactRecord struct {
	RunID string
	MarketOpsDSMArtifactRecord
}

type MarketOpsBacktestGraphProposalRecord struct {
	RunID string
	MarketOpsDSMGraphProposalRecord
}

type MarketOpsBacktestPolicyResultRecord struct {
	RunID              string
	PolicyResultID     string
	ProposalID         string
	ArtifactID         string
	SignalID           string
	TenantID           string
	SubjectSymbol      string
	CandidateType      string
	Recommendation     string
	Reason             string
	PolicyVersion      string
	Confidence         float64
	DecisionInputsJSON []byte
	CreatedAt          time.Time
}

type MarketOpsBacktestCalibrationSummaryRecord struct {
	SummaryID              string
	TenantID               string
	AppID                  string
	Domain                 string
	UseCase                string
	SourceID               string
	Dataset                string
	DetectorID             string
	StatusFilter           string
	RequestedBy            string
	RunIDs                 []string
	RunCount               int
	SucceededCount         int
	FailedCount            int
	ZeroInputCount         int
	Scanned                int
	Signals                int
	Artifacts              int
	GraphProposals         int
	PolicyResults          int
	SignalYield            float64
	PolicyResultsPerSignal float64
	RecommendationCounts   []byte
	RecommendationShares   []byte
	DominantRecommendation []byte
	FiltersJSON            []byte
	ParametersJSON         []byte
	CreatedAt              time.Time
}

const (
	MarketOpsBacktestCalibrationBaselineStatusActive   = "active"
	MarketOpsBacktestCalibrationBaselineStatusArchived = "archived"

	MarketOpsBacktestCalibrationRecommendationNeedsMoreData = "needs_more_data"
	MarketOpsBacktestCalibrationRecommendationRegression    = "regression_candidate"
	MarketOpsBacktestCalibrationRecommendationImprovement   = "improvement_candidate"
	MarketOpsBacktestCalibrationRecommendationNeutral       = "neutral_candidate"
	MarketOpsBacktestCalibrationRecommendationManualReview  = "manual_review_required"
)

type MarketOpsBacktestCalibrationBaselineRecord struct {
	BaselineID  string
	TenantID    string
	AppID       string
	Domain      string
	UseCase     string
	Name        string
	Description string
	SummaryID   string
	DetectorID  string
	Dataset     string
	ScopeJSON   []byte
	Status      string
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type MarketOpsBacktestCalibrationComparisonRecord struct {
	ComparisonID          string
	TenantID              string
	BaselineID            string
	BaselineSummaryID     string
	CandidateSummaryID    string
	DetectorID            string
	Dataset               string
	ComparisonMetricsJSON []byte
	Recommendation        string
	RecommendationReason  string
	CreatedBy             string
	CreatedAt             time.Time
}

type SchedulerRunRepository interface {
	UpsertSchedulerRun(ctx context.Context, record SchedulerRunRecord) error
	InsertProviderUsage(ctx context.Context, record ProviderUsageRecord) error
}

type ReplayJobRepository interface {
	UpsertReplayJob(ctx context.Context, record ReplayJobRecord) error
	ClaimNextReplayJob(ctx context.Context, workerID string, claimedAt time.Time) (ReplayJobRecord, error)
	CompleteReplayJob(ctx context.Context, replayJobID string, completedAt time.Time, resultJSON []byte) (ReplayJobRecord, error)
	FailReplayJob(ctx context.Context, replayJobID string, failedAt time.Time, errorMessage string, resultJSON []byte) (ReplayJobRecord, error)
	CancelReplayJob(ctx context.Context, replayJobID string, actor string, canceledAt time.Time, reason string, resultJSON []byte) (ReplayJobRecord, error)
}

type ReplayWorkerHeartbeatRepository interface {
	UpsertReplayWorkerHeartbeat(ctx context.Context, record ReplayWorkerHeartbeatRecord) error
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

type MarketOpsDSMArtifactRepository interface {
	ListMarketOpsDSMArtifacts(ctx context.Context, filter MarketOpsDSMArtifactFilter) ([]MarketOpsDSMArtifactRecord, error)
	GetMarketOpsDSMArtifact(ctx context.Context, artifactID string) (MarketOpsDSMArtifactRecord, error)
}

type MarketOpsDSMGraphProposalRepository interface {
	ListMarketOpsDSMGraphProposals(ctx context.Context, filter MarketOpsDSMGraphProposalFilter) ([]MarketOpsDSMGraphProposalRecord, error)
	GetMarketOpsDSMGraphProposal(ctx context.Context, proposalID string) (MarketOpsDSMGraphProposalRecord, error)
	MutateMarketOpsDSMGraphProposal(ctx context.Context, mutation MarketOpsDSMGraphProposalMutation) (MarketOpsDSMGraphProposalRecord, error)
}

type MarketOpsBacktestRepository interface {
	CreateMarketOpsBacktestRun(ctx context.Context, record MarketOpsBacktestRunRecord) error
	CompleteMarketOpsBacktestRun(ctx context.Context, runID string, completedAt time.Time, metricsJSON []byte) (MarketOpsBacktestRunRecord, error)
	FailMarketOpsBacktestRun(ctx context.Context, runID string, failedAt time.Time, errorMessage string, metricsJSON []byte) (MarketOpsBacktestRunRecord, error)
	PersistMarketOpsBacktestBatch(ctx context.Context, run MarketOpsBacktestRunRecord, signals []MarketOpsBacktestSignalRecord, artifacts []MarketOpsBacktestArtifactRecord, proposals []MarketOpsBacktestGraphProposalRecord, policyResults []MarketOpsBacktestPolicyResultRecord) error
	ListMarketOpsBacktestRuns(ctx context.Context, filter MarketOpsBacktestRunFilter) ([]MarketOpsBacktestRunRecord, error)
	GetMarketOpsBacktestRun(ctx context.Context, runID string) (MarketOpsBacktestRunRecord, error)
	ListMarketOpsBacktestSignals(ctx context.Context, filter MarketOpsBacktestSignalFilter) ([]MarketOpsBacktestSignalRecord, error)
	ListMarketOpsBacktestGraphProposals(ctx context.Context, filter MarketOpsBacktestGraphProposalFilter) ([]MarketOpsBacktestGraphProposalRecord, error)
	ListMarketOpsBacktestPolicyResults(ctx context.Context, filter MarketOpsBacktestGraphProposalFilter) ([]MarketOpsBacktestPolicyResultRecord, error)
	ListMarketOpsBacktestNormalizedEvents(ctx context.Context, filter MarketOpsBacktestEventFilter) ([]NormalizedEventLedgerRecord, error)
	UpsertMarketOpsBacktestCalibrationSummary(ctx context.Context, record MarketOpsBacktestCalibrationSummaryRecord) error
	ListMarketOpsBacktestCalibrationSummaries(ctx context.Context, filter MarketOpsBacktestCalibrationSummaryFilter) ([]MarketOpsBacktestCalibrationSummaryRecord, error)
	GetMarketOpsBacktestCalibrationSummary(ctx context.Context, summaryID string) (MarketOpsBacktestCalibrationSummaryRecord, error)
	UpsertMarketOpsBacktestCalibrationBaseline(ctx context.Context, record MarketOpsBacktestCalibrationBaselineRecord) error
	ListMarketOpsBacktestCalibrationBaselines(ctx context.Context, filter MarketOpsBacktestCalibrationBaselineFilter) ([]MarketOpsBacktestCalibrationBaselineRecord, error)
	GetMarketOpsBacktestCalibrationBaseline(ctx context.Context, baselineID string) (MarketOpsBacktestCalibrationBaselineRecord, error)
	UpsertMarketOpsBacktestCalibrationComparison(ctx context.Context, record MarketOpsBacktestCalibrationComparisonRecord) error
	ListMarketOpsBacktestCalibrationComparisons(ctx context.Context, filter MarketOpsBacktestCalibrationComparisonFilter) ([]MarketOpsBacktestCalibrationComparisonRecord, error)
	GetMarketOpsBacktestCalibrationComparison(ctx context.Context, comparisonID string) (MarketOpsBacktestCalibrationComparisonRecord, error)
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
	AppID    string
	Domain   string
	UseCase  string
	SourceID string
	Dataset  string
	Limit    int
}

type SignalLedgerFilter struct {
	TenantID   string
	AppID      string
	Domain     string
	UseCase    string
	SourceID   string
	Dataset    string
	DetectorID string
	Severity   string
	Limit      int
}

type AlertLedgerFilter struct {
	TenantID string
	AppID    string
	Domain   string
	UseCase  string
	SourceID string
	Dataset  string
	Severity string
	Status   string
	Limit    int
}

type InsightLedgerFilter struct {
	TenantID    string
	AppID       string
	Domain      string
	UseCase     string
	SourceID    string
	Dataset     string
	InsightType string
	Status      string
	Limit       int
}

type MarketOpsDSMArtifactFilter struct {
	TenantID      string
	AppID         string
	Domain        string
	UseCase       string
	SignalType    string
	Severity      string
	SubjectSymbol string
	Limit         int
}

type MarketOpsDSMGraphProposalFilter struct {
	TenantID      string
	AppID         string
	Domain        string
	UseCase       string
	ArtifactID    string
	SignalID      string
	SignalType    string
	SubjectSymbol string
	CandidateType string
	Status        string
	Limit         int
}

type MarketOpsDSMGraphProposalMutation struct {
	ProposalID   string
	Status       string
	ReviewedBy   string
	DecisionNote string
	DecidedAt    time.Time
}

type MarketOpsBacktestRunFilter struct {
	TenantID   string
	AppID      string
	Domain     string
	UseCase    string
	SourceID   string
	Dataset    string
	DetectorID string
	Status     string
	Limit      int
}

type MarketOpsBacktestCalibrationSummaryFilter struct {
	TenantID   string
	AppID      string
	Domain     string
	UseCase    string
	SourceID   string
	Dataset    string
	DetectorID string
	Limit      int
}

type MarketOpsBacktestCalibrationBaselineFilter struct {
	TenantID   string
	AppID      string
	Domain     string
	UseCase    string
	DetectorID string
	Dataset    string
	Status     string
	Limit      int
}

type MarketOpsBacktestCalibrationComparisonFilter struct {
	TenantID       string
	BaselineID     string
	DetectorID     string
	Dataset        string
	Recommendation string
	Limit          int
}

type MarketOpsBacktestSignalFilter struct {
	RunID         string
	TenantID      string
	SignalType    string
	SubjectSymbol string
	Limit         int
}

type MarketOpsBacktestGraphProposalFilter struct {
	RunID          string
	TenantID       string
	SignalType     string
	SubjectSymbol  string
	CandidateType  string
	Recommendation string
	Limit          int
}

type MarketOpsBacktestEventFilter struct {
	TenantID      string
	AppID         string
	Domain        string
	UseCase       string
	SourceID      string
	SourceAdapter string
	Dataset       string
	Symbols       []string
	WindowStart   time.Time
	WindowEnd     time.Time
	Limit         int
	Offset        int
}

type ReplayJobFilter struct {
	TenantID   string
	SourceID   string
	Dataset    string
	SourceKind string
	Status     string
	Limit      int
}

type QueryRepository interface {
	ReplayJobRepository
	ReplayWorkerHeartbeatRepository
	MarketOpsBacktestRepository
	ListSchedulerRuns(ctx context.Context, limit int) ([]SchedulerRunRecord, error)
	GetSchedulerRun(ctx context.Context, runID string) (SchedulerRunRecord, error)
	ListReplayJobs(ctx context.Context, filter ReplayJobFilter) ([]ReplayJobRecord, error)
	GetReplayJob(ctx context.Context, replayJobID string) (ReplayJobRecord, error)
	CountReplayJobsByStatus(ctx context.Context, tenantID string) ([]ReplayJobStatusCount, error)
	ListReplayWorkerHeartbeats(ctx context.Context, limit int) ([]ReplayWorkerHeartbeatRecord, error)
	ListReplayRawEvents(ctx context.Context, job ReplayJobRecord, limit int, offset int) ([]RawEventLedgerRecord, error)
	ListReplayNormalizedEvents(ctx context.Context, job ReplayJobRecord, limit int, offset int) ([]NormalizedEventLedgerRecord, error)
	ListReplaySignals(ctx context.Context, job ReplayJobRecord, limit int, offset int) ([]SignalLedgerRecord, error)
	ListProviderUsage(ctx context.Context, runID string, limit int) ([]ProviderUsageRecord, error)
	ListRawEventLedger(ctx context.Context, filter RawEventLedgerFilter) ([]RawEventLedgerRecord, error)
	GetRawEventLedger(ctx context.Context, eventID string) (RawEventLedgerRecord, error)
	ListNormalizedEventLedger(ctx context.Context, filter RawEventLedgerFilter) ([]NormalizedEventLedgerRecord, error)
	GetNormalizedEventLedger(ctx context.Context, eventID string) (NormalizedEventLedgerRecord, error)
	ListSignalLedger(ctx context.Context, filter SignalLedgerFilter) ([]SignalLedgerRecord, error)
	GetSignalLedger(ctx context.Context, signalID string) (SignalLedgerRecord, error)
	ListAlertLedger(ctx context.Context, filter AlertLedgerFilter) ([]AlertLedgerRecord, error)
	GetAlertLedger(ctx context.Context, alertID string) (AlertLedgerRecord, error)
	MutateAlertLifecycle(ctx context.Context, mutation AlertLifecycleMutation) (AlertLedgerRecord, error)
	ListInsightLedger(ctx context.Context, filter InsightLedgerFilter) ([]InsightLedgerRecord, error)
	GetInsightLedger(ctx context.Context, insightID string) (InsightLedgerRecord, error)
	ListMarketOpsDSMArtifacts(ctx context.Context, filter MarketOpsDSMArtifactFilter) ([]MarketOpsDSMArtifactRecord, error)
	GetMarketOpsDSMArtifact(ctx context.Context, artifactID string) (MarketOpsDSMArtifactRecord, error)
	ListMarketOpsDSMGraphProposals(ctx context.Context, filter MarketOpsDSMGraphProposalFilter) ([]MarketOpsDSMGraphProposalRecord, error)
	GetMarketOpsDSMGraphProposal(ctx context.Context, proposalID string) (MarketOpsDSMGraphProposalRecord, error)
	MutateMarketOpsDSMGraphProposal(ctx context.Context, mutation MarketOpsDSMGraphProposalMutation) (MarketOpsDSMGraphProposalRecord, error)
	MutateInsightLifecycle(ctx context.Context, mutation InsightLifecycleMutation) (InsightLedgerRecord, error)
	GetIdempotencyRecord(ctx context.Context, tenantID string, sourceID string, idempotencyKey string) (IdempotencyRecord, error)
	ListCatalogSources(ctx context.Context, tenantID string, limit int) ([]CatalogSourceRecord, error)
	ListCatalogPipelines(ctx context.Context, tenantID string, limit int) ([]CatalogPipelineRecord, error)
	ListCatalogRules(ctx context.Context, tenantID string, limit int) ([]CatalogRuleRecord, error)
	ListMarketOpsAssets(ctx context.Context, tenantID string, universeGroup string, activeOnly bool, limit int) ([]MarketOpsAssetRecord, error)
}

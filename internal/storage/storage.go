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
	AlgorithmDefinitionStatusDraft      = "draft"
	AlgorithmDefinitionStatusActive     = "active"
	AlgorithmDefinitionStatusDisabled   = "disabled"
	AlgorithmDefinitionStatusDeprecated = "deprecated"
)

const (
	AlgorithmRuntimePythonPlugin    = "python_plugin"
	AlgorithmRuntimeContainerPlugin = "container_plugin"
	AlgorithmRuntimeHTTPPlugin      = "http_plugin"
)

const (
	AlgorithmExecutionStatusQueued    = "queued"
	AlgorithmExecutionStatusRunning   = "running"
	AlgorithmExecutionStatusSucceeded = "succeeded"
	AlgorithmExecutionStatusFailed    = "failed"
	AlgorithmExecutionStatusCanceled  = "canceled"
)

const (
	AlgorithmSignalProposalStatusProposed   = "proposed"
	AlgorithmSignalProposalStatusReviewed   = "reviewed"
	AlgorithmSignalProposalStatusRejected   = "rejected"
	AlgorithmSignalProposalStatusSuperseded = "superseded"
)

const (
	AlgorithmSignalMaterializationStatusRequested  = "requested"
	AlgorithmSignalMaterializationStatusRunning    = "running"
	AlgorithmSignalMaterializationStatusSucceeded  = "succeeded"
	AlgorithmSignalMaterializationStatusDuplicate  = "duplicate"
	AlgorithmSignalMaterializationStatusBlocked    = "blocked"
	AlgorithmSignalMaterializationStatusFailed     = "failed"
	AlgorithmSignalMaterializationStatusSuperseded = "superseded"
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
	SyncraticInsightStatusActive     = "active"
	SyncraticInsightStatusReviewed   = "reviewed"
	SyncraticInsightStatusDismissed  = "dismissed"
	SyncraticInsightStatusArchived   = "archived"
	SyncraticInsightStatusSuperseded = "superseded"
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

const (
	MarketOpsFeatureDefinitionStatusDraft      = "draft"
	MarketOpsFeatureDefinitionStatusActive     = "active"
	MarketOpsFeatureDefinitionStatusDisabled   = "disabled"
	MarketOpsFeatureDefinitionStatusDeprecated = "deprecated"
)

const (
	MarketOpsQualityUsable            = "usable"
	MarketOpsQualityUsableWithWarning = "usable_with_warning"
	MarketOpsQualityPartial           = "partial"
	MarketOpsQualitySparse            = "sparse"
	MarketOpsQualityStale             = "stale"
	MarketOpsQualityInvalid           = "invalid"
	MarketOpsQualityMissing           = "missing"
	MarketOpsQualityNotApplicable     = "not_applicable"
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

type SyncraticContextWindowRecord struct {
	ContextWindowID            string
	TenantID                   string
	AppID                      string
	Domain                     string
	UseCase                    string
	SubjectType                string
	SubjectID                  string
	SubjectSymbol              string
	WindowStart                time.Time
	WindowEnd                  time.Time
	ContextStrategy            string
	ContextBuilderVersion      string
	SignalTypes                []string
	DetectorIDs                []string
	EventIDs                   []string
	SignalIDs                  []string
	AlertIDs                   []string
	ArtifactIDs                []string
	GraphProposalIDs           []string
	LabelIDs                   []string
	BaselineRefsJSON           []byte
	EvaluationRefsJSON         []byte
	PromotionCandidateRefsJSON []byte
	SummaryMetricsJSON         []byte
	EvidenceDigest             string
	IdempotencyKey             string
	Status                     string
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}

type SyncraticInsightRecord struct {
	SyncraticInsightID      string
	TenantID                string
	AppID                   string
	Domain                  string
	UseCase                 string
	ContextWindowID         string
	InsightType             string
	SubjectType             string
	SubjectID               string
	SubjectSymbol           string
	Status                  string
	Severity                string
	Confidence              float64
	Title                   string
	Summary                 string
	Explanation             string
	SupportingAlertIDs      []string
	SupportingSignalIDs     []string
	SupportingEventIDs      []string
	SupportingArtifactIDs   []string
	RelatedGraphProposalIDs []string
	RelatedLabelIDs         []string
	MetricsJSON             []byte
	RecommendationJSON      []byte
	BuilderVersion          string
	CreatedAt               time.Time
	UpdatedAt               time.Time
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

type AlgorithmDefinitionRecord struct {
	AlgorithmID     string
	TenantID        string
	Name            string
	Description     string
	AlgorithmType   string
	RuntimeType     string
	InputFeatures   []string
	InputEventTypes []string
	OutputSchema    []byte
	ConfigSchema    []byte
	DefaultConfig   []byte
	Version         string
	Status          string
	MetadataJSON    []byte
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type AlgorithmExecutionRequestRecord struct {
	ExecutionRequestID string
	TenantID           string
	AlgorithmID        string
	AlgorithmVersion   string
	EventIDs           []string
	FeatureRefs        []string
	EntityRefs         []string
	WindowRef          string
	ConfigJSON         []byte
	CorrelationID      string
	Status             string
	RequestedBy        string
	ResultJSON         []byte
	ErrorMessage       string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type AlgorithmResultRecord struct {
	AlgorithmResultID  string
	TenantID           string
	AlgorithmID        string
	AlgorithmVersion   string
	ExecutionRequestID string
	ResultType         string
	Score              float64
	Confidence         float64
	Severity           string
	ResultPayloadJSON  []byte
	SourceEventIDs     []string
	FeatureValueIDs    []string
	EvidenceRefs       []string
	CorrelationID      string
	CreatedAt          time.Time
}

type AlgorithmSignalProposalRecord struct {
	ProposalID          string
	TenantID            string
	AlgorithmResultID   string
	AlgorithmID         string
	AlgorithmVersion    string
	ExecutionRequestID  string
	ProposedSignalType  string
	Status              string
	Score               float64
	Confidence          float64
	Severity            string
	ProposalPayloadJSON []byte
	RationaleJSON       []byte
	SourceEventIDs      []string
	EvidenceRefs        []string
	CorrelationID       string
	CreatedBy           string
	ReviewedBy          string
	DecisionNote        string
	DecidedAt           *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type AlgorithmSignalProposalMutation struct {
	TenantID     string
	ProposalID   string
	Status       string
	ReviewedBy   string
	DecisionNote string
	DecidedAt    time.Time
	MetadataJSON []byte
}

type AlgorithmSignalMaterializationRecord struct {
	MaterializationID            string
	TenantID                     string
	ProposalID                   string
	AlgorithmResultID            string
	ExecutionRequestID           string
	AlgorithmID                  string
	AlgorithmVersion             string
	ProposedSignalType           string
	SignalID                     string
	MaterializationStatus        string
	MaterializationPolicyVersion string
	IdempotencyKey               string
	DuplicateOfSignalID          string
	RequestedBy                  string
	RequestedAt                  time.Time
	StartedAt                    *time.Time
	CompletedAt                  *time.Time
	FailedAt                     *time.Time
	ErrorCode                    string
	ErrorMessage                 string
	RequestMetadataJSON          []byte
	PreflightSnapshotJSON        []byte
	SignalPayloadPreviewJSON     []byte
	CreatedAt                    time.Time
	UpdatedAt                    time.Time
}

type AlgorithmDefinitionFilter struct {
	TenantID      string
	AlgorithmType string
	RuntimeType   string
	Status        string
	Limit         int
}

type AlgorithmExecutionRequestFilter struct {
	TenantID      string
	AlgorithmID   string
	Status        string
	CorrelationID string
	Limit         int
}

type AlgorithmResultFilter struct {
	TenantID           string
	AlgorithmID        string
	ExecutionRequestID string
	ResultType         string
	Severity           string
	CorrelationID      string
	Limit              int
}

type AlgorithmSignalProposalFilter struct {
	TenantID           string
	AlgorithmID        string
	ExecutionRequestID string
	AlgorithmResultID  string
	Status             string
	Severity           string
	CorrelationID      string
	Limit              int
}

type AlgorithmSignalMaterializationFilter struct {
	TenantID              string
	ProposalID            string
	AlgorithmResultID     string
	ExecutionRequestID    string
	AlgorithmID           string
	MaterializationStatus string
	SignalID              string
	Limit                 int
}

type AlgorithmSignalMaterializationMutation struct {
	Record AlgorithmSignalMaterializationRecord
}

type AlgorithmSignalProposalSummaryRecord struct {
	TenantID                    string
	TotalProposals              int
	ProposedCount               int
	ReviewedCount               int
	RejectedCount               int
	SupersededCount             int
	ReviewedRatio               float64
	HighCriticalUnreviewedCount int
	StatusCounts                map[string]int
	SeverityCounts              map[string]int
	ProposedSignalTypeCounts    map[string]int
	AlgorithmIDCounts           map[string]int
	ReviewerCounts              map[string]int
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

type MarketOpsOptionsChainRecord struct {
	TenantID          string
	Symbol            string
	TradeDate         time.Time
	OptionTicker      string
	Provider          string
	SourceID          string
	IngestionRunID    string
	ContractType      string
	ExpirationDate    time.Time
	StrikePrice       float64
	UnderlyingClose   *float64
	Moneyness         *float64
	Open              *float64
	High              *float64
	Low               *float64
	Close             *float64
	VWAP              *float64
	Volume            *int64
	OpenInterest      *int64
	ImpliedVolatility *float64
	Delta             *float64
	Gamma             *float64
	Theta             *float64
	Vega              *float64
	ProviderRequestID string
	PayloadHash       string
	RawPayloadJSON    []byte
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type MarketOpsOptionsChainFilter struct {
	TenantID     string
	Symbol       string
	TradeDate    time.Time
	WindowStart  time.Time
	WindowEnd    time.Time
	ContractType string
	Limit        int
}

type MarketOpsOptionsCoverageRecord struct {
	TenantID       string
	Symbol         string
	TradeDayCount  int
	ContractCount  int
	FirstTradeDate time.Time
	LastTradeDate  time.Time
	LastUpdatedAt  time.Time
}

type MarketOpsOptionsDistributionRecord struct {
	TenantID                   string
	Symbol                     string
	TradeDate                  time.Time
	WindowName                 string
	SourceID                   string
	Provider                   string
	TradeDays                  int
	ContractCount              int
	CallContractCount          int
	PutContractCount           int
	TotalCallOpenInterest      int64
	TotalPutOpenInterest       int64
	TotalCallVolume            int64
	TotalPutVolume             int64
	MissingOpenInterestCount   int
	CallPutOpenInterestRatio   float64
	CallPutVolumeRatio         float64
	RatioDelta                 float64
	RatioChangePct             float64
	RatioZScore                float64
	ChangePointScore           float64
	Confidence                 float64
	MoneynessDistributionJSON  []byte
	ExpirationDistributionJSON []byte
	MetricsJSON                []byte
	SourceTradeDates           []time.Time
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}

type MarketOpsOptionsDistributionFilter struct {
	TenantID   string
	Symbol     string
	WindowName string
	Limit      int
}

type MarketOpsFeatureDefinitionRecord struct {
	TenantID        string
	FeatureKey      string
	FeatureVersion  string
	Domain          string
	Title           string
	Description     string
	ValueType       string
	Unit            string
	CalculationSpec []byte
	RequiredInputs  []byte
	QualityPolicy   []byte
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type MarketOpsFeatureDefinitionFilter struct {
	TenantID       string
	FeatureKey     string
	FeatureVersion string
	Domain         string
	Status         string
	Limit          int
}

type MarketOpsFeatureObservationRecord struct {
	FeatureObservationID string
	TenantID             string
	AppID                string
	AssetID              string
	Symbol               string
	SessionDate          time.Time
	AsOfTime             time.Time
	FeatureKey           string
	FeatureVersion       string
	DimensionsJSON       []byte
	NumericValue         *float64
	TextValue            *string
	BooleanValue         *bool
	QualityState         string
	QualityScore         *float64
	QualityDetailsJSON   []byte
	SourceEventIDs       []string
	SourceArtifactIDs    []string
	CalculationRunID     string
	DeterministicKey     string
	CreatedAt            time.Time
}

type MarketOpsFeatureObservationFilter struct {
	TenantID              string
	AppID                 string
	AssetID               string
	Symbol                string
	FeatureKey            string
	FeatureVersion        string
	Domain                string
	QualityState          string
	DimensionsJSON        []byte
	SessionStart          time.Time
	SessionEnd            time.Time
	FeatureObservationIDs []string
	Limit                 int
}

type MarketOpsMarketStateRecord struct {
	MarketStateID         string
	TenantID              string
	AppID                 string
	AssetID               string
	Symbol                string
	SessionDate           time.Time
	AsOfTime              time.Time
	StateSchemaVersion    string
	StatePayloadJSON      []byte
	FeatureObservationIDs []string
	FeatureCount          int
	RequiredFeatureCount  int
	CompletenessRatio     float64
	QualityState          string
	QualityScore          *float64
	QualitySummaryJSON    []byte
	EligibleHypotheses    []string
	BuildRunID            string
	DeterministicKey      string
	CreatedAt             time.Time
}

type MarketOpsMarketStateFilter struct {
	TenantID           string
	AppID              string
	AssetID            string
	Symbol             string
	StateSchemaVersion string
	QualityState       string
	SessionStart       time.Time
	SessionEnd         time.Time
	Limit              int
}

type MarketOpsStateTransitionRecord struct {
	TransitionID          string
	TenantID              string
	AppID                 string
	AssetID               string
	Symbol                string
	SessionDate           time.Time
	AsOfTime              time.Time
	CurrentStateID        string
	BaselineStateID       string
	FeatureKey            string
	FeatureVersion        string
	DimensionsJSON        []byte
	TransitionType        string
	LookbackSessions      *int
	CurrentValue          *float64
	BaselineValue         *float64
	TransitionValue       *float64
	ZScore                *float64
	Percentile            *float64
	PersistenceSessions   *int
	Direction             string
	QualityState          string
	TransitionPayloadJSON []byte
	CalculationRunID      string
	DeterministicKey      string
	CreatedAt             time.Time
}

type MarketOpsStateTransitionFilter struct {
	TenantID       string
	AppID          string
	AssetID        string
	Symbol         string
	CurrentStateID string
	FeatureKey     string
	FeatureVersion string
	TransitionType string
	QualityState   string
	SessionStart   time.Time
	SessionEnd     time.Time
	Limit          int
}

type MarketOpsEvidenceRecord struct {
	EvidenceID          string
	TenantID            string
	AppID               string
	AssetID             string
	Symbol              string
	SessionDate         time.Time
	AsOfTime            time.Time
	EvidenceType        string
	EvidenceVersion     string
	Domain              string
	Direction           string
	Magnitude           *float64
	RarityScore         *float64
	PersistenceScore    *float64
	QualityScore        *float64
	Statement           string
	EvidencePayloadJSON []byte
	SourceFeatureIDs    []string
	SourceTransitionIDs []string
	DeterministicKey    string
	CreatedAt           time.Time
}

type MarketOpsEvidenceFilter struct {
	TenantID        string
	AppID           string
	AssetID         string
	Symbol          string
	EvidenceType    string
	EvidenceVersion string
	Domain          string
	Direction       string
	SessionStart    time.Time
	SessionEnd      time.Time
	Limit           int
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

type MarketOpsBacktestEvaluationLabelRecord struct {
	LabelID          string
	TenantID         string
	AppID            string
	Domain           string
	UseCase          string
	SourceProposalID string
	ArtifactID       string
	SignalID         string
	SubjectSymbol    string
	CandidateType    string
	GraphFactKey     string
	DecisionStatus   string
	Label            string
	LabelSource      string
	LabeledBy        string
	LabeledAt        time.Time
	LabelVersion     string
	MetadataJSON     []byte
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type MarketOpsBacktestEvaluationRecord struct {
	EvaluationID       string
	TenantID           string
	AppID              string
	Domain             string
	UseCase            string
	RunID              string
	DetectorID         string
	Dataset            string
	LabelSource        string
	LabelVersion       string
	ScoringVersion     string
	RequestedBy        string
	CandidateCount     int
	LabeledCount       int
	PositiveCount      int
	NegativeCount      int
	SupersededCount    int
	UnresolvedCount    int
	TruePositive       int
	FalsePositive      int
	TrueNegative       int
	FalseNegative      int
	ManualReviewCount  int
	UnscoredCount      int
	Precision          float64
	Recall             float64
	Specificity        float64
	Accuracy           float64
	LabelCoverage      float64
	Recommendation     string
	RecommendationNote string
	MetricsJSON        []byte
	CreatedAt          time.Time
}

const (
	MarketOpsBacktestPromotionCandidateStatusProposed             = "proposed"
	MarketOpsBacktestPromotionCandidateStatusApprovedForPromotion = "approved_for_promotion"
	MarketOpsBacktestPromotionCandidateStatusRejected             = "rejected"
	MarketOpsBacktestPromotionCandidateStatusDeferred             = "deferred"
	MarketOpsBacktestPromotionCandidateStatusSuperseded           = "superseded"

	MarketOpsBacktestPromotionReadinessReadyForReview       = "ready_for_review"
	MarketOpsBacktestPromotionReadinessNeedsMoreData        = "needs_more_data"
	MarketOpsBacktestPromotionReadinessManualReviewRequired = "manual_review_required"
	MarketOpsBacktestPromotionReadinessRegressionDetected   = "regression_detected"
	MarketOpsBacktestPromotionReadinessBlocked              = "blocked"
)

type MarketOpsBacktestPromotionCandidateRecord struct {
	CandidateID      string
	TenantID         string
	AppID            string
	Domain           string
	UseCase          string
	BaselineID       string
	ComparisonID     string
	EvaluationID     string
	RunID            string
	DetectorID       string
	DetectorVersion  string
	Dataset          string
	PolicyVersion    string
	CandidateVersion string
	ReadinessStatus  string
	ReadinessReasons []string
	EvidenceJSON     []byte
	Status           string
	RequestedBy      string
	ReviewedBy       string
	ReviewedAt       *time.Time
	DecisionNote     string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

const (
	MarketOpsBacktestCalibrationReadinessReady                   = "calibration_ready"
	MarketOpsBacktestCalibrationReadinessNeedsMoreHistoricalData = "needs_more_historical_data"
	MarketOpsBacktestCalibrationReadinessNeedsMoreLabels         = "needs_more_labels"
	MarketOpsBacktestCalibrationReadinessLabelQualityBlocked     = "label_quality_blocked"
	MarketOpsBacktestCalibrationReadinessRegressionDetected      = "regression_detected"
	MarketOpsBacktestCalibrationReadinessManualReviewRequired    = "manual_review_required"
	MarketOpsBacktestCalibrationReadinessBlocked                 = "blocked"
)

type MarketOpsBacktestCalibrationReadinessRecord struct {
	ReadinessID           string
	TenantID              string
	AppID                 string
	Domain                string
	UseCase               string
	BaselineID            string
	ComparisonID          string
	EvaluationID          string
	CandidateID           string
	DetectorID            string
	DatasetScope          []string
	UniverseGroup         string
	WindowStart           *time.Time
	WindowEnd             *time.Time
	ReadinessStatus       string
	ReadinessReasons      []string
	CoverageMetricsJSON   []byte
	LabelMetricsJSON      []byte
	EvaluationMetricsJSON []byte
	ThresholdsJSON        []byte
	EvidenceJSON          []byte
	RequestedBy           string
	CreatedAt             time.Time
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

type MarketOpsBacktestCoverageRecord struct {
	TenantID      string
	AppID         string
	Domain        string
	UseCase       string
	SourceID      string
	SourceAdapter string
	Dataset       string
	SubjectSymbol string
	EventCount    int
	FirstObserved time.Time
	LastObserved  time.Time
}

type MarketOpsBacktestCampaignRecord struct {
	CampaignID      string
	TenantID        string
	AppID           string
	Domain          string
	UseCase         string
	SourceID        string
	SourceAdapter   string
	DetectorID      string
	DetectorVersion string
	RequestedBy     string
	UniverseGroup   string
	DatasetScope    []string
	Symbols         []string
	WindowStart     time.Time
	WindowEnd       time.Time
	WindowStepDays  int
	MaxSymbols      int
	MaxWindows      int
	MaxRuns         int
	MaxRecords      int
	BatchSize       int
	Status          string
	ChildRunIDs     []string
	MetricsJSON     []byte
	ErrorMessage    string
	StartedAt       time.Time
	CompletedAt     *time.Time
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

type SyncraticRepository interface {
	UpsertSyncraticContextWindow(ctx context.Context, record SyncraticContextWindowRecord) error
	ListSyncraticContextWindows(ctx context.Context, filter SyncraticContextWindowFilter) ([]SyncraticContextWindowRecord, error)
	GetSyncraticContextWindow(ctx context.Context, contextWindowID string) (SyncraticContextWindowRecord, error)
	UpsertSyncraticInsight(ctx context.Context, record SyncraticInsightRecord) error
	ListSyncraticInsights(ctx context.Context, filter SyncraticInsightFilter) ([]SyncraticInsightRecord, error)
	GetSyncraticInsight(ctx context.Context, syncraticInsightID string) (SyncraticInsightRecord, error)
}

type MarketOpsMarketStateWriteRepository interface {
	UpsertMarketOpsFeatureDefinition(ctx context.Context, record MarketOpsFeatureDefinitionRecord) error
	UpsertMarketOpsFeatureObservation(ctx context.Context, record MarketOpsFeatureObservationRecord) error
	UpsertMarketOpsMarketState(ctx context.Context, record MarketOpsMarketStateRecord) error
	UpsertMarketOpsStateTransition(ctx context.Context, record MarketOpsStateTransitionRecord) error
	UpsertMarketOpsEvidence(ctx context.Context, record MarketOpsEvidenceRecord) error
}

type MarketOpsMarketStateQueryRepository interface {
	ListMarketOpsFeatureDefinitions(ctx context.Context, filter MarketOpsFeatureDefinitionFilter) ([]MarketOpsFeatureDefinitionRecord, error)
	ListMarketOpsFeatureObservations(ctx context.Context, filter MarketOpsFeatureObservationFilter) ([]MarketOpsFeatureObservationRecord, error)
	ListMarketOpsMarketStates(ctx context.Context, filter MarketOpsMarketStateFilter) ([]MarketOpsMarketStateRecord, error)
	GetMarketOpsMarketState(ctx context.Context, marketStateID string) (MarketOpsMarketStateRecord, error)
	ListMarketOpsStateTransitions(ctx context.Context, filter MarketOpsStateTransitionFilter) ([]MarketOpsStateTransitionRecord, error)
	ListMarketOpsEvidence(ctx context.Context, filter MarketOpsEvidenceFilter) ([]MarketOpsEvidenceRecord, error)
	GetMarketOpsEvidence(ctx context.Context, evidenceID string) (MarketOpsEvidenceRecord, error)
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
	ListMarketOpsBacktestCoverage(ctx context.Context, filter MarketOpsBacktestCoverageFilter) ([]MarketOpsBacktestCoverageRecord, error)
	UpsertMarketOpsBacktestCampaign(ctx context.Context, record MarketOpsBacktestCampaignRecord) error
	ListMarketOpsBacktestCampaigns(ctx context.Context, filter MarketOpsBacktestCampaignFilter) ([]MarketOpsBacktestCampaignRecord, error)
	GetMarketOpsBacktestCampaign(ctx context.Context, campaignID string) (MarketOpsBacktestCampaignRecord, error)
	ListMarketOpsBacktestSignals(ctx context.Context, filter MarketOpsBacktestSignalFilter) ([]MarketOpsBacktestSignalRecord, error)
	ListMarketOpsBacktestGraphProposals(ctx context.Context, filter MarketOpsBacktestGraphProposalFilter) ([]MarketOpsBacktestGraphProposalRecord, error)
	ListMarketOpsBacktestPolicyResults(ctx context.Context, filter MarketOpsBacktestGraphProposalFilter) ([]MarketOpsBacktestPolicyResultRecord, error)
	ListMarketOpsBacktestNormalizedEvents(ctx context.Context, filter MarketOpsBacktestEventFilter) ([]NormalizedEventLedgerRecord, error)
	UpsertMarketOpsBacktestCalibrationSummary(ctx context.Context, record MarketOpsBacktestCalibrationSummaryRecord) error
	ListMarketOpsBacktestCalibrationSummaries(ctx context.Context, filter MarketOpsBacktestCalibrationSummaryFilter) ([]MarketOpsBacktestCalibrationSummaryRecord, error)
	GetMarketOpsBacktestCalibrationSummary(ctx context.Context, summaryID string) (MarketOpsBacktestCalibrationSummaryRecord, error)
	UpsertMarketOpsBacktestEvaluationLabel(ctx context.Context, record MarketOpsBacktestEvaluationLabelRecord) error
	ListMarketOpsBacktestEvaluationLabels(ctx context.Context, filter MarketOpsBacktestEvaluationLabelFilter) ([]MarketOpsBacktestEvaluationLabelRecord, error)
	GetMarketOpsBacktestEvaluationLabel(ctx context.Context, labelID string) (MarketOpsBacktestEvaluationLabelRecord, error)
	UpsertMarketOpsBacktestEvaluation(ctx context.Context, record MarketOpsBacktestEvaluationRecord) error
	ListMarketOpsBacktestEvaluations(ctx context.Context, filter MarketOpsBacktestEvaluationFilter) ([]MarketOpsBacktestEvaluationRecord, error)
	GetMarketOpsBacktestEvaluation(ctx context.Context, evaluationID string) (MarketOpsBacktestEvaluationRecord, error)
	UpsertMarketOpsBacktestPromotionCandidate(ctx context.Context, record MarketOpsBacktestPromotionCandidateRecord) error
	ListMarketOpsBacktestPromotionCandidates(ctx context.Context, filter MarketOpsBacktestPromotionCandidateFilter) ([]MarketOpsBacktestPromotionCandidateRecord, error)
	GetMarketOpsBacktestPromotionCandidate(ctx context.Context, candidateID string) (MarketOpsBacktestPromotionCandidateRecord, error)
	MutateMarketOpsBacktestPromotionCandidateDecision(ctx context.Context, mutation MarketOpsBacktestPromotionCandidateDecisionMutation) (MarketOpsBacktestPromotionCandidateRecord, error)
	UpsertMarketOpsBacktestCalibrationReadiness(ctx context.Context, record MarketOpsBacktestCalibrationReadinessRecord) error
	ListMarketOpsBacktestCalibrationReadiness(ctx context.Context, filter MarketOpsBacktestCalibrationReadinessFilter) ([]MarketOpsBacktestCalibrationReadinessRecord, error)
	GetMarketOpsBacktestCalibrationReadiness(ctx context.Context, readinessID string) (MarketOpsBacktestCalibrationReadinessRecord, error)
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

type AlgorithmRepository interface {
	UpsertAlgorithmDefinition(ctx context.Context, record AlgorithmDefinitionRecord) error
	ListAlgorithmDefinitions(ctx context.Context, filter AlgorithmDefinitionFilter) ([]AlgorithmDefinitionRecord, error)
	GetAlgorithmDefinition(ctx context.Context, tenantID string, algorithmID string) (AlgorithmDefinitionRecord, error)
	UpsertAlgorithmExecutionRequest(ctx context.Context, record AlgorithmExecutionRequestRecord) error
	ListAlgorithmExecutionRequests(ctx context.Context, filter AlgorithmExecutionRequestFilter) ([]AlgorithmExecutionRequestRecord, error)
	GetAlgorithmExecutionRequest(ctx context.Context, tenantID string, executionRequestID string) (AlgorithmExecutionRequestRecord, error)
	InsertAlgorithmResult(ctx context.Context, record AlgorithmResultRecord) error
	ListAlgorithmResults(ctx context.Context, filter AlgorithmResultFilter) ([]AlgorithmResultRecord, error)
	GetAlgorithmResult(ctx context.Context, tenantID string, algorithmResultID string) (AlgorithmResultRecord, error)
	InsertAlgorithmSignalProposal(ctx context.Context, record AlgorithmSignalProposalRecord) (bool, error)
	ListAlgorithmSignalProposals(ctx context.Context, filter AlgorithmSignalProposalFilter) ([]AlgorithmSignalProposalRecord, error)
	GetAlgorithmSignalProposal(ctx context.Context, tenantID string, proposalID string) (AlgorithmSignalProposalRecord, error)
	SummarizeAlgorithmSignalProposals(ctx context.Context, filter AlgorithmSignalProposalFilter) (AlgorithmSignalProposalSummaryRecord, error)
	MutateAlgorithmSignalProposal(ctx context.Context, mutation AlgorithmSignalProposalMutation) (AlgorithmSignalProposalRecord, error)
	UpsertAlgorithmSignalMaterialization(ctx context.Context, record AlgorithmSignalMaterializationRecord) (AlgorithmSignalMaterializationRecord, error)
	ListAlgorithmSignalMaterializations(ctx context.Context, filter AlgorithmSignalMaterializationFilter) ([]AlgorithmSignalMaterializationRecord, error)
	GetAlgorithmSignalMaterialization(ctx context.Context, tenantID string, materializationID string) (AlgorithmSignalMaterializationRecord, error)
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

type SyncraticContextWindowFilter struct {
	TenantID        string
	AppID           string
	Domain          string
	UseCase         string
	SubjectSymbol   string
	ContextStrategy string
	Status          string
	Limit           int
}

type SyncraticInsightFilter struct {
	TenantID        string
	AppID           string
	Domain          string
	UseCase         string
	ContextWindowID string
	InsightType     string
	SubjectSymbol   string
	Status          string
	Limit           int
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

type MarketOpsBacktestCoverageFilter struct {
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

type MarketOpsBacktestCampaignFilter struct {
	TenantID      string
	AppID         string
	Domain        string
	UseCase       string
	SourceID      string
	DetectorID    string
	UniverseGroup string
	Status        string
	Limit         int
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

type MarketOpsBacktestEvaluationLabelFilter struct {
	TenantID         string
	AppID            string
	Domain           string
	UseCase          string
	SourceProposalID string
	ArtifactID       string
	SignalID         string
	SubjectSymbol    string
	CandidateType    string
	DecisionStatus   string
	Label            string
	LabelSource      string
	Limit            int
}

type MarketOpsBacktestEvaluationFilter struct {
	TenantID       string
	AppID          string
	Domain         string
	UseCase        string
	RunID          string
	DetectorID     string
	Dataset        string
	Recommendation string
	Limit          int
}

type MarketOpsBacktestPromotionCandidateFilter struct {
	TenantID        string
	AppID           string
	Domain          string
	UseCase         string
	BaselineID      string
	ComparisonID    string
	EvaluationID    string
	RunID           string
	DetectorID      string
	Dataset         string
	ReadinessStatus string
	Status          string
	Limit           int
}

type MarketOpsBacktestPromotionCandidateDecisionMutation struct {
	CandidateID  string
	Status       string
	ReviewedBy   string
	ReviewedAt   time.Time
	DecisionNote string
}

type MarketOpsBacktestCalibrationReadinessFilter struct {
	TenantID        string
	AppID           string
	Domain          string
	UseCase         string
	BaselineID      string
	ComparisonID    string
	EvaluationID    string
	CandidateID     string
	DetectorID      string
	ReadinessStatus string
	Limit           int
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
	SyncraticRepository
	AlgorithmRepository
	MarketOpsMarketStateQueryRepository
	SignalLedgerRepository
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
	ListMarketOpsOptionsChain(ctx context.Context, filter MarketOpsOptionsChainFilter) ([]MarketOpsOptionsChainRecord, error)
	GetMarketOpsOptionsCoverage(ctx context.Context, tenantID string, symbol string) (MarketOpsOptionsCoverageRecord, error)
	ListMarketOpsOptionsDistributions(ctx context.Context, filter MarketOpsOptionsDistributionFilter) ([]MarketOpsOptionsDistributionRecord, error)
}

package storage

import (
	"context"
	"time"
)

const (
	MarketOpsAlgorithmEvaluationModeRetrospective = "retrospective"
	MarketOpsAlgorithmEvaluationModeWalkForward   = "walk_forward"

	MarketOpsAlgorithmEvaluationStatusRunning   = "running"
	MarketOpsAlgorithmEvaluationStatusSucceeded = "succeeded"
	MarketOpsAlgorithmEvaluationStatusPartial   = "partial"
	MarketOpsAlgorithmEvaluationStatusFailed    = "failed"

	MarketOpsAlgorithmEvaluationProfileDirectional    = "directional"
	MarketOpsAlgorithmEvaluationProfileEventStudy     = "event_study"
	MarketOpsAlgorithmEvaluationProfileForecast       = "forecast"
	MarketOpsAlgorithmEvaluationProfileClassification = "classification"

	MarketOpsAlgorithmBackfillStatusPlanned   = "planned"
	MarketOpsAlgorithmBackfillStatusRunning   = "running"
	MarketOpsAlgorithmBackfillStatusSucceeded = "succeeded"
	MarketOpsAlgorithmBackfillStatusPartial   = "partial"
	MarketOpsAlgorithmBackfillStatusFailed    = "failed"
)

type MarketOpsAlgorithmEvaluationRunRecord struct {
	RunID          string
	TenantID       string
	AppID          string
	UniverseGroup  string
	AlgorithmIDs   []string
	Modes          []string
	WindowStart    time.Time
	WindowEnd      time.Time
	AsOfDate       time.Time
	Status         string
	ParametersJSON []byte
	CoverageJSON   []byte
	MetricsJSON    []byte
	ErrorMessage   string
	RequestedBy    string
	StartedAt      time.Time
	CompletedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type MarketOpsAlgorithmEvaluationRunFilter struct {
	TenantID    string
	AlgorithmID string
	Status      string
	Limit       int
}

type MarketOpsAlgorithmEvaluationResultRecord struct {
	EvaluationResultID     string
	RunID                  string
	TenantID               string
	AlgorithmID            string
	AlgorithmVersion       string
	EvaluationMode         string
	EvaluationProfile      string
	ResultType             string
	Symbol                 string
	ObservationSessionDate time.Time
	Score                  float64
	Confidence             float64
	Severity               string
	Direction              string
	ResultPayloadJSON      []byte
	InputProvenanceJSON    []byte
	SourceEventIDs         []string
	FeatureValueIDs        []string
	DeterministicKey       string
	CreatedAt              time.Time
}

type MarketOpsAlgorithmEvaluationResultFilter struct {
	TenantID       string
	RunID          string
	AlgorithmID    string
	Symbol         string
	EvaluationMode string
	Limit          int
}

type MarketOpsAlgorithmEvaluationOutcomeRecord struct {
	EvaluationOutcomeID   string
	RunID                 string
	EvaluationResultID    string
	TenantID              string
	HorizonSessions       int
	OutcomeStatus         string
	MaturedSessionDate    *time.Time
	ForwardReturn         *float64
	AbsoluteForwardReturn *float64
	MaxFavorableExcursion *float64
	MaxAdverseExcursion   *float64
	MaximumDrawdown       *float64
	RealizedVolChange     *float64
	DirectionalHit        *bool
	ThresholdHit          *bool
	OutcomeEventIDs       []string
	OutcomePayloadJSON    []byte
	DeterministicKey      string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type MarketOpsAlgorithmEvaluationOutcomeFilter struct {
	TenantID           string
	RunID              string
	EvaluationResultID string
	OutcomeStatus      string
	HorizonSessions    int
	Limit              int
}

type MarketOpsAlgorithmEvaluationBackfillCampaignRecord struct {
	CampaignID     string
	TenantID       string
	UniverseGroup  string
	WindowStart    time.Time
	WindowEnd      time.Time
	Status         string
	ParametersJSON []byte
	CoverageJSON   []byte
	ChildRunIDs    []string
	ErrorMessage   string
	RequestedBy    string
	StartedAt      *time.Time
	CompletedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type MarketOpsAlgorithmEvaluationBackfillCampaignFilter struct {
	TenantID string
	Status   string
	Limit    int
}

type MarketOpsAlgorithmEvaluationRepository interface {
	UpsertMarketOpsAlgorithmEvaluationRun(context.Context, MarketOpsAlgorithmEvaluationRunRecord) error
	GetMarketOpsAlgorithmEvaluationRun(context.Context, string, string) (MarketOpsAlgorithmEvaluationRunRecord, error)
	ListMarketOpsAlgorithmEvaluationRuns(context.Context, MarketOpsAlgorithmEvaluationRunFilter) ([]MarketOpsAlgorithmEvaluationRunRecord, error)
	InsertMarketOpsAlgorithmEvaluationResult(context.Context, MarketOpsAlgorithmEvaluationResultRecord) error
	ListMarketOpsAlgorithmEvaluationResults(context.Context, MarketOpsAlgorithmEvaluationResultFilter) ([]MarketOpsAlgorithmEvaluationResultRecord, error)
	UpsertMarketOpsAlgorithmEvaluationOutcome(context.Context, MarketOpsAlgorithmEvaluationOutcomeRecord) error
	ListMarketOpsAlgorithmEvaluationOutcomes(context.Context, MarketOpsAlgorithmEvaluationOutcomeFilter) ([]MarketOpsAlgorithmEvaluationOutcomeRecord, error)
	UpsertMarketOpsAlgorithmEvaluationBackfillCampaign(context.Context, MarketOpsAlgorithmEvaluationBackfillCampaignRecord) error
	GetMarketOpsAlgorithmEvaluationBackfillCampaign(context.Context, string, string) (MarketOpsAlgorithmEvaluationBackfillCampaignRecord, error)
	ListMarketOpsAlgorithmEvaluationBackfillCampaigns(context.Context, MarketOpsAlgorithmEvaluationBackfillCampaignFilter) ([]MarketOpsAlgorithmEvaluationBackfillCampaignRecord, error)
}

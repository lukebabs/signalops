package storage

import (
	"context"
	"encoding/json"
	"time"
)

const (
	SignalAssuranceModeLive     = "LIVE"
	SignalAssuranceModeBacktest = "BACKTEST"
	SignalAssuranceModeResearch = "RESEARCH"

	SignalAssertionActive       = "ACTIVE"
	SignalAssertionMaterialized = "MATERIALIZED"
	SignalAssertionInvalidated  = "INVALIDATED"
	SignalAssertionSuperseded   = "SUPERSEDED"
	SignalAssertionExpired      = "EXPIRED"
	SignalAssertionClosed       = "CLOSED"

	SignalAssuranceInputComplete   = "COMPLETE"
	SignalAssuranceInputIncomplete = "INCOMPLETE"
)

// SignalValidationContractRecord is append-only policy used to evaluate one
// assertion. Changes create a new contract ID/version rather than mutating a
// contract captured by an existing assertion.
type SignalValidationContractRecord struct {
	ContractID            string
	SignalType            string
	ContractVersion       string
	Algorithm             string
	AlgorithmVersion      string
	Direction             string
	PrimaryMetric         string
	ComparisonOperator    string
	Threshold             float64
	EvaluationWindowsJSON []byte
	MaxHorizonTradingDays int
	MaterializationPolicy string
	InvalidationPolicy    string
	ConfigJSON            []byte
	Active                bool
	ContractScopeKey      string
	CreatedAt             time.Time
}

type SignalAssuranceEligibleEvent struct {
	EligibleEventID         string          `json:"eligible_event_id"`
	TenantID                string          `json:"tenant_id"`
	SignalID                string          `json:"signal_id"`
	SignalLedgerID          string          `json:"signal_ledger_id"`
	AssetID                 string          `json:"asset_id"`
	Symbol                  string          `json:"symbol"`
	SignalType              string          `json:"signal_type"`
	Direction               string          `json:"direction"`
	Score                   *float64        `json:"score,omitempty"`
	Confidence              *float64        `json:"confidence,omitempty"`
	Status                  string          `json:"status"`
	Algorithm               string          `json:"algorithm"`
	AlgorithmVersion        string          `json:"algorithm_version"`
	ConfirmedAt             time.Time       `json:"confirmed_at"`
	EventAvailableAt        time.Time       `json:"event_available_at"`
	ConfirmationRuleVersion string          `json:"confirmation_rule_version"`
	ValidationContractRef   string          `json:"validation_contract_ref"`
	BaselineSnapshot        json.RawMessage `json:"baseline_snapshot"`
	BaselineProvenance      json.RawMessage `json:"baseline_provenance"`
	EvaluationMode          string          `json:"evaluation_mode"`
	EvaluationRunID         string          `json:"evaluation_run_id,omitempty"`
}

type SignalAssertionRecord struct {
	AssertionID                string
	TenantID                   string
	AssetID                    string
	Symbol                     string
	SignalID                   string
	SourceLedgerSignalID       string
	SignalType                 string
	SignalDirection            string
	SignalScore                *float64
	Confidence                 *float64
	Algorithm                  string
	AlgorithmVersion           string
	ConfirmedAt                time.Time
	State                      string
	EvaluationMode             string
	EvaluationRunID            string
	RegistrationIdempotencyKey string
	ValidationContractID       string
	ValidationContractVersion  string
	ValidationContractJSON     []byte
	EvaluationEngineVersion    string
	BaselineSnapshotJSON       []byte
	BaselineProvenanceJSON     []byte
	MaterializedAt             *time.Time
	InvalidatedAt              *time.Time
	SupersededAt               *time.Time
	ExpiredAt                  *time.Time
	ClosedAt                   *time.Time
	TransitionSequence         int
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}

type SignalAssertionEvaluationRecord struct {
	EvaluationID                string
	AssertionID                 string
	EvaluatedAt                 time.Time
	EvaluationSessionDate       time.Time
	EvaluationMode              string
	EvaluationRunID             string
	InputSnapshotJSON           []byte
	InputCompleteness           string
	TransitionSequence          int
	TradingDaysActive           int
	CalendarDaysActive          int
	AssetPrice                  *float64
	BenchmarkPrice              *float64
	AbsoluteReturn              *float64
	BenchmarkReturn             *float64
	BenchmarkRelativeReturn     *float64
	SectorRelativeReturn        *float64
	MFE                         *float64
	MAE                         *float64
	MaterializationConditionMet bool
	InvalidationConditionMet    bool
	EvaluationVersion           string
	CreatedAt                   time.Time
}

type SignalAssertionEventRecord struct {
	EventID            string
	AssertionID        string
	EventType          string
	PreviousState      string
	NewState           string
	ReasonCode         string
	DetailsJSON        []byte
	OccurredAt         time.Time
	TransitionSequence int
	EvaluationID       string
	EvaluationMode     string
	EvaluationRunID    string
	IdempotencyKey     string
	PublishedAt        *time.Time
}

type SignalAssuranceRegistration struct {
	Event       SignalAssuranceEligibleEvent
	Contract    SignalValidationContractRecord
	Assertion   SignalAssertionRecord
	PayloadJSON []byte
}

type SignalAssuranceEvaluationPersistence struct {
	Evaluation       SignalAssertionEvaluationRecord
	PreviousState    string
	NextState        string
	ReasonCode       string
	EventDetailsJSON []byte
}

type SignalAssuranceAssertionFilter struct {
	TenantID       string
	State          string
	EvaluationMode string
	Symbol         string
	Limit          int
}

type SignalAssuranceEvaluationFilter struct {
	AssertionID string
	Limit       int
}

type SignalAssuranceEffectivenessFilter struct {
	TenantID         string
	EvidenceSource   string
	EvaluationMode   string
	Dimension        string
	DimensionValue   string
	OutcomeNotBefore *time.Time
	Limit            int
}

type SignalAssuranceEffectivenessRecord struct {
	EvidenceSource                 string
	Dimension                      string
	DimensionValue                 string
	SampleSize                     int
	DirectionalHits                int
	MaterializedCount              int
	InvalidatedCount               int
	ExpiredCount                   int
	CensoredCount                  int
	ExcludedCount                  int
	DirectionalAccuracy            *float64
	AccuracyLowerBound             *float64
	AccuracyUpperBound             *float64
	MaterializationRate            *float64
	AverageReturn                  *float64
	AverageRelativeReturn          *float64
	AverageSectorRelativeReturn    *float64
	BroadMarketBenchmarkSampleSize int
	SectorBenchmarkSampleSize      int
	AverageMFE                     *float64
	AverageMAE                     *float64
	Exploratory                    bool
	AsOf                           time.Time
	MetricDefinitionVersion        string
}

type SignalAssuranceRecommendationRecord struct {
	RecommendationID        string
	EvidenceSource          string
	Dimension               string
	DimensionValue          string
	Priority                string
	Kind                    string
	Summary                 string
	SampleSize              int
	DirectionalAccuracy     *float64
	AccuracyUpperBound      *float64
	MetricDefinitionVersion string
	AsOf                    time.Time
}

// SignalAssuranceEffectivenessObservationRecord is one terminal, complete
// observation included in an effectiveness cohort. ReferenceID identifies the
// immutable source record: an SAF assertion or the historical opportunity.
// Consumers use it to retrieve the existing, read-only audit/provenance view.
type SignalAssuranceEffectivenessObservationRecord struct {
	EvidenceSource            string
	ObservationID             string
	ReferenceID               string
	Symbol                    string
	SignalType                string
	Direction                 string
	Algorithm                 string
	AlgorithmVersion          string
	State                     string
	EvaluationMode            string
	HorizonSessions           int
	SignalScore               *float64
	Confidence                *float64
	DirectionalHit            *bool
	AbsoluteReturn            *float64
	DirectionalReturn         *float64
	RelativeReturn            *float64
	SectorRelativeReturn      *float64
	MFE                       *float64
	MAE                       *float64
	OriginAt                  *time.Time
	OutcomeAt                 *time.Time
	CalculationVersion        string
	CalculationRunID          string
	BroadMarketBenchmarkState string
	SectorBenchmarkState      string
}

type SignalAssuranceEffectivenessRepository interface {
	ListSignalAssuranceEffectiveness(context.Context, SignalAssuranceEffectivenessFilter) ([]SignalAssuranceEffectivenessRecord, error)
	ListSignalAssuranceEffectivenessObservations(context.Context, SignalAssuranceEffectivenessFilter) ([]SignalAssuranceEffectivenessObservationRecord, error)
	ListSignalAssuranceRecommendations(context.Context, SignalAssuranceEffectivenessFilter) ([]SignalAssuranceRecommendationRecord, error)
}

type SignalAssuranceQueryRepository interface {
	ListSignalAssuranceAssertions(context.Context, SignalAssuranceAssertionFilter) ([]SignalAssertionRecord, error)
	GetSignalAssuranceAssertion(context.Context, string, string) (SignalAssertionRecord, error)
	ListSignalAssuranceEvaluations(context.Context, SignalAssuranceEvaluationFilter) ([]SignalAssertionEvaluationRecord, error)
	GetSignalValidationContract(context.Context, string) (SignalValidationContractRecord, error)
	SignalAssuranceEffectivenessRepository
}

type SignalAssuranceWriteRepository interface {
	UpsertSignalValidationContract(context.Context, SignalValidationContractRecord) error
	RegisterSignalAssuranceAssertion(context.Context, SignalAssuranceRegistration) (SignalAssertionRecord, bool, error)
	PersistSignalAssuranceEvaluation(context.Context, SignalAssuranceEvaluationPersistence) (SignalAssertionEvaluationRecord, bool, error)
}

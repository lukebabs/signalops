package storage

import (
	"context"
	"time"
)

const (
	MarketOpsHypothesisLifecycleDraft         = "draft"
	MarketOpsHypothesisLifecycleResearch      = "research"
	MarketOpsHypothesisLifecycleBacktestReady = "backtest_ready"
	MarketOpsHypothesisLifecycleCalibration   = "calibration"
	MarketOpsHypothesisLifecycleCandidate     = "candidate"
	MarketOpsHypothesisLifecycleApproved      = "approved"
	MarketOpsHypothesisLifecyclePaused        = "paused"
	MarketOpsHypothesisLifecycleRetired       = "retired"
)

type MarketOpsHypothesisDefinitionRecord struct {
	TenantID                  string
	HypothesisKey             string
	HypothesisVersion         string
	Title                     string
	Domain                    string
	Direction                 string
	Description               string
	Rationale                 string
	RequiredFeaturesJSON      []byte
	RequiredTransitionsJSON   []byte
	QualityPolicyJSON         []byte
	EligibilityExpressionJSON []byte
	TriggerExpressionJSON     []byte
	PersistenceRuleJSON       []byte
	CorroborationRuleJSON     []byte
	InvalidationRuleJSON      []byte
	ExpectedOutcomesJSON      []byte
	ScoringConfigJSON         []byte
	CalibrationPolicyJSON     []byte
	LifecycleStatus           string
	Owner                     string
	ApprovedBy                string
	ApprovedAt                *time.Time
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

type MarketOpsHypothesisDefinitionFilter struct {
	TenantID          string
	HypothesisKey     string
	HypothesisVersion string
	Domain            string
	LifecycleStatus   string
	Limit             int
}

type MarketOpsHypothesisEvaluationRecord struct {
	EvaluationID          string
	TenantID              string
	AppID                 string
	HypothesisKey         string
	HypothesisVersion     string
	MarketStateID         string
	AssetID               string
	Symbol                string
	SessionDate           time.Time
	AsOfTime              time.Time
	Eligible              bool
	Triggered             bool
	TriggerScore          *float64
	ConfidenceScore       *float64
	MagnitudeScore        *float64
	RarityScore           *float64
	PersistenceScore      *float64
	CorroborationScore    *float64
	QualityScore          *float64
	Invalidated           bool
	EvidenceIDs           []string
	ReasonCodes           []string
	EvaluationPayloadJSON []byte
	EvaluationRunID       string
	DeterministicKey      string
	CreatedAt             time.Time
}

type MarketOpsHypothesisEvaluationFilter struct {
	TenantID          string
	AppID             string
	HypothesisKey     string
	HypothesisVersion string
	MarketStateID     string
	AssetID           string
	Symbol            string
	Eligible          *bool
	Triggered         *bool
	Invalidated       *bool
	SessionStart      time.Time
	SessionEnd        time.Time
	Limit             int
}

type MarketOpsHypothesisWriteRepository interface {
	UpsertMarketOpsHypothesisDefinition(context.Context, MarketOpsHypothesisDefinitionRecord) error
	UpsertMarketOpsHypothesisEvaluation(context.Context, MarketOpsHypothesisEvaluationRecord) error
}

type MarketOpsHypothesisQueryRepository interface {
	ListMarketOpsHypothesisDefinitions(context.Context, MarketOpsHypothesisDefinitionFilter) ([]MarketOpsHypothesisDefinitionRecord, error)
	GetMarketOpsHypothesisDefinition(context.Context, string, string, string) (MarketOpsHypothesisDefinitionRecord, error)
	ListMarketOpsHypothesisEvaluations(context.Context, MarketOpsHypothesisEvaluationFilter) ([]MarketOpsHypothesisEvaluationRecord, error)
}

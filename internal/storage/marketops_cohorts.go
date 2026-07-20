package storage

import (
	"context"
	"time"
)

const (
	MarketOpsCohortRunPlanned   = "planned"
	MarketOpsCohortRunRunning   = "running"
	MarketOpsCohortRunSucceeded = "succeeded"
	MarketOpsCohortRunPartial   = "partial"
	MarketOpsCohortRunFailed    = "failed"
	MarketOpsCohortRunDryRun    = "dry_run"
)

type MarketOpsIntelligenceCohortRunRecord struct {
	RunID            string
	TenantID         string
	AppID            string
	UniverseGroup    string
	RequestedSymbols []string
	ResolvedSymbols  []string
	Stages           []string
	MaxSymbols       int
	DryRun           bool
	ContinueOnError  bool
	Status           string
	AggregateJSON    []byte
	ErrorsJSON       []byte
	Actor            string
	SessionStart     time.Time
	SessionEnd       time.Time
	StartedAt        time.Time
	CompletedAt      *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type MarketOpsIntelligenceCohortSymbolResultRecord struct {
	ResultID                   string
	RunID                      string
	TenantID                   string
	UniverseGroup              string
	Symbol                     string
	AssetID                    string
	StageStatusJSON            []byte
	StageErrorsJSON            []byte
	InputCoverageJSON          []byte
	LatestMarketStateID        string
	LatestStateDate            *time.Time
	LatestStateSchemaVersion   string
	LatestStateQuality         string
	LatestStateCompleteness    float64
	RequiredFeatureCoverage    float64
	SurfaceCoverage            float64
	EvaluationCount            int
	EligibleCount              int
	TriggeredCount             int
	EvaluationRejectionReasons []string
	OpportunityCount           int
	PendingOutcomeCount        int
	MaturedOutcomeCount        int
	ProposalStatusCountsJSON   []byte
	ExactCalibrationCount      int
	CalibrationBelowMinimum    bool
	CoverageState              string
	EvaluationState            string
	GovernanceState            string
	CalibrationState           string
	OutcomeState               string
	RolloutStatus              string
	ReadinessReasons           []string
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}

type MarketOpsIntelligenceCohortRunFilter struct {
	TenantID      string
	UniverseGroup string
	Status        string
	Limit         int
}

type MarketOpsIntelligenceReadinessFilter struct {
	TenantID      string
	UniverseGroup string
	Symbols       []string
	LatestSession time.Time
	RolloutStatus string
	Limit         int
}

type MarketOpsIntelligenceCohortRepository interface {
	UpsertMarketOpsIntelligenceCohortRun(context.Context, MarketOpsIntelligenceCohortRunRecord) error
	UpsertMarketOpsIntelligenceCohortSymbolResult(context.Context, MarketOpsIntelligenceCohortSymbolResultRecord) error
	ListMarketOpsIntelligenceCohortRuns(context.Context, MarketOpsIntelligenceCohortRunFilter) ([]MarketOpsIntelligenceCohortRunRecord, error)
	GetMarketOpsIntelligenceCohortRun(context.Context, string, string) (MarketOpsIntelligenceCohortRunRecord, error)
	ListMarketOpsIntelligenceCohortSymbolResults(context.Context, string, string) ([]MarketOpsIntelligenceCohortSymbolResultRecord, error)
	ListMarketOpsIntelligenceReadiness(context.Context, MarketOpsIntelligenceReadinessFilter) ([]MarketOpsIntelligenceCohortSymbolResultRecord, error)
}

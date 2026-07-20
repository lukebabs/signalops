package storage

import (
	"context"
	"time"
)

const (
	MarketOpsOpportunityEmerging                     = "emerging"
	MarketOpsOpportunityActive                       = "active"
	MarketOpsOpportunityStrengthening                = "strengthening"
	MarketOpsOpportunityWeakening                    = "weakening"
	MarketOpsOpportunityInvalidated                  = "invalidated"
	MarketOpsOpportunityResolved                     = "resolved"
	MarketOpsOpportunityExpired                      = "expired"
	MarketOpsOpportunityDispositionWatch             = "watch"
	MarketOpsOpportunityDispositionAdvance           = "advance"
	MarketOpsOpportunityDispositionNeedsMoreEvidence = "needs_more_evidence"
	MarketOpsOpportunityDispositionDismiss           = "dismiss"
	MarketOpsOpportunityDispositionResolved          = "resolved"
)

type MarketOpsOpportunityRecord struct {
	OpportunityID            string
	TenantID                 string
	AppID                    string
	AssetID                  string
	Symbol                   string
	OpenedSessionDate        time.Time
	LastEvaluatedDate        time.Time
	Direction                string
	Horizon                  string
	LifecycleStatus          string
	OpportunityScore         float64
	ConfidenceScore          float64
	DomainDiversityScore     float64
	ConflictScore            float64
	HypothesisEvaluationIDs  []string
	ConflictingEvaluationIDs []string
	SignalIDs                []string
	SupportingEvidenceIDs    []string
	InvalidatingEvidenceIDs  []string
	Summary                  string
	OpportunityPayloadJSON   []byte
	Version                  int
	ResearchOnly             bool
	BuildRunID               string
	DeterministicKey         string
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

type MarketOpsOpportunityFilter struct {
	TenantID        string
	AppID           string
	OpportunityID   string
	AssetID         string
	Symbol          string
	Direction       string
	Horizon         string
	LifecycleStatus string
	ResearchOnly    *bool
	SessionStart    time.Time
	SessionEnd      time.Time
	Limit           int
}

type MarketOpsOpportunityDispositionRecord struct {
	DispositionID string
	TenantID      string
	OpportunityID string
	Disposition   string
	Actor         string
	Note          string
	MetadataJSON  []byte
	CreatedAt     time.Time
}

type MarketOpsOpportunityDispositionFilter struct {
	TenantID      string
	OpportunityID string
	Disposition   string
	Limit         int
}

type MarketOpsOpportunityWriteRepository interface {
	UpsertMarketOpsOpportunity(context.Context, MarketOpsOpportunityRecord) error
	InsertMarketOpsOpportunityDisposition(context.Context, MarketOpsOpportunityDispositionRecord) error
}

type MarketOpsOpportunityQueryRepository interface {
	InsertMarketOpsOpportunityDisposition(context.Context, MarketOpsOpportunityDispositionRecord) error
	ListMarketOpsOpportunities(context.Context, MarketOpsOpportunityFilter) ([]MarketOpsOpportunityRecord, error)
	GetMarketOpsOpportunity(context.Context, string, string) (MarketOpsOpportunityRecord, error)
	ListMarketOpsOpportunityDispositions(context.Context, MarketOpsOpportunityDispositionFilter) ([]MarketOpsOpportunityDispositionRecord, error)
}

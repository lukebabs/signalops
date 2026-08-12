package storage

import (
	"context"
	"time"
)

const SubscriberGlobalCatalogShadowVersion = "s1-shadow-v1"

// SubscriberGlobalCatalogSeedRequest identifies a controlled platform-owned
// import. It is not tenant browser input and does not grant tenant access.
type SubscriberGlobalCatalogSeedRequest struct {
	SourceTenantID string
	SeedRunID      string
	ActorIdentity  string
	CorrelationID  string
	ObservedAt     time.Time
}

type SubscriberGlobalCatalogSeedResult struct {
	SeedRunID            string
	SourceTenantID       string
	SourceRows           int
	ActiveSourceRows     int
	DistinctGlobalAssets int
	InsertedGlobalAssets int
	ObservedReferences   int
	CompletedAt          time.Time
}

// SubscriberGlobalCatalogParityRecord is read-only S1 compatibility evidence;
// it is intentionally not a tenant-facing projection.
type SubscriberGlobalCatalogParityRecord struct {
	SourceTenantID        string
	SourceUniverseGroup   string
	SourceTicker          string
	SourceIsActive        bool
	GlobalAssetID         string
	CanonicalSymbol       string
	EligibilityStatus     string
	CoverageState         string
	CoverageExecutionMode string
	ActiveSourceRows      int
}

type SubscriberGlobalCatalogRepository interface {
	SeedSubscriberGlobalCatalogShadow(context.Context, SubscriberGlobalCatalogSeedRequest) (SubscriberGlobalCatalogSeedResult, error)
	ListSubscriberGlobalCatalogParity(context.Context, string, int) ([]SubscriberGlobalCatalogParityRecord, error)
}

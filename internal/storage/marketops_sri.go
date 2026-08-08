package storage

import (
	"context"
	"time"
)

const (
	MarketOpsSRIStateLeading   = "LEADING"
	MarketOpsSRIStateImproving = "IMPROVING"
	MarketOpsSRIStateNeutral   = "NEUTRAL"
	MarketOpsSRIStateWeakening = "WEAKENING"
	MarketOpsSRIStateLagging   = "LAGGING"
)

type MarketOpsSRISegmentRecord struct {
	TenantID, SegmentID, SegmentKey, Name, SegmentType, ParentSegmentKey, RegistryVersion string
	Active                                                                                bool
	MetadataJSON                                                                          []byte
}
type MarketOpsSRIETFRecord struct {
	TenantID, ETFSymbol, SegmentID, Role, RegistryVersion string
	BenchmarkPriority                                     int
	Active                                                bool
	ConfigJSON                                            []byte
}
type MarketOpsSRISnapshotRecord struct {
	SnapshotID, TenantID, SegmentID, State, QualityState, AlgorithmVersion, ConfigurationVersion, CalculationRunID string
	SessionDate, AsOfTime                                                                                          time.Time
	CompositeScore, RelativeStrengthScore, MomentumScore, MomentumAcceleration                                     *float64
	Rank, RankChange5D                                                                                             *int
	EvidenceQuality                                                                                                *float64
	QualityFlagsJSON, ComponentsJSON, InputProvenanceJSON                                                          []byte
	DeterministicKey                                                                                               string
}
type MarketOpsSRISnapshotFilter struct {
	TenantID, SegmentID, SegmentType, State string
	SessionStart, SessionEnd                time.Time
	Limit                                   int
}
type MarketOpsSRIRepository interface {
	ListMarketOpsSRISegments(context.Context, string, bool, int) ([]MarketOpsSRISegmentRecord, error)
	ListMarketOpsSRIETFRegistry(context.Context, string, string) ([]MarketOpsSRIETFRecord, error)
	UpsertMarketOpsSRISegment(context.Context, MarketOpsSRISegmentRecord) error
	UpsertMarketOpsSRIETF(context.Context, MarketOpsSRIETFRecord) error
	UpsertMarketOpsSRISnapshot(context.Context, MarketOpsSRISnapshotRecord) error
	ListMarketOpsSRISnapshots(context.Context, MarketOpsSRISnapshotFilter) ([]MarketOpsSRISnapshotRecord, error)
}

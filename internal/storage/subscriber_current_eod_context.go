package storage

import (
	"context"
	"time"
)

// SubscriberCurrentEODContextRecord is the narrow, tenant-authorized current
// MarketOps EOD projection. It never represents historical assurance input.
type SubscriberCurrentEODContextRecord struct {
	GlobalAssetID           string
	Symbol                  string
	SessionDate             time.Time
	Open                    *float64
	High                    *float64
	Low                     *float64
	Close                   *float64
	Volume                  *int64
	VWAP                    *float64
	Provider                string
	SelectedObservationRole string
	SelectionPolicyVersion  string
	PayloadFingerprint      string
	SourceEventID           string
	SourceRunID             string
	AlgorithmVersion        string
	QualityState            string
	AsOfTime                time.Time
}

// SubscriberCurrentEODContextRepository only returns a row when the requested
// symbol is active in the caller tenant's MarketOps universe.
type SubscriberCurrentEODContextRepository interface {
	GetSubscriberCurrentEODContext(context.Context, string, string) (SubscriberCurrentEODContextRecord, error)
}

// SubscriberCurrentEODContextBatchRepository returns latest global EOD context
// for an already-authorized set of watchlist symbols in one query. Missing symbols
// are omitted; callers must authorize every supplied symbol before invocation.
type SubscriberCurrentEODContextBatchRepository interface {
	ListSubscriberCurrentEODContexts(context.Context, []string) ([]SubscriberCurrentEODContextRecord, error)
}

type SubscriberGlobalRiskRewardSnapshotRepository interface {
	ListSubscriberGlobalRiskRewardSnapshots(context.Context, []string, time.Time, int) ([]MarketOpsRiskRewardSnapshotRecord, error)
}

type SubscriberGlobalOptionsDistributionRepository interface {
	ListSubscriberGlobalOptionsDistributions(context.Context, []string, time.Time, int) ([]MarketOpsOptionsDistributionRecord, error)
}

// SubscriberGlobalMarketStateRepository returns only the platform-owned,
// parity-approved Market State projection. Callers authorize symbols through
// a selected watchlist before invoking it.
type SubscriberGlobalMarketStateRepository interface {
	ListSubscriberGlobalMarketOpsMarketStates(context.Context, []string, MarketOpsMarketStateFilter) ([]MarketOpsMarketStateRecord, error)
}

// SubscriberGlobalEROCRepository reads parity-approved platform EROC results.
// Symbol authorization remains the responsibility of the selected watchlist.
type SubscriberGlobalEROCRepository interface {
	ListSubscriberGlobalEROCResults(context.Context, []string, int) ([]MarketOpsValuationResultRecord, error)
}

// SubscriberGlobalValuationRepository reads parity-approved platform VC/DOSM
// valuation results. Symbol authorization remains the selected watchlist's
// responsibility; tactical valuation is intentionally a later reader slice.
type SubscriberGlobalValuationRepository interface {
	ListSubscriberGlobalValuationResults(context.Context, []string, bool, int) ([]MarketOpsValuationResultRecord, error)
}

type SubscriberGlobalAnnualFinancialTaskRepository interface {
	ListSubscriberGlobalAnnualFinancialTasks(context.Context, int) ([]MarketOpsTaskItemRecord, error)
}

// SubscriberGlobalEEOMRepository reads parity-approved platform EEOM results.
// Symbol authorization remains the selected watchlist's responsibility.
type SubscriberGlobalEEOMRepository interface {
	ListSubscriberGlobalEEOMResults(context.Context, []string, MarketOpsEEOMFilter) ([]MarketOpsEEOMResultRecord, error)
}

// SubscriberGlobalSignalAssuranceEffectivenessRepository reads only
// parity-approved, platform-owned historical outcome observations. It does
// not manufacture SAF assertions: an empty SAF cohort is a meaningful result.
// The Gateway supplies symbols authorized by the selected watchlist.
type SubscriberGlobalSignalAssuranceEffectivenessRepository interface {
	ListSubscriberGlobalSignalAssuranceEffectiveness(context.Context, []string, SignalAssuranceEffectivenessFilter) ([]SignalAssuranceEffectivenessRecord, error)
	ListSubscriberGlobalSignalAssuranceEffectivenessObservations(context.Context, []string, SignalAssuranceEffectivenessFilter) ([]SignalAssuranceEffectivenessObservationRecord, error)
	ListSubscriberGlobalSignalAssuranceRecommendations(context.Context, []string, SignalAssuranceEffectivenessFilter) ([]SignalAssuranceRecommendationRecord, error)
}

// SubscriberGlobalSRIRepository reads the platform-owned sector intelligence
// foundation. SRI is market-wide context, while the selected watchlist remains
// the caller's authorized UI context; no tenant-local SRI result is a fallback.
type SubscriberGlobalSRIRepository interface {
	ListSubscriberGlobalSRISegments(context.Context, bool, int) ([]MarketOpsSRISegmentRecord, error)
	ListSubscriberGlobalSRIETFRegistry(context.Context, string) ([]MarketOpsSRIETFRecord, error)
	ListSubscriberGlobalSRISnapshots(context.Context, MarketOpsSRISnapshotFilter) ([]MarketOpsSRISnapshotRecord, error)
	GetLatestSubscriberGlobalSRIETFHoldingsSnapshot(context.Context, string) (MarketOpsSRIETFHoldingsSnapshotRecord, bool, error)
	ListSubscriberGlobalSRIETFHoldings(context.Context, string, int) ([]MarketOpsSRIETFHoldingRecord, error)
}

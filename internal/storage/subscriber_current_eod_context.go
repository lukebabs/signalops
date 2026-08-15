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

type SubscriberGlobalRiskRewardSnapshotRepository interface {
	ListSubscriberGlobalRiskRewardSnapshots(context.Context, []string, time.Time, int) ([]MarketOpsRiskRewardSnapshotRecord, error)
}

type SubscriberGlobalOptionsDistributionRepository interface {
	ListSubscriberGlobalOptionsDistributions(context.Context, []string, time.Time, int) ([]MarketOpsOptionsDistributionRecord, error)
}

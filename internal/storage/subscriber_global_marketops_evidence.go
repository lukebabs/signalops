package storage

import (
	"context"
	"time"
)

// SubscriberGlobalMarketOpsEvidenceRun records a platform-owned, append-only
// reconciliation run. The foundation deliberately permits shadow/read-only
// runs only; no tenant-local result becomes globally canonical by writing here.
type SubscriberGlobalMarketOpsEvidenceRun struct {
	EvidenceRunID            string
	EvidenceKind             string
	AlgorithmID              string
	AlgorithmVersion         string
	SourceScope              string
	SessionStartDate         time.Time
	SessionEndDate           time.Time
	InputManifestFingerprint string
	ValidationContractRef    string
	ImmutableBaselineRef     string
	ProvenanceJSON           []byte
	RecordedBy               string
	CorrelationID            string
	RecordedAt               time.Time
}

// SubscriberGlobalMarketOpsEvidenceRecord is an immutable result for one
// globally governed asset and market session. payload and provenance are
// retained together so later projections can explain the result without
// consulting a tenant-local ledger.
type SubscriberGlobalMarketOpsEvidenceRecord struct {
	GlobalEvidenceID      string
	EvidenceRunID         string
	GlobalAssetID         string
	SessionDate           time.Time
	EvidenceKind          string
	AlgorithmID           string
	AlgorithmVersion      string
	QualityState          string
	SourceSystem          string
	SourceEventID         string
	SourceRunID           string
	EvidenceFingerprint   string
	ValidationContractRef string
	ImmutableBaselineRef  string
	PayloadJSON           []byte
	ProvenanceJSON        []byte
	ObservedAt            time.Time
}

// SubscriberGlobalMarketOpsEvidenceRepository is intentionally writer-only.
// Type-specific, tenant-authorized reader projections are added only after
// their respective parity and UX gates are approved.
type SubscriberGlobalMarketOpsEvidenceRepository interface {
	RecordSubscriberGlobalMarketOpsEvidenceRun(context.Context, SubscriberGlobalMarketOpsEvidenceRun) (SubscriberGlobalMarketOpsEvidenceRun, error)
	AppendSubscriberGlobalMarketOpsEvidence(context.Context, SubscriberGlobalMarketOpsEvidenceRecord) (SubscriberGlobalMarketOpsEvidenceRecord, error)
}

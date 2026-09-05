package storage

import (
	"context"
	"time"
)

const SubscriberGlobalEligibilityPolicyVersion = "s2-us-common-stock-v1"

type SubscriberGlobalAssetEligibilityDecision struct {
	DecisionID          string
	GlobalAssetID       string
	Decision            string
	PolicyVersion       string
	ReasonCode          string
	ProviderReferenceAt *time.Time
	EvidenceJSON        []byte
	ProvenanceJSON      []byte
	DecidedBy           string
	DecidedAt           time.Time
}

type SubscriberGlobalReferenceCandidate struct {
	GlobalAssetID   string
	SourceID        string
	ProviderSymbol  string
	CanonicalSymbol string
}

type SubscriberGlobalCatalogEligibilityRepository interface {
	ListSubscriberGlobalReferenceCandidates(context.Context, int) ([]SubscriberGlobalReferenceCandidate, error)
	RecordSubscriberGlobalAssetEligibilityDecision(context.Context, SubscriberGlobalAssetEligibilityDecision) (SubscriberGlobalAssetEligibilityDecision, error)
}

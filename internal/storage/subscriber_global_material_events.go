package storage

import (
	"context"
	"time"
)

// SubscriberGlobalMaterialEventRecord is a canonical, point-in-time-known
// MarketOps event. Payload carries the provider-normalized event contract and
// its retained provenance is exposed only through the restricted projection.
type SubscriberGlobalMaterialEventRecord struct {
	GlobalAssetID string
	Symbol        string
	EventID       string
	SessionDate   time.Time
	ObservedAt    time.Time
	QualityState  string
	PayloadJSON   []byte
}

type SubscriberGlobalMaterialEventRepository interface {
	ListSubscriberGlobalMaterialEvents(context.Context, []string, time.Time, int) ([]SubscriberGlobalMaterialEventRecord, error)
}

// SubscriberGlobalAssetResolver maps a symbol through canonical identity
// before a central provider capture is appended.
type SubscriberGlobalAssetResolver interface {
	ResolveSubscriberGlobalCanonicalAssetID(context.Context, string) (string, error)
}

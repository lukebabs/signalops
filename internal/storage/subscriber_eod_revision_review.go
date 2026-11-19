package storage

import (
	"context"
	"time"
)

// SubscriberEODRevisionDeltaRecord is immutable analyst review evidence. It
// compares the original capture and later provider re-observation; it does not
// select either value for historical assurance or live market context.
type SubscriberEODRevisionDeltaRecord struct {
	Symbol                    string
	SessionDate               time.Time
	FieldName                 string
	InitialValue              *float64
	RevisedValue              *float64
	DeltaClass                string
	Materiality               string
	InitialObservedAt         time.Time
	RevisedObservedAt         time.Time
	InitialSourceEventID      string
	RevisedSourceEventID      string
	InitialSourceRunID        string
	RevisedSourceRunID        string
	InitialPayloadFingerprint string
	RevisedPayloadFingerprint string
	InitialAlgorithmVersion   string
	RevisedAlgorithmVersion   string
}

// SubscriberEODRevisionReviewRepository returns immutable delta rows only for
// a symbol that is active in the request tenant's MarketOps universe.
type SubscriberEODRevisionReviewRepository interface {
	ListSubscriberEODRevisionDeltas(context.Context, string, string, int) ([]SubscriberEODRevisionDeltaRecord, error)
}

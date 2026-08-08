package storage

import (
	"context"
	"time"
)

// SignalAssuranceOutboxRepository exposes lifecycle events as a transactional
// outbox. Publishing is at-least-once; consumers deduplicate by event_id.
type SignalAssuranceOutboxRepository interface {
	ListUndeliveredSignalAssertionEvents(context.Context, int) ([]SignalAssertionEventRecord, error)
	MarkSignalAssertionEventPublished(context.Context, string, time.Time) error
}

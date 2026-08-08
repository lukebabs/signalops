package storage

import (
	"context"
	"time"
)

type SignalAssuranceAggregateRepository interface {
	RefreshSignalAssuranceEffectiveness(context.Context, string, string, time.Time, string) error
}

package storage

import (
	"context"
	"time"
)

type RetentionPolicyRecord struct {
	TenantID, PolicyID, AppID, Domain, DataClass string
	RetentionDays                                int
	Mode, PreservationRule, Description          string
	UpdatedAt                                    time.Time
}
type RetentionRunRecord struct {
	RunID, TenantID, PolicyID, Mode, Status string
	CandidateRows, AffectedRows             int64
	OldestCandidateAt, NewestCandidateAt    *time.Time
	DetailJSON                              []byte
	StartedAt                               time.Time
	CompletedAt                             *time.Time
}
type RetentionGovernanceRepository interface {
	ListRetentionPolicies(context.Context, string) ([]RetentionPolicyRecord, error)
	ListRetentionRuns(context.Context, string, int) ([]RetentionRunRecord, error)
}

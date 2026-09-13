package api

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type syncraticWorkerCompletionJobRepo struct {
	completeErr error
	failCode    string
	failMessage string
}

func (r *syncraticWorkerCompletionJobRepo) UpsertSyncraticIntelligenceJob(context.Context, storage.SyncraticIntelligenceJobRecord) error {
	return nil
}

func (r *syncraticWorkerCompletionJobRepo) ListSyncraticIntelligenceJobs(context.Context, storage.SyncraticIntelligenceJobFilter) ([]storage.SyncraticIntelligenceJobRecord, error) {
	return nil, nil
}

func (r *syncraticWorkerCompletionJobRepo) ClaimSyncraticIntelligenceJob(context.Context, time.Time, time.Duration) (storage.SyncraticIntelligenceJobRecord, error) {
	return storage.SyncraticIntelligenceJobRecord{}, storage.ErrNotFound
}

func (r *syncraticWorkerCompletionJobRepo) CompleteSyncraticIntelligenceJob(context.Context, string, string, string, time.Time) error {
	return r.completeErr
}

func (r *syncraticWorkerCompletionJobRepo) FailSyncraticIntelligenceJob(_ context.Context, _ string, errorCode, errorMessage string, _ time.Time) error {
	r.failCode = errorCode
	r.failMessage = errorMessage
	return nil
}

func TestCompleteSyncraticIntelligenceJobRecordsCompletionFailure(t *testing.T) {
	repo := &syncraticWorkerCompletionJobRepo{completeErr: errors.New("foreign key violation")}

	completeSyncraticIntelligenceJob(context.Background(), repo, "synjob-test", "", "")

	if repo.failCode != "job_completion_failed" {
		t.Fatalf("expected job_completion_failed, got %q", repo.failCode)
	}
	if repo.failMessage == "" {
		t.Fatal("expected completion failure to be retained for operator visibility")
	}
}

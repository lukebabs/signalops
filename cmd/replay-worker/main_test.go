package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
	"github.com/lukebabs/signalops/pkg/broker"
)

type fakeReplaySourceRepository struct {
	job      storage.ReplayJobRecord
	raw      []storage.RawEventLedgerRecord
	getCalls int
	cancelAt int
	batches  []struct{ limit, offset int }
}

func (r *fakeReplaySourceRepository) GetReplayJob(context.Context, string) (storage.ReplayJobRecord, error) {
	r.getCalls++
	job := r.job
	if r.cancelAt > 0 && r.getCalls >= r.cancelAt {
		job.Status = storage.ReplayJobStatusCanceled
	}
	return job, nil
}

func (r *fakeReplaySourceRepository) ListReplayRawEvents(_ context.Context, _ storage.ReplayJobRecord, limit int, offset int) ([]storage.RawEventLedgerRecord, error) {
	r.batches = append(r.batches, struct{ limit, offset int }{limit: limit, offset: offset})
	if offset >= len(r.raw) {
		return nil, nil
	}
	end := offset + limit
	if end > len(r.raw) {
		end = len(r.raw)
	}
	return r.raw[offset:end], nil
}

func (r *fakeReplaySourceRepository) ListReplayNormalizedEvents(context.Context, storage.ReplayJobRecord, int, int) ([]storage.NormalizedEventLedgerRecord, error) {
	return nil, nil
}

func (r *fakeReplaySourceRepository) ListReplaySignals(context.Context, storage.ReplayJobRecord, int, int) ([]storage.SignalLedgerRecord, error) {
	return nil, nil
}

type fakeReplayPublisher struct {
	failuresBeforeSuccess int
	calls                 int
	messages              []broker.Message
}

func (p *fakeReplayPublisher) Publish(_ context.Context, msg broker.Message) (broker.PublishResult, error) {
	p.calls++
	p.messages = append(p.messages, msg)
	if p.calls <= p.failuresBeforeSuccess {
		return broker.PublishResult{}, errors.New("publish failed")
	}
	return broker.PublishResult{Topic: msg.Topic, Partition: int32(p.calls), Offset: int64(100 + p.calls)}, nil
}

func (p *fakeReplayPublisher) Close(context.Context) error { return nil }

func TestExecuteReplayJobProcessesPagedBatches(t *testing.T) {
	job := replayTestJob()
	repo := &fakeReplaySourceRepository{job: job, raw: []storage.RawEventLedgerRecord{
		replayRawEvent("event-1", "key-1"),
		replayRawEvent("event-2", "key-2"),
		replayRawEvent("event-3", "key-3"),
	}}
	publisher := &fakeReplayPublisher{}

	result, err := executeReplayJob(context.Background(), repo, publisher, replayTestTopics(), job, workerConfig{MaxRecords: 3, BatchSize: 2, PublishMaxAttempts: 1})
	if err != nil {
		t.Fatalf("executeReplayJob error = %v", err)
	}
	if result.Scanned != 3 || result.Published != 3 || result.Failed != 0 || result.Batches != 2 {
		t.Fatalf("result = %+v", result)
	}
	if len(result.Records) != 3 || result.Records[2].SourceID != "event-3" || result.Records[2].Attempts != 1 {
		t.Fatalf("record results = %+v", result.Records)
	}
	if len(repo.batches) != 2 || repo.batches[0].limit != 2 || repo.batches[0].offset != 0 || repo.batches[1].limit != 1 || repo.batches[1].offset != 2 {
		t.Fatalf("batches = %+v", repo.batches)
	}
}

func TestExecuteReplayJobRetriesPublishFailures(t *testing.T) {
	job := replayTestJob()
	repo := &fakeReplaySourceRepository{job: job, raw: []storage.RawEventLedgerRecord{replayRawEvent("event-1", "key-1")}}
	publisher := &fakeReplayPublisher{failuresBeforeSuccess: 1}

	result, err := executeReplayJob(context.Background(), repo, publisher, replayTestTopics(), job, workerConfig{MaxRecords: 1, BatchSize: 1, PublishMaxAttempts: 2})
	if err != nil {
		t.Fatalf("executeReplayJob error = %v", err)
	}
	if publisher.calls != 2 || result.Published != 1 || len(result.Records) != 1 || result.Records[0].Attempts != 2 {
		t.Fatalf("publisher calls = %d result = %+v", publisher.calls, result)
	}
}

func TestExecuteReplayJobStopsWhenJobCanceled(t *testing.T) {
	job := replayTestJob()
	repo := &fakeReplaySourceRepository{job: job, cancelAt: 2, raw: []storage.RawEventLedgerRecord{
		replayRawEvent("event-1", "key-1"),
		replayRawEvent("event-2", "key-2"),
	}}
	publisher := &fakeReplayPublisher{}

	result, err := executeReplayJob(context.Background(), repo, publisher, replayTestTopics(), job, workerConfig{MaxRecords: 2, BatchSize: 1, PublishMaxAttempts: 1})
	if err == nil {
		t.Fatalf("expected cancellation error")
	}
	if !result.Canceled || result.Scanned != 1 || result.Published != 1 || publisher.calls != 1 {
		t.Fatalf("result = %+v calls = %d", result, publisher.calls)
	}
}

func replayTestJob() storage.ReplayJobRecord {
	return storage.ReplayJobRecord{
		ReplayJobID: "replay-test",
		TenantID:    "tenant-local",
		SourceKind:  storage.ReplaySourceRaw,
		ReplayMode:  storage.ReplayModeOriginal,
		Status:      storage.ReplayJobStatusRunning,
		WindowStart: time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC),
		WindowEnd:   time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
	}
}

func replayRawEvent(eventID string, key string) storage.RawEventLedgerRecord {
	return storage.RawEventLedgerRecord{
		EventID:         eventID,
		TenantID:        "tenant-local",
		SourceID:        "src-test",
		Dataset:         "events",
		IdempotencyKey:  key,
		ObservationTime: time.Date(2026, 7, 9, 1, 0, 0, 0, time.UTC),
		PayloadJSON:     []byte(`{"value":1}`),
	}
}

func replayTestTopics() map[string]string {
	return map[string]string{storage.ReplaySourceRaw: "signalops.dev.raw"}
}

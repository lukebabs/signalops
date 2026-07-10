package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	kafkabroker "github.com/lukebabs/signalops/internal/broker/kafka"
	"github.com/lukebabs/signalops/internal/config"
	"github.com/lukebabs/signalops/internal/storage"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
	"github.com/lukebabs/signalops/pkg/broker"
)

const defaultPollInterval = 5 * time.Second

type replayRepository interface {
	storage.ReplayJobRepository
	storage.ReplayWorkerHeartbeatRepository
	replaySourceRepository
}

type replaySourceRepository interface {
	GetReplayJob(ctx context.Context, replayJobID string) (storage.ReplayJobRecord, error)
	ListReplayRawEvents(ctx context.Context, job storage.ReplayJobRecord, limit int, offset int) ([]storage.RawEventLedgerRecord, error)
	ListReplayNormalizedEvents(ctx context.Context, job storage.ReplayJobRecord, limit int, offset int) ([]storage.NormalizedEventLedgerRecord, error)
	ListReplaySignals(ctx context.Context, job storage.ReplayJobRecord, limit int, offset int) ([]storage.SignalLedgerRecord, error)
}

type workerConfig struct {
	WorkerID           string
	OneShot            bool
	MaxRecords         int
	BatchSize          int
	PublishMaxAttempts int
	PollInterval       time.Duration
}

type replayResult struct {
	ReplayJobID string               `json:"replay_job_id"`
	SourceKind  string               `json:"source_kind"`
	Scanned     int                  `json:"scanned"`
	Published   int                  `json:"published"`
	Failed      int                  `json:"failed"`
	Batches     int                  `json:"batches"`
	MaxRecords  int                  `json:"max_records"`
	BatchSize   int                  `json:"batch_size"`
	Canceled    bool                 `json:"canceled"`
	StartedAt   string               `json:"started_at"`
	CompletedAt string               `json:"completed_at,omitempty"`
	Records     []replayRecordResult `json:"records,omitempty"`
}

type replayRecordResult struct {
	SourceID  string `json:"source_id"`
	Key       string `json:"key"`
	Status    string `json:"status"`
	Topic     string `json:"topic,omitempty"`
	Partition *int32 `json:"partition,omitempty"`
	Offset    *int64 `json:"offset,omitempty"`
	Attempts  int    `json:"attempts"`
	Error     string `json:"error,omitempty"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("signalops replay worker failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg := config.Load()
	if strings.TrimSpace(cfg.DatabaseURL) == "" || strings.TrimSpace(cfg.TemporalDatabaseURL) == "" {
		return errors.New("SIGNALOPS_DATABASE_URL and SIGNALOPS_TEMPORAL_DATABASE_URL are required")
	}
	workerCfg := loadWorkerConfig()
	processStartedAt := time.Now().UTC()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	repository, err := postgresstorage.OpenWithTemporal(ctx, cfg.DatabaseURL, cfg.TemporalDatabaseURL)
	if err != nil {
		return err
	}
	defer repository.Close()
	client, err := kafkabroker.NewClient(kafkabroker.Config{Brokers: strings.Split(cfg.BrokerBrokers, ","), ClientID: "signalops-replay-worker"})
	if err != nil {
		return err
	}
	defer closePublisher(logger, client)
	topics := map[string]string{
		storage.ReplaySourceRaw:        broker.TopicName(cfg.Environment, broker.RawTopic),
		storage.ReplaySourceNormalized: broker.TopicName(cfg.Environment, broker.NormalizedTopic),
		storage.ReplaySourceSignals:    broker.TopicName(cfg.Environment, broker.SignalTopic),
	}
	logger.Info("signalops replay worker started", "worker_id", workerCfg.WorkerID, "one_shot", workerCfg.OneShot, "max_records", workerCfg.MaxRecords, "batch_size", workerCfg.BatchSize, "publish_max_attempts", workerCfg.PublishMaxAttempts)
	reportHeartbeat(ctx, logger, repository, heartbeatRecord(workerCfg, processStartedAt, "idle"))
	defer reportHeartbeat(context.Background(), logger, repository, heartbeatRecord(workerCfg, processStartedAt, "stopping"))
	for {
		job, err := repository.ClaimNextReplayJob(ctx, workerCfg.WorkerID, time.Now().UTC())
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				reportHeartbeat(ctx, logger, repository, heartbeatRecord(workerCfg, processStartedAt, "idle"))
				if workerCfg.OneShot {
					logger.Info("no queued replay job found")
					return nil
				}
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(workerCfg.PollInterval):
					continue
				}
			}
			errAt := time.Now().UTC()
			reportHeartbeat(ctx, logger, repository, heartbeatRecord(workerCfg, processStartedAt, "error", heartbeatError(errAt, err)))
			return err
		}
		claimedAt := time.Now().UTC()
		reportHeartbeat(ctx, logger, repository, heartbeatRecord(workerCfg, processStartedAt, "running", heartbeatClaim(claimedAt, job.ReplayJobID)))
		logger.Info("claimed replay job", "replay_job_id", job.ReplayJobID, "source_kind", job.SourceKind, "window_start", job.WindowStart, "window_end", job.WindowEnd)
		result, runErr := executeReplayJob(ctx, repository, client, topics, job, workerCfg)
		resultJSON, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			return marshalErr
		}
		if runErr != nil {
			if result.Canceled {
				completedAt := time.Now().UTC()
				if _, err := repository.CancelReplayJob(ctx, job.ReplayJobID, workerCfg.WorkerID, completedAt, "canceled during replay", resultJSON); err != nil {
					errAt := time.Now().UTC()
					reportHeartbeat(ctx, logger, repository, heartbeatRecord(workerCfg, processStartedAt, "error", heartbeatError(errAt, err)))
					return err
				}
				reportHeartbeat(ctx, logger, repository, heartbeatRecord(workerCfg, processStartedAt, "idle", heartbeatCompleted(completedAt, job.ReplayJobID)))
				logger.Info("replay job canceled", "replay_job_id", job.ReplayJobID, "scanned", result.Scanned, "published", result.Published)
				if workerCfg.OneShot {
					return nil
				}
				continue
			}
			failedAt := time.Now().UTC()
			if _, err := repository.FailReplayJob(ctx, job.ReplayJobID, failedAt, runErr.Error(), resultJSON); err != nil {
				errAt := time.Now().UTC()
				reportHeartbeat(ctx, logger, repository, heartbeatRecord(workerCfg, processStartedAt, "error", heartbeatError(errAt, err)))
				return err
			}
			reportHeartbeat(ctx, logger, repository, heartbeatRecord(workerCfg, processStartedAt, "error", heartbeatError(failedAt, runErr), heartbeatCompleted(failedAt, job.ReplayJobID)))
			logger.Error("replay job failed", "replay_job_id", job.ReplayJobID, "error", runErr)
			if workerCfg.OneShot {
				return runErr
			}
			continue
		}
		completedAt := time.Now().UTC()
		if _, err := repository.CompleteReplayJob(ctx, job.ReplayJobID, completedAt, resultJSON); err != nil {
			errAt := time.Now().UTC()
			reportHeartbeat(ctx, logger, repository, heartbeatRecord(workerCfg, processStartedAt, "error", heartbeatError(errAt, err)))
			return err
		}
		reportHeartbeat(ctx, logger, repository, heartbeatRecord(workerCfg, processStartedAt, "idle", heartbeatCompleted(completedAt, job.ReplayJobID)))
		logger.Info("replay job completed", "replay_job_id", job.ReplayJobID, "published", result.Published, "scanned", result.Scanned, "failed", result.Failed, "batches", result.Batches)
		if workerCfg.OneShot {
			return nil
		}
	}
}

func executeReplayJob(ctx context.Context, repo replaySourceRepository, publisher broker.Publisher, topics map[string]string, job storage.ReplayJobRecord, cfg workerConfig) (replayResult, error) {
	startedAt := time.Now().UTC()
	result := replayResult{ReplayJobID: job.ReplayJobID, SourceKind: job.SourceKind, MaxRecords: cfg.MaxRecords, BatchSize: cfg.BatchSize, StartedAt: startedAt.Format(time.RFC3339Nano)}
	topic := topics[job.SourceKind]
	if strings.TrimSpace(topic) == "" {
		return result, fmt.Errorf("unsupported replay source kind %q", job.SourceKind)
	}
	maxRecords := cfg.MaxRecords
	if maxRecords <= 0 {
		maxRecords = 50
	}
	batchSize := cfg.BatchSize
	if batchSize <= 0 || batchSize > maxRecords {
		batchSize = maxRecords
	}
	remaining := maxRecords
	offset := 0
	for remaining > 0 {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		latest, err := repo.GetReplayJob(ctx, job.ReplayJobID)
		if err != nil {
			return result, err
		}
		if latest.Status == storage.ReplayJobStatusCanceled {
			result.Canceled = true
			result.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
			return result, fmt.Errorf("replay job canceled")
		}
		limit := batchSize
		if remaining < limit {
			limit = remaining
		}
		batch, err := loadReplayBatch(ctx, repo, job, limit, offset)
		if err != nil {
			return result, err
		}
		if len(batch) == 0 {
			break
		}
		result.Batches++
		result.Scanned += len(batch)
		for _, item := range batch {
			recordResult := replayRecordResult{SourceID: item.sourceID, Key: item.key}
			value, err := replayPayload(item.value, job)
			if err != nil {
				recordResult.Status, recordResult.Error = "failed", err.Error()
				recordResult.Attempts = 0
				result.Failed++
				result.Records = appendRecordResult(result.Records, recordResult)
				return result, err
			}
			publishResult, attempts, err := publishReplayWithRetry(ctx, publisher, topic, item.key, value, job, item.sourceID, cfg.PublishMaxAttempts)
			recordResult.Attempts = attempts
			if err != nil {
				recordResult.Status, recordResult.Error = "failed", err.Error()
				result.Failed++
				result.Records = appendRecordResult(result.Records, recordResult)
				return result, err
			}
			recordResult.Status = "published"
			recordResult.Topic = publishResult.Topic
			partition, offsetValue := publishResult.Partition, publishResult.Offset
			recordResult.Partition, recordResult.Offset = &partition, &offsetValue
			result.Published++
			result.Records = appendRecordResult(result.Records, recordResult)
		}
		remaining -= len(batch)
		offset += len(batch)
		if len(batch) < limit {
			break
		}
	}
	result.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
	return result, nil
}

type replayBatchItem struct {
	sourceID string
	key      string
	value    []byte
}

func loadReplayBatch(ctx context.Context, repo replaySourceRepository, job storage.ReplayJobRecord, limit int, offset int) ([]replayBatchItem, error) {
	switch job.SourceKind {
	case storage.ReplaySourceRaw:
		records, err := repo.ListReplayRawEvents(ctx, job, limit, offset)
		if err != nil {
			return nil, err
		}
		items := make([]replayBatchItem, 0, len(records))
		for _, record := range records {
			items = append(items, replayBatchItem{sourceID: record.EventID, key: record.IdempotencyKey, value: record.PayloadJSON})
		}
		return items, nil
	case storage.ReplaySourceNormalized:
		records, err := repo.ListReplayNormalizedEvents(ctx, job, limit, offset)
		if err != nil {
			return nil, err
		}
		items := make([]replayBatchItem, 0, len(records))
		for _, record := range records {
			items = append(items, replayBatchItem{sourceID: record.EventID, key: record.IdempotencyKey, value: record.EventJSON})
		}
		return items, nil
	case storage.ReplaySourceSignals:
		records, err := repo.ListReplaySignals(ctx, job, limit, offset)
		if err != nil {
			return nil, err
		}
		items := make([]replayBatchItem, 0, len(records))
		for _, record := range records {
			items = append(items, replayBatchItem{sourceID: record.SignalID, key: record.SignalID, value: record.EventJSON})
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported replay source kind %q", job.SourceKind)
	}
}

func appendRecordResult(records []replayRecordResult, record replayRecordResult) []replayRecordResult {
	const maxRecordResults = 100
	if len(records) >= maxRecordResults {
		return records
	}
	return append(records, record)
}

func replayPayload(original []byte, job storage.ReplayJobRecord) ([]byte, error) {
	var payload map[string]any
	if err := json.Unmarshal(original, &payload); err != nil {
		return nil, fmt.Errorf("decode replay payload: %w", err)
	}
	payload["replay_job_id"] = job.ReplayJobID
	payload["ingestion_mode"] = "replay"
	metadata, _ := payload["metadata"].(map[string]any)
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["replay"] = map[string]any{"replay_job_id": job.ReplayJobID, "replay_mode": job.ReplayMode, "replayed_at": time.Now().UTC().Format(time.RFC3339Nano)}
	payload["metadata"] = metadata
	return json.Marshal(payload)
}

func publishReplayWithRetry(ctx context.Context, publisher broker.Publisher, topic string, key string, value []byte, job storage.ReplayJobRecord, sourceID string, maxAttempts int) (broker.PublishResult, int, error) {
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result, err := publishReplay(ctx, publisher, topic, key, value, job, sourceID)
		if err == nil {
			return result, attempt, nil
		}
		lastErr = err
		if attempt < maxAttempts {
			select {
			case <-ctx.Done():
				return broker.PublishResult{}, attempt, ctx.Err()
			case <-time.After(time.Duration(attempt) * 250 * time.Millisecond):
			}
		}
	}
	return broker.PublishResult{}, maxAttempts, lastErr
}

func publishReplay(ctx context.Context, publisher broker.Publisher, topic string, key string, value []byte, job storage.ReplayJobRecord, sourceID string) (broker.PublishResult, error) {
	return publisher.Publish(ctx, broker.Message{Topic: topic, Key: key, Value: value, Headers: map[string]string{"content_type": "application/json", "signalops_replay_job_id": job.ReplayJobID, "signalops_replay_source_kind": job.SourceKind, "signalops_replay_source_id": sourceID}, CorrelationID: "replay:" + job.ReplayJobID, CausationID: sourceID, TraceID: "replay:" + job.ReplayJobID, PublishedAt: time.Now().UTC()})
}

func loadWorkerConfig() workerConfig {
	maxRecords := envInt("SIGNALOPS_REPLAY_MAX_RECORDS", 50)
	batchSize := envInt("SIGNALOPS_REPLAY_BATCH_SIZE", 50)
	if batchSize > 200 {
		batchSize = 200
	}
	return workerConfig{WorkerID: envOrDefault("SIGNALOPS_REPLAY_WORKER_ID", "signalops-replay-worker"), OneShot: envBool("SIGNALOPS_REPLAY_ONESHOT", false), MaxRecords: maxRecords, BatchSize: batchSize, PublishMaxAttempts: envInt("SIGNALOPS_REPLAY_PUBLISH_MAX_ATTEMPTS", 3), PollInterval: envDuration("SIGNALOPS_REPLAY_POLL_INTERVAL", defaultPollInterval)}
}
func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
func envBool(key string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes" || value == "on"
}
func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

type heartbeatOption func(*storage.ReplayWorkerHeartbeatRecord)

func heartbeatClaim(claimedAt time.Time, replayJobID string) heartbeatOption {
	return func(record *storage.ReplayWorkerHeartbeatRecord) {
		record.LastClaimedAt = &claimedAt
		record.LastClaimedReplayJobID = replayJobID
	}
}

func heartbeatCompleted(completedAt time.Time, replayJobID string) heartbeatOption {
	return func(record *storage.ReplayWorkerHeartbeatRecord) {
		record.LastCompletedAt = &completedAt
		record.LastCompletedReplayJobID = replayJobID
	}
}

func heartbeatError(errorAt time.Time, err error) heartbeatOption {
	return func(record *storage.ReplayWorkerHeartbeatRecord) {
		record.LastErrorAt = &errorAt
		if err != nil {
			record.LastErrorMessage = err.Error()
		}
	}
}

func heartbeatRecord(cfg workerConfig, processStartedAt time.Time, status string, options ...heartbeatOption) storage.ReplayWorkerHeartbeatRecord {
	metadata, _ := json.Marshal(map[string]any{
		"one_shot":             cfg.OneShot,
		"max_records":          cfg.MaxRecords,
		"batch_size":           cfg.BatchSize,
		"publish_max_attempts": cfg.PublishMaxAttempts,
		"poll_interval":        cfg.PollInterval.String(),
	})
	record := storage.ReplayWorkerHeartbeatRecord{
		WorkerID:         cfg.WorkerID,
		Status:           status,
		ProcessStartedAt: processStartedAt,
		LastSeenAt:       time.Now().UTC(),
		MetadataJSON:     metadata,
	}
	for _, option := range options {
		option(&record)
	}
	return record
}

func reportHeartbeat(ctx context.Context, logger *slog.Logger, repo storage.ReplayWorkerHeartbeatRepository, record storage.ReplayWorkerHeartbeatRecord) {
	if err := repo.UpsertReplayWorkerHeartbeat(ctx, record); err != nil {
		logger.Error("upsert replay worker heartbeat", "error", err, "worker_id", record.WorkerID, "status", record.Status)
	}
}

func closePublisher(logger *slog.Logger, publisher broker.Publisher) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := publisher.Close(ctx); err != nil {
		logger.Error("close replay worker broker", "error", err)
	}
}

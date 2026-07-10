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
	storage.QueryRepository
}

type workerConfig struct {
	WorkerID     string
	OneShot      bool
	MaxRecords   int
	PollInterval time.Duration
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
	logger.Info("signalops replay worker started", "worker_id", workerCfg.WorkerID, "one_shot", workerCfg.OneShot, "max_records", workerCfg.MaxRecords)
	for {
		job, err := repository.ClaimNextReplayJob(ctx, workerCfg.WorkerID, time.Now().UTC())
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
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
			return err
		}
		logger.Info("claimed replay job", "replay_job_id", job.ReplayJobID, "source_kind", job.SourceKind, "window_start", job.WindowStart, "window_end", job.WindowEnd)
		result, runErr := executeReplayJob(ctx, repository, client, topics, job, workerCfg.MaxRecords)
		resultJSON, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			return marshalErr
		}
		if runErr != nil {
			if _, err := repository.FailReplayJob(ctx, job.ReplayJobID, time.Now().UTC(), runErr.Error(), resultJSON); err != nil {
				return err
			}
			logger.Error("replay job failed", "replay_job_id", job.ReplayJobID, "error", runErr)
			if workerCfg.OneShot {
				return runErr
			}
			continue
		}
		if _, err := repository.CompleteReplayJob(ctx, job.ReplayJobID, time.Now().UTC(), resultJSON); err != nil {
			return err
		}
		logger.Info("replay job completed", "replay_job_id", job.ReplayJobID, "published", result["published"])
		if workerCfg.OneShot {
			return nil
		}
	}
}

func executeReplayJob(ctx context.Context, repo replayRepository, publisher broker.Publisher, topics map[string]string, job storage.ReplayJobRecord, limit int) (map[string]any, error) {
	result := map[string]any{"replay_job_id": job.ReplayJobID, "source_kind": job.SourceKind, "published": 0, "scanned": 0, "max_records": limit}
	topic := topics[job.SourceKind]
	if strings.TrimSpace(topic) == "" {
		return result, fmt.Errorf("unsupported replay source kind %q", job.SourceKind)
	}
	switch job.SourceKind {
	case storage.ReplaySourceRaw:
		records, err := repo.ListReplayRawEvents(ctx, job, limit)
		if err != nil {
			return result, err
		}
		result["scanned"] = len(records)
		for _, record := range records {
			value, err := replayPayload(record.PayloadJSON, job)
			if err != nil {
				return result, err
			}
			if _, err := publishReplay(ctx, publisher, topic, record.IdempotencyKey, value, job, record.EventID); err != nil {
				return result, err
			}
			result["published"] = result["published"].(int) + 1
		}
	case storage.ReplaySourceNormalized:
		records, err := repo.ListReplayNormalizedEvents(ctx, job, limit)
		if err != nil {
			return result, err
		}
		result["scanned"] = len(records)
		for _, record := range records {
			value, err := replayPayload(record.EventJSON, job)
			if err != nil {
				return result, err
			}
			if _, err := publishReplay(ctx, publisher, topic, record.IdempotencyKey, value, job, record.EventID); err != nil {
				return result, err
			}
			result["published"] = result["published"].(int) + 1
		}
	case storage.ReplaySourceSignals:
		records, err := repo.ListReplaySignals(ctx, job, limit)
		if err != nil {
			return result, err
		}
		result["scanned"] = len(records)
		for _, record := range records {
			value, err := replayPayload(record.EventJSON, job)
			if err != nil {
				return result, err
			}
			if _, err := publishReplay(ctx, publisher, topic, record.SignalID, value, job, record.SignalID); err != nil {
				return result, err
			}
			result["published"] = result["published"].(int) + 1
		}
	default:
		return result, fmt.Errorf("unsupported replay source kind %q", job.SourceKind)
	}
	result["completed_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	return result, nil
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

func publishReplay(ctx context.Context, publisher broker.Publisher, topic string, key string, value []byte, job storage.ReplayJobRecord, sourceID string) (broker.PublishResult, error) {
	return publisher.Publish(ctx, broker.Message{Topic: topic, Key: key, Value: value, Headers: map[string]string{"content_type": "application/json", "signalops_replay_job_id": job.ReplayJobID, "signalops_replay_source_kind": job.SourceKind, "signalops_replay_source_id": sourceID}, CorrelationID: "replay:" + job.ReplayJobID, CausationID: sourceID, TraceID: "replay:" + job.ReplayJobID, PublishedAt: time.Now().UTC()})
}

func loadWorkerConfig() workerConfig {
	return workerConfig{WorkerID: envOrDefault("SIGNALOPS_REPLAY_WORKER_ID", "signalops-replay-worker"), OneShot: envBool("SIGNALOPS_REPLAY_ONESHOT", false), MaxRecords: envInt("SIGNALOPS_REPLAY_MAX_RECORDS", 50), PollInterval: envDuration("SIGNALOPS_REPLAY_POLL_INTERVAL", defaultPollInterval)}
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

func closePublisher(logger *slog.Logger, publisher broker.Publisher) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := publisher.Close(ctx); err != nil {
		logger.Error("close replay worker broker", "error", err)
	}
}

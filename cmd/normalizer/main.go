package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	kafkabroker "github.com/lukebabs/signalops/internal/broker/kafka"
	"github.com/lukebabs/signalops/internal/config"
	connect "github.com/lukebabs/signalops/internal/cyberops/connect"
	"github.com/lukebabs/signalops/internal/normalization"
	"github.com/lukebabs/signalops/internal/platformregistry"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
	"github.com/lukebabs/signalops/pkg/broker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("signalops normalizer failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg := config.Load()
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		return errors.New("SIGNALOPS_DATABASE_URL is required")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	client, err := kafkabroker.NewClient(kafkabroker.Config{Brokers: strings.Split(cfg.BrokerBrokers, ","), ClientID: "signalops-normalizer"})
	if err != nil {
		return err
	}
	defer closeBroker(logger, client)
	inputTopic := broker.TopicName(cfg.Environment, broker.RawTopic)
	outputTopic := broker.TopicName(cfg.Environment, broker.NormalizedTopic)
	dlqTopic := broker.TopicName(cfg.Environment, broker.DLQAlgorithmTopic)
	consumer, err := client.NewConsumer("signalops.normalizer.v1", []string{inputTopic})
	if err != nil {
		return err
	}
	defer closeConsumer(logger, consumer)
	repository, err := postgresstorage.OpenWithTemporal(ctx, cfg.DatabaseURL, cfg.TemporalDatabaseURL)
	if err != nil {
		return err
	}
	defer repository.Close()
	marketRepository := repository
	if strings.TrimSpace(cfg.MarketOpsDatabaseURL) != "" {
		marketRepository, err = postgresstorage.OpenWithTemporal(ctx, cfg.MarketOpsDatabaseURL, cfg.MarketOpsTemporalDatabaseURL)
		if err != nil {
			return err
		}
		defer marketRepository.Close()
	}
	processor := normalization.Processor{Publisher: client, Repository: repository, OutputTopic: outputTopic}
	marketProcessor := normalization.Processor{Publisher: client, Repository: marketRepository, OutputTopic: outputTopic}
	if envBoolOrDefault("SIGNALOPS_PLATFORM_REGISTRY_ENFORCEMENT", false) {
		processor.DefinitionValidator = platformregistry.MassiveRawEventDefinitionValidator{Lister: repository}
		marketProcessor.DefinitionValidator = platformregistry.MassiveRawEventDefinitionValidator{Lister: marketRepository}
		logger.Info("platform registry enforcement enabled for Massive scheduled-pull events")
	}
	logger.Info("signalops normalizer started", "input_topic", inputTopic, "output_topic", outputTopic)
	for {
		message, err := consumer.Consume(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
		if connect.IsCandidate(message.Value, message.Headers) {
			if err := consumer.Commit(ctx, message); err != nil {
				return err
			}
			logger.Info("connect candidate reserved for cyberops accepted-raw path", "topic", message.Topic, "partition", message.Partition, "offset", message.Offset)
			continue
		}
		activeProcessor := processor
		if marketRepository != repository && marketOpsAppID(message.Value) {
			activeProcessor = marketProcessor
		}
		record, err := activeProcessor.Process(ctx, message)
		if err != nil {
			var invalid normalization.InvalidEventError
			if errors.As(err, &invalid) {
				if err := normalization.PublishInvalidEvent(ctx, client, dlqTopic, message, invalid); err != nil {
					return err
				}
				if err := consumer.Commit(ctx, message); err != nil {
					return err
				}
				logger.Warn("invalid raw event sent to normalization DLQ", "error", invalid, "topic", message.Topic, "partition", message.Partition, "offset", message.Offset)
				continue
			}
			logger.Error("normalization failed; source offset remains uncommitted", "error", err, "topic", message.Topic, "partition", message.Partition, "offset", message.Offset)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(time.Second):
			}
			continue
		}
		if err := consumer.Commit(ctx, message); err != nil {
			return err
		}
		logger.Info("normalized event persisted", "event_id", record.EventID, "raw_partition", record.RawPartition, "raw_offset", record.RawOffset, "normalized_partition", record.NormalizedPartition, "normalized_offset", record.NormalizedOffset)
	}
}

func marketOpsAppID(value []byte) bool {
	var envelope struct {
		AppID string `json:"app_id"`
	}
	return json.Unmarshal(value, &envelope) == nil && strings.EqualFold(strings.TrimSpace(envelope.AppID), "marketops")
}

func closeBroker(logger *slog.Logger, publisher broker.Publisher) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := publisher.Close(ctx); err != nil {
		logger.Error("close normalizer broker", "error", err)
	}
}
func closeConsumer(logger *slog.Logger, consumer broker.Consumer) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := consumer.Close(ctx); err != nil {
		logger.Error("close normalizer consumer", "error", err)
	}
}

func envBoolOrDefault(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

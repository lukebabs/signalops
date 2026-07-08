package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	kafkabroker "github.com/lukebabs/signalops/internal/broker/kafka"
	"github.com/lukebabs/signalops/internal/config"
	"github.com/lukebabs/signalops/internal/signals"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
	"github.com/lukebabs/signalops/pkg/broker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("signalops signal persister failed", "error", err)
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

	client, err := kafkabroker.NewClient(kafkabroker.Config{
		Brokers: strings.Split(cfg.BrokerBrokers, ","), ClientID: "signalops-signal-persister",
	})
	if err != nil {
		return err
	}
	defer closeBroker(logger, client)

	inputTopic := broker.TopicName(cfg.Environment, broker.SignalTopic)
	dlqTopic := broker.TopicName(cfg.Environment, broker.DLQAlgorithmTopic)
	consumer, err := client.NewConsumer("signalops.signal-persister.v1", []string{inputTopic})
	if err != nil {
		return err
	}
	defer closeConsumer(logger, consumer)

	repository, err := postgresstorage.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer repository.Close()
	processor := signals.Processor{Repository: repository}
	logger.Info("signalops signal persister started", "input_topic", inputTopic)

	for {
		message, err := consumer.Consume(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
		record, err := processor.Process(ctx, message)
		if err != nil {
			var invalid signals.InvalidEventError
			if errors.As(err, &invalid) {
				if err := signals.PublishInvalidEvent(ctx, client, dlqTopic, message, invalid); err != nil {
					return err
				}
				if err := consumer.Commit(ctx, message); err != nil {
					return err
				}
				logger.Warn("invalid signal sent to DLQ", "error", invalid,
					"topic", message.Topic, "partition", message.Partition, "offset", message.Offset)
				continue
			}
			logger.Error("signal persistence failed; source offset remains uncommitted",
				"error", err, "topic", message.Topic, "partition", message.Partition, "offset", message.Offset)
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
		logger.Info("signal persisted", "signal_id", record.SignalID, "detector_id", record.DetectorID,
			"partition", record.BrokerPartition, "offset", record.BrokerOffset)
	}
}

func closeBroker(logger *slog.Logger, publisher broker.Publisher) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := publisher.Close(ctx); err != nil {
		logger.Error("close signal persister broker", "error", err)
	}
}

func closeConsumer(logger *slog.Logger, consumer broker.Consumer) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := consumer.Close(ctx); err != nil {
		logger.Error("close signal persister consumer", "error", err)
	}
}

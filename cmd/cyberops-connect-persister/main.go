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
	connect "github.com/lukebabs/signalops/internal/cyberops/connect"
	"github.com/lukebabs/signalops/internal/normalization"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
	"github.com/lukebabs/signalops/pkg/broker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("cyberops connect persister failed", "error", err)
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
	client, err := kafkabroker.NewClient(kafkabroker.Config{Brokers: strings.Split(cfg.BrokerBrokers, ","), ClientID: "signalops-connect-raw-persistence"})
	if err != nil {
		return err
	}
	defer client.Close(context.Background())
	repo, err := postgresstorage.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	rawTopic := broker.TopicName(cfg.Environment, broker.RawTopic)
	acceptedTopic := broker.TopicName(cfg.Environment, broker.ConnectAcceptedRawTopic)
	dlqTopic := broker.TopicName(cfg.Environment, broker.DLQAlgorithmTopic)
	consumer, err := client.NewConsumer("signalops."+cfg.Environment+".connect-raw-persistence.v1", []string{rawTopic})
	if err != nil {
		return err
	}
	defer consumer.Close(context.Background())
	processor := connect.Processor{Repository: repo, AcceptedTopic: acceptedTopic}
	for {
		message, err := consumer.Consume(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
		result, err := processor.Process(ctx, message)
		if err != nil {
			var invalid connect.InvalidEventError
			if errors.As(err, &invalid) {
				if err := normalization.PublishInvalidEvent(ctx, client, dlqTopic, message, invalid); err != nil {
					return err
				}
				if err := consumer.Commit(ctx, message); err != nil {
					return err
				}
				logger.Warn("cyberops connect event sent to DLQ", "error", invalid)
				continue
			}
			logger.Error("cyberops connect persistence failed; offset remains uncommitted", "error", err)
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
		logger.Info("cyberops connect event processed", "ignored", result.Ignored, "duplicate", result.Duplicate, "integrity_failure", result.IntegrityFailure)
	}
}

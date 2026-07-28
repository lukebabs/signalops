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
	"github.com/lukebabs/signalops/internal/normalization"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
	"github.com/lukebabs/signalops/pkg/broker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("cyberops normalizer failed", "error", err)
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
	client, err := kafkabroker.NewClient(kafkabroker.Config{Brokers: strings.Split(cfg.BrokerBrokers, ","), ClientID: "signalops-cyberops-normalizer"})
	if err != nil {
		return err
	}
	defer client.Close(context.Background())
	repo, err := postgresstorage.OpenWithTemporal(ctx, cfg.DatabaseURL, cfg.TemporalDatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	input := broker.TopicName(cfg.Environment, broker.ConnectAcceptedRawTopic)
	output := broker.TopicName(cfg.Environment, broker.NormalizedTopic)
	dlq := broker.TopicName(cfg.Environment, broker.DLQAlgorithmTopic)
	consumer, err := client.NewConsumer("signalops."+cfg.Environment+".cyberops-normalizer.v1", []string{input})
	if err != nil {
		return err
	}
	defer consumer.Close(context.Background())
	processor := normalization.Processor{Publisher: client, Repository: repo, OutputTopic: output}
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
			var invalid normalization.InvalidEventError
			if errors.As(err, &invalid) {
				if err := normalization.PublishInvalidEvent(ctx, client, dlq, message, invalid); err != nil {
					return err
				}
				if err := consumer.Commit(ctx, message); err != nil {
					return err
				}
				continue
			}
			logger.Error("cyberops normalization failed; offset remains uncommitted", "error", err)
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
		logger.Info("cyberops event normalized", "event_id", record.EventID)
	}
}

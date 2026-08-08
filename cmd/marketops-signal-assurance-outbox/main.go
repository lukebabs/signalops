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
	"github.com/lukebabs/signalops/internal/storage"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
	"github.com/lukebabs/signalops/pkg/broker"
)

type repository interface {
	storage.SignalAssuranceOutboxRepository
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("signal assurance outbox failed", "error", err)
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
	client, err := kafkabroker.NewClient(kafkabroker.Config{Brokers: strings.Split(cfg.BrokerBrokers, ","), ClientID: "signalops-signal-assurance-outbox"})
	if err != nil {
		return err
	}
	defer closePublisher(client)
	repo, err := postgresstorage.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	topic := broker.TopicName(cfg.Environment, broker.SignalAssertionTopic)
	for {
		published, err := flush(ctx, repo, client, topic)
		if err != nil {
			return err
		}
		if published == 0 {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(time.Second):
			}
		} else {
			logger.Info("signal assurance outbox published", "events", published)
		}
	}
}
func flush(ctx context.Context, repo repository, publisher broker.Publisher, topic string) (int, error) {
	events, err := repo.ListUndeliveredSignalAssertionEvents(ctx, 100)
	if err != nil {
		return 0, err
	}
	for _, event := range events {
		value, err := json.Marshal(event)
		if err != nil {
			return 0, err
		}
		if _, err = publisher.Publish(ctx, broker.Message{Topic: topic, Key: event.AssertionID, Value: value, CorrelationID: event.EventID, PublishedAt: event.OccurredAt}); err != nil {
			return 0, err
		}
		if err = repo.MarkSignalAssertionEventPublished(ctx, event.EventID, time.Now().UTC()); err != nil {
			return 0, err
		}
	}
	return len(events), nil
}
func closePublisher(publisher broker.Publisher) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = publisher.Close(ctx)
}

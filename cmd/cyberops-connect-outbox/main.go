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

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("cyberops connect outbox failed", "error", err)
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
	client, err := kafkabroker.NewClient(kafkabroker.Config{Brokers: strings.Split(cfg.BrokerBrokers, ","), ClientID: "signalops-connect-outbox"})
	if err != nil {
		return err
	}
	defer client.Close(context.Background())
	repo, err := postgresstorage.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	for {
		if err := flush(ctx, repo, client); err != nil {
			logger.Error("flush outbox", "error", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Second):
		}
	}
}
func flush(ctx context.Context, repo storage.CyberOpsConnectRepository, publisher broker.Publisher) error {
	items, err := repo.ListPendingCyberOpsOutbox(ctx, 50)
	if err != nil {
		return err
	}
	for _, item := range items {
		headers := map[string]string{}
		if err := json.Unmarshal(item.HeadersJSON, &headers); err != nil {
			return err
		}
		_, err = publisher.Publish(ctx, broker.Message{Topic: item.Topic, Key: item.MessageKey, Value: item.MessageValue, Headers: headers, CorrelationID: item.CorrelationID, CausationID: item.CausationID, TraceID: item.TraceID, PublishedAt: time.Now().UTC()})
		if err != nil {
			_ = repo.MarkCyberOpsOutboxAttempt(ctx, item.OutboxID)
			continue
		}
		if err = repo.MarkCyberOpsOutboxPublished(ctx, item.OutboxID, time.Now().UTC()); err != nil {
			return err
		}
	}
	return nil
}

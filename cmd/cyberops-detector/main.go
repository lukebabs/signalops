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
	"github.com/lukebabs/signalops/internal/cyberops/detection"
	"github.com/lukebabs/signalops/internal/storage/postgres"
	"github.com/lukebabs/signalops/pkg/broker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("cyberops detector failed", "error", err)
		os.Exit(1)
	}
}
func run(logger *slog.Logger) error {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	client, err := kafkabroker.NewClient(kafkabroker.Config{Brokers: strings.Split(cfg.BrokerBrokers, ","), ClientID: "signalops-cyberops-detector"})
	if err != nil {
		return err
	}
	defer client.Close(context.Background())
	repo, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	input := broker.TopicName(cfg.Environment, broker.NormalizedTopic)
	signals := broker.TopicName(cfg.Environment, broker.SignalTopic)
	state := broker.TopicName(cfg.Environment, "cyberops-allowed-service-state")
	consumer, err := client.NewConsumer("signalops."+cfg.Environment+".cyberops-detector.v2", []string{input, state})
	if err != nil {
		return err
	}
	defer consumer.Close(context.Background())
	processor := &detection.Processor{Publisher: client, SignalTopic: signals, StateTopic: state, IoTCIDRs: func(ctx context.Context, tenantID string) ([]string, error) {
		item, err := repo.GetCyberOpsIoTNetworkConfig(ctx, tenantID)
		return item.InternalCIDRs, err
	}}
	for {
		message, err := consumer.Consume(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
		if message.Topic == state {
			if err := processor.Restore(message.Key, message.Value); err != nil {
				return err
			}
			if err := consumer.Commit(ctx, message); err != nil {
				return err
			}
			continue
		}
		if err := processor.Process(ctx, message); err != nil {
			logger.Error("detector failed; offset remains uncommitted", "error", err)
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
	}
}

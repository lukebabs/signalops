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
	"github.com/lukebabs/signalops/internal/marketops/signalassurance"
	"github.com/lukebabs/signalops/internal/storage"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
	"github.com/lukebabs/signalops/pkg/broker"
)

type repository interface {
	ResolveSignalValidationContract(context.Context, storage.SignalAssuranceEligibleEvent) (storage.SignalValidationContractRecord, error)
	RegisterSignalAssuranceAssertion(context.Context, storage.SignalAssuranceRegistration) (storage.SignalAssertionRecord, bool, error)
	UpsertSignalValidationContract(context.Context, storage.SignalValidationContractRecord) error
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("signal assurance registrar failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg := config.Load()
	databaseURL := cfg.DatabaseURL
	temporalDatabaseURL := cfg.TemporalDatabaseURL
	if strings.TrimSpace(cfg.MarketOpsDatabaseURL) != "" {
		databaseURL = cfg.MarketOpsDatabaseURL
		temporalDatabaseURL = cfg.MarketOpsTemporalDatabaseURL
		logger.Info("SAF registrar writes are routed to the dedicated MarketOps data boundary")
	}
	if strings.TrimSpace(databaseURL) == "" {
		return errors.New("SIGNALOPS_DATABASE_URL is required")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	client, err := kafkabroker.NewClient(kafkabroker.Config{Brokers: strings.Split(cfg.BrokerBrokers, ","), ClientID: "signalops-signal-assurance-registrar"})
	if err != nil {
		return err
	}
	defer closeClient(client)
	topic := broker.TopicName(cfg.Environment, broker.MarketOpsSignalAssuranceEligibleTopic)
	consumer, err := client.NewConsumer("signalops.signal-assurance-registrar.v1", []string{topic})
	if err != nil {
		return err
	}
	defer closeConsumer(consumer)
	repo, err := postgresstorage.OpenWithTemporal(ctx, databaseURL, temporalDatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	for {
		message, consumeErr := consumer.Consume(ctx)
		if consumeErr != nil {
			if errors.Is(consumeErr, context.Canceled) {
				return nil
			}
			return consumeErr
		}
		event, decodeErr := signalassurance.DecodeEligibleEvent(message.Value)
		if decodeErr != nil {
			logger.Error("invalid SAF eligible event; source offset remains uncommitted", "error", decodeErr)
			continue
		}
		contract, resolveErr := repo.ResolveSignalValidationContract(ctx, event)
		if resolveErr != nil && event.EvaluationMode == storage.SignalAssuranceModeResearch {
			provisional := signalassurance.ResearchContractFor(event)
			if upsertErr := repo.UpsertSignalValidationContract(ctx, provisional); upsertErr != nil {
				resolveErr = upsertErr
			} else {
				contract, resolveErr = repo.ResolveSignalValidationContract(ctx, event)
			}
		}
		if resolveErr != nil {
			logger.Error("SAF contract resolution failed; source offset remains uncommitted", "error", resolveErr, "eligible_event_id", event.EligibleEventID)
			continue
		}
		registration, registerErr := signalassurance.AssertionRegistration(event, contract)
		if registerErr == nil {
			_, _, registerErr = repo.RegisterSignalAssuranceAssertion(ctx, registration)
		}
		if registerErr != nil {
			logger.Error("SAF assertion registration failed; source offset remains uncommitted", "error", registerErr, "eligible_event_id", event.EligibleEventID)
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
		logger.Info("SAF assertion registered", "eligible_event_id", event.EligibleEventID, "signal_id", event.SignalID)
	}
}
func closeClient(client broker.Publisher) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = client.Close(ctx)
}
func closeConsumer(consumer broker.Consumer) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = consumer.Close(ctx)
}

package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/lukebabs/signalops/internal/api"
	kafkabroker "github.com/lukebabs/signalops/internal/broker/kafka"
	"github.com/lukebabs/signalops/internal/config"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
	"github.com/lukebabs/signalops/pkg/broker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg := config.Load()
	brokerClient, err := kafkabroker.NewClient(kafkabroker.Config{
		Brokers:  strings.Split(cfg.BrokerBrokers, ","),
		ClientID: "signalops-gateway",
	})
	if err != nil {
		logger.Error("signalops gateway broker setup failed", "error", err)
		os.Exit(1)
	}

	var queryRepo *postgresstorage.Repository
	if strings.TrimSpace(cfg.DatabaseURL) != "" {
		queryRepo, err = postgresstorage.Open(context.Background(), cfg.DatabaseURL)
		if err != nil {
			logger.Error("signalops gateway storage setup failed", "error", err)
			os.Exit(1)
		}
	}

	routerConfig := api.RouterConfig{
		ServiceName: "signalops-gateway",
		Publisher:   brokerClient,
		RawTopic:    broker.TopicName(cfg.Environment, broker.RawTopic),
	}
	if queryRepo != nil {
		routerConfig.QueryRepository = queryRepo
		routerConfig.PublishRepository = queryRepo
	}
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.NewRouter(routerConfig),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("signalops gateway starting", "addr", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		logger.Error("signalops gateway failed", "error", err)
		os.Exit(1)
	case sig := <-stopCh:
		logger.Info("signalops gateway stopping", "signal", sig.String())
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("signalops gateway shutdown failed", "error", err)
		os.Exit(1)
	}
	if err := brokerClient.Close(shutdownCtx); err != nil {
		logger.Error("signalops gateway broker shutdown failed", "error", err)
		os.Exit(1)
	}
	if queryRepo != nil {
		if err := queryRepo.Close(); err != nil {
			logger.Error("signalops gateway storage shutdown failed", "error", err)
			os.Exit(1)
		}
	}

	logger.Info("signalops gateway stopped")
}

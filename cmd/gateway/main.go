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

	"github.com/lukebabs/signalops/internal/adapters/marketdata/massive"
	"github.com/lukebabs/signalops/internal/api"
	kafkabroker "github.com/lukebabs/signalops/internal/broker/kafka"
	"github.com/lukebabs/signalops/internal/config"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
	"github.com/lukebabs/signalops/internal/syncratic/userapi"
	"github.com/lukebabs/signalops/pkg/broker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg := config.Load()
	if err := cfg.ValidateAuthConfiguration(); err != nil {
		logger.Error("signalops gateway authentication configuration is invalid", "error", err)
		os.Exit(1)
	}
	brokerClient, err := kafkabroker.NewClient(kafkabroker.Config{
		Brokers:  strings.Split(cfg.BrokerBrokers, ","),
		ClientID: "signalops-gateway",
	})
	if err != nil {
		logger.Error("signalops gateway broker setup failed", "error", err)
		os.Exit(1)
	}

	var queryRepo *postgresstorage.Repository
	var marketOpsQueryRepo *postgresstorage.Repository
	var subscriberWatchlistRepo *postgresstorage.Repository
	if strings.TrimSpace(cfg.DatabaseURL) != "" {
		queryRepo, err = postgresstorage.OpenWithTemporal(context.Background(), cfg.DatabaseURL, cfg.TemporalDatabaseURL)
		if err != nil {
			logger.Error("signalops gateway storage setup failed", "error", err)
			os.Exit(1)
		}
	}

	if strings.TrimSpace(cfg.MarketOpsDatabaseURL) != "" {
		marketOpsQueryRepo, err = postgresstorage.OpenWithTemporal(context.Background(), cfg.MarketOpsDatabaseURL, cfg.MarketOpsTemporalDatabaseURL)
		if err != nil {
			logger.Error("signalops gateway MarketOps storage setup failed", "error", err)
			os.Exit(1)
		}
		logger.Info("MarketOps gateway reads are routed to the dedicated data boundary")
	}

	routerConfig := api.RouterConfig{
		ServiceName: "signalops-gateway",
		Publisher:   brokerClient,
		RawTopic:    broker.TopicName(cfg.Environment, broker.RawTopic),
		Environment: cfg.Environment,
		Auth: api.AuthConfig{
			Enabled:  cfg.AuthEnabled,
			Issuer:   cfg.AuthIssuer,
			JWKSURL:  cfg.AuthJWKSURL,
			Audience: cfg.AuthAudience,
		},
		NotificationEncryptionKey:      cfg.NotificationEncryptionKey,
		SubscriberListsEnabled:         cfg.SubscriberListsEnabled,
		SubscriberSubscriptionsEnabled: cfg.SubscriberSubscriptionsEnabled,
		SubscriberListsPilotTenants:    subscriberPilotTenants(cfg.SubscriberListsPilotTenants),
		StripeWebhookSecret:            cfg.StripeWebhookSecret,
	}
	if cfg.SubscriberSubscriptionsEnabled && !cfg.SubscriberListsEnabled {
		logger.Error("subscription enforcement requires the subscriber catalog gateway database")
		os.Exit(1)
	}
	if cfg.SubscriberListsEnabled {
		if strings.TrimSpace(cfg.SubscriberListsDatabaseURL) == "" {
			logger.Error("subscriber lists require SIGNALOPS_SUBSCRIBER_GATEWAY_DATABASE_URL")
			os.Exit(1)
		}
		subscriberWatchlistRepo, err = postgresstorage.Open(context.Background(), cfg.SubscriberListsDatabaseURL)
		if err != nil {
			logger.Error("subscriber watchlist storage setup failed", "error", err)
			os.Exit(1)
		}
		routerConfig.SubscriberWatchlistRepository = subscriberWatchlistRepo
		routerConfig.SubscriberCatalogRepository = subscriberWatchlistRepo
		routerConfig.SubscriberEntitlementRepository = subscriberWatchlistRepo
		routerConfig.SubscriberCatalogMembershipRepository = subscriberWatchlistRepo
		routerConfig.SubscriberSubscriptionRepository = subscriberWatchlistRepo
		routerConfig.SubscriberSubscriptionAdministrationRepository = subscriberWatchlistRepo
	}

	if key := strings.TrimSpace(os.Getenv("SIGNALOPS_MASSIVE_API_KEY")); key != "" {
		if client, clientErr := massive.NewClient(massive.LoadClientConfigFromEnv()); clientErr == nil {
			routerConfig.MarketQuoteClient = client
		} else {
			logger.Warn("massive quote client disabled", "error", clientErr)
		}
	}
	if queryRepo != nil {
		routerConfig.QueryRepository = queryRepo
		if marketOpsQueryRepo != nil {
			routerConfig.MarketOpsQueryRepository = marketOpsQueryRepo
		}
		routerConfig.AccessRepository = queryRepo
		routerConfig.CyberOpsConnectRepository = queryRepo
		routerConfig.PlatformDefinitionRepository = queryRepo
		routerConfig.PublishRepository = queryRepo
	}
	if strings.TrimSpace(os.Getenv("SYNCRATIC_API_BASE_URL")) != "" {
		synClient, err := userapi.New(userapi.ConfigFromEnv())
		if err != nil {
			logger.Error("signalops gateway syncratic user api setup failed", "error", err)
			os.Exit(1)
		}
		routerConfig.SyncraticAskClient = synClient
	}
	workerCtx, stopSyncraticWorker := context.WithCancel(context.Background())
	defer stopSyncraticWorker()
	syncraticWorkerRepo := queryRepo
	if marketOpsQueryRepo != nil {
		syncraticWorkerRepo = marketOpsQueryRepo
	}
	if syncraticWorkerRepo != nil && routerConfig.SyncraticAskClient != nil {
		api.StartSyncraticIntelligenceWorker(workerCtx, syncraticWorkerRepo, routerConfig.SyncraticAskClient)
		logger.Info("syncratic intelligence worker enabled")
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
	if key := strings.TrimSpace(os.Getenv("SIGNALOPS_MASSIVE_API_KEY")); key != "" {
		if client, clientErr := massive.NewClient(massive.LoadClientConfigFromEnv()); clientErr == nil {
			routerConfig.MarketQuoteClient = client
		} else {
			logger.Warn("massive quote client disabled", "error", clientErr)
		}
	}
	if subscriberWatchlistRepo != nil {
		if err := subscriberWatchlistRepo.Close(); err != nil {
			logger.Error("subscriber watchlist storage shutdown failed", "error", err)
			os.Exit(1)
		}
	}
	if marketOpsQueryRepo != nil {
		if err := marketOpsQueryRepo.Close(); err != nil {
			logger.Error("signalops gateway MarketOps storage shutdown failed", "error", err)
			os.Exit(1)
		}
	}

	if queryRepo != nil {
		if err := queryRepo.Close(); err != nil {
			logger.Error("signalops gateway storage shutdown failed", "error", err)
			os.Exit(1)
		}
	}

	logger.Info("signalops gateway stopped")
}

func subscriberPilotTenants(raw string) map[string]struct{} {
	values := map[string]struct{}{}
	for _, value := range strings.Split(raw, ",") {
		if tenantID := strings.TrimSpace(value); tenantID != "" {
			values[tenantID] = struct{}{}
		}
	}
	return values
}

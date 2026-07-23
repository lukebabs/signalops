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

	"github.com/lukebabs/signalops/internal/adapters/marketdata/massive"
	kafkabroker "github.com/lukebabs/signalops/internal/broker/kafka"
	"github.com/lukebabs/signalops/internal/config"
	"github.com/lukebabs/signalops/internal/storage"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
)

const workerID = "marketops-asset-backfill-worker"

type repository interface {
	storage.MarketOpsAssetBackfillRepository
	storage.EquityReconciliationRepository
	storage.PublishRepository
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("marketops asset backfill worker failed", "error", err)
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
	repo, err := postgresstorage.OpenWithTemporal(ctx, cfg.DatabaseURL, cfg.TemporalDatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	client, err := massive.NewClient(massive.LoadClientConfigFromEnv())
	if err != nil {
		return err
	}
	publisher, err := kafkabroker.NewClient(kafkabroker.Config{Brokers: strings.Split(cfg.BrokerBrokers, ","), ClientID: workerID})
	if err != nil {
		return err
	}
	defer publisher.Close(context.Background())
	for {
		if err := processAwaiting(ctx, repo); err != nil {
			logger.Error("asset backfill normalization sweep failed", "error", err)
		}
		job, err := repo.ClaimNextMarketOpsAssetBackfillJob(ctx, workerID, time.Now().UTC())
		if errors.Is(err, storage.ErrNotFound) {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(5 * time.Second):
				continue
			}
		}
		if err != nil {
			return err
		}
		if err := processJob(ctx, repo, client, publisher, job); err != nil {
			logger.Error("asset backfill job failed", "job_id", job.BackfillJobID, "error", err)
		}
	}
}

func processJob(ctx context.Context, repo repository, client massive.ScheduledPullClient, publisher *kafkabroker.Client, job storage.MarketOpsAssetBackfillJobRecord) error {
	environment := strings.TrimSpace(os.Getenv("SIGNALOPS_ENV"))
	if environment == "" {
		environment = "local"
	}
	requests, sessions, failures := 0, 0, []string{}
	for day := job.StartDate; !day.After(job.EndDate); day = day.AddDate(0, 0, 1) {
		if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
			continue
		}
		report, err := massive.RunScheduledPull(ctx, massive.ScheduledPullConfig{TenantID: job.TenantID, SourceID: "src-massive", Environment: environment, ObservationDate: day, Companies: []massive.MegacapCompanySeed{{Ticker: job.Symbol, Company: job.Symbol}}, IncludeEquityEOD: true, DryRun: false, ContinueOnError: true, RequestDelay: 250 * time.Millisecond, MaxRetries: 2, RetryBackoff: time.Second, PublishRepository: repo, CorrelationPrefix: job.BackfillJobID}, client, publisher)
		requests += report.ProviderRequests
		if noEquityBarResponse(report.Errors) {
			continue // Market holiday or non-trading session: it is not a failed backfill session.
		}
		if err != nil || report.Failures > 0 {
			failures = append(failures, day.Format("2006-01-02"))
			continue
		}
		sessions++
	}
	job.RequestedSessions = sessions
	job.ProviderRequests = requests
	job.FailedSessions = len(failures)
	job.ResultJSON, _ = json.Marshal(map[string]any{"dataset": "equity_eod_prices", "options_history": "unavailable", "failed_dates": failures})
	return finalize(ctx, repo, &job)
}

func noEquityBarResponse(errors []string) bool {
	if len(errors) == 0 {
		return false
	}
	for _, item := range errors {
		if !strings.Contains(item, "massive equity aggregate response contained no bars") {
			return false
		}
	}
	return true
}

func processAwaiting(ctx context.Context, repo repository) error {
	jobs, err := repo.ListMarketOpsAssetBackfillJobs(ctx, storage.MarketOpsAssetBackfillJobFilter{Status: "awaiting_normalization", Limit: 50})
	if err != nil {
		return err
	}
	for i := range jobs {
		if err := finalize(ctx, repo, &jobs[i]); err != nil {
			return err
		}
	}
	return nil
}

func finalize(ctx context.Context, repo repository, job *storage.MarketOpsAssetBackfillJobRecord) error {
	completed := 0
	for day := job.StartDate; !day.After(job.EndDate); day = day.AddDate(0, 0, 1) {
		if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
			continue
		}
		ok, err := repo.HasNormalizedEquity(ctx, job.TenantID, "src-massive", job.Symbol, day)
		if err != nil {
			return err
		}
		if ok {
			completed++
		}
	}
	job.CompletedSessions = completed
	if completed+job.FailedSessions > job.RequestedSessions {
		job.RequestedSessions = completed + job.FailedSessions
	}
	now := time.Now().UTC()
	if job.FailedSessions > 0 {
		job.Status = "partial"
		job.CompletedAt = &now
		job.ErrorMessage = "one or more provider sessions failed"
	} else if completed == job.RequestedSessions {
		job.Status = "succeeded"
		job.CompletedAt = &now
		job.ErrorMessage = ""
	} else {
		job.Status = "awaiting_normalization"
		job.ErrorMessage = "waiting for normalized equity history"
	}
	return repo.UpdateMarketOpsAssetBackfillJob(ctx, *job)
}

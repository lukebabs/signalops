package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/lukebabs/signalops/internal/adapters/marketdata/massive"
	kafkabroker "github.com/lukebabs/signalops/internal/broker/kafka"
	"github.com/lukebabs/signalops/internal/config"
	"github.com/lukebabs/signalops/internal/storage"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger, os.Args[1:]); err != nil {
		logger.Error("signalops massive scheduler failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger, args []string) error {
	cfg := config.Load()
	observationDate := envOrDefault("SIGNALOPS_MASSIVE_OBSERVATION_DATE", "")
	datasets := envOrDefault("SIGNALOPS_MASSIVE_DATASETS", "equity,options")
	maxCompanies := envIntOrDefault("SIGNALOPS_MASSIVE_MAX_COMPANIES", 50)
	optionsLimit := envIntOrDefault("SIGNALOPS_MASSIVE_OPTIONS_LIMIT", 100)
	requestDelay := envDurationOrDefault("SIGNALOPS_MASSIVE_REQUEST_DELAY", 250*time.Millisecond)
	maxRetries := envIntOrDefault("SIGNALOPS_MASSIVE_MAX_RETRIES", 1)
	retryBackoff := envDurationOrDefault("SIGNALOPS_MASSIVE_RETRY_BACKOFF", time.Second)
	maxProviderRequests := envIntOrDefault("SIGNALOPS_MASSIVE_MAX_PROVIDER_REQUESTS", 0)
	maxEventsBuilt := envIntOrDefault("SIGNALOPS_MASSIVE_MAX_EVENTS_BUILT", 0)
	maxEventsPublished := envIntOrDefault("SIGNALOPS_MASSIVE_MAX_EVENTS_PUBLISHED", 0)
	dryRun := envBoolOrDefault("SIGNALOPS_MASSIVE_DRY_RUN", true)
	continueOnError := envBoolOrDefault("SIGNALOPS_MASSIVE_CONTINUE_ON_ERROR", true)
	tenantID := envOrDefault("SIGNALOPS_MASSIVE_TENANT_ID", "tenant-local")
	sourceID := envOrDefault("SIGNALOPS_MASSIVE_SOURCE_ID", "src-massive")
	scheduleInterval := envDurationOrDefault("SIGNALOPS_MASSIVE_SCHEDULE_INTERVAL", 24*time.Hour)
	maxRuns := envIntOrDefault("SIGNALOPS_MASSIVE_SCHEDULE_MAX_RUNS", 0)
	runImmediately := envBoolOrDefault("SIGNALOPS_MASSIVE_SCHEDULE_RUN_IMMEDIATELY", true)
	continueOnRunError := envBoolOrDefault("SIGNALOPS_MASSIVE_SCHEDULE_CONTINUE_ON_ERROR", true)

	flags := flag.NewFlagSet("massive-scheduler", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.StringVar(&observationDate, "date", observationDate, "observation date in YYYY-MM-DD; defaults to previous UTC day")
	flags.StringVar(&datasets, "datasets", datasets, "comma-separated datasets: equity, options")
	flags.IntVar(&maxCompanies, "max-companies", maxCompanies, "maximum megacap companies to process")
	flags.IntVar(&optionsLimit, "options-limit", optionsLimit, "provider option contract listing limit per underlying")
	flags.DurationVar(&requestDelay, "request-delay", requestDelay, "delay before each provider request, such as 250ms or 1s")
	flags.IntVar(&maxRetries, "max-retries", maxRetries, "maximum retry attempts for each provider request")
	flags.DurationVar(&retryBackoff, "retry-backoff", retryBackoff, "base retry backoff, multiplied by retry attempt")
	flags.IntVar(&maxProviderRequests, "max-provider-requests", maxProviderRequests, "maximum provider requests allowed per run; 0 disables the limit")
	flags.IntVar(&maxEventsBuilt, "max-events-built", maxEventsBuilt, "maximum raw events allowed to be built per run; 0 disables the limit")
	flags.IntVar(&maxEventsPublished, "max-events-published", maxEventsPublished, "maximum raw events allowed to be published per run; 0 disables the limit")
	flags.BoolVar(&dryRun, "dry-run", dryRun, "build events without publishing")
	flags.BoolVar(&continueOnError, "continue-on-error", continueOnError, "continue processing symbols after provider/build/publish failures")
	flags.StringVar(&tenantID, "tenant-id", tenantID, "tenant id for emitted raw events")
	flags.StringVar(&sourceID, "source-id", sourceID, "source id for emitted raw events")
	flags.DurationVar(&scheduleInterval, "schedule-interval", scheduleInterval, "interval between scheduled pull runs")
	flags.IntVar(&maxRuns, "max-runs", maxRuns, "maximum scheduled runs before exit; 0 means run until stopped")
	flags.BoolVar(&runImmediately, "run-immediately", runImmediately, "run once immediately before waiting for the interval")
	flags.BoolVar(&continueOnRunError, "continue-on-run-error", continueOnRunError, "continue scheduling after a pull run returns an error")
	if err := flags.Parse(args); err != nil {
		return err
	}

	includeEquity, includeOptions, err := parseDatasets(datasets)
	if err != nil {
		return err
	}
	day, err := parseObservationDate(observationDate)
	if err != nil {
		return err
	}
	companies, err := massive.TopMegacapCompanies()
	if err != nil {
		return fmt.Errorf("load megacap universe: %w", err)
	}
	if maxCompanies > 0 && maxCompanies < len(companies) {
		companies = companies[:maxCompanies]
	}

	massiveClient, err := massive.NewClient(massive.LoadClientConfigFromEnv())
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var brokerClient *kafkabroker.Client
	if !dryRun {
		brokerClient, err = kafkabroker.NewClient(kafkabroker.Config{
			Brokers:  strings.Split(cfg.BrokerBrokers, ","),
			ClientID: "signalops-massive-scheduler",
		})
		if err != nil {
			return err
		}
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if closeErr := brokerClient.Close(shutdownCtx); closeErr != nil {
				logger.Error("signalops massive scheduler broker shutdown failed", "error", closeErr)
			}
		}()
	}

	var runRepo storage.SchedulerRunRepository
	var repoCloser interface{ Close() error }
	if strings.TrimSpace(cfg.DatabaseURL) != "" {
		repo, err := postgresstorage.Open(ctx, cfg.DatabaseURL)
		if err != nil {
			return err
		}
		runRepo = repo
		repoCloser = repo
		defer func() {
			if closeErr := repoCloser.Close(); closeErr != nil {
				logger.Error("signalops massive scheduler storage shutdown failed", "error", closeErr)
			}
		}()
	}

	pullCfg := massive.ScheduledPullConfig{
		TenantID:            tenantID,
		SourceID:            sourceID,
		Environment:         cfg.Environment,
		ObservationDate:     day,
		Companies:           companies,
		IncludeEquityEOD:    includeEquity,
		IncludeOptions:      includeOptions,
		OptionsLimit:        optionsLimit,
		RequestDelay:        requestDelay,
		MaxRetries:          maxRetries,
		RetryBackoff:        retryBackoff,
		MaxProviderRequests: maxProviderRequests,
		MaxEventsBuilt:      maxEventsBuilt,
		MaxEventsPublished:  maxEventsPublished,
		DryRun:              dryRun,
		ContinueOnError:     continueOnError,
	}

	loopReport, err := massive.RunScheduledLoop(ctx, massive.ScheduledLoopConfig{
		Interval:           scheduleInterval,
		MaxRuns:            maxRuns,
		RunImmediately:     runImmediately,
		ContinueOnRunError: continueOnRunError,
	}, func(runCtx context.Context) (massive.ScheduledPullReport, error) {
		startedAt := time.Now().UTC()
		report, runErr := massive.RunScheduledPull(runCtx, pullCfg, massiveClient, brokerClient)
		completedAt := time.Now().UTC()
		encoded, marshalErr := json.Marshal(report)
		if marshalErr == nil {
			logger.Info("signalops massive scheduler run report", "report", string(encoded))
		}
		if runRepo != nil {
			if persistErr := persistSchedulerRun(runCtx, runRepo, schedulerPersistInput{
				TenantID:         tenantID,
				SourceID:         sourceID,
				Datasets:         selectedDatasetNames(includeEquity, includeOptions),
				PullConfig:       pullCfg,
				Report:           report,
				RunErr:           runErr,
				StartedAt:        startedAt,
				CompletedAt:      completedAt,
				LoopMaxRuns:      maxRuns,
				ScheduleInterval: scheduleInterval,
			}); persistErr != nil {
				logger.Error("signalops massive scheduler persist failed", "error", persistErr)
				if runErr == nil {
					return report, persistErr
				}
			}
		}
		return report, runErr
	})

	encoded, marshalErr := json.Marshal(loopReport)
	if marshalErr == nil {
		logger.Info("signalops massive scheduler loop report", "report", string(encoded))
	}
	return err
}

func parseObservationDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		now := time.Now().UTC().AddDate(0, 0, -1)
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC), nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid observation date %q: %w", value, err)
	}
	return parsed.UTC(), nil
}

func parseDatasets(value string) (bool, bool, error) {
	includeEquity := false
	includeOptions := false
	for _, part := range strings.Split(value, ",") {
		switch strings.ToLower(strings.TrimSpace(part)) {
		case "", "none":
		case "equity", "equities", "eod", "equity_eod":
			includeEquity = true
		case "options", "option", "option_contracts":
			includeOptions = true
		default:
			return false, false, fmt.Errorf("unsupported massive dataset %q", part)
		}
	}
	if !includeEquity && !includeOptions {
		return false, false, fmt.Errorf("at least one massive dataset must be selected")
	}
	return includeEquity, includeOptions, nil
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envIntOrDefault(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envBoolOrDefault(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDurationOrDefault(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

type schedulerPersistInput struct {
	TenantID         string
	SourceID         string
	Datasets         []string
	PullConfig       massive.ScheduledPullConfig
	Report           massive.ScheduledPullReport
	RunErr           error
	StartedAt        time.Time
	CompletedAt      time.Time
	LoopMaxRuns      int
	ScheduleInterval time.Duration
}

func persistSchedulerRun(ctx context.Context, repo storage.SchedulerRunRepository, input schedulerPersistInput) error {
	status := storage.RunStatusSucceeded
	errorMessage := ""
	if input.RunErr != nil {
		status = storage.RunStatusFailed
		errorMessage = input.RunErr.Error()
	}
	configJSON, err := json.Marshal(map[string]any{
		"dry_run":               input.PullConfig.DryRun,
		"include_equity_eod":    input.PullConfig.IncludeEquityEOD,
		"include_options":       input.PullConfig.IncludeOptions,
		"options_limit":         input.PullConfig.OptionsLimit,
		"max_provider_requests": input.PullConfig.MaxProviderRequests,
		"max_events_built":      input.PullConfig.MaxEventsBuilt,
		"max_events_published":  input.PullConfig.MaxEventsPublished,
		"max_retries":           input.PullConfig.MaxRetries,
		"request_delay":         input.PullConfig.RequestDelay.String(),
		"retry_backoff":         input.PullConfig.RetryBackoff.String(),
		"schedule_interval":     input.ScheduleInterval.String(),
		"schedule_max_runs":     input.LoopMaxRuns,
	})
	if err != nil {
		return fmt.Errorf("marshal scheduler config: %w", err)
	}
	reportJSON, err := json.Marshal(input.Report)
	if err != nil {
		return fmt.Errorf("marshal scheduler report: %w", err)
	}
	runID := schedulerRunID(input.SourceID, input.StartedAt)
	if err := repo.UpsertSchedulerRun(ctx, storage.SchedulerRunRecord{
		RunID:            runID,
		TenantID:         input.TenantID,
		SourceID:         input.SourceID,
		SourceAdapter:    massive.AdapterID,
		Datasets:         input.Datasets,
		ObservationDate:  input.Report.ObservationDate,
		DryRun:           input.Report.DryRun,
		Status:           status,
		StartedAt:        input.StartedAt,
		CompletedAt:      &input.CompletedAt,
		EventsBuilt:      input.Report.EventsBuilt,
		EventsPublished:  input.Report.EventsPublished,
		ProviderRequests: input.Report.ProviderRequests,
		ProviderRetries:  input.Report.ProviderRetries,
		Failures:         input.Report.Failures,
		ConfigJSON:       configJSON,
		ReportJSON:       reportJSON,
		ErrorMessage:     errorMessage,
	}); err != nil {
		return err
	}
	for _, dataset := range providerUsageDatasets(input.Datasets) {
		if err := repo.InsertProviderUsage(ctx, storage.ProviderUsageRecord{
			UsageID:      runID + ":" + dataset,
			RunID:        runID,
			Provider:     "massive",
			Dataset:      dataset,
			RequestCount: providerRequestCountForDataset(input.Report),
			RetryCount:   providerRetryCountForDataset(input.Report),
			EventCount:   providerEventCountForDataset(input.Report, dataset),
			BudgetJSON:   configJSON,
		}); err != nil {
			return err
		}
	}
	return nil
}

func schedulerRunID(sourceID string, startedAt time.Time) string {
	cleanSource := strings.NewReplacer(" ", "_", ":", "_", "/", "_").Replace(strings.TrimSpace(sourceID))
	if cleanSource == "" {
		cleanSource = "src"
	}
	return "massive:" + cleanSource + ":" + startedAt.UTC().Format("20060102T150405.000000000Z")
}

func selectedDatasetNames(includeEquity bool, includeOptions bool) []string {
	datasets := []string{}
	if includeEquity {
		datasets = append(datasets, massive.DatasetEquityEODPrices)
	}
	if includeOptions {
		datasets = append(datasets, massive.DatasetOptionsContractsDaily)
	}
	return datasets
}

func providerUsageDatasets(datasets []string) []string {
	if len(datasets) > 1 {
		return []string{"all"}
	}
	return datasets
}

func providerRequestCountForDataset(report massive.ScheduledPullReport) int {
	return report.ProviderRequests
}

func providerRetryCountForDataset(report massive.ScheduledPullReport) int {
	return report.ProviderRetries
}

func providerEventCountForDataset(report massive.ScheduledPullReport, dataset string) int {
	if dataset != "all" {
		return report.EventsByDataset[dataset]
	}
	total := 0
	for _, count := range report.EventsByDataset {
		total += count
	}
	return total
}

package main

import (
	"context"
	"encoding/json"
	"errors"
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
		logger.Error("signalops massive puller failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger, args []string) error {
	cfg := config.Load()
	observationDate := envOrDefault("SIGNALOPS_MASSIVE_OBSERVATION_DATE", "")
	startDate := ""
	endDate := ""
	symbols := ""
	maxObservationDays := 1
	datasets := envOrDefault("SIGNALOPS_MASSIVE_DATASETS", "equity,options")
	maxCompanies := envIntOrDefault("SIGNALOPS_MASSIVE_MAX_COMPANIES", 50)
	optionsLimit := envIntOrDefault("SIGNALOPS_MASSIVE_OPTIONS_LIMIT", 100)
	requestDelay := envDurationOrDefault("SIGNALOPS_MASSIVE_REQUEST_DELAY", 0)
	maxRetries := envIntOrDefault("SIGNALOPS_MASSIVE_MAX_RETRIES", 1)
	retryBackoff := envDurationOrDefault("SIGNALOPS_MASSIVE_RETRY_BACKOFF", time.Second)
	maxProviderRequests := envIntOrDefault("SIGNALOPS_MASSIVE_MAX_PROVIDER_REQUESTS", 0)
	maxEventsBuilt := envIntOrDefault("SIGNALOPS_MASSIVE_MAX_EVENTS_BUILT", 0)
	maxEventsPublished := envIntOrDefault("SIGNALOPS_MASSIVE_MAX_EVENTS_PUBLISHED", 0)
	dryRun := envBoolOrDefault("SIGNALOPS_MASSIVE_DRY_RUN", true)
	continueOnError := envBoolOrDefault("SIGNALOPS_MASSIVE_CONTINUE_ON_ERROR", false)
	tenantID := envOrDefault("SIGNALOPS_MASSIVE_TENANT_ID", "tenant-local")
	sourceID := envOrDefault("SIGNALOPS_MASSIVE_SOURCE_ID", "src-massive")
	flags := flag.NewFlagSet("massive-puller", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.StringVar(&observationDate, "date", observationDate, "single observation date in YYYY-MM-DD; defaults to previous UTC day")
	flags.StringVar(&startDate, "start-date", startDate, "inclusive historical observation start date")
	flags.StringVar(&endDate, "end-date", endDate, "inclusive historical observation end date")
	flags.StringVar(&symbols, "symbols", symbols, "comma-separated exact Top 50 symbols; empty uses seed order")
	flags.IntVar(&maxObservationDays, "max-observation-days", maxObservationDays, "maximum weekdays in an explicit date range")
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
	if err := flags.Parse(args); err != nil {
		return err
	}

	includeEquity, includeOptions, err := parseDatasets(datasets)
	if err != nil {
		return err
	}
	days, err := parseObservationDates(observationDate, startDate, endDate, maxObservationDays)
	if err != nil {
		return err
	}
	companies, err := massive.TopMegacapCompanies()
	if err != nil {
		return fmt.Errorf("load megacap universe: %w", err)
	}
	companies, err = selectCompanies(companies, symbols)
	if err != nil {
		return err
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

	var publishRepo storage.PublishRepository
	var repoCloser interface{ Close() error }
	if strings.TrimSpace(cfg.DatabaseURL) != "" {
		repo, err := postgresstorage.OpenWithTemporal(ctx, cfg.DatabaseURL, cfg.TemporalDatabaseURL)
		if err != nil {
			return err
		}
		publishRepo = repo
		repoCloser = repo
		defer func() {
			if closeErr := repoCloser.Close(); closeErr != nil {
				logger.Error("signalops massive puller storage shutdown failed", "error", closeErr)
			}
		}()
	}

	var brokerClient *kafkabroker.Client
	if !dryRun {
		brokerClient, err = kafkabroker.NewClient(kafkabroker.Config{
			Brokers:  strings.Split(cfg.BrokerBrokers, ","),
			ClientID: "signalops-massive-puller",
		})
		if err != nil {
			return err
		}
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if closeErr := brokerClient.Close(shutdownCtx); closeErr != nil {
				logger.Error("signalops massive puller broker shutdown failed", "error", closeErr)
			}
		}()
	}

	reports := make([]massive.ScheduledPullReport, 0, len(days))
	totalRequests, totalRetries, totalFailures, totalBuilt, totalPublished := 0, 0, 0, 0, 0
	for _, day := range days {
		remainingRequests, budgetErr := remainingBudget(maxProviderRequests, totalRequests, "provider request")
		if budgetErr != nil {
			return budgetErr
		}
		remainingBuilt, budgetErr := remainingBudget(maxEventsBuilt, totalBuilt, "built event")
		if budgetErr != nil {
			return budgetErr
		}
		remainingPublished, budgetErr := remainingBudget(maxEventsPublished, totalPublished, "published event")
		if budgetErr != nil {
			return budgetErr
		}
		report, pullErr := massive.RunScheduledPull(ctx, massive.ScheduledPullConfig{
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
			MaxProviderRequests: remainingRequests,
			MaxEventsBuilt:      remainingBuilt,
			MaxEventsPublished:  remainingPublished,
			DryRun:              dryRun,
			ContinueOnError:     continueOnError,
			PublishRepository:   publishRepo,
		}, massiveClient, brokerClient)
		report.ObservationDate = day
		reports = append(reports, report)
		totalRequests += report.ProviderRequests
		totalRetries += report.ProviderRetries
		totalFailures += report.Failures
		totalBuilt += report.EventsBuilt
		totalPublished += report.EventsPublished
		if pullErr != nil {
			return pullErr
		}
	}

	encoded, marshalErr := json.Marshal(map[string]any{"observation_days": len(days), "reports": reports, "provider_requests": totalRequests, "provider_retries": totalRetries, "failures": totalFailures, "events_built": totalBuilt, "events_published": totalPublished})
	if marshalErr == nil {
		logger.Info("signalops massive puller report", "report", string(encoded))
	}
	return nil
}

func parseObservationDates(single, start, end string, maxDays int) ([]time.Time, error) {
	start = strings.TrimSpace(start)
	end = strings.TrimSpace(end)
	if start == "" && end == "" {
		day, err := parseObservationDate(single)
		if err != nil {
			return nil, err
		}
		return []time.Time{day}, nil
	}
	if start == "" || end == "" {
		return nil, errors.New("start-date and end-date must be provided together")
	}
	if maxDays <= 0 || maxDays > 366 {
		return nil, errors.New("max-observation-days must be between 1 and 366")
	}
	first, err := parseObservationDate(start)
	if err != nil {
		return nil, fmt.Errorf("start-date: %w", err)
	}
	last, err := parseObservationDate(end)
	if err != nil {
		return nil, fmt.Errorf("end-date: %w", err)
	}
	if last.Before(first) {
		return nil, errors.New("end-date must not precede start-date")
	}
	days := []time.Time{}
	for day := first; !day.After(last); day = day.AddDate(0, 0, 1) {
		if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
			continue
		}
		days = append(days, day)
		if len(days) > maxDays {
			return nil, fmt.Errorf("date range exceeds max-observation-days %d", maxDays)
		}
	}
	if len(days) == 0 {
		return nil, errors.New("date range contains no weekdays")
	}
	return days, nil
}

func selectCompanies(companies []massive.MegacapCompanySeed, value string) ([]massive.MegacapCompanySeed, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return companies, nil
	}
	bySymbol := map[string]massive.MegacapCompanySeed{}
	for _, company := range companies {
		bySymbol[strings.ToUpper(strings.TrimSpace(company.Ticker))] = company
	}
	selected := []massive.MegacapCompanySeed{}
	seen := map[string]bool{}
	for _, item := range strings.Split(value, ",") {
		symbol := strings.ToUpper(strings.TrimSpace(item))
		if symbol == "" || seen[symbol] {
			continue
		}
		company, ok := bySymbol[symbol]
		if !ok {
			return nil, fmt.Errorf("symbol %s is not in the Top 50 seed", symbol)
		}
		selected = append(selected, company)
		seen[symbol] = true
	}
	if len(selected) == 0 {
		return nil, errors.New("symbols must contain at least one Top 50 ticker")
	}
	return selected, nil
}

func remainingBudget(configured, used int, name string) (int, error) {
	if configured <= 0 {
		return 0, nil
	}
	remaining := configured - used
	if remaining <= 0 {
		return 0, fmt.Errorf("%s budget exceeded: limit %d", name, configured)
	}
	return remaining, nil
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

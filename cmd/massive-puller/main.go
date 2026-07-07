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
	datasets := envOrDefault("SIGNALOPS_MASSIVE_DATASETS", "equity,options")
	maxCompanies := envIntOrDefault("SIGNALOPS_MASSIVE_MAX_COMPANIES", 50)
	optionsLimit := envIntOrDefault("SIGNALOPS_MASSIVE_OPTIONS_LIMIT", 100)
	requestDelay := envDurationOrDefault("SIGNALOPS_MASSIVE_REQUEST_DELAY", 0)
	maxRetries := envIntOrDefault("SIGNALOPS_MASSIVE_MAX_RETRIES", 1)
	retryBackoff := envDurationOrDefault("SIGNALOPS_MASSIVE_RETRY_BACKOFF", time.Second)
	dryRun := envBoolOrDefault("SIGNALOPS_MASSIVE_DRY_RUN", true)
	continueOnError := envBoolOrDefault("SIGNALOPS_MASSIVE_CONTINUE_ON_ERROR", false)
	tenantID := envOrDefault("SIGNALOPS_MASSIVE_TENANT_ID", "tenant-local")
	sourceID := envOrDefault("SIGNALOPS_MASSIVE_SOURCE_ID", "src-massive")
	flags := flag.NewFlagSet("massive-puller", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.StringVar(&observationDate, "date", observationDate, "observation date in YYYY-MM-DD; defaults to previous UTC day")
	flags.StringVar(&datasets, "datasets", datasets, "comma-separated datasets: equity, options")
	flags.IntVar(&maxCompanies, "max-companies", maxCompanies, "maximum megacap companies to process")
	flags.IntVar(&optionsLimit, "options-limit", optionsLimit, "provider option contract listing limit per underlying")
	flags.DurationVar(&requestDelay, "request-delay", requestDelay, "delay before each provider request, such as 250ms or 1s")
	flags.IntVar(&maxRetries, "max-retries", maxRetries, "maximum retry attempts for each provider request")
	flags.DurationVar(&retryBackoff, "retry-backoff", retryBackoff, "base retry backoff, multiplied by retry attempt")
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

	report, err := massive.RunScheduledPull(ctx, massive.ScheduledPullConfig{
		TenantID:         tenantID,
		SourceID:         sourceID,
		Environment:      cfg.Environment,
		ObservationDate:  day,
		Companies:        companies,
		IncludeEquityEOD: includeEquity,
		IncludeOptions:   includeOptions,
		OptionsLimit:     optionsLimit,
		RequestDelay:     requestDelay,
		MaxRetries:       maxRetries,
		RetryBackoff:     retryBackoff,
		DryRun:           dryRun,
		ContinueOnError:  continueOnError,
	}, massiveClient, brokerClient)

	encoded, marshalErr := json.Marshal(report)
	if marshalErr == nil {
		logger.Info("signalops massive puller report", "report", string(encoded))
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

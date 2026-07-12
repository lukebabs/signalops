package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/config"
	"github.com/lukebabs/signalops/internal/marketops/dsm"
	"github.com/lukebabs/signalops/internal/signals"
	"github.com/lukebabs/signalops/internal/storage"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
	"github.com/lukebabs/signalops/pkg/broker"
)

type cliConfig struct {
	RunID                string
	TenantID             string
	SourceID             string
	SourceAdapter        string
	Dataset              string
	DetectorID           string
	DetectorVersion      string
	RequestedBy          string
	WindowStart          time.Time
	WindowEnd            time.Time
	Symbols              []string
	MaxRecords           int
	BatchSize            int
	AutoAcceptConfidence float64
	PythonBin            string
}

type runMetrics struct {
	RunID                string         `json:"run_id"`
	Scanned              int            `json:"scanned"`
	Signals              int            `json:"signals"`
	Artifacts            int            `json:"artifacts"`
	GraphProposals       int            `json:"graph_proposals"`
	PolicyResults        int            `json:"policy_results"`
	RecommendationCounts map[string]int `json:"recommendation_counts"`
	Batches              int            `json:"batches"`
	MaxRecords           int            `json:"max_records"`
	BatchSize            int            `json:"batch_size"`
	StartedAt            string         `json:"started_at"`
	CompletedAt          string         `json:"completed_at,omitempty"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("marketops backtest failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg := config.Load()
	if strings.TrimSpace(cfg.DatabaseURL) == "" || strings.TrimSpace(cfg.TemporalDatabaseURL) == "" {
		return errors.New("SIGNALOPS_DATABASE_URL and SIGNALOPS_TEMPORAL_DATABASE_URL are required")
	}
	cli, err := loadCLIConfig()
	if err != nil {
		return err
	}
	ctx := context.Background()
	repo, err := postgresstorage.OpenWithTemporal(ctx, cfg.DatabaseURL, cfg.TemporalDatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()

	filters, _ := json.Marshal(map[string]any{"symbols": cli.Symbols, "max_records": cli.MaxRecords})
	params, _ := json.Marshal(map[string]any{"detector_id": cli.DetectorID, "detector_version": cli.DetectorVersion, "auto_accept_confidence": cli.AutoAcceptConfidence})
	now := time.Now().UTC()
	runRecord := storage.MarketOpsBacktestRunRecord{
		RunID: cli.RunID, TenantID: cli.TenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance",
		SourceID: cli.SourceID, SourceAdapter: cli.SourceAdapter, Dataset: cli.Dataset, DetectorID: cli.DetectorID,
		DetectorVersion: cli.DetectorVersion, Status: storage.RunStatusStarted, RequestedBy: cli.RequestedBy,
		WindowStart: cli.WindowStart, WindowEnd: cli.WindowEnd, StartedAt: now, FiltersJSON: filters, ParametersJSON: params, MetricsJSON: []byte(`{}`),
	}
	if err := repo.CreateMarketOpsBacktestRun(ctx, runRecord); err != nil {
		return err
	}
	metrics := runMetrics{RunID: cli.RunID, MaxRecords: cli.MaxRecords, BatchSize: cli.BatchSize, StartedAt: now.Format(time.RFC3339Nano), RecommendationCounts: map[string]int{}}
	logger.Info("marketops backtest run started", "run_id", cli.RunID, "detector_id", cli.DetectorID, "window_start", cli.WindowStart, "window_end", cli.WindowEnd)

	runErr := executeBacktest(ctx, repo, cli, &metrics)
	completedAt := time.Now().UTC()
	metrics.CompletedAt = completedAt.Format(time.RFC3339Nano)
	metricsJSON, marshalErr := json.Marshal(metrics)
	if marshalErr != nil {
		return marshalErr
	}
	if runErr != nil {
		_, _ = repo.FailMarketOpsBacktestRun(context.Background(), cli.RunID, completedAt, runErr.Error(), metricsJSON)
		return runErr
	}
	record, err := repo.CompleteMarketOpsBacktestRun(ctx, cli.RunID, completedAt, metricsJSON)
	if err != nil {
		return err
	}
	out, _ := json.MarshalIndent(map[string]any{"backtest_run": record, "metrics": metrics}, "", "  ")
	fmt.Println(string(out))
	logger.Info("marketops backtest run completed", "run_id", cli.RunID, "signals", metrics.Signals, "graph_proposals", metrics.GraphProposals)
	return nil
}

func executeBacktest(ctx context.Context, repo storage.MarketOpsBacktestRepository, cli cliConfig, metrics *runMetrics) error {
	remaining := cli.MaxRecords
	if remaining <= 0 {
		remaining = 50
	}
	batchSize := cli.BatchSize
	if batchSize <= 0 || batchSize > remaining {
		batchSize = remaining
	}
	offset := 0
	evaluator := dsm.NewPolicyEvaluator(cli.AutoAcceptConfidence)
	for remaining > 0 {
		limit := batchSize
		if remaining < limit {
			limit = remaining
		}
		events, err := repo.ListMarketOpsBacktestNormalizedEvents(ctx, storage.MarketOpsBacktestEventFilter{TenantID: cli.TenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SourceID: cli.SourceID, SourceAdapter: cli.SourceAdapter, Dataset: cli.Dataset, Symbols: cli.Symbols, WindowStart: cli.WindowStart, WindowEnd: cli.WindowEnd, Limit: limit, Offset: offset})
		if err != nil {
			return err
		}
		if len(events) == 0 {
			break
		}
		metrics.Batches++
		metrics.Scanned += len(events)
		signalValues, err := runDetectorBatch(ctx, cli, events)
		if err != nil {
			return err
		}
		signalsBatch := []storage.MarketOpsBacktestSignalRecord{}
		artifactsBatch := []storage.MarketOpsBacktestArtifactRecord{}
		proposalsBatch := []storage.MarketOpsBacktestGraphProposalRecord{}
		policyBatch := []storage.MarketOpsBacktestPolicyResultRecord{}
		for i, value := range signalValues {
			record, err := signals.LedgerRecordFromEventJSON(value, broker.ConsumedMessage{Message: broker.Message{Topic: "marketops.backtest.signal.v1", Key: fmt.Sprintf("%s:%d", cli.RunID, i), Value: value, CorrelationID: cli.RunID}, Partition: -1, Offset: int64(metrics.Signals + i)})
			if err != nil {
				return err
			}
			signalsBatch = append(signalsBatch, storage.MarketOpsBacktestSignalRecord{RunID: cli.RunID, SignalLedgerRecord: record})
			artifacts, err := dsm.ExtractArtifacts(record)
			if err != nil {
				return err
			}
			for _, artifact := range artifacts {
				artifactsBatch = append(artifactsBatch, storage.MarketOpsBacktestArtifactRecord{RunID: cli.RunID, MarketOpsDSMArtifactRecord: artifact})
				proposals, err := dsm.ExtractGraphProposals(artifact)
				if err != nil {
					return err
				}
				for _, proposal := range proposals {
					proposalsBatch = append(proposalsBatch, storage.MarketOpsBacktestGraphProposalRecord{RunID: cli.RunID, MarketOpsDSMGraphProposalRecord: proposal})
					policy, err := evaluator.Evaluate(cli.RunID, proposal)
					if err != nil {
						return err
					}
					policyBatch = append(policyBatch, policy)
					metrics.RecommendationCounts[policy.Recommendation]++
				}
			}
		}
		if err := repo.PersistMarketOpsBacktestBatch(ctx, storage.MarketOpsBacktestRunRecord{RunID: cli.RunID}, signalsBatch, artifactsBatch, proposalsBatch, policyBatch); err != nil {
			return err
		}
		metrics.Signals += len(signalsBatch)
		metrics.Artifacts += len(artifactsBatch)
		metrics.GraphProposals += len(proposalsBatch)
		metrics.PolicyResults += len(policyBatch)
		remaining -= len(events)
		offset += len(events)
	}
	return nil
}

func runDetectorBatch(ctx context.Context, cli cliConfig, events []storage.NormalizedEventLedgerRecord) ([][]byte, error) {
	var stdin bytes.Buffer
	enc := json.NewEncoder(&stdin)
	for _, event := range events {
		mapped, err := normalizedEventForDetector(event)
		if err != nil {
			return nil, err
		}
		if err := enc.Encode(mapped); err != nil {
			return nil, err
		}
	}
	cmd := exec.CommandContext(ctx, cli.PythonBin, "-m", "signalops_workers.backtest_detector", "--detector-id", cli.DetectorID, "--worker-id", "marketops-backtest")
	cmd.Stdin = &stdin
	cmd.Env = pythonEnv(os.Environ())
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("run python backtest detector: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	out := [][]byte{}
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		out = append(out, append([]byte(nil), line...))
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func normalizedEventForDetector(record storage.NormalizedEventLedgerRecord) (map[string]any, error) {
	payload := map[string]any{}
	if len(record.NormalizedPayload) > 0 {
		if err := json.Unmarshal(record.NormalizedPayload, &payload); err != nil {
			return nil, err
		}
	}
	entities := []any{}
	if len(record.EntitiesJSON) > 0 {
		_ = json.Unmarshal(record.EntitiesJSON, &entities)
	}
	evidence := []any{}
	if len(record.EvidenceJSON) > 0 {
		_ = json.Unmarshal(record.EvidenceJSON, &evidence)
	}
	return map[string]any{
		"event_id": record.EventID, "tenant_id": record.TenantID, "source_id": record.SourceID,
		"app_id": record.AppID, "domain": record.Domain, "use_case": record.UseCase, "source_domain": record.Domain,
		"source_adapter": record.SourceAdapter, "ingestion_mode": "scheduled_pull", "dataset": record.Dataset,
		"idempotency_key": record.IdempotencyKey, "correlation_id": record.EventID,
		"observation_time": record.ObservationTime.Format(time.RFC3339Nano), "effective_time": record.ObservationTime.Format(time.RFC3339Nano), "processing_time": time.Now().UTC().Format(time.RFC3339Nano),
		"normalized_payload": payload, "entities": entities, "evidence": evidence,
	}, nil
}

func loadCLIConfig() (cliConfig, error) {
	var start, end, symbols string
	cli := cliConfig{}
	flag.StringVar(&cli.RunID, "run-id", "", "backtest run id")
	flag.StringVar(&cli.TenantID, "tenant-id", "tenant-local", "tenant id")
	flag.StringVar(&cli.SourceID, "source-id", "", "optional source id filter")
	flag.StringVar(&cli.SourceAdapter, "source-adapter", "market_data.massive", "source adapter filter")
	flag.StringVar(&cli.Dataset, "dataset", "equity_eod_prices", "dataset filter")
	flag.StringVar(&cli.DetectorID, "detector-id", "marketops.dsm.taxonomy_v1", "detector id")
	flag.StringVar(&cli.DetectorVersion, "detector-version", "", "detector version label")
	flag.StringVar(&cli.RequestedBy, "requested-by", "operator-local", "requesting operator")
	flag.StringVar(&start, "window-start", "", "inclusive RFC3339 observation start")
	flag.StringVar(&end, "window-end", "", "exclusive RFC3339 observation end")
	flag.StringVar(&symbols, "symbols", "", "comma-separated symbol filter")
	flag.IntVar(&cli.MaxRecords, "max-records", 50, "maximum normalized events to scan")
	flag.IntVar(&cli.BatchSize, "batch-size", 50, "normalized events per detector batch")
	flag.Float64Var(&cli.AutoAcceptConfidence, "auto-accept-confidence", 0.75, "policy auto-accept confidence threshold")
	flag.StringVar(&cli.PythonBin, "python-bin", "python3", "python executable")
	flag.Parse()
	if strings.TrimSpace(cli.RunID) == "" {
		cli.RunID = "bt_marketops_" + randomHex(12)
	}
	if strings.TrimSpace(start) == "" || strings.TrimSpace(end) == "" {
		return cliConfig{}, errors.New("window-start and window-end are required")
	}
	var err error
	cli.WindowStart, err = time.Parse(time.RFC3339Nano, strings.TrimSpace(start))
	if err != nil {
		return cliConfig{}, errors.New("window-start must be RFC3339")
	}
	cli.WindowEnd, err = time.Parse(time.RFC3339Nano, strings.TrimSpace(end))
	if err != nil {
		return cliConfig{}, errors.New("window-end must be RFC3339")
	}
	if !cli.WindowEnd.After(cli.WindowStart) {
		return cliConfig{}, errors.New("window-end must be after window-start")
	}
	for _, symbol := range strings.Split(symbols, ",") {
		if strings.TrimSpace(symbol) != "" {
			cli.Symbols = append(cli.Symbols, strings.ToUpper(strings.TrimSpace(symbol)))
		}
	}
	return cli, nil
}

func pythonEnv(env []string) []string {
	for i, item := range env {
		if strings.HasPrefix(item, "PYTHONPATH=") {
			env[i] = item + string(os.PathListSeparator) + "python"
			return env
		}
	}
	return append(env, "PYTHONPATH=python")
}

func randomHex(n int) string { b := make([]byte, n); _, _ = rand.Read(b); return hex.EncodeToString(b) }

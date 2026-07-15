package algorithms

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const (
	ZScoreAnomalyAlgorithmID = "signalops.algorithms.zscore_anomaly_v1"
	DefaultAlgorithmVersion  = "v1"
	DefaultZScoreFeature     = "daily_return_pct"
)

type Repository interface {
	storage.AlgorithmRepository
	ListMarketOpsBacktestNormalizedEvents(ctx context.Context, filter storage.MarketOpsBacktestEventFilter) ([]storage.NormalizedEventLedgerRecord, error)
}

type Config struct {
	ExecutionRequestID string
	TenantID           string
	AlgorithmID        string
	AlgorithmVersion   string
	RequestedBy        string
	CorrelationID      string
	AppID              string
	Domain             string
	UseCase            string
	SourceID           string
	SourceAdapter      string
	Dataset            string
	Symbols            []string
	WindowStart        time.Time
	WindowEnd          time.Time
	MaxRecords         int
	BatchSize          int
	Feature            string
	ZThreshold         float64
	MinSamples         int
}

type Metrics struct {
	ExecutionRequestID string  `json:"execution_request_id"`
	AlgorithmID        string  `json:"algorithm_id"`
	AlgorithmVersion   string  `json:"algorithm_version"`
	Feature            string  `json:"feature"`
	Scanned            int     `json:"scanned"`
	UsableSamples      int     `json:"usable_samples"`
	Results            int     `json:"results"`
	Mean               float64 `json:"mean"`
	Stddev             float64 `json:"stddev"`
	ZThreshold         float64 `json:"z_threshold"`
	MinSamples         int     `json:"min_samples"`
	StartedAt          string  `json:"started_at"`
	CompletedAt        string  `json:"completed_at,omitempty"`
}

type Result struct {
	ExecutionRequest storage.AlgorithmExecutionRequestRecord
	Metrics          Metrics
}

type sample struct {
	event  storage.NormalizedEventLedgerRecord
	value  float64
	symbol string
}

func Run(ctx context.Context, repo Repository, cfg Config) (Result, error) {
	if repo == nil {
		return Result{}, errors.New("algorithm repository is required")
	}
	cfg = cfg.withDefaults()
	if err := cfg.validate(); err != nil {
		return Result{}, err
	}
	startedAt := time.Now().UTC()
	metrics := Metrics{ExecutionRequestID: cfg.ExecutionRequestID, AlgorithmID: cfg.AlgorithmID, AlgorithmVersion: cfg.AlgorithmVersion, Feature: cfg.Feature, ZThreshold: cfg.ZThreshold, MinSamples: cfg.MinSamples, StartedAt: startedAt.Format(time.RFC3339Nano)}

	request := cfg.executionRequest(storage.AlgorithmExecutionStatusRunning, nil, "")
	if err := repo.UpsertAlgorithmExecutionRequest(ctx, request); err != nil {
		return Result{}, err
	}

	runErr := executeZScore(ctx, repo, cfg, &metrics)
	completedAt := time.Now().UTC()
	metrics.CompletedAt = completedAt.Format(time.RFC3339Nano)
	resultJSON, marshalErr := json.Marshal(metrics)
	if marshalErr != nil {
		return Result{}, marshalErr
	}
	status := storage.AlgorithmExecutionStatusSucceeded
	errorMessage := ""
	if runErr != nil {
		status = storage.AlgorithmExecutionStatusFailed
		errorMessage = runErr.Error()
	}
	request = cfg.executionRequest(status, resultJSON, errorMessage)
	if err := repo.UpsertAlgorithmExecutionRequest(context.Background(), request); err != nil && runErr == nil {
		return Result{}, err
	}
	stored, getErr := repo.GetAlgorithmExecutionRequest(context.Background(), cfg.TenantID, cfg.ExecutionRequestID)
	if getErr != nil && runErr == nil {
		return Result{}, getErr
	}
	if runErr != nil {
		return Result{ExecutionRequest: stored, Metrics: metrics}, runErr
	}
	return Result{ExecutionRequest: stored, Metrics: metrics}, nil
}

func executeZScore(ctx context.Context, repo Repository, cfg Config, metrics *Metrics) error {
	if cfg.AlgorithmID != ZScoreAnomalyAlgorithmID {
		return fmt.Errorf("unsupported algorithm_id %q", cfg.AlgorithmID)
	}
	events, err := scanEvents(ctx, repo, cfg)
	if err != nil {
		return err
	}
	metrics.Scanned = len(events)
	samples := make([]sample, 0, len(events))
	for _, event := range events {
		value, symbol, ok := eventFeatureValue(event, cfg.Feature)
		if !ok {
			continue
		}
		samples = append(samples, sample{event: event, value: value, symbol: symbol})
	}
	metrics.UsableSamples = len(samples)
	if len(samples) < cfg.MinSamples {
		return nil
	}
	mean, stddev := sampleStats(samples)
	metrics.Mean = round(mean, 6)
	metrics.Stddev = round(stddev, 6)
	for _, item := range samples {
		z := 0.0
		if stddev > 0 {
			z = (item.value - mean) / stddev
		}
		record, err := algorithmResult(cfg, item, mean, stddev, z)
		if err != nil {
			return err
		}
		if err := repo.InsertAlgorithmResult(ctx, record); err != nil {
			return err
		}
		metrics.Results++
	}
	return nil
}

func scanEvents(ctx context.Context, repo Repository, cfg Config) ([]storage.NormalizedEventLedgerRecord, error) {
	remaining := cfg.MaxRecords
	offset := 0
	batchSize := cfg.BatchSize
	events := []storage.NormalizedEventLedgerRecord{}
	for remaining > 0 {
		limit := batchSize
		if remaining < limit {
			limit = remaining
		}
		batch, err := repo.ListMarketOpsBacktestNormalizedEvents(ctx, storage.MarketOpsBacktestEventFilter{TenantID: cfg.TenantID, AppID: cfg.AppID, Domain: cfg.Domain, UseCase: cfg.UseCase, SourceID: cfg.SourceID, SourceAdapter: cfg.SourceAdapter, Dataset: cfg.Dataset, Symbols: cfg.Symbols, WindowStart: cfg.WindowStart, WindowEnd: cfg.WindowEnd, Limit: limit, Offset: offset})
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		events = append(events, batch...)
		remaining -= len(batch)
		offset += len(batch)
		if len(batch) < limit {
			break
		}
	}
	return events, nil
}

func eventFeatureValue(event storage.NormalizedEventLedgerRecord, feature string) (float64, string, bool) {
	payload := map[string]any{}
	if len(event.NormalizedPayload) > 0 {
		if err := json.Unmarshal(event.NormalizedPayload, &payload); err != nil {
			return 0, "", false
		}
	}
	symbol := strings.ToUpper(firstString(payload["symbol"], payload["ticker"], payload["underlying_symbol"]))
	if features, ok := payload["features"].(map[string]any); ok {
		if value, ok := number(features[feature]); ok {
			return value, symbol, true
		}
	}
	if value, ok := number(payload[feature]); ok {
		return value, symbol, true
	}
	if feature == DefaultZScoreFeature {
		closePrice, hasClose := number(payload["close"])
		previousClose, hasPrevious := number(payload["previous_close"])
		if hasClose && hasPrevious && previousClose > 0 {
			return round(((closePrice-previousClose)/previousClose)*100, 6), symbol, true
		}
	}
	return 0, symbol, false
}

func sampleStats(samples []sample) (float64, float64) {
	if len(samples) == 0 {
		return 0, 0
	}
	sum := 0.0
	for _, item := range samples {
		sum += item.value
	}
	mean := sum / float64(len(samples))
	variance := 0.0
	for _, item := range samples {
		delta := item.value - mean
		variance += delta * delta
	}
	return mean, math.Sqrt(variance / float64(len(samples)))
}

func algorithmResult(cfg Config, item sample, mean float64, stddev float64, z float64) (storage.AlgorithmResultRecord, error) {
	absZ := math.Abs(z)
	payload, err := json.Marshal(map[string]any{"feature": cfg.Feature, "value": round(item.value, 6), "mean": round(mean, 6), "stddev": round(stddev, 6), "z_score": round(z, 6), "abs_z_score": round(absZ, 6), "z_threshold": cfg.ZThreshold, "symbol": item.symbol, "observation_time": item.event.ObservationTime.Format(time.RFC3339Nano)})
	if err != nil {
		return storage.AlgorithmResultRecord{}, err
	}
	return storage.AlgorithmResultRecord{AlgorithmResultID: stableResultID(cfg, item.event.EventID), TenantID: cfg.TenantID, AlgorithmID: cfg.AlgorithmID, AlgorithmVersion: cfg.AlgorithmVersion, ExecutionRequestID: cfg.ExecutionRequestID, ResultType: "z_score", Score: round(absZ, 6), Confidence: zScoreConfidence(absZ, cfg.ZThreshold), Severity: zScoreSeverity(absZ, cfg.ZThreshold), ResultPayloadJSON: payload, SourceEventIDs: []string{item.event.EventID}, FeatureValueIDs: []string{item.event.EventID + ":" + cfg.Feature}, EvidenceRefs: []string{"normalized_event:" + item.event.EventID}, CorrelationID: cfg.CorrelationID}, nil
}

func (cfg Config) executionRequest(status string, resultJSON []byte, errorMessage string) storage.AlgorithmExecutionRequestRecord {
	configJSON, _ := json.Marshal(map[string]any{"feature": cfg.Feature, "z_threshold": cfg.ZThreshold, "min_samples": cfg.MinSamples, "max_records": cfg.MaxRecords, "batch_size": cfg.BatchSize, "symbols": cfg.Symbols, "window_start": cfg.WindowStart.Format(time.RFC3339Nano), "window_end": cfg.WindowEnd.Format(time.RFC3339Nano), "app_id": cfg.AppID, "domain": cfg.Domain, "use_case": cfg.UseCase, "source_id": cfg.SourceID, "source_adapter": cfg.SourceAdapter, "dataset": cfg.Dataset})
	if len(resultJSON) == 0 {
		resultJSON = []byte(`{}`)
	}
	return storage.AlgorithmExecutionRequestRecord{ExecutionRequestID: cfg.ExecutionRequestID, TenantID: cfg.TenantID, AlgorithmID: cfg.AlgorithmID, AlgorithmVersion: cfg.AlgorithmVersion, FeatureRefs: []string{cfg.Feature}, EntityRefs: entityRefs(cfg.Symbols), WindowRef: cfg.WindowStart.Format(time.RFC3339Nano) + "/" + cfg.WindowEnd.Format(time.RFC3339Nano), ConfigJSON: configJSON, CorrelationID: cfg.CorrelationID, Status: status, RequestedBy: cfg.RequestedBy, ResultJSON: resultJSON, ErrorMessage: errorMessage}
}

func (cfg Config) withDefaults() Config {
	if strings.TrimSpace(cfg.ExecutionRequestID) == "" {
		cfg.ExecutionRequestID = "algexec_" + stableHash(time.Now().UTC().Format(time.RFC3339Nano))[:24]
	}
	if strings.TrimSpace(cfg.AlgorithmID) == "" {
		cfg.AlgorithmID = ZScoreAnomalyAlgorithmID
	}
	if strings.TrimSpace(cfg.AlgorithmVersion) == "" {
		cfg.AlgorithmVersion = DefaultAlgorithmVersion
	}
	if strings.TrimSpace(cfg.RequestedBy) == "" {
		cfg.RequestedBy = "operator-local"
	}
	if strings.TrimSpace(cfg.CorrelationID) == "" {
		cfg.CorrelationID = cfg.ExecutionRequestID
	}
	if strings.TrimSpace(cfg.AppID) == "" {
		cfg.AppID = "marketops"
	}
	if strings.TrimSpace(cfg.Domain) == "" {
		cfg.Domain = "market_data"
	}
	if strings.TrimSpace(cfg.UseCase) == "" {
		cfg.UseCase = "daily_market_surveillance"
	}
	if strings.TrimSpace(cfg.SourceAdapter) == "" {
		cfg.SourceAdapter = "market_data.massive"
	}
	if strings.TrimSpace(cfg.Dataset) == "" {
		cfg.Dataset = "equity_eod_prices"
	}
	if strings.TrimSpace(cfg.Feature) == "" {
		cfg.Feature = DefaultZScoreFeature
	}
	if cfg.ZThreshold <= 0 {
		cfg.ZThreshold = 3.0
	}
	if cfg.MinSamples <= 0 {
		cfg.MinSamples = 3
	}
	if cfg.MaxRecords <= 0 {
		cfg.MaxRecords = 50
	}
	if cfg.BatchSize <= 0 || cfg.BatchSize > cfg.MaxRecords {
		cfg.BatchSize = cfg.MaxRecords
	}
	for i, symbol := range cfg.Symbols {
		cfg.Symbols[i] = strings.ToUpper(strings.TrimSpace(symbol))
	}
	return cfg
}

func (cfg Config) validate() error {
	if strings.TrimSpace(cfg.ExecutionRequestID) == "" {
		return errors.New("execution_request_id is required")
	}
	if strings.TrimSpace(cfg.TenantID) == "" {
		return errors.New("tenant_id is required")
	}
	if cfg.WindowStart.IsZero() || cfg.WindowEnd.IsZero() {
		return errors.New("window_start and window_end are required")
	}
	if !cfg.WindowEnd.After(cfg.WindowStart) {
		return errors.New("window_end must be after window_start")
	}
	if cfg.MaxRecords <= 0 || cfg.MaxRecords > 1000 {
		return errors.New("max_records must be between 1 and 1000")
	}
	if cfg.MinSamples < 2 {
		return errors.New("min_samples must be at least 2")
	}
	return nil
}

func stableResultID(cfg Config, eventID string) string {
	return "algres_" + stableHash(strings.Join([]string{cfg.TenantID, cfg.AlgorithmID, cfg.AlgorithmVersion, cfg.ExecutionRequestID, eventID, cfg.Feature}, "|"))[:32]
}

func stableHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func zScoreConfidence(absZ float64, threshold float64) float64 {
	if threshold <= 0 || absZ == 0 {
		return 0
	}
	return round(math.Min(0.99, absZ/(threshold*1.5)), 6)
}

func zScoreSeverity(absZ float64, threshold float64) string {
	if threshold <= 0 {
		threshold = 3
	}
	switch {
	case absZ >= threshold*1.5:
		return "critical"
	case absZ >= threshold:
		return "high"
	case absZ >= threshold*0.75:
		return "medium"
	case absZ >= threshold*0.5:
		return "low"
	default:
		return "info"
	}
}

func entityRefs(symbols []string) []string {
	refs := []string{}
	seen := map[string]struct{}{}
	for _, symbol := range symbols {
		symbol = strings.ToUpper(strings.TrimSpace(symbol))
		if symbol == "" {
			continue
		}
		ref := "ticker:" + symbol
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		refs = append(refs, ref)
	}
	return refs
}

func firstString(values ...any) string {
	for _, value := range values {
		if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func number(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		v, err := typed.Float64()
		return v, err == nil
	default:
		return 0, false
	}
}

func round(value float64, places int) float64 {
	factor := math.Pow10(places)
	return math.Round(value*factor) / factor
}

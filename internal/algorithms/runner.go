package algorithms

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const (
	ZScoreAnomalyAlgorithmID       = "signalops.algorithms.zscore_anomaly_v1"
	RiverAnomalyAlgorithmID        = "signalops.algorithms.river_anomaly_v1"
	RupturesChangePointAlgorithmID = "signalops.algorithms.ruptures_change_point_v1"
	StatsmodelsForecastAlgorithmID = "signalops.algorithms.statsmodels_forecast_v1"
	SklearnClassifierAlgorithmID   = "signalops.algorithms.sklearn_classifier_v1"
	SklearnIsolationForestID       = "signalops.algorithms.sklearn_isolation_forest_v1"
	DefaultAlgorithmVersion        = "v1"
	DefaultZScoreFeature           = "daily_return_pct"
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
	index  int
}

type scoredSample struct {
	sample     sample
	resultType string
	score      float64
	confidence float64
	severity   string
	payload    map[string]any
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

	runErr := executeAlgorithm(ctx, repo, cfg, &metrics)
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

func executeAlgorithm(ctx context.Context, repo Repository, cfg Config, metrics *Metrics) error {
	if !supportedAlgorithm(cfg.AlgorithmID) {
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
		samples = append(samples, sample{event: event, value: value, symbol: symbol, index: len(samples)})
	}
	metrics.UsableSamples = len(samples)
	if len(samples) < cfg.MinSamples {
		return nil
	}
	mean, stddev := sampleStats(samples)
	metrics.Mean = round(mean, 6)
	metrics.Stddev = round(stddev, 6)

	var scored []scoredSample
	switch cfg.AlgorithmID {
	case ZScoreAnomalyAlgorithmID:
		scored = scoreZScore(cfg, samples, mean, stddev)
	case RiverAnomalyAlgorithmID:
		scored = scoreRiverAnomaly(cfg, samples)
	case RupturesChangePointAlgorithmID:
		scored = scoreChangePoints(cfg, samples, stddev)
	case StatsmodelsForecastAlgorithmID:
		scored = scoreForecastResiduals(cfg, samples)
	case SklearnClassifierAlgorithmID:
		scored = scoreClassifier(cfg, samples, mean, stddev)
	case SklearnIsolationForestID:
		scored = scoreIsolation(cfg, samples)
	default:
		return fmt.Errorf("unsupported algorithm_id %q", cfg.AlgorithmID)
	}
	for _, item := range scored {
		record, err := algorithmResult(cfg, item)
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

func scoreZScore(cfg Config, samples []sample, mean float64, stddev float64) []scoredSample {
	out := make([]scoredSample, 0, len(samples))
	for _, item := range samples {
		z := 0.0
		if stddev > 0 {
			z = (item.value - mean) / stddev
		}
		absZ := math.Abs(z)
		out = append(out, scoredSample{sample: item, resultType: "z_score", score: round(absZ, 6), confidence: zScoreConfidence(absZ, cfg.ZThreshold), severity: zScoreSeverity(absZ, cfg.ZThreshold), payload: basePayload(cfg, item, map[string]any{"mean": round(mean, 6), "stddev": round(stddev, 6), "z_score": round(z, 6), "abs_z_score": round(absZ, 6), "z_threshold": cfg.ZThreshold})})
	}
	return out
}

func scoreRiverAnomaly(cfg Config, samples []sample) []scoredSample {
	out := make([]scoredSample, 0, len(samples))
	seen := []sample{}
	for _, item := range samples {
		priorMean, priorStddev := sampleStats(seen)
		score := 0.0
		if len(seen) >= cfg.MinSamples-1 && priorStddev > 0 {
			score = math.Abs((item.value - priorMean) / priorStddev)
		}
		out = append(out, scoredSample{sample: item, resultType: "online_anomaly_score", score: round(score, 6), confidence: zScoreConfidence(score, cfg.ZThreshold), severity: zScoreSeverity(score, cfg.ZThreshold), payload: basePayload(cfg, item, map[string]any{"online_mean": round(priorMean, 6), "online_stddev": round(priorStddev, 6), "online_z_score": round(score, 6), "training_samples_before_event": len(seen), "z_threshold": cfg.ZThreshold})})
		seen = append(seen, item)
	}
	return out
}

func scoreChangePoints(cfg Config, samples []sample, stddev float64) []scoredSample {
	out := make([]scoredSample, 0, len(samples)-1)
	if len(samples) < 2 {
		return out
	}
	for i := 1; i < len(samples); i++ {
		current := samples[i]
		previous := samples[i-1]
		delta := current.value - previous.value
		score := math.Abs(delta)
		if stddev > 0 {
			score = score / stddev
		}
		out = append(out, scoredSample{sample: current, resultType: "change_point_score", score: round(score, 6), confidence: zScoreConfidence(score, cfg.ZThreshold), severity: zScoreSeverity(score, cfg.ZThreshold), payload: basePayload(cfg, current, map[string]any{"previous_event_id": previous.event.EventID, "previous_value": round(previous.value, 6), "delta": round(delta, 6), "normalized_delta": round(score, 6), "z_threshold": cfg.ZThreshold})})
	}
	return out
}

func scoreForecastResiduals(cfg Config, samples []sample) []scoredSample {
	out := make([]scoredSample, 0, len(samples))
	intercept, slope := linearFit(samples)
	residualSamples := make([]sample, 0, len(samples))
	for _, item := range samples {
		predicted := intercept + slope*float64(item.index)
		residualSamples = append(residualSamples, sample{value: item.value - predicted})
	}
	_, residualStddev := sampleStats(residualSamples)
	for _, item := range samples {
		predicted := intercept + slope*float64(item.index)
		residual := item.value - predicted
		score := math.Abs(residual)
		if residualStddev > 0 {
			score = score / residualStddev
		}
		out = append(out, scoredSample{sample: item, resultType: "forecast_residual", score: round(score, 6), confidence: zScoreConfidence(score, cfg.ZThreshold), severity: zScoreSeverity(score, cfg.ZThreshold), payload: basePayload(cfg, item, map[string]any{"predicted_value": round(predicted, 6), "residual": round(residual, 6), "residual_stddev": round(residualStddev, 6), "trend_intercept": round(intercept, 6), "trend_slope": round(slope, 6), "z_threshold": cfg.ZThreshold})})
	}
	return out
}

func scoreClassifier(cfg Config, samples []sample, mean float64, stddev float64) []scoredSample {
	out := make([]scoredSample, 0, len(samples))
	for _, item := range samples {
		z := 0.0
		if stddev > 0 {
			z = (item.value - mean) / stddev
		}
		absZ := math.Abs(z)
		label := "baseline"
		if absZ >= cfg.ZThreshold {
			label = "candidate_anomaly"
		}
		out = append(out, scoredSample{sample: item, resultType: "classifier_label", score: round(absZ, 6), confidence: zScoreConfidence(absZ, cfg.ZThreshold), severity: zScoreSeverity(absZ, cfg.ZThreshold), payload: basePayload(cfg, item, map[string]any{"classification_label": label, "classification_reason": "deterministic v0 threshold classifier until trained model artifact is introduced", "mean": round(mean, 6), "stddev": round(stddev, 6), "z_score": round(z, 6), "z_threshold": cfg.ZThreshold})})
	}
	return out
}

func scoreIsolation(cfg Config, samples []sample) []scoredSample {
	out := make([]scoredSample, 0, len(samples))
	median := sampleMedian(samples)
	deviations := make([]sample, 0, len(samples))
	for _, item := range samples {
		deviations = append(deviations, sample{value: math.Abs(item.value - median)})
	}
	mad := sampleMedian(deviations)
	for _, item := range samples {
		score := math.Abs(item.value - median)
		if mad > 0 {
			score = score / mad
		}
		out = append(out, scoredSample{sample: item, resultType: "isolation_score", score: round(score, 6), confidence: zScoreConfidence(score, cfg.ZThreshold), severity: zScoreSeverity(score, cfg.ZThreshold), payload: basePayload(cfg, item, map[string]any{"median": round(median, 6), "median_absolute_deviation": round(mad, 6), "isolation_score": round(score, 6), "z_threshold": cfg.ZThreshold})})
	}
	return out
}

func supportedAlgorithm(algorithmID string) bool {
	switch algorithmID {
	case ZScoreAnomalyAlgorithmID, RiverAnomalyAlgorithmID, RupturesChangePointAlgorithmID, StatsmodelsForecastAlgorithmID, SklearnClassifierAlgorithmID, SklearnIsolationForestID:
		return true
	default:
		return false
	}
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

func sampleMedian(samples []sample) float64 {
	if len(samples) == 0 {
		return 0
	}
	values := make([]float64, 0, len(samples))
	for _, item := range samples {
		values = append(values, item.value)
	}
	sort.Float64s(values)
	mid := len(values) / 2
	if len(values)%2 == 1 {
		return values[mid]
	}
	return (values[mid-1] + values[mid]) / 2
}

func linearFit(samples []sample) (float64, float64) {
	if len(samples) == 0 {
		return 0, 0
	}
	n := float64(len(samples))
	sumX, sumY, sumXY, sumXX := 0.0, 0.0, 0.0, 0.0
	for _, item := range samples {
		x := float64(item.index)
		sumX += x
		sumY += item.value
		sumXY += x * item.value
		sumXX += x * x
	}
	denominator := n*sumXX - sumX*sumX
	if denominator == 0 {
		return sumY / n, 0
	}
	slope := (n*sumXY - sumX*sumY) / denominator
	intercept := (sumY - slope*sumX) / n
	return intercept, slope
}

func algorithmResult(cfg Config, item scoredSample) (storage.AlgorithmResultRecord, error) {
	payload, err := json.Marshal(item.payload)
	if err != nil {
		return storage.AlgorithmResultRecord{}, err
	}
	return storage.AlgorithmResultRecord{AlgorithmResultID: stableResultID(cfg, item.sample.event.EventID), TenantID: cfg.TenantID, AlgorithmID: cfg.AlgorithmID, AlgorithmVersion: cfg.AlgorithmVersion, ExecutionRequestID: cfg.ExecutionRequestID, ResultType: item.resultType, Score: item.score, Confidence: item.confidence, Severity: item.severity, ResultPayloadJSON: payload, SourceEventIDs: []string{item.sample.event.EventID}, FeatureValueIDs: []string{item.sample.event.EventID + ":" + cfg.Feature}, EvidenceRefs: []string{"normalized_event:" + item.sample.event.EventID}, CorrelationID: cfg.CorrelationID}, nil
}

func basePayload(cfg Config, item sample, extra map[string]any) map[string]any {
	payload := map[string]any{"algorithm_id": cfg.AlgorithmID, "feature": cfg.Feature, "value": round(item.value, 6), "symbol": item.symbol, "observation_time": item.event.ObservationTime.Format(time.RFC3339Nano)}
	for key, value := range extra {
		payload[key] = value
	}
	return payload
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

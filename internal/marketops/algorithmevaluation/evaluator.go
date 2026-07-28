package algorithmevaluation

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

	"github.com/lukebabs/signalops/internal/platformregistry"
	"github.com/lukebabs/signalops/internal/storage"
)

const CalculationVersion = "marketops.algorithm_evaluation.v1"

var DefaultHorizons = []int{1, 5, 10, 20}

var DefaultAlgorithmIDs = []string{
	"signalops.algorithms.zscore_anomaly_v1",
	"signalops.algorithms.river_anomaly_v1",
	"signalops.algorithms.ruptures_change_point_v1",
	"signalops.algorithms.statsmodels_forecast_v1",
	"signalops.algorithms.sklearn_classifier_v1",
	"signalops.algorithms.sklearn_isolation_forest_v1",
	"signalops.algorithms.risk_reward_temporal_v1",
}

type Repository interface {
	storage.MarketOpsAlgorithmEvaluationRepository
	platformregistry.DefinitionLister
	ListMarketOpsBacktestNormalizedEvents(context.Context, storage.MarketOpsBacktestEventFilter) ([]storage.NormalizedEventLedgerRecord, error)
	ListMarketOpsFeatureObservations(context.Context, storage.MarketOpsFeatureObservationFilter) ([]storage.MarketOpsFeatureObservationRecord, error)
}

type Config struct {
	RunID               string
	TenantID            string
	UniverseGroup       string
	Symbols             []string
	AlgorithmIDs        []string
	Modes               []string
	WindowStart         time.Time
	WindowEnd           time.Time
	AsOf                time.Time
	Feature             string
	LookbackSessions    int
	MinSamples          int
	Threshold           float64
	RequestedBy         string
	DryRun              bool
	RegistryEnforcement bool
}

type Metrics struct {
	RunID             string                    `json:"run_id"`
	EvaluationResults int                       `json:"evaluation_results"`
	Outcomes          int                       `json:"outcomes"`
	Matured           int                       `json:"matured"`
	Pending           int                       `json:"pending"`
	MissingPrice      int                       `json:"missing_price"`
	ProfileMetrics    map[string]ProfileMetrics `json:"profile_metrics"`
	Coverage          map[string]int            `json:"coverage"`
	StartedAt         string                    `json:"started_at"`
	CompletedAt       string                    `json:"completed_at"`
}

type ProfileMetrics struct {
	Matured               int      `json:"matured"`
	DirectionalSamples    int      `json:"directional_samples"`
	DirectionalHits       int      `json:"directional_hits"`
	DirectionalHitRate    *float64 `json:"directional_hit_rate,omitempty"`
	DirectionalHitCILower *float64 `json:"directional_hit_ci_lower,omitempty"`
	DirectionalHitCIUpper *float64 `json:"directional_hit_ci_upper,omitempty"`
	MeanForwardReturn     *float64 `json:"mean_forward_return,omitempty"`
	MeanReturnCILower     *float64 `json:"mean_return_ci_lower,omitempty"`
	MeanReturnCIUpper     *float64 `json:"mean_return_ci_upper,omitempty"`
	MeanAbsoluteReturn    *float64 `json:"mean_absolute_forward_return,omitempty"`
	returns               []float64
	absoluteReturns       []float64
}

type Result struct {
	Run     storage.MarketOpsAlgorithmEvaluationRunRecord
	Metrics Metrics
}

type sample struct {
	eventID  string
	symbol   string
	session  time.Time
	value    float64
	close    float64
	high     float64
	low      float64
	metadata map[string]any
}

type evaluated struct {
	record storage.MarketOpsAlgorithmEvaluationResultRecord
	sample sample
}

func Run(ctx context.Context, repo Repository, cfg Config) (Result, error) {
	if repo == nil {
		return Result{}, errors.New("algorithm evaluation repository is required")
	}
	cfg = defaults(cfg)
	if err := validate(cfg); err != nil {
		return Result{}, err
	}
	now := time.Now().UTC()
	metrics := Metrics{RunID: cfg.RunID, ProfileMetrics: map[string]ProfileMetrics{}, Coverage: map[string]int{}, StartedAt: now.Format(time.RFC3339Nano)}
	run := storage.MarketOpsAlgorithmEvaluationRunRecord{
		RunID: cfg.RunID, TenantID: cfg.TenantID, AppID: "marketops", UniverseGroup: cfg.UniverseGroup,
		AlgorithmIDs: cfg.AlgorithmIDs, Modes: cfg.Modes, WindowStart: cfg.WindowStart, WindowEnd: cfg.WindowEnd,
		AsOfDate: cfg.AsOf, Status: storage.MarketOpsAlgorithmEvaluationStatusRunning,
		ParametersJSON: mustJSON(map[string]any{"feature": cfg.Feature, "lookback_sessions": cfg.LookbackSessions, "min_samples": cfg.MinSamples, "threshold": cfg.Threshold, "dry_run": cfg.DryRun, "registry_enforcement": cfg.RegistryEnforcement}),
		CoverageJSON:   []byte(`{}`), MetricsJSON: []byte(`{}`), RequestedBy: cfg.RequestedBy, StartedAt: now,
	}
	if !cfg.DryRun {
		if err := repo.UpsertMarketOpsAlgorithmEvaluationRun(ctx, run); err != nil {
			return Result{}, err
		}
	}

	events, err := repo.ListMarketOpsBacktestNormalizedEvents(ctx, storage.MarketOpsBacktestEventFilter{
		TenantID: cfg.TenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", Dataset: "equity_eod_prices",
		Symbols: cfg.Symbols, WindowStart: cfg.WindowStart.AddDate(0, 0, -400), WindowEnd: cfg.AsOf.AddDate(0, 0, 1), Limit: 50000,
	})
	if err != nil {
		return complete(ctx, repo, cfg, run, metrics, err)
	}
	if cfg.RegistryEnforcement {
		validator := platformregistry.MassiveNormalizedEventDefinitionValidator{Lister: repo}
		for _, event := range events {
			if err := validator.ValidateNormalizedEvent(ctx, event); err != nil {
				return complete(ctx, repo, cfg, run, metrics, fmt.Errorf("validate normalized event %s platform definitions: %w", event.EventID, err))
			}
		}
	}
	bySymbol := eventSamples(events, cfg.Feature)
	for symbol, values := range bySymbol {
		metrics.Coverage[symbol] = len(values)
	}
	coverage := mustJSON(metrics.Coverage)
	run.CoverageJSON = coverage

	evaluations := []evaluated{}
	for _, algorithmID := range cfg.AlgorithmIDs {
		if algorithmID == "signalops.algorithms.risk_reward_temporal_v1" {
			special, err := evaluateRiskReward(ctx, repo, cfg, algorithmID)
			if err != nil {
				return complete(ctx, repo, cfg, run, metrics, err)
			}
			evaluations = append(evaluations, special...)
			continue
		}
		for symbol, values := range bySymbol {
			_ = symbol
			for _, mode := range cfg.Modes {
				evaluations = append(evaluations, evaluateScalar(cfg, algorithmID, mode, values)...)
			}
		}
	}
	for _, item := range evaluations {
		if !cfg.DryRun {
			if err := repo.InsertMarketOpsAlgorithmEvaluationResult(ctx, item.record); err != nil {
				return complete(ctx, repo, cfg, run, metrics, err)
			}
		}
		metrics.EvaluationResults++
		outcomes := buildOutcomes(cfg, item, bySymbol[item.sample.symbol])
		for _, outcome := range outcomes {
			if !cfg.DryRun {
				if err := repo.UpsertMarketOpsAlgorithmEvaluationOutcome(ctx, outcome); err != nil {
					return complete(ctx, repo, cfg, run, metrics, err)
				}
			}
			metrics.Outcomes++
			switch outcome.OutcomeStatus {
			case storage.MarketOpsOutcomeMatured:
				metrics.Matured++
			case storage.MarketOpsOutcomePending:
				metrics.Pending++
			case storage.MarketOpsOutcomeMissingPrice:
				metrics.MissingPrice++
			}
			metrics.ProfileMetrics[item.record.EvaluationProfile] = aggregate(metrics.ProfileMetrics[item.record.EvaluationProfile], item.record, outcome)
		}
	}
	metrics.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
	for profile, values := range metrics.ProfileMetrics {
		metrics.ProfileMetrics[profile] = finalize(values)
	}
	run.CoverageJSON = coverage
	run.MetricsJSON = mustJSON(metrics)
	run.Status = storage.MarketOpsAlgorithmEvaluationStatusSucceeded
	if metrics.Pending > 0 || metrics.MissingPrice > 0 {
		run.Status = storage.MarketOpsAlgorithmEvaluationStatusPartial
	}
	completed := time.Now().UTC()
	run.CompletedAt = &completed
	if !cfg.DryRun {
		if err := repo.UpsertMarketOpsAlgorithmEvaluationRun(ctx, run); err != nil {
			return Result{}, err
		}
	}
	return Result{Run: run, Metrics: metrics}, nil
}

func complete(ctx context.Context, repo Repository, cfg Config, run storage.MarketOpsAlgorithmEvaluationRunRecord, metrics Metrics, runErr error) (Result, error) {
	run.Status = storage.MarketOpsAlgorithmEvaluationStatusFailed
	run.ErrorMessage = runErr.Error()
	run.MetricsJSON = mustJSON(metrics)
	completed := time.Now().UTC()
	run.CompletedAt = &completed
	if !cfg.DryRun {
		_ = repo.UpsertMarketOpsAlgorithmEvaluationRun(ctx, run)
	}
	return Result{Run: run, Metrics: metrics}, runErr
}

func defaults(cfg Config) Config {
	if cfg.UniverseGroup == "" {
		cfg.UniverseGroup = "top50_megacap"
	}
	if len(cfg.AlgorithmIDs) == 0 {
		cfg.AlgorithmIDs = append([]string(nil), DefaultAlgorithmIDs...)
	}
	if len(cfg.Modes) == 0 {
		cfg.Modes = []string{storage.MarketOpsAlgorithmEvaluationModeWalkForward}
	}
	if cfg.Feature == "" {
		cfg.Feature = "daily_return_pct"
	}
	if cfg.LookbackSessions <= 0 {
		cfg.LookbackSessions = 60
	}
	if cfg.MinSamples <= 0 {
		cfg.MinSamples = 20
	}
	if cfg.Threshold <= 0 {
		cfg.Threshold = 0.02
	}
	if cfg.RequestedBy == "" {
		cfg.RequestedBy = "operator-local"
	}
	if cfg.RunID == "" {
		cfg.RunID = "algeval_" + stableID("", fmt.Sprint(time.Now().UTC().UnixNano()))[:16]
	}
	cfg.Symbols = uniqueUpper(cfg.Symbols)
	cfg.AlgorithmIDs = uniqueStrings(cfg.AlgorithmIDs)
	cfg.Modes = uniqueStrings(cfg.Modes)
	return cfg
}

func validate(cfg Config) error {
	if cfg.TenantID == "" || cfg.RunID == "" || cfg.UniverseGroup == "" {
		return errors.New("tenant id, run id, and universe group are required")
	}
	if len(cfg.Symbols) == 0 {
		return errors.New("at least one symbol is required")
	}
	if cfg.WindowStart.IsZero() || cfg.WindowEnd.IsZero() || cfg.AsOf.IsZero() || !cfg.WindowEnd.After(cfg.WindowStart) || cfg.AsOf.Before(cfg.WindowStart) {
		return errors.New("evaluation window and as-of are invalid")
	}
	if cfg.LookbackSessions < 20 || cfg.LookbackSessions > 400 || cfg.MinSamples < 2 || cfg.MinSamples > cfg.LookbackSessions {
		return errors.New("lookback or minimum samples are invalid")
	}
	if cfg.Threshold <= 0 || cfg.Threshold >= 1 {
		return errors.New("threshold must be between zero and one")
	}
	for _, mode := range cfg.Modes {
		if mode != storage.MarketOpsAlgorithmEvaluationModeRetrospective && mode != storage.MarketOpsAlgorithmEvaluationModeWalkForward {
			return fmt.Errorf("unsupported evaluation mode %q", mode)
		}
	}
	for _, algorithmID := range cfg.AlgorithmIDs {
		if !contains(DefaultAlgorithmIDs, algorithmID) {
			return fmt.Errorf("unsupported algorithm id %q", algorithmID)
		}
	}
	return nil
}

func eventSamples(events []storage.NormalizedEventLedgerRecord, feature string) map[string][]sample {
	out := map[string][]sample{}
	for _, event := range events {
		var payload map[string]any
		if json.Unmarshal(event.NormalizedPayload, &payload) != nil {
			continue
		}
		symbol := strings.ToUpper(stringValue(payload["symbol"]))
		if symbol == "" {
			symbol = strings.ToUpper(stringValue(payload["ticker"]))
		}
		if symbol == "" {
			continue
		}
		value, ok := payloadFeature(payload, feature)
		if !ok {
			value = math.NaN()
		}
		close, high, low := priceValues(event, value)
		metadata := map[string]any{}
		_ = json.Unmarshal(event.MetadataJSON, &metadata)
		out[symbol] = append(out[symbol], sample{eventID: event.EventID, symbol: symbol, session: day(event.ObservationTime), value: value, close: close, high: high, low: low, metadata: metadata})
	}
	for symbol, values := range out {
		sort.Slice(values, func(i, j int) bool { return values[i].session.Before(values[j].session) })
		usable := make([]sample, 0, len(values))
		for index := range values {
			if math.IsNaN(values[index].value) && feature == "daily_return_pct" && index > 0 && values[index-1].close > 0 && values[index].close > 0 {
				values[index].value = (values[index].close/values[index-1].close - 1) * 100
			}
			if !math.IsNaN(values[index].value) {
				usable = append(usable, values[index])
			}
		}
		out[symbol] = usable
	}
	return out
}

func payloadFeature(payload map[string]any, feature string) (float64, bool) {
	raw := payload[feature]
	if features, ok := payload["features"].(map[string]any); ok {
		raw = features[feature]
	}
	return number(raw)
}

func evaluateScalar(cfg Config, algorithmID, mode string, values []sample) []evaluated {
	profile, resultType := scalarProfile(algorithmID)
	out := []evaluated{}
	for index, current := range values {
		if !current.session.Before(cfg.WindowEnd) || current.session.After(cfg.AsOf) {
			continue
		}
		source := values
		if mode == storage.MarketOpsAlgorithmEvaluationModeWalkForward {
			source = values[max(0, index-cfg.LookbackSessions):index]
		} else {
			source = values[max(0, index-cfg.LookbackSessions):min(len(values), index+cfg.LookbackSessions+1)]
		}
		if len(source) < cfg.MinSamples {
			continue
		}
		mean, deviation := sampleStats(source)
		score, signed, prediction := scoreScalar(algorithmID, current, source, mean, deviation)
		severity := severityFor(score)
		direction := "non_directional"
		if algorithmID == "signalops.algorithms.statsmodels_forecast_v1" {
			if signed > 0 {
				direction = "upside"
			} else if signed < 0 {
				direction = "downside"
			}
		}
		payload := map[string]any{"feature": cfg.Feature, "mode": mode, "lookback_samples": len(source), "mean": round(mean), "stddev": round(deviation), "value": round(current.value), "prediction": prediction}
		if profile == storage.MarketOpsAlgorithmEvaluationProfileClassification {
			payload["classification_label"] = map[bool]string{true: "candidate_anomaly", false: "baseline"}[score >= 3]
		}
		provenance := map[string]any{"platform_definition_versions": current.metadata["platform_definition_versions"], "quality": current.metadata["quality"]}
		record := storage.MarketOpsAlgorithmEvaluationResultRecord{
			EvaluationResultID: stableID("aevalres_", cfg.RunID, algorithmID, mode, current.eventID), RunID: cfg.RunID, TenantID: cfg.TenantID,
			AlgorithmID: algorithmID, AlgorithmVersion: "v1", EvaluationMode: mode, EvaluationProfile: profile, ResultType: resultType,
			Symbol: current.symbol, ObservationSessionDate: current.session, Score: round(score), Confidence: confidence(score), Severity: severity,
			Direction: direction, ResultPayloadJSON: mustJSON(payload), InputProvenanceJSON: mustJSON(provenance), SourceEventIDs: []string{current.eventID},
			FeatureValueIDs: []string{current.eventID + ":" + cfg.Feature}, DeterministicKey: stableID("key_", cfg.RunID, algorithmID, mode, current.eventID),
		}
		out = append(out, evaluated{record: record, sample: current})
	}
	return out
}

func scalarProfile(algorithmID string) (string, string) {
	switch algorithmID {
	case "signalops.algorithms.statsmodels_forecast_v1":
		return storage.MarketOpsAlgorithmEvaluationProfileForecast, "walk_forward_forecast"
	case "signalops.algorithms.sklearn_classifier_v1":
		return storage.MarketOpsAlgorithmEvaluationProfileClassification, "classifier_label"
	case "signalops.algorithms.zscore_anomaly_v1":
		return storage.MarketOpsAlgorithmEvaluationProfileEventStudy, "z_score"
	case "signalops.algorithms.river_anomaly_v1":
		return storage.MarketOpsAlgorithmEvaluationProfileEventStudy, "online_anomaly_score"
	case "signalops.algorithms.ruptures_change_point_v1":
		return storage.MarketOpsAlgorithmEvaluationProfileEventStudy, "change_point_score"
	default:
		return storage.MarketOpsAlgorithmEvaluationProfileEventStudy, "isolation_score"
	}
}

func scoreScalar(algorithmID string, current sample, source []sample, mean, deviation float64) (float64, float64, any) {
	z := 0.0
	if deviation > 0 {
		z = (current.value - mean) / deviation
	}
	switch algorithmID {
	case "signalops.algorithms.ruptures_change_point_v1":
		previous := source[len(source)-1].value
		score := math.Abs(current.value - previous)
		if deviation > 0 {
			score /= deviation
		}
		return score, current.value - previous, nil
	case "signalops.algorithms.statsmodels_forecast_v1":
		slope := 0.0
		if len(source) > 1 {
			slope = (source[len(source)-1].value - source[0].value) / float64(len(source)-1)
		}
		predicted := source[len(source)-1].value + slope
		residual := current.value - predicted
		score := math.Abs(residual)
		if deviation > 0 {
			score /= deviation
		}
		return score, residual, map[string]any{"predicted_next_value": round(predicted), "residual": round(residual)}
	case "signalops.algorithms.sklearn_isolation_forest_v1":
		median := sampleMedian(source)
		deviations := make([]float64, len(source))
		for i := range source {
			deviations[i] = math.Abs(source[i].value - median)
		}
		mad := medianFloat(deviations)
		score := math.Abs(current.value - median)
		if mad > 0 {
			score /= mad
		}
		return score, 0, nil
	default:
		return math.Abs(z), z, nil
	}
}

func validateFeatureObservationProvenance(observations []storage.MarketOpsFeatureObservationRecord) error {
	for _, observation := range observations {
		if observation.NumericValue == nil || (observation.QualityState != storage.MarketOpsQualityUsable && observation.QualityState != storage.MarketOpsQualityUsableWithWarning) {
			continue
		}
		var details map[string]any
		if err := json.Unmarshal(observation.QualityDetailsJSON, &details); err != nil {
			return fmt.Errorf("decode feature observation %s provenance: %w", observation.FeatureObservationID, err)
		}
		provenance, ok := details["input_provenance"].([]any)
		if !ok || len(provenance) == 0 {
			return fmt.Errorf("feature observation %s input provenance is required", observation.FeatureObservationID)
		}
	}
	return nil
}

func evaluateRiskReward(ctx context.Context, repo Repository, cfg Config, algorithmID string) ([]evaluated, error) {
	out := []evaluated{}
	required := []string{"range_position_252d", "rsi_14", "return_5d", "volume_ratio_10d", "distance_sma_50_pct", "distance_sma_200_pct", "sma_50_slope_20d_pct", "atr_14_pct"}
	for _, symbol := range cfg.Symbols {
		observations, err := repo.ListMarketOpsFeatureObservations(ctx, storage.MarketOpsFeatureObservationFilter{TenantID: cfg.TenantID, AppID: "marketops", Symbol: symbol, SessionStart: cfg.WindowStart, SessionEnd: cfg.WindowEnd.AddDate(0, 0, -1), Limit: 50000})
		if err != nil {
			return nil, err
		}
		if cfg.RegistryEnforcement {
			if err := validateFeatureObservationProvenance(observations); err != nil {
				return nil, err
			}
		}
		grouped := map[string]map[string]storage.MarketOpsFeatureObservationRecord{}
		for _, row := range observations {
			if row.NumericValue == nil || (row.QualityState != "usable" && row.QualityState != "usable_with_warning") {
				continue
			}
			date := day(row.SessionDate).Format("2006-01-02")
			if grouped[date] == nil {
				grouped[date] = map[string]storage.MarketOpsFeatureObservationRecord{}
			}
			if _, ok := grouped[date][row.FeatureKey]; !ok {
				grouped[date][row.FeatureKey] = row
			}
		}
		dates := make([]string, 0, len(grouped))
		for date := range grouped {
			dates = append(dates, date)
		}
		sort.Strings(dates)
		for _, date := range dates {
			rows := grouped[date]
			values := map[string]float64{}
			ids := []string{}
			provenance := []any{}
			for _, key := range required {
				row, ok := rows[key]
				if !ok {
					continue
				}
				values[key] = *row.NumericValue
				ids = append(ids, row.FeatureObservationID)
				var details map[string]any
				_ = json.Unmarshal(row.QualityDetailsJSON, &details)
				if entry, ok := details["input_provenance"]; ok {
					provenance = append(provenance, entry)
				}
			}
			if len(values) < len(required) {
				continue
			}
			signed := 0.0
			add := func(up bool, down bool, weight float64) {
				if up {
					signed += weight
				}
				if down {
					signed -= weight
				}
			}
			add(values["range_position_252d"] <= 10, values["range_position_252d"] >= 90, 25)
			add(values["rsi_14"] <= 40, values["rsi_14"] >= 60, 20)
			add(values["return_5d"] < 0 && values["volume_ratio_10d"] > 1.2, values["return_5d"] > 0 && values["volume_ratio_10d"] < .8, 15)
			add(values["distance_sma_50_pct"] > 0 && values["distance_sma_200_pct"] > 0 && values["sma_50_slope_20d_pct"] > 0, values["distance_sma_50_pct"] < 0 && values["distance_sma_200_pct"] < 0 && values["sma_50_slope_20d_pct"] < 0, 25)
			add(values["return_5d"] <= -5, values["return_5d"] >= 5, 15)
			direction := "non_directional"
			textDirection := "neutral"
			if signed >= 25 {
				direction = "upside"
				textDirection = "bullish"
			} else if signed <= -25 {
				direction = "downside"
				textDirection = "bearish"
			}
			session, _ := time.Parse("2006-01-02", date)
			payload := map[string]any{"technical_direction": textDirection, "technical_score": round(signed), "options_corroboration": "unavailable", "technical_only_historical": true, "required_technical_inputs": required}
			if contains(cfg.Modes, storage.MarketOpsAlgorithmEvaluationModeWalkForward) {
				record := storage.MarketOpsAlgorithmEvaluationResultRecord{EvaluationResultID: stableID("aevalres_", cfg.RunID, algorithmID, "walk_forward", symbol, date), RunID: cfg.RunID, TenantID: cfg.TenantID, AlgorithmID: algorithmID, AlgorithmVersion: "v1", EvaluationMode: storage.MarketOpsAlgorithmEvaluationModeWalkForward, EvaluationProfile: storage.MarketOpsAlgorithmEvaluationProfileDirectional, ResultType: "risk_reward_temporal", Symbol: symbol, ObservationSessionDate: session, Score: round(math.Abs(signed) / 100), Confidence: 1, Severity: severityFor(math.Abs(signed) / 25), Direction: direction, ResultPayloadJSON: mustJSON(payload), InputProvenanceJSON: mustJSON(map[string]any{"feature_input_provenance": provenance, "options_corroboration": "unavailable"}), FeatureValueIDs: ids, DeterministicKey: stableID("key_", cfg.RunID, algorithmID, "walk_forward", symbol, date)}
				out = append(out, evaluated{record: record, sample: sample{symbol: symbol, session: session}})
			}
			if contains(cfg.Modes, storage.MarketOpsAlgorithmEvaluationModeRetrospective) {
				record := storage.MarketOpsAlgorithmEvaluationResultRecord{EvaluationResultID: stableID("aevalres_", cfg.RunID, algorithmID, "retrospective", symbol, date), RunID: cfg.RunID, TenantID: cfg.TenantID, AlgorithmID: algorithmID, AlgorithmVersion: "v1", EvaluationMode: storage.MarketOpsAlgorithmEvaluationModeRetrospective, EvaluationProfile: storage.MarketOpsAlgorithmEvaluationProfileDirectional, ResultType: "risk_reward_temporal", Symbol: symbol, ObservationSessionDate: session, Score: round(math.Abs(signed) / 100), Confidence: 1, Severity: severityFor(math.Abs(signed) / 25), Direction: direction, ResultPayloadJSON: mustJSON(payload), InputProvenanceJSON: mustJSON(map[string]any{"feature_input_provenance": provenance, "options_corroboration": "unavailable"}), FeatureValueIDs: ids, DeterministicKey: stableID("key_", cfg.RunID, algorithmID, "retrospective", symbol, date)}
				out = append(out, evaluated{record: record, sample: sample{symbol: symbol, session: session}})
			}
		}
	}
	return out, nil
}

func buildOutcomes(cfg Config, item evaluated, prices []sample) []storage.MarketOpsAlgorithmEvaluationOutcomeRecord {
	origin := -1
	for i := range prices {
		if prices[i].session.Equal(item.record.ObservationSessionDate) {
			origin = i
			break
		}
	}
	out := []storage.MarketOpsAlgorithmEvaluationOutcomeRecord{}
	for _, horizon := range DefaultHorizons {
		outcome := storage.MarketOpsAlgorithmEvaluationOutcomeRecord{EvaluationOutcomeID: stableID("aevalout_", item.record.EvaluationResultID, fmt.Sprint(horizon)), RunID: cfg.RunID, EvaluationResultID: item.record.EvaluationResultID, TenantID: cfg.TenantID, HorizonSessions: horizon, OutcomeStatus: storage.MarketOpsOutcomePending, OutcomePayloadJSON: []byte(`{}`), DeterministicKey: stableID("key_", item.record.EvaluationResultID, fmt.Sprint(horizon))}
		if origin < 0 || prices[origin].close <= 0 {
			outcome.OutcomeStatus = storage.MarketOpsOutcomeMissingPrice
			out = append(out, outcome)
			continue
		}
		if origin+horizon >= len(prices) || prices[origin+horizon].session.After(cfg.AsOf) {
			out = append(out, outcome)
			continue
		}
		window := prices[origin+1 : origin+horizon+1]
		valid := true
		for _, point := range window {
			if point.close <= 0 {
				valid = false
			}
		}
		if !valid {
			outcome.OutcomeStatus = storage.MarketOpsOutcomeMissingPrice
			out = append(out, outcome)
			continue
		}
		final := window[len(window)-1]
		ret := final.close/prices[origin].close - 1
		abs := math.Abs(ret)
		mfe, mae := excursions(item.record.Direction, prices[origin], window)
		dd := drawdown(prices[origin : origin+horizon+1])
		vol := realizedVol(window)
		outcome.OutcomeStatus = storage.MarketOpsOutcomeMatured
		outcome.MaturedSessionDate = timePtr(final.session)
		outcome.ForwardReturn = floatPtr(round(ret))
		outcome.AbsoluteForwardReturn = floatPtr(round(abs))
		outcome.MaxFavorableExcursion = floatPtr(round(mfe))
		outcome.MaxAdverseExcursion = floatPtr(round(mae))
		outcome.MaximumDrawdown = floatPtr(round(dd))
		outcome.RealizedVolChange = floatPtr(round(vol))
		if item.record.Direction != "non_directional" {
			hit := (item.record.Direction == "upside" && ret > 0) || (item.record.Direction == "downside" && ret < 0)
			outcome.DirectionalHit = &hit
		}
		hitThreshold := abs >= cfg.Threshold
		outcome.ThresholdHit = &hitThreshold
		ids := []string{}
		for _, point := range window {
			ids = append(ids, point.eventID)
		}
		outcome.OutcomeEventIDs = ids
		outcome.OutcomePayloadJSON = mustJSON(map[string]any{"evaluation_profile": item.record.EvaluationProfile, "threshold": cfg.Threshold, "calculation_version": CalculationVersion})
		out = append(out, outcome)
	}
	return out
}

func aggregate(value ProfileMetrics, result storage.MarketOpsAlgorithmEvaluationResultRecord, outcome storage.MarketOpsAlgorithmEvaluationOutcomeRecord) ProfileMetrics {
	if outcome.OutcomeStatus != storage.MarketOpsOutcomeMatured {
		return value
	}
	value.Matured++
	if outcome.ForwardReturn != nil {
		value.returns = append(value.returns, *outcome.ForwardReturn)
	}
	if outcome.AbsoluteForwardReturn != nil {
		value.absoluteReturns = append(value.absoluteReturns, *outcome.AbsoluteForwardReturn)
	}
	if result.Direction != "non_directional" && outcome.DirectionalHit != nil {
		value.DirectionalSamples++
		if *outcome.DirectionalHit {
			value.DirectionalHits++
		}
	}
	return value
}

func finalize(value ProfileMetrics) ProfileMetrics {
	if len(value.returns) > 0 {
		mean, std := floatStats(value.returns)
		value.MeanForwardReturn = floatPtr(round(mean))
		margin := 1.96 * std / math.Sqrt(float64(len(value.returns)))
		value.MeanReturnCILower = floatPtr(round(mean - margin))
		value.MeanReturnCIUpper = floatPtr(round(mean + margin))
	}
	if len(value.absoluteReturns) > 0 {
		mean, _ := floatStats(value.absoluteReturns)
		value.MeanAbsoluteReturn = floatPtr(round(mean))
	}
	if value.DirectionalSamples > 0 {
		rate := float64(value.DirectionalHits) / float64(value.DirectionalSamples)
		low, high := wilson(value.DirectionalHits, value.DirectionalSamples)
		value.DirectionalHitRate = floatPtr(round(rate))
		value.DirectionalHitCILower = floatPtr(round(low))
		value.DirectionalHitCIUpper = floatPtr(round(high))
	}
	value.returns = nil
	value.absoluteReturns = nil
	return value
}

// internal aggregate samples are intentionally omitted from serialized metrics.
func (p *ProfileMetrics) UnmarshalJSON(data []byte) error {
	type alias ProfileMetrics
	var parsed alias
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}
	*p = ProfileMetrics(parsed)
	return nil
}

func priceValues(event storage.NormalizedEventLedgerRecord, fallback float64) (float64, float64, float64) {
	var payload map[string]any
	_ = json.Unmarshal(event.NormalizedPayload, &payload)
	close, _ := number(payload["close"])
	high, _ := number(payload["high"])
	low, _ := number(payload["low"])
	if close == 0 {
		close = 1 + fallback/100
	}
	if high == 0 {
		high = close
	}
	if low == 0 {
		low = close
	}
	return close, high, low
}
func number(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case json.Number:
		x, e := v.Float64()
		return x, e == nil
	default:
		return 0, false
	}
}
func stringValue(value any) string {
	if v, ok := value.(string); ok {
		return v
	}
	return ""
}
func sampleStats(values []sample) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}
	nums := make([]float64, len(values))
	for i := range values {
		nums[i] = values[i].value
	}
	return floatStats(nums)
}
func floatStats(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}
	sum := 0.
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))
	variance := 0.
	for _, v := range values {
		d := v - mean
		variance += d * d
	}
	return mean, math.Sqrt(variance / float64(len(values)))
}
func sampleMedian(values []sample) float64 {
	nums := make([]float64, len(values))
	for i := range values {
		nums[i] = values[i].value
	}
	return medianFloat(nums)
}
func medianFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}
func severityFor(score float64) string {
	if score >= 3 {
		return "high"
	}
	if score >= 2 {
		return "medium"
	}
	if score >= 1 {
		return "low"
	}
	return "info"
}
func confidence(score float64) float64 { return round(math.Min(1, score/4)) }
func excursions(direction string, origin sample, window []sample) (float64, float64) {
	best, worst := 0., 0.
	for _, point := range window {
		up := point.high/origin.close - 1
		down := point.low/origin.close - 1
		if direction == "downside" {
			up, down = -down, -up
		}
		if up > best {
			best = up
		}
		if down < worst {
			worst = down
		}
	}
	return best, worst
}
func drawdown(values []sample) float64 {
	peak := values[0].close
	worst := 0.
	for _, value := range values {
		if value.close > peak {
			peak = value.close
		}
		if peak > 0 {
			d := value.close/peak - 1
			if d < worst {
				worst = d
			}
		}
	}
	return worst
}
func realizedVol(values []sample) float64 {
	if len(values) < 2 {
		return 0
	}
	returns := []float64{}
	for i := 1; i < len(values); i++ {
		if values[i-1].close > 0 {
			returns = append(returns, values[i].close/values[i-1].close-1)
		}
	}
	_, std := floatStats(returns)
	return std * math.Sqrt(252)
}
func wilson(successes, total int) (float64, float64) {
	if total == 0 {
		return 0, 0
	}
	z := 1.96
	n := float64(total)
	p := float64(successes) / n
	den := 1 + z*z/n
	center := (p + z*z/(2*n)) / den
	margin := z * math.Sqrt((p*(1-p)+z*z/(4*n))/n) / den
	return center - margin, center + margin
}
func stableID(prefix string, parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		hash.Write([]byte(part))
		hash.Write([]byte{0})
	}
	return prefix + hex.EncodeToString(hash.Sum(nil))[:32]
}
func mustJSON(value any) []byte { encoded, _ := json.Marshal(value); return encoded }
func day(value time.Time) time.Time {
	return time.Date(value.UTC().Year(), value.UTC().Month(), value.UTC().Day(), 0, 0, 0, 0, time.UTC)
}
func round(value float64) float64        { return math.Round(value*1e8) / 1e8 }
func floatPtr(value float64) *float64    { return &value }
func timePtr(value time.Time) *time.Time { return &value }
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
func uniqueStrings(values []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}
func uniqueUpper(values []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.ToUpper(strings.TrimSpace(value))
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

package hypothesisbacktest

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const (
	SummaryVersion                   = "marketops.hypothesis_calibration.v1"
	DefaultOutcomeCalculationVersion = "marketops.forward_outcome.v1"
	ModeSingle                       = "single"
	ModeComparison                   = "comparison"
	ModeWalkForward                  = "walk_forward"
)

type Config struct {
	TenantID                  string
	HypothesisKey             string
	HypothesisVersions        []string
	OutcomeCalculationVersion string
	Symbols                   []string
	WindowStart               time.Time
	WindowEnd                 time.Time
	AsOf                      time.Time
	Mode                      string
	MinimumSampleSize         int
	TrainSessions             int
	TestSessions              int
	MaximumFolds              int
}

type Input struct {
	Evaluations  []storage.MarketOpsHypothesisEvaluationRecord
	Outcomes     []storage.MarketOpsSignalOutcomeRecord
	Observations []storage.MarketOpsFeatureObservationRecord
}

type ConfidenceBandMetrics struct {
	Samples            int      `json:"samples"`
	DirectionalHitRate *float64 `json:"directional_hit_rate,omitempty"`
	CalibrationError   *float64 `json:"calibration_error,omitempty"`
}

type Metrics struct {
	Evaluations              int                              `json:"evaluations"`
	EligibleStates           int                              `json:"eligible_states"`
	Triggers                 int                              `json:"triggers"`
	TriggerRate              *float64                         `json:"trigger_rate,omitempty"`
	IndependentSamples       int                              `json:"independent_samples"`
	MaturedOutcomeSamples    int                              `json:"matured_outcome_samples"`
	DirectionalHitRate       *float64                         `json:"directional_hit_rate,omitempty"`
	MeanForwardReturn        *float64                         `json:"mean_forward_return,omitempty"`
	MedianForwardReturn      *float64                         `json:"median_forward_return,omitempty"`
	MeanFavorableExcursion   *float64                         `json:"mean_favorable_excursion,omitempty"`
	MedianFavorableExcursion *float64                         `json:"median_favorable_excursion,omitempty"`
	MeanAdverseExcursion     *float64                         `json:"mean_adverse_excursion,omitempty"`
	MedianAdverseExcursion   *float64                         `json:"median_adverse_excursion,omitempty"`
	DrawdownIncidence        *float64                         `json:"drawdown_incidence,omitempty"`
	MeanRealizedVolChange    *float64                         `json:"mean_realized_volatility_change,omitempty"`
	CalibrationError         *float64                         `json:"calibration_error,omitempty"`
	ConfidenceBands          map[string]ConfidenceBandMetrics `json:"confidence_bands"`
	BelowMinimumSampleSize   bool                             `json:"below_minimum_sample_size"`
}

type VersionReport struct {
	HypothesisVersion  string             `json:"hypothesis_version"`
	Overall            Metrics            `json:"overall"`
	ByHorizon          map[string]Metrics `json:"by_horizon"`
	ByAsset            map[string]Metrics `json:"by_asset"`
	ByYear             map[string]Metrics `json:"by_year"`
	ByVolatilityRegime map[string]Metrics `json:"by_volatility_regime"`
	ByEarningsWindow   map[string]Metrics `json:"by_earnings_window"`
}

type Comparison struct {
	BaselineVersion       string   `json:"baseline_version"`
	CandidateVersion      string   `json:"candidate_version"`
	TriggerRateDelta      *float64 `json:"trigger_rate_delta,omitempty"`
	DirectionalHitDelta   *float64 `json:"directional_hit_rate_delta,omitempty"`
	MeanReturnDelta       *float64 `json:"mean_forward_return_delta,omitempty"`
	CalibrationErrorDelta *float64 `json:"calibration_error_delta,omitempty"`
	AdvisoryOnly          bool     `json:"advisory_only"`
}

type WalkForwardFold struct {
	Fold       int           `json:"fold"`
	TrainStart string        `json:"train_start"`
	TrainEnd   string        `json:"train_end"`
	TestStart  string        `json:"test_start"`
	TestEnd    string        `json:"test_end"`
	Train      VersionReport `json:"train"`
	Test       VersionReport `json:"test"`
	Warnings   []string      `json:"warnings"`
}

type Report struct {
	SummaryVersion            string                   `json:"summary_version"`
	Mode                      string                   `json:"mode"`
	HypothesisKey             string                   `json:"hypothesis_key"`
	HypothesisVersions        []string                 `json:"hypothesis_versions"`
	OutcomeCalculationVersion string                   `json:"outcome_calculation_version"`
	Symbols                   []string                 `json:"symbols"`
	WindowStart               string                   `json:"window_start"`
	WindowEnd                 string                   `json:"window_end"`
	AsOf                      string                   `json:"as_of"`
	MinimumSampleSize         int                      `json:"minimum_sample_size"`
	Versions                  map[string]VersionReport `json:"versions"`
	Comparison                *Comparison              `json:"comparison,omitempty"`
	WalkForward               []WalkForwardFold        `json:"walk_forward,omitempty"`
	Warnings                  []string                 `json:"warnings"`
	PromotionAllowed          bool                     `json:"promotion_allowed"`
}

type segment struct {
	earnings   string
	volatility string
}

type sample struct {
	evaluation storage.MarketOpsHypothesisEvaluationRecord
	outcome    storage.MarketOpsSignalOutcomeRecord
	segment    segment
}

func Build(cfg Config, input Input) (Report, error) {
	cfg = normalize(cfg)
	if err := validate(cfg); err != nil {
		return Report{}, err
	}
	selected := map[string]bool{}
	for _, version := range cfg.HypothesisVersions {
		selected[version] = true
	}
	evaluationByID := map[string]storage.MarketOpsHypothesisEvaluationRecord{}
	for _, record := range input.Evaluations {
		if record.TenantID != cfg.TenantID || record.HypothesisKey != cfg.HypothesisKey || !selected[record.HypothesisVersion] {
			return Report{}, fmt.Errorf("G145 evaluation %s violates exact hypothesis-version isolation", record.EvaluationID)
		}
		if !within(record.SessionDate, cfg.WindowStart, cfg.WindowEnd) || day(record.AsOfTime).After(cfg.AsOf) {
			return Report{}, fmt.Errorf("G145 evaluation %s is outside the point-in-time window", record.EvaluationID)
		}
		if _, exists := evaluationByID[record.EvaluationID]; exists {
			return Report{}, fmt.Errorf("G145 duplicate evaluation %s", record.EvaluationID)
		}
		evaluationByID[record.EvaluationID] = record
	}
	for _, record := range input.Outcomes {
		if record.SourceType != storage.MarketOpsOutcomeSourceHypothesisEvaluation || record.TenantID != cfg.TenantID || record.HypothesisKey != cfg.HypothesisKey || !selected[record.HypothesisVersion] || record.CalculationVersion != cfg.OutcomeCalculationVersion {
			return Report{}, fmt.Errorf("G145 outcome %s violates exact hypothesis-version isolation", record.OutcomeID)
		}
		evaluation, ok := evaluationByID[record.SourceID]
		if !ok || evaluation.HypothesisVersion != record.HypothesisVersion {
			return Report{}, fmt.Errorf("G145 outcome %s does not match an isolated evaluation", record.OutcomeID)
		}
	}

	segments := segmentEvaluations(input.Evaluations, input.Observations)
	report := Report{
		SummaryVersion:            SummaryVersion,
		Mode:                      cfg.Mode,
		HypothesisKey:             cfg.HypothesisKey,
		OutcomeCalculationVersion: cfg.OutcomeCalculationVersion,
		HypothesisVersions:        append([]string(nil), cfg.HypothesisVersions...),
		Symbols:                   append([]string(nil), cfg.Symbols...),
		WindowStart:               dateString(cfg.WindowStart),
		WindowEnd:                 dateString(cfg.WindowEnd),
		AsOf:                      dateString(cfg.AsOf),
		MinimumSampleSize:         cfg.MinimumSampleSize,
		Versions:                  map[string]VersionReport{},
		Warnings:                  []string{},
		PromotionAllowed:          false,
	}
	for _, version := range cfg.HypothesisVersions {
		evaluations := filterEvaluations(input.Evaluations, version, cfg.WindowStart, cfg.WindowEnd)
		samples := joinSamples(evaluations, input.Outcomes, segments, cfg.AsOf)
		versionReport := buildVersionReport(version, evaluations, samples, segments, cfg.MinimumSampleSize)
		report.Versions[version] = versionReport
		if versionReport.Overall.BelowMinimumSampleSize {
			report.Warnings = append(report.Warnings, fmt.Sprintf("sample_size_below_minimum:%s:%d/%d", version, versionReport.Overall.IndependentSamples, cfg.MinimumSampleSize))
		}
	}
	if cfg.Mode == ModeComparison {
		baseline := report.Versions[cfg.HypothesisVersions[0]].Overall
		candidate := report.Versions[cfg.HypothesisVersions[1]].Overall
		report.Comparison = &Comparison{
			BaselineVersion:       cfg.HypothesisVersions[0],
			CandidateVersion:      cfg.HypothesisVersions[1],
			TriggerRateDelta:      delta(candidate.TriggerRate, baseline.TriggerRate),
			DirectionalHitDelta:   delta(candidate.DirectionalHitRate, baseline.DirectionalHitRate),
			MeanReturnDelta:       delta(candidate.MeanForwardReturn, baseline.MeanForwardReturn),
			CalibrationErrorDelta: delta(candidate.CalibrationError, baseline.CalibrationError),
			AdvisoryOnly:          true,
		}
	}
	if cfg.Mode == ModeWalkForward {
		version := cfg.HypothesisVersions[0]
		evaluations := filterEvaluations(input.Evaluations, version, cfg.WindowStart, cfg.WindowEnd)
		report.WalkForward = buildWalkForward(cfg, version, evaluations, input.Outcomes, segments)
		if len(report.WalkForward) == 0 {
			report.Warnings = append(report.Warnings, "insufficient_sessions_for_walk_forward")
		}
	}
	sort.Strings(report.Warnings)
	return report, nil
}

func normalize(cfg Config) Config {
	cfg.TenantID = strings.TrimSpace(cfg.TenantID)
	cfg.HypothesisKey = strings.ToUpper(strings.TrimSpace(cfg.HypothesisKey))
	cfg.Mode = strings.ToLower(strings.TrimSpace(cfg.Mode))
	cfg.OutcomeCalculationVersion = strings.TrimSpace(cfg.OutcomeCalculationVersion)
	if cfg.OutcomeCalculationVersion == "" {
		cfg.OutcomeCalculationVersion = DefaultOutcomeCalculationVersion
	}
	if cfg.Mode == "" {
		cfg.Mode = ModeSingle
	}
	seenVersions := map[string]bool{}
	versions := []string{}
	for _, value := range cfg.HypothesisVersions {
		value = strings.TrimSpace(value)
		if value != "" && !seenVersions[value] {
			seenVersions[value] = true
			versions = append(versions, value)
		}
	}
	cfg.HypothesisVersions = versions
	seenSymbols := map[string]bool{}
	symbols := []string{}
	for _, value := range cfg.Symbols {
		value = strings.ToUpper(strings.TrimSpace(value))
		if value != "" && !seenSymbols[value] {
			seenSymbols[value] = true
			symbols = append(symbols, value)
		}
	}
	sort.Strings(symbols)
	cfg.Symbols = symbols
	cfg.WindowStart, cfg.WindowEnd, cfg.AsOf = day(cfg.WindowStart), day(cfg.WindowEnd), day(cfg.AsOf)
	if cfg.MinimumSampleSize == 0 {
		cfg.MinimumSampleSize = 100
	}
	if cfg.TrainSessions == 0 {
		cfg.TrainSessions = 60
	}
	if cfg.TestSessions == 0 {
		cfg.TestSessions = 20
	}
	if cfg.MaximumFolds == 0 {
		cfg.MaximumFolds = 6
	}
	return cfg
}

func validate(cfg Config) error {
	if cfg.TenantID == "" || cfg.HypothesisKey == "" || len(cfg.HypothesisVersions) == 0 || len(cfg.Symbols) == 0 {
		return errors.New("G145 tenant, hypothesis key/version, and explicit symbols are required")
	}
	if len(cfg.Symbols) > 10 {
		return errors.New("G145 supports at most 10 explicit symbols")
	}
	if cfg.WindowStart.IsZero() || cfg.WindowEnd.IsZero() || cfg.AsOf.IsZero() || cfg.WindowEnd.Before(cfg.WindowStart) || cfg.AsOf.Before(cfg.WindowEnd) {
		return errors.New("G145 window and as-of dates are invalid")
	}
	if cfg.MinimumSampleSize < 1 || cfg.MinimumSampleSize > 10000 {
		return errors.New("G145 minimum sample size must be between 1 and 10000")
	}
	switch cfg.Mode {
	case ModeSingle, ModeWalkForward:
		if len(cfg.HypothesisVersions) != 1 {
			return fmt.Errorf("G145 %s mode requires exactly one hypothesis version", cfg.Mode)
		}
	case ModeComparison:
		if len(cfg.HypothesisVersions) != 2 {
			return errors.New("G145 comparison mode requires exactly two distinct hypothesis versions")
		}
	default:
		return fmt.Errorf("unsupported G145 mode %s", cfg.Mode)
	}
	if cfg.TrainSessions < 1 || cfg.TestSessions < 1 || cfg.MaximumFolds < 1 || cfg.MaximumFolds > 20 {
		return errors.New("G145 walk-forward bounds are invalid")
	}
	return nil
}

func buildVersionReport(version string, evaluations []storage.MarketOpsHypothesisEvaluationRecord, samples []sample, segments map[string]segment, minimum int) VersionReport {
	report := VersionReport{
		HypothesisVersion:  version,
		ByHorizon:          map[string]Metrics{},
		ByAsset:            map[string]Metrics{},
		ByYear:             map[string]Metrics{},
		ByVolatilityRegime: map[string]Metrics{},
		ByEarningsWindow:   map[string]Metrics{},
	}
	report.Overall = metricSet(evaluations, samples, minimum)
	for _, horizon := range []int{1, 5, 10, 20} {
		report.ByHorizon[fmt.Sprint(horizon)] = metricSet(evaluations, filterSamples(samples, func(item sample) bool {
			return item.outcome.HorizonSessions == horizon
		}), minimum)
	}
	for _, symbol := range distinctEvaluationValue(evaluations, func(record storage.MarketOpsHypothesisEvaluationRecord) string { return record.Symbol }) {
		selected := filterEvaluationPredicate(evaluations, func(record storage.MarketOpsHypothesisEvaluationRecord) bool { return record.Symbol == symbol })
		report.ByAsset[symbol] = metricSet(selected, filterSamplesByEvaluations(samples, selected), minimum)
	}
	for _, year := range distinctEvaluationValue(evaluations, func(record storage.MarketOpsHypothesisEvaluationRecord) string {
		return fmt.Sprint(record.SessionDate.Year())
	}) {
		selected := filterEvaluationPredicate(evaluations, func(record storage.MarketOpsHypothesisEvaluationRecord) bool {
			return fmt.Sprint(record.SessionDate.Year()) == year
		})
		report.ByYear[year] = metricSet(selected, filterSamplesByEvaluations(samples, selected), minimum)
	}
	for _, value := range distinctSegmentValue(evaluations, segments, func(item segment) string { return item.volatility }) {
		selectedEvaluations := filterEvaluationPredicate(evaluations, func(record storage.MarketOpsHypothesisEvaluationRecord) bool {
			return segments[record.EvaluationID].volatility == value
		})
		report.ByVolatilityRegime[value] = metricSet(selectedEvaluations, filterSamplesByEvaluations(samples, selectedEvaluations), minimum)
	}
	for _, value := range distinctSegmentValue(evaluations, segments, func(item segment) string { return item.earnings }) {
		selectedEvaluations := filterEvaluationPredicate(evaluations, func(record storage.MarketOpsHypothesisEvaluationRecord) bool {
			return segments[record.EvaluationID].earnings == value
		})
		report.ByEarningsWindow[value] = metricSet(selectedEvaluations, filterSamplesByEvaluations(samples, selectedEvaluations), minimum)
	}
	return report
}

func metricSet(evaluations []storage.MarketOpsHypothesisEvaluationRecord, samples []sample, minimum int) Metrics {
	metrics := Metrics{
		Evaluations:            len(evaluations),
		ConfidenceBands:        map[string]ConfidenceBandMetrics{},
		IndependentSamples:     independentSampleCount(samples),
		MaturedOutcomeSamples:  len(samples),
		BelowMinimumSampleSize: independentSampleCount(samples) < minimum,
	}
	for _, record := range evaluations {
		if record.Eligible && !record.Invalidated {
			metrics.EligibleStates++
		}
		if record.Triggered && record.Eligible && !record.Invalidated {
			metrics.Triggers++
		}
	}
	if metrics.EligibleStates > 0 {
		metrics.TriggerRate = pointer(float64(metrics.Triggers) / float64(metrics.EligibleStates))
	}
	returns, favorable, adverse := []float64{}, []float64{}, []float64{}
	drawdowns, volatility, brier := []float64{}, []float64{}, []float64{}
	hits, knownHits := 0, 0
	bands := map[string][]sample{
		"low_0_0.50":       {},
		"medium_0.50_0.75": {},
		"high_0.75_1.00":   {},
	}
	for _, item := range samples {
		if item.outcome.ForwardReturn != nil {
			returns = append(returns, *item.outcome.ForwardReturn)
		}
		if item.outcome.MaxFavorableExcursion != nil {
			favorable = append(favorable, *item.outcome.MaxFavorableExcursion)
		}
		if item.outcome.MaxAdverseExcursion != nil {
			adverse = append(adverse, *item.outcome.MaxAdverseExcursion)
		}
		if item.outcome.MaximumDrawdown != nil {
			drawdowns = append(drawdowns, *item.outcome.MaximumDrawdown)
		}
		if item.outcome.RealizedVolChange != nil {
			volatility = append(volatility, *item.outcome.RealizedVolChange)
		}
		if item.outcome.DirectionalHit != nil {
			knownHits++
			target := 0.0
			if *item.outcome.DirectionalHit {
				hits++
				target = 1
			}
			if item.evaluation.ConfidenceScore != nil {
				difference := *item.evaluation.ConfidenceScore - target
				brier = append(brier, difference*difference)
			}
		}
		if item.evaluation.ConfidenceScore != nil {
			key := "low_0_0.50"
			if *item.evaluation.ConfidenceScore >= .75 {
				key = "high_0.75_1.00"
			} else if *item.evaluation.ConfidenceScore >= .5 {
				key = "medium_0.50_0.75"
			}
			bands[key] = append(bands[key], item)
		}
	}
	if knownHits > 0 {
		metrics.DirectionalHitRate = pointer(float64(hits) / float64(knownHits))
	}
	metrics.MeanForwardReturn, metrics.MedianForwardReturn = mean(returns), median(returns)
	metrics.MeanFavorableExcursion, metrics.MedianFavorableExcursion = mean(favorable), median(favorable)
	metrics.MeanAdverseExcursion, metrics.MedianAdverseExcursion = mean(adverse), median(adverse)
	if len(drawdowns) > 0 {
		count := 0
		for _, value := range drawdowns {
			if value < 0 {
				count++
			}
		}
		metrics.DrawdownIncidence = pointer(float64(count) / float64(len(drawdowns)))
	}
	metrics.MeanRealizedVolChange, metrics.CalibrationError = mean(volatility), mean(brier)
	for key, items := range bands {
		bandHits, bandKnown := 0, 0
		bandBrier := []float64{}
		for _, item := range items {
			if item.outcome.DirectionalHit == nil || item.evaluation.ConfidenceScore == nil {
				continue
			}
			bandKnown++
			target := 0.0
			if *item.outcome.DirectionalHit {
				bandHits++
				target = 1
			}
			difference := *item.evaluation.ConfidenceScore - target
			bandBrier = append(bandBrier, difference*difference)
		}
		entry := ConfidenceBandMetrics{Samples: len(items), CalibrationError: mean(bandBrier)}
		if bandKnown > 0 {
			entry.DirectionalHitRate = pointer(float64(bandHits) / float64(bandKnown))
		}
		metrics.ConfidenceBands[key] = entry
	}
	return metrics
}

func buildWalkForward(cfg Config, version string, evaluations []storage.MarketOpsHypothesisEvaluationRecord, outcomes []storage.MarketOpsSignalOutcomeRecord, segments map[string]segment) []WalkForwardFold {
	dates := distinctDates(evaluations)
	folds := []WalkForwardFold{}
	for trainEnd := cfg.TrainSessions; trainEnd < len(dates) && len(folds) < cfg.MaximumFolds; trainEnd += cfg.TestSessions {
		testEnd := trainEnd + cfg.TestSessions
		if testEnd > len(dates) {
			testEnd = len(dates)
		}
		trainStartDate, trainEndDate := dates[0], dates[trainEnd-1]
		testStartDate, testEndDate := dates[trainEnd], dates[testEnd-1]
		trainEvaluations := filterEvaluations(evaluations, version, trainStartDate, trainEndDate)
		testEvaluations := filterEvaluations(evaluations, version, testStartDate, testEndDate)
		trainSamples := joinSamples(trainEvaluations, outcomes, segments, cfg.AsOf)
		testSamples := joinSamples(testEvaluations, outcomes, segments, cfg.AsOf)
		fold := WalkForwardFold{
			Fold:       len(folds) + 1,
			TrainStart: dateString(trainStartDate),
			TrainEnd:   dateString(trainEndDate),
			TestStart:  dateString(testStartDate),
			TestEnd:    dateString(testEndDate),
			Train:      buildVersionReport(version, trainEvaluations, trainSamples, segments, cfg.MinimumSampleSize),
			Test:       buildVersionReport(version, testEvaluations, testSamples, segments, cfg.MinimumSampleSize),
			Warnings:   []string{},
		}
		if fold.Train.Overall.BelowMinimumSampleSize {
			fold.Warnings = append(fold.Warnings, "train_sample_size_below_minimum")
		}
		if fold.Test.Overall.BelowMinimumSampleSize {
			fold.Warnings = append(fold.Warnings, "test_sample_size_below_minimum")
		}
		folds = append(folds, fold)
		if testEnd == len(dates) {
			break
		}
	}
	return folds
}

func segmentEvaluations(evaluations []storage.MarketOpsHypothesisEvaluationRecord, observations []storage.MarketOpsFeatureObservationRecord) map[string]segment {
	result := map[string]segment{}
	for _, evaluation := range evaluations {
		item := segment{earnings: "unknown", volatility: "unknown"}
		selected := map[string]storage.MarketOpsFeatureObservationRecord{}
		for _, observation := range observations {
			if observation.TenantID != evaluation.TenantID || observation.Symbol != evaluation.Symbol ||
				!day(observation.SessionDate).Equal(day(evaluation.SessionDate)) ||
				observation.AsOfTime.After(evaluation.AsOfTime) || observation.TextValue == nil ||
				!usable(observation.QualityState) {
				continue
			}
			if observation.FeatureKey != "earnings_window_state" && observation.FeatureKey != "term_structure_state" {
				continue
			}
			current, exists := selected[observation.FeatureKey]
			if !exists || observation.AsOfTime.After(current.AsOfTime) ||
				(observation.AsOfTime.Equal(current.AsOfTime) && observation.FeatureObservationID > current.FeatureObservationID) {
				selected[observation.FeatureKey] = observation
			}
		}
		if observation, ok := selected["earnings_window_state"]; ok {
			item.earnings = strings.TrimSpace(*observation.TextValue)
			if item.earnings == "outside_window" {
				item.earnings = "outside_earnings"
			}
		}
		if observation, ok := selected["term_structure_state"]; ok {
			item.volatility = strings.TrimSpace(*observation.TextValue)
		}
		if item.earnings == "" {
			item.earnings = "unknown"
		}
		if item.volatility == "" {
			item.volatility = "unknown"
		}
		result[evaluation.EvaluationID] = item
	}
	return result
}

func joinSamples(evaluations []storage.MarketOpsHypothesisEvaluationRecord, outcomes []storage.MarketOpsSignalOutcomeRecord, segments map[string]segment, asOf time.Time) []sample {
	byID := map[string]storage.MarketOpsHypothesisEvaluationRecord{}
	for _, evaluation := range evaluations {
		byID[evaluation.EvaluationID] = evaluation
	}
	seen := map[string]bool{}
	out := []sample{}
	for _, outcome := range outcomes {
		evaluation, ok := byID[outcome.SourceID]
		if !ok || outcome.OutcomeStatus != storage.MarketOpsOutcomeMatured ||
			outcome.MaturedSessionDate == nil || day(*outcome.MaturedSessionDate).After(day(asOf)) ||
			!evaluation.Eligible || !evaluation.Triggered || evaluation.Invalidated {
			continue
		}
		key := outcome.SourceID + "\x00" + fmt.Sprint(outcome.HorizonSessions)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, sample{evaluation: evaluation, outcome: outcome, segment: segments[evaluation.EvaluationID]})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].evaluation.SessionDate.Equal(out[j].evaluation.SessionDate) {
			return out[i].outcome.HorizonSessions < out[j].outcome.HorizonSessions
		}
		return out[i].evaluation.SessionDate.Before(out[j].evaluation.SessionDate)
	})
	return out
}

func filterEvaluations(records []storage.MarketOpsHypothesisEvaluationRecord, version string, start, end time.Time) []storage.MarketOpsHypothesisEvaluationRecord {
	return filterEvaluationPredicate(records, func(record storage.MarketOpsHypothesisEvaluationRecord) bool {
		return record.HypothesisVersion == version && within(record.SessionDate, start, end)
	})
}

func filterEvaluationPredicate(records []storage.MarketOpsHypothesisEvaluationRecord, keep func(storage.MarketOpsHypothesisEvaluationRecord) bool) []storage.MarketOpsHypothesisEvaluationRecord {
	out := []storage.MarketOpsHypothesisEvaluationRecord{}
	for _, record := range records {
		if keep(record) {
			out = append(out, record)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SessionDate.Equal(out[j].SessionDate) {
			return out[i].EvaluationID < out[j].EvaluationID
		}
		return out[i].SessionDate.Before(out[j].SessionDate)
	})
	return out
}

func filterSamples(records []sample, keep func(sample) bool) []sample {
	out := []sample{}
	for _, record := range records {
		if keep(record) {
			out = append(out, record)
		}
	}
	return out
}

func filterSamplesByEvaluations(records []sample, evaluations []storage.MarketOpsHypothesisEvaluationRecord) []sample {
	ids := map[string]bool{}
	for _, record := range evaluations {
		ids[record.EvaluationID] = true
	}
	return filterSamples(records, func(item sample) bool { return ids[item.evaluation.EvaluationID] })
}

func distinctEvaluationValue(records []storage.MarketOpsHypothesisEvaluationRecord, value func(storage.MarketOpsHypothesisEvaluationRecord) string) []string {
	set := map[string]bool{}
	for _, record := range records {
		set[value(record)] = true
	}
	return sortedKeys(set)
}

func distinctSegmentValue(records []storage.MarketOpsHypothesisEvaluationRecord, segments map[string]segment, value func(segment) string) []string {
	set := map[string]bool{}
	for _, record := range records {
		set[value(segments[record.EvaluationID])] = true
	}
	return sortedKeys(set)
}

func sortedKeys(values map[string]bool) []string {
	out := []string{}
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func distinctDates(records []storage.MarketOpsHypothesisEvaluationRecord) []time.Time {
	values := map[string]time.Time{}
	for _, record := range records {
		values[dateString(record.SessionDate)] = day(record.SessionDate)
	}
	out := make([]time.Time, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Before(out[j]) })
	return out
}

func independentSampleCount(samples []sample) int {
	ids := map[string]bool{}
	for _, item := range samples {
		ids[item.evaluation.EvaluationID] = true
	}
	return len(ids)
}

func mean(values []float64) *float64 {
	if len(values) == 0 {
		return nil
	}
	total := 0.0
	for _, value := range values {
		total += value
	}
	return pointer(total / float64(len(values)))
}

func median(values []float64) *float64 {
	if len(values) == 0 {
		return nil
	}
	values = append([]float64(nil), values...)
	sort.Float64s(values)
	middle := len(values) / 2
	if len(values)%2 == 1 {
		return pointer(values[middle])
	}
	return pointer((values[middle-1] + values[middle]) / 2)
}

func delta(left, right *float64) *float64 {
	if left == nil || right == nil {
		return nil
	}
	return pointer(*left - *right)
}

func pointer(value float64) *float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return nil
	}
	return &value
}

func usable(value string) bool {
	return value == storage.MarketOpsQualityUsable || value == storage.MarketOpsQualityUsableWithWarning
}

func within(value, start, end time.Time) bool {
	value = day(value)
	return !value.Before(day(start)) && !value.After(day(end))
}

func day(value time.Time) time.Time {
	if value.IsZero() {
		return value
	}
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func dateString(value time.Time) string {
	return day(value).Format("2006-01-02")
}

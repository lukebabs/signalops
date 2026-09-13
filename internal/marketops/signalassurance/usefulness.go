package signalassurance

import (
	"math"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

const (
	UsefulnessPolicyVersion = "saf_usefulness.v1"

	LifecycleConfirmed      = "confirmed"
	LifecycleDeveloping     = "developing"
	LifecycleMaterialized   = "materialized"
	LifecycleOutperformed   = "outperformed"
	LifecycleAdverseWarning = "adverse_warning"
	LifecycleInvalidated    = "invalidated"
	LifecycleExpired        = "expired"
	LifecycleCensored       = "censored"
)

type UsefulnessAssessment struct {
	LifecycleState                string
	Score                         *float64
	Components                    map[string]float64
	PolicyVersion                 string
	TimeToMaterializationSessions *int
}

type UsefulnessObservationInput struct {
	EvidenceSource       string
	State                string
	Direction            string
	HorizonSessions      int
	DirectionalHit       *bool
	DirectionalReturn    *float64
	RelativeReturn       *float64
	SectorRelativeReturn *float64
	MFE                  *float64
	MAE                  *float64
}

func AssessUsefulnessObservation(input UsefulnessObservationInput) UsefulnessAssessment {
	lifecycle := usefulnessLifecycle(input)
	components, complete := usefulnessComponents(input)
	if !complete {
		return UsefulnessAssessment{LifecycleState: lifecycle, PolicyVersion: UsefulnessPolicyVersion, Components: components}
	}
	score := 10 * (0.25*components["directional_resolution"] + 0.25*components["materialization_strength"] + 0.20*components["adverse_excursion_control"] + 0.20*components["benchmark_relative_performance"] + 0.10*components["timeliness_persistence"])
	if score < 0 {
		score = 0
	}
	if score > 10 {
		score = 10
	}
	var ttm *int
	if lifecycle == LifecycleMaterialized || lifecycle == LifecycleOutperformed {
		value := input.HorizonSessions
		if value > 0 {
			ttm = &value
		}
	}
	return UsefulnessAssessment{LifecycleState: lifecycle, Score: &score, Components: components, PolicyVersion: UsefulnessPolicyVersion, TimeToMaterializationSessions: ttm}
}

func AssessUsefulnessEffectiveness(row storage.SignalAssuranceEffectivenessRecord) UsefulnessAssessment {
	components := map[string]float64{}
	complete := true
	if row.DirectionalAccuracy == nil {
		complete = false
	} else {
		components["directional_resolution"] = clamp01((*row.DirectionalAccuracy - 0.40) / 0.30)
	}
	if row.AverageMFE == nil {
		complete = false
	} else {
		components["materialization_strength"] = clamp01(*row.AverageMFE / 0.05)
	}
	if row.AverageMAE == nil {
		complete = false
	} else {
		components["adverse_excursion_control"] = clamp01(1 - math.Abs(math.Min(*row.AverageMAE, 0))/0.05)
	}
	benchmark, ok := benchmarkComponent(row.AverageRelativeReturn, row.AverageSectorRelativeReturn)
	if !ok {
		complete = false
	} else {
		components["benchmark_relative_performance"] = benchmark
	}
	if row.MaterializationRate != nil {
		components["timeliness_persistence"] = clamp01(*row.MaterializationRate)
	} else if row.SampleSize > 0 && row.DirectionalAccuracy != nil {
		components["timeliness_persistence"] = clamp01(*row.DirectionalAccuracy)
	} else {
		complete = false
	}
	if !complete {
		return UsefulnessAssessment{LifecycleState: lifecycleForEffectiveness(row), PolicyVersion: UsefulnessPolicyVersion, Components: components}
	}
	score := 10 * (0.25*components["directional_resolution"] + 0.25*components["materialization_strength"] + 0.20*components["adverse_excursion_control"] + 0.20*components["benchmark_relative_performance"] + 0.10*components["timeliness_persistence"])
	return UsefulnessAssessment{LifecycleState: lifecycleForEffectiveness(row), Score: &score, Components: components, PolicyVersion: UsefulnessPolicyVersion}
}

func usefulnessLifecycle(input UsefulnessObservationInput) string {
	state := strings.ToLower(strings.TrimSpace(input.State))
	if state == strings.ToLower(storage.SignalAssertionInvalidated) {
		return LifecycleInvalidated
	}
	if state == strings.ToLower(storage.SignalAssertionExpired) || state == strings.ToLower(storage.SignalAssertionSuperseded) || state == strings.ToLower(storage.SignalAssertionClosed) {
		return LifecycleExpired
	}
	if state == strings.ToLower(storage.SignalAssertionActive) {
		if input.MAE != nil && *input.MAE <= -0.03 {
			return LifecycleAdverseWarning
		}
		return LifecycleCensored
	}
	if input.DirectionalHit != nil && *input.DirectionalHit {
		if positive(input.RelativeReturn) || positive(input.SectorRelativeReturn) {
			return LifecycleOutperformed
		}
		return LifecycleMaterialized
	}
	if input.HorizonSessions <= 1 {
		if input.MAE != nil && *input.MAE <= -0.03 {
			return LifecycleAdverseWarning
		}
		return LifecycleDeveloping
	}
	if input.MFE != nil && *input.MFE > 0 {
		return LifecycleDeveloping
	}
	return LifecycleExpired
}

func lifecycleForEffectiveness(row storage.SignalAssuranceEffectivenessRecord) string {
	if row.SampleSize == 0 && row.CensoredCount > 0 {
		return LifecycleCensored
	}
	if row.InvalidatedCount > 0 && row.MaterializedCount == 0 {
		return LifecycleInvalidated
	}
	if row.MaterializedCount > 0 && row.AverageRelativeReturn != nil && *row.AverageRelativeReturn > 0 {
		return LifecycleOutperformed
	}
	if row.MaterializedCount > 0 || row.DirectionalHits > 0 {
		return LifecycleMaterialized
	}
	if row.SampleSize > 0 {
		return LifecycleExpired
	}
	return LifecycleConfirmed
}

func usefulnessComponents(input UsefulnessObservationInput) (map[string]float64, bool) {
	out := map[string]float64{}
	complete := true
	if input.DirectionalReturn == nil && input.DirectionalHit == nil {
		complete = false
	} else if input.DirectionalReturn != nil {
		out["directional_resolution"] = clamp01((*input.DirectionalReturn + 0.02) / 0.06)
	} else if *input.DirectionalHit {
		out["directional_resolution"] = 1
	} else {
		out["directional_resolution"] = 0
	}
	if input.MFE == nil {
		complete = false
	} else {
		out["materialization_strength"] = clamp01(*input.MFE / 0.05)
	}
	if input.MAE == nil {
		complete = false
	} else {
		out["adverse_excursion_control"] = clamp01(1 - math.Abs(math.Min(*input.MAE, 0))/0.05)
	}
	benchmark, ok := benchmarkComponent(input.RelativeReturn, input.SectorRelativeReturn)
	if !ok {
		complete = false
	} else {
		out["benchmark_relative_performance"] = benchmark
	}
	horizon := input.HorizonSessions
	if horizon <= 0 {
		complete = false
	} else {
		out["timeliness_persistence"] = clamp01(1 - (float64(horizon-1) / 19.0))
	}
	return out, complete
}

func benchmarkComponent(relative, sectorRelative *float64) (float64, bool) {
	values := []float64{}
	if relative != nil {
		values = append(values, clamp01((*relative+0.02)/0.04))
	}
	if sectorRelative != nil {
		values = append(values, clamp01((*sectorRelative+0.02)/0.04))
	}
	if len(values) == 0 {
		return 0, false
	}
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values)), true
}

func positive(value *float64) bool {
	return value != nil && *value > 0
}

func clamp01(value float64) float64 {
	if value < 0 || math.IsNaN(value) {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

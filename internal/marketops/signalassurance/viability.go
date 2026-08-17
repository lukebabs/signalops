package signalassurance

import (
	"fmt"
	"math"

	"github.com/lukebabs/signalops/internal/storage"
)

const (
	ViabilityPolicyVersion             = "saf_viability.v1"
	MinimumViabilitySample             = 30
	ViabilityInsufficientEvidence      = "insufficient_evidence"
	ViabilityBenchmarkPending          = "benchmark_pending"
	ViabilityOutcomeProfilePending     = "outcome_profile_pending"
	ViabilityNotDemonstrated           = "not_demonstrated"
	ViabilityResearchSupportedInSample = "research_supported_in_sample"
)

// ViabilityAssessment is a read-only, conservative interpretation of an
// effectiveness cohort. It cannot promote, mutate, or execute a signal.
type ViabilityAssessment struct {
	State   string
	Reasons []string
}

// AssessViability applies the predeclared SAF-V1 research gates. A cohort can
// only be supported in sample; independent out-of-sample evidence is required
// by a later governed evaluation before it can be considered validated.
func AssessViability(row storage.SignalAssuranceEffectivenessRecord) ViabilityAssessment {
	if row.SampleSize < MinimumViabilitySample {
		return ViabilityAssessment{
			State:   ViabilityInsufficientEvidence,
			Reasons: []string{fmt.Sprintf("%d complete matured observations; at least %d are required before viability can be assessed.", row.SampleSize, MinimumViabilitySample)},
		}
	}
	if row.DirectionalAccuracy == nil || row.AccuracyLowerBound == nil {
		return ViabilityAssessment{
			State:   ViabilityInsufficientEvidence,
			Reasons: []string{"Directional accuracy confidence bounds are unavailable for this matured cohort."},
		}
	}
	if row.AverageRelativeReturn == nil {
		return ViabilityAssessment{
			State:   ViabilityBenchmarkPending,
			Reasons: []string{"Matched benchmark-relative return is unavailable, so incremental value cannot be assessed."},
		}
	}
	if row.AverageMFE == nil || row.AverageMAE == nil {
		return ViabilityAssessment{
			State:   ViabilityOutcomeProfilePending,
			Reasons: []string{"Favorable/adverse excursion evidence is incomplete for this cohort."},
		}
	}
	if *row.AccuracyLowerBound <= .5 {
		return ViabilityAssessment{
			State:   ViabilityNotDemonstrated,
			Reasons: []string{"The lower 95% directional-accuracy bound does not exceed the 50% reference."},
		}
	}
	if *row.AverageRelativeReturn <= 0 {
		return ViabilityAssessment{
			State:   ViabilityNotDemonstrated,
			Reasons: []string{"Average matched benchmark-relative return is not positive."},
		}
	}
	if *row.AverageMFE <= math.Abs(*row.AverageMAE) {
		return ViabilityAssessment{
			State:   ViabilityNotDemonstrated,
			Reasons: []string{"Average favorable excursion does not exceed average adverse excursion."},
		}
	}
	return ViabilityAssessment{
		State: ViabilityResearchSupportedInSample,
		Reasons: []string{
			"Completed in-sample evidence clears the SAF-V1 directional, benchmark-relative, and excursion gates.",
			"An independent frozen out-of-sample cohort is still required before research validation.",
		},
	}
}

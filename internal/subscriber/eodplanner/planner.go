// Package eodplanner provides the deterministic, side-effect-free S2 hot-set
// selection policy. Persistence and worker activation are deliberately separate.
package eodplanner

import (
	"errors"
	"math"
	"sort"
	"strings"
)

const (
	PlannerVersion        = "s2-eod-hot-set-shadow-v1"
	MaximumHotSetCapacity = 1000
)

type Candidate struct {
	GlobalAssetID     string
	EligibilityStatus string
	ActiveSourceRows  int
	BestSourceRank    int
}

type Member struct {
	GlobalAssetID string
	Priority      int
	SourceRank    int
}

type Plan struct {
	Capacity         int
	CandidateCount   int
	EligibleCount    int
	ExcludedCount    int
	ExcludedByReason map[string]int
	Members          []Member
}

func Build(candidates []Candidate, capacity int) (Plan, error) {
	if capacity <= 0 || capacity > MaximumHotSetCapacity {
		return Plan{}, errors.New("hot-set capacity must be between 1 and 1000")
	}
	plan := Plan{Capacity: capacity, CandidateCount: len(candidates), ExcludedByReason: map[string]int{}}
	eligible := make([]Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		candidate.GlobalAssetID = strings.TrimSpace(candidate.GlobalAssetID)
		if candidate.GlobalAssetID == "" {
			return Plan{}, errors.New("hot-set candidate global asset id is required")
		}
		if candidate.EligibilityStatus != "eligible" {
			plan.ExcludedByReason["not_eligible"]++
			continue
		}
		if candidate.ActiveSourceRows <= 0 {
			plan.ExcludedByReason["no_active_source"]++
			continue
		}
		eligible = append(eligible, candidate)
	}
	sort.Slice(eligible, func(i, j int) bool {
		left, right := normalizedRank(eligible[i].BestSourceRank), normalizedRank(eligible[j].BestSourceRank)
		if left != right {
			return left < right
		}
		return eligible[i].GlobalAssetID < eligible[j].GlobalAssetID
	})
	plan.EligibleCount = len(eligible)
	for index, candidate := range eligible {
		if index >= capacity {
			plan.ExcludedByReason["capacity"]++
			continue
		}
		plan.Members = append(plan.Members, Member{GlobalAssetID: candidate.GlobalAssetID, Priority: index + 1, SourceRank: candidate.BestSourceRank})
	}
	for _, count := range plan.ExcludedByReason {
		plan.ExcludedCount += count
	}
	return plan, nil
}

func normalizedRank(rank int) int {
	if rank <= 0 {
		return math.MaxInt
	}
	return rank
}

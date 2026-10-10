package optionsdemand

import (
	"sort"
	"strings"
)

// BuildCandidates ranks aggregate-only database output. It never accepts a
// tenant, list, or subject identity.
func BuildCandidates(cfg Config, candidates []Candidate) (Plan, error) {
	if cfg.MaxSymbols < 1 || cfg.MaxSymbols > 1000 {
		return Plan{}, ErrInvalidConfig
	}
	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.GlobalAssetID) == "" || candidate.HighestTierRank < 0 || candidate.EligibleTenantCount < 1 || candidate.EligibleWatcherCount < 1 || candidate.DeferredSessions < 0 {
			return Plan{}, ErrInvalidConfig
		}
		if _, ok := seen[candidate.GlobalAssetID]; ok {
			return Plan{}, ErrInvalidConfig
		}
		seen[candidate.GlobalAssetID] = struct{}{}
	}
	sort.Slice(candidates, func(left, right int) bool {
		a, b := candidates[left], candidates[right]
		if a.HighestTierRank != b.HighestTierRank {
			return a.HighestTierRank > b.HighestTierRank
		}
		if a.EligibleTenantCount != b.EligibleTenantCount {
			return a.EligibleTenantCount > b.EligibleTenantCount
		}
		if a.EligibleWatcherCount != b.EligibleWatcherCount {
			return a.EligibleWatcherCount > b.EligibleWatcherCount
		}
		if a.DeferredSessions != b.DeferredSessions {
			return a.DeferredSessions > b.DeferredSessions
		}
		return a.GlobalAssetID < b.GlobalAssetID
	})
	plan := Plan{Selected: append([]Candidate(nil), candidates...)}
	if len(plan.Selected) > cfg.MaxSymbols {
		plan.Deferred = append([]Candidate(nil), plan.Selected[cfg.MaxSymbols:]...)
		plan.Selected = append([]Candidate(nil), plan.Selected[:cfg.MaxSymbols]...)
	}
	return plan, nil
}

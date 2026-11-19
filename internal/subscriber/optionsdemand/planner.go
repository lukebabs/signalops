// Package optionsdemand deterministically unions eligible subscriber demand
// before any Options provider work is considered. It is pure and side-effect
// free: callers must persist an immutable snapshot separately, and this
// package never calls a provider or creates a capture request.
package optionsdemand

import (
	"errors"
	"sort"
	"strings"
)

var ErrInvalidConfig = errors.New("options demand planner configuration is invalid")

// Demand is one already-authorized watcher contribution from an immutable
// snapshot. TierRank and DeferredSessions are platform policy inputs, not
// browser-supplied values.
type Demand struct {
	GlobalAssetID    string
	TenantID         string
	Subject          string
	TierRank         int
	DeferredSessions int
}

// Candidate is the non-identifying, per-asset result of deduplicating demand.
// It intentionally does not retain tenant or subject membership information.
type Candidate struct {
	GlobalAssetID        string
	HighestTierRank      int
	EligibleTenantCount  int
	EligibleWatcherCount int
	DeferredSessions     int
}

type Config struct {
	MaxSymbols int
}

type Plan struct {
	Selected []Candidate
	Deferred []Candidate
}

// Build unions eligible demand by global asset and ranks it by product tier,
// tenant reach, watcher reach, deferred age, then asset ID. The output is
// deterministic and contains no duplicate asset.
func Build(cfg Config, demands []Demand) (Plan, error) {
	if cfg.MaxSymbols < 1 || cfg.MaxSymbols > 1000 {
		return Plan{}, ErrInvalidConfig
	}
	type aggregate struct {
		candidate Candidate
		tenants   map[string]struct{}
		watchers  map[string]struct{}
	}
	aggregates := map[string]*aggregate{}
	for _, demand := range demands {
		assetID := strings.TrimSpace(demand.GlobalAssetID)
		tenantID := strings.TrimSpace(demand.TenantID)
		subject := strings.TrimSpace(demand.Subject)
		if assetID == "" || tenantID == "" || subject == "" || demand.TierRank < 0 || demand.DeferredSessions < 0 {
			return Plan{}, ErrInvalidConfig
		}
		entry := aggregates[assetID]
		if entry == nil {
			entry = &aggregate{candidate: Candidate{GlobalAssetID: assetID}, tenants: map[string]struct{}{}, watchers: map[string]struct{}{}}
			aggregates[assetID] = entry
		}
		entry.tenants[tenantID] = struct{}{}
		entry.watchers[tenantID+"\x00"+subject] = struct{}{}
		if demand.TierRank > entry.candidate.HighestTierRank {
			entry.candidate.HighestTierRank = demand.TierRank
		}
		if demand.DeferredSessions > entry.candidate.DeferredSessions {
			entry.candidate.DeferredSessions = demand.DeferredSessions
		}
	}
	candidates := make([]Candidate, 0, len(aggregates))
	for _, entry := range aggregates {
		entry.candidate.EligibleTenantCount = len(entry.tenants)
		entry.candidate.EligibleWatcherCount = len(entry.watchers)
		candidates = append(candidates, entry.candidate)
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
	plan := Plan{Selected: candidates}
	if len(plan.Selected) > cfg.MaxSymbols {
		plan.Deferred = append([]Candidate(nil), plan.Selected[cfg.MaxSymbols:]...)
		plan.Selected = append([]Candidate(nil), plan.Selected[:cfg.MaxSymbols]...)
	}
	return plan, nil
}

package optionsdemand

import "testing"

func TestBuildCandidatesUsesAggregateOnlyValues(t *testing.T) {
	plan, err := BuildCandidates(Config{MaxSymbols: 1}, []Candidate{
		{GlobalAssetID: "aapl", HighestTierRank: 1, EligibleTenantCount: 1, EligibleWatcherCount: 1},
		{GlobalAssetID: "nvda", HighestTierRank: 1, EligibleTenantCount: 1, EligibleWatcherCount: 1},
	})
	if err != nil || len(plan.Selected) != 1 || plan.Selected[0].GlobalAssetID != "aapl" || len(plan.Deferred) != 1 {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
}

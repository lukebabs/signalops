package eodplanner

import "testing"

func TestBuildSelectsEligibleActiveCandidatesByRankThenID(t *testing.T) {
	plan, err := Build([]Candidate{
		{GlobalAssetID: "c", EligibilityStatus: "eligible", ActiveSourceRows: 1, BestSourceRank: 2},
		{GlobalAssetID: "b", EligibilityStatus: "eligible", ActiveSourceRows: 1, BestSourceRank: 1},
		{GlobalAssetID: "a", EligibilityStatus: "eligible", ActiveSourceRows: 1, BestSourceRank: 1},
		{GlobalAssetID: "cold", EligibilityStatus: "discovered", ActiveSourceRows: 1, BestSourceRank: 1},
		{GlobalAssetID: "inactive", EligibilityStatus: "eligible", ActiveSourceRows: 0, BestSourceRank: 1},
	}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if plan.EligibleCount != 3 || plan.ExcludedCount != 3 || len(plan.Members) != 2 {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	if plan.Members[0].GlobalAssetID != "a" || plan.Members[1].GlobalAssetID != "b" {
		t.Fatalf("unexpected deterministic ordering: %+v", plan.Members)
	}
	if plan.ExcludedByReason["not_eligible"] != 1 || plan.ExcludedByReason["no_active_source"] != 1 || plan.ExcludedByReason["capacity"] != 1 {
		t.Fatalf("unexpected exclusions: %+v", plan.ExcludedByReason)
	}
}

func TestBuildRejectsUnsafeCapacity(t *testing.T) {
	for _, capacity := range []int{0, 1001} {
		if _, err := Build([]Candidate{}, capacity); err == nil {
			t.Fatalf("capacity %d should fail", capacity)
		}
	}
}

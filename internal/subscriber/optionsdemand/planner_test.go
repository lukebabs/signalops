package optionsdemand

import "testing"

func TestBuildUnionsDemandAndRanksDeterministically(t *testing.T) {
	plan, err := Build(Config{MaxSymbols: 2}, []Demand{
		{GlobalAssetID: "nvda", TenantID: "tenant-a", Subject: "u1", TierRank: 1},
		{GlobalAssetID: "nvda", TenantID: "tenant-b", Subject: "u2", TierRank: 2},
		{GlobalAssetID: "aapl", TenantID: "tenant-a", Subject: "u1", TierRank: 2},
		{GlobalAssetID: "msft", TenantID: "tenant-a", Subject: "u1", TierRank: 2, DeferredSessions: 3},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Selected) != 2 || len(plan.Deferred) != 1 {
		t.Fatalf("selected=%d deferred=%d", len(plan.Selected), len(plan.Deferred))
	}
	if plan.Selected[0].GlobalAssetID != "nvda" || plan.Selected[0].EligibleTenantCount != 2 || plan.Selected[0].EligibleWatcherCount != 2 {
		t.Fatalf("unexpected first candidate: %+v", plan.Selected[0])
	}
	if plan.Selected[1].GlobalAssetID != "msft" || plan.Deferred[0].GlobalAssetID != "aapl" {
		t.Fatalf("unexpected order: %+v / %+v", plan.Selected, plan.Deferred)
	}
}

func TestBuildRejectsUnsafeInput(t *testing.T) {
	if _, err := Build(Config{}, nil); err != ErrInvalidConfig {
		t.Fatalf("err=%v", err)
	}
	if _, err := Build(Config{MaxSymbols: 1}, []Demand{{GlobalAssetID: "aapl", TenantID: "tenant-a", Subject: "", TierRank: 1}}); err != ErrInvalidConfig {
		t.Fatalf("err=%v", err)
	}
}

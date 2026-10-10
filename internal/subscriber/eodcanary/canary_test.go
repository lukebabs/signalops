package eodcanary

import "testing"

func TestSelectUsesFrozenPlanPriority(t *testing.T) {
	selected, err := Select([]Member{
		{GlobalAssetID: "asset-c", Priority: 3, SourceRank: 3},
		{GlobalAssetID: "asset-a", Priority: 1, SourceRank: 1},
		{GlobalAssetID: "asset-b", Priority: 2, SourceRank: 2},
	}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 2 || selected[0].GlobalAssetID != "asset-a" || selected[1].GlobalAssetID != "asset-b" {
		t.Fatalf("unexpected deterministic selection: %+v", selected)
	}
}

func TestSelectRejectsInvalidOrUnsafeInput(t *testing.T) {
	if _, err := Select(nil, 1); err == nil {
		t.Fatal("expected empty plan to be rejected")
	}
	if _, err := Select([]Member{{GlobalAssetID: "asset-a", Priority: 1}}, 51); err == nil {
		t.Fatal("expected oversized canary to be rejected")
	}
	if _, err := Select([]Member{{GlobalAssetID: "asset-a", Priority: 1}, {GlobalAssetID: "asset-a", Priority: 2}}, 2); err == nil {
		t.Fatal("expected duplicate asset to be rejected")
	}
}

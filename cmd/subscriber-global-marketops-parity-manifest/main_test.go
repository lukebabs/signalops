package main

import "testing"

func TestParseEvidenceKinds(t *testing.T) {
	kinds, err := parseEvidenceKinds("valuation,market_state,valuation")
	if err != nil || len(kinds) != 2 || kinds[0] != "market_state" || kinds[1] != "valuation" {
		t.Fatalf("kinds=%v err=%v", kinds, err)
	}
	if _, err := parseEvidenceKinds("market_state,unknown"); err == nil {
		t.Fatal("unsupported kind accepted")
	}
}

func TestMappingStates(t *testing.T) {
	for matches, want := range map[int]string{0: "unmapped", 1: "mapped", 2: "ambiguous"} {
		got, _ := mappingStates(matches)
		if got != want {
			t.Fatalf("matches=%d got=%s want=%s", matches, got, want)
		}
	}
}

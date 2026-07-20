package main

import "testing"

func TestParseStagesUsesDependencyOrderAndKeepsUnknownForValidation(t *testing.T) {
	got := parseStages("outcome_materialization,preflight,state_materialization,unknown")
	want := []string{"preflight", "state_materialization", "outcome_materialization", "unknown"}
	if len(got) != len(want) {
		t.Fatalf("stages=%v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("stages=%v", got)
		}
	}
}

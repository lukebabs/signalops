package main

import "testing"

func TestParseKindsCanonicalizesAndRejectsUnknownValues(t *testing.T) {
	kinds, err := parseKinds("outcome,market_state,outcome")
	if err != nil {
		t.Fatalf("parse kinds: %v", err)
	}
	if got, want := len(kinds), 2; got != want || kinds[0] != "market_state" || kinds[1] != "outcome" {
		t.Fatalf("unexpected canonical kinds: %#v", kinds)
	}
	if _, err := parseKinds("market_state,not_a_kind"); err == nil {
		t.Fatal("expected unknown kind rejection")
	}
}

func TestMaterializationIdentifiersAreStable(t *testing.T) {
	item := entry{kind: "market_state", algorithmID: "market_state", algorithmVersion: "v1"}
	if got, want := groupKey(item), "market_state\x1fmarket_state\x1fv1"; got != want {
		t.Fatalf("group key = %q, want %q", got, want)
	}
	if got := fingerprint("same-input"); got != fingerprint("same-input") || len(got) != 64 {
		t.Fatalf("fingerprint is not stable: %q", got)
	}
}

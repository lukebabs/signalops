package state

import (
	"strings"
	"testing"
)

func TestNewIdentityIsStableAndKindScoped(t *testing.T) {
	first, err := NewIdentity(IdentityMarketState, "tenant-1", "asset:AAPL", "2026-07-19", "marketops.state.v1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewIdentity(IdentityMarketState, " tenant-1 ", "asset:AAPL", "2026-07-19", "marketops.state.v1")
	if err != nil {
		t.Fatal(err)
	}
	transition, err := NewIdentity(IdentityStateTransition, "tenant-1", "asset:AAPL", "2026-07-19", "marketops.state.v1")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("stable identities differ: %+v != %+v", first, second)
	}
	if first == transition || !strings.HasPrefix(first.ID, "mstate_") || !strings.HasPrefix(first.DeterministicKey, "marketops.market_state.v1:") {
		t.Fatalf("unexpected identities: state=%+v transition=%+v", first, transition)
	}
}

func TestNewIdentityUsesUnambiguousComponents(t *testing.T) {
	first, err := NewIdentity(IdentityEvidence, "ab", "c")
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewIdentity(IdentityEvidence, "a", "bc")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatalf("component boundaries collided: %+v", first)
	}
}

func TestNewIdentityRejectsInvalidInputs(t *testing.T) {
	if _, err := NewIdentity("unknown", "value"); err == nil {
		t.Fatal("expected unsupported kind error")
	}
	if _, err := NewIdentity(IdentityEvidence, ""); err == nil {
		t.Fatal("expected empty component error")
	}
}

func TestCanonicalDimensions(t *testing.T) {
	first, err := CanonicalDimensions([]byte(`{"target_dte":30,"option_type":"put"}`))
	if err != nil {
		t.Fatal(err)
	}
	second, err := CanonicalDimensions([]byte(`{"option_type":"put","target_dte":30.0}`))
	if err != nil {
		t.Fatal(err)
	}
	if first != second || first != `{"option_type":"put","target_dte":30}` {
		t.Fatalf("canonical dimensions differ: %s != %s", first, second)
	}
	if _, err := CanonicalDimensions([]byte(`[]`)); err == nil {
		t.Fatal("expected non-object dimensions error")
	}
}

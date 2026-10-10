package worker

import "testing"

func TestDefinitionsAreLeastPrivilegeAndImmutable(t *testing.T) {
	for _, identity := range Identities() {
		definition, ok := DefinitionFor(identity)
		if !ok || definition.Identity != identity || len(definition.Scopes) == 0 {
			t.Fatalf("definition for %q = %+v, %t", identity, definition, ok)
		}
		seen := map[Scope]struct{}{}
		for _, scope := range definition.Scopes {
			if scope == "" {
				t.Fatalf("identity %q has an empty scope", identity)
			}
			if _, duplicate := seen[scope]; duplicate {
				t.Fatalf("identity %q repeats scope %q", identity, scope)
			}
			seen[scope] = struct{}{}
		}
	}

	first, _ := DefinitionFor(IdentityGlobalEODReconciler)
	first.Scopes[0] = "mutated"
	second, _ := DefinitionFor(IdentityGlobalEODReconciler)
	if second.Scopes[0] == "mutated" {
		t.Fatal("definition scopes must not expose mutable manifest state")
	}
}

func TestOptionsCaptureCannotReserveQuotaOrReadMemberships(t *testing.T) {
	definition, ok := DefinitionFor(IdentityOptionsCapture)
	if !ok {
		t.Fatal("options capture identity is missing")
	}
	for _, forbidden := range []Scope{ScopeQuotaReserve, ScopeEntitlementRead, ScopeListMembershipSnapshotRead, ScopeDecisionAuditWrite} {
		for _, actual := range definition.Scopes {
			if actual == forbidden {
				t.Fatalf("options capture must not have %q", forbidden)
			}
		}
	}
}

func TestUnknownIdentityIsRejected(t *testing.T) {
	if _, ok := DefinitionFor(Identity("subscriber-browser-admin")); ok {
		t.Fatal("unknown identity must not resolve")
	}
}

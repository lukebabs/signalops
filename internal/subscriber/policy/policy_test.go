package policy

import (
	"testing"
	"time"
)

func TestEvaluateDefaultsToEntitlementDeny(t *testing.T) {
	decision := Evaluate(Entitlement{TenantID: "tenant-a"}, Request{
		TenantID: "tenant-a", Capability: CapabilityCatalogSearch, RequestedUnits: 1,
	}, 0)
	if decision.Allowed || decision.Reason != DecisionBlockedEntitlement {
		t.Fatalf("decision = %+v, want blocked entitlement", decision)
	}
}

func TestEvaluateRequiresMatchingTenantAndQuota(t *testing.T) {
	entitlement := Entitlement{
		TenantID: "tenant-a",
		Version:  "provisioning-2026-08-11",
		EnabledCapabilities: map[Capability]bool{
			CapabilityEODActivation: true,
		},
		QuotaLimits: map[Capability]int{
			CapabilityEODActivation: 10,
		},
	}

	mismatched := Evaluate(entitlement, Request{TenantID: "tenant-b", Capability: CapabilityEODActivation, RequestedUnits: 1}, 0)
	if mismatched.Allowed || mismatched.Reason != DecisionBlockedEntitlement {
		t.Fatalf("mismatched decision = %+v, want blocked entitlement", mismatched)
	}

	decision := Evaluate(entitlement, Request{
		TenantID: "tenant-a", Subject: "subject-a", Capability: CapabilityEODActivation, RequestedUnits: 3,
		CorrelationID: "request-1", RequestedAt: time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC),
	}, 7)
	if !decision.Allowed || decision.Reason != DecisionAllowed || decision.RemainingUnits != 0 {
		t.Fatalf("decision = %+v, want allowed decision with no remaining quota", decision)
	}
	if decision.EntitlementVersion != entitlement.Version || decision.PolicyVersion != DefaultPolicyVersion {
		t.Fatalf("decision version = %+v, want entitlement and policy versions", decision)
	}
}

func TestEvaluateDefersWhenQuotaIsMissingOrExceeded(t *testing.T) {
	entitlement := Entitlement{
		TenantID: "tenant-a",
		EnabledCapabilities: map[Capability]bool{
			CapabilityOptionsDemand: true,
		},
		QuotaLimits: map[Capability]int{
			CapabilityOptionsDemand: 2,
		},
	}

	for _, testCase := range []struct {
		name      string
		requested int
		consumed  int
		wantLeft  int
	}{
		{name: "requested units exceed remaining", requested: 2, consumed: 1, wantLeft: 1},
		{name: "already exhausted", requested: 1, consumed: 2, wantLeft: 0},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			decision := Evaluate(entitlement, Request{TenantID: "tenant-a", Capability: CapabilityOptionsDemand, RequestedUnits: testCase.requested}, testCase.consumed)
			if decision.Allowed || decision.Reason != DecisionDeferredQuota || decision.RemainingUnits != testCase.wantLeft {
				t.Fatalf("decision = %+v, want deferred quota with %d remaining", decision, testCase.wantLeft)
			}
		})
	}
}

func TestEvaluateRejectsInvalidRequest(t *testing.T) {
	decision := Evaluate(Entitlement{TenantID: "tenant-a"}, Request{TenantID: "tenant-a", Capability: CapabilityCatalogSearch}, 0)
	if decision.Allowed || decision.Reason != DecisionInvalidRequest {
		t.Fatalf("decision = %+v, want invalid request", decision)
	}
}

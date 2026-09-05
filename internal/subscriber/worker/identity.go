// Package worker defines the least-privilege identity contract for future
// subscriber shared-processing workers. It is a static manifest only: current
// tenant-owned jobs do not consume these definitions yet.
package worker

import "sort"

// Scope is one narrow capability granted to a machine identity. It is not a
// browser role, MarketOps use-case grant, or database superuser permission.
type Scope string

const (
	ScopeCatalogEligibilityRead     Scope = "subscriber.catalog_eligibility.read"
	ScopeCatalogWrite               Scope = "subscriber.catalog.write"
	ScopeCoverageRead               Scope = "subscriber.coverage.read"
	ScopeCoverageWrite              Scope = "subscriber.coverage.write"
	ScopeEntitlementRead            Scope = "subscriber.entitlement.read"
	ScopeListMembershipSnapshotRead Scope = "subscriber.list_membership_snapshot.read"
	ScopeQuotaReserve               Scope = "subscriber.quota.reserve"
	ScopeDecisionAuditWrite         Scope = "subscriber.decision_audit.write"
	ScopeDemandPlanRead             Scope = "subscriber.demand_plan.read"
	ScopeDemandPlanWrite            Scope = "subscriber.demand_plan.write"
	ScopeProviderEODRead            Scope = "subscriber.provider_eod.read"
	ScopeProviderOptionsRead        Scope = "subscriber.provider_options.read"
	ScopeRawEvidenceWrite           Scope = "subscriber.raw_evidence.write"
	ScopeNormalizedEvidenceWrite    Scope = "subscriber.normalized_evidence.write"
	ScopeOptionsCaptureWrite        Scope = "subscriber.options_capture.write"
)

// Identity identifies one future machine principal. A worker may receive only
// the scopes in its definition and must never use a subscriber browser session
// or tenant-administration role.
type Identity string

const (
	IdentityCatalogReferenceSync Identity = "subscriber-catalog-reference-sync"
	IdentityGlobalEODReconciler  Identity = "subscriber-global-eod-reconciler"
	IdentityOptionsDemandPlanner Identity = "subscriber-options-demand-planner"
	IdentityOptionsCapture       Identity = "subscriber-options-capture"
)

// Definition is the immutable intended privilege boundary for one identity.
type Definition struct {
	Identity Identity
	Scopes   []Scope
}

var definitions = map[Identity]Definition{
	IdentityCatalogReferenceSync: {
		Identity: IdentityCatalogReferenceSync,
		Scopes: []Scope{
			ScopeCatalogEligibilityRead,
			ScopeCatalogWrite,
		},
	},
	IdentityGlobalEODReconciler: {
		Identity: IdentityGlobalEODReconciler,
		Scopes: []Scope{
			ScopeCoverageRead,
			ScopeCoverageWrite,
			ScopeProviderEODRead,
			ScopeRawEvidenceWrite,
			ScopeNormalizedEvidenceWrite,
		},
	},
	IdentityOptionsDemandPlanner: {
		Identity: IdentityOptionsDemandPlanner,
		Scopes: []Scope{
			ScopeEntitlementRead,
			ScopeListMembershipSnapshotRead,
			ScopeQuotaReserve,
			ScopeDecisionAuditWrite,
			ScopeDemandPlanWrite,
		},
	},
	IdentityOptionsCapture: {
		Identity: IdentityOptionsCapture,
		Scopes: []Scope{
			ScopeDemandPlanRead,
			ScopeProviderOptionsRead,
			ScopeRawEvidenceWrite,
			ScopeOptionsCaptureWrite,
		},
	},
}

// DefinitionFor returns a copy so a caller cannot mutate the manifest.
func DefinitionFor(identity Identity) (Definition, bool) {
	definition, ok := definitions[identity]
	if !ok {
		return Definition{}, false
	}
	definition.Scopes = append([]Scope(nil), definition.Scopes...)
	sort.Slice(definition.Scopes, func(left, right int) bool { return definition.Scopes[left] < definition.Scopes[right] })
	return definition, true
}

// Identities lists all defined identities in stable order for provisioning
// validation and deployment review.
func Identities() []Identity {
	identities := make([]Identity, 0, len(definitions))
	for identity := range definitions {
		identities = append(identities, identity)
	}
	sort.Slice(identities, func(left, right int) bool { return identities[left] < identities[right] })
	return identities
}

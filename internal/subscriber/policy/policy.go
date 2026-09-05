// Package policy defines the subscriber product-policy decision contract.
//
// It deliberately does not consult the existing MarketOps read/write grants.
// Those grants authorize use-case access; this package determines whether an
// entitled tenant may use a separately metered subscriber capability.
package policy

import (
	"strings"
	"time"
)

const DefaultPolicyVersion = "subscriber-entitlement-v1"

// Capability identifies a separately entitled subscriber product capability.
type Capability string

const (
	CapabilityCatalogSearch Capability = "catalog_search"
	CapabilityEODActivation Capability = "eod_activation"
	CapabilityOptionsDemand Capability = "options_demand"
)

// Entitlement is the provisioned, tenant-level product policy used for one
// decision. Product tier names and commercial limits remain provisioning
// concerns; the evaluator intentionally has no implicit tier defaults.
type Entitlement struct {
	TenantID            string
	Version             string
	EnabledCapabilities map[Capability]bool
	QuotaLimits         map[Capability]int
}

// Request describes one attempted use of a subscriber capability. Subject and
// correlation ID are retained in the decision for durable audit integration.
type Request struct {
	TenantID       string
	Subject        string
	Capability     Capability
	RequestedUnits int
	CorrelationID  string
	RequestedAt    time.Time
}

// DecisionReason explains a deny or allow result without exposing a tenant’s
// other capabilities, memberships, or usage.
type DecisionReason string

const (
	DecisionAllowed            DecisionReason = "allowed"
	DecisionInvalidRequest     DecisionReason = "invalid_request"
	DecisionBlockedEntitlement DecisionReason = "blocked_entitlement"
	DecisionDeferredQuota      DecisionReason = "deferred_quota"
)

// Decision is an audit-ready, deterministic policy result. The caller owns
// usage reservation and durable audit persistence; Evaluate has no side effect.
type Decision struct {
	Allowed            bool
	Reason             DecisionReason
	TenantID           string
	Subject            string
	Capability         Capability
	RequestedUnits     int
	ConsumedUnits      int
	QuotaLimit         int
	RemainingUnits     int
	EntitlementVersion string
	PolicyVersion      string
	CorrelationID      string
	DecidedAt          time.Time
}

// Evaluate applies an explicit tenant entitlement and current usage snapshot.
// It defaults to deny: an absent capability, absent quota, or exhausted quota
// cannot accidentally enable collection or subscriber access.
func Evaluate(entitlement Entitlement, request Request, consumedUnits int) Decision {
	decision := Decision{
		Reason:             DecisionInvalidRequest,
		TenantID:           strings.TrimSpace(request.TenantID),
		Subject:            strings.TrimSpace(request.Subject),
		Capability:         request.Capability,
		RequestedUnits:     request.RequestedUnits,
		ConsumedUnits:      consumedUnits,
		EntitlementVersion: strings.TrimSpace(entitlement.Version),
		PolicyVersion:      DefaultPolicyVersion,
		CorrelationID:      strings.TrimSpace(request.CorrelationID),
		DecidedAt:          request.RequestedAt.UTC(),
	}
	if decision.DecidedAt.IsZero() {
		decision.DecidedAt = time.Now().UTC()
	}
	if decision.TenantID == "" || strings.TrimSpace(string(request.Capability)) == "" || request.RequestedUnits <= 0 || consumedUnits < 0 {
		return decision
	}
	if strings.TrimSpace(entitlement.TenantID) != decision.TenantID || !entitlement.EnabledCapabilities[request.Capability] {
		decision.Reason = DecisionBlockedEntitlement
		return decision
	}
	decision.QuotaLimit = entitlement.QuotaLimits[request.Capability]
	if decision.QuotaLimit <= 0 || consumedUnits >= decision.QuotaLimit || request.RequestedUnits > decision.QuotaLimit-consumedUnits {
		decision.Reason = DecisionDeferredQuota
		decision.RemainingUnits = max(0, decision.QuotaLimit-consumedUnits)
		return decision
	}
	decision.Allowed = true
	decision.Reason = DecisionAllowed
	decision.RemainingUnits = decision.QuotaLimit - consumedUnits - request.RequestedUnits
	return decision
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}

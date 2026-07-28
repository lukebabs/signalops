package platformregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/adapters/marketdata/massive"
	"github.com/lukebabs/signalops/internal/storage"
)

const (
	massiveRawIngestPipelineKey       = "marketops.massive.raw_ingest"
	normalizedEventQualityPolicyKey   = "signalops.normalized_event_quality"
	normalizedEventQualityPolicyScope = "normalized_event"
)

type DefinitionLister interface {
	ListPlatformPrimitiveDefinitions(context.Context, storage.PlatformPrimitiveDefinitionFilter) ([]storage.PlatformPrimitiveDefinitionRecord, error)
}

// DefinitionValidationError identifies an event whose declared definition versions
// are incomplete or do not match the active registry entries.
type DefinitionValidationError struct{ Err error }

func (e DefinitionValidationError) Error() string  { return e.Err.Error() }
func (e DefinitionValidationError) Unwrap() error  { return e.Err }
func (DefinitionValidationError) InvalidRawEvent() {}

type qualityStateGateContract struct {
	PolicyKind    string   `json:"policy_kind"`
	Scope         string   `json:"scope"`
	AllowedStates []string `json:"allowed_states"`
	DefaultState  string   `json:"default_state"`
	FailureAction string   `json:"failure_action"`
	FailsClosed   bool     `json:"fails_closed"`
}

func normalizedEventQualityPolicy(ctx context.Context, lister DefinitionLister, tenantID string) (storage.PlatformPrimitiveDefinitionRecord, qualityStateGateContract, error) {
	policy, err := oneActiveByKey(ctx, lister, tenantID, "policy", normalizedEventQualityPolicyKey)
	if err != nil {
		return storage.PlatformPrimitiveDefinitionRecord{}, qualityStateGateContract{}, err
	}
	var contract qualityStateGateContract
	if err := json.Unmarshal(policy.ContractJSON, &contract); err != nil {
		return storage.PlatformPrimitiveDefinitionRecord{}, qualityStateGateContract{}, fmt.Errorf("decode normalized event quality policy contract: %w", err)
	}
	if contract.PolicyKind != "quality_state_gate" || contract.Scope != normalizedEventQualityPolicyScope || !contract.FailsClosed || strings.TrimSpace(contract.DefaultState) == "" || len(contract.AllowedStates) == 0 {
		return storage.PlatformPrimitiveDefinitionRecord{}, qualityStateGateContract{}, fmt.Errorf("normalized event quality policy contract is invalid")
	}
	if !qualityStateAllowed(contract.AllowedStates, contract.DefaultState) {
		return storage.PlatformPrimitiveDefinitionRecord{}, qualityStateGateContract{}, fmt.Errorf("normalized event quality policy default state %q is not allowed", contract.DefaultState)
	}
	return policy, contract, nil
}

func qualityStateAllowed(allowed []string, state string) bool {
	state = strings.TrimSpace(state)
	for _, value := range allowed {
		if strings.TrimSpace(value) == state {
			return true
		}
	}
	return false
}

type MassiveRawEventDefinitionValidator struct{ Lister DefinitionLister }

// MassiveNormalizedEventDefinitionValidator verifies the normalized-ledger
// provenance consumed by downstream MarketOps materializers. It is deliberately
// limited to Massive datasets that have a registry mapping.
type MassiveNormalizedEventDefinitionValidator struct{ Lister DefinitionLister }

// ValidateRawEvent verifies definition-version provenance for Massive scheduled-pull
// messages. Non-Massive messages remain outside this targeted rollout.
func (v MassiveRawEventDefinitionValidator) ValidateRawEvent(ctx context.Context, value []byte) error {
	if v.Lister == nil {
		return fmt.Errorf("platform primitive definition repository is required")
	}
	var raw struct {
		TenantID      string         `json:"tenant_id"`
		SourceID      string         `json:"source_id"`
		SourceAdapter string         `json:"source_adapter"`
		IngestionMode string         `json:"ingestion_mode"`
		Dataset       string         `json:"dataset"`
		Metadata      map[string]any `json:"metadata"`
	}
	if err := json.Unmarshal(value, &raw); err != nil {
		return nil // BuildEvent returns the canonical invalid-envelope error.
	}
	if raw.SourceAdapter != massive.AdapterID || raw.IngestionMode != "scheduled_pull" {
		return nil
	}
	return validateMassiveEventProvenance(ctx, v.Lister, strings.TrimSpace(raw.TenantID), strings.TrimSpace(raw.SourceID), raw.Dataset, raw.Metadata, "raw")
}

// ValidateNormalizedEvent checks the definition versions and quality envelope
// retained in a normalized ledger record. Other adapters and unmapped Massive
// datasets are outside the targeted rollout.
func (v MassiveNormalizedEventDefinitionValidator) ValidateNormalizedEvent(ctx context.Context, event storage.NormalizedEventLedgerRecord) error {
	if v.Lister == nil {
		return fmt.Errorf("platform primitive definition repository is required")
	}
	if event.SourceAdapter != massive.AdapterID {
		return nil
	}
	if _, ok := massiveDatasetDefinitionKey(event.Dataset); !ok {
		return nil
	}
	metadata := map[string]any{}
	if err := json.Unmarshal(event.MetadataJSON, &metadata); err != nil {
		return invalidDefinition(fmt.Errorf("decode Massive normalized event metadata: %w", err))
	}
	return validateMassiveEventProvenance(ctx, v.Lister, strings.TrimSpace(event.TenantID), strings.TrimSpace(event.SourceID), event.Dataset, metadata, "normalized")
}

func validateMassiveEventProvenance(ctx context.Context, lister DefinitionLister, tenantID, sourceID, datasetName string, metadata map[string]any, eventKind string) error {
	datasetKey, ok := massiveDatasetDefinitionKey(datasetName)
	if !ok {
		return invalidDefinition(fmt.Errorf("no platform dataset definition mapping for Massive dataset %q", datasetName))
	}
	versions, ok := metadata["platform_definition_versions"].(map[string]any)
	if !ok {
		return invalidDefinition(fmt.Errorf("Massive %s event platform_definition_versions metadata is required", eventKind))
	}
	source, err := oneActiveByID(ctx, lister, tenantID, "source", sourceID)
	if err != nil {
		return classifyDefinitionLookupError(err)
	}
	pipeline, err := oneActiveByKey(ctx, lister, tenantID, "pipeline", massiveRawIngestPipelineKey)
	if err != nil {
		return classifyDefinitionLookupError(err)
	}
	dataset, err := oneActiveByKey(ctx, lister, tenantID, "dataset", datasetKey)
	if err != nil {
		return classifyDefinitionLookupError(err)
	}
	for _, definition := range []storage.PlatformPrimitiveDefinitionRecord{source, pipeline, dataset} {
		if err := requireActiveReferencedPolicies(ctx, lister, definition); err != nil {
			return classifyDefinitionLookupError(err)
		}
	}
	for key, expected := range map[string]string{"source": source.Version, "pipeline": pipeline.Version, "dataset": dataset.Version} {
		actual, _ := versions[key].(string)
		if strings.TrimSpace(actual) != expected {
			return invalidDefinition(fmt.Errorf("Massive %s event %s definition version %q does not match active version %q", eventKind, key, strings.TrimSpace(actual), expected))
		}
	}
	qualityPolicy, qualityContract, err := normalizedEventQualityPolicy(ctx, lister, tenantID)
	if err != nil {
		return classifyDefinitionLookupError(err)
	}
	quality, ok := metadata["quality"].(map[string]any)
	if !ok {
		return invalidDefinition(fmt.Errorf("Massive %s event quality metadata is required", eventKind))
	}
	policyID, _ := quality["quality_policy_id"].(string)
	policyVersion, _ := quality["quality_policy_version"].(string)
	state, _ := quality["quality_state"].(string)
	if strings.TrimSpace(policyID) != qualityPolicy.DefinitionID || strings.TrimSpace(policyVersion) != qualityPolicy.Version {
		return invalidDefinition(fmt.Errorf("Massive %s event quality policy does not match active normalized-event quality policy", eventKind))
	}
	if !qualityStateAllowed(qualityContract.AllowedStates, state) {
		return invalidDefinition(fmt.Errorf("Massive %s event quality state %q is not allowed by the active normalized-event quality policy", eventKind, strings.TrimSpace(state)))
	}
	return nil
}

func massiveDatasetDefinitionKey(dataset string) (string, bool) {
	switch strings.TrimSpace(dataset) {
	case massive.DatasetEquityEODPrices:
		return "market.equity.daily_bar", true
	case massive.DatasetOptionsContractsDaily:
		return "market.options.contracts_daily", true
	default:
		return "", false
	}
}

func invalidDefinition(err error) error { return DefinitionValidationError{Err: err} }

func classifyDefinitionLookupError(err error) error {
	if strings.HasPrefix(err.Error(), "no active ") || strings.HasPrefix(err.Error(), "ambiguous active ") {
		return invalidDefinition(err)
	}
	return err
}

// ResolveMassiveScheduledPull requires one active source, pipeline, and dataset definition
// for every selected Massive dataset. The returned versions are embedded in emitted events.
func ResolveMassiveScheduledPull(ctx context.Context, lister DefinitionLister, tenantID, sourceID string, includeEquity, includeOptions bool) (map[string]string, error) {
	if lister == nil {
		return nil, fmt.Errorf("platform primitive definition repository is required")
	}
	tenantID = strings.TrimSpace(tenantID)
	sourceID = strings.TrimSpace(sourceID)
	if tenantID == "" || sourceID == "" {
		return nil, fmt.Errorf("platform registry tenant id and source id are required")
	}
	versions := map[string]string{}
	source, err := oneActiveByID(ctx, lister, tenantID, "source", sourceID)
	if err != nil {
		return nil, err
	}
	if err := requireActiveReferencedPolicies(ctx, lister, source); err != nil {
		return nil, err
	}
	versions["source"] = source.Version
	pipeline, err := oneActiveByKey(ctx, lister, tenantID, "pipeline", massiveRawIngestPipelineKey)
	if err != nil {
		return nil, err
	}
	if err := requireActiveReferencedPolicies(ctx, lister, pipeline); err != nil {
		return nil, err
	}
	versions["pipeline"] = pipeline.Version
	for _, requirement := range []struct {
		enabled bool
		dataset string
		key     string
	}{
		{includeEquity, massive.DatasetEquityEODPrices, "market.equity.daily_bar"},
		{includeOptions, massive.DatasetOptionsContractsDaily, "market.options.contracts_daily"},
	} {
		if !requirement.enabled {
			continue
		}
		definition, err := oneActiveByKey(ctx, lister, tenantID, "dataset", requirement.key)
		if err != nil {
			return nil, err
		}
		if err := requireActiveReferencedPolicies(ctx, lister, definition); err != nil {
			return nil, err
		}
		versions[requirement.dataset] = definition.Version
	}
	qualityPolicy, qualityContract, err := normalizedEventQualityPolicy(ctx, lister, tenantID)
	if err != nil {
		return nil, err
	}
	versions["normalized_event_quality_policy_id"] = qualityPolicy.DefinitionID
	versions["normalized_event_quality_policy_version"] = qualityPolicy.Version
	versions["normalized_event_quality_default_state"] = qualityContract.DefaultState
	return versions, nil
}

func requireActiveReferencedPolicies(ctx context.Context, lister DefinitionLister, definition storage.PlatformPrimitiveDefinitionRecord) error {
	for _, policyID := range []string{definition.QualityPolicyID, definition.RetentionPolicyID, definition.LineagePolicyID} {
		policyID = strings.TrimSpace(policyID)
		if policyID == "" {
			continue
		}
		if _, err := oneActiveByID(ctx, lister, definition.TenantID, "policy", policyID); err != nil {
			return fmt.Errorf("resolve active policy %q for %s definition %q: %w", policyID, definition.PrimitiveType, definition.DefinitionKey, err)
		}
	}
	return nil
}

func oneActiveByID(ctx context.Context, lister DefinitionLister, tenantID, primitiveType, definitionID string) (storage.PlatformPrimitiveDefinitionRecord, error) {
	records, err := lister.ListPlatformPrimitiveDefinitions(ctx, storage.PlatformPrimitiveDefinitionFilter{TenantID: tenantID, PrimitiveType: primitiveType, Status: storage.PlatformDefinitionStatusActive, Limit: 200})
	if err != nil {
		return storage.PlatformPrimitiveDefinitionRecord{}, fmt.Errorf("list active %s definitions: %w", primitiveType, err)
	}
	matched := make([]storage.PlatformPrimitiveDefinitionRecord, 0, 1)
	for _, record := range records {
		if record.DefinitionID == definitionID {
			matched = append(matched, record)
		}
	}
	return exactlyOneActive(primitiveType, definitionID, matched)
}

func oneActiveByKey(ctx context.Context, lister DefinitionLister, tenantID, primitiveType, definitionKey string) (storage.PlatformPrimitiveDefinitionRecord, error) {
	records, err := lister.ListPlatformPrimitiveDefinitions(ctx, storage.PlatformPrimitiveDefinitionFilter{TenantID: tenantID, PrimitiveType: primitiveType, DefinitionKey: definitionKey, Status: storage.PlatformDefinitionStatusActive, Limit: 200})
	if err != nil {
		return storage.PlatformPrimitiveDefinitionRecord{}, fmt.Errorf("list active %s definition %q: %w", primitiveType, definitionKey, err)
	}
	return exactlyOneActive(primitiveType, definitionKey, records)
}

func exactlyOneActive(primitiveType, identity string, records []storage.PlatformPrimitiveDefinitionRecord) (storage.PlatformPrimitiveDefinitionRecord, error) {
	if len(records) == 0 {
		return storage.PlatformPrimitiveDefinitionRecord{}, fmt.Errorf("no active %s definition for %q", primitiveType, identity)
	}
	if len(records) != 1 {
		return storage.PlatformPrimitiveDefinitionRecord{}, fmt.Errorf("ambiguous active %s definitions for %q", primitiveType, identity)
	}
	return records[0], nil
}

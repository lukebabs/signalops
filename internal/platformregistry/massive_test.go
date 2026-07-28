package platformregistry

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/lukebabs/signalops/internal/adapters/marketdata/massive"
	"github.com/lukebabs/signalops/internal/storage"
)

type fakeDefinitionLister struct {
	records []storage.PlatformPrimitiveDefinitionRecord
	err     error
}

func (f fakeDefinitionLister) ListPlatformPrimitiveDefinitions(_ context.Context, filter storage.PlatformPrimitiveDefinitionFilter) ([]storage.PlatformPrimitiveDefinitionRecord, error) {
	if f.err != nil {
		return nil, f.err
	}
	result := []storage.PlatformPrimitiveDefinitionRecord{}
	for _, record := range f.records {
		if record.TenantID != filter.TenantID || record.PrimitiveType != filter.PrimitiveType || record.Status != filter.Status {
			continue
		}
		if filter.DefinitionKey != "" && record.DefinitionKey != filter.DefinitionKey {
			continue
		}
		result = append(result, record)
	}
	return result, nil
}

func normalizedEventQualityPolicyRecord(tenantID string) storage.PlatformPrimitiveDefinitionRecord {
	return storage.PlatformPrimitiveDefinitionRecord{
		TenantID: tenantID, PrimitiveType: "policy", DefinitionID: "policy_signalops_normalized_event_quality_v1",
		DefinitionKey: normalizedEventQualityPolicyKey, Status: storage.PlatformDefinitionStatusActive, Version: "1.0.0",
		ContractJSON: []byte(`{"policy_kind":"quality_state_gate","scope":"normalized_event","allowed_states":["usable","invalid"],"default_state":"usable","failure_action":"dlq","fails_closed":true}`),
	}
}

func TestResolveMassiveScheduledPull(t *testing.T) {
	versions, err := ResolveMassiveScheduledPull(context.Background(), fakeDefinitionLister{records: []storage.PlatformPrimitiveDefinitionRecord{
		{TenantID: "tenant-1", PrimitiveType: "source", DefinitionID: "src-massive", Status: storage.PlatformDefinitionStatusActive, Version: "1.0.0"},
		{TenantID: "tenant-1", PrimitiveType: "pipeline", DefinitionKey: massiveRawIngestPipelineKey, Status: storage.PlatformDefinitionStatusActive, Version: "1.1.0"},
		{TenantID: "tenant-1", PrimitiveType: "dataset", DefinitionKey: "market.equity.daily_bar", Status: storage.PlatformDefinitionStatusActive, Version: "2.0.0"},
		{TenantID: "tenant-1", PrimitiveType: "dataset", DefinitionKey: "market.options.contracts_daily", Status: storage.PlatformDefinitionStatusActive, Version: "3.0.0"},
		normalizedEventQualityPolicyRecord("tenant-1"),
	}}, "tenant-1", "src-massive", true, true)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if versions["source"] != "1.0.0" || versions["pipeline"] != "1.1.0" || versions[massive.DatasetEquityEODPrices] != "2.0.0" || versions[massive.DatasetOptionsContractsDaily] != "3.0.0" || versions["normalized_event_quality_policy_id"] != "policy_signalops_normalized_event_quality_v1" || versions["normalized_event_quality_default_state"] != "usable" {
		t.Fatalf("versions = %#v", versions)
	}
}

func TestResolveMassiveScheduledPullRejectsAmbiguousDataset(t *testing.T) {
	_, err := ResolveMassiveScheduledPull(context.Background(), fakeDefinitionLister{records: []storage.PlatformPrimitiveDefinitionRecord{
		{TenantID: "tenant-1", PrimitiveType: "source", DefinitionID: "src-massive", Status: storage.PlatformDefinitionStatusActive, Version: "1.0.0"},
		{TenantID: "tenant-1", PrimitiveType: "pipeline", DefinitionKey: massiveRawIngestPipelineKey, Status: storage.PlatformDefinitionStatusActive, Version: "1.0.0"},
		{TenantID: "tenant-1", PrimitiveType: "dataset", DefinitionKey: "market.equity.daily_bar", Status: storage.PlatformDefinitionStatusActive, Version: "1.0.0"},
		{TenantID: "tenant-1", PrimitiveType: "dataset", DefinitionKey: "market.equity.daily_bar", Status: storage.PlatformDefinitionStatusActive, Version: "1.1.0"},
	}}, "tenant-1", "src-massive", true, false)
	if err == nil {
		t.Fatal("expected ambiguous active dataset error")
	}
}

func TestMassiveRawEventDefinitionValidator(t *testing.T) {
	lister := fakeDefinitionLister{records: []storage.PlatformPrimitiveDefinitionRecord{
		{TenantID: "tenant-1", PrimitiveType: "source", DefinitionID: "src-massive", Status: storage.PlatformDefinitionStatusActive, Version: "1.0.0"},
		{TenantID: "tenant-1", PrimitiveType: "pipeline", DefinitionKey: massiveRawIngestPipelineKey, Status: storage.PlatformDefinitionStatusActive, Version: "1.0.0"},
		{TenantID: "tenant-1", PrimitiveType: "dataset", DefinitionKey: "market.equity.daily_bar", Status: storage.PlatformDefinitionStatusActive, Version: "1.0.0"},
		normalizedEventQualityPolicyRecord("tenant-1"),
	}}
	validator := MassiveRawEventDefinitionValidator{Lister: lister}
	value, err := json.Marshal(map[string]any{
		"tenant_id": "tenant-1", "source_id": "src-massive", "source_adapter": massive.AdapterID,
		"ingestion_mode": "scheduled_pull", "dataset": massive.DatasetEquityEODPrices,
		"metadata": map[string]any{"platform_definition_versions": map[string]string{"source": "1.0.0", "pipeline": "1.0.0", "dataset": "1.0.0"}, "quality": map[string]string{"quality_state": "usable", "quality_policy_id": "policy_signalops_normalized_event_quality_v1", "quality_policy_version": "1.0.0"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := validator.ValidateRawEvent(context.Background(), value); err != nil {
		t.Fatalf("validate: %v", err)
	}

	value, err = json.Marshal(map[string]any{
		"tenant_id": "tenant-1", "source_id": "src-massive", "source_adapter": massive.AdapterID,
		"ingestion_mode": "scheduled_pull", "dataset": massive.DatasetEquityEODPrices,
		"metadata": map[string]any{"platform_definition_versions": map[string]string{"source": "1.0.0", "pipeline": "1.0.0", "dataset": "0.9.0"}, "quality": map[string]string{"quality_state": "usable", "quality_policy_id": "policy_signalops_normalized_event_quality_v1", "quality_policy_version": "1.0.0"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	err = validator.ValidateRawEvent(context.Background(), value)
	var invalid DefinitionValidationError
	if !errors.As(err, &invalid) {
		t.Fatalf("expected definition validation error, got %v", err)
	}

	value, err = json.Marshal(map[string]any{
		"tenant_id": "tenant-1", "source_id": "src-massive", "source_adapter": massive.AdapterID,
		"ingestion_mode": "scheduled_pull", "dataset": massive.DatasetEquityEODPrices,
		"metadata": map[string]any{"platform_definition_versions": map[string]string{"source": "1.0.0", "pipeline": "1.0.0", "dataset": "1.0.0"}, "quality": map[string]string{"quality_state": "partial", "quality_policy_id": "policy_signalops_normalized_event_quality_v1", "quality_policy_version": "1.0.0"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	err = validator.ValidateRawEvent(context.Background(), value)
	if !errors.As(err, &invalid) {
		t.Fatalf("expected disallowed quality state error, got %v", err)
	}
}

func TestMassiveRawEventDefinitionValidatorReturnsLookupFailureForRetry(t *testing.T) {
	validator := MassiveRawEventDefinitionValidator{Lister: fakeDefinitionLister{err: errors.New("database unavailable")}}
	value := []byte(`{"tenant_id":"tenant-1","source_id":"src-massive","source_adapter":"market_data.massive","ingestion_mode":"scheduled_pull","dataset":"equity_eod_prices","metadata":{"platform_definition_versions":{"source":"1.0.0","pipeline":"1.0.0","dataset":"1.0.0"}}}`)
	err := validator.ValidateRawEvent(context.Background(), value)
	var invalid DefinitionValidationError
	if err == nil || errors.As(err, &invalid) {
		t.Fatalf("expected retriable lookup error, got %v", err)
	}
}

func TestResolveMassiveScheduledPullRequiresActiveReferencedPolicies(t *testing.T) {
	_, err := ResolveMassiveScheduledPull(context.Background(), fakeDefinitionLister{records: []storage.PlatformPrimitiveDefinitionRecord{
		{TenantID: "tenant-1", PrimitiveType: "source", DefinitionID: "src-massive", Status: storage.PlatformDefinitionStatusActive, Version: "1.0.0", QualityPolicyID: "policy-source-quality"},
		{TenantID: "tenant-1", PrimitiveType: "pipeline", DefinitionKey: massiveRawIngestPipelineKey, Status: storage.PlatformDefinitionStatusActive, Version: "1.0.0"},
		{TenantID: "tenant-1", PrimitiveType: "dataset", DefinitionKey: "market.equity.daily_bar", Status: storage.PlatformDefinitionStatusActive, Version: "1.0.0"},
	}}, "tenant-1", "src-massive", true, false)
	if err == nil {
		t.Fatal("expected missing active policy error")
	}
}

func TestMassiveNormalizedEventDefinitionValidator(t *testing.T) {
	lister := fakeDefinitionLister{records: []storage.PlatformPrimitiveDefinitionRecord{
		{TenantID: "tenant-1", PrimitiveType: "source", DefinitionID: "src-massive", Status: storage.PlatformDefinitionStatusActive, Version: "1.0.0"},
		{TenantID: "tenant-1", PrimitiveType: "pipeline", DefinitionKey: massiveRawIngestPipelineKey, Status: storage.PlatformDefinitionStatusActive, Version: "1.0.0"},
		{TenantID: "tenant-1", PrimitiveType: "dataset", DefinitionKey: "market.equity.daily_bar", Status: storage.PlatformDefinitionStatusActive, Version: "1.0.0"},
		normalizedEventQualityPolicyRecord("tenant-1"),
	}}
	metadata, err := json.Marshal(map[string]any{"platform_definition_versions": map[string]string{"source": "1.0.0", "pipeline": "1.0.0", "dataset": "1.0.0"}, "quality": map[string]string{"quality_state": "usable", "quality_policy_id": "policy_signalops_normalized_event_quality_v1", "quality_policy_version": "1.0.0"}})
	if err != nil {
		t.Fatal(err)
	}
	event := storage.NormalizedEventLedgerRecord{TenantID: "tenant-1", SourceID: "src-massive", SourceAdapter: massive.AdapterID, Dataset: massive.DatasetEquityEODPrices, MetadataJSON: metadata}
	validator := MassiveNormalizedEventDefinitionValidator{Lister: lister}
	if err := validator.ValidateNormalizedEvent(context.Background(), event); err != nil {
		t.Fatalf("validate: %v", err)
	}
	event.MetadataJSON = []byte(`{"platform_definition_versions":{"source":"1.0.0","pipeline":"1.0.0","dataset":"0.9.0"},"quality":{"quality_state":"usable","quality_policy_id":"policy_signalops_normalized_event_quality_v1","quality_policy_version":"1.0.0"}}`)
	err = validator.ValidateNormalizedEvent(context.Background(), event)
	var invalid DefinitionValidationError
	if !errors.As(err, &invalid) {
		t.Fatalf("expected definition validation error, got %v", err)
	}
}

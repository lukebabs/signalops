package postgres

import "testing"

func TestSameImmutableLineageCanonicalizesJSONBMetadata(t *testing.T) {
	existing := []byte(`{"delivery_id":"delivery-1","mapping_key":"rfc5424","tenant_id":"tenant-local","processing_run_id":"run-1","payload_hash":"abc","ingress_event_id":"ing-1"}`)
	incoming := []byte(`{"tenant_id":"tenant-local","ingress_event_id":"ing-1","payload_hash":"abc","mapping_key":"rfc5424"}`)
	if !sameImmutableLineage(existing, incoming) {
		t.Fatal("sameImmutableLineage() = false, want true for equivalent immutable lineage")
	}
}

func TestSameImmutableLineageRejectsChangedImmutableField(t *testing.T) {
	existing := []byte(`{"tenant_id":"tenant-local","ingress_event_id":"ing-1","payload_hash":"abc","connector_id":"opnsense","processing_run_id":"run-1","delivery_id":"delivery-1"}`)
	incoming := []byte(`{"tenant_id":"tenant-local","ingress_event_id":"ing-1","payload_hash":"def","connector_id":"opnsense"}`)
	if sameImmutableLineage(existing, incoming) {
		t.Fatal("sameImmutableLineage() = true, want false for changed immutable lineage")
	}
}

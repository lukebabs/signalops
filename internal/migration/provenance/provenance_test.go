package provenance

import (
	"strings"
	"testing"
)

func TestValidateAcceptsCompleteApprovedRecord(t *testing.T) {
	if err := Validate(completeRecord()); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateRejectsMissingRequiredEvidence(t *testing.T) {
	record := completeRecord()
	delete(record.Fields, "stream_key")
	if err := Validate(record); err == nil || !strings.Contains(err.Error(), "stream_key provenance is required") {
		t.Fatalf("Validate() error = %v, want missing stream_key provenance", err)
	}
}

func TestValidateRejectsUnapprovedFallback(t *testing.T) {
	record := completeRecord()
	field := record.Fields["payload_hash"]
	field.Status = StatusFallback
	field.ContractReference = ""
	record.Fields["payload_hash"] = field
	if err := Validate(record); err == nil || !strings.Contains(err.Error(), "contract_reference") {
		t.Fatalf("Validate() error = %v, want rejected fallback", err)
	}
}

func TestValidateRejectsInventedExportField(t *testing.T) {
	record := completeRecord()
	record.Fields["gateway_local_offset"] = record.Fields["stream_key"]
	if err := Validate(record); err == nil || !strings.Contains(err.Error(), "not an approved export field") {
		t.Fatalf("Validate() error = %v, want rejected invented field", err)
	}
}

func completeRecord() Record {
	field := Field{
		EvidencePath:      "raw_event.metadata.source_id",
		RetentionBoundary: "migration evidence retention",
		DataType:          "string",
		Canonicalization:  "source-contract-v1",
		OrderingScope:     "source stream key",
		SourceOfTruth:     "source-owner approved Gateway evidence",
		Status:            StatusAvailable,
	}
	return Record{
		MigrationID: "mig-test-20260729",
		TenantID:    "tenant-test",
		Version:     "1.0.0",
		Fields: map[string]Field{
			"source_message_id":         field,
			"stream_key":                field,
			"source_sequence_or_cursor": field,
			"source_occurred_at":        field,
			"payload_hash":              field,
			"mapping_output_hash":       field,
			"duplicate_classification":  field,
			"continuity_annotation":     field,
		},
	}
}

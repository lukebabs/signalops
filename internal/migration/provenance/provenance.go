// Package provenance validates the source-owner/SignalOps field record that
// must exist before a Phase 0 Gateway observation export can be implemented.
package provenance

import (
	"fmt"
	"strings"
)

const (
	StatusAvailable = "available"
	StatusFallback  = "fallback"
	StatusDeferred  = "deferred"
)

var requiredFields = []string{
	"source_message_id",
	"stream_key",
	"source_sequence_or_cursor",
	"source_occurred_at",
	"payload_hash",
	"mapping_output_hash",
	"duplicate_classification",
	"continuity_annotation",
}

// Field identifies one contract export field and the approved evidence that
// gives it meaning. The validator intentionally accepts no inferred defaults.
type Field struct {
	EvidencePath      string
	RetentionBoundary string
	DataType          string
	Canonicalization  string
	OrderingScope     string
	SourceOfTruth     string
	Status            string
	ContractReference string
}

// Record is the minimum source-specific hand-off accepted by SignalOps and the
// source owner before the Gateway observation export is assigned for coding.
type Record struct {
	MigrationID string
	TenantID    string
	Version     string
	Fields      map[string]Field
}

// Validate rejects incomplete, ambiguous, or unapproved provenance records.
// A fallback/deferred field is permitted only when the accepted source contract
// is named explicitly; later export code must use that status rather than guess.
func Validate(record Record) error {
	if strings.TrimSpace(record.MigrationID) == "" {
		return fmt.Errorf("migration_id is required")
	}
	if strings.TrimSpace(record.TenantID) == "" {
		return fmt.Errorf("tenant_id is required")
	}
	if strings.TrimSpace(record.Version) == "" {
		return fmt.Errorf("version is required")
	}
	for _, name := range requiredFields {
		field, ok := record.Fields[name]
		if !ok {
			return fmt.Errorf("%s provenance is required", name)
		}
		if err := validateField(name, field); err != nil {
			return err
		}
	}
	for name := range record.Fields {
		if !isRequiredField(name) {
			return fmt.Errorf("%s is not an approved export field", name)
		}
	}
	return nil
}

func validateField(name string, field Field) error {
	values := map[string]string{
		"evidence_path":      field.EvidencePath,
		"retention_boundary": field.RetentionBoundary,
		"data_type":          field.DataType,
		"canonicalization":   field.Canonicalization,
		"ordering_scope":     field.OrderingScope,
		"source_of_truth":    field.SourceOfTruth,
	}
	for label, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s %s is required", name, label)
		}
	}
	switch field.Status {
	case StatusAvailable:
	case StatusFallback, StatusDeferred:
		if strings.TrimSpace(field.ContractReference) == "" {
			return fmt.Errorf("%s %s status requires contract_reference", name, field.Status)
		}
	default:
		return fmt.Errorf("%s status must be available, fallback, or deferred", name)
	}
	return nil
}

func isRequiredField(name string) bool {
	for _, required := range requiredFields {
		if name == required {
			return true
		}
	}
	return false
}

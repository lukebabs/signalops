package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func extractMarketOpsDSMGraphProposals(artifact storage.MarketOpsDSMArtifactRecord) ([]storage.MarketOpsDSMGraphProposalRecord, error) {
	if artifact.AppID != "marketops" || artifact.Domain != "market_data" || artifact.UseCase != "daily_market_surveillance" || artifact.ArtifactType != "marketops.dsm.signal_artifact.v1" {
		return nil, nil
	}
	if len(artifact.GraphTargetsJSON) == 0 {
		return nil, nil
	}
	var candidates []map[string]any
	if err := json.Unmarshal(artifact.GraphTargetsJSON, &candidates); err != nil {
		return nil, fmt.Errorf("extract marketops dsm graph targets: %w", err)
	}
	records := []storage.MarketOpsDSMGraphProposalRecord{}
	for _, candidate := range candidates {
		candidateType := firstMapString(candidate, "type")
		record := storage.MarketOpsDSMGraphProposalRecord{
			TenantID:            artifact.TenantID,
			AppID:               artifact.AppID,
			Domain:              artifact.Domain,
			UseCase:             artifact.UseCase,
			SourceID:            artifact.SourceID,
			SourceAdapter:       artifact.SourceAdapter,
			Dataset:             artifact.Dataset,
			ArtifactID:          artifact.ArtifactID,
			SignalID:            artifact.SignalID,
			SignalType:          artifact.SignalType,
			DetectorID:          artifact.DetectorID,
			Severity:            artifact.Severity,
			Confidence:          artifact.Confidence,
			ProposalSource:      storage.MarketOpsGraphProposalSourceDSMSignal,
			SourceRecordType:    storage.MarketOpsGraphProposalSourceDSMSignal,
			SourceRecordID:      artifact.SignalID,
			SourceRecordVersion: artifact.DetectorID,
			EventIDs:            append([]string(nil), artifact.EventIDs...),
			SubjectSymbol:       artifact.SubjectSymbol,
			CandidateType:       candidateType,
			Labels:              stringSliceOrEmpty(candidate["labels"]),
			Status:              storage.MarketOpsDSMGraphProposalStatusProposed,
		}
		switch candidateType {
		case "node_candidate":
			record.NodeID = firstMapString(candidate, "node_id")
			if record.NodeID == "" {
				continue
			}
			record.ProposalID = stableMarketOpsDSMGraphProposalID(artifact.ArtifactID, artifact.SignalID, candidateType, record.NodeID)
		case "relationship_candidate":
			record.FromNode = firstMapString(candidate, "from")
			record.Relationship = firstMapString(candidate, "relationship")
			record.ToNode = firstMapString(candidate, "to")
			if record.FromNode == "" || record.Relationship == "" || record.ToNode == "" {
				continue
			}
			record.ProposalID = stableMarketOpsDSMGraphProposalID(artifact.ArtifactID, artifact.SignalID, candidateType, record.FromNode, record.Relationship, record.ToNode)
		default:
			continue
		}
		properties, ok := candidate["properties"].(map[string]any)
		if !ok {
			properties = map[string]any{}
		}
		propertiesJSON, err := json.Marshal(properties)
		if err != nil {
			return nil, fmt.Errorf("marshal marketops dsm graph proposal properties: %w", err)
		}
		rawCandidate, err := json.Marshal(candidate)
		if err != nil {
			return nil, fmt.Errorf("marshal marketops dsm graph proposal candidate: %w", err)
		}
		record.PropertiesJSON = propertiesJSON
		record.RawCandidate = rawCandidate
		record.SourceRefsJSON, _ = json.Marshal(map[string]any{"artifact_id": artifact.ArtifactID, "signal_id": artifact.SignalID})
		record.LineageRefsJSON, _ = json.Marshal(map[string]any{"event_ids": artifact.EventIDs})
		records = append(records, record)
	}
	return records, nil
}

func stableMarketOpsDSMGraphProposalID(parts ...string) string {
	h := sha256.New()
	for _, part := range append([]string{"marketops.dsm.graph_proposal_v1"}, parts...) {
		h.Write([]byte(strings.TrimSpace(part)))
		h.Write([]byte{0})
	}
	return "graphprop_marketops_dsm_v1_" + hex.EncodeToString(h.Sum(nil))[:24]
}

func (r *Repository) UpsertMarketOpsDSMGraphProposal(ctx context.Context, record storage.MarketOpsDSMGraphProposalRecord) error {
	return upsertMarketOpsDSMGraphProposal(ctx, r.db, record)
}

func upsertMarketOpsDSMGraphProposal(ctx context.Context, executor statementExecutor, record storage.MarketOpsDSMGraphProposalRecord) error {
	if err := validateMarketOpsDSMGraphProposal(record); err != nil {
		return err
	}
	proposalSource := graphProposalSourceOrDefault(record.ProposalSource)
	artifactID, signalID, signalType := sql.NullString{}, sql.NullString{}, sql.NullString{}
	detectorID, severity := sql.NullString{}, sql.NullString{}
	var confidence any
	if proposalSource == storage.MarketOpsGraphProposalSourceDSMSignal {
		artifactID, signalID, signalType = nullString(record.ArtifactID), nullString(record.SignalID), nullString(record.SignalType)
		detectorID, severity, confidence = nullString(record.DetectorID), nullString(record.Severity), record.Confidence
	}
	eventIDs := record.EventIDs
	if eventIDs == nil {
		eventIDs = []string{}
	}
	labels := record.Labels
	if labels == nil {
		labels = []string{}
	}
	_, err := executor.ExecContext(ctx, `
INSERT INTO marketops_dsm_graph_proposals (
 proposal_id, tenant_id, app_id, domain, use_case, source_id, source_adapter, dataset, artifact_id,
 signal_id, signal_type, detector_id, severity, confidence, proposal_source, source_record_type, source_record_id,
 source_record_version, source_refs, lineage_refs, event_ids, subject_symbol, candidate_type, node_id, from_node,
 relationship, to_node, labels, properties, raw_candidate, status
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31)
ON CONFLICT (proposal_id) DO UPDATE SET
 tenant_id=EXCLUDED.tenant_id, app_id=EXCLUDED.app_id, domain=EXCLUDED.domain, use_case=EXCLUDED.use_case,
 source_id=EXCLUDED.source_id, source_adapter=EXCLUDED.source_adapter, dataset=EXCLUDED.dataset,
 artifact_id=EXCLUDED.artifact_id, signal_id=EXCLUDED.signal_id, signal_type=EXCLUDED.signal_type,
 detector_id=EXCLUDED.detector_id, severity=EXCLUDED.severity, confidence=EXCLUDED.confidence,
 proposal_source=EXCLUDED.proposal_source, source_record_type=EXCLUDED.source_record_type,
 source_record_id=EXCLUDED.source_record_id, source_record_version=EXCLUDED.source_record_version,
 source_refs=EXCLUDED.source_refs, lineage_refs=EXCLUDED.lineage_refs,
 event_ids=EXCLUDED.event_ids, subject_symbol=EXCLUDED.subject_symbol, candidate_type=EXCLUDED.candidate_type,
 node_id=EXCLUDED.node_id, from_node=EXCLUDED.from_node, relationship=EXCLUDED.relationship, to_node=EXCLUDED.to_node,
 labels=EXCLUDED.labels, properties=EXCLUDED.properties, raw_candidate=EXCLUDED.raw_candidate, updated_at=now()`,
		record.ProposalID, record.TenantID, recordAppID(record.AppID), recordDomain(record.Domain), recordUseCase(record.UseCase),
		record.SourceID, record.SourceAdapter, record.Dataset, artifactID, signalID, signalType, detectorID, severity, confidence,
		proposalSource, strings.TrimSpace(record.SourceRecordType), strings.TrimSpace(record.SourceRecordID),
		strings.TrimSpace(record.SourceRecordVersion), jsonOrEmpty(record.SourceRefsJSON), jsonOrEmpty(record.LineageRefsJSON),
		eventIDs, record.SubjectSymbol, record.CandidateType, record.NodeID, record.FromNode, record.Relationship,
		record.ToNode, labels, jsonOrEmpty(record.PropertiesJSON), jsonOrEmpty(record.RawCandidate),
		graphProposalStatusOrDefault(record.Status))
	if err != nil {
		return fmt.Errorf("upsert marketops dsm graph proposal: %w", err)
	}
	return nil
}

func (r *Repository) ListMarketOpsDSMGraphProposals(ctx context.Context, filter storage.MarketOpsDSMGraphProposalFilter) ([]storage.MarketOpsDSMGraphProposalRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsDSMGraphProposalSelect+`
WHERE ($1 = '' OR tenant_id = $1) AND ($2 = '' OR app_id = $2) AND ($3 = '' OR domain = $3) AND ($4 = '' OR use_case = $4)
 AND ($5 = '' OR artifact_id = $5) AND ($6 = '' OR signal_id = $6) AND ($7 = '' OR signal_type = $7)
 AND ($8 = '' OR subject_symbol = $8) AND ($9 = '' OR candidate_type = $9) AND ($10 = '' OR status = $10)
 AND ($11 = '' OR proposal_source = $11) AND ($12 = '' OR source_record_type = $12) AND ($13 = '' OR source_record_id = $13)
ORDER BY updated_at DESC LIMIT $14`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AppID), strings.TrimSpace(filter.Domain), strings.TrimSpace(filter.UseCase),
		strings.TrimSpace(filter.ArtifactID), strings.TrimSpace(filter.SignalID), strings.TrimSpace(filter.SignalType), strings.TrimSpace(filter.SubjectSymbol),
		strings.TrimSpace(filter.CandidateType), strings.TrimSpace(filter.Status), strings.TrimSpace(filter.ProposalSource),
		strings.TrimSpace(filter.SourceRecordType), strings.TrimSpace(filter.SourceRecordID), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops dsm graph proposals: %w", err)
	}
	defer rows.Close()
	records := []storage.MarketOpsDSMGraphProposalRecord{}
	for rows.Next() {
		record, err := scanMarketOpsDSMGraphProposal(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list marketops dsm graph proposals rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetMarketOpsDSMGraphProposal(ctx context.Context, proposalID string) (storage.MarketOpsDSMGraphProposalRecord, error) {
	record, err := scanMarketOpsDSMGraphProposal(r.db.QueryRowContext(ctx, marketOpsDSMGraphProposalSelect+` WHERE proposal_id = $1`, strings.TrimSpace(proposalID)))
	if err != nil {
		return storage.MarketOpsDSMGraphProposalRecord{}, err
	}
	return record, nil
}

func (r *Repository) MutateMarketOpsDSMGraphProposal(ctx context.Context, mutation storage.MarketOpsDSMGraphProposalMutation) (storage.MarketOpsDSMGraphProposalRecord, error) {
	if strings.TrimSpace(mutation.ProposalID) == "" {
		return storage.MarketOpsDSMGraphProposalRecord{}, fmt.Errorf("marketops dsm graph proposal id is required")
	}
	if !validMarketOpsDSMGraphProposalStatus(mutation.Status) {
		return storage.MarketOpsDSMGraphProposalRecord{}, fmt.Errorf("marketops dsm graph proposal status is invalid")
	}
	decidedAt := mutation.DecidedAt.UTC()
	if decidedAt.IsZero() {
		decidedAt = time.Now().UTC()
	}
	record, err := scanMarketOpsDSMGraphProposal(r.db.QueryRowContext(ctx, `
UPDATE marketops_dsm_graph_proposals SET status=$2, reviewed_by=$3, decision_note=$4, decided_at=$5, updated_at=now()
WHERE proposal_id=$1 `+marketOpsDSMGraphProposalReturning, strings.TrimSpace(mutation.ProposalID), strings.TrimSpace(mutation.Status), strings.TrimSpace(mutation.ReviewedBy), strings.TrimSpace(mutation.DecisionNote), decidedAt))
	if err != nil {
		return storage.MarketOpsDSMGraphProposalRecord{}, err
	}
	return record, nil
}

const marketOpsDSMGraphProposalSelect = `
SELECT proposal_id, tenant_id, app_id, domain, use_case, source_id, source_adapter, dataset, artifact_id,
 signal_id, signal_type, detector_id, severity, confidence, proposal_source, source_record_type, source_record_id,
 source_record_version, source_refs, lineage_refs, COALESCE(array_to_json(event_ids), '[]'::json)::text,
 subject_symbol, candidate_type, node_id, from_node, relationship, to_node,
 COALESCE(array_to_json(labels), '[]'::json)::text, properties, raw_candidate, status, reviewed_by, decision_note,
 decided_at, created_at, updated_at FROM marketops_dsm_graph_proposals`

const marketOpsDSMGraphProposalReturning = `RETURNING proposal_id, tenant_id, app_id, domain, use_case, source_id, source_adapter, dataset, artifact_id,
 signal_id, signal_type, detector_id, severity, confidence, proposal_source, source_record_type, source_record_id,
 source_record_version, source_refs, lineage_refs, COALESCE(array_to_json(event_ids), '[]'::json)::text,
 subject_symbol, candidate_type, node_id, from_node, relationship, to_node,
 COALESCE(array_to_json(labels), '[]'::json)::text, properties, raw_candidate, status, reviewed_by, decision_note,
 decided_at, created_at, updated_at`

type marketOpsDSMGraphProposalScanner interface{ Scan(dest ...any) error }

func scanMarketOpsDSMGraphProposal(scanner marketOpsDSMGraphProposalScanner) (storage.MarketOpsDSMGraphProposalRecord, error) {
	var record storage.MarketOpsDSMGraphProposalRecord
	var eventIDsJSON, labelsJSON string
	var artifactID, signalID, signalType, detectorID, severity sql.NullString
	var confidence sql.NullFloat64
	if err := scanner.Scan(&record.ProposalID, &record.TenantID, &record.AppID, &record.Domain, &record.UseCase,
		&record.SourceID, &record.SourceAdapter, &record.Dataset, &artifactID, &signalID, &signalType,
		&detectorID, &severity, &confidence, &record.ProposalSource, &record.SourceRecordType, &record.SourceRecordID,
		&record.SourceRecordVersion, &record.SourceRefsJSON, &record.LineageRefsJSON, &eventIDsJSON, &record.SubjectSymbol, &record.CandidateType,
		&record.NodeID, &record.FromNode, &record.Relationship, &record.ToNode, &labelsJSON, &record.PropertiesJSON,
		&record.RawCandidate, &record.Status, &record.ReviewedBy, &record.DecisionNote, &record.DecidedAt, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return storage.MarketOpsDSMGraphProposalRecord{}, mapScanError("scan marketops dsm graph proposal", err)
	}
	record.ArtifactID = artifactID.String
	record.SignalID = signalID.String
	record.SignalType = signalType.String
	record.DetectorID = detectorID.String
	record.Severity = severity.String
	record.Confidence = confidence.Float64
	if err := json.Unmarshal([]byte(eventIDsJSON), &record.EventIDs); err != nil {
		return storage.MarketOpsDSMGraphProposalRecord{}, fmt.Errorf("scan marketops dsm graph proposal event ids: %w", err)
	}
	if err := json.Unmarshal([]byte(labelsJSON), &record.Labels); err != nil {
		return storage.MarketOpsDSMGraphProposalRecord{}, fmt.Errorf("scan marketops dsm graph proposal labels: %w", err)
	}
	return record, nil
}

func validateMarketOpsDSMGraphProposal(record storage.MarketOpsDSMGraphProposalRecord) error {
	proposalSource := graphProposalSourceOrDefault(record.ProposalSource)
	if !validMarketOpsGraphProposalSource(proposalSource) {
		return fmt.Errorf("marketops graph proposal source is invalid")
	}
	required := map[string]string{
		"proposal_id":    record.ProposalID,
		"tenant_id":      record.TenantID,
		"source_id":      record.SourceID,
		"source_adapter": record.SourceAdapter,
		"dataset":        record.Dataset,
		"candidate_type": record.CandidateType,
	}
	if proposalSource == storage.MarketOpsGraphProposalSourceDSMSignal {
		required["artifact_id"] = record.ArtifactID
		required["signal_id"] = record.SignalID
		required["signal_type"] = record.SignalType
		required["detector_id"] = record.DetectorID
		required["severity"] = record.Severity
	} else {
		required["source_record_type"] = record.SourceRecordType
		required["source_record_id"] = record.SourceRecordID
	}
	for name, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("marketops dsm graph proposal %s is required", name)
		}
	}
	if proposalSource == storage.MarketOpsGraphProposalSourceDSMSignal && (record.Confidence < 0 || record.Confidence > 1) {
		return fmt.Errorf("marketops dsm graph proposal confidence must be between 0 and 1")
	}
	switch record.CandidateType {
	case "node_candidate":
		if strings.TrimSpace(record.NodeID) == "" {
			return fmt.Errorf("marketops dsm graph proposal node_id is required")
		}
	case "relationship_candidate":
		if strings.TrimSpace(record.FromNode) == "" || strings.TrimSpace(record.Relationship) == "" || strings.TrimSpace(record.ToNode) == "" {
			return fmt.Errorf("marketops dsm graph proposal relationship identity is required")
		}
	default:
		return fmt.Errorf("marketops dsm graph proposal candidate_type is invalid")
	}
	if !validMarketOpsDSMGraphProposalStatus(graphProposalStatusOrDefault(record.Status)) {
		return fmt.Errorf("marketops dsm graph proposal status is invalid")
	}
	if err := validateJSONObject("marketops dsm graph proposal properties", jsonOrEmpty(record.PropertiesJSON)); err != nil {
		return err
	}
	if err := validateJSONObject("marketops graph proposal source refs", jsonOrEmpty(record.SourceRefsJSON)); err != nil {
		return err
	}
	if err := validateJSONObject("marketops graph proposal lineage refs", jsonOrEmpty(record.LineageRefsJSON)); err != nil {
		return err
	}
	return validateJSONObject("marketops dsm graph proposal candidate", jsonOrEmpty(record.RawCandidate))
}

func stringSliceOrEmpty(value any) []string {
	items := stringSlice(value)
	if items == nil {
		return []string{}
	}
	return items
}

func graphProposalSourceOrDefault(source string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return storage.MarketOpsGraphProposalSourceDSMSignal
	}
	return source
}

func validMarketOpsGraphProposalSource(source string) bool {
	switch strings.TrimSpace(source) {
	case storage.MarketOpsGraphProposalSourceDSMSignal,
		storage.MarketOpsGraphProposalSourceMarketState,
		storage.MarketOpsGraphProposalSourceStateTransition,
		storage.MarketOpsGraphProposalSourceHypothesisDefinition,
		storage.MarketOpsGraphProposalSourceHypothesisEvaluation,
		storage.MarketOpsGraphProposalSourceOpportunity,
		storage.MarketOpsGraphProposalSourceOutcome:
		return true
	default:
		return false
	}
}

func graphProposalStatusOrDefault(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return storage.MarketOpsDSMGraphProposalStatusProposed
	}
	return status
}

func validMarketOpsDSMGraphProposalStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case storage.MarketOpsDSMGraphProposalStatusProposed, storage.MarketOpsDSMGraphProposalStatusAccepted, storage.MarketOpsDSMGraphProposalStatusRejected, storage.MarketOpsDSMGraphProposalStatusSuperseded:
		return true
	default:
		return false
	}
}

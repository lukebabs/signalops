package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func extractMarketOpsDSMArtifacts(signal storage.SignalLedgerRecord) ([]storage.MarketOpsDSMArtifactRecord, error) {
	if signal.AppID != "marketops" || signal.Domain != "market_data" || signal.DetectorID == "" {
		return nil, nil
	}
	var semanticItems []map[string]any
	if len(signal.SemanticEvidenceJSON) == 0 {
		return nil, nil
	}
	if err := json.Unmarshal(signal.SemanticEvidenceJSON, &semanticItems); err != nil {
		return nil, fmt.Errorf("extract marketops dsm semantic evidence: %w", err)
	}
	records := []storage.MarketOpsDSMArtifactRecord{}
	for _, item := range semanticItems {
		itemType, _ := item["type"].(string)
		if itemType != "dsm_artifact_proposal" {
			continue
		}
		artifactMap, ok := item["artifact"].(map[string]any)
		if !ok {
			continue
		}
		artifactID := firstMapString(item, "artifact_id")
		if artifactID == "" {
			artifactID = firstMapString(artifactMap, "artifact_id")
		}
		artifactType := firstMapString(artifactMap, "artifact_type")
		if artifactID == "" || artifactType == "" {
			continue
		}
		subjectSymbol := ""
		if subject, ok := artifactMap["subject"].(map[string]any); ok {
			subjectSymbol = firstMapString(subject, "symbol")
		}
		artifactJSON, err := json.Marshal(artifactMap)
		if err != nil {
			return nil, fmt.Errorf("marshal marketops dsm artifact: %w", err)
		}
		semanticJSON, err := json.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("marshal marketops dsm semantic evidence: %w", err)
		}
		records = append(records, storage.MarketOpsDSMArtifactRecord{
			ArtifactID:           artifactID,
			TenantID:             signal.TenantID,
			AppID:                signal.AppID,
			Domain:               signal.Domain,
			UseCase:              signal.UseCase,
			SourceID:             signal.SourceID,
			SourceAdapter:        signal.SourceAdapter,
			Dataset:              signal.Dataset,
			SignalID:             signal.SignalID,
			SignalType:           signal.SignalType,
			DetectorID:           signal.DetectorID,
			Severity:             signal.Severity,
			Confidence:           signal.Confidence,
			EventIDs:             append([]string(nil), signal.EventIDs...),
			SubjectSymbol:        subjectSymbol,
			ArtifactType:         artifactType,
			ArtifactJSON:         artifactJSON,
			SemanticEvidenceJSON: semanticJSON,
			GraphTargetsJSON:     append([]byte(nil), signal.GraphTargetsJSON...),
			SupportingMetrics:    append([]byte(nil), signal.SupportingMetrics...),
			QualityIssues:        stringSlice(artifactMap["quality_issues"]),
		})
	}
	return records, nil
}

func upsertMarketOpsDSMArtifact(ctx context.Context, executor statementExecutor, record storage.MarketOpsDSMArtifactRecord) error {
	if err := validateMarketOpsDSMArtifact(record); err != nil {
		return err
	}
	_, err := executor.ExecContext(ctx, `
INSERT INTO marketops_dsm_artifacts (
 artifact_id, tenant_id, app_id, domain, use_case, source_id, source_adapter, dataset, signal_id,
 signal_type, detector_id, severity, confidence, event_ids, subject_symbol, artifact_type,
 artifact, semantic_evidence, graph_targets, supporting_metrics, quality_issues
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)
ON CONFLICT (artifact_id) DO UPDATE SET
 tenant_id=EXCLUDED.tenant_id, app_id=EXCLUDED.app_id, domain=EXCLUDED.domain, use_case=EXCLUDED.use_case,
 source_id=EXCLUDED.source_id, source_adapter=EXCLUDED.source_adapter, dataset=EXCLUDED.dataset,
 signal_id=EXCLUDED.signal_id, signal_type=EXCLUDED.signal_type, detector_id=EXCLUDED.detector_id,
 severity=EXCLUDED.severity, confidence=EXCLUDED.confidence, event_ids=EXCLUDED.event_ids,
 subject_symbol=EXCLUDED.subject_symbol, artifact_type=EXCLUDED.artifact_type, artifact=EXCLUDED.artifact,
 semantic_evidence=EXCLUDED.semantic_evidence, graph_targets=EXCLUDED.graph_targets,
 supporting_metrics=EXCLUDED.supporting_metrics, quality_issues=EXCLUDED.quality_issues, updated_at=now()`,
		record.ArtifactID, record.TenantID, recordAppID(record.AppID), recordDomain(record.Domain), recordUseCase(record.UseCase),
		record.SourceID, record.SourceAdapter, record.Dataset, record.SignalID, record.SignalType,
		record.DetectorID, record.Severity, record.Confidence, record.EventIDs, record.SubjectSymbol,
		record.ArtifactType, jsonOrEmpty(record.ArtifactJSON), jsonOrEmpty(record.SemanticEvidenceJSON),
		jsonArrayOrEmpty(record.GraphTargetsJSON), jsonOrEmpty(record.SupportingMetrics), record.QualityIssues)
	if err != nil {
		return fmt.Errorf("upsert marketops dsm artifact: %w", err)
	}
	return nil
}

func (r *Repository) ListMarketOpsDSMArtifacts(ctx context.Context, filter storage.MarketOpsDSMArtifactFilter) ([]storage.MarketOpsDSMArtifactRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsDSMArtifactSelect+`
WHERE ($1 = '' OR tenant_id = $1) AND ($2 = '' OR app_id = $2) AND ($3 = '' OR domain = $3) AND ($4 = '' OR use_case = $4)
 AND ($5 = '' OR signal_type = $5) AND ($6 = '' OR severity = $6) AND ($7 = '' OR subject_symbol = $7)
ORDER BY updated_at DESC LIMIT $8`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AppID), strings.TrimSpace(filter.Domain), strings.TrimSpace(filter.UseCase),
		strings.TrimSpace(filter.SignalType), strings.TrimSpace(filter.Severity), strings.TrimSpace(filter.SubjectSymbol), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops dsm artifacts: %w", err)
	}
	defer rows.Close()
	records := []storage.MarketOpsDSMArtifactRecord{}
	for rows.Next() {
		record, err := scanMarketOpsDSMArtifact(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list marketops dsm artifacts rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetMarketOpsDSMArtifact(ctx context.Context, artifactID string) (storage.MarketOpsDSMArtifactRecord, error) {
	record, err := scanMarketOpsDSMArtifact(r.db.QueryRowContext(ctx, marketOpsDSMArtifactSelect+` WHERE artifact_id = $1`, strings.TrimSpace(artifactID)))
	if err != nil {
		return storage.MarketOpsDSMArtifactRecord{}, err
	}
	return record, nil
}

const marketOpsDSMArtifactSelect = `
SELECT artifact_id, tenant_id, app_id, domain, use_case, source_id, source_adapter, dataset, signal_id,
 signal_type, detector_id, severity, confidence, COALESCE(array_to_json(event_ids), '[]'::json)::text,
 subject_symbol, artifact_type, artifact, semantic_evidence, graph_targets, supporting_metrics,
 COALESCE(array_to_json(quality_issues), '[]'::json)::text, created_at, updated_at FROM marketops_dsm_artifacts`

type marketOpsDSMArtifactScanner interface{ Scan(dest ...any) error }

func scanMarketOpsDSMArtifact(scanner marketOpsDSMArtifactScanner) (storage.MarketOpsDSMArtifactRecord, error) {
	var record storage.MarketOpsDSMArtifactRecord
	var eventIDsJSON, qualityIssuesJSON string
	if err := scanner.Scan(&record.ArtifactID, &record.TenantID, &record.AppID, &record.Domain, &record.UseCase,
		&record.SourceID, &record.SourceAdapter, &record.Dataset, &record.SignalID, &record.SignalType,
		&record.DetectorID, &record.Severity, &record.Confidence, &eventIDsJSON, &record.SubjectSymbol,
		&record.ArtifactType, &record.ArtifactJSON, &record.SemanticEvidenceJSON, &record.GraphTargetsJSON,
		&record.SupportingMetrics, &qualityIssuesJSON, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return storage.MarketOpsDSMArtifactRecord{}, mapScanError("scan marketops dsm artifact", err)
	}
	if err := json.Unmarshal([]byte(eventIDsJSON), &record.EventIDs); err != nil {
		return storage.MarketOpsDSMArtifactRecord{}, fmt.Errorf("scan marketops dsm artifact event ids: %w", err)
	}
	if err := json.Unmarshal([]byte(qualityIssuesJSON), &record.QualityIssues); err != nil {
		return storage.MarketOpsDSMArtifactRecord{}, fmt.Errorf("scan marketops dsm artifact quality issues: %w", err)
	}
	return record, nil
}

func validateMarketOpsDSMArtifact(record storage.MarketOpsDSMArtifactRecord) error {
	for name, value := range map[string]string{
		"artifact_id":    record.ArtifactID,
		"tenant_id":      record.TenantID,
		"source_id":      record.SourceID,
		"source_adapter": record.SourceAdapter,
		"dataset":        record.Dataset,
		"signal_id":      record.SignalID,
		"signal_type":    record.SignalType,
		"detector_id":    record.DetectorID,
		"severity":       record.Severity,
		"artifact_type":  record.ArtifactType,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("marketops dsm artifact %s is required", name)
		}
	}
	if record.Confidence < 0 || record.Confidence > 1 {
		return fmt.Errorf("marketops dsm artifact confidence must be between 0 and 1")
	}
	if err := validateJSONObject("marketops dsm artifact", jsonOrEmpty(record.ArtifactJSON)); err != nil {
		return err
	}
	if err := validateJSONObject("marketops dsm semantic evidence", jsonOrEmpty(record.SemanticEvidenceJSON)); err != nil {
		return err
	}
	if err := validateJSONArray("marketops dsm graph targets", jsonArrayOrEmpty(record.GraphTargetsJSON)); err != nil {
		return err
	}
	return validateJSONObject("marketops dsm supporting metrics", jsonOrEmpty(record.SupportingMetrics))
}

func firstMapString(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}

func stringSlice(value any) []string {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := []string{}
	for _, item := range raw {
		if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
			out = append(out, strings.TrimSpace(text))
		}
	}
	return out
}

func validateJSONArray(name string, value []byte) error {
	var decoded []any
	if err := json.Unmarshal(value, &decoded); err != nil {
		return fmt.Errorf("%s must be a JSON array: %w", name, err)
	}
	if decoded == nil {
		return fmt.Errorf("%s must be a JSON array", name)
	}
	return nil
}

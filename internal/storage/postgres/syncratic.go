package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) UpsertSyncraticContextWindow(ctx context.Context, record storage.SyncraticContextWindowRecord) error {
	if strings.TrimSpace(record.ContextWindowID) == "" || strings.TrimSpace(record.TenantID) == "" || strings.TrimSpace(record.SubjectSymbol) == "" || strings.TrimSpace(record.ContextStrategy) == "" || record.WindowStart.IsZero() || record.WindowEnd.IsZero() || strings.TrimSpace(record.ContextBuilderVersion) == "" || strings.TrimSpace(record.EvidenceDigest) == "" {
		return fmt.Errorf("syncratic context_window_id, tenant_id, subject_symbol, context_strategy, window, builder_version, and evidence_digest are required")
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO syncratic_context_windows (
 context_window_id, tenant_id, app_id, domain, use_case, subject_type, subject_id, subject_symbol,
 window_start, window_end, context_strategy, context_builder_version, context_payload_version, signal_types, detector_ids,
 event_ids, signal_ids, alert_ids, artifact_ids, graph_proposal_ids, label_ids, market_state_ids, state_transition_ids, marketops_evidence_ids, hypothesis_evaluation_ids, opportunity_ids, outcome_ids, calibration_summary_ids, baseline_refs,
 evaluation_refs, promotion_candidate_refs, summary_metrics, quality_warnings, lineage_refs, evidence_digest, idempotency_key, status
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37)
ON CONFLICT (tenant_id, use_case, context_strategy, subject_symbol, window_start, window_end, context_builder_version)
DO UPDATE SET
 context_window_id=EXCLUDED.context_window_id, app_id=EXCLUDED.app_id, domain=EXCLUDED.domain,
 subject_type=EXCLUDED.subject_type, subject_id=EXCLUDED.subject_id, context_payload_version=EXCLUDED.context_payload_version, signal_types=EXCLUDED.signal_types,
 detector_ids=EXCLUDED.detector_ids, event_ids=EXCLUDED.event_ids, signal_ids=EXCLUDED.signal_ids,
 alert_ids=EXCLUDED.alert_ids, artifact_ids=EXCLUDED.artifact_ids, graph_proposal_ids=EXCLUDED.graph_proposal_ids,
 label_ids=EXCLUDED.label_ids, market_state_ids=EXCLUDED.market_state_ids, state_transition_ids=EXCLUDED.state_transition_ids,
 marketops_evidence_ids=EXCLUDED.marketops_evidence_ids, hypothesis_evaluation_ids=EXCLUDED.hypothesis_evaluation_ids,
 opportunity_ids=EXCLUDED.opportunity_ids, outcome_ids=EXCLUDED.outcome_ids, calibration_summary_ids=EXCLUDED.calibration_summary_ids,
 baseline_refs=EXCLUDED.baseline_refs, evaluation_refs=EXCLUDED.evaluation_refs,
 promotion_candidate_refs=EXCLUDED.promotion_candidate_refs, summary_metrics=EXCLUDED.summary_metrics,
 quality_warnings=EXCLUDED.quality_warnings, lineage_refs=EXCLUDED.lineage_refs,
 evidence_digest=EXCLUDED.evidence_digest, idempotency_key=EXCLUDED.idempotency_key, status=EXCLUDED.status,
 updated_at=now()`, record.ContextWindowID, strings.TrimSpace(record.TenantID), recordAppID(record.AppID), recordDomain(record.Domain), recordUseCase(record.UseCase), firstNonEmptyString(record.SubjectType, "ticker"), strings.TrimSpace(record.SubjectID), strings.TrimSpace(record.SubjectSymbol), record.WindowStart.UTC(), record.WindowEnd.UTC(), strings.TrimSpace(record.ContextStrategy), strings.TrimSpace(record.ContextBuilderVersion), firstNonEmptyString(record.ContextPayloadVersion, "signalops.syncratic.context_payload.v1"), pqArray(record.SignalTypes), pqArray(record.DetectorIDs), pqArray(record.EventIDs), pqArray(record.SignalIDs), pqArray(record.AlertIDs), pqArray(record.ArtifactIDs), pqArray(record.GraphProposalIDs), pqArray(record.LabelIDs), pqArray(record.MarketStateIDs), pqArray(record.StateTransitionIDs), pqArray(record.MarketOpsEvidenceIDs), pqArray(record.HypothesisEvaluationIDs), pqArray(record.OpportunityIDs), pqArray(record.OutcomeIDs), pqArray(record.CalibrationSummaryIDs), jsonArrayOrEmpty(record.BaselineRefsJSON), jsonArrayOrEmpty(record.EvaluationRefsJSON), jsonArrayOrEmpty(record.PromotionCandidateRefsJSON), jsonOrEmpty(record.SummaryMetricsJSON), jsonArrayOrEmpty(record.QualityWarningsJSON), jsonOrEmpty(record.LineageRefsJSON), strings.TrimSpace(record.EvidenceDigest), strings.TrimSpace(record.IdempotencyKey), firstNonEmptyString(record.Status, "active"))
	if err != nil {
		return fmt.Errorf("upsert syncratic context window: %w", err)
	}
	return nil
}

func (r *Repository) ListSyncraticContextWindows(ctx context.Context, filter storage.SyncraticContextWindowFilter) ([]storage.SyncraticContextWindowRecord, error) {
	rows, err := r.db.QueryContext(ctx, syncraticContextWindowSelect+`
WHERE ($1='' OR tenant_id=$1) AND ($2='' OR app_id=$2) AND ($3='' OR domain=$3) AND ($4='' OR use_case=$4)
 AND ($5='' OR subject_symbol=$5) AND ($6='' OR context_strategy=$6) AND ($7='' OR status=$7)
ORDER BY updated_at DESC LIMIT $8`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AppID), strings.TrimSpace(filter.Domain), strings.TrimSpace(filter.UseCase), strings.TrimSpace(filter.SubjectSymbol), strings.TrimSpace(filter.ContextStrategy), strings.TrimSpace(filter.Status), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list syncratic context windows: %w", err)
	}
	defer rows.Close()
	records := []storage.SyncraticContextWindowRecord{}
	for rows.Next() {
		record, err := scanSyncraticContextWindow(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list syncratic context windows rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetSyncraticContextWindow(ctx context.Context, contextWindowID string) (storage.SyncraticContextWindowRecord, error) {
	return scanSyncraticContextWindow(r.db.QueryRowContext(ctx, syncraticContextWindowSelect+` WHERE context_window_id=$1`, strings.TrimSpace(contextWindowID)))
}

func (r *Repository) UpsertSyncraticInsight(ctx context.Context, record storage.SyncraticInsightRecord) error {
	if strings.TrimSpace(record.SyncraticInsightID) == "" || strings.TrimSpace(record.TenantID) == "" || strings.TrimSpace(record.ContextWindowID) == "" || strings.TrimSpace(record.InsightType) == "" || strings.TrimSpace(record.SubjectSymbol) == "" || strings.TrimSpace(record.BuilderVersion) == "" {
		return fmt.Errorf("syncratic insight_id, tenant_id, context_window_id, insight_type, subject_symbol, and builder_version are required")
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO syncratic_insights (
 syncratic_insight_id, tenant_id, app_id, domain, use_case, context_window_id, insight_type,
 subject_type, subject_id, subject_symbol, status, severity, confidence, title, summary, explanation,
 supporting_alert_ids, supporting_signal_ids, supporting_event_ids, supporting_artifact_ids,
 related_graph_proposal_ids, related_label_ids, metrics, recommendation, builder_version
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)
ON CONFLICT (context_window_id, insight_type, builder_version)
DO UPDATE SET
 syncratic_insight_id=EXCLUDED.syncratic_insight_id, tenant_id=EXCLUDED.tenant_id, app_id=EXCLUDED.app_id,
 domain=EXCLUDED.domain, use_case=EXCLUDED.use_case, subject_type=EXCLUDED.subject_type,
 subject_id=EXCLUDED.subject_id, subject_symbol=EXCLUDED.subject_symbol, status=EXCLUDED.status,
 severity=EXCLUDED.severity, confidence=EXCLUDED.confidence, title=EXCLUDED.title, summary=EXCLUDED.summary,
 explanation=EXCLUDED.explanation, supporting_alert_ids=EXCLUDED.supporting_alert_ids,
 supporting_signal_ids=EXCLUDED.supporting_signal_ids, supporting_event_ids=EXCLUDED.supporting_event_ids,
 supporting_artifact_ids=EXCLUDED.supporting_artifact_ids, related_graph_proposal_ids=EXCLUDED.related_graph_proposal_ids,
 related_label_ids=EXCLUDED.related_label_ids, metrics=EXCLUDED.metrics, recommendation=EXCLUDED.recommendation,
 updated_at=now()`, record.SyncraticInsightID, strings.TrimSpace(record.TenantID), recordAppID(record.AppID), recordDomain(record.Domain), recordUseCase(record.UseCase), strings.TrimSpace(record.ContextWindowID), strings.TrimSpace(record.InsightType), firstNonEmptyString(record.SubjectType, "ticker"), strings.TrimSpace(record.SubjectID), strings.TrimSpace(record.SubjectSymbol), firstNonEmptyString(record.Status, storage.SyncraticInsightStatusActive), firstNonEmptyString(record.Severity, "medium"), record.Confidence, strings.TrimSpace(record.Title), strings.TrimSpace(record.Summary), strings.TrimSpace(record.Explanation), pqArray(record.SupportingAlertIDs), pqArray(record.SupportingSignalIDs), pqArray(record.SupportingEventIDs), pqArray(record.SupportingArtifactIDs), pqArray(record.RelatedGraphProposalIDs), pqArray(record.RelatedLabelIDs), jsonOrEmpty(record.MetricsJSON), jsonOrEmpty(record.RecommendationJSON), strings.TrimSpace(record.BuilderVersion))
	if err != nil {
		return fmt.Errorf("upsert syncratic insight: %w", err)
	}
	return nil
}

func (r *Repository) ListSyncraticInsights(ctx context.Context, filter storage.SyncraticInsightFilter) ([]storage.SyncraticInsightRecord, error) {
	rows, err := r.db.QueryContext(ctx, syncraticInsightSelect+`
WHERE ($1='' OR tenant_id=$1) AND ($2='' OR app_id=$2) AND ($3='' OR domain=$3) AND ($4='' OR use_case=$4)
 AND ($5='' OR context_window_id=$5) AND ($6='' OR insight_type=$6) AND ($7='' OR subject_symbol=$7) AND ($8='' OR status=$8)
ORDER BY updated_at DESC LIMIT $9`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AppID), strings.TrimSpace(filter.Domain), strings.TrimSpace(filter.UseCase), strings.TrimSpace(filter.ContextWindowID), strings.TrimSpace(filter.InsightType), strings.TrimSpace(filter.SubjectSymbol), strings.TrimSpace(filter.Status), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list syncratic insights: %w", err)
	}
	defer rows.Close()
	records := []storage.SyncraticInsightRecord{}
	for rows.Next() {
		record, err := scanSyncraticInsight(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list syncratic insights rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetSyncraticInsight(ctx context.Context, syncraticInsightID string) (storage.SyncraticInsightRecord, error) {
	return scanSyncraticInsight(r.db.QueryRowContext(ctx, syncraticInsightSelect+` WHERE syncratic_insight_id=$1`, strings.TrimSpace(syncraticInsightID)))
}

const syncraticContextWindowSelect = `SELECT context_window_id, tenant_id, app_id, domain, use_case, subject_type, subject_id, subject_symbol,
 window_start, window_end, context_strategy, context_builder_version, context_payload_version, COALESCE(array_to_json(signal_types), '[]'::json)::text,
 COALESCE(array_to_json(detector_ids), '[]'::json)::text, COALESCE(array_to_json(event_ids), '[]'::json)::text,
 COALESCE(array_to_json(signal_ids), '[]'::json)::text, COALESCE(array_to_json(alert_ids), '[]'::json)::text,
 COALESCE(array_to_json(artifact_ids), '[]'::json)::text, COALESCE(array_to_json(graph_proposal_ids), '[]'::json)::text,
 COALESCE(array_to_json(label_ids), '[]'::json)::text,
 COALESCE(array_to_json(market_state_ids), '[]'::json)::text, COALESCE(array_to_json(state_transition_ids), '[]'::json)::text,
 COALESCE(array_to_json(marketops_evidence_ids), '[]'::json)::text, COALESCE(array_to_json(hypothesis_evaluation_ids), '[]'::json)::text,
 COALESCE(array_to_json(opportunity_ids), '[]'::json)::text, COALESCE(array_to_json(outcome_ids), '[]'::json)::text,
 COALESCE(array_to_json(calibration_summary_ids), '[]'::json)::text, baseline_refs, evaluation_refs, promotion_candidate_refs,
 summary_metrics, quality_warnings, lineage_refs, evidence_digest, idempotency_key, status, created_at, updated_at FROM syncratic_context_windows`

type syncraticContextWindowScanner interface{ Scan(dest ...any) error }

func scanSyncraticContextWindow(scanner syncraticContextWindowScanner) (storage.SyncraticContextWindowRecord, error) {
	var record storage.SyncraticContextWindowRecord
	var signalTypesJSON, detectorIDsJSON, eventIDsJSON, signalIDsJSON, alertIDsJSON, artifactIDsJSON, graphProposalIDsJSON, labelIDsJSON, marketStateIDsJSON, transitionIDsJSON, marketOpsEvidenceIDsJSON, hypothesisEvaluationIDsJSON, opportunityIDsJSON, outcomeIDsJSON, calibrationSummaryIDsJSON string
	if err := scanner.Scan(&record.ContextWindowID, &record.TenantID, &record.AppID, &record.Domain, &record.UseCase, &record.SubjectType, &record.SubjectID, &record.SubjectSymbol, &record.WindowStart, &record.WindowEnd, &record.ContextStrategy, &record.ContextBuilderVersion, &record.ContextPayloadVersion, &signalTypesJSON, &detectorIDsJSON, &eventIDsJSON, &signalIDsJSON, &alertIDsJSON, &artifactIDsJSON, &graphProposalIDsJSON, &labelIDsJSON, &marketStateIDsJSON, &transitionIDsJSON, &marketOpsEvidenceIDsJSON, &hypothesisEvaluationIDsJSON, &opportunityIDsJSON, &outcomeIDsJSON, &calibrationSummaryIDsJSON, &record.BaselineRefsJSON, &record.EvaluationRefsJSON, &record.PromotionCandidateRefsJSON, &record.SummaryMetricsJSON, &record.QualityWarningsJSON, &record.LineageRefsJSON, &record.EvidenceDigest, &record.IdempotencyKey, &record.Status, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return storage.SyncraticContextWindowRecord{}, mapScanError("scan syncratic context window", err)
	}
	for _, item := range []struct {
		raw  string
		dest *[]string
	}{{signalTypesJSON, &record.SignalTypes}, {detectorIDsJSON, &record.DetectorIDs}, {eventIDsJSON, &record.EventIDs}, {signalIDsJSON, &record.SignalIDs}, {alertIDsJSON, &record.AlertIDs}, {artifactIDsJSON, &record.ArtifactIDs}, {graphProposalIDsJSON, &record.GraphProposalIDs}, {labelIDsJSON, &record.LabelIDs}, {marketStateIDsJSON, &record.MarketStateIDs}, {transitionIDsJSON, &record.StateTransitionIDs}, {marketOpsEvidenceIDsJSON, &record.MarketOpsEvidenceIDs}, {hypothesisEvaluationIDsJSON, &record.HypothesisEvaluationIDs}, {opportunityIDsJSON, &record.OpportunityIDs}, {outcomeIDsJSON, &record.OutcomeIDs}, {calibrationSummaryIDsJSON, &record.CalibrationSummaryIDs}} {
		if err := json.Unmarshal([]byte(item.raw), item.dest); err != nil {
			return storage.SyncraticContextWindowRecord{}, err
		}
	}
	return record, nil
}

const syncraticInsightSelect = `SELECT syncratic_insight_id, tenant_id, app_id, domain, use_case, context_window_id, insight_type,
 subject_type, subject_id, subject_symbol, status, severity, confidence, title, summary, explanation,
 COALESCE(array_to_json(supporting_alert_ids), '[]'::json)::text, COALESCE(array_to_json(supporting_signal_ids), '[]'::json)::text,
 COALESCE(array_to_json(supporting_event_ids), '[]'::json)::text, COALESCE(array_to_json(supporting_artifact_ids), '[]'::json)::text,
 COALESCE(array_to_json(related_graph_proposal_ids), '[]'::json)::text, COALESCE(array_to_json(related_label_ids), '[]'::json)::text,
 metrics, recommendation, builder_version, created_at, updated_at FROM syncratic_insights`

type syncraticInsightScanner interface{ Scan(dest ...any) error }

func scanSyncraticInsight(scanner syncraticInsightScanner) (storage.SyncraticInsightRecord, error) {
	var record storage.SyncraticInsightRecord
	var alertIDsJSON, signalIDsJSON, eventIDsJSON, artifactIDsJSON, graphProposalIDsJSON, labelIDsJSON string
	if err := scanner.Scan(&record.SyncraticInsightID, &record.TenantID, &record.AppID, &record.Domain, &record.UseCase, &record.ContextWindowID, &record.InsightType, &record.SubjectType, &record.SubjectID, &record.SubjectSymbol, &record.Status, &record.Severity, &record.Confidence, &record.Title, &record.Summary, &record.Explanation, &alertIDsJSON, &signalIDsJSON, &eventIDsJSON, &artifactIDsJSON, &graphProposalIDsJSON, &labelIDsJSON, &record.MetricsJSON, &record.RecommendationJSON, &record.BuilderVersion, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return storage.SyncraticInsightRecord{}, mapScanError("scan syncratic insight", err)
	}
	for _, item := range []struct {
		raw  string
		dest *[]string
	}{{alertIDsJSON, &record.SupportingAlertIDs}, {signalIDsJSON, &record.SupportingSignalIDs}, {eventIDsJSON, &record.SupportingEventIDs}, {artifactIDsJSON, &record.SupportingArtifactIDs}, {graphProposalIDsJSON, &record.RelatedGraphProposalIDs}, {labelIDsJSON, &record.RelatedLabelIDs}} {
		if err := json.Unmarshal([]byte(item.raw), item.dest); err != nil {
			return storage.SyncraticInsightRecord{}, err
		}
	}
	return record, nil
}

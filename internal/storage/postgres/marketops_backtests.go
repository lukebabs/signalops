package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) CreateMarketOpsBacktestRun(ctx context.Context, record storage.MarketOpsBacktestRunRecord) error {
	if strings.TrimSpace(record.RunID) == "" || strings.TrimSpace(record.TenantID) == "" || strings.TrimSpace(record.DetectorID) == "" {
		return fmt.Errorf("marketops backtest run_id, tenant_id, and detector_id are required")
	}
	startedAt := record.StartedAt.UTC()
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	status := strings.TrimSpace(record.Status)
	if status == "" {
		status = storage.RunStatusStarted
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO marketops_backtest_runs (
 run_id, tenant_id, app_id, domain, use_case, source_id, source_adapter, dataset, detector_id,
 detector_version, status, requested_by, window_start, window_end, started_at, completed_at,
 filters, parameters, metrics, error_message, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,now())
ON CONFLICT (run_id) DO UPDATE SET
 tenant_id=EXCLUDED.tenant_id, app_id=EXCLUDED.app_id, domain=EXCLUDED.domain, use_case=EXCLUDED.use_case,
 source_id=EXCLUDED.source_id, source_adapter=EXCLUDED.source_adapter, dataset=EXCLUDED.dataset,
 detector_id=EXCLUDED.detector_id, detector_version=EXCLUDED.detector_version, status=EXCLUDED.status,
 requested_by=EXCLUDED.requested_by, window_start=EXCLUDED.window_start, window_end=EXCLUDED.window_end,
 started_at=EXCLUDED.started_at, completed_at=EXCLUDED.completed_at, filters=EXCLUDED.filters,
 parameters=EXCLUDED.parameters, metrics=EXCLUDED.metrics, error_message=EXCLUDED.error_message, updated_at=now()`,
		record.RunID, record.TenantID, recordAppID(record.AppID), recordDomain(record.Domain), recordUseCase(record.UseCase),
		record.SourceID, record.SourceAdapter, record.Dataset, record.DetectorID, record.DetectorVersion, status,
		firstNonEmptyString(record.RequestedBy, "operator-local"), record.WindowStart, record.WindowEnd, startedAt, record.CompletedAt,
		jsonOrEmpty(record.FiltersJSON), jsonOrEmpty(record.ParametersJSON), jsonOrEmpty(record.MetricsJSON), strings.TrimSpace(record.ErrorMessage))
	if err != nil {
		return fmt.Errorf("create marketops backtest run: %w", err)
	}
	return nil
}

func (r *Repository) CompleteMarketOpsBacktestRun(ctx context.Context, runID string, completedAt time.Time, metricsJSON []byte) (storage.MarketOpsBacktestRunRecord, error) {
	return r.updateMarketOpsBacktestRunTerminal(ctx, runID, storage.RunStatusSucceeded, completedAt, "", metricsJSON)
}

func (r *Repository) FailMarketOpsBacktestRun(ctx context.Context, runID string, failedAt time.Time, errorMessage string, metricsJSON []byte) (storage.MarketOpsBacktestRunRecord, error) {
	return r.updateMarketOpsBacktestRunTerminal(ctx, runID, storage.RunStatusFailed, failedAt, errorMessage, metricsJSON)
}

func (r *Repository) updateMarketOpsBacktestRunTerminal(ctx context.Context, runID string, status string, completedAt time.Time, errorMessage string, metricsJSON []byte) (storage.MarketOpsBacktestRunRecord, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE marketops_backtest_runs SET status=$2, completed_at=$3, metrics=$4, error_message=$5, updated_at=now() WHERE run_id=$1`, strings.TrimSpace(runID), status, completedAt.UTC(), jsonOrEmpty(metricsJSON), strings.TrimSpace(errorMessage))
	if err != nil {
		return storage.MarketOpsBacktestRunRecord{}, fmt.Errorf("update marketops backtest run terminal: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return storage.MarketOpsBacktestRunRecord{}, err
	}
	if changed == 0 {
		return storage.MarketOpsBacktestRunRecord{}, storage.ErrNotFound
	}
	return r.GetMarketOpsBacktestRun(ctx, runID)
}

func (r *Repository) PersistMarketOpsBacktestBatch(ctx context.Context, run storage.MarketOpsBacktestRunRecord, signals []storage.MarketOpsBacktestSignalRecord, artifacts []storage.MarketOpsBacktestArtifactRecord, proposals []storage.MarketOpsBacktestGraphProposalRecord, policyResults []storage.MarketOpsBacktestPolicyResultRecord) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin marketops backtest batch: %w", err)
	}
	defer tx.Rollback()
	for _, signal := range signals {
		if err := upsertMarketOpsBacktestSignal(ctx, tx, signal); err != nil {
			return err
		}
	}
	for _, artifact := range artifacts {
		if err := upsertMarketOpsBacktestArtifact(ctx, tx, artifact); err != nil {
			return err
		}
	}
	for _, proposal := range proposals {
		if err := upsertMarketOpsBacktestGraphProposal(ctx, tx, proposal); err != nil {
			return err
		}
	}
	for _, result := range policyResults {
		if err := upsertMarketOpsBacktestPolicyResult(ctx, tx, result); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit marketops backtest batch: %w", err)
	}
	return nil
}

func upsertMarketOpsBacktestSignal(ctx context.Context, executor statementExecutor, record storage.MarketOpsBacktestSignalRecord) error {
	s := record.SignalLedgerRecord
	_, err := executor.ExecContext(ctx, `
INSERT INTO marketops_backtest_signals (
 run_id, signal_id, tenant_id, source_id, app_id, domain, use_case, source_domain, source_adapter, ingestion_mode, dataset,
 event_ids, artifact_ids, signal_type, detector_id, detector_version, model_version, signal_time,
 observation_time, effective_time, processing_time, window_start, window_end, confidence, severity,
 entities, supporting_metrics, graph_targets, semantic_evidence, evidence, recommendation,
 correlation_id, trace_id, causation_id, replay_job_id, event, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,now())
ON CONFLICT (run_id, signal_id) DO UPDATE SET
 tenant_id=EXCLUDED.tenant_id, source_id=EXCLUDED.source_id, app_id=EXCLUDED.app_id, domain=EXCLUDED.domain, use_case=EXCLUDED.use_case,
 source_domain=EXCLUDED.source_domain, source_adapter=EXCLUDED.source_adapter, ingestion_mode=EXCLUDED.ingestion_mode, dataset=EXCLUDED.dataset,
 event_ids=EXCLUDED.event_ids, artifact_ids=EXCLUDED.artifact_ids, signal_type=EXCLUDED.signal_type, detector_id=EXCLUDED.detector_id,
 detector_version=EXCLUDED.detector_version, model_version=EXCLUDED.model_version, signal_time=EXCLUDED.signal_time, observation_time=EXCLUDED.observation_time,
 effective_time=EXCLUDED.effective_time, processing_time=EXCLUDED.processing_time, window_start=EXCLUDED.window_start, window_end=EXCLUDED.window_end,
 confidence=EXCLUDED.confidence, severity=EXCLUDED.severity, entities=EXCLUDED.entities, supporting_metrics=EXCLUDED.supporting_metrics,
 graph_targets=EXCLUDED.graph_targets, semantic_evidence=EXCLUDED.semantic_evidence, evidence=EXCLUDED.evidence, recommendation=EXCLUDED.recommendation,
 correlation_id=EXCLUDED.correlation_id, trace_id=EXCLUDED.trace_id, causation_id=EXCLUDED.causation_id, replay_job_id=EXCLUDED.replay_job_id, event=EXCLUDED.event, updated_at=now()`,
		record.RunID, s.SignalID, s.TenantID, s.SourceID, recordAppID(s.AppID), recordDomain(s.Domain), recordUseCase(s.UseCase), s.SourceDomain, s.SourceAdapter,
		s.IngestionMode, s.Dataset, s.EventIDs, s.ArtifactIDs, s.SignalType, s.DetectorID, s.DetectorVersion, s.ModelVersion, s.SignalTime,
		s.ObservationTime, s.EffectiveTime, s.ProcessingTime, s.WindowStart, s.WindowEnd, s.Confidence, s.Severity,
		jsonArrayOrEmpty(s.EntitiesJSON), jsonOrEmpty(s.SupportingMetrics), jsonArrayOrEmpty(s.GraphTargetsJSON), jsonArrayOrEmpty(s.SemanticEvidenceJSON), jsonArrayOrEmpty(s.EvidenceJSON), nullableJSON(s.RecommendationJSON),
		s.CorrelationID, s.TraceID, s.CausationID, s.ReplayJobID, jsonOrEmpty(s.EventJSON))
	if err != nil {
		return fmt.Errorf("upsert marketops backtest signal: %w", err)
	}
	return nil
}

func upsertMarketOpsBacktestArtifact(ctx context.Context, executor statementExecutor, record storage.MarketOpsBacktestArtifactRecord) error {
	a := record.MarketOpsDSMArtifactRecord
	_, err := executor.ExecContext(ctx, `
INSERT INTO marketops_backtest_artifacts (
 run_id, artifact_id, tenant_id, app_id, domain, use_case, source_id, source_adapter, dataset, signal_id,
 signal_type, detector_id, severity, confidence, event_ids, subject_symbol, artifact_type,
 artifact, semantic_evidence, graph_targets, supporting_metrics, quality_issues, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,now())
ON CONFLICT (run_id, artifact_id) DO UPDATE SET
 tenant_id=EXCLUDED.tenant_id, app_id=EXCLUDED.app_id, domain=EXCLUDED.domain, use_case=EXCLUDED.use_case, source_id=EXCLUDED.source_id,
 source_adapter=EXCLUDED.source_adapter, dataset=EXCLUDED.dataset, signal_id=EXCLUDED.signal_id, signal_type=EXCLUDED.signal_type,
 detector_id=EXCLUDED.detector_id, severity=EXCLUDED.severity, confidence=EXCLUDED.confidence, event_ids=EXCLUDED.event_ids,
 subject_symbol=EXCLUDED.subject_symbol, artifact_type=EXCLUDED.artifact_type, artifact=EXCLUDED.artifact, semantic_evidence=EXCLUDED.semantic_evidence,
 graph_targets=EXCLUDED.graph_targets, supporting_metrics=EXCLUDED.supporting_metrics, quality_issues=EXCLUDED.quality_issues, updated_at=now()`,
		record.RunID, a.ArtifactID, a.TenantID, recordAppID(a.AppID), recordDomain(a.Domain), recordUseCase(a.UseCase), a.SourceID, a.SourceAdapter,
		a.Dataset, a.SignalID, a.SignalType, a.DetectorID, a.Severity, a.Confidence, a.EventIDs, a.SubjectSymbol, a.ArtifactType,
		jsonOrEmpty(a.ArtifactJSON), jsonOrEmpty(a.SemanticEvidenceJSON), jsonArrayOrEmpty(a.GraphTargetsJSON), jsonOrEmpty(a.SupportingMetrics), a.QualityIssues)
	if err != nil {
		return fmt.Errorf("upsert marketops backtest artifact: %w", err)
	}
	return nil
}

func upsertMarketOpsBacktestGraphProposal(ctx context.Context, executor statementExecutor, record storage.MarketOpsBacktestGraphProposalRecord) error {
	p := record.MarketOpsDSMGraphProposalRecord
	_, err := executor.ExecContext(ctx, `
INSERT INTO marketops_backtest_graph_proposals (
 run_id, proposal_id, tenant_id, app_id, domain, use_case, source_id, source_adapter, dataset, artifact_id,
 signal_id, signal_type, detector_id, severity, confidence, event_ids, subject_symbol, candidate_type,
 node_id, from_node, relationship, to_node, labels, properties, raw_candidate, status, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,now())
ON CONFLICT (run_id, proposal_id) DO UPDATE SET
 tenant_id=EXCLUDED.tenant_id, app_id=EXCLUDED.app_id, domain=EXCLUDED.domain, use_case=EXCLUDED.use_case, source_id=EXCLUDED.source_id,
 source_adapter=EXCLUDED.source_adapter, dataset=EXCLUDED.dataset, artifact_id=EXCLUDED.artifact_id, signal_id=EXCLUDED.signal_id,
 signal_type=EXCLUDED.signal_type, detector_id=EXCLUDED.detector_id, severity=EXCLUDED.severity, confidence=EXCLUDED.confidence,
 event_ids=EXCLUDED.event_ids, subject_symbol=EXCLUDED.subject_symbol, candidate_type=EXCLUDED.candidate_type, node_id=EXCLUDED.node_id,
 from_node=EXCLUDED.from_node, relationship=EXCLUDED.relationship, to_node=EXCLUDED.to_node, labels=EXCLUDED.labels,
 properties=EXCLUDED.properties, raw_candidate=EXCLUDED.raw_candidate, status=EXCLUDED.status, updated_at=now()`,
		record.RunID, p.ProposalID, p.TenantID, recordAppID(p.AppID), recordDomain(p.Domain), recordUseCase(p.UseCase), p.SourceID, p.SourceAdapter,
		p.Dataset, p.ArtifactID, p.SignalID, p.SignalType, p.DetectorID, p.Severity, p.Confidence, p.EventIDs, p.SubjectSymbol,
		p.CandidateType, p.NodeID, p.FromNode, p.Relationship, p.ToNode, p.Labels, jsonOrEmpty(p.PropertiesJSON), jsonOrEmpty(p.RawCandidate), graphProposalStatusOrDefault(p.Status))
	if err != nil {
		return fmt.Errorf("upsert marketops backtest graph proposal: %w", err)
	}
	return nil
}

func upsertMarketOpsBacktestPolicyResult(ctx context.Context, executor statementExecutor, record storage.MarketOpsBacktestPolicyResultRecord) error {
	_, err := executor.ExecContext(ctx, `
INSERT INTO marketops_backtest_policy_results (
 run_id, policy_result_id, proposal_id, artifact_id, signal_id, tenant_id, subject_symbol, candidate_type,
 recommendation, reason, policy_version, confidence, decision_inputs
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
ON CONFLICT (run_id, policy_result_id) DO UPDATE SET
 proposal_id=EXCLUDED.proposal_id, artifact_id=EXCLUDED.artifact_id, signal_id=EXCLUDED.signal_id, tenant_id=EXCLUDED.tenant_id,
 subject_symbol=EXCLUDED.subject_symbol, candidate_type=EXCLUDED.candidate_type, recommendation=EXCLUDED.recommendation,
 reason=EXCLUDED.reason, policy_version=EXCLUDED.policy_version, confidence=EXCLUDED.confidence, decision_inputs=EXCLUDED.decision_inputs`,
		record.RunID, record.PolicyResultID, record.ProposalID, record.ArtifactID, record.SignalID, record.TenantID, record.SubjectSymbol,
		record.CandidateType, record.Recommendation, record.Reason, record.PolicyVersion, record.Confidence, jsonOrEmpty(record.DecisionInputsJSON))
	if err != nil {
		return fmt.Errorf("upsert marketops backtest policy result: %w", err)
	}
	return nil
}

func (r *Repository) ListMarketOpsBacktestRuns(ctx context.Context, filter storage.MarketOpsBacktestRunFilter) ([]storage.MarketOpsBacktestRunRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsBacktestRunSelect+`
WHERE ($1='' OR tenant_id=$1) AND ($2='' OR app_id=$2) AND ($3='' OR domain=$3) AND ($4='' OR use_case=$4)
 AND ($5='' OR source_id=$5) AND ($6='' OR dataset=$6) AND ($7='' OR detector_id=$7) AND ($8='' OR status=$8)
ORDER BY started_at DESC LIMIT $9`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AppID), strings.TrimSpace(filter.Domain), strings.TrimSpace(filter.UseCase), strings.TrimSpace(filter.SourceID), strings.TrimSpace(filter.Dataset), strings.TrimSpace(filter.DetectorID), strings.TrimSpace(filter.Status), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops backtest runs: %w", err)
	}
	defer rows.Close()
	records := []storage.MarketOpsBacktestRunRecord{}
	for rows.Next() {
		rec, err := scanMarketOpsBacktestRun(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list marketops backtest runs rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetMarketOpsBacktestRun(ctx context.Context, runID string) (storage.MarketOpsBacktestRunRecord, error) {
	record, err := scanMarketOpsBacktestRun(r.db.QueryRowContext(ctx, marketOpsBacktestRunSelect+` WHERE run_id=$1`, strings.TrimSpace(runID)))
	if err != nil {
		return storage.MarketOpsBacktestRunRecord{}, err
	}
	return record, nil
}

const marketOpsBacktestRunSelect = `SELECT run_id, tenant_id, app_id, domain, use_case, source_id, source_adapter, dataset, detector_id, detector_version, status, requested_by, window_start, window_end, started_at, completed_at, filters, parameters, metrics, error_message, created_at, updated_at FROM marketops_backtest_runs`

type marketOpsBacktestRunScanner interface{ Scan(dest ...any) error }

func scanMarketOpsBacktestRun(scanner marketOpsBacktestRunScanner) (storage.MarketOpsBacktestRunRecord, error) {
	var record storage.MarketOpsBacktestRunRecord
	var completedAt sql.NullTime
	var errorMessage sql.NullString
	if err := scanner.Scan(&record.RunID, &record.TenantID, &record.AppID, &record.Domain, &record.UseCase, &record.SourceID, &record.SourceAdapter, &record.Dataset, &record.DetectorID, &record.DetectorVersion, &record.Status, &record.RequestedBy, &record.WindowStart, &record.WindowEnd, &record.StartedAt, &completedAt, &record.FiltersJSON, &record.ParametersJSON, &record.MetricsJSON, &errorMessage, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return storage.MarketOpsBacktestRunRecord{}, mapScanError("scan marketops backtest run", err)
	}
	if completedAt.Valid {
		record.CompletedAt = &completedAt.Time
	}
	record.ErrorMessage = errorMessage.String
	return record, nil
}

const marketOpsBacktestSignalSelect = `
SELECT run_id, signal_id, tenant_id, source_id, app_id, domain, use_case, source_domain, source_adapter, ingestion_mode, dataset,
 COALESCE(array_to_json(event_ids), '[]'::json)::text, COALESCE(array_to_json(artifact_ids), '[]'::json)::text,
 signal_type, detector_id, detector_version, model_version, signal_time, observation_time, effective_time,
 processing_time, window_start, window_end, confidence, severity, entities, supporting_metrics, graph_targets,
 semantic_evidence, evidence, recommendation, correlation_id, trace_id, causation_id, replay_job_id,
 '' AS broker_topic, -1::integer AS broker_partition, -1::bigint AS broker_offset, event, created_at, updated_at
FROM marketops_backtest_signals `

func (r *Repository) ListMarketOpsBacktestSignals(ctx context.Context, filter storage.MarketOpsBacktestSignalFilter) ([]storage.MarketOpsBacktestSignalRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsBacktestSignalSelect+`
WHERE run_id=$1 AND ($2='' OR tenant_id=$2) AND ($3='' OR signal_type=$3)
ORDER BY signal_time DESC LIMIT $4`, strings.TrimSpace(filter.RunID), strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.SignalType), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops backtest signals: %w", err)
	}
	defer rows.Close()
	records := []storage.MarketOpsBacktestSignalRecord{}
	for rows.Next() {
		var runID string
		sig, err := scanMarketOpsBacktestSignal(rows, &runID)
		if err != nil {
			return nil, err
		}
		records = append(records, storage.MarketOpsBacktestSignalRecord{RunID: runID, SignalLedgerRecord: sig})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list marketops backtest signals rows: %w", err)
	}
	return records, nil
}

func scanMarketOpsBacktestSignal(scanner signalLedgerScanner, runID *string) (storage.SignalLedgerRecord, error) {
	var record storage.SignalLedgerRecord
	var eventIDsJSON, artifactIDsJSON string
	if err := scanner.Scan(runID, &record.SignalID, &record.TenantID, &record.SourceID, &record.AppID, &record.Domain, &record.UseCase, &record.SourceDomain, &record.SourceAdapter, &record.IngestionMode, &record.Dataset, &eventIDsJSON, &artifactIDsJSON, &record.SignalType, &record.DetectorID, &record.DetectorVersion, &record.ModelVersion, &record.SignalTime, &record.ObservationTime, &record.EffectiveTime, &record.ProcessingTime, &record.WindowStart, &record.WindowEnd, &record.Confidence, &record.Severity, &record.EntitiesJSON, &record.SupportingMetrics, &record.GraphTargetsJSON, &record.SemanticEvidenceJSON, &record.EvidenceJSON, &record.RecommendationJSON, &record.CorrelationID, &record.TraceID, &record.CausationID, &record.ReplayJobID, &record.BrokerTopic, &record.BrokerPartition, &record.BrokerOffset, &record.EventJSON, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return storage.SignalLedgerRecord{}, mapScanError("scan marketops backtest signal", err)
	}
	if err := json.Unmarshal([]byte(eventIDsJSON), &record.EventIDs); err != nil {
		return storage.SignalLedgerRecord{}, err
	}
	if err := json.Unmarshal([]byte(artifactIDsJSON), &record.ArtifactIDs); err != nil {
		return storage.SignalLedgerRecord{}, err
	}
	return record, nil
}

func (r *Repository) ListMarketOpsBacktestGraphProposals(ctx context.Context, filter storage.MarketOpsBacktestGraphProposalFilter) ([]storage.MarketOpsBacktestGraphProposalRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT run_id, proposal_id, tenant_id, app_id, domain, use_case, source_id, source_adapter, dataset, artifact_id, signal_id, signal_type, detector_id, severity, confidence, COALESCE(array_to_json(event_ids), '[]'::json)::text, subject_symbol, candidate_type, node_id, from_node, relationship, to_node, COALESCE(array_to_json(labels), '[]'::json)::text, properties, raw_candidate, status, '' AS reviewed_by, '' AS decision_note, NULL::timestamptz AS decided_at, created_at, updated_at FROM marketops_backtest_graph_proposals
WHERE run_id=$1 AND ($2='' OR tenant_id=$2) AND ($3='' OR signal_type=$3) AND ($4='' OR subject_symbol=$4) AND ($5='' OR candidate_type=$5)
ORDER BY updated_at DESC LIMIT $6`, strings.TrimSpace(filter.RunID), strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.SignalType), strings.TrimSpace(filter.SubjectSymbol), strings.TrimSpace(filter.CandidateType), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops backtest graph proposals: %w", err)
	}
	defer rows.Close()
	records := []storage.MarketOpsBacktestGraphProposalRecord{}
	for rows.Next() {
		var runID string
		prop, err := scanMarketOpsBacktestGraphProposal(rows, &runID)
		if err != nil {
			return nil, err
		}
		records = append(records, storage.MarketOpsBacktestGraphProposalRecord{RunID: runID, MarketOpsDSMGraphProposalRecord: prop})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list marketops backtest graph proposals rows: %w", err)
	}
	return records, nil
}

func scanMarketOpsBacktestGraphProposal(scanner marketOpsDSMGraphProposalScanner, runID *string) (storage.MarketOpsDSMGraphProposalRecord, error) {
	var record storage.MarketOpsDSMGraphProposalRecord
	var eventIDsJSON, labelsJSON string
	if err := scanner.Scan(runID, &record.ProposalID, &record.TenantID, &record.AppID, &record.Domain, &record.UseCase, &record.SourceID, &record.SourceAdapter, &record.Dataset, &record.ArtifactID, &record.SignalID, &record.SignalType, &record.DetectorID, &record.Severity, &record.Confidence, &eventIDsJSON, &record.SubjectSymbol, &record.CandidateType, &record.NodeID, &record.FromNode, &record.Relationship, &record.ToNode, &labelsJSON, &record.PropertiesJSON, &record.RawCandidate, &record.Status, &record.ReviewedBy, &record.DecisionNote, &record.DecidedAt, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return storage.MarketOpsDSMGraphProposalRecord{}, mapScanError("scan marketops backtest graph proposal", err)
	}
	if err := json.Unmarshal([]byte(eventIDsJSON), &record.EventIDs); err != nil {
		return storage.MarketOpsDSMGraphProposalRecord{}, err
	}
	if err := json.Unmarshal([]byte(labelsJSON), &record.Labels); err != nil {
		return storage.MarketOpsDSMGraphProposalRecord{}, err
	}
	return record, nil
}

func (r *Repository) ListMarketOpsBacktestPolicyResults(ctx context.Context, filter storage.MarketOpsBacktestGraphProposalFilter) ([]storage.MarketOpsBacktestPolicyResultRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT run_id, policy_result_id, proposal_id, artifact_id, signal_id, tenant_id, subject_symbol, candidate_type, recommendation, reason, policy_version, confidence, decision_inputs, created_at FROM marketops_backtest_policy_results
WHERE run_id=$1 AND ($2='' OR tenant_id=$2) AND ($3='' OR subject_symbol=$3) AND ($4='' OR candidate_type=$4) AND ($5='' OR recommendation=$5)
ORDER BY created_at DESC LIMIT $6`, strings.TrimSpace(filter.RunID), strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.SubjectSymbol), strings.TrimSpace(filter.CandidateType), strings.TrimSpace(filter.Recommendation), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops backtest policy results: %w", err)
	}
	defer rows.Close()
	records := []storage.MarketOpsBacktestPolicyResultRecord{}
	for rows.Next() {
		var rec storage.MarketOpsBacktestPolicyResultRecord
		if err := rows.Scan(&rec.RunID, &rec.PolicyResultID, &rec.ProposalID, &rec.ArtifactID, &rec.SignalID, &rec.TenantID, &rec.SubjectSymbol, &rec.CandidateType, &rec.Recommendation, &rec.Reason, &rec.PolicyVersion, &rec.Confidence, &rec.DecisionInputsJSON, &rec.CreatedAt); err != nil {
			return nil, mapScanError("scan marketops backtest policy result", err)
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list marketops backtest policy results rows: %w", err)
	}
	return records, nil
}

func (r *Repository) ListMarketOpsBacktestNormalizedEvents(ctx context.Context, filter storage.MarketOpsBacktestEventFilter) ([]storage.NormalizedEventLedgerRecord, error) {
	rows, err := r.temporal().QueryContext(ctx, normalizedEventSelect+`
WHERE tenant_id=$1 AND ($2='' OR app_id=$2) AND ($3='' OR domain=$3) AND ($4='' OR use_case=$4)
 AND ($5='' OR source_id=$5) AND ($6='' OR source_adapter=$6) AND ($7='' OR dataset=$7)
 AND observation_time >= $8 AND observation_time < $9
 AND ($10::text[] = '{}'::text[] OR upper(COALESCE(normalized_payload->>'symbol', normalized_payload->>'ticker', normalized_payload->>'underlying_symbol', '')) = ANY($10::text[]))
ORDER BY observation_time ASC, event_id ASC LIMIT $11 OFFSET $12`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AppID), strings.TrimSpace(filter.Domain), strings.TrimSpace(filter.UseCase), strings.TrimSpace(filter.SourceID), strings.TrimSpace(filter.SourceAdapter), strings.TrimSpace(filter.Dataset), filter.WindowStart, filter.WindowEnd, pqArray(upperStrings(filter.Symbols)), clampLimit(filter.Limit), nonNegativeOffset(filter.Offset))
	if err != nil {
		return nil, fmt.Errorf("list marketops backtest normalized events: %w", err)
	}
	defer rows.Close()
	records := []storage.NormalizedEventLedgerRecord{}
	for rows.Next() {
		rec, err := scanNormalizedEventLedger(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list marketops backtest normalized events rows: %w", err)
	}
	return records, nil
}

func firstNonEmptyString(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
func upperStrings(values []string) []string {
	out := []string{}
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			out = append(out, strings.ToUpper(strings.TrimSpace(v)))
		}
	}
	return out
}

func (r *Repository) UpsertMarketOpsBacktestCalibrationSummary(ctx context.Context, record storage.MarketOpsBacktestCalibrationSummaryRecord) error {
	if strings.TrimSpace(record.SummaryID) == "" || strings.TrimSpace(record.TenantID) == "" {
		return fmt.Errorf("marketops backtest calibration summary_id and tenant_id are required")
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO marketops_backtest_calibration_summaries (
 summary_id, tenant_id, app_id, domain, use_case, source_id, dataset, detector_id, status_filter, requested_by,
 run_ids, run_count, succeeded_count, failed_count, zero_input_count, scanned, signals, artifacts, graph_proposals,
 policy_results, signal_yield, policy_results_per_signal, recommendation_counts, recommendation_shares,
 dominant_recommendation, filters, parameters
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27)
ON CONFLICT (summary_id) DO UPDATE SET
 tenant_id=EXCLUDED.tenant_id, app_id=EXCLUDED.app_id, domain=EXCLUDED.domain, use_case=EXCLUDED.use_case,
 source_id=EXCLUDED.source_id, dataset=EXCLUDED.dataset, detector_id=EXCLUDED.detector_id, status_filter=EXCLUDED.status_filter,
 requested_by=EXCLUDED.requested_by, run_ids=EXCLUDED.run_ids, run_count=EXCLUDED.run_count, succeeded_count=EXCLUDED.succeeded_count,
 failed_count=EXCLUDED.failed_count, zero_input_count=EXCLUDED.zero_input_count, scanned=EXCLUDED.scanned, signals=EXCLUDED.signals,
 artifacts=EXCLUDED.artifacts, graph_proposals=EXCLUDED.graph_proposals, policy_results=EXCLUDED.policy_results,
 signal_yield=EXCLUDED.signal_yield, policy_results_per_signal=EXCLUDED.policy_results_per_signal,
 recommendation_counts=EXCLUDED.recommendation_counts, recommendation_shares=EXCLUDED.recommendation_shares,
 dominant_recommendation=EXCLUDED.dominant_recommendation, filters=EXCLUDED.filters, parameters=EXCLUDED.parameters`,
		record.SummaryID, record.TenantID, recordAppID(record.AppID), recordDomain(record.Domain), recordUseCase(record.UseCase),
		record.SourceID, record.Dataset, record.DetectorID, record.StatusFilter, firstNonEmptyString(record.RequestedBy, "operator-local"),
		pqArray(record.RunIDs), record.RunCount, record.SucceededCount, record.FailedCount, record.ZeroInputCount, record.Scanned,
		record.Signals, record.Artifacts, record.GraphProposals, record.PolicyResults, record.SignalYield, record.PolicyResultsPerSignal,
		jsonOrEmpty(record.RecommendationCounts), jsonOrEmpty(record.RecommendationShares), jsonOrEmpty(record.DominantRecommendation),
		jsonOrEmpty(record.FiltersJSON), jsonOrEmpty(record.ParametersJSON))
	if err != nil {
		return fmt.Errorf("upsert marketops backtest calibration summary: %w", err)
	}
	return nil
}

func (r *Repository) ListMarketOpsBacktestCalibrationSummaries(ctx context.Context, filter storage.MarketOpsBacktestCalibrationSummaryFilter) ([]storage.MarketOpsBacktestCalibrationSummaryRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsBacktestCalibrationSummarySelect+`
WHERE ($1='' OR tenant_id=$1) AND ($2='' OR app_id=$2) AND ($3='' OR domain=$3) AND ($4='' OR use_case=$4)
 AND ($5='' OR source_id=$5) AND ($6='' OR dataset=$6) AND ($7='' OR detector_id=$7)
ORDER BY created_at DESC LIMIT $8`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AppID), strings.TrimSpace(filter.Domain), strings.TrimSpace(filter.UseCase), strings.TrimSpace(filter.SourceID), strings.TrimSpace(filter.Dataset), strings.TrimSpace(filter.DetectorID), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops backtest calibration summaries: %w", err)
	}
	defer rows.Close()
	records := []storage.MarketOpsBacktestCalibrationSummaryRecord{}
	for rows.Next() {
		rec, err := scanMarketOpsBacktestCalibrationSummary(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list marketops backtest calibration summaries rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetMarketOpsBacktestCalibrationSummary(ctx context.Context, summaryID string) (storage.MarketOpsBacktestCalibrationSummaryRecord, error) {
	record, err := scanMarketOpsBacktestCalibrationSummary(r.db.QueryRowContext(ctx, marketOpsBacktestCalibrationSummarySelect+` WHERE summary_id=$1`, strings.TrimSpace(summaryID)))
	if err != nil {
		return storage.MarketOpsBacktestCalibrationSummaryRecord{}, err
	}
	return record, nil
}

const marketOpsBacktestCalibrationSummarySelect = `SELECT summary_id, tenant_id, app_id, domain, use_case, source_id, dataset, detector_id, status_filter, requested_by,
 COALESCE(array_to_json(run_ids), '[]'::json)::text, run_count, succeeded_count, failed_count, zero_input_count, scanned, signals,
 artifacts, graph_proposals, policy_results, signal_yield, policy_results_per_signal, recommendation_counts, recommendation_shares,
 dominant_recommendation, filters, parameters, created_at FROM marketops_backtest_calibration_summaries`

type marketOpsBacktestCalibrationSummaryScanner interface{ Scan(dest ...any) error }

func scanMarketOpsBacktestCalibrationSummary(scanner marketOpsBacktestCalibrationSummaryScanner) (storage.MarketOpsBacktestCalibrationSummaryRecord, error) {
	var record storage.MarketOpsBacktestCalibrationSummaryRecord
	var runIDsJSON string
	if err := scanner.Scan(&record.SummaryID, &record.TenantID, &record.AppID, &record.Domain, &record.UseCase, &record.SourceID, &record.Dataset, &record.DetectorID, &record.StatusFilter, &record.RequestedBy, &runIDsJSON, &record.RunCount, &record.SucceededCount, &record.FailedCount, &record.ZeroInputCount, &record.Scanned, &record.Signals, &record.Artifacts, &record.GraphProposals, &record.PolicyResults, &record.SignalYield, &record.PolicyResultsPerSignal, &record.RecommendationCounts, &record.RecommendationShares, &record.DominantRecommendation, &record.FiltersJSON, &record.ParametersJSON, &record.CreatedAt); err != nil {
		return storage.MarketOpsBacktestCalibrationSummaryRecord{}, mapScanError("scan marketops backtest calibration summary", err)
	}
	if err := json.Unmarshal([]byte(runIDsJSON), &record.RunIDs); err != nil {
		return storage.MarketOpsBacktestCalibrationSummaryRecord{}, err
	}
	return record, nil
}

func (r *Repository) UpsertMarketOpsBacktestCalibrationBaseline(ctx context.Context, record storage.MarketOpsBacktestCalibrationBaselineRecord) error {
	if strings.TrimSpace(record.BaselineID) == "" || strings.TrimSpace(record.TenantID) == "" || strings.TrimSpace(record.Name) == "" || strings.TrimSpace(record.SummaryID) == "" {
		return fmt.Errorf("marketops backtest calibration baseline_id, tenant_id, name, and summary_id are required")
	}
	status := strings.TrimSpace(record.Status)
	if status == "" {
		status = storage.MarketOpsBacktestCalibrationBaselineStatusActive
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO marketops_backtest_calibration_baselines (
 baseline_id, tenant_id, app_id, domain, use_case, name, description, summary_id, detector_id, dataset, scope, status, created_by
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
ON CONFLICT (baseline_id) DO UPDATE SET
 tenant_id=EXCLUDED.tenant_id, app_id=EXCLUDED.app_id, domain=EXCLUDED.domain, use_case=EXCLUDED.use_case,
 name=EXCLUDED.name, description=EXCLUDED.description, summary_id=EXCLUDED.summary_id, detector_id=EXCLUDED.detector_id,
 dataset=EXCLUDED.dataset, scope=EXCLUDED.scope, status=EXCLUDED.status, created_by=EXCLUDED.created_by, updated_at=now()`,
		record.BaselineID, strings.TrimSpace(record.TenantID), recordAppID(record.AppID), recordDomain(record.Domain), recordUseCase(record.UseCase),
		strings.TrimSpace(record.Name), strings.TrimSpace(record.Description), strings.TrimSpace(record.SummaryID), strings.TrimSpace(record.DetectorID), strings.TrimSpace(record.Dataset),
		jsonOrEmpty(record.ScopeJSON), status, firstNonEmptyString(record.CreatedBy, "operator-local"))
	if err != nil {
		return fmt.Errorf("upsert marketops backtest calibration baseline: %w", err)
	}
	return nil
}

func (r *Repository) ListMarketOpsBacktestCalibrationBaselines(ctx context.Context, filter storage.MarketOpsBacktestCalibrationBaselineFilter) ([]storage.MarketOpsBacktestCalibrationBaselineRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsBacktestCalibrationBaselineSelect+`
WHERE ($1='' OR tenant_id=$1) AND ($2='' OR app_id=$2) AND ($3='' OR domain=$3) AND ($4='' OR use_case=$4)
 AND ($5='' OR detector_id=$5) AND ($6='' OR dataset=$6) AND ($7='' OR status=$7)
ORDER BY created_at DESC LIMIT $8`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AppID), strings.TrimSpace(filter.Domain), strings.TrimSpace(filter.UseCase), strings.TrimSpace(filter.DetectorID), strings.TrimSpace(filter.Dataset), strings.TrimSpace(filter.Status), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops backtest calibration baselines: %w", err)
	}
	defer rows.Close()
	records := []storage.MarketOpsBacktestCalibrationBaselineRecord{}
	for rows.Next() {
		rec, err := scanMarketOpsBacktestCalibrationBaseline(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list marketops backtest calibration baselines rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetMarketOpsBacktestCalibrationBaseline(ctx context.Context, baselineID string) (storage.MarketOpsBacktestCalibrationBaselineRecord, error) {
	record, err := scanMarketOpsBacktestCalibrationBaseline(r.db.QueryRowContext(ctx, marketOpsBacktestCalibrationBaselineSelect+` WHERE baseline_id=$1`, strings.TrimSpace(baselineID)))
	if err != nil {
		return storage.MarketOpsBacktestCalibrationBaselineRecord{}, err
	}
	return record, nil
}

const marketOpsBacktestCalibrationBaselineSelect = `SELECT baseline_id, tenant_id, app_id, domain, use_case, name, description, summary_id, detector_id, dataset, scope, status, created_by, created_at, updated_at FROM marketops_backtest_calibration_baselines`

type marketOpsBacktestCalibrationBaselineScanner interface{ Scan(dest ...any) error }

func scanMarketOpsBacktestCalibrationBaseline(scanner marketOpsBacktestCalibrationBaselineScanner) (storage.MarketOpsBacktestCalibrationBaselineRecord, error) {
	var record storage.MarketOpsBacktestCalibrationBaselineRecord
	if err := scanner.Scan(&record.BaselineID, &record.TenantID, &record.AppID, &record.Domain, &record.UseCase, &record.Name, &record.Description, &record.SummaryID, &record.DetectorID, &record.Dataset, &record.ScopeJSON, &record.Status, &record.CreatedBy, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return storage.MarketOpsBacktestCalibrationBaselineRecord{}, mapScanError("scan marketops backtest calibration baseline", err)
	}
	return record, nil
}

func (r *Repository) UpsertMarketOpsBacktestCalibrationComparison(ctx context.Context, record storage.MarketOpsBacktestCalibrationComparisonRecord) error {
	if strings.TrimSpace(record.ComparisonID) == "" || strings.TrimSpace(record.TenantID) == "" || strings.TrimSpace(record.BaselineID) == "" || strings.TrimSpace(record.BaselineSummaryID) == "" || strings.TrimSpace(record.CandidateSummaryID) == "" || strings.TrimSpace(record.Recommendation) == "" {
		return fmt.Errorf("marketops backtest calibration comparison_id, tenant_id, baseline_id, summary ids, and recommendation are required")
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO marketops_backtest_calibration_comparisons (
 comparison_id, tenant_id, baseline_id, baseline_summary_id, candidate_summary_id, detector_id, dataset,
 comparison_metrics, recommendation, recommendation_reason, created_by
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT (comparison_id) DO UPDATE SET
 tenant_id=EXCLUDED.tenant_id, baseline_id=EXCLUDED.baseline_id, baseline_summary_id=EXCLUDED.baseline_summary_id,
 candidate_summary_id=EXCLUDED.candidate_summary_id, detector_id=EXCLUDED.detector_id, dataset=EXCLUDED.dataset,
 comparison_metrics=EXCLUDED.comparison_metrics, recommendation=EXCLUDED.recommendation,
 recommendation_reason=EXCLUDED.recommendation_reason, created_by=EXCLUDED.created_by`,
		record.ComparisonID, strings.TrimSpace(record.TenantID), strings.TrimSpace(record.BaselineID), strings.TrimSpace(record.BaselineSummaryID), strings.TrimSpace(record.CandidateSummaryID),
		strings.TrimSpace(record.DetectorID), strings.TrimSpace(record.Dataset), jsonOrEmpty(record.ComparisonMetricsJSON), strings.TrimSpace(record.Recommendation), strings.TrimSpace(record.RecommendationReason), firstNonEmptyString(record.CreatedBy, "operator-local"))
	if err != nil {
		return fmt.Errorf("upsert marketops backtest calibration comparison: %w", err)
	}
	return nil
}

func (r *Repository) ListMarketOpsBacktestCalibrationComparisons(ctx context.Context, filter storage.MarketOpsBacktestCalibrationComparisonFilter) ([]storage.MarketOpsBacktestCalibrationComparisonRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsBacktestCalibrationComparisonSelect+`
WHERE ($1='' OR tenant_id=$1) AND ($2='' OR baseline_id=$2) AND ($3='' OR detector_id=$3) AND ($4='' OR dataset=$4) AND ($5='' OR recommendation=$5)
ORDER BY created_at DESC LIMIT $6`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.BaselineID), strings.TrimSpace(filter.DetectorID), strings.TrimSpace(filter.Dataset), strings.TrimSpace(filter.Recommendation), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops backtest calibration comparisons: %w", err)
	}
	defer rows.Close()
	records := []storage.MarketOpsBacktestCalibrationComparisonRecord{}
	for rows.Next() {
		rec, err := scanMarketOpsBacktestCalibrationComparison(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list marketops backtest calibration comparisons rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetMarketOpsBacktestCalibrationComparison(ctx context.Context, comparisonID string) (storage.MarketOpsBacktestCalibrationComparisonRecord, error) {
	record, err := scanMarketOpsBacktestCalibrationComparison(r.db.QueryRowContext(ctx, marketOpsBacktestCalibrationComparisonSelect+` WHERE comparison_id=$1`, strings.TrimSpace(comparisonID)))
	if err != nil {
		return storage.MarketOpsBacktestCalibrationComparisonRecord{}, err
	}
	return record, nil
}

const marketOpsBacktestCalibrationComparisonSelect = `SELECT comparison_id, tenant_id, baseline_id, baseline_summary_id, candidate_summary_id, detector_id, dataset, comparison_metrics, recommendation, recommendation_reason, created_by, created_at FROM marketops_backtest_calibration_comparisons`

type marketOpsBacktestCalibrationComparisonScanner interface{ Scan(dest ...any) error }

func scanMarketOpsBacktestCalibrationComparison(scanner marketOpsBacktestCalibrationComparisonScanner) (storage.MarketOpsBacktestCalibrationComparisonRecord, error) {
	var record storage.MarketOpsBacktestCalibrationComparisonRecord
	if err := scanner.Scan(&record.ComparisonID, &record.TenantID, &record.BaselineID, &record.BaselineSummaryID, &record.CandidateSummaryID, &record.DetectorID, &record.Dataset, &record.ComparisonMetricsJSON, &record.Recommendation, &record.RecommendationReason, &record.CreatedBy, &record.CreatedAt); err != nil {
		return storage.MarketOpsBacktestCalibrationComparisonRecord{}, mapScanError("scan marketops backtest calibration comparison", err)
	}
	return record, nil
}

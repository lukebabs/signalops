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

const cyberOpsLifecycleVersion = "cyberops-lifecycle-v1"

type lifecyclePolicyRow struct {
	ID, Version, Mode, Hash string
	Selector                []byte
}

type cyberEvidence struct {
	Source, Destination, Protocol string
	Port                          int
}

// PersistPolicySignalLifecycle is intentionally opt-in. A tenant with no enabled
// CyberOps policy uses the historical generic lifecycle unchanged.
func (r *Repository) PersistPolicySignalLifecycle(ctx context.Context, signal storage.SignalLedgerRecord, alerts []storage.AlertLedgerRecord, insights []storage.InsightLedgerRecord) (bool, error) {
	if signal.AppID != "cyberops" {
		return false, nil
	}
	policies, err := r.cyberOpsPolicies(ctx, signal.TenantID)
	if err != nil {
		return false, err
	}
	if len(policies) == 0 {
		return false, nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin cyberops lifecycle: %w", err)
	}
	defer tx.Rollback()
	if err := upsertSignalLedger(ctx, tx, signal); err != nil {
		return false, err
	}
	evidence := cyberEvidenceFromSignal(signal)
	for _, policy := range policies {
		if err := r.applyCyberOpsPolicy(ctx, tx, signal, policy, evidence); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit cyberops lifecycle: %w", err)
	}
	return true, nil
}

func (r *Repository) cyberOpsPolicies(ctx context.Context, tenant string) ([]lifecyclePolicyRow, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT policy_id,policy_version,mode,selector,policy_hash FROM cyberops_lifecycle_policies WHERE tenant_id=$1 AND mode <> 'disabled' ORDER BY policy_id`, tenant)
	if err != nil {
		return nil, fmt.Errorf("load cyberops lifecycle policies: %w", err)
	}
	defer rows.Close()
	out := []lifecyclePolicyRow{}
	for rows.Next() {
		var p lifecyclePolicyRow
		if err := rows.Scan(&p.ID, &p.Version, &p.Mode, &p.Selector, &p.Hash); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) applyCyberOpsPolicy(ctx context.Context, tx *sql.Tx, signal storage.SignalLedgerRecord, policy lifecyclePolicyRow, e cyberEvidence) error {
	disposition, reason := "record_only", "signal retained as durable evidence"
	fingerprint := strings.Join([]string{signal.TenantID, e.Source, e.Destination, e.Protocol, fmt.Sprint(e.Port)}, "|")
	switch policy.ID {
	case "external-deny":
		if signal.SignalType != "cyberops.firewall.external_deny" {
			return nil
		}
		reason = "external firewall deny is evidence-only under lifecycle v1"
	case "public-service-exposure":
		if signal.SignalType != "cyberops.firewall.new_public_service_exposure" {
			return nil
		}
		approved, err := approvedCyberOpsService(ctx, tx, signal.TenantID, e)
		if err != nil {
			return err
		}
		if approved {
			reason = "approved public service; retained as evidence only"
		} else {
			disposition = "create_or_update_insight"
			reason = "unapproved public service exposure requires review"
		}
	case "port-scan":
		if signal.SignalType != "cyberops.firewall.external_deny" {
			return nil
		}
		count, err := deniedPortCount(ctx, tx, signal.TenantID, e, signal.SignalTime)
		if err != nil {
			return err
		}
		if count < 10 {
			return nil
		}
		disposition = "create_or_update_alert"
		reason = fmt.Sprintf("%d distinct denied destination ports from public source in five minutes", count)
		fingerprint = strings.Join([]string{signal.TenantID, e.Source, e.Destination, e.Protocol, signal.SignalTime.UTC().Truncate(5 * time.Minute).Format(time.RFC3339)}, "|")
	default:
		return nil
	}
	episodeID := lifecycleID("cybep", policy.ID+"|"+fingerprint)
	decisionID := lifecycleID("cybdec", policy.ID+"|"+signal.SignalID)
	insightID, alertID := "", ""
	if policy.Mode == "enforced" && disposition == "create_or_update_insight" {
		insightID = lifecycleID("insight", policy.Version+"|"+fingerprint)
		if err := upsertInsightLedger(ctx, tx, cyberInsight(signal, insightID, reason, policy, fingerprint)); err != nil {
			return err
		}
	}
	if policy.Mode == "enforced" && disposition == "create_or_update_alert" {
		alertID = lifecycleID("alert", policy.Version+"|"+fingerprint)
		if err := upsertAlertLedger(ctx, tx, cyberAlert(signal, alertID, reason, policy, fingerprint)); err != nil {
			return err
		}
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO cyberops_lifecycle_episodes (episode_id,tenant_id,policy_id,fingerprint,disposition,first_observed_at,last_observed_at,observation_count,signal_ids,insight_id,alert_id) VALUES ($1,$2,$3,$4,$5,$6,$6,1,$7,$8,$9) ON CONFLICT (tenant_id,policy_id,fingerprint) DO UPDATE SET last_observed_at=GREATEST(cyberops_lifecycle_episodes.last_observed_at,EXCLUDED.last_observed_at), observation_count=cyberops_lifecycle_episodes.observation_count+1, signal_ids=(ARRAY(SELECT DISTINCT x FROM unnest(cyberops_lifecycle_episodes.signal_ids || EXCLUDED.signal_ids) x ORDER BY x DESC LIMIT 100)), insight_id=CASE WHEN EXCLUDED.insight_id<>'' THEN EXCLUDED.insight_id ELSE cyberops_lifecycle_episodes.insight_id END, alert_id=CASE WHEN EXCLUDED.alert_id<>'' THEN EXCLUDED.alert_id ELSE cyberops_lifecycle_episodes.alert_id END`, episodeID, signal.TenantID, policy.ID, fingerprint, disposition, signal.SignalTime, []string{signal.SignalID}, insightID, alertID)
	if err != nil {
		return fmt.Errorf("upsert cyberops lifecycle episode: %w", err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO cyberops_lifecycle_decisions (decision_id,tenant_id,signal_id,policy_id,policy_version,mode,disposition,reason,fingerprint,episode_id,insight_id,alert_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) ON CONFLICT (tenant_id,signal_id,policy_id) DO NOTHING`, decisionID, signal.TenantID, signal.SignalID, policy.ID, policy.Version, policy.Mode, disposition, reason, fingerprint, episodeID, insightID, alertID)
	return err
}

func approvedCyberOpsService(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, tenant string, e cyberEvidence) (bool, error) {
	var exists bool
	err := q.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM cyberops_approved_services WHERE tenant_id=$1 AND destination_ip=$2::inet AND protocol=$3 AND destination_port=$4)`, tenant, e.Destination, e.Protocol, e.Port).Scan(&exists)
	return exists, err
}
func deniedPortCount(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, tenant string, e cyberEvidence, at time.Time) (int, error) {
	var count int
	err := q.QueryRowContext(ctx, `SELECT COUNT(DISTINCT semantic_evidence->0->>'destination_port') FROM signal_ledger WHERE tenant_id=$1 AND signal_type='cyberops.firewall.external_deny' AND signal_time >= $2 AND signal_time <= $3 AND entities->0->>'value'=$4 AND entities->1->>'value'=$5 AND semantic_evidence->0->>'protocol'=$6`, tenant, at.Add(-5*time.Minute), at, e.Source, e.Destination, e.Protocol).Scan(&count)
	return count, err
}
func cyberEvidenceFromSignal(s storage.SignalLedgerRecord) cyberEvidence {
	var entities []map[string]any
	var semantic []map[string]any
	_ = json.Unmarshal(s.EntitiesJSON, &entities)
	_ = json.Unmarshal(s.SemanticEvidenceJSON, &semantic)
	e := cyberEvidence{}
	for _, v := range entities {
		if v["role"] == "source" {
			e.Source = fmt.Sprint(v["value"])
		}
		if v["role"] == "destination" {
			e.Destination = fmt.Sprint(v["value"])
		}
	}
	if len(semantic) > 0 {
		e.Protocol = strings.ToLower(fmt.Sprint(semantic[0]["protocol"]))
		switch v := semantic[0]["destination_port"].(type) {
		case float64:
			e.Port = int(v)
		case int:
			e.Port = v
		case string:
			fmt.Sscan(v, &e.Port)
		}
	}
	return e
}
func lifecycleID(prefix, value string) string {
	sum := sha256.Sum256([]byte(value))
	return prefix + "_" + hex.EncodeToString(sum[:16])
}
func lifecycleMetadata(policy lifecyclePolicyRow, fingerprint, reason string) []byte {
	out, _ := json.Marshal(map[string]any{"lifecycle_policy_version": policy.Version, "lifecycle_policy_id": policy.ID, "lifecycle_policy_hash": policy.Hash, "lifecycle_fingerprint": fingerprint, "lifecycle_reason": reason})
	return out
}
func cyberInsight(s storage.SignalLedgerRecord, id, reason string, p lifecyclePolicyRow, fingerprint string) storage.InsightLedgerRecord {
	return storage.InsightLedgerRecord{InsightID: id, TenantID: s.TenantID, SourceID: s.SourceID, AppID: s.AppID, Domain: s.Domain, UseCase: s.UseCase, SourceDomain: s.SourceDomain, SourceAdapter: s.SourceAdapter, Dataset: s.Dataset, SignalID: s.SignalID, DetectorID: s.DetectorID, InsightType: s.SignalType, Status: storage.InsightStatusActive, Title: "Unapproved public service exposure", Summary: reason, Confidence: s.Confidence, Severity: "low", EventIDs: s.EventIDs, EntitiesJSON: s.EntitiesJSON, SupportingMetrics: s.SupportingMetrics, SemanticEvidenceJSON: s.SemanticEvidenceJSON, RecommendationJSON: s.RecommendationJSON, CorrelationID: s.CorrelationID, ObservedAt: s.SignalTime, MetadataJSON: lifecycleMetadata(p, fingerprint, reason)}
}
func cyberAlert(s storage.SignalLedgerRecord, id, reason string, p lifecyclePolicyRow, fingerprint string) storage.AlertLedgerRecord {
	return storage.AlertLedgerRecord{AlertID: id, TenantID: s.TenantID, SourceID: s.SourceID, AppID: s.AppID, Domain: s.Domain, UseCase: s.UseCase, SourceDomain: s.SourceDomain, SourceAdapter: s.SourceAdapter, Dataset: s.Dataset, SignalID: s.SignalID, DetectorID: s.DetectorID, AlertType: "cyberops.network.port_scan", Severity: "high", Status: storage.AlertStatusOpen, Title: "Potential external port scan", Summary: reason, Confidence: s.Confidence, EventIDs: s.EventIDs, EntitiesJSON: s.EntitiesJSON, EvidenceJSON: s.EvidenceJSON, RecommendationJSON: s.RecommendationJSON, CorrelationID: s.CorrelationID, FirstObservedAt: s.SignalTime, LastObservedAt: s.SignalTime, MetadataJSON: lifecycleMetadata(p, fingerprint, reason)}
}

func (r *Repository) ListCyberOpsLifecyclePolicies(ctx context.Context, tenantID string) ([]storage.CyberOpsLifecyclePolicy, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT tenant_id,policy_id,policy_version,mode,selector,policy_hash,updated_at FROM cyberops_lifecycle_policies WHERE tenant_id=$1 ORDER BY policy_id`, strings.TrimSpace(tenantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []storage.CyberOpsLifecyclePolicy{}
	for rows.Next() {
		var item storage.CyberOpsLifecyclePolicy
		if err := rows.Scan(&item.TenantID, &item.PolicyID, &item.PolicyVersion, &item.Mode, &item.SelectorJSON, &item.PolicyHash, &item.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
func (r *Repository) ListCyberOpsApprovedServices(ctx context.Context, tenantID string) ([]storage.CyberOpsApprovedService, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT tenant_id,destination_ip::text,protocol,destination_port,approved_by,reason,created_at FROM cyberops_approved_services WHERE tenant_id=$1 ORDER BY destination_ip,protocol,destination_port`, strings.TrimSpace(tenantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []storage.CyberOpsApprovedService{}
	for rows.Next() {
		var item storage.CyberOpsApprovedService
		if err := rows.Scan(&item.TenantID, &item.DestinationIP, &item.Protocol, &item.DestinationPort, &item.ApprovedBy, &item.Reason, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
func (r *Repository) UpsertCyberOpsApprovedService(ctx context.Context, s storage.CyberOpsApprovedService, actor string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var before []byte
	_ = tx.QueryRowContext(ctx, `SELECT jsonb_build_object('destination_ip',destination_ip::text,'protocol',protocol,'destination_port',destination_port,'reason',reason) FROM cyberops_approved_services WHERE tenant_id=$1 AND destination_ip=$2::inet AND protocol=$3 AND destination_port=$4`, s.TenantID, s.DestinationIP, s.Protocol, s.DestinationPort).Scan(&before)
	_, err = tx.ExecContext(ctx, `INSERT INTO cyberops_approved_services (tenant_id,destination_ip,protocol,destination_port,approved_by,reason) VALUES ($1,$2::inet,$3,$4,$5,$6) ON CONFLICT (tenant_id,destination_ip,protocol,destination_port) DO UPDATE SET approved_by=EXCLUDED.approved_by,reason=EXCLUDED.reason`, s.TenantID, s.DestinationIP, s.Protocol, s.DestinationPort, s.ApprovedBy, s.Reason)
	if err != nil {
		return err
	}
	after, _ := json.Marshal(map[string]any{"destination_ip": s.DestinationIP, "protocol": s.Protocol, "destination_port": s.DestinationPort, "reason": s.Reason})
	_, err = tx.ExecContext(ctx, `INSERT INTO cyberops_lifecycle_policy_audit (tenant_id,mutation,destination_ip,protocol,destination_port,actor,before_value,after_value) VALUES ($1,'approve_service',$2::inet,$3,$4,$5,$6,$7)`, s.TenantID, s.DestinationIP, s.Protocol, s.DestinationPort, actor, jsonOrEmpty(before), after)
	if err != nil {
		return err
	}
	return tx.Commit()
}
func (r *Repository) DeleteCyberOpsApprovedService(ctx context.Context, tenantID, destinationIP, protocol string, port int, actor string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var before []byte
	err = tx.QueryRowContext(ctx, `DELETE FROM cyberops_approved_services WHERE tenant_id=$1 AND destination_ip=$2::inet AND protocol=$3 AND destination_port=$4 RETURNING jsonb_build_object('destination_ip',destination_ip::text,'protocol',protocol,'destination_port',destination_port,'reason',reason)`, tenantID, destinationIP, protocol, port).Scan(&before)
	if err == sql.ErrNoRows {
		return storage.ErrNotFound
	}
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO cyberops_lifecycle_policy_audit (tenant_id,mutation,destination_ip,protocol,destination_port,actor,before_value) VALUES ($1,'remove_service',$2::inet,$3,$4,$5,$6)`, tenantID, destinationIP, protocol, port, actor, before)
	if err != nil {
		return err
	}
	return tx.Commit()
}
func (r *Repository) ListCyberOpsLifecycleEpisodes(ctx context.Context, tenantID string, max int) ([]storage.CyberOpsLifecycleEpisode, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT episode_id,tenant_id,policy_id,fingerprint,disposition,first_observed_at,last_observed_at,observation_count,signal_ids,insight_id,alert_id FROM cyberops_lifecycle_episodes WHERE tenant_id=$1 ORDER BY last_observed_at DESC LIMIT $2`, tenantID, clampLimit(max))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []storage.CyberOpsLifecycleEpisode{}
	for rows.Next() {
		var x storage.CyberOpsLifecycleEpisode
		if err := rows.Scan(&x.EpisodeID, &x.TenantID, &x.PolicyID, &x.Fingerprint, &x.Disposition, &x.FirstObservedAt, &x.LastObservedAt, &x.ObservationCount, &x.SignalIDs, &x.InsightID, &x.AlertID); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *Repository) ListCyberOpsLifecycleDecisions(ctx context.Context, tenantID string, max int) ([]storage.CyberOpsLifecycleDecision, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT decision_id,tenant_id,signal_id,policy_id,policy_version,mode,disposition,reason,fingerprint,episode_id,insight_id,alert_id,created_at FROM cyberops_lifecycle_decisions WHERE tenant_id=$1 ORDER BY created_at DESC LIMIT $2`, tenantID, clampLimit(max))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []storage.CyberOpsLifecycleDecision{}
	for rows.Next() {
		var x storage.CyberOpsLifecycleDecision
		if err := rows.Scan(&x.DecisionID, &x.TenantID, &x.SignalID, &x.PolicyID, &x.PolicyVersion, &x.Mode, &x.Disposition, &x.Reason, &x.Fingerprint, &x.EpisodeID, &x.InsightID, &x.AlertID, &x.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

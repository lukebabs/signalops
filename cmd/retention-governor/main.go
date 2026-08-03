package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lukebabs/signalops/internal/config"
	"log/slog"
	"os"
	"strings"
	"time"
)

type policy struct {
	ID, AppID, Domain, DataClass, Mode, PreservationRule, Description string
	RetentionDays                                                     int
}
type target struct {
	table, timeColumn, where string
	args                     []any
	preserveReceipts         bool
}

func main() {
	if err := run(context.Background()); err != nil {
		slog.Error("retention governance failed", "error", err)
		os.Exit(1)
	}
}
func run(ctx context.Context) error {
	tenant := flag.String("tenant-id", "tenant-local", "")
	execute := flag.Bool("execute", false, "apply only policies explicitly set to enforced")
	policyID := flag.String("policy-id", "", "optional policy")
	flag.Parse()
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		return fmt.Errorf("SIGNALOPS_DATABASE_URL is required")
	}
	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	policies, err := loadPolicies(ctx, db, *tenant, *policyID)
	if err != nil {
		return err
	}
	for _, p := range policies {
		if err := runPolicy(ctx, db, *tenant, p, *execute); err != nil {
			return err
		}
	}
	return nil
}
func loadPolicies(ctx context.Context, db *sql.DB, tenant, id string) ([]policy, error) {
	rows, err := db.QueryContext(ctx, `SELECT policy_id,app_id,domain,data_class,retention_days,mode,preservation_rule,description FROM retention_policies WHERE tenant_id=$1 AND ($2='' OR policy_id=$2) ORDER BY policy_id`, tenant, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []policy{}
	for rows.Next() {
		var p policy
		if err = rows.Scan(&p.ID, &p.AppID, &p.Domain, &p.DataClass, &p.RetentionDays, &p.Mode, &p.PreservationRule, &p.Description); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
func runPolicy(ctx context.Context, db *sql.DB, tenant string, p policy, execute bool) error {
	runID := fmt.Sprintf("retention_%s_%d", strings.ReplaceAll(p.ID, ".", "_"), time.Now().UTC().UnixNano())
	mode := "dry_run"
	if execute && p.Mode == "enforced" {
		mode = "enforced"
	}
	_, err := db.ExecContext(ctx, `INSERT INTO retention_runs(run_id,tenant_id,policy_id,mode,status) VALUES($1,$2,$3,$4,'running')`, runID, tenant, p.ID, mode)
	if err != nil {
		return err
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -p.RetentionDays)
	targets := policyTargets(p, tenant, cutoff)
	var candidates, affected int64
	var oldest, newest *time.Time
	details := []map[string]any{}
	for _, t := range targets {
		count, first, last, err := measure(ctx, db, t)
		if err != nil {
			return finish(ctx, db, runID, "failed", 0, 0, nil, nil, map[string]any{"error": err.Error()})
		}
		candidates += count
		if first != nil && (oldest == nil || first.Before(*oldest)) {
			oldest = first
		}
		if last != nil && (newest == nil || last.After(*newest)) {
			newest = last
		}
		detail := map[string]any{"table": t.table, "candidate_rows": count}
		if mode == "enforced" && count > 0 {
			if t.preserveReceipts {
				receipts, err := preserveCyberReceipts(ctx, db, tenant, cutoff)
				if err != nil {
					return finish(ctx, db, runID, "blocked", candidates, affected, oldest, newest, map[string]any{"error": err.Error(), "table": t.table})
				}
				detail["evidence_receipts"] = receipts
			}
			result, err := db.ExecContext(ctx, `DELETE FROM `+t.table+` WHERE `+t.where, t.args...)
			if err != nil {
				return finish(ctx, db, runID, "failed", candidates, affected, oldest, newest, map[string]any{"error": err.Error(), "table": t.table})
			}
			n, _ := result.RowsAffected()
			affected += n
			detail["affected_rows"] = n
		}
		details = append(details, detail)
	}
	return finish(ctx, db, runID, "succeeded", candidates, affected, oldest, newest, map[string]any{"policy": p.ID, "mode": mode, "cutoff": cutoff, "targets": details, "preservation_rule": p.PreservationRule})
}
func policyTargets(p policy, tenant string, cutoff time.Time) []target {
	arg := []any{tenant, cutoff}
	switch p.ID {
	case "marketops.raw_events_30d":
		return []target{{"raw_event_ledger", "observation_time", "tenant_id=$1 AND app_id='marketops' AND observation_time < $2", arg, false}}
	case "marketops.equity_metadata_12m":
		return []target{{"normalized_event_ledger", "observation_time", "tenant_id=$1 AND app_id='marketops' AND dataset='equity_eod_prices' AND observation_time < $2", arg, false}}
	case "marketops.options_detail_3m":
		return []target{{"marketops_options_chain_daily", "trade_date", "tenant_id=$1 AND trade_date < $2::date", arg, false}}
	case "marketops.financial_metadata_4y":
		return []target{{"marketops_financial_snapshots", "evaluation_date", "tenant_id=$1 AND evaluation_date < $2::date", arg, false}, {"marketops_financial_statements", "fiscal_period_end", "tenant_id=$1 AND fiscal_period_end < $2::date AND NOT EXISTS (SELECT 1 FROM marketops_financial_snapshots f WHERE marketops_financial_statements.statement_id = ANY(f.statement_ids))", arg, false}}
	case "cyberops.raw_events_30d":
		return []target{{"raw_event_ledger", "observation_time", "tenant_id=$1 AND app_id='cyberops' AND observation_time < $2", arg, false}, {"cyberops_connect_raw_events", "occurred_at", "tenant_id=$1 AND occurred_at < $2", arg, false}}
	case "cyberops.high_resolution_30d":
		return []target{{"normalized_event_ledger", "observation_time", "tenant_id=$1 AND app_id='cyberops' AND observation_time < $2", arg, true}, {"cyberops_iot_hourly_features", "hour", "tenant_id=$1 AND hour < $2", arg, false}}
	case "cyberops.metadata_12m":
		return []target{{"cyberops_iot_daily_features", "feature_date", "tenant_id=$1 AND feature_date < $2::date", arg, false}, {"algorithm_results", "created_at", "tenant_id=$1 AND algorithm_id='signalops.algorithms.cyberops_iot_anomaly_v1' AND created_at < $2", arg, false}}
	case "platform.idempotency_35d":
		return []target{{"idempotency_records", "last_seen_at", "tenant_id=$1 AND last_seen_at < $2", arg, false}}
	}
	return nil
}
func measure(ctx context.Context, db *sql.DB, t target) (int64, *time.Time, *time.Time, error) {
	query := `SELECT count(*),min(` + t.timeColumn + `),max(` + t.timeColumn + `) FROM ` + t.table + ` WHERE ` + t.where
	var count int64
	var first, last sql.NullTime
	err := db.QueryRowContext(ctx, query, t.args...).Scan(&count, &first, &last)
	if err != nil {
		return 0, nil, nil, err
	}
	var f, l *time.Time
	if first.Valid {
		f = &first.Time
	}
	if last.Valid {
		l = &last.Time
	}
	return count, f, l, nil
}
func preserveCyberReceipts(ctx context.Context, db *sql.DB, tenant string, cutoff time.Time) (int64, error) {
	result, err := db.ExecContext(ctx, `INSERT INTO retention_evidence_receipts(tenant_id,app_id,domain,event_id,source_id,dataset,observed_at,payload_hash,parser_version,metadata) SELECT n.tenant_id,n.app_id,n.domain,n.event_id,n.source_id,n.dataset,n.observation_time,md5(n.normalized_payload::text),'opnsense.filterlog.v1',jsonb_build_object('event_type',n.normalized_payload->>'event_type','schema_id',n.normalized_payload->>'schema_id','quality',n.normalized_payload #> '{metadata,quality}','preserved_for','derived_signal') FROM normalized_event_ledger n WHERE n.tenant_id=$1 AND n.app_id='cyberops' AND n.observation_time < $2 AND EXISTS (SELECT 1 FROM signal_ledger s WHERE s.tenant_id=n.tenant_id AND n.event_id=ANY(s.event_ids)) ON CONFLICT (tenant_id,event_id) DO NOTHING`, tenant, cutoff)
	if err != nil {
		return 0, err
	}
	n, _ := result.RowsAffected()
	return n, nil
}
func finish(ctx context.Context, db *sql.DB, id, status string, candidates, affected int64, oldest, newest *time.Time, detail map[string]any) error {
	raw, _ := json.Marshal(detail)
	_, err := db.ExecContext(ctx, `UPDATE retention_runs SET status=$2,candidate_rows=$3,affected_rows=$4,oldest_candidate_at=$5,newest_candidate_at=$6,detail=$7,completed_at=now() WHERE run_id=$1`, id, status, candidates, affected, oldest, newest, raw)
	return err
}

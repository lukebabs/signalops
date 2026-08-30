package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lukebabs/signalops/internal/adapters/marketdata/fmp"
	"net"
	"os"
	"strings"
	"time"
)

const worker = "subscriber-global-eod-reconciler"
const taskType = "subscriber_global_annual_financial"
const algo = "marketops.fundamental_annual.fmp"
const version = "v2"
const contract = "subscriber-global-annual-financial-capture/v2"
const baseline = "fmp-starter-annual-v1"

type task struct {
	id, asset, symbol string
	attempt, max      int
}

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "annual financial task worker failed:", err)
		os.Exit(1)
	}
}
func run(ctx context.Context, args []string) error {
	f := flag.NewFlagSet("subscriber-global-annual-financial-task-worker", flag.ContinueOnError)
	u := f.String("database-url", os.Getenv("SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL"), "dedicated primary URL")
	d := f.String("session-date", "", "completed session")
	n := f.Int("max-assets", 1000, "warm asset limit")
	dry := f.Bool("dry-run", false, "no writes")
	exec := f.Bool("execute", false, "write")
	if err := f.Parse(args); err != nil {
		return err
	}
	if *dry == *exec || strings.TrimSpace(*u) == "" || *n < 1 || *n > 1000 {
		return errors.New("pass one mode, database URL, and 1-1000 assets")
	}
	s, err := session(*d)
	if err != nil {
		return err
	}
	db, err := sql.Open("pgx", *u)
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	defer db.Close()
	if _, err = db.ExecContext(ctx, "SET ROLE signalops_subscriber_global_eod"); err != nil {
		return err
	}
	defer db.ExecContext(context.Background(), "RESET ROLE")
	if *dry {
		var c int
		err = db.QueryRowContext(ctx, `SELECT count(*) FROM (SELECT 1 FROM subscriber_global_warm_eod_assets LIMIT $1)x`, *n).Scan(&c)
		fmt.Printf("dry_run=true warm_assets=%d session_date=%s\n", c, s.Format("2006-01-02"))
		return err
	}
	if err = seed(ctx, db, s, *n); err != nil {
		return err
	}
	cfg := fmp.LoadClientConfigFromEnv()
	cfg.MinRequestInterval = 250 * time.Millisecond
	c, err := fmp.NewClient(cfg)
	if err != nil {
		return err
	}
	processed := 0
	for {
		items, err := claim(ctx, db, s, 50)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			break
		}
		for _, x := range items {
			snap, e := c.GetAnnualFinancialSnapshot(ctx, x.symbol)
			st, cl, next := outcome(e, x)
			if err = save(ctx, db, x, s, snap, e, st, cl, next); err != nil {
				return err
			}
			processed++
		}
	}
	var coverage string
	err = db.QueryRowContext(ctx, `SELECT COALESCE(jsonb_object_agg(status,n),'{}')::text FROM (SELECT status,count(*) n FROM subscriber_global_annual_financial_tasks WHERE workflow_id=$1 GROUP BY status)x`, wid(s)).Scan(&coverage)
	if err != nil {
		return err
	}
	status := "succeeded"
	if strings.Contains(coverage, "queued") || strings.Contains(coverage, "retry") || strings.Contains(coverage, "failed") || strings.Contains(coverage, "blocked") || strings.Contains(coverage, "deferred") {
		status = "degraded"
	}
	_, err = db.ExecContext(ctx, `UPDATE subscriber_global_annual_financial_workflows SET status=$1,coverage=$2::jsonb,completed_at=now(),updated_at=now() WHERE workflow_id=$3`, status, coverage, wid(s))
	fmt.Printf("processed=%d fmp_calls=%d coverage=%s\n", processed, c.Calls(), coverage)
	return err
}
func session(v string) (time.Time, error) {
	if v != "" {
		return time.Parse("2006-01-02", v)
	}
	d := time.Now().UTC().Truncate(24 * time.Hour)
	for d.Weekday() == 0 || d.Weekday() == 6 {
		d = d.AddDate(0, 0, -1)
	}
	return d, nil
}
func wid(s time.Time) string { return "subglobalannualworkflow-" + s.Format("20060102") }
func seed(c context.Context, d *sql.DB, s time.Time, n int) error {
	w := wid(s)
	_, e := d.ExecContext(c, `INSERT INTO subscriber_global_annual_financial_workflows(workflow_id,session_date,status,started_at)VALUES($1,$2,'running',now()) ON CONFLICT(session_date) DO UPDATE SET status=CASE WHEN subscriber_global_annual_financial_workflows.status='succeeded' THEN 'succeeded' ELSE 'running' END`, w, s)
	if e != nil {
		return e
	}
	_, e = d.ExecContext(c, `INSERT INTO subscriber_global_annual_financial_tasks(task_id,workflow_id,global_asset_id,symbol,status) SELECT 'subglobalannualtask-'||substr(md5(global_asset_id||$1),1,24),$2,global_asset_id,canonical_symbol,'queued' FROM subscriber_global_warm_eod_assets ORDER BY priority LIMIT $3 ON CONFLICT(workflow_id,global_asset_id) DO NOTHING`, s.Format("2006-01-02"), w, n)
	return e
}
func claim(c context.Context, d *sql.DB, s time.Time, n int) ([]task, error) {
	r, e := d.QueryContext(c, `WITH x AS(SELECT task_id FROM subscriber_global_annual_financial_tasks WHERE workflow_id=$1 AND((status IN('queued','retry_scheduled')AND next_attempt_at<=now())OR(status='running'AND lease_expires_at<=now())OR(status='succeeded' AND NOT (result ? 'periods')))ORDER BY next_attempt_at FOR UPDATE SKIP LOCKED LIMIT $2)UPDATE subscriber_global_annual_financial_tasks t SET status='running',attempt_count=t.attempt_count+1,lease_expires_at=now()+interval '20 minutes',updated_at=now() FROM x WHERE t.task_id=x.task_id RETURNING t.task_id,t.global_asset_id,t.symbol,t.attempt_count,t.max_attempts`, wid(s), n)
	if e != nil {
		return nil, e
	}
	defer r.Close()
	out := []task{}
	for r.Next() {
		var x task
		if e = r.Scan(&x.id, &x.asset, &x.symbol, &x.attempt, &x.max); e != nil {
			return nil, e
		}
		out = append(out, x)
	}
	return out, r.Err()
}
func outcome(e error, x task) (string, string, time.Time) {
	if e == nil {
		return "succeeded", "", time.Time{}
	}
	var h *fmp.HTTPError
	if errors.As(e, &h) {
		if h.StatusCode() == 401 || h.StatusCode() == 403 {
			return "blocked_entitlement", "provider_entitlement", time.Time{}
		}
		if h.StatusCode() == 402 {
			return "deferred_quota", "provider_quota", time.Time{}
		}
		if (h.StatusCode() == 429 || h.StatusCode() >= 500) && x.attempt < x.max {
			return "retry_scheduled", "provider_transient", time.Now().UTC().Add(time.Minute)
		}
	}
	var networkErr net.Error
	if (errors.Is(e, context.DeadlineExceeded) || errors.As(e, &networkErr)) && x.attempt < x.max {
		return "retry_scheduled", "provider_transient", time.Now().UTC().Add(time.Minute)
	}
	if x.attempt >= x.max {
		return "failed_terminal", "retry_exhausted", time.Time{}
	}
	return "skipped_no_data", "provider_no_data", time.Time{}
}
func save(c context.Context, d *sql.DB, x task, s time.Time, snap fmp.AnnualFinancialSnapshot, e error, st, cl string, next time.Time) error {
	p, q, source, at := payload(x.symbol, snap, e)
	fp := hash(string(p))
	identitySeed := strings.Join([]string{x.asset, s.Format("2006-01-02"), algo, version, fp}, "\x1f")
	run := "subglobalannual-" + hash(identitySeed)[:24]
	tx, er := d.BeginTx(c, nil)
	if er != nil {
		return er
	}
	defer tx.Rollback()
	prov, _ := json.Marshal(map[string]any{"provider": "fmp", "task_id": x.id, "attempt": x.attempt})
	_, er = tx.ExecContext(c, `INSERT INTO subscriber_global_marketops_evidence_runs(evidence_run_id,evidence_kind,algorithm_id,algorithm_version,execution_mode,source_scope,session_start_date,session_end_date,input_manifest_fingerprint,validation_contract_ref,immutable_baseline_ref,provenance,recorded_by,recorded_at)VALUES($1,'fundamental_annual',$2,$3,'provider_capture','global_provider_capture',$4,$4,$5,$6,$7,$8::jsonb,$9,now())ON CONFLICT DO NOTHING`, run, algo, version, s, "sha256:"+fp, contract, baseline, string(prov), worker)
	if er != nil {
		return er
	}
	_, er = tx.ExecContext(c, `INSERT INTO subscriber_global_marketops_evidence_records(global_evidence_id,evidence_run_id,global_asset_id,session_date,evidence_kind,algorithm_id,algorithm_version,quality_state,source_system,source_event_id,source_run_id,evidence_fingerprint,validation_contract_ref,immutable_baseline_ref,payload,provenance,observed_at)VALUES($1,$2,$3,$4,'fundamental_annual',$5,$6,$7,'fmp',$8,$2,$9,$10,$11,$12::jsonb,$13::jsonb,$14)ON CONFLICT DO NOTHING`, `subglobalannualrec-`+hash(identitySeed)[:24], run, x.asset, s, algo, version, q, source, fp, contract, baseline, string(p), string(prov), at)
	if er != nil {
		return er
	}
	msg := ""
	if e != nil {
		msg = e.Error()
	}
	var done any = time.Now().UTC()
	if st == "retry_scheduled" {
		done = nil
	}
	_, er = tx.ExecContext(c, `UPDATE subscriber_global_annual_financial_tasks SET status=$1,failure_class=$2,error_message=$3,next_attempt_at=COALESCE($4,now()),lease_expires_at=NULL,result=$5::jsonb,completed_at=$6,updated_at=now() WHERE task_id=$7`, st, cl, msg, nullable(next), string(p), done, x.id)
	if er != nil {
		return er
	}
	return tx.Commit()
}
func payload(sym string, s fmp.AnnualFinancialSnapshot, e error) ([]byte, string, string, time.Time) {
	if e != nil {
		p, _ := json.Marshal(map[string]any{"symbol": sym, "capture_error": e.Error(), "annual_periods": 0})
		return p, "invalid", "fmp:annual:error:" + sym, time.Now().UTC()
	}
	p, _ := json.Marshal(map[string]any{"symbol": sym, "periods": s.Periods, "ratio_references_by_period": s.RatioReferences, "key_metric_references_by_period": s.KeyMetricReferences, "provider_request_ids": s.ProviderRequestIDs, "retrieved_at": s.RetrievedAt.UTC().Format(time.RFC3339Nano)})
	return p, "usable", "fmp:annual:" + sym + ":" + s.Periods[0].PeriodEnd.Format("2006-01-02"), s.RetrievedAt
}
func nullable(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}
func hash(v string) string { z := sha256.Sum256([]byte(v)); return hex.EncodeToString(z[:]) }

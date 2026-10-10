// subscriber-global-eod-canary-parity compares the completed global canary
// baseline with the current tenant-local normalized ledger. It makes no
// provider request and records missing comparison data as an incomplete result.
package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "subscriber global EOD canary parity failed:", err)
		os.Exit(1)
	}
}
func run(args []string) error {
	f := flag.NewFlagSet("subscriber-global-eod-canary-parity", flag.ContinueOnError)
	globalDSN := f.String("database-url", os.Getenv("SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL"), "dedicated global-EOD database URL")
	temporalDSN := f.String("temporal-database-url", os.Getenv("SIGNALOPS_TEMPORAL_DATABASE_URL"), "read-only normalized-ledger database URL")
	authID := f.String("authorization-id", "", "required live authorization id")
	execute := f.Bool("execute", false, "persist parity report")
	if err := f.Parse(args); err != nil {
		return err
	}
	if !*execute || strings.TrimSpace(*globalDSN) == "" || strings.TrimSpace(*temporalDSN) == "" || strings.TrimSpace(*authID) == "" {
		return fmt.Errorf("pass --execute and provide global, temporal, and authorization configuration")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	global, err := sql.Open("pgx", *globalDSN)
	if err != nil {
		return err
	}
	defer global.Close()
	temporal, err := sql.Open("pgx", *temporalDSN)
	if err != nil {
		return err
	}
	defer temporal.Close()
	rows, err := global.QueryContext(ctx, `SELECT result.live_run_id,result.global_asset_id,result.symbol,result.session_date,result.normalized_payload::text,result.normalized_fingerprint FROM subscriber_global_eod_canary_baseline_results result JOIN subscriber_global_eod_canary_live_runs run ON run.live_run_id=result.live_run_id WHERE run.authorization_id=$1 ORDER BY result.symbol`, *authID)
	if err != nil {
		return err
	}
	defer rows.Close()
	type result struct {
		run, asset, symbol   string
		session              time.Time
		payload, fingerprint string
	}
	values := []result{}
	for rows.Next() {
		var v result
		if err := rows.Scan(&v.run, &v.asset, &v.symbol, &v.session, &v.payload, &v.fingerprint); err != nil {
			return err
		}
		values = append(values, v)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(values) != 2 {
		return fmt.Errorf("expected exactly two completed global baseline results, got %d", len(values))
	}
	failed := false
	for _, v := range values {
		var localEvent, localPayload string
		err := temporal.QueryRowContext(ctx, `SELECT event_id,normalized_payload::text FROM normalized_event_ledger WHERE tenant_id='tenant-local' AND source_id='src-massive' AND dataset='equity_eod_prices' AND normalized_payload->>'symbol'=$1 AND normalized_payload->>'observation_date'=$2 ORDER BY processing_time DESC LIMIT 1`, v.symbol, v.session.Format("2006-01-02")).Scan(&localEvent, &localPayload)
		status, reason, compFingerprint := "matched", "", ""
		if err == sql.ErrNoRows {
			status, reason = "missing", "tenant-local normalized EOD event is absent"
			failed = true
		} else if err != nil {
			return fmt.Errorf("read %s tenant-local comparison: %w", v.symbol, err)
		} else {
			globalCanonical, err := canonical(v.payload)
			if err != nil {
				return err
			}
			localCanonical, err := canonical(localPayload)
			if err != nil {
				return err
			}
			compFingerprint = fingerprint(localCanonical)
			if string(globalCanonical) != string(localCanonical) {
				status, reason = "mismatched", "canonical OHLCV/provider payload differs"
				failed = true
			}
		}
		prov, _ := json.Marshal(map[string]any{"comparison_contract": "tenant-local/src-massive/equity_eod_prices/same-symbol-session", "algorithm_version": "subscriber-global-eod-baseline-v1"})
		_, err = global.ExecContext(ctx, `INSERT INTO subscriber_global_eod_canary_parity_reports (parity_report_id,live_run_id,global_asset_id,comparison_tenant_id,comparison_event_id,global_fingerprint,comparison_fingerprint,parity_status,mismatch_reason,provenance,compared_at) VALUES ($1,$2,$3,'tenant-local',$4,$5,$6,$7,$8,$9::jsonb,$10)`, newID("subeodparity"), v.run, v.asset, localEvent, v.fingerprint, compFingerprint, status, reason, string(prov), time.Now().UTC())
		if err != nil {
			return err
		}
		fmt.Printf("%s %s\n", v.symbol, status)
	}
	if failed {
		return fmt.Errorf("parity is incomplete or mismatched")
	}
	return nil
}
func canonical(raw string) ([]byte, error) {
	var v map[string]any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return nil, err
	}
	out := map[string]any{}
	for _, k := range []string{"provider", "dataset", "provider_event_id", "symbol", "observation_date", "open", "high", "low", "close", "volume", "vwap"} {
		out[k] = v[k]
	}
	return json.Marshal(out)
}
func fingerprint(v []byte) string { s := sha256.Sum256(v); return "sha256:" + hex.EncodeToString(s[:]) }
func newID(prefix string) string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}

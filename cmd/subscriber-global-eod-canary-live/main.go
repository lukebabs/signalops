// subscriber-global-eod-canary-live performs one explicitly authorized, two-
// symbol Massive EOD canary. It does not schedule future work or accept a
// browser request. Provider call intent is written before each call.
package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lukebabs/signalops/internal/adapters/marketdata/massive"
)

const workerIdentity = "subscriber-global-eod-reconciler"

type member struct {
	id, symbol string
	ordinal    int
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "subscriber global EOD live canary failed:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("subscriber-global-eod-canary-live", flag.ContinueOnError)
	dsn := flags.String("database-url", os.Getenv("SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL"), "dedicated global-EOD database URL")
	authID := flags.String("authorization-id", "", "required live authorization id")
	correlation := flags.String("correlation-id", "", "required immutable trace correlation id")
	actor := flags.String("actor", workerIdentity, "must be subscriber-global-eod-reconciler")
	execute := flags.Bool("execute", false, "make the two authorized provider requests")
	ack := flags.Bool("acknowledge-exactly-two-provider-requests", false, "confirm exactly two no-retry provider calls")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !*execute || !*ack {
		return errors.New("refusing to execute: pass --execute and --acknowledge-exactly-two-provider-requests")
	}
	if strings.TrimSpace(*dsn) == "" || strings.TrimSpace(*authID) == "" || strings.TrimSpace(*correlation) == "" || strings.TrimSpace(*actor) != workerIdentity {
		return errors.New("dedicated database URL, authorization id, correlation id, and controlled worker identity are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	db, err := sql.Open("pgx", *dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return err
	}
	var session time.Time
	var budget int
	var state string
	err = db.QueryRowContext(ctx, `SELECT authorization.session_date,authorization.provider_request_budget,authorization.authorization_state
FROM subscriber_global_eod_canary_live_authorizations authorization
JOIN subscriber_global_eod_canary_execution_plans plan ON plan.execution_plan_id=authorization.execution_plan_id
WHERE authorization.authorization_id=$1 AND authorization.authorized_worker_identity=$2 AND authorization.authorized_provider='massive'`, strings.TrimSpace(*authID), workerIdentity).Scan(&session, &budget, &state)
	if err != nil {
		return fmt.Errorf("load live authorization: %w", err)
	}
	if state != "authorized" || budget != 2 {
		return errors.New("live authorization is not an exact two-request authorization")
	}
	rows, err := db.QueryContext(ctx, `SELECT global_asset_id,expected_symbol,request_ordinal FROM subscriber_global_eod_canary_live_authorization_members WHERE authorization_id=$1 ORDER BY request_ordinal`, strings.TrimSpace(*authID))
	if err != nil {
		return err
	}
	members := []member{}
	for rows.Next() {
		var m member
		if err := rows.Scan(&m.id, &m.symbol, &m.ordinal); err != nil {
			rows.Close()
			return err
		}
		members = append(members, m)
	}
	rows.Close()
	if len(members) != 2 || members[0].ordinal != 1 || members[1].ordinal != 2 || members[0].symbol == members[1].symbol {
		return errors.New("authorization must contain exactly two unique frozen request slots")
	}
	liveRunID := newID("subeodlive")
	now := time.Now().UTC()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO subscriber_global_eod_canary_live_runs (live_run_id,authorization_id,worker_identity,provider_request_budget,session_date,correlation_id,started_at) VALUES ($1,$2,$3,2,$4,$5,$6)`, liveRunID, *authID, workerIdentity, session, *correlation, now); err != nil {
		return fmt.Errorf("insert live run: %w", err)
	}
	for _, m := range members {
		if _, err = tx.ExecContext(ctx, `INSERT INTO subscriber_global_eod_canary_live_run_members (live_run_id,global_asset_id,request_ordinal,expected_symbol) VALUES ($1,$2,$3,$4)`, liveRunID, m.id, m.ordinal, m.symbol); err != nil {
			return fmt.Errorf("insert live member: %w", err)
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO subscriber_global_eod_canary_live_evidence_events (evidence_event_id,live_run_id,global_asset_id,evidence_kind,event_ordinal,payload,provenance,recorded_at) VALUES ($1,$2,$3,'provider_request_started',1,$4::jsonb,$5::jsonb,$6)`, newID("subeodliveev"), liveRunID, m.id, `{"max_retries":0,"request_budget":2}`, fmt.Sprintf(`{"authorization_id":%q,"request_ordinal":%d,"correlation_id":%q}`, *authID, m.ordinal, *correlation), now); err != nil {
			return fmt.Errorf("reserve provider request: %w", err)
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	client, err := massive.NewClient(massive.LoadClientConfigFromEnv())
	if err != nil {
		return err
	}
	for _, m := range members {
		record, callErr := client.GetEquityDailyBar(ctx, m.symbol, session)
		occurred := time.Now().UTC()
		if callErr != nil {
			_, _ = db.ExecContext(ctx, `INSERT INTO subscriber_global_eod_canary_live_evidence_events (evidence_event_id,live_run_id,global_asset_id,evidence_kind,event_ordinal,payload,provenance,recorded_at) VALUES ($1,$2,$3,'provider_request_failed',1,$4::jsonb,$5::jsonb,$6)`, newID("subeodliveev"), liveRunID, m.id, fmt.Sprintf(`{"error":%q}`, callErr.Error()), fmt.Sprintf(`{"authorization_id":%q}`, *authID), occurred)
			return fmt.Errorf("provider request for %s: %w", m.symbol, callErr)
		}
		if strings.ToUpper(strings.TrimSpace(record.Symbol)) != m.symbol || !sameDay(record.ObservationDate, session) {
			return fmt.Errorf("provider response did not match frozen session/symbol for %s", m.symbol)
		}
		payload := canonicalPayload(record)
		encoded, _ := json.Marshal(payload)
		fingerprint := fingerprint(encoded)
		provenance, _ := json.Marshal(map[string]any{"authorization_id": *authID, "correlation_id": *correlation, "provider": "massive", "provider_raw": record.Raw, "request_ordinal": m.ordinal})
		tx, err = db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO subscriber_global_eod_canary_live_evidence_events (evidence_event_id,live_run_id,global_asset_id,evidence_kind,event_ordinal,payload,provenance,recorded_at) VALUES ($1,$2,$3,'provider_response_received',1,$4::jsonb,$5::jsonb,$6),($7,$2,$3,'normalization_completed',1,$8::jsonb,$5::jsonb,$6)`, newID("subeodliveev"), liveRunID, m.id, string(encoded), string(provenance), occurred, newID("subeodliveev"), fmt.Sprintf(`{"normalized_fingerprint":%q,"quality_state":"usable","algorithm_version":"subscriber-global-eod-baseline-v1"}`, fingerprint))
		if err == nil {
			_, err = tx.ExecContext(ctx, `INSERT INTO subscriber_global_eod_canary_baseline_results (live_run_id,global_asset_id,session_date,symbol,provider_event_id,normalized_payload,normalized_fingerprint,algorithm_version,quality_state,provenance,calculated_at) VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7,'subscriber-global-eod-baseline-v1','usable',$8::jsonb,$9)`, liveRunID, m.id, session, m.symbol, record.ProviderEventID, string(encoded), fingerprint, string(provenance), occurred)
		}
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("record %s baseline: %w", m.symbol, err)
		}
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"live_run_id": liveRunID, "authorization_id": *authID, "session_date": session.Format("2006-01-02"), "provider_requests": 2, "provider_retries": 0, "symbols": []string{members[0].symbol, members[1].symbol}, "parity": "pending"})
}
func canonicalPayload(r massive.EquityEODPriceRecord) map[string]any {
	return map[string]any{"provider": "massive", "dataset": "equity_eod_prices", "provider_event_id": strings.TrimSpace(r.ProviderEventID), "symbol": strings.ToUpper(strings.TrimSpace(r.Symbol)), "observation_date": r.ObservationDate.UTC().Format("2006-01-02"), "open": r.Open, "high": r.High, "low": r.Low, "close": r.Close, "volume": r.Volume, "vwap": r.VWAP}
}
func fingerprint(v []byte) string {
	sum := sha256.Sum256(v)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func sameDay(a, b time.Time) bool {
	return a.UTC().Format("2006-01-02") == b.UTC().Format("2006-01-02")
}
func newID(prefix string) string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}

// subscriber-global-eod-history-materializer imports only canonical EOD bars
// already retained in the dedicated temporal ledger. It never calls a market
// data provider and cannot copy tenant-owned results to another tenant.
package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	historyWorkerIdentity = "subscriber-global-eod-history-materializer"
	historyAlgorithmID    = "marketops.equity_eod.initial_capture"
	historyAlgorithmV1    = "v1"
	historyContract       = "subscriber-global-eod-history-initial-capture/v1"
)

type warmAsset struct{ id, symbol string }
type sourceBar struct {
	eventID, symbol, payload     string
	session, observed, processed time.Time
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "subscriber global EOD history materialization failed:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	f := flag.NewFlagSet("subscriber-global-eod-history-materializer", flag.ContinueOnError)
	primaryURL := f.String("database-url", os.Getenv("SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL"), "dedicated primary global-worker database URL")
	temporalURL := f.String("temporal-database-url", firstEnv("SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_TEMPORAL_DATABASE_URL", "SIGNALOPS_TEMPORAL_DATABASE_URL"), "dedicated temporal EOD source database URL")
	start := f.String("start-date", "", "inclusive session date YYYY-MM-DD; empty uses retained history")
	end := f.String("end-date", "", "inclusive session date YYYY-MM-DD; empty uses retained history")
	limit := f.Int("limit", 50000, "maximum initial-capture EOD rows to import (1-50000)")
	correlation := f.String("correlation-id", "", "operator correlation id")
	actor := f.String("actor", historyWorkerIdentity, "must be the controlled history materializer identity")
	dryRun := f.Bool("dry-run", false, "validate source coverage without appending global evidence")
	execute := f.Bool("execute", false, "append global evidence")
	if err := f.Parse(args); err != nil {
		return err
	}
	if (*execute && *dryRun) || (!*execute && !*dryRun) || strings.TrimSpace(*primaryURL) == "" || strings.TrimSpace(*temporalURL) == "" || strings.TrimSpace(*actor) != historyWorkerIdentity {
		return fmt.Errorf("pass exactly one of --dry-run or --execute, dedicated primary and temporal URLs, and actor %q", historyWorkerIdentity)
	}
	if *limit < 1 || *limit > 50000 {
		return fmt.Errorf("limit must be between 1 and 50000")
	}
	startDate, err := parseDate(*start)
	if err != nil {
		return err
	}
	endDate, err := parseDate(*end)
	if err != nil {
		return err
	}
	if !startDate.IsZero() && !endDate.IsZero() && startDate.After(endDate) {
		return fmt.Errorf("start-date must not be after end-date")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	primary, err := sql.Open("pgx", strings.TrimSpace(*primaryURL))
	if err != nil {
		return err
	}
	defer primary.Close()
	temporal, err := sql.Open("pgx", strings.TrimSpace(*temporalURL))
	if err != nil {
		return err
	}
	defer temporal.Close()
	primary.SetMaxOpenConns(1)
	temporal.SetMaxOpenConns(1)
	if err := primary.PingContext(ctx); err != nil {
		return fmt.Errorf("connect primary: %w", err)
	}
	if err := temporal.PingContext(ctx); err != nil {
		return fmt.Errorf("connect temporal: %w", err)
	}
	if _, err := primary.ExecContext(ctx, "SET ROLE signalops_subscriber_global_eod"); err != nil {
		return fmt.Errorf("assume controlled global-EOD role: %w", err)
	}
	defer primary.ExecContext(context.Background(), "RESET ROLE")
	assets, err := loadWarmAssets(ctx, primary)
	if err != nil {
		return err
	}
	if len(assets) == 0 {
		return fmt.Errorf("no enabled warm EOD assets are available")
	}
	bars, err := loadInitialBars(ctx, temporal, assets, startDate, endDate, *limit)
	if err != nil {
		return err
	}
	if len(bars) == 0 {
		return fmt.Errorf("no retained EOD bars matched the enabled warm cohort")
	}
	if *dryRun {
		symbols := map[string]struct{}{}
		for _, bar := range bars {
			symbols[bar.symbol] = struct{}{}
		}
		fmt.Printf("dry_run=true warm_assets=%d source_symbols=%d selected=%d first_session=%s last_session=%s policy=initial_capture\n", len(assets), len(symbols), len(bars), bars[0].session.Format("2006-01-02"), bars[len(bars)-1].session.Format("2006-01-02"))
		return nil
	}
	inserted, err := appendEvidence(ctx, primary, bars, *correlation)
	if err != nil {
		return err
	}
	symbols := map[string]struct{}{}
	for _, bar := range bars {
		symbols[bar.symbol] = struct{}{}
	}
	fmt.Printf("warm_assets=%d source_symbols=%d selected=%d inserted=%d first_session=%s last_session=%s policy=initial_capture\n", len(assets), len(symbols), len(bars), inserted, bars[0].session.Format("2006-01-02"), bars[len(bars)-1].session.Format("2006-01-02"))
	return nil
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}
func parseDate(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, nil
	}
	d, e := time.Parse("2006-01-02", strings.TrimSpace(value))
	if e != nil {
		return time.Time{}, fmt.Errorf("invalid date %q", value)
	}
	return d.UTC(), nil
}
func loadWarmAssets(ctx context.Context, db *sql.DB) ([]warmAsset, error) {
	rows, err := db.QueryContext(ctx, `SELECT global_asset_id,canonical_symbol FROM subscriber_global_warm_eod_assets ORDER BY priority,global_asset_id`)
	if err != nil {
		return nil, fmt.Errorf("load warm cohort: %w", err)
	}
	defer rows.Close()
	out := []warmAsset{}
	for rows.Next() {
		var x warmAsset
		if err := rows.Scan(&x.id, &x.symbol); err != nil {
			return nil, err
		}
		x.symbol = strings.ToUpper(strings.TrimSpace(x.symbol))
		if x.symbol != "" {
			out = append(out, x)
		}
	}
	return out, rows.Err()
}
func loadInitialBars(ctx context.Context, db *sql.DB, assets []warmAsset, start, end time.Time, limit int) ([]sourceBar, error) {
	bySymbol := map[string]struct{}{}
	symbols := []string{}
	for _, a := range assets {
		if _, ok := bySymbol[a.symbol]; !ok {
			bySymbol[a.symbol] = struct{}{}
			symbols = append(symbols, a.symbol)
		}
	}
	sort.Strings(symbols)
	rows, err := db.QueryContext(ctx, `SELECT event_id,upper(normalized_payload->>'symbol'),normalized_payload::text,COALESCE((normalized_payload->>'observation_date')::date,observation_time::date),observation_time,processing_time
FROM (
  SELECT DISTINCT ON (upper(normalized_payload->>'symbol'),COALESCE((normalized_payload->>'observation_date')::date,observation_time::date))
    event_id,normalized_payload,observation_time,processing_time
  FROM normalized_event_ledger
  WHERE tenant_id='tenant-local' AND source_id='src-massive' AND dataset='equity_eod_prices'
    AND upper(normalized_payload->>'symbol') = ANY($1)
    AND ($2::date IS NULL OR COALESCE((normalized_payload->>'observation_date')::date,observation_time::date) >= $2::date)
    AND ($3::date IS NULL OR COALESCE((normalized_payload->>'observation_date')::date,observation_time::date) <= $3::date)
  ORDER BY upper(normalized_payload->>'symbol'),COALESCE((normalized_payload->>'observation_date')::date,observation_time::date),processing_time,event_id
) source ORDER BY 4,2,1 LIMIT $4`, symbols, nullDate(start), nullDate(end), limit)
	if err != nil {
		return nil, fmt.Errorf("read retained initial EOD captures: %w", err)
	}
	defer rows.Close()
	out := []sourceBar{}
	for rows.Next() {
		var x sourceBar
		if err := rows.Scan(&x.eventID, &x.symbol, &x.payload, &x.session, &x.observed, &x.processed); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func appendEvidence(ctx context.Context, db *sql.DB, bars []sourceBar, correlation string) (int, error) {
	byAsset := map[string]string{}
	rows, err := db.QueryContext(ctx, `SELECT global_asset_id,upper(canonical_symbol) FROM subscriber_global_warm_eod_assets`)
	if err != nil {
		return 0, err
	}
	for rows.Next() {
		var id, symbol string
		if err := rows.Scan(&id, &symbol); err != nil {
			rows.Close()
			return 0, err
		}
		byAsset[symbol] = id
	}
	rows.Close()
	fingerprints := make([]string, 0, len(bars))
	for _, bar := range bars {
		fingerprints = append(fingerprints, fingerprint(bar.eventID+"\x1f"+bar.payload))
	}
	sort.Strings(fingerprints)
	runID := "subglobaleodhist-" + fingerprint(strings.Join(fingerprints, "\x1f"))[:24]
	first, last := bars[0].session, bars[0].session
	for _, bar := range bars {
		if bar.session.Before(first) {
			first = bar.session
		}
		if bar.session.After(last) {
			last = bar.session
		}
	}
	runProvenance, _ := json.Marshal(map[string]any{"source_scope": "legacy_temporal_initial_capture", "source_tenant_id": "tenant-local", "source_dataset": "equity_eod_prices", "selection_policy": "initial_capture", "warm_cohort_only": true})
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO subscriber_global_marketops_evidence_runs (evidence_run_id,evidence_kind,algorithm_id,algorithm_version,execution_mode,source_scope,session_start_date,session_end_date,input_manifest_fingerprint,validation_contract_ref,immutable_baseline_ref,provenance,recorded_by,correlation_id,recorded_at) VALUES ($1,'eod_bar',$2,$3,'legacy_materialized','legacy_materialization',$4,$5,$6,$7,$8,$9::jsonb,'subscriber-global-eod-reconciler',$10,now()) ON CONFLICT (evidence_run_id) DO NOTHING`, runID, historyAlgorithmID, historyAlgorithmV1, first, last, fingerprint(strings.Join(fingerprints, "\x1f")), historyContract, "initial-capture:tenant-local", string(runProvenance), strings.TrimSpace(correlation))
	if err != nil {
		return 0, fmt.Errorf("record EOD history run: %w", err)
	}
	inserted := 0
	for _, bar := range bars {
		id := byAsset[bar.symbol]
		if id == "" {
			continue
		}
		fp := fingerprint(bar.eventID + "\x1f" + bar.payload)
		prov, _ := json.Marshal(map[string]any{"legacy_event_id": bar.eventID, "legacy_processing_time": bar.processed.UTC().Format(time.RFC3339Nano), "selection_policy": "initial_capture"})
		result, err := tx.ExecContext(ctx, `INSERT INTO subscriber_global_marketops_evidence_records (global_evidence_id,evidence_run_id,global_asset_id,session_date,evidence_kind,algorithm_id,algorithm_version,quality_state,source_system,source_event_id,source_run_id,evidence_fingerprint,validation_contract_ref,immutable_baseline_ref,payload,provenance,observed_at) VALUES ($1,$2,$3,$4,'eod_bar',$5,$6,'usable','massive',$7,'',$8,$9,$10,$11::jsonb,$12::jsonb,$13) ON CONFLICT (global_asset_id,session_date,evidence_kind,algorithm_id,algorithm_version,evidence_fingerprint) DO NOTHING`, "subglobaleod-"+fingerprint(id + "\x1f" + fp)[:24], runID, id, bar.session, historyAlgorithmID, historyAlgorithmV1, bar.eventID, fp, historyContract, "initial-capture:tenant-local", bar.payload, string(prov), bar.observed.UTC())
		if err != nil {
			return 0, fmt.Errorf("append EOD bar %s: %w", bar.eventID, err)
		}
		count, _ := result.RowsAffected()
		inserted += int(count)
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return inserted, nil
}
func nullDate(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}
func fingerprint(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

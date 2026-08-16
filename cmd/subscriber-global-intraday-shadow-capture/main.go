// subscriber-global-intraday-shadow-capture captures current quotes for the
// aggregate hot selector. It writes only append-only global evidence and
// shadow parity records; tenant-local current-state tables remain authoritative.
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
	"os"
	"sort"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lukebabs/signalops/internal/adapters/marketdata/massive"
)

const (
	workerIdentity   = "subscriber-global-eod-reconciler"
	selectorVersion  = "subscriber-watchlist-context-v1"
	algorithmID      = "marketops.intraday_conditions"
	algorithmVersion = "v2-global-shadow"
	contractRef      = "subscriber-global-intraday-shadow-capture/v1"
	baselineRef      = "legacy-hot-current-state/v1"
)

type selectedAsset struct{ id, symbol string }
type legacySnapshot struct {
	id   string
	asOf time.Time
}
type captureEntry struct {
	asset      selectedAsset
	evidenceID string
	asOf       time.Time
	legacy     legacySnapshot
	status     string
	failure    string
}

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "subscriber global intraday shadow capture failed:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("subscriber-global-intraday-shadow-capture", flag.ContinueOnError)
	databaseURL := flags.String("database-url", os.Getenv("SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL"), "dedicated MarketOps primary URL")
	maxAssets := flags.Int("max-assets", 1000, "maximum aggregate hot assets (1-1000)")
	dryRun := flags.Bool("dry-run", false, "inspect selector and session only; make no provider request or write")
	execute := flags.Bool("execute", false, "append shadow evidence and parity records")
	correlationID := flags.String("correlation-id", "", "optional operator correlation ID")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *dryRun == *execute || strings.TrimSpace(*databaseURL) == "" || *maxAssets < 1 || *maxAssets > 1000 {
		return errors.New("pass exactly one of --dry-run or --execute, a dedicated database URL, and 1-1000 max assets")
	}
	db, err := sql.Open("pgx", strings.TrimSpace(*databaseURL))
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("connect to dedicated MarketOps primary: %w", err)
	}
	if _, err := db.ExecContext(ctx, "SET ROLE signalops_subscriber_global_eod"); err != nil {
		return fmt.Errorf("assume controlled global-EOD role: %w", err)
	}
	defer func() { _, _ = db.ExecContext(context.Background(), "RESET ROLE") }()
	if err := validateWorker(ctx, db); err != nil {
		return err
	}
	assets, err := selected(ctx, db, *maxAssets)
	if err != nil {
		return err
	}
	status, open := marketSession(time.Now().UTC())
	if *dryRun {
		fmt.Printf("dry_run=true selected_assets=%d market_open=%t market_session=%s\n", len(assets), open, status)
		return nil
	}
	if !open {
		fmt.Printf("execute=false reason=non_trading_session selected_assets=%d\n", len(assets))
		return nil
	}
	if len(assets) == 0 {
		fmt.Println("execute=false reason=no_explicit_watchlist_selection")
		return nil
	}
	legacy, err := latestLegacy(ctx, db)
	if err != nil {
		return err
	}
	client, err := massive.NewClient(massive.LoadClientConfigFromEnv())
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	entries := make([]captureEntry, 0, len(assets))
	for _, asset := range assets {
		entry := captureEntry{asset: asset, legacy: legacy[strings.ToUpper(asset.symbol)]}
		quote, quoteErr := client.GetEquityQuote(ctx, asset.symbol)
		if quoteErr != nil {
			entry.status, entry.failure = "provider_failure", classifyFailure(quoteErr)
			entries = append(entries, entry)
			continue
		}
		if quote.MarketStatus == "intraday" {
			quote.MarketStatus, quote.Stale = status, false
		}
		entry.asOf = now.Truncate(15 * time.Minute)
		if quote.MarketStatus == "end_of_day" && !quote.Timestamp.IsZero() {
			entry.asOf = quote.Timestamp.UTC()
		}
		payload, fp := evidencePayload(quote, entry.asOf)
		entry.evidenceID = "subglobalintraday-" + fingerprint(asset.id + "\x1f" + fp)[:24]
		entry.status = comparison(entry.asOf, entry.legacy)
		if err := appendEvidence(ctx, db, asset, entry, payload, fp, now, *correlationID); err != nil {
			return err
		}
		entries = append(entries, entry)
	}
	if err := recordCapture(ctx, db, entries, status, now, *correlationID); err != nil {
		return err
	}
	counts := summarize(entries)
	fmt.Printf("capture_run_id=%s selected=%d provider_success=%d provider_failures=%d legacy_overlap=%d freshness_matches=%d freshness_mismatches=%d result=%s\n", captureID(now, assets), len(entries), counts.success, counts.failure, counts.overlap, counts.match, counts.mismatch, counts.result)
	return nil
}

func validateWorker(ctx context.Context, db *sql.DB) error {
	var user string
	var superuser, createRole, bypass, member, selectorRead, legacyRead, evidenceWrite, runWrite, rawWatchlistRead bool
	err := db.QueryRowContext(ctx, `SELECT current_user,rolsuper,rolcreaterole,rolbypassrls,pg_has_role(current_user,'signalops_subscriber_global_eod','member'),has_table_privilege(current_user,'subscriber_global_hot_intraday_assets','SELECT'),has_table_privilege(current_user,'subscriber_global_marketops_legacy_parity_source_v3','SELECT'),has_table_privilege(current_user,'subscriber_global_marketops_evidence_runs','SELECT,INSERT'),has_table_privilege(current_user,'subscriber_global_intraday_shadow_capture_runs','SELECT,INSERT'),has_table_privilege(current_user,'subscriber_watchlist_memberships','SELECT') FROM pg_roles WHERE rolname=current_user`).Scan(&user, &superuser, &createRole, &bypass, &member, &selectorRead, &legacyRead, &evidenceWrite, &runWrite, &rawWatchlistRead)
	if err != nil {
		return fmt.Errorf("inspect shadow worker identity: %w", err)
	}
	if superuser || createRole || bypass || !member || !selectorRead || !legacyRead || !evidenceWrite || !runWrite || rawWatchlistRead {
		return fmt.Errorf("shadow worker must be the least-privileged global-EOD role with aggregate selector, legacy projection, and append-only evidence grants only (got %s)", user)
	}
	return nil
}
func selected(ctx context.Context, db *sql.DB, limit int) ([]selectedAsset, error) {
	rows, err := db.QueryContext(ctx, `SELECT global_asset_id,canonical_symbol FROM subscriber_global_hot_intraday_assets ORDER BY canonical_symbol,global_asset_id LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	assets := []selectedAsset{}
	for rows.Next() {
		var x selectedAsset
		if err := rows.Scan(&x.id, &x.symbol); err != nil {
			return nil, err
		}
		assets = append(assets, x)
	}
	return assets, rows.Err()
}
func latestLegacy(ctx context.Context, db *sql.DB) (map[string]legacySnapshot, error) {
	rows, err := db.QueryContext(ctx, `SELECT DISTINCT ON (upper(legacy_symbol)) upper(legacy_symbol),legacy_record_id,(legacy_payload->>'as_of_time')::timestamptz FROM subscriber_global_marketops_legacy_parity_source_v3 WHERE evidence_kind='intraday_snapshot' AND NULLIF(legacy_payload->>'as_of_time','') IS NOT NULL ORDER BY upper(legacy_symbol),(legacy_payload->>'as_of_time')::timestamptz DESC,legacy_record_id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]legacySnapshot{}
	for rows.Next() {
		var s string
		var v legacySnapshot
		if err := rows.Scan(&s, &v.id, &v.asOf); err != nil {
			return nil, err
		}
		out[s] = v
	}
	return out, rows.Err()
}
func appendEvidence(ctx context.Context, db *sql.DB, asset selectedAsset, entry captureEntry, payload []byte, fp string, now time.Time, correlation string) error {
	runID := "subglobalintradayrun-" + fingerprint(entry.asOf.Format(time.RFC3339) + "\x1f" + fp)[:24]
	p, _ := json.Marshal(map[string]any{"selector_version": selectorVersion, "execution_mode": "shadow_provider_capture", "correlation_id": correlation})
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO subscriber_global_marketops_evidence_runs(evidence_run_id,evidence_kind,algorithm_id,algorithm_version,execution_mode,source_scope,session_start_date,session_end_date,input_manifest_fingerprint,validation_contract_ref,immutable_baseline_ref,provenance,recorded_by,correlation_id,recorded_at) VALUES($1,'intraday_snapshot',$2,$3,'provider_capture','global_provider_capture',$4,$4,$5,$6,$7,$8::jsonb,$9,$10,$11) ON CONFLICT DO NOTHING`, runID, algorithmID, algorithmVersion, entry.asOf.UTC(), "sha256:"+fp, contractRef, baselineRef, string(p), workerIdentity, correlation, now)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO subscriber_global_marketops_evidence_records(global_evidence_id,evidence_run_id,global_asset_id,session_date,evidence_kind,algorithm_id,algorithm_version,quality_state,source_system,source_event_id,source_run_id,evidence_fingerprint,validation_contract_ref,immutable_baseline_ref,payload,provenance,observed_at) VALUES($1,$2,$3,$4,'intraday_snapshot',$5,$6,$7,'massive',$8,$2,$9,$10,$11,$12::jsonb,$13::jsonb,$14) ON CONFLICT DO NOTHING`, entry.evidenceID, runID, asset.id, entry.asOf.UTC(), algorithmID, algorithmVersion, quality(payload), "massive:quote:"+asset.symbol+":"+entry.asOf.Format(time.RFC3339), fp, contractRef, baselineRef, string(payload), string(p), entry.asOf.UTC())
	if err != nil {
		return err
	}
	return tx.Commit()
}
func recordCapture(ctx context.Context, db *sql.DB, entries []captureEntry, session string, now time.Time, correlation string) error {
	counts := summarize(entries)
	runID := captureID(now, assetsOf(entries))
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO subscriber_global_intraday_shadow_capture_runs(capture_run_id,selector_version,execution_mode,market_session,selected_count,provider_success_count,provider_failure_count,legacy_overlap_count,freshness_match_count,freshness_mismatch_count,result_status,validation_contract_ref,immutable_baseline_ref,correlation_id,recorded_by,recorded_at) VALUES($1,$2,'shadow_provider_capture',$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`, runID, selectorVersion, session, len(entries), counts.success, counts.failure, counts.overlap, counts.match, counts.mismatch, counts.result, contractRef, baselineRef, correlation, workerIdentity, now)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		_, err = tx.ExecContext(ctx, `INSERT INTO subscriber_global_intraday_shadow_capture_entries(capture_run_id,global_asset_id,canonical_symbol,central_evidence_id,central_as_of_time,legacy_snapshot_id,legacy_as_of_time,comparison_status,failure_class,recorded_at) VALUES($1,$2,$3,NULLIF($4,''),$5,$6,$7,$8,$9,$10)`, runID, entry.asset.id, entry.asset.symbol, entry.evidenceID, nullable(entry.asOf), entry.legacy.id, nullable(entry.legacy.asOf), entry.status, entry.failure, now)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

type summary struct {
	success, failure, overlap, match, mismatch int
	result                                     string
}

func summarize(entries []captureEntry) summary {
	out := summary{result: "complete"}
	for _, e := range entries {
		if e.status == "provider_failure" {
			out.failure++
			out.result = "degraded"
			continue
		}
		out.success++
		if e.legacy.id != "" {
			out.overlap++
		}
		if e.status == "freshness_match" {
			out.match++
		}
		if e.status == "freshness_mismatch" {
			out.mismatch++
			out.result = "degraded"
		}
		if e.status == "legacy_missing" {
			out.result = "degraded"
		}
	}
	return out
}
func assetsOf(entries []captureEntry) []selectedAsset {
	out := make([]selectedAsset, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.asset)
	}
	return out
}
func captureID(now time.Time, assets []selectedAsset) string {
	values := make([]string, 0, len(assets))
	for _, a := range assets {
		values = append(values, a.id)
	}
	sort.Strings(values)
	return "subglobalintradaycap-" + fingerprint(now.UTC().Format(time.RFC3339Nano) + "\x1f" + strings.Join(values, "\x1f"))[:24]
}
func comparison(asOf time.Time, legacy legacySnapshot) string {
	if legacy.id == "" {
		return "legacy_missing"
	}
	if asOf.UTC().Truncate(15 * time.Minute).Equal(legacy.asOf.UTC().Truncate(15 * time.Minute)) {
		return "freshness_match"
	}
	return "freshness_mismatch"
}
func classifyFailure(err error) string {
	var h *massive.HTTPError
	if errors.As(err, &h) {
		if h.StatusCode() == 401 || h.StatusCode() == 403 {
			return "provider_entitlement"
		}
		if h.StatusCode() == 429 {
			return "provider_rate_limited"
		}
		if h.StatusCode() >= 500 {
			return "provider_transient"
		}
	}
	return "provider_unavailable"
}
func nullable(v time.Time) any {
	if v.IsZero() {
		return nil
	}
	return v.UTC()
}
func fingerprint(v string) string { sum := sha256.Sum256([]byte(v)); return hex.EncodeToString(sum[:]) }
func evidencePayload(q massive.EquityQuote, asOf time.Time) ([]byte, string) {
	conditions := derive(q)
	source := map[string]any{"price": q.Price, "previous_close": q.PreviousClose, "change": q.Change, "change_percent": q.ChangePercent, "week52_low": q.Week52Low, "week52_high": q.Week52High, "quote_timestamp": q.Timestamp.UTC().Format(time.RFC3339Nano), "provider": "massive"}
	p, _ := json.Marshal(map[string]any{"snapshot_id": "", "universe_group": "subscriber_hot", "as_of_time": asOf.UTC().Format(time.RFC3339Nano), "market_status": q.MarketStatus, "stale": q.Stale, "conditions": conditions, "source_payload": source, "current_only_source": true})
	return p, fingerprint(string(p))
}
func quality(p []byte) string {
	var v struct {
		Stale      bool  `json:"stale"`
		Conditions []any `json:"conditions"`
	}
	_ = json.Unmarshal(p, &v)
	if v.Stale || len(v.Conditions) == 0 {
		return "partial"
	}
	return "usable"
}
func derive(q massive.EquityQuote) []map[string]any {
	out := []map[string]any{}
	if q.ChangePercent != nil {
		if *q.ChangePercent >= 1 {
			out = append(out, map[string]any{"key": "session_move_up", "tone": "positive", "score": *q.ChangePercent})
		}
		if *q.ChangePercent <= -1 {
			out = append(out, map[string]any{"key": "session_move_down", "tone": "negative", "score": -*q.ChangePercent})
		}
	}
	return out
}
func marketSession(now time.Time) (string, bool) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return "", false
	}
	t := now.In(loc)
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return "", false
	}
	m := t.Hour()*60 + t.Minute()
	if m >= 9*60+30 && m < 16*60 {
		return "regular", true
	}
	if m >= 16*60 && m <= 20*60 {
		return "extended", true
	}
	return "", false
}

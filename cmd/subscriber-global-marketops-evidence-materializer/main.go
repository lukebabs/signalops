// subscriber-global-marketops-evidence-materializer promotes only mapped,
// immutable parity-manifest entries into the global analytical evidence store.
// It never calls a provider, changes a tenant-local source row, or exposes a
// gateway projection. Type-specific reader releases remain separate gates.
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
	workerIdentity = "subscriber-global-eod-reconciler"
	contractRef    = "subscriber-global-legacy-materialization/v1"
)

var allowedKinds = map[string]struct{}{
	"feature_vector": {}, "market_state": {}, "valuation": {}, "eeom": {}, "signal_assertion": {}, "outcome": {}, "options_snapshot": {}, "risk_reward": {}, "intraday_snapshot": {},
}

type entry struct {
	kind, id, symbol, algorithmID, algorithmVersion, quality, payload, fingerprint, globalAssetID string
	session, created                                                                              time.Time
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "subscriber global evidence materialization failed:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("subscriber-global-marketops-evidence-materializer", flag.ContinueOnError)
	databaseURL := flags.String("database-url", os.Getenv("SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL"), "dedicated global-worker database URL")
	parityRunID := flags.String("parity-run-id", "", "immutable parity manifest run to materialize")
	evidenceKinds := flags.String("evidence-kinds", "feature_vector,market_state,valuation,eeom,signal_assertion,outcome,options_snapshot,risk_reward,intraday_snapshot", "comma-separated supported evidence kinds")
	algorithmID := flags.String("algorithm-id", "", "optional exact legacy algorithm identifier")
	limit := flags.Int("limit", 1000, "maximum entries to materialize (1-50000)")
	correlationID := flags.String("correlation-id", "", "optional operator correlation ID")
	actor := flags.String("actor", workerIdentity, "must be subscriber-global-eod-reconciler")
	execute := flags.Bool("execute", false, "perform append-only materialization")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !*execute || strings.TrimSpace(*databaseURL) == "" || strings.TrimSpace(*parityRunID) == "" || strings.TrimSpace(*actor) != workerIdentity {
		return fmt.Errorf("pass --execute, --parity-run-id, a dedicated database URL, and actor %q", workerIdentity)
	}
	if *limit < 1 || *limit > 50000 {
		return fmt.Errorf("limit must be between 1 and 50000")
	}
	kinds, err := parseKinds(*evidenceKinds)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	db, err := sql.Open("pgx", strings.TrimSpace(*databaseURL))
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("connect to global worker database: %w", err)
	}
	if _, err := db.ExecContext(ctx, "SET ROLE signalops_subscriber_global_eod"); err != nil {
		return fmt.Errorf("assume controlled global-EOD role: %w", err)
	}
	defer func() { _, _ = db.ExecContext(context.Background(), "RESET ROLE") }()
	if err := validateWorkload(ctx, db); err != nil {
		return err
	}
	entries, err := readEntries(ctx, db, strings.TrimSpace(*parityRunID), kinds, strings.TrimSpace(*algorithmID), *limit)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return fmt.Errorf("no mapped pending entries found for parity run %q", strings.TrimSpace(*parityRunID))
	}
	groups := map[string][]entry{}
	for _, item := range entries {
		groups[groupKey(item)] = append(groups[groupKey(item)], item)
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	inserted := 0
	for _, key := range keys {
		count, err := materializeGroup(ctx, db, strings.TrimSpace(*parityRunID), strings.TrimSpace(*correlationID), groups[key])
		if err != nil {
			return err
		}
		inserted += count
	}
	fmt.Printf("parity_run_id=%s selected=%d inserted=%d groups=%d\n", strings.TrimSpace(*parityRunID), len(entries), inserted, len(groups))
	return nil
}

func validateWorkload(ctx context.Context, db *sql.DB) error {
	var currentUser string
	var superuser, createRole, bypassRLS, member, sourceRead, manifestRead, evidenceWrite, rawRead, rawOptionsRead, rawRiskRewardRead, rawIntradayRead bool
	if err := db.QueryRowContext(ctx, `SELECT current_user,rolsuper,rolcreaterole,rolbypassrls,
  pg_has_role(current_user,'signalops_subscriber_global_eod','member'),
  has_table_privilege(current_user,'subscriber_global_marketops_legacy_parity_source_v3','SELECT'),
  has_table_privilege(current_user,'subscriber_global_marketops_legacy_parity_manifest_entries','SELECT'),
  has_table_privilege(current_user,'subscriber_global_marketops_evidence_runs','SELECT,INSERT'),
  has_table_privilege(current_user,'marketops_feature_observations','SELECT'),
  has_table_privilege(current_user,'marketops_options_distribution_daily','SELECT'),
  has_table_privilege(current_user,'marketops_risk_reward_snapshots','SELECT'),
  has_table_privilege(current_user,'marketops_intraday_condition_snapshots','SELECT')
FROM pg_roles WHERE rolname=current_user`).Scan(&currentUser, &superuser, &createRole, &bypassRLS, &member, &sourceRead, &manifestRead, &evidenceWrite, &rawRead, &rawOptionsRead, &rawRiskRewardRead, &rawIntradayRead); err != nil {
		return fmt.Errorf("inspect materializer workload identity: %w", err)
	}
	if superuser || createRole || bypassRLS || !member || !sourceRead || !manifestRead || !evidenceWrite || rawRead || rawOptionsRead || rawRiskRewardRead || rawIntradayRead {
		return fmt.Errorf("materializer workload must be a non-privileged global-EOD member with only parity-source, manifest, and evidence-write grants (got %s)", currentUser)
	}
	return nil
}

func readEntries(ctx context.Context, db *sql.DB, parityRunID string, kinds []string, algorithmID string, limit int) ([]entry, error) {
	quoted := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		quoted = append(quoted, "'"+kind+"'")
	}
	rows, err := db.QueryContext(ctx, `SELECT entry.evidence_kind,entry.legacy_record_id,entry.legacy_symbol,entry.legacy_session_date,
  entry.legacy_algorithm_id,entry.legacy_algorithm_version,entry.legacy_quality_state,
  source.legacy_payload::text,entry.legacy_fingerprint,entry.global_asset_id,source.legacy_created_at
FROM subscriber_global_marketops_legacy_parity_manifest_entries entry
JOIN subscriber_global_marketops_legacy_parity_source_v3 source
  ON source.evidence_kind=entry.evidence_kind AND source.legacy_record_id=entry.legacy_record_id
WHERE entry.parity_run_id=$1 AND entry.mapping_status='mapped'
  AND entry.global_materialization_status='pending_global_materialization'
  AND entry.evidence_kind IN (`+strings.Join(quoted, ",")+`)
  AND ($2='' OR entry.legacy_algorithm_id=$2)
ORDER BY entry.evidence_kind,entry.legacy_algorithm_id,entry.legacy_algorithm_version,entry.legacy_session_date,entry.legacy_record_id
LIMIT $3`, parityRunID, algorithmID, limit)
	if err != nil {
		return nil, fmt.Errorf("read parity-gated source entries: %w", err)
	}
	defer rows.Close()
	items := []entry{}
	for rows.Next() {
		var item entry
		if err := rows.Scan(&item.kind, &item.id, &item.symbol, &item.session, &item.algorithmID, &item.algorithmVersion, &item.quality, &item.payload, &item.fingerprint, &item.globalAssetID, &item.created); err != nil {
			return nil, fmt.Errorf("scan parity-gated source entry: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func materializeGroup(ctx context.Context, db *sql.DB, parityRunID, correlationID string, entries []entry) (int, error) {
	first, last := entries[0].session, entries[0].session
	parts := make([]string, 0, len(entries))
	for _, item := range entries {
		if item.session.Before(first) {
			first = item.session
		}
		if item.session.After(last) {
			last = item.session
		}
		parts = append(parts, item.fingerprint)
	}
	manifestFingerprint := fingerprint(strings.Join(parts, "\x1f"))
	runID := "subglobalevrun-" + fingerprint(parityRunID + "\x1f" + groupKey(entries[0]) + "\x1f" + manifestFingerprint)[:24]
	now := time.Now().UTC()
	provenance, _ := json.Marshal(map[string]any{"parity_run_id": parityRunID, "source_tenant_id": "tenant-local", "materialization_contract": contractRef, "source": "immutable_parity_manifest"})
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin global materialization: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `INSERT INTO subscriber_global_marketops_evidence_runs
  (evidence_run_id,evidence_kind,algorithm_id,algorithm_version,execution_mode,source_scope,session_start_date,session_end_date,input_manifest_fingerprint,validation_contract_ref,immutable_baseline_ref,provenance,recorded_by,correlation_id,recorded_at)
VALUES ($1,$2,$3,$4,'legacy_materialized','legacy_materialization',$5,$6,$7,$8,$9,$10::jsonb,$11,$12,$13)
ON CONFLICT (evidence_run_id) DO NOTHING`, runID, entries[0].kind, entries[0].algorithmID, entries[0].algorithmVersion, first, last, manifestFingerprint, contractRef, "parity:"+parityRunID, string(provenance), workerIdentity, correlationID, now); err != nil {
		return 0, fmt.Errorf("record global evidence run: %w", err)
	}
	inserted := 0
	for _, item := range entries {
		recordProvenance, _ := json.Marshal(map[string]any{"parity_run_id": parityRunID, "legacy_record_id": item.id, "legacy_symbol": item.symbol, "legacy_created_at": item.created.UTC().Format(time.RFC3339Nano), "legacy_fingerprint": item.fingerprint})
		result, err := tx.ExecContext(ctx, `INSERT INTO subscriber_global_marketops_evidence_records
  (global_evidence_id,evidence_run_id,global_asset_id,session_date,evidence_kind,algorithm_id,algorithm_version,quality_state,source_system,source_event_id,source_run_id,evidence_fingerprint,validation_contract_ref,immutable_baseline_ref,payload,provenance,observed_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'legacy_parity_review',$9,$10,$11,$12,$13,$14::jsonb,$15::jsonb,$16)
ON CONFLICT (global_asset_id,session_date,evidence_kind,algorithm_id,algorithm_version,evidence_fingerprint) DO NOTHING`,
			"subglobalev-"+fingerprint(item.globalAssetID + "\x1f" + item.fingerprint)[:24], runID, item.globalAssetID, item.session, item.kind, item.algorithmID, item.algorithmVersion, normalizeQuality(item.quality), item.id, parityRunID, item.fingerprint, contractRef, "parity:"+parityRunID, item.payload, string(recordProvenance), item.created.UTC())
		if err != nil {
			return 0, fmt.Errorf("append global evidence for %s: %w", item.id, err)
		}
		count, _ := result.RowsAffected()
		inserted += int(count)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit global materialization: %w", err)
	}
	return inserted, nil
}

func parseKinds(raw string) ([]string, error) {
	set := map[string]struct{}{}
	for _, value := range strings.Split(raw, ",") {
		kind := strings.TrimSpace(value)
		if _, ok := allowedKinds[kind]; !ok {
			return nil, fmt.Errorf("unsupported evidence kind %q", kind)
		}
		set[kind] = struct{}{}
	}
	if len(set) == 0 {
		return nil, fmt.Errorf("at least one evidence kind is required")
	}
	items := make([]string, 0, len(set))
	for kind := range set {
		items = append(items, kind)
	}
	sort.Strings(items)
	return items, nil
}

func groupKey(item entry) string {
	return item.kind + "\x1f" + item.algorithmID + "\x1f" + item.algorithmVersion
}
func normalizeQuality(value string) string {
	if value == "usable" || value == "partial" || value == "invalid" {
		return value
	}
	return "partial"
}
func fingerprint(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

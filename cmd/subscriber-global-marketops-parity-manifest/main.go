// subscriber-global-marketops-parity-manifest records a bounded, immutable
// source manifest for tenant-local analytical records. It never calls a data
// provider, mutates a legacy table, or writes global analytical evidence.
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
	"sort"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	parityVersion  = "subscriber-global-analytical-parity-v1"
	workerIdentity = "subscriber-global-eod-reconciler"
)

var supportedEvidenceKinds = map[string]struct{}{
	"feature_vector": {}, "market_state": {}, "valuation": {}, "eeom": {}, "signal_assertion": {}, "outcome": {}, "options_snapshot": {}, "risk_reward": {},
}

type sourceRecord struct {
	kind, id, symbol, algorithmID, algorithmVersion, qualityState, payload string
	session                                                                time.Time
	created                                                                time.Time
	globalAssetID                                                          sql.NullString
	globalMatches                                                          int
}

type manifestEntry struct {
	sourceRecord
	fingerprint, mappingStatus, materializationStatus string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "subscriber global analytical parity manifest failed:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("subscriber-global-marketops-parity-manifest", flag.ContinueOnError)
	databaseURL := flags.String("database-url", os.Getenv("SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL"), "dedicated MarketOps global-worker database URL")
	evidenceKinds := flags.String("evidence-kinds", "feature_vector,market_state,valuation,eeom,signal_assertion,outcome,options_snapshot,risk_reward", "comma-separated supported evidence kinds")
	newestFirst := flags.Bool("newest-first", false, "select newest unmanifested source records first")
	limit := flags.Int("limit", 1000, "bounded source rows per immutable manifest (1-50000)")
	correlationID := flags.String("correlation-id", "", "optional operator correlation ID")
	actor := flags.String("actor", workerIdentity, "must be subscriber-global-eod-reconciler")
	execute := flags.Bool("execute", false, "persist the immutable parity manifest")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !*execute || strings.TrimSpace(*databaseURL) == "" || strings.TrimSpace(*actor) != workerIdentity {
		return fmt.Errorf("pass --execute, a dedicated database URL, and actor %q", workerIdentity)
	}
	if *limit < 1 || *limit > 50000 {
		return fmt.Errorf("limit must be between 1 and 50000")
	}
	kinds, err := parseEvidenceKinds(*evidenceKinds)
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

	entries, err := readManifestEntries(ctx, db, kinds, *limit, *newestFirst)
	if err != nil {
		return err
	}
	manifestFingerprint := manifestFingerprint(entries)
	now := time.Now().UTC()
	runID := newID("subglobalparity")
	report, err := manifestReport(kinds, entries, manifestFingerprint)
	if err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin parity manifest: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	mapped, unmapped, ambiguous := manifestCounts(entries)
	if _, err := tx.ExecContext(ctx, `INSERT INTO subscriber_global_marketops_legacy_parity_runs
  (parity_run_id,parity_version,source_tenant_id,execution_mode,requested_evidence_kinds,selected_record_count,mapped_record_count,unmapped_record_count,ambiguous_record_count,manifest_fingerprint,report,recorded_by,correlation_id,source_read_at,recorded_at)
VALUES ($1,$2,'tenant-local','shadow_read_only',$3,$4,$5,$6,$7,$8,$9::jsonb,$10,$11,$12,$12)`,
		runID, parityVersion, kinds, len(entries), mapped, unmapped, ambiguous, manifestFingerprint, string(report), workerIdentity, strings.TrimSpace(*correlationID), now); err != nil {
		return fmt.Errorf("insert parity manifest run: %w", err)
	}
	for _, entry := range entries {
		var globalAssetID any
		if entry.mappingStatus == "mapped" {
			globalAssetID = entry.globalAssetID.String
		}
		provenance, err := json.Marshal(map[string]any{
			"parity_version": parityVersion, "source_tenant_id": "tenant-local", "legacy_created_at": entry.created.UTC().Format(time.RFC3339Nano),
			"legacy_payload_fingerprint": entry.fingerprint, "mapping_basis": "canonical_symbol_to_global_identity_resolution",
		})
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO subscriber_global_marketops_legacy_parity_manifest_entries
  (parity_entry_id,parity_run_id,evidence_kind,legacy_record_id,legacy_symbol,legacy_session_date,legacy_algorithm_id,legacy_algorithm_version,legacy_quality_state,legacy_fingerprint,global_asset_id,mapping_status,global_materialization_status,provenance,manifested_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14::jsonb,$15)`,
			newID("subglobalparityentry"), runID, entry.kind, entry.id, entry.symbol, entry.session, entry.algorithmID,
			entry.algorithmVersion, entry.qualityState, entry.fingerprint, globalAssetID, entry.mappingStatus,
			entry.materializationStatus, string(provenance), now); err != nil {
			return fmt.Errorf("insert parity manifest entry %s: %w", entry.id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit parity manifest: %w", err)
	}
	fmt.Printf("parity_run_id=%s selected=%d mapped=%d unmapped=%d ambiguous=%d manifest_fingerprint=%s\n", runID, len(entries), mapped, unmapped, ambiguous, manifestFingerprint)
	return nil
}

func validateWorkload(ctx context.Context, db *sql.DB) error {
	var currentUser string
	var superuser, createRole, bypassRLS, member bool
	if err := db.QueryRowContext(ctx, `SELECT current_user,rolsuper,rolcreaterole,rolbypassrls,
  pg_has_role(current_user,'signalops_subscriber_global_eod', 'member')
FROM pg_roles WHERE rolname=current_user`).Scan(&currentUser, &superuser, &createRole, &bypassRLS, &member); err != nil {
		return fmt.Errorf("inspect parity workload identity: %w", err)
	}
	if superuser || createRole || bypassRLS || !member {
		return fmt.Errorf("parity workload must be a non-privileged member of signalops_subscriber_global_eod (got %s)", currentUser)
	}
	var viewRead, manifestWrite, rawFeatureRead, rawStateRead, rawOptionsRead, rawRiskRewardRead bool
	if err := db.QueryRowContext(ctx, `SELECT
  has_table_privilege(current_user,'subscriber_global_marketops_legacy_parity_source_v2', 'SELECT'),
  has_table_privilege(current_user,'subscriber_global_marketops_legacy_parity_runs', 'SELECT,INSERT'),
  has_table_privilege(current_user,'marketops_feature_observations', 'SELECT'),
  has_table_privilege(current_user,'marketops_market_states', 'SELECT'),
  has_table_privilege(current_user,'marketops_options_distribution_daily', 'SELECT'),
  has_table_privilege(current_user,'marketops_risk_reward_snapshots', 'SELECT')`).Scan(&viewRead, &manifestWrite, &rawFeatureRead, &rawStateRead, &rawOptionsRead, &rawRiskRewardRead); err != nil {
		return fmt.Errorf("inspect parity workload grants: %w", err)
	}
	if !viewRead || !manifestWrite || rawFeatureRead || rawStateRead || rawOptionsRead || rawRiskRewardRead {
		return fmt.Errorf("parity workload grants must allow manifest view/write and deny direct legacy-table reads")
	}
	return nil
}

func readManifestEntries(ctx context.Context, db *sql.DB, kinds []string, limit int, newestFirst bool) ([]manifestEntry, error) {
	quotedKinds := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		quotedKinds = append(quotedKinds, fmt.Sprintf("'%s'", kind))
	}
	orderDirection := "ASC"
	if newestFirst {
		orderDirection = "DESC"
	}
	rows, err := db.QueryContext(ctx, `SELECT source.evidence_kind,source.legacy_record_id,source.legacy_symbol,source.legacy_session_date,
  source.legacy_algorithm_id,source.legacy_algorithm_version,source.legacy_quality_state,source.legacy_payload::text,source.legacy_created_at,
  mapping.canonical_global_asset_id,mapping.match_count
FROM subscriber_global_marketops_legacy_parity_source_v2 source
LEFT JOIN LATERAL (
  SELECT min(resolution.canonical_global_asset_id) AS canonical_global_asset_id,
    count(DISTINCT resolution.canonical_global_asset_id)::integer AS match_count
  FROM subscriber_global_assets asset
  JOIN subscriber_global_asset_identity_resolutions resolution ON resolution.source_global_asset_id=asset.global_asset_id
  WHERE upper(asset.canonical_symbol)=upper(source.legacy_symbol)
) mapping ON true
WHERE source.evidence_kind IN (`+strings.Join(quotedKinds, ",")+`)
  AND NOT EXISTS (
    SELECT 1 FROM subscriber_global_marketops_legacy_parity_manifest_entries prior
    WHERE prior.evidence_kind=source.evidence_kind
      AND prior.legacy_record_id=source.legacy_record_id
      AND prior.mapping_status='mapped'
  )
ORDER BY source.evidence_kind,source.legacy_session_date `+orderDirection+`,source.legacy_record_id `+orderDirection+`
LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("read tenant-local parity source: %w", err)
	}
	defer rows.Close()
	entries := []manifestEntry{}
	for rows.Next() {
		var entry manifestEntry
		if err := rows.Scan(&entry.kind, &entry.id, &entry.symbol, &entry.session, &entry.algorithmID, &entry.algorithmVersion,
			&entry.qualityState, &entry.payload, &entry.created, &entry.globalAssetID, &entry.globalMatches); err != nil {
			return nil, fmt.Errorf("scan parity source: %w", err)
		}
		entry.fingerprint = sourceFingerprint(entry.sourceRecord)
		entry.mappingStatus, entry.materializationStatus = mappingStates(entry.globalMatches)
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func parseEvidenceKinds(raw string) ([]string, error) {
	set := map[string]struct{}{}
	for _, value := range strings.Split(raw, ",") {
		kind := strings.TrimSpace(value)
		if _, ok := supportedEvidenceKinds[kind]; !ok {
			return nil, fmt.Errorf("unsupported evidence kind %q", kind)
		}
		set[kind] = struct{}{}
	}
	if len(set) == 0 {
		return nil, fmt.Errorf("at least one evidence kind is required")
	}
	kinds := make([]string, 0, len(set))
	for kind := range set {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return kinds, nil
}

func sourceFingerprint(record sourceRecord) string {
	return fingerprint(strings.Join([]string{record.kind, record.id, strings.ToUpper(record.symbol), record.session.UTC().Format("2006-01-02"), record.algorithmID, record.algorithmVersion, record.qualityState, record.payload}, "\x1f"))
}

func mappingStates(matches int) (string, string) {
	switch {
	case matches == 1:
		return "mapped", "pending_global_materialization"
	case matches == 0:
		return "unmapped", "not_mappable"
	default:
		return "ambiguous", "not_mappable"
	}
}

func manifestFingerprint(entries []manifestEntry) string {
	parts := make([]string, 0, len(entries))
	for _, entry := range entries {
		parts = append(parts, strings.Join([]string{entry.kind, entry.id, entry.fingerprint, entry.mappingStatus, entry.globalAssetID.String}, "\x1f"))
	}
	return fingerprint(strings.Join(parts, "\x1e"))
}

func manifestCounts(entries []manifestEntry) (mapped, unmapped, ambiguous int) {
	for _, entry := range entries {
		switch entry.mappingStatus {
		case "mapped":
			mapped++
		case "unmapped":
			unmapped++
		case "ambiguous":
			ambiguous++
		}
	}
	return mapped, unmapped, ambiguous
}

func manifestReport(kinds []string, entries []manifestEntry, manifest string) ([]byte, error) {
	mapped, unmapped, ambiguous := manifestCounts(entries)
	byKind := map[string]int{}
	for _, entry := range entries {
		byKind[entry.kind]++
	}
	return json.Marshal(map[string]any{
		"parity_version": parityVersion, "source_scope": "tenant-local/read-only-view", "requested_evidence_kinds": kinds,
		"selected_by_kind": byKind, "mapped": mapped, "unmapped": unmapped, "ambiguous": ambiguous,
		"manifest_fingerprint": manifest, "global_evidence_imported": false,
	})
}

func fingerprint(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func newID(prefix string) string {
	var bytes [12]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(bytes[:])
}

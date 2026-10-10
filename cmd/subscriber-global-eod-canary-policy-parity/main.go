// subscriber-global-eod-canary-policy-parity verifies a completed global EOD
// canary against each declared immutable revision-selection context. It never
// makes a provider request and fails closed unless every comparison matches.
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

const (
	historicalAssurance  = "historical_assurance"
	currentMarketContext = "current_market_context"
)

type baselineResult struct {
	run, asset, symbol, payload, fingerprint string
	session                                  time.Time
}

type selectedObservation struct {
	payload, fingerprint, role, policyVersion string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "subscriber global EOD policy parity failed:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("subscriber-global-eod-canary-policy-parity", flag.ContinueOnError)
	globalDSN := flags.String("database-url", os.Getenv("SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL"), "dedicated global-EOD database URL")
	temporalDSN := flags.String("temporal-database-url", os.Getenv("SIGNALOPS_TEMPORAL_DATABASE_URL"), "read-only normalized-ledger database URL")
	authorizationID := flags.String("authorization-id", "", "required live authorization id")
	execute := flags.Bool("execute", false, "persist policy-aware parity reports")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !*execute || strings.TrimSpace(*globalDSN) == "" || strings.TrimSpace(*temporalDSN) == "" || strings.TrimSpace(*authorizationID) == "" {
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

	results, err := baselineResults(ctx, global, *authorizationID)
	if err != nil {
		return err
	}
	if len(results) != 2 {
		return fmt.Errorf("expected exactly two completed global baseline results, got %d", len(results))
	}

	failed := false
	for _, result := range results {
		for _, usageContext := range []string{historicalAssurance, currentMarketContext} {
			selected, err := resolveObservation(ctx, global, result.asset, result.session, usageContext)
			if err != nil {
				return fmt.Errorf("resolve %s %s: %w", result.symbol, usageContext, err)
			}
			comparisonPayload, comparisonEventID, comparisonSource, err := comparisonForContext(ctx, temporal, result, usageContext)
			status, reason, comparisonFingerprint := "matched", "", ""
			if err == sql.ErrNoRows {
				status, reason = "missing", "required comparison observation is absent"
				failed = true
			} else if err != nil {
				return fmt.Errorf("read %s %s comparison: %w", result.symbol, usageContext, err)
			} else {
				selectedCanonical, err := canonical(selected.payload)
				if err != nil {
					return err
				}
				comparisonCanonical, err := canonical(comparisonPayload)
				if err != nil {
					return err
				}
				comparisonFingerprint = fingerprint(comparisonCanonical)
				if string(selectedCanonical) != string(comparisonCanonical) {
					status, reason = "mismatched", "selected immutable observation differs from its declared context baseline"
					failed = true
				}
			}

			provenance, err := policyParityProvenance(ctx, global, result, selected, usageContext, comparisonSource)
			if err != nil {
				return err
			}
			if _, err := global.ExecContext(ctx, `INSERT INTO subscriber_global_eod_canary_policy_parity_reports
			  (policy_parity_report_id,live_run_id,global_asset_id,usage_context,selected_observation_role,selection_policy_version,comparison_source,comparison_event_id,selected_fingerprint,comparison_fingerprint,parity_status,mismatch_reason,provenance,compared_at)
			  VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13::jsonb,$14)`,
				newID("subeodpolicyparity"), result.run, result.asset, usageContext, selected.role, selected.policyVersion,
				comparisonSource, comparisonEventID, selected.fingerprint, comparisonFingerprint, status, reason, string(provenance), time.Now().UTC()); err != nil {
				return err
			}
			fmt.Printf("%s %s %s\n", result.symbol, usageContext, status)
		}
	}
	if failed {
		return fmt.Errorf("policy-aware parity is incomplete or mismatched")
	}
	return nil
}

func baselineResults(ctx context.Context, db *sql.DB, authorizationID string) ([]baselineResult, error) {
	rows, err := db.QueryContext(ctx, `SELECT result.live_run_id,result.global_asset_id,result.symbol,result.session_date,result.normalized_payload::text,result.normalized_fingerprint
		FROM subscriber_global_eod_canary_baseline_results result
		JOIN subscriber_global_eod_canary_live_runs run ON run.live_run_id=result.live_run_id
		WHERE run.authorization_id=$1 ORDER BY result.symbol`, authorizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := []baselineResult{}
	for rows.Next() {
		var result baselineResult
		if err := rows.Scan(&result.run, &result.asset, &result.symbol, &result.session, &result.payload, &result.fingerprint); err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, rows.Err()
}

func resolveObservation(ctx context.Context, db *sql.DB, asset string, session time.Time, usageContext string) (selectedObservation, error) {
	var selected selectedObservation
	err := db.QueryRowContext(ctx, `SELECT payload::text,payload_fingerprint,selected_observation_role,selection_policy_version
		FROM subscriber_global_eod_resolved_observations
		WHERE global_asset_id=$1 AND session_date=$2 AND usage_context=$3`,
		asset, session.Format("2006-01-02"), usageContext).Scan(&selected.payload, &selected.fingerprint, &selected.role, &selected.policyVersion)
	return selected, err
}

func comparisonForContext(ctx context.Context, temporal *sql.DB, result baselineResult, usageContext string) (payload, eventID, source string, err error) {
	if usageContext == currentMarketContext {
		return result.payload, "", "global_canary_baseline", nil
	}
	err = temporal.QueryRowContext(ctx, `SELECT event_id,normalized_payload::text
		FROM normalized_event_ledger
		WHERE tenant_id='tenant-local' AND source_id='src-massive' AND dataset='equity_eod_prices'
		  AND normalized_payload->>'symbol'=$1 AND normalized_payload->>'observation_date'=$2
		ORDER BY processing_time DESC LIMIT 1`, result.symbol, result.session.Format("2006-01-02")).Scan(&eventID, &payload)
	return payload, eventID, "tenant_local_initial_capture", err
}

func policyParityProvenance(ctx context.Context, db *sql.DB, result baselineResult, selected selectedObservation, usageContext, comparisonSource string) ([]byte, error) {
	var reviewRequiredFields []string
	rows, err := db.QueryContext(ctx, `SELECT field_name FROM subscriber_global_eod_provider_revision_field_deltas
		WHERE global_asset_id=$1 AND session_date=$2 AND delta_class='provider_revision' AND materiality='review_required'
		ORDER BY field_name`, result.asset, result.session.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var field string
		if err := rows.Scan(&field); err != nil {
			return nil, err
		}
		reviewRequiredFields = append(reviewRequiredFields, field)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{
		"comparison_contract":          "s4-policy-aware-parity-v1",
		"usage_context":                usageContext,
		"selection_policy_version":     selected.policyVersion,
		"selected_observation_role":    selected.role,
		"comparison_source":            comparisonSource,
		"algorithm_version":            "subscriber-global-eod-baseline-v1",
		"review_required_fields":       reviewRequiredFields,
		"legacy_parity_interpretation": "raw tenant-local versus global comparison may differ when a provider revision is retained; it is not a policy-parity failure",
	})
}

func canonical(raw string) ([]byte, error) {
	var value map[string]any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, err
	}
	out := map[string]any{}
	for _, key := range []string{"provider", "dataset", "provider_event_id", "symbol", "observation_date", "open", "high", "low", "close", "volume", "vwap"} {
		out[key] = value[key]
	}
	return json.Marshal(out)
}

func fingerprint(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func newID(prefix string) string {
	var bytes [12]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(bytes[:])
}

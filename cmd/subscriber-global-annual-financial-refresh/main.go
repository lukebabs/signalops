// subscriber-global-annual-financial-refresh centrally captures annual FMP
// statements and reference ratios for the governed warm-EOD cohort. It never
// reads a subscriber list, creates tenant copies, or changes valuation scores.
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

	"github.com/lukebabs/signalops/internal/adapters/marketdata/fmp"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	workerIdentity = "subscriber-global-eod-reconciler"
	algorithmID    = "marketops.fundamental_annual.fmp"
	algorithmV1    = "v1"
	contractRef    = "subscriber-global-annual-financial-capture/v1"
)

type asset struct{ id, symbol string }
type outcome struct {
	asset    asset
	snapshot fmp.AnnualFinancialSnapshot
	err      error
}

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "subscriber global annual financial refresh failed:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("subscriber-global-annual-financial-refresh", flag.ContinueOnError)
	databaseURL := flags.String("database-url", strings.TrimSpace(os.Getenv("SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL")), "dedicated primary global-worker database URL")
	limit := flags.Int("max-assets", 1000, "maximum warm assets to refresh (1-1000)")
	interval := flags.Duration("request-interval", 250*time.Millisecond, "minimum interval between FMP calls")
	sessionValue := flags.String("session-date", "", "capture date YYYY-MM-DD; default is current UTC date")
	correlationID := flags.String("correlation-id", "", "operator correlation id")
	dryRun := flags.Bool("dry-run", false, "call FMP and report coverage without writing evidence")
	execute := flags.Bool("execute", false, "append immutable global annual evidence")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if (*dryRun && *execute) || (!*dryRun && !*execute) || strings.TrimSpace(*databaseURL) == "" {
		return errors.New("pass exactly one of --dry-run or --execute and a dedicated database URL")
	}
	if *limit < 1 || *limit > 1000 {
		return errors.New("max-assets must be between 1 and 1000")
	}
	if *interval < 250*time.Millisecond {
		return errors.New("request-interval must be at least 250ms (240 FMP calls/minute)")
	}
	session, err := captureSession(*sessionValue)
	if err != nil {
		return err
	}
	db, err := sql.Open("pgx", strings.TrimSpace(*databaseURL))
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("connect dedicated primary: %w", err)
	}
	if _, err := db.ExecContext(ctx, "SET ROLE signalops_subscriber_global_eod"); err != nil {
		return fmt.Errorf("assume global annual evidence role: %w", err)
	}
	defer db.ExecContext(context.Background(), "RESET ROLE")
	assets, err := loadWarmAssets(ctx, db, *limit)
	if err != nil {
		return err
	}
	clientConfig := fmp.LoadClientConfigFromEnv()
	clientConfig.MinRequestInterval = *interval
	client, err := fmp.NewClient(clientConfig)
	if err != nil {
		return err
	}
	outcomes := make([]outcome, 0, len(assets))
	for _, item := range assets {
		snapshot, fetchErr := getWithRetry(ctx, client, item.symbol)
		outcomes = append(outcomes, outcome{asset: item, snapshot: snapshot, err: fetchErr})
	}
	succeeded, failed := summarize(outcomes)
	if *dryRun {
		fmt.Printf("dry_run=true warm_assets=%d succeeded=%d failed=%d fmp_calls=%d session_date=%s\n", len(assets), succeeded, failed, client.Calls(), session.Format("2006-01-02"))
		return nil
	}
	correlation := strings.TrimSpace(*correlationID)
	if correlation == "" {
		correlation = "subscriber-global-annual-financial-" + session.Format("20060102")
	}
	inserted, err := appendOutcomes(ctx, db, outcomes, session, correlation)
	if err != nil {
		return err
	}
	fmt.Printf("warm_assets=%d succeeded=%d failed=%d inserted=%d fmp_calls=%d session_date=%s correlation_id=%s\n", len(assets), succeeded, failed, inserted, client.Calls(), session.Format("2006-01-02"), correlation)
	return nil
}

func captureSession(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Now().UTC().Truncate(24 * time.Hour), nil
	}
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid session-date %q", value)
	}
	return parsed.UTC(), nil
}

func loadWarmAssets(ctx context.Context, db *sql.DB, limit int) ([]asset, error) {
	rows, err := db.QueryContext(ctx, `SELECT global_asset_id,canonical_symbol FROM subscriber_global_warm_eod_assets ORDER BY priority,global_asset_id LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("load global warm cohort: %w", err)
	}
	defer rows.Close()
	assets := []asset{}
	for rows.Next() {
		var item asset
		if err := rows.Scan(&item.id, &item.symbol); err != nil {
			return nil, err
		}
		item.symbol = strings.ToUpper(strings.TrimSpace(item.symbol))
		if item.id != "" && item.symbol != "" {
			assets = append(assets, item)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(assets) == 0 {
		return nil, errors.New("no enabled global warm assets are available")
	}
	return assets, nil
}

func getWithRetry(ctx context.Context, client *fmp.Client, symbol string) (fmp.AnnualFinancialSnapshot, error) {
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		snapshot, err := client.GetAnnualFinancialSnapshot(ctx, symbol)
		if err == nil {
			return snapshot, nil
		}
		last = err
		if !retryable(err) || attempt == 2 {
			break
		}
		delay := time.Duration(attempt+1) * time.Second
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmp.AnnualFinancialSnapshot{}, ctx.Err()
		case <-timer.C:
		}
	}
	return fmp.AnnualFinancialSnapshot{}, last
}

func retryable(err error) bool {
	var status *fmp.HTTPError
	if errors.As(err, &status) {
		return status.StatusCode() == 429 || status.StatusCode() >= 500
	}
	return true
}

func appendOutcomes(ctx context.Context, db *sql.DB, outcomes []outcome, session time.Time, correlation string) (int, error) {
	fingerprints := make([]string, 0, len(outcomes))
	for _, value := range outcomes {
		fingerprints = append(fingerprints, outcomeFingerprint(value))
	}
	sort.Strings(fingerprints)
	runID := "subglobalannual-" + digest(strings.Join(fingerprints, "\x1f"))[:24]
	succeeded, failed := summarize(outcomes)
	provenance, _ := json.Marshal(map[string]any{
		"provider": "fmp", "coverage_scope": "subscriber_global_warm_eod_assets", "annual_period_limit": 5,
		"ratio_usage": "reference_only_local_scoring_required", "succeeded_assets": succeeded, "failed_assets": failed,
	})
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO subscriber_global_marketops_evidence_runs
  (evidence_run_id,evidence_kind,algorithm_id,algorithm_version,execution_mode,source_scope,session_start_date,session_end_date,input_manifest_fingerprint,validation_contract_ref,immutable_baseline_ref,provenance,recorded_by,correlation_id,recorded_at)
VALUES ($1,'fundamental_annual',$2,$3,'provider_capture','global_provider_capture',$4,$4,$5,$6,$7,$8::jsonb,$9,$10,now()) ON CONFLICT (evidence_run_id) DO NOTHING`,
		runID, algorithmID, algorithmV1, session, "sha256:"+digest(strings.Join(fingerprints, "\x1f")), contractRef, "fmp-starter-annual-v1", string(provenance), workerIdentity, correlation)
	if err != nil {
		return 0, fmt.Errorf("record annual financial evidence run: %w", err)
	}
	inserted := 0
	for _, value := range outcomes {
		payload, quality, sourceEventID, observedAt := outcomePayload(value)
		fingerprint := digest(string(payload))
		recordProvenance, _ := json.Marshal(map[string]any{"provider": "fmp", "capture_kind": "annual_financials", "warm_cohort": true})
		result, err := tx.ExecContext(ctx, `INSERT INTO subscriber_global_marketops_evidence_records
  (global_evidence_id,evidence_run_id,global_asset_id,session_date,evidence_kind,algorithm_id,algorithm_version,quality_state,source_system,source_event_id,source_run_id,evidence_fingerprint,validation_contract_ref,immutable_baseline_ref,payload,provenance,observed_at)
VALUES ($1,$2,$3,$4,'fundamental_annual',$5,$6,$7,'fmp',$8,$2,$9,$10,$11,$12::jsonb,$13::jsonb,$14)
ON CONFLICT (global_asset_id,session_date,evidence_kind,algorithm_id,algorithm_version,evidence_fingerprint) DO NOTHING`,
			"subglobalannualrec-"+digest(value.asset.id + "\x1f" + fingerprint)[:24], runID, value.asset.id, session, algorithmID, algorithmV1, quality,
			sourceEventID, fingerprint, contractRef, "fmp-starter-annual-v1", string(payload), string(recordProvenance), observedAt)
		if err != nil {
			return 0, fmt.Errorf("append annual financial evidence for %s: %w", value.asset.symbol, err)
		}
		count, _ := result.RowsAffected()
		inserted += int(count)
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return inserted, nil
}

func outcomePayload(value outcome) ([]byte, string, string, time.Time) {
	if value.err != nil {
		payload, _ := json.Marshal(map[string]any{"symbol": value.asset.symbol, "capture_error": truncate(value.err.Error(), 512), "annual_periods": 0})
		return payload, "invalid", "fmp:annual:error:" + value.asset.symbol, time.Now().UTC()
	}
	payload, _ := json.Marshal(map[string]any{
		"symbol": value.asset.symbol, "annual_periods": value.snapshot.Periods,
		"ratio_references_by_period":      value.snapshot.RatioReferences,
		"key_metric_references_by_period": value.snapshot.KeyMetricReferences,
		"provider_request_ids":            value.snapshot.ProviderRequestIDs,
		"retrieved_at":                    value.snapshot.RetrievedAt.UTC().Format(time.RFC3339Nano),
		"ratio_usage":                     "reference_only_local_scoring_required",
	})
	latest := value.snapshot.Periods[0].PeriodEnd.Format("2006-01-02")
	return payload, "usable", "fmp:annual:" + value.asset.symbol + ":" + latest, value.snapshot.RetrievedAt.UTC()
}

func summarize(outcomes []outcome) (int, int) {
	succeeded := 0
	for _, value := range outcomes {
		if value.err == nil {
			succeeded++
		}
	}
	return succeeded, len(outcomes) - succeeded
}

func outcomeFingerprint(value outcome) string {
	payload, _, _, _ := outcomePayload(value)
	return digest(value.asset.id + "\x1f" + string(payload))
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}

// subscriber-global-saf-benchmark-materializer appends point-in-time matched
// benchmarks to the preserved historical-outcome projection. It never updates
// marketops_signal_outcomes or rewrites legacy evidence.
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
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	calculationVersion = "saf_benchmark.v1"
	selectionPolicy    = "historical_assurance_initial_capture.v1"
	legacyDefaultList  = "sublist-tenant-local-legacy-default"
)

type observation struct {
	id, assetID, symbol, sector string
	origin, maturity            time.Time
	forward                     float64
}
type price struct {
	eventID, fingerprint string
	close                float64
	observed, processed  time.Time
}
type benchmark struct {
	observation                     observation
	kind, symbol, segment, state    string
	origin, maturity                *price
	benchmarkReturn, relativeReturn *float64
}

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "subscriber global SAF benchmark materializer failed:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("subscriber-global-saf-benchmark-materializer", flag.ContinueOnError)
	primaryURL := flags.String("database-url", strings.TrimSpace(os.Getenv("SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL")), "dedicated MarketOps primary database URL")
	temporalURL := flags.String("temporal-database-url", strings.TrimSpace(os.Getenv("SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_TEMPORAL_DATABASE_URL")), "dedicated MarketOps temporal database URL")
	limit := flags.Int("max-observations", 500, "maximum legacy observations to inspect (1-500)")
	correlation := flags.String("correlation-id", "", "operator correlation id")
	dryRun := flags.Bool("dry-run", false, "calculate without writes")
	execute := flags.Bool("execute", false, "append matched benchmark observations")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if (*dryRun && *execute) || (!*dryRun && !*execute) {
		return errors.New("pass exactly one of --dry-run or --execute")
	}
	if strings.TrimSpace(*primaryURL) == "" || strings.TrimSpace(*temporalURL) == "" {
		return errors.New("dedicated primary and temporal database URLs are required")
	}
	if *limit < 1 || *limit > 500 {
		return errors.New("max-observations must be between 1 and 500")
	}
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
		return fmt.Errorf("assume SAF benchmark writer role: %w", err)
	}
	defer primary.ExecContext(context.Background(), "RESET ROLE")
	items, err := loadObservations(ctx, primary, *limit)
	if err != nil {
		return err
	}
	calculated := make([]benchmark, 0, len(items)*2)
	cache := map[string]*price{}
	for _, item := range items {
		for _, choice := range benchmarkChoices(item) {
			choice.origin, err = loadInitialPrice(ctx, temporal, cache, choice.symbol, item.origin)
			if err != nil {
				return err
			}
			choice.maturity, err = loadInitialPrice(ctx, temporal, cache, choice.symbol, item.maturity)
			if err != nil {
				return err
			}
			if choice.state == "matched" {
				if choice.origin == nil {
					choice.state = "origin_price_unavailable"
				} else if choice.maturity == nil {
					choice.state = "maturity_price_unavailable"
				} else {
					value := choice.maturity.close/choice.origin.close - 1
					relative := item.forward - value
					choice.benchmarkReturn, choice.relativeReturn = &value, &relative
				}
			}
			calculated = append(calculated, choice)
		}
	}
	matched, unmapped, unavailable := 0, 0, 0
	for _, item := range calculated {
		switch item.state {
		case "matched":
			matched++
		case "sector_unmapped":
			unmapped++
		default:
			unavailable++
		}
	}
	if *dryRun {
		fmt.Printf("dry_run=true legacy_list=%s observations=%d benchmark_rows=%d matched=%d sector_unmapped=%d price_unavailable=%d calculation_version=%s\n", legacyDefaultList, len(items), len(calculated), matched, unmapped, unavailable, calculationVersion)
		return nil
	}
	if strings.TrimSpace(*correlation) == "" {
		*correlation = "saf-benchmark-" + time.Now().UTC().Format("20060102")
	}
	inserted, err := appendBenchmarks(ctx, primary, calculated, *correlation)
	if err != nil {
		return err
	}
	fmt.Printf("legacy_list=%s observations=%d benchmark_rows=%d inserted=%d matched=%d sector_unmapped=%d price_unavailable=%d calculation_version=%s correlation_id=%s\n", legacyDefaultList, len(items), len(calculated), inserted, matched, unmapped, unavailable, calculationVersion, *correlation)
	return nil
}

func loadObservations(ctx context.Context, db *sql.DB, limit int) ([]observation, error) {
	rows, err := db.QueryContext(ctx, `SELECT observation_id, observation.global_asset_id, observation.symbol, COALESCE(asset.sector,''), origin_session_date, matured_session_date, forward_return
FROM subscriber_gateway_global_signal_assurance_observations observation
JOIN subscriber_global_assets asset ON asset.global_asset_id=observation.global_asset_id
JOIN subscriber_watchlist_memberships member ON member.global_asset_id=observation.global_asset_id AND member.list_id=$1
WHERE matured_session_date IS NOT NULL AND forward_return IS NOT NULL
ORDER BY matured_session_date, observation_id LIMIT $2`, legacyDefaultList, limit)
	if err != nil {
		return nil, fmt.Errorf("load legacy SAF observations: %w", err)
	}
	defer rows.Close()
	out := []observation{}
	for rows.Next() {
		var item observation
		if err := rows.Scan(&item.id, &item.assetID, &item.symbol, &item.sector, &item.origin, &item.maturity, &item.forward); err != nil {
			return nil, err
		}
		item.symbol = strings.ToUpper(strings.TrimSpace(item.symbol))
		out = append(out, item)
	}
	return out, rows.Err()
}

func benchmarkChoices(item observation) []benchmark {
	sector, symbol := sectorBenchmark(item.sector)
	choices := []benchmark{{observation: item, kind: "broad_market", symbol: "SPY", state: "matched"}}
	if symbol == "" {
		choices = append(choices, benchmark{observation: item, kind: "sector", state: "sector_unmapped"})
	} else {
		choices = append(choices, benchmark{observation: item, kind: "sector", symbol: symbol, segment: sector, state: "matched"})
	}
	return choices
}

func sectorBenchmark(raw string) (string, string) {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case strings.Contains(v, "technology") || strings.Contains(v, "software") || strings.Contains(v, "semiconductor") || strings.Contains(v, "computer"):
		return "sector_technology", "XLK"
	case strings.Contains(v, "financial") || strings.Contains(v, "bank") || strings.Contains(v, "insurance"):
		return "sector_financials", "XLF"
	case strings.Contains(v, "health") || strings.Contains(v, "biotech"):
		return "sector_healthcare", "XLV"
	case strings.Contains(v, "energy") || strings.Contains(v, "oil"):
		return "sector_energy", "XLE"
	case strings.Contains(v, "consumer staples"):
		return "sector_staples", "XLP"
	case strings.Contains(v, "consumer discretionary") || strings.Contains(v, "retail") || strings.Contains(v, "automotive") || strings.Contains(v, "aircraft") || strings.Contains(v, "amusement"):
		return "sector_discretionary", "XLY"
	case strings.Contains(v, "communication") || strings.Contains(v, "television") || strings.Contains(v, "cable"):
		return "sector_communication", "XLC"
	case strings.Contains(v, "industrial") || strings.Contains(v, "aerospace") || strings.Contains(v, "transportation"):
		return "sector_industrials", "XLI"
	case strings.Contains(v, "material"):
		return "sector_materials", "XLB"
	case strings.Contains(v, "real estate"):
		return "sector_real_estate", "XLRE"
	case strings.Contains(v, "utilit"):
		return "sector_utilities", "XLU"
	default:
		return "", ""
	}
}

func loadInitialPrice(ctx context.Context, db *sql.DB, cache map[string]*price, symbol string, session time.Time) (*price, error) {
	if symbol == "" {
		return nil, nil
	}
	key := symbol + "|" + session.Format("2006-01-02")
	if value, ok := cache[key]; ok {
		return value, nil
	}
	var value price
	err := db.QueryRowContext(ctx, `SELECT event_id, NULLIF(normalized_payload->>'close','')::double precision, observation_time, processing_time
FROM normalized_event_ledger WHERE tenant_id='tenant-local' AND dataset='equity_eod_prices'
 AND upper(normalized_payload->>'symbol')=upper($1) AND observation_time::date=$2::date
 AND NULLIF(normalized_payload->>'close','')::double precision > 0
ORDER BY processing_time ASC, event_id ASC LIMIT 1`, symbol, session).Scan(&value.eventID, &value.close, &value.observed, &value.processed)
	if err == sql.ErrNoRows {
		cache[key] = nil
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load initial %s benchmark price: %w", symbol, err)
	}
	value.fingerprint = hash(value.eventID + "|" + fmt.Sprintf("%.12g", value.close) + "|" + value.observed.UTC().Format(time.RFC3339Nano) + "|" + value.processed.UTC().Format(time.RFC3339Nano))
	cache[key] = &value
	return &value, nil
}

func appendBenchmarks(ctx context.Context, db *sql.DB, rows []benchmark, correlation string) (int, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	inserted := 0
	for _, item := range rows {
		provenance, _ := json.Marshal(map[string]any{"correlation_id": correlation, "source_scope": "tenant-local legacy default 132", "outcome_immutable": true, "selection_policy": selectionPolicy, "sector_source_value": item.observation.sector, "origin_price_observed_at": timeString(item.origin), "maturity_price_observed_at": timeString(item.maturity)})
		result, err := tx.ExecContext(ctx, `INSERT INTO subscriber_global_saf_benchmark_observations (benchmark_observation_id,source_observation_id,global_asset_id,benchmark_kind,benchmark_symbol,benchmark_segment_key,benchmark_resolution_state,origin_session_date,matured_session_date,origin_price,matured_price,benchmark_return,benchmark_relative_return,source_origin_event_id,source_matured_event_id,source_origin_fingerprint,source_matured_fingerprint,selection_policy_version,calculation_version,calculation_run_id,provenance,observed_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21::jsonb,now()) ON CONFLICT (source_observation_id,benchmark_kind,calculation_version) DO NOTHING`,
			"safbench-"+hash(item.observation.id + "|" + item.kind + "|" + calculationVersion)[:24], item.observation.id, item.observation.assetID, item.kind, item.symbol, item.segment, item.state, item.observation.origin, item.observation.maturity, priceValue(item.origin), priceValue(item.maturity), item.benchmarkReturn, item.relativeReturn, eventID(item.origin), eventID(item.maturity), fingerprint(item.origin), fingerprint(item.maturity), selectionPolicy, calculationVersion, "safbench-"+hash(correlation)[:20], string(provenance))
		if err != nil {
			return inserted, fmt.Errorf("append %s benchmark for %s: %w", item.kind, item.observation.id, err)
		}
		count, _ := result.RowsAffected()
		inserted += int(count)
	}
	if err := tx.Commit(); err != nil {
		return inserted, err
	}
	return inserted, nil
}
func priceValue(p *price) any {
	if p == nil {
		return nil
	}
	return p.close
}
func eventID(p *price) string {
	if p == nil {
		return ""
	}
	return p.eventID
}
func fingerprint(p *price) string {
	if p == nil {
		return ""
	}
	return p.fingerprint
}
func timeString(p *price) string {
	if p == nil {
		return ""
	}
	return p.observed.UTC().Format(time.RFC3339Nano)
}
func hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

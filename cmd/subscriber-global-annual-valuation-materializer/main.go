// subscriber-global-annual-valuation-materializer derives parallel VC/DOSM v4
// research results from immutable global annual financial evidence and global
// completed-session EOD closes. It neither calls providers nor reads tenant
// lists, and it never overwrites the live v3 profile.
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
	"github.com/lukebabs/signalops/internal/marketops/valuation"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	annualValuationContract = "subscriber-global-vc-dosm-annual-v4/v1"
	annualBaseline          = "fmp-starter-annual-v1"
	vcAlgorithm             = "signalops.algorithms.valuation_composite_v4_annual"
	dosmAlgorithm           = "signalops.algorithms.distressed_opportunity_scoring_v4_annual"
	workerIdentity          = "subscriber-global-eod-reconciler"
)

type sourceCandidate struct {
	globalAssetID, symbol, annualEvidenceID, annualFingerprint, priceEvidenceID, priceFingerprint string
	annualPayload                                                                                 []byte
	price                                                                                         float64
	priceOK                                                                                       bool
	priceSession                                                                                  time.Time
}

type resultCandidate struct {
	source            sourceCandidate
	input             valuation.AnnualInput
	result            valuation.AnnualResult
	annualAvailableAt time.Time
}

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "subscriber global annual valuation materializer failed:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("subscriber-global-annual-valuation-materializer", flag.ContinueOnError)
	databaseURL := flags.String("database-url", strings.TrimSpace(os.Getenv("SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL")), "dedicated primary global-worker database URL")
	limit := flags.Int("max-assets", 1000, "maximum warm assets to evaluate (1-1000)")
	sessionValue := flags.String("session-date", "", "completed-session anchor YYYY-MM-DD; default is latest weekday")
	correlationID := flags.String("correlation-id", "", "operator correlation id")
	dryRun := flags.Bool("dry-run", false, "calculate annual v4 results without writing evidence")
	execute := flags.Bool("execute", false, "append immutable annual v4 valuation evidence")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if (*dryRun == *execute) || strings.TrimSpace(*databaseURL) == "" {
		return errors.New("pass exactly one of --dry-run or --execute and a dedicated database URL")
	}
	if *limit < 1 || *limit > 1000 {
		return errors.New("max-assets must be between 1 and 1000")
	}
	anchor, err := completedSession(*sessionValue)
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
	sources, err := loadSources(ctx, db, anchor, *limit)
	if err != nil {
		return err
	}
	results := make([]resultCandidate, 0, len(sources))
	for _, source := range sources {
		results = append(results, evaluate(source, anchor))
	}
	eligible, missingAnnual, missingPrice := summarize(results)
	if *dryRun {
		fmt.Printf("dry_run=true warm_assets=%d eligible=%d missing_annual=%d missing_price=%d session_date=%s model=%s\n", len(results), eligible, missingAnnual, missingPrice, anchor.Format("2006-01-02"), valuation.AnnualModelVersion)
		return nil
	}
	correlation := strings.TrimSpace(*correlationID)
	if correlation == "" {
		correlation = "subscriber-global-annual-valuation-" + anchor.Format("20060102")
	}
	inserted, err := appendResults(ctx, db, results, anchor, correlation)
	if err != nil {
		return err
	}
	fmt.Printf("warm_assets=%d eligible=%d missing_annual=%d missing_price=%d inserted=%d session_date=%s model=%s correlation_id=%s\n", len(results), eligible, missingAnnual, missingPrice, inserted, anchor.Format("2006-01-02"), valuation.AnnualModelVersion, correlation)
	return nil
}

func completedSession(value string) (time.Time, error) {
	if strings.TrimSpace(value) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid session-date %q", value)
		}
		return parsed.UTC(), nil
	}
	date := time.Now().UTC().Truncate(24 * time.Hour)
	for date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
		date = date.AddDate(0, 0, -1)
	}
	return date, nil
}

func loadSources(ctx context.Context, db *sql.DB, anchor time.Time, limit int) ([]sourceCandidate, error) {
	rows, err := db.QueryContext(ctx, `WITH annual AS (
  SELECT DISTINCT ON (record.global_asset_id) record.global_asset_id,record.global_evidence_id,record.evidence_fingerprint,record.payload
  FROM subscriber_global_marketops_evidence_records record
  JOIN subscriber_global_marketops_evidence_runs run ON run.evidence_run_id=record.evidence_run_id
  WHERE record.evidence_kind='fundamental_annual' AND record.algorithm_id='marketops.fundamental_annual.fmp'
    AND record.algorithm_version='v1' AND record.quality_state='usable' AND run.source_scope='global_provider_capture'
  ORDER BY record.global_asset_id,record.observed_at DESC,record.global_evidence_id DESC
), price AS (
  SELECT DISTINCT ON (record.global_asset_id) record.global_asset_id,record.global_evidence_id,record.evidence_fingerprint,record.session_date,(record.payload->>'close')::double precision AS close
  FROM subscriber_global_marketops_evidence_records record
  JOIN subscriber_global_marketops_evidence_runs run ON run.evidence_run_id=record.evidence_run_id
  WHERE record.evidence_kind='eod_bar' AND record.algorithm_id='marketops.equity_eod.initial_capture'
    AND record.algorithm_version='v1' AND record.quality_state='usable' AND run.source_scope IN ('global_provider_capture','legacy_materialization')
    AND record.session_date <= $1::date
  ORDER BY record.global_asset_id,record.session_date DESC,record.observed_at DESC,record.global_evidence_id DESC
)
SELECT warm.global_asset_id,warm.canonical_symbol,
       COALESCE(annual.global_evidence_id,''),COALESCE(annual.evidence_fingerprint,''),COALESCE(annual.payload::text,''),
       COALESCE(price.global_evidence_id,''),COALESCE(price.evidence_fingerprint,''),price.close,price.session_date
FROM subscriber_global_warm_eod_assets warm
LEFT JOIN annual ON annual.global_asset_id=warm.global_asset_id
LEFT JOIN price ON price.global_asset_id=warm.global_asset_id
ORDER BY warm.priority,warm.global_asset_id LIMIT $2`, anchor, limit)
	if err != nil {
		return nil, fmt.Errorf("load annual/eod global evidence: %w", err)
	}
	defer rows.Close()
	out := []sourceCandidate{}
	for rows.Next() {
		var value sourceCandidate
		var close sql.NullFloat64
		var priceSession sql.NullTime
		if err := rows.Scan(&value.globalAssetID, &value.symbol, &value.annualEvidenceID, &value.annualFingerprint, &value.annualPayload, &value.priceEvidenceID, &value.priceFingerprint, &close, &priceSession); err != nil {
			return nil, err
		}
		value.symbol = strings.ToUpper(strings.TrimSpace(value.symbol))
		value.price, value.priceOK = close.Float64, close.Valid && close.Float64 > 0
		if priceSession.Valid {
			value.priceSession = priceSession.Time.UTC()
		}
		out = append(out, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, errors.New("no enabled global warm assets are available")
	}
	return out, nil
}

func evaluate(source sourceCandidate, anchor time.Time) resultCandidate {
	candidate := resultCandidate{source: source}
	if source.annualEvidenceID == "" {
		candidate.result = valuation.EvaluateAnnual(valuation.AnnualInput{Ticker: source.symbol, Price: source.price})
		candidate.result.Reasons = append(candidate.result.Reasons, "immutable global annual financial evidence unavailable")
		return candidate
	}
	var payload struct {
		Periods []fmp.AnnualFinancialPeriod `json:"annual_periods"`
	}
	if err := json.Unmarshal(source.annualPayload, &payload); err != nil || len(payload.Periods) == 0 {
		candidate.result = valuation.EvaluateAnnual(valuation.AnnualInput{Ticker: source.symbol, Price: source.price})
		candidate.result.Reasons = append(candidate.result.Reasons, "annual financial evidence payload is invalid")
		return candidate
	}
	session := anchor
	if !source.priceSession.IsZero() {
		session = source.priceSession
	}
	candidate.input, candidate.annualAvailableAt, _ = valuation.AnnualInputFromFMP(source.symbol, payload.Periods, source.price, session)
	candidate.result = valuation.EvaluateAnnual(candidate.input)
	return candidate
}

func summarize(results []resultCandidate) (eligible, missingAnnual, missingPrice int) {
	for _, result := range results {
		if result.result.Eligible {
			eligible++
		}
		if result.source.annualEvidenceID == "" {
			missingAnnual++
		}
		if !result.source.priceOK {
			missingPrice++
		}
	}
	return
}

func appendResults(ctx context.Context, db *sql.DB, results []resultCandidate, anchor time.Time, correlation string) (int, error) {
	inserted := 0
	for _, output := range []struct{ algorithmID, metric string }{{vcAlgorithm, "vc"}, {dosmAlgorithm, "dosm"}} {
		runID, err := appendOutput(ctx, db, results, anchor, correlation, output.algorithmID, output.metric)
		if err != nil {
			return inserted, err
		}
		inserted += runID
	}
	return inserted, nil
}

func appendOutput(ctx context.Context, db *sql.DB, results []resultCandidate, anchor time.Time, correlation, algorithmID, metric string) (int, error) {
	fingerprints := make([]string, 0, len(results))
	for _, candidate := range results {
		fingerprints = append(fingerprints, outputFingerprint(candidate, metric))
	}
	sort.Strings(fingerprints)
	runID := "subglobalannualval-" + digest(algorithmID + "\x1f" + strings.Join(fingerprints, "\x1f"))[:24]
	provenance, _ := json.Marshal(map[string]any{"input_scope": "platform_global", "financial_evidence_kind": "fundamental_annual", "price_evidence_kind": "eod_bar", "profile": "annual_5y", "model_version": valuation.AnnualModelVersion})
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO subscriber_global_marketops_evidence_runs
 (evidence_run_id,evidence_kind,algorithm_id,algorithm_version,execution_mode,source_scope,session_start_date,session_end_date,input_manifest_fingerprint,validation_contract_ref,immutable_baseline_ref,provenance,recorded_by,correlation_id,recorded_at)
 VALUES ($1,'valuation',$2,$3,'provider_capture','global_provider_capture',$4,$4,$5,$6,$7,$8::jsonb,$9,$10,now()) ON CONFLICT (evidence_run_id) DO NOTHING`,
		runID, algorithmID, valuation.AnnualModelVersion, anchor, "sha256:"+digest(strings.Join(fingerprints, "\x1f")), annualValuationContract, annualBaseline, string(provenance), workerIdentity, correlation)
	if err != nil {
		return 0, fmt.Errorf("append %s run: %w", metric, err)
	}
	inserted := 0
	for _, candidate := range results {
		payload, score, fair, classification := outputPayload(candidate, metric)
		fingerprint := digest(string(payload))
		quality := "usable"
		if !candidate.result.Eligible {
			quality = "partial"
		}
		observation := anchor
		if !candidate.source.priceSession.IsZero() {
			observation = candidate.source.priceSession
		}
		provenance, _ := json.Marshal(map[string]any{"annual_financial_evidence_id": candidate.source.annualEvidenceID, "annual_financial_fingerprint": candidate.source.annualFingerprint, "eod_price_evidence_id": candidate.source.priceEvidenceID, "eod_price_fingerprint": candidate.source.priceFingerprint, "annual_available_at": candidate.annualAvailableAt.UTC().Format(time.RFC3339Nano)})
		result, err := tx.ExecContext(ctx, `INSERT INTO subscriber_global_marketops_evidence_records
 (global_evidence_id,evidence_run_id,global_asset_id,session_date,evidence_kind,algorithm_id,algorithm_version,quality_state,source_system,source_event_id,source_run_id,evidence_fingerprint,validation_contract_ref,immutable_baseline_ref,payload,provenance,observed_at)
 VALUES ($1,$2,$3,$4,'valuation',$5,$6,$7,'marketops',$8,$9,$10,$11,$12,$13::jsonb,$14::jsonb,now())
 ON CONFLICT (global_asset_id,session_date,evidence_kind,algorithm_id,algorithm_version,evidence_fingerprint) DO NOTHING`,
			"subglobalannualvalrec-"+digest(candidate.source.globalAssetID + "\x1f" + algorithmID + "\x1f" + fingerprint)[:24], runID, candidate.source.globalAssetID, observation,
			algorithmID, valuation.AnnualModelVersion, quality, "annual-v4:"+candidate.source.symbol+":"+metric+":"+observation.Format("2006-01-02"), candidate.source.annualEvidenceID, fingerprint, annualValuationContract, annualBaseline, string(payload), string(provenance))
		if err != nil {
			return 0, fmt.Errorf("append %s %s result: %w", candidate.source.symbol, metric, err)
		}
		count, _ := result.RowsAffected()
		inserted += int(count)
		_ = score
		_ = fair
		_ = classification
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return inserted, nil
}

func outputPayload(candidate resultCandidate, metric string) ([]byte, float64, float64, string) {
	score, fair, classification := candidate.result.VCScore, candidate.result.VCFairValue, candidate.result.VCClassification
	if metric == "dosm" {
		score, fair, classification = candidate.result.DOSMScore, candidate.result.DOSMFairValue, candidate.result.DOSMClassification
	}
	payload, _ := json.Marshal(map[string]any{
		"snapshot_id": "annual-v4:" + candidate.source.annualEvidenceID,
		"score":       score, "fair_value": fair, "classification": classification,
		"confidence": candidate.result.Confidence, "confidence_label": candidate.result.ConfidenceLabel,
		"evaluation_status": candidate.result.Status, "eligible": candidate.result.Eligible,
		"data_profile": candidate.result.DataProfile, "result_json": candidate.result,
	})
	return payload, score, fair, classification
}

func outputFingerprint(candidate resultCandidate, metric string) string {
	payload, _, _, _ := outputPayload(candidate, metric)
	return digest(candidate.source.globalAssetID + "\x1f" + string(payload))
}
func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

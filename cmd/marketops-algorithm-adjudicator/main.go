package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/lukebabs/signalops/internal/config"
	"github.com/lukebabs/signalops/internal/storage"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
	"os"
	"strings"
	"time"
)

const version = "marketops_algorithm_adjudicator.v1"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	var tenant, correlation string
	flag.StringVar(&tenant, "tenant-id", "tenant-local", "")
	flag.StringVar(&correlation, "correlation-id", "", "")
	flag.Parse()
	if correlation == "" {
		return fmt.Errorf("correlation-id is required")
	}
	cfg := config.Load()
	repo, err := postgresstorage.OpenWithTemporal(context.Background(), cfg.DatabaseURL, cfg.TemporalDatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	ctx := context.Background()
	evals, err := repo.ListMarketOpsHypothesisEvaluations(ctx, storage.MarketOpsHypothesisEvaluationFilter{TenantID: tenant, Triggered: boolPtr(true), Limit: 500})
	if err != nil {
		return err
	}
	results, err := repo.ListAlgorithmResults(ctx, storage.AlgorithmResultFilter{TenantID: tenant, CorrelationID: correlation, Limit: 1000})
	if err != nil {
		return err
	}
	definitions := map[string]string{}
	defs, err := repo.ListMarketOpsHypothesisDefinitions(ctx, storage.MarketOpsHypothesisDefinitionFilter{TenantID: tenant, Limit: 100})
	if err != nil {
		return err
	}
	for _, d := range defs {
		definitions[d.HypothesisKey+"|"+d.HypothesisVersion] = d.Direction
	}
	wrote := 0
	for _, ev := range evals {
		for _, ar := range results {
			if !platformID(ar.AlgorithmID) {
				continue
			}
			payload := map[string]any{}
			_ = json.Unmarshal(ar.ResultPayloadJSON, &payload)
			if strings.ToUpper(stringValue(payload["symbol"])) != strings.ToUpper(ev.Symbol) {
				continue
			}
			observation := stringValue(payload["observation_time"])
			if observation == "" {
				continue
			}
			t, err := time.Parse(time.RFC3339Nano, observation)
			if err != nil || t.UTC().Format("2006-01-02") != ev.SessionDate.UTC().Format("2006-01-02") {
				continue
			}
			verdict, explanation := adjudicate(definitions[ev.HypothesisKey+"|"+ev.HypothesisVersion], stringValue(payload["direction"]), ar, payload)
			body, _ := json.Marshal(explanation)
			rec := storage.MarketOpsAlgorithmAdjudicationRecord{AdjudicationID: stable("madj", tenant, ev.EvaluationID, ar.AlgorithmResultID, version), TenantID: tenant, HypothesisEvaluationID: ev.EvaluationID, AlgorithmResultID: ar.AlgorithmResultID, HypothesisKey: ev.HypothesisKey, HypothesisVersion: ev.HypothesisVersion, Symbol: ev.Symbol, SessionDate: ev.SessionDate, Verdict: verdict, Confidence: ar.Confidence, ExplanationJSON: body, CorrelationID: correlation, AdjudicatorVersion: version}
			if err := repo.UpsertMarketOpsAlgorithmAdjudication(ctx, rec); err != nil {
				return err
			}
			wrote++
		}
	}
	fmt.Printf("adjudications=%d\n", wrote)
	return nil
}
func boolPtr(v bool) *bool { return &v }
func platformID(v string) bool {
	return v == "signalops.algorithms.river_anomaly_v1" || v == "signalops.algorithms.ruptures_change_point_v1" || v == "signalops.algorithms.statsmodels_forecast_v1"
}
func stringValue(v any) string {
	s, ok := v.(string)
	if ok {
		return s
	}
	return ""
}
func adjudicate(hyp, observed string, ar storage.AlgorithmResultRecord, p map[string]any) (string, map[string]any) {
	expected := ""
	if hyp == "upside" {
		expected = "up"
	}
	if hyp == "downside" {
		expected = "down"
	}
	verdict := "inconclusive"
	if (ar.Severity == "medium" || ar.Severity == "high" || ar.Severity == "critical") && expected != "" && observed != "" {
		if expected == observed {
			verdict = "confirmed"
		} else {
			verdict = "contradicted"
		}
	}
	return verdict, map[string]any{"algorithm_id": ar.AlgorithmID, "result_type": ar.ResultType, "score": ar.Score, "hypothesis_direction": hyp, "observed_direction": observed, "rule": "directional platform evidence only; no hypothesis mutation"}
}
func stable(prefix string, parts ...string) string {
	h := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return prefix + "_" + hex.EncodeToString(h[:])[:24]
}

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/config"
	"github.com/lukebabs/signalops/internal/marketops/opportunities"
	"github.com/lukebabs/signalops/internal/storage"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
)

type repository interface {
	ListMarketOpsHypothesisDefinitions(context.Context, storage.MarketOpsHypothesisDefinitionFilter) ([]storage.MarketOpsHypothesisDefinitionRecord, error)
	ListMarketOpsHypothesisEvaluations(context.Context, storage.MarketOpsHypothesisEvaluationFilter) ([]storage.MarketOpsHypothesisEvaluationRecord, error)
	UpsertMarketOpsOpportunity(context.Context, storage.MarketOpsOpportunityRecord) error
	ExpireMarketOpsConvergenceOpportunities(context.Context, string, string, time.Time) (int, error)
}

type cliConfig struct {
	TenantID, Symbol, RunID  string
	SessionStart, SessionEnd time.Time
	MaxSessions              int
	DryRun                   bool
	CohortRunID              string
}

type metrics struct {
	RunID             string         `json:"run_id"`
	TenantID          string         `json:"tenant_id"`
	Symbol            string         `json:"symbol"`
	Evaluations       int            `json:"evaluations"`
	Triggered         int            `json:"triggered"`
	Opportunities     int            `json:"opportunities"`
	Emerging          int            `json:"emerging"`
	Active            int            `json:"active"`
	OverlapSuppressed int            `json:"overlap_suppressed"`
	ConflictLinks     int            `json:"conflict_links"`
	Expired           int            `json:"expired"`
	SkippedReasons    map[string]int `json:"skipped_reasons"`
	DryRun            bool           `json:"dry_run"`
	StartedAt         string         `json:"started_at"`
	CompletedAt       string         `json:"completed_at"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("marketops opportunity builder failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	app := config.Load()
	if strings.TrimSpace(app.DatabaseURL) == "" {
		return errors.New("SIGNALOPS_DATABASE_URL is required")
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	ctx := context.Background()
	repo, err := postgresstorage.Open(ctx, app.DatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	result, err := build(ctx, repo, cfg)
	if err != nil {
		return err
	}
	encoded, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(encoded))
	logger.Info("marketops opportunity builder completed", "run_id", result.RunID, "opportunities", result.Opportunities, "dry_run", result.DryRun)
	return nil
}

func build(ctx context.Context, repo repository, cfg cliConfig) (metrics, error) {
	if err := cfg.validate(); err != nil {
		return metrics{}, err
	}
	started := time.Now().UTC()
	result := metrics{RunID: cfg.RunID, TenantID: cfg.TenantID, Symbol: cfg.Symbol, SkippedReasons: map[string]int{}, DryRun: cfg.DryRun, StartedAt: started.Format(time.RFC3339Nano)}
	definitions, err := repo.ListMarketOpsHypothesisDefinitions(ctx, storage.MarketOpsHypothesisDefinitionFilter{TenantID: cfg.TenantID, Limit: 100})
	if err != nil {
		return result, err
	}
	evaluations, err := repo.ListMarketOpsHypothesisEvaluations(ctx, storage.MarketOpsHypothesisEvaluationFilter{
		TenantID: cfg.TenantID, AppID: "marketops", Symbol: cfg.Symbol,
		SessionStart: cfg.SessionStart, SessionEnd: cfg.SessionEnd, Limit: cfg.MaxSessions * 4,
	})
	if err != nil {
		return result, err
	}
	built, err := opportunities.Build(cfg.RunID, definitions, evaluations)
	if err != nil {
		return result, err
	}
	if active, ok := repo.(activeConvergenceReader); ok {
		inputs, err := activeContributions(ctx, active, cfg)
		if err != nil {
			return result, err
		}
		convergence, err := opportunities.BuildConvergence(cfg.RunID, inputs)
		if err != nil {
			return result, err
		}
		built.Opportunities = append(built.Opportunities, convergence.Opportunities...)
		for reason, count := range convergence.SkippedReasons {
			built.SkippedReasons[reason] += count
		}
	}
	result.Evaluations, result.Triggered = built.Evaluations, built.Triggered
	result.Opportunities, result.OverlapSuppressed, result.ConflictLinks = len(built.Opportunities), built.OverlapSuppressed, built.ConflictLinks
	result.SkippedReasons = built.SkippedReasons
	if !cfg.DryRun {
		expired, err := repo.ExpireMarketOpsConvergenceOpportunities(ctx, cfg.TenantID, cfg.Symbol, cfg.SessionEnd)
		if err != nil {
			return result, err
		}
		result.Expired = expired
	}
	for _, opportunity := range built.Opportunities {
		switch opportunity.LifecycleStatus {
		case storage.MarketOpsOpportunityEmerging:
			result.Emerging++
		case storage.MarketOpsOpportunityActive:
			result.Active++
		}
		if !cfg.DryRun {
			if err := repo.UpsertMarketOpsOpportunity(ctx, opportunity); err != nil {
				return result, err
			}
		}
	}
	result.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
	return result, nil
}

func loadConfig() (cliConfig, error) {
	now := time.Now().UTC()
	var startValue, endValue string
	cfg := cliConfig{}
	flag.StringVar(&cfg.TenantID, "tenant-id", "tenant-local", "tenant id")
	flag.StringVar(&cfg.Symbol, "symbol", "AAPL", "asset symbol")
	flag.StringVar(&startValue, "session-start", now.AddDate(-1, 0, 0).Format("2006-01-02"), "inclusive session start")
	flag.StringVar(&endValue, "session-end", now.Format("2006-01-02"), "inclusive session end")
	flag.IntVar(&cfg.MaxSessions, "max-sessions", 50, "maximum source sessions (1-50)")
	flag.StringVar(&cfg.RunID, "run-id", "", "build run id")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "build without writes")
	flag.StringVar(&cfg.CohortRunID, "cohort-run-id", "", "bounded G148 cohort run marker")
	flag.Parse()
	var err error
	if cfg.SessionStart, err = time.Parse("2006-01-02", strings.TrimSpace(startValue)); err != nil {
		return cfg, err
	}
	if cfg.SessionEnd, err = time.Parse("2006-01-02", strings.TrimSpace(endValue)); err != nil {
		return cfg, err
	}
	cfg.TenantID, cfg.Symbol, cfg.RunID = strings.TrimSpace(cfg.TenantID), strings.ToUpper(strings.TrimSpace(cfg.Symbol)), strings.TrimSpace(cfg.RunID)
	cfg.CohortRunID = strings.TrimSpace(cfg.CohortRunID)
	if cfg.RunID == "" {
		cfg.RunID = "oppbuild_" + randomHex(12)
	}
	return cfg, nil
}

func (cfg cliConfig) validate() error {
	if cfg.TenantID == "" || cfg.RunID == "" {
		return errors.New("tenant-id and run-id are required")
	}
	if cfg.SessionStart.IsZero() || cfg.SessionEnd.IsZero() || cfg.SessionEnd.Before(cfg.SessionStart) {
		return errors.New("session-end must not precede session-start")
	}
	if cfg.MaxSessions <= 0 || cfg.MaxSessions > 50 {
		return errors.New("max-sessions must be between 1 and 50")
	}
	return nil
}

func randomHex(length int) string {
	value := make([]byte, length)
	_, _ = rand.Read(value)
	return hex.EncodeToString(value)
}

type activeConvergenceReader interface {
	ListMarketOpsRiskRewardSnapshots(context.Context, storage.MarketOpsRiskRewardSnapshotFilter) ([]storage.MarketOpsRiskRewardSnapshotRecord, error)
	ListMarketOpsValuationResults(context.Context, storage.MarketOpsValuationFilter) ([]storage.MarketOpsValuationResultRecord, error)
	ListMarketOpsOptionsDistributions(context.Context, storage.MarketOpsOptionsDistributionFilter) ([]storage.MarketOpsOptionsDistributionRecord, error)
}

func activeContributions(ctx context.Context, repo activeConvergenceReader, cfg cliConfig) ([]opportunities.ConvergenceContribution, error) {
	risk, err := repo.ListMarketOpsRiskRewardSnapshots(ctx, storage.MarketOpsRiskRewardSnapshotFilter{TenantID: cfg.TenantID, Symbol: cfg.Symbol, SessionStart: cfg.SessionStart, SessionEnd: cfg.SessionEnd, EligibleOnly: true, Limit: 5000})
	if err != nil {
		return nil, err
	}
	values, err := repo.ListMarketOpsValuationResults(ctx, storage.MarketOpsValuationFilter{TenantID: cfg.TenantID, Symbol: cfg.Symbol, Limit: 10000})
	if err != nil {
		return nil, err
	}
	out := []opportunities.ConvergenceContribution{}
	for _, row := range risk {
		direction := ""
		if row.TechnicalDirection == "bullish" {
			direction = "upside"
		} else if row.TechnicalDirection == "bearish" {
			direction = "downside"
		}
		if direction != "" {
			out = append(out, opportunities.ConvergenceContribution{TenantID: row.TenantID, AssetID: "ticker:" + row.Symbol, Symbol: row.Symbol, Source: "risk_reward", Direction: direction, SessionDate: row.SessionDate, Strength: abs(row.TechnicalScore) / 100, EvidenceIDs: []string{row.AlgorithmResultID}})
		}
	}
	for _, row := range values {
		if row.AlgorithmID != "signalops.algorithms.eroc_v6" && row.AlgorithmID != "signalops.algorithms.tactical_market_posture_v1" {
			continue
		}
		payload := map[string]any{}
		_ = json.Unmarshal(row.ResultJSON, &payload)
		direction := ""
		source := ""
		if row.AlgorithmID == "signalops.algorithms.eroc_v6" {
			if trace, ok := payload["trace"].(map[string]any); ok && trace["reversal_candidate"] == true {
				if trace["direction"] == "BULLISH" {
					direction = "upside"
				} else if trace["direction"] == "BEARISH" {
					direction = "downside"
				}
				source = "exhaustive_reversal"
			}
		} else if row.Classification == "constructive" {
			direction = "upside"
			source = "tactical_posture"
		} else if row.Classification == "caution" {
			direction = "downside"
			source = "tactical_posture"
		}
		if direction != "" {
			out = append(out, opportunities.ConvergenceContribution{TenantID: row.TenantID, AssetID: "ticker:" + row.Symbol, Symbol: row.Symbol, Source: source, Direction: direction, SessionDate: row.SessionDate, Strength: abs(row.Score) / 100, EvidenceIDs: []string{row.ResultID}})
		}
	}
	seenOptions := map[string]bool{}
	for _, value := range out {
		if seenOptions[value.Symbol] {
			continue
		}
		seenOptions[value.Symbol] = true
		rows, err := repo.ListMarketOpsOptionsDistributions(ctx, storage.MarketOpsOptionsDistributionFilter{TenantID: cfg.TenantID, Symbol: value.Symbol, WindowName: "10_trade_days", Limit: 1})
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			continue
		}
		item := rows[0]
		if item.CallPutVolumeRatio < .30 {
			out = append(out, opportunities.ConvergenceContribution{TenantID: item.TenantID, AssetID: "ticker:" + item.Symbol, Symbol: item.Symbol, Source: "options_flow", Direction: "upside", SessionDate: item.TradeDate, Strength: clamp01((.30 - item.CallPutVolumeRatio) / .30), EvidenceIDs: []string{item.SourceID}})
		}
		if item.CallPutVolumeRatio > 1.20 {
			out = append(out, opportunities.ConvergenceContribution{TenantID: item.TenantID, AssetID: "ticker:" + item.Symbol, Symbol: item.Symbol, Source: "options_flow", Direction: "downside", SessionDate: item.TradeDate, Strength: clamp01((item.CallPutVolumeRatio - 1.20) / 1.20), EvidenceIDs: []string{item.SourceID}})
		}
	}
	return out, nil
}
func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

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
	result.Evaluations, result.Triggered = built.Evaluations, built.Triggered
	result.Opportunities, result.OverlapSuppressed, result.ConflictLinks = len(built.Opportunities), built.OverlapSuppressed, built.ConflictLinks
	result.SkippedReasons = built.SkippedReasons
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
	flag.StringVar(&cfg.Symbol, "symbol", "AAPL", "G139 asset symbol; AAPL only")
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
	if cfg.Symbol != "AAPL" && cfg.CohortRunID == "" {
		return errors.New("G139 is intentionally bounded to AAPL")
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

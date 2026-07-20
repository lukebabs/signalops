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
	marketopsproposals "github.com/lukebabs/signalops/internal/marketops/proposals"
	"github.com/lukebabs/signalops/internal/storage"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
)

type repository interface {
	ListMarketOpsHypothesisDefinitions(context.Context, storage.MarketOpsHypothesisDefinitionFilter) ([]storage.MarketOpsHypothesisDefinitionRecord, error)
	ListMarketOpsHypothesisEvaluations(context.Context, storage.MarketOpsHypothesisEvaluationFilter) ([]storage.MarketOpsHypothesisEvaluationRecord, error)
	InsertAlgorithmSignalProposal(context.Context, storage.AlgorithmSignalProposalRecord) (bool, error)
}

type cliConfig struct {
	TenantID, Symbol, RunID, CreatedBy string
	SessionStart, SessionEnd           time.Time
	MaxSessions                        int
	DryRun                             bool
	CohortRunID                        string
}

type metrics struct {
	RunID    string         `json:"run_id"`
	TenantID string         `json:"tenant_id"`
	Symbol   string         `json:"symbol"`
	Scanned  int            `json:"scanned"`
	Built    int            `json:"built"`
	Inserted int            `json:"inserted"`
	Skipped  map[string]int `json:"skipped_reasons"`
	DryRun   bool           `json:"dry_run"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("marketops hypothesis proposal generator failed", "error", err)
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
	repo, err := postgresstorage.Open(context.Background(), app.DatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	result, err := generate(context.Background(), repo, cfg)
	if err != nil {
		return err
	}
	encoded, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(encoded))
	logger.Info("marketops hypothesis proposal generation completed", "run_id", result.RunID, "built", result.Built, "inserted", result.Inserted, "dry_run", result.DryRun)
	return nil
}

func generate(ctx context.Context, repo repository, cfg cliConfig) (metrics, error) {
	if err := cfg.validate(); err != nil {
		return metrics{}, err
	}
	definitions, err := repo.ListMarketOpsHypothesisDefinitions(ctx, storage.MarketOpsHypothesisDefinitionFilter{TenantID: cfg.TenantID, Limit: 100})
	if err != nil {
		return metrics{}, err
	}
	evaluations, err := repo.ListMarketOpsHypothesisEvaluations(ctx, storage.MarketOpsHypothesisEvaluationFilter{
		TenantID: cfg.TenantID, AppID: "marketops", Symbol: cfg.Symbol,
		SessionStart: cfg.SessionStart, SessionEnd: cfg.SessionEnd, Limit: cfg.MaxSessions * 8,
	})
	if err != nil {
		return metrics{}, err
	}
	built, err := marketopsproposals.Build(cfg.RunID, cfg.CreatedBy, definitions, evaluations)
	if err != nil {
		return metrics{}, err
	}
	result := metrics{RunID: cfg.RunID, TenantID: cfg.TenantID, Symbol: cfg.Symbol, Scanned: built.Scanned, Built: len(built.Proposals), Skipped: built.SkippedReasons, DryRun: cfg.DryRun}
	for _, proposal := range built.Proposals {
		if cfg.DryRun {
			continue
		}
		inserted, err := repo.InsertAlgorithmSignalProposal(ctx, proposal)
		if err != nil {
			return result, err
		}
		if inserted {
			result.Inserted++
		}
	}
	return result, nil
}

func loadConfig() (cliConfig, error) {
	now := time.Now().UTC()
	var startValue, endValue string
	cfg := cliConfig{}
	flag.StringVar(&cfg.TenantID, "tenant-id", "tenant-local", "tenant id")
	flag.StringVar(&cfg.Symbol, "symbol", "AAPL", "G146 asset symbol; AAPL only")
	flag.StringVar(&startValue, "session-start", now.AddDate(-1, 0, 0).Format("2006-01-02"), "inclusive session start")
	flag.StringVar(&endValue, "session-end", now.Format("2006-01-02"), "inclusive session end")
	flag.IntVar(&cfg.MaxSessions, "max-sessions", 50, "maximum source sessions (1-50)")
	flag.StringVar(&cfg.RunID, "run-id", "", "proposal run id")
	flag.StringVar(&cfg.CreatedBy, "created-by", "marketops-proposal-generator", "proposal creator")
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
	cfg.TenantID, cfg.Symbol = strings.TrimSpace(cfg.TenantID), strings.ToUpper(strings.TrimSpace(cfg.Symbol))
	cfg.RunID, cfg.CreatedBy = strings.TrimSpace(cfg.RunID), strings.TrimSpace(cfg.CreatedBy)
	cfg.CohortRunID = strings.TrimSpace(cfg.CohortRunID)
	if cfg.RunID == "" {
		cfg.RunID = "hypprop_" + randomHex(12)
	}
	return cfg, nil
}

func (cfg cliConfig) validate() error {
	if cfg.TenantID == "" || cfg.RunID == "" || cfg.CreatedBy == "" {
		return errors.New("tenant-id, run-id, and created-by are required")
	}
	if cfg.Symbol != "AAPL" && cfg.CohortRunID == "" {
		return errors.New("G146 is intentionally bounded to AAPL")
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

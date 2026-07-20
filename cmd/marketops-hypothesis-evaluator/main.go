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
	"github.com/lukebabs/signalops/internal/marketops/hypotheses"
	"github.com/lukebabs/signalops/internal/storage"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
)

type repository interface {
	ListMarketOpsMarketStates(context.Context, storage.MarketOpsMarketStateFilter) ([]storage.MarketOpsMarketStateRecord, error)
	ListMarketOpsFeatureObservations(context.Context, storage.MarketOpsFeatureObservationFilter) ([]storage.MarketOpsFeatureObservationRecord, error)
	ListMarketOpsStateTransitions(context.Context, storage.MarketOpsStateTransitionFilter) ([]storage.MarketOpsStateTransitionRecord, error)
	ListMarketOpsEvidence(context.Context, storage.MarketOpsEvidenceFilter) ([]storage.MarketOpsEvidenceRecord, error)
	UpsertMarketOpsHypothesisDefinition(context.Context, storage.MarketOpsHypothesisDefinitionRecord) error
	UpsertMarketOpsHypothesisEvaluation(context.Context, storage.MarketOpsHypothesisEvaluationRecord) error
}

type cliConfig struct {
	TenantID, Symbol, RunID  string
	SessionStart, SessionEnd time.Time
	MaxSessions              int
	DryRun                   bool
}

type metrics struct {
	RunID        string         `json:"run_id"`
	TenantID     string         `json:"tenant_id"`
	Symbol       string         `json:"symbol"`
	States       int            `json:"states"`
	Definitions  int            `json:"definitions"`
	Evaluations  int            `json:"evaluations"`
	Eligible     int            `json:"eligible"`
	Triggered    int            `json:"triggered"`
	Rejected     int            `json:"rejected"`
	ReasonCounts map[string]int `json:"reason_counts"`
	DryRun       bool           `json:"dry_run"`
	StartedAt    string         `json:"started_at"`
	CompletedAt  string         `json:"completed_at"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("marketops hypothesis evaluator failed", "error", err)
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
	result, err := evaluate(ctx, repo, cfg)
	if err != nil {
		return err
	}
	encoded, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(encoded))
	logger.Info("marketops hypothesis evaluator completed", "run_id", result.RunID, "evaluations", result.Evaluations, "triggered", result.Triggered, "dry_run", result.DryRun)
	return nil
}

func evaluate(ctx context.Context, repo repository, cfg cliConfig) (metrics, error) {
	if err := cfg.validate(); err != nil {
		return metrics{}, err
	}
	started := time.Now().UTC()
	result := metrics{RunID: cfg.RunID, TenantID: cfg.TenantID, Symbol: cfg.Symbol, ReasonCounts: map[string]int{}, DryRun: cfg.DryRun, StartedAt: started.Format(time.RFC3339Nano)}
	definitions := hypotheses.ResearchDefinitions(cfg.TenantID)
	result.Definitions = len(definitions)
	states, err := repo.ListMarketOpsMarketStates(ctx, storage.MarketOpsMarketStateFilter{TenantID: cfg.TenantID, AppID: "marketops", Symbol: cfg.Symbol, SessionStart: cfg.SessionStart, SessionEnd: cfg.SessionEnd, Limit: cfg.MaxSessions})
	if err != nil {
		return result, err
	}
	result.States = len(states)
	if !cfg.DryRun {
		for _, definition := range definitions {
			if err := repo.UpsertMarketOpsHypothesisDefinition(ctx, definition); err != nil {
				return result, err
			}
		}
	}
	for _, state := range states {
		observations, err := repo.ListMarketOpsFeatureObservations(ctx, storage.MarketOpsFeatureObservationFilter{TenantID: cfg.TenantID, AppID: "marketops", FeatureObservationIDs: state.FeatureObservationIDs, Limit: len(state.FeatureObservationIDs)})
		if err != nil {
			return result, err
		}
		transitions, err := repo.ListMarketOpsStateTransitions(ctx, storage.MarketOpsStateTransitionFilter{TenantID: cfg.TenantID, AppID: "marketops", CurrentStateID: state.MarketStateID, Limit: 200})
		if err != nil {
			return result, err
		}
		evidence, err := repo.ListMarketOpsEvidence(ctx, storage.MarketOpsEvidenceFilter{TenantID: cfg.TenantID, AppID: "marketops", Symbol: state.Symbol, SessionStart: state.SessionDate, SessionEnd: state.SessionDate, Limit: 200})
		if err != nil {
			return result, err
		}
		for _, definition := range definitions {
			record, err := hypotheses.Evaluate(cfg.RunID, definition, state, observations, transitions, evidence)
			if err != nil {
				return result, err
			}
			result.Evaluations++
			if record.Eligible {
				result.Eligible++
			} else {
				result.Rejected++
			}
			if record.Triggered {
				result.Triggered++
			}
			for _, reason := range record.ReasonCodes {
				result.ReasonCounts[reason]++
			}
			if !cfg.DryRun {
				if err := repo.UpsertMarketOpsHypothesisEvaluation(ctx, record); err != nil {
					return result, err
				}
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
	flag.StringVar(&cfg.Symbol, "symbol", "AAPL", "G138 asset symbol; AAPL only")
	flag.StringVar(&startValue, "session-start", now.AddDate(-1, 0, 0).Format("2006-01-02"), "inclusive session start")
	flag.StringVar(&endValue, "session-end", now.Format("2006-01-02"), "inclusive session end")
	flag.IntVar(&cfg.MaxSessions, "max-sessions", 100, "maximum states")
	flag.StringVar(&cfg.RunID, "run-id", "", "evaluation run id")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "evaluate without writes")
	flag.Parse()
	var err error
	if cfg.SessionStart, err = time.Parse("2006-01-02", strings.TrimSpace(startValue)); err != nil {
		return cfg, err
	}
	if cfg.SessionEnd, err = time.Parse("2006-01-02", strings.TrimSpace(endValue)); err != nil {
		return cfg, err
	}
	cfg.TenantID = strings.TrimSpace(cfg.TenantID)
	cfg.Symbol = strings.ToUpper(strings.TrimSpace(cfg.Symbol))
	cfg.RunID = strings.TrimSpace(cfg.RunID)
	if cfg.RunID == "" {
		cfg.RunID = "hypeval_" + randomHex(12)
	}
	return cfg, nil
}

func (cfg cliConfig) validate() error {
	if cfg.TenantID == "" || cfg.RunID == "" {
		return errors.New("tenant-id and run-id are required")
	}
	if cfg.Symbol != "AAPL" {
		return errors.New("G138 is intentionally bounded to AAPL")
	}
	if cfg.SessionStart.IsZero() || cfg.SessionEnd.IsZero() || cfg.SessionEnd.Before(cfg.SessionStart) {
		return errors.New("session-end must not precede session-start")
	}
	if cfg.MaxSessions <= 0 || cfg.MaxSessions > 1000 {
		return errors.New("max-sessions must be between 1 and 1000")
	}
	return nil
}
func randomHex(n int) string {
	value := make([]byte, n)
	_, _ = rand.Read(value)
	return hex.EncodeToString(value)
}

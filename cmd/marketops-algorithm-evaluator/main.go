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
	"github.com/lukebabs/signalops/internal/marketops/algorithmevaluation"
	"github.com/lukebabs/signalops/internal/storage"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
)

type cliConfig struct {
	TenantID, UniverseGroup, RunID, Feature, RequestedBy string
	Symbols, AlgorithmIDs, Modes                         []string
	WindowStart, WindowEnd, AsOf                         time.Time
	MaxSymbols, LookbackSessions, MinSamples             int
	Threshold                                            float64
	DryRun                                               bool
	RegistryEnforcement                                  bool
}

type repository interface {
	algorithmevaluation.Repository
	ListMarketOpsAssets(context.Context, string, string, bool, int) ([]storage.MarketOpsAssetRecord, error)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("marketops algorithm evaluator failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	app := config.Load()
	if strings.TrimSpace(app.DatabaseURL) == "" || strings.TrimSpace(app.TemporalDatabaseURL) == "" {
		return errors.New("SIGNALOPS_DATABASE_URL and SIGNALOPS_TEMPORAL_DATABASE_URL are required")
	}
	cfg, err := loadConfig(os.Args[1:])
	if err != nil {
		return err
	}
	ctx := context.Background()
	repo, err := postgresstorage.OpenWithTemporal(ctx, app.DatabaseURL, app.TemporalDatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	if len(cfg.Symbols) == 0 {
		assets, err := repo.ListMarketOpsAssets(ctx, cfg.TenantID, cfg.UniverseGroup, true, cfg.MaxSymbols+1)
		if err != nil {
			return err
		}
		if len(assets) > cfg.MaxSymbols {
			return fmt.Errorf("resolved universe exceeds max-symbols %d", cfg.MaxSymbols)
		}
		for _, asset := range assets {
			cfg.Symbols = append(cfg.Symbols, strings.ToUpper(asset.Ticker))
		}
	}
	if len(cfg.Symbols) == 0 {
		return errors.New("evaluation resolved no symbols")
	}
	result, err := algorithmevaluation.Run(ctx, repo, algorithmevaluation.Config{RunID: cfg.RunID, TenantID: cfg.TenantID, UniverseGroup: cfg.UniverseGroup, Symbols: cfg.Symbols, AlgorithmIDs: cfg.AlgorithmIDs, Modes: cfg.Modes, WindowStart: cfg.WindowStart, WindowEnd: cfg.WindowEnd, AsOf: cfg.AsOf, Feature: cfg.Feature, LookbackSessions: cfg.LookbackSessions, MinSamples: cfg.MinSamples, Threshold: cfg.Threshold, RequestedBy: cfg.RequestedBy, DryRun: cfg.DryRun, RegistryEnforcement: cfg.RegistryEnforcement})
	if err != nil {
		return err
	}
	encoded, _ := json.MarshalIndent(map[string]any{"evaluation_run": result.Run, "metrics": result.Metrics}, "", "  ")
	fmt.Println(string(encoded))
	logger.Info("marketops algorithm evaluation completed", "run_id", result.Run.RunID, "status", result.Run.Status, "results", result.Metrics.EvaluationResults, "matured", result.Metrics.Matured, "dry_run", cfg.DryRun)
	return nil
}

func loadConfig(args []string) (cliConfig, error) {
	now := time.Now().UTC()
	cfg := cliConfig{}
	var start, end, asOf, symbols, algorithms, modes string
	flags := flag.NewFlagSet("marketops-algorithm-evaluator", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.StringVar(&cfg.TenantID, "tenant-id", "tenant-local", "tenant id")
	flags.StringVar(&cfg.UniverseGroup, "universe-group", "top50_megacap", "active universe used when symbols are omitted")
	flags.StringVar(&symbols, "symbols", "", "comma-separated symbols; empty resolves active universe")
	flags.IntVar(&cfg.MaxSymbols, "max-symbols", 50, "maximum symbols to resolve or evaluate (1-50)")
	flags.StringVar(&algorithms, "algorithm-ids", strings.Join(algorithmevaluation.DefaultAlgorithmIDs, ","), "comma-separated seeded algorithm ids")
	flags.StringVar(&modes, "modes", "retrospective,walk_forward", "comma-separated evaluation modes")
	flags.StringVar(&start, "window-start", now.AddDate(0, 0, -400).Format("2006-01-02"), "inclusive evaluation session start")
	flags.StringVar(&end, "window-end", now.AddDate(0, 0, 1).Format("2006-01-02"), "exclusive evaluation session end")
	flags.StringVar(&asOf, "as-of", now.Format("2006-01-02"), "point-in-time outcome cutoff")
	flags.StringVar(&cfg.Feature, "feature", "daily_return_pct", "numeric feature for scalar adapters")
	flags.IntVar(&cfg.LookbackSessions, "lookback-sessions", 60, "point-in-time scalar lookback (20-400)")
	flags.IntVar(&cfg.MinSamples, "min-samples", 20, "minimum prior samples (2-lookback)")
	flags.Float64Var(&cfg.Threshold, "threshold", .02, "absolute forward-return event threshold (0-1)")
	flags.StringVar(&cfg.RequestedBy, "requested-by", "operator-local", "requesting operator")
	flags.StringVar(&cfg.RunID, "run-id", "", "stable evaluation run id")
	flags.BoolVar(&cfg.DryRun, "dry-run", false, "calculate without isolated ledger writes")
	flags.BoolVar(&cfg.RegistryEnforcement, "registry-enforcement", envBoolOrDefault("SIGNALOPS_PLATFORM_REGISTRY_ENFORCEMENT", false), "require active platform-definition provenance")
	if err := flags.Parse(args); err != nil {
		return cliConfig{}, err
	}
	var err error
	cfg.WindowStart, err = time.Parse("2006-01-02", strings.TrimSpace(start))
	if err != nil {
		return cfg, fmt.Errorf("window-start: %w", err)
	}
	cfg.WindowEnd, err = time.Parse("2006-01-02", strings.TrimSpace(end))
	if err != nil {
		return cfg, fmt.Errorf("window-end: %w", err)
	}
	cfg.AsOf, err = time.Parse("2006-01-02", strings.TrimSpace(asOf))
	if err != nil {
		return cfg, fmt.Errorf("as-of: %w", err)
	}
	cfg.TenantID = strings.TrimSpace(cfg.TenantID)
	cfg.UniverseGroup = strings.TrimSpace(cfg.UniverseGroup)
	cfg.Feature = strings.TrimSpace(cfg.Feature)
	cfg.RequestedBy = strings.TrimSpace(cfg.RequestedBy)
	cfg.RunID = strings.TrimSpace(cfg.RunID)
	cfg.Symbols = parseList(symbols, true)
	cfg.AlgorithmIDs = parseList(algorithms, false)
	cfg.Modes = parseList(modes, false)
	if cfg.RunID == "" {
		cfg.RunID = "algeval_" + randomHex(12)
	}
	if cfg.MaxSymbols < 1 || cfg.MaxSymbols > 50 {
		return cfg, errors.New("max-symbols must be between 1 and 50")
	}
	if len(cfg.Symbols) > cfg.MaxSymbols {
		return cfg, fmt.Errorf("explicit symbols exceed max-symbols %d", cfg.MaxSymbols)
	}
	return cfg, nil
}

func parseList(value string, upper bool) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if upper {
			item = strings.ToUpper(item)
		}
		if item != "" && !seen[item] {
			seen[item] = true
			out = append(out, item)
		}
	}
	return out
}
func randomHex(length int) string {
	value := make([]byte, length)
	_, _ = rand.Read(value)
	return hex.EncodeToString(value)
}

func envBoolOrDefault(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes"
}

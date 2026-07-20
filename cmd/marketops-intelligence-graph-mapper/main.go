package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/config"
	"github.com/lukebabs/signalops/internal/marketops/graph"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "marketops intelligence graph mapper:", err)
		os.Exit(1)
	}
}

func run() error {
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
	result, err := graph.Map(ctx, repo, cfg)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func loadConfig() (graph.Config, error) {
	now := time.Now().UTC()
	var sessionStart, sessionEnd, sourceTypes string
	var write bool
	cfg := graph.Config{}
	flag.StringVar(&cfg.TenantID, "tenant-id", "", "tenant id (required)")
	flag.StringVar(&cfg.Symbol, "symbol", "AAPL", "asset symbol")
	flag.StringVar(&sessionStart, "session-start", now.AddDate(0, -1, 0).Format("2006-01-02"), "inclusive source session start (YYYY-MM-DD)")
	flag.StringVar(&sessionEnd, "session-end", now.Format("2006-01-02"), "inclusive source session end (YYYY-MM-DD)")
	flag.StringVar(&sourceTypes, "source-types", "market_state,state_transition,hypothesis_definition,hypothesis_evaluation,opportunity,outcome", "comma-separated persisted source types")
	flag.IntVar(&cfg.MaxSourceRecords, "max-source-records", 100, "maximum persisted source records (1-1000)")
	flag.IntVar(&cfg.MaxProposals, "max-proposals", 500, "maximum proposal candidates (1-5000)")
	flag.BoolVar(&cfg.DryRun, "dry-run", true, "calculate and report without writes")
	flag.BoolVar(&write, "write", false, "explicitly acknowledge proposal-ledger writes; never writes graph state")
	flag.Parse()

	var err error
	cfg.SessionStart, err = time.Parse("2006-01-02", strings.TrimSpace(sessionStart))
	if err != nil {
		return cfg, fmt.Errorf("parse session-start: %w", err)
	}
	cfg.SessionEnd, err = time.Parse("2006-01-02", strings.TrimSpace(sessionEnd))
	if err != nil {
		return cfg, fmt.Errorf("parse session-end: %w", err)
	}
	cfg.SourceTypes = strings.Split(sourceTypes, ",")
	if write {
		cfg.DryRun = false
	} else if !cfg.DryRun {
		return cfg, errors.New("--dry-run=false requires explicit --write acknowledgement")
	}
	return cfg, nil
}

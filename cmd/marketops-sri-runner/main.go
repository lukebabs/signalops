package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/lukebabs/signalops/internal/config"
	"github.com/lukebabs/signalops/internal/marketops/sri"
	postgres "github.com/lukebabs/signalops/internal/storage/postgres"
	"log/slog"
	"os"
	"strings"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(); err != nil {
		logger.Error("sri runner failed", "error", err)
		os.Exit(1)
	}
}
func run() error {
	app := config.Load()
	if app.DatabaseURL == "" || app.TemporalDatabaseURL == "" {
		return fmt.Errorf("SIGNALOPS_DATABASE_URL and SIGNALOPS_TEMPORAL_DATABASE_URL are required")
	}
	tenant := flag.String("tenant-id", "platform-global", "output data scope")
	inputTenant := flag.String("input-tenant-id", "tenant-local", "canonical EOD input scope")
	runID := flag.String("run-id", "sri_"+time.Now().UTC().Format("20060102T150405Z"), "run id")
	asOf := flag.String("as-of", time.Now().UTC().Format("2006-01-02"), "as of date")
	backfillSessions := flag.Int("backfill-sessions", 0, "materialize this many recent common SRI sessions (1-120)")
	flag.Parse()
	date, err := time.Parse("2006-01-02", strings.TrimSpace(*asOf))
	if err != nil {
		return err
	}
	repo, err := postgres.OpenWithTemporal(context.Background(), app.DatabaseURL, app.TemporalDatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	cfg := sri.Config{TenantID: strings.TrimSpace(*tenant), InputTenantID: strings.TrimSpace(*inputTenant), RunID: strings.TrimSpace(*runID), AsOf: date}
	if *backfillSessions > 0 {
		result, err := sri.RunRecentSessions(context.Background(), repo, cfg, *backfillSessions)
		if err != nil {
			return err
		}
		fmt.Printf("{\"sessions\":%d,\"first_session\":\"%s\",\"last_session\":\"%s\",\"snapshots\":%d,\"partial\":%d}\n", result.Sessions, result.FirstSession.Format("2006-01-02"), result.LastSession.Format("2006-01-02"), result.Snapshots, result.Partial)
		return nil
	}
	result, err := sri.Run(context.Background(), repo, cfg)
	if err != nil {
		return err
	}
	fmt.Printf("{\"segments\":%d,\"snapshots\":%d,\"partial\":%d}\n", result.Segments, result.Snapshots, result.Partial)
	return nil
}

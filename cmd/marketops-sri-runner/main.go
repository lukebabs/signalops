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
	tenant := flag.String("tenant-id", "tenant-local", "tenant")
	runID := flag.String("run-id", "sri_"+time.Now().UTC().Format("20060102T150405Z"), "run id")
	asOf := flag.String("as-of", time.Now().UTC().Format("2006-01-02"), "as of date")
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
	result, err := sri.Run(context.Background(), repo, sri.Config{TenantID: strings.TrimSpace(*tenant), RunID: strings.TrimSpace(*runID), AsOf: date})
	if err != nil {
		return err
	}
	fmt.Printf("{\"segments\":%d,\"snapshots\":%d,\"partial\":%d}\n", result.Segments, result.Snapshots, result.Partial)
	return nil
}

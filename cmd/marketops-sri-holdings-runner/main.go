package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/lukebabs/signalops/internal/adapters/marketdata/statestreet"
	"github.com/lukebabs/signalops/internal/config"
	"github.com/lukebabs/signalops/internal/marketops/sri"
	postgres "github.com/lukebabs/signalops/internal/storage/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(); err != nil {
		logger.Error("SRI holdings runner failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	app := config.Load()
	if app.DatabaseURL == "" {
		return fmt.Errorf("SIGNALOPS_DATABASE_URL is required")
	}
	tenant := flag.String("tenant-id", "tenant-local", "tenant")
	flag.Parse()
	repo, err := postgres.Open(context.Background(), app.DatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	result, err := sri.RefreshStateStreetHoldings(context.Background(), repo, statestreet.Client{}, strings.TrimSpace(*tenant))
	if err != nil {
		return err
	}
	fmt.Printf("{\"etfs\":%d,\"snapshots\":%d,\"holdings\":%d,\"unsupported_primary_etfs\":%d}\n", result.ETFs, result.Snapshots, result.Holdings, result.Unsupported)
	return nil
}

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/lukebabs/signalops/internal/api"
	"github.com/lukebabs/signalops/internal/config"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
)

func main() {
	tenantID := flag.String("tenant-id", "tenant-local", "tenant identifier")
	sessionDate := flag.String("session-date", "", "market session date (YYYY-MM-DD)")
	dryRun := flag.Bool("dry-run", false, "calculate without writing")
	flag.Parse()
	if *sessionDate == "" { log.Fatal("--session-date is required") }
	cfg := config.Load()
	if cfg.DatabaseURL == "" { log.Fatal("DATABASE_URL is required") }
	repo, err := postgresstorage.OpenWithTemporal(context.Background(), cfg.DatabaseURL, cfg.TemporalDatabaseURL)
	if err != nil { log.Fatal(err) }
	defer repo.Close()
	result, err := api.MaterializeSyncraticPostClose(context.Background(), repo, *tenantID, *sessionDate, *dryRun)
	if err != nil { log.Fatal(err) }
	fmt.Fprintf(os.Stdout, "syncratic post-close contexts=%d insights=%d queued=%d\n", result.MaterializedContextWindows, result.MaterializedInsights, len(result.QueuedJobIDs))
}

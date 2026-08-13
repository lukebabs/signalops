// subscriber-global-eod-canary-preparer records a bounded S4 canary cohort.
// It cannot call a provider, enqueue a task, alter scheduler configuration, or
// enable a production collection path.
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

	"github.com/lukebabs/signalops/internal/storage"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
	"github.com/lukebabs/signalops/internal/subscriber/eodcanary"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "subscriber global EOD canary preparation failed:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("subscriber-global-eod-canary-preparer", flag.ContinueOnError)
	databaseURL := flags.String("database-url", os.Getenv("SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL"), "dedicated Subscriber global-EOD PostgreSQL URL")
	planRunID := flags.String("plan-run-id", "", "required S2 shadow plan run id")
	sessionDate := flags.String("session-date", "", "required completed market-session date (YYYY-MM-DD)")
	maxSymbols := flags.Int("max-symbols", 10, "bounded canary size, from 1 through 50")
	actor := flags.String("actor", "subscriber-global-eod-reconciler", "controlled global EOD worker identity")
	correlationID := flags.String("correlation-id", "", "optional trace correlation id")
	execute := flags.Bool("execute", false, "persist the prepared canary cohort")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !*execute {
		return errors.New("refusing to mutate: pass --execute after migration and workload-identity preflight")
	}
	if strings.TrimSpace(*databaseURL) == "" || strings.TrimSpace(*planRunID) == "" || strings.TrimSpace(*sessionDate) == "" {
		return errors.New("dedicated database URL, plan run id, and session date are required")
	}
	parsedSessionDate, err := time.Parse("2006-01-02", strings.TrimSpace(*sessionDate))
	if err != nil {
		return fmt.Errorf("parse session date: %w", err)
	}
	if *maxSymbols <= 0 || *maxSymbols > eodcanary.MaximumCanarySize {
		return errors.New("max symbols must be between 1 and 50")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	repo, err := postgresstorage.Open(ctx, *databaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	prepared, err := repo.PrepareSubscriberGlobalEODCanary(ctx, storage.SubscriberGlobalEODCanaryPreparation{
		PlanRunID: *planRunID, SessionDate: parsedSessionDate, MaxSymbols: *maxSymbols,
		PreparedBy: *actor, CorrelationID: *correlationID,
	})
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(prepared)
}

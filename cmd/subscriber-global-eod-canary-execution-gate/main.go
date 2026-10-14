// subscriber-global-eod-canary-execution-gate writes an append-only disabled
// execution control for a prepared S4 canary. It cannot call a provider.
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
	"github.com/lukebabs/signalops/internal/subscriber/eodcanaryexecution"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "subscriber global EOD canary execution gate failed:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("subscriber-global-eod-canary-execution-gate", flag.ContinueOnError)
	databaseURL := flags.String("database-url", os.Getenv("SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL"), "dedicated Subscriber global-EOD PostgreSQL URL")
	canaryRunID := flags.String("canary-run-id", "", "required prepared S4 canary run id")
	actor := flags.String("actor", eodcanaryexecution.ExpectedWorkerIdentity, "controlled global EOD worker identity")
	correlationID := flags.String("correlation-id", "", "required immutable trace correlation id")
	acknowledge := flags.Bool("acknowledge-provider-disabled", false, "confirm this command only writes a disabled plan and makes no provider request")
	execute := flags.Bool("execute", false, "persist the disabled execution control")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !*execute || !*acknowledge {
		return errors.New("refusing to mutate: pass --execute and --acknowledge-provider-disabled")
	}
	if strings.TrimSpace(*databaseURL) == "" || strings.TrimSpace(*canaryRunID) == "" || strings.TrimSpace(*correlationID) == "" {
		return errors.New("dedicated database URL, canary run id, and correlation id are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	repo, err := postgresstorage.Open(ctx, *databaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	gate, err := repo.PrepareSubscriberGlobalEODCanaryExecutionGate(ctx, storage.SubscriberGlobalEODCanaryExecutionGate{
		CanaryRunID: *canaryRunID, ExpectedWorkerIdentity: *actor, MaxProviderRequests: eodcanaryexecution.MaximumProviderRequests,
		PlannedBy: *actor, CorrelationID: *correlationID,
	})
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(gate)
}

// subscriber-options-capture-canary-gate creates only a disabled one-asset
// S6 capture plan from a selected demand snapshot. It cannot call Massive.
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
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "subscriber Options capture canary gate failed:", err)
		os.Exit(1)
	}
}
func run(args []string) error {
	flags := flag.NewFlagSet("subscriber-options-capture-canary-gate", flag.ContinueOnError)
	databaseURL := flags.String("database-url", os.Getenv("SIGNALOPS_SUBSCRIBER_OPTIONS_CAPTURE_DATABASE_URL"), "dedicated Options-capture database URL")
	snapshotRunID := flags.String("snapshot-run-id", "", "selected S6 shadow snapshot")
	correlationID := flags.String("correlation-id", "", "change correlation id")
	actor := flags.String("actor", "subscriber-options-capture", "required capture identity")
	ack := flags.Bool("acknowledge-provider-disabled", false, "confirm this writes only a disabled capture plan")
	execute := flags.Bool("execute", false, "persist disabled one-asset capture plan")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !*execute || !*ack {
		return errors.New("refusing to mutate: pass --execute and --acknowledge-provider-disabled")
	}
	if strings.TrimSpace(*databaseURL) == "" || strings.TrimSpace(*snapshotRunID) == "" || strings.TrimSpace(*correlationID) == "" || strings.TrimSpace(*actor) != "subscriber-options-capture" {
		return errors.New("dedicated database URL, snapshot run id, correlation id, and subscriber-options-capture identity are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	repo, err := postgresstorage.Open(ctx, *databaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	if err := repo.AssumeSubscriberOptionsCaptureRole(ctx); err != nil {
		return err
	}
	gate, err := repo.PrepareSubscriberOptionsCaptureCanaryGate(ctx, storage.SubscriberOptionsCaptureCanaryGate{SnapshotRunID: *snapshotRunID, PlannedBy: *actor, CorrelationID: *correlationID})
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(gate)
}

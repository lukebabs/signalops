// subscriber-options-capture-authorization-request persists a pending review
// request only. It cannot grant approval or call a market-data provider.
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
		fmt.Fprintln(os.Stderr, "subscriber Options capture authorization request failed:", err)
		os.Exit(1)
	}
}
func run(args []string) error {
	flags := flag.NewFlagSet("subscriber-options-capture-authorization-request", flag.ContinueOnError)
	databaseURL := flags.String("database-url", os.Getenv("SIGNALOPS_SUBSCRIBER_OPTIONS_CAPTURE_DATABASE_URL"), "dedicated Options-capture database URL")
	capturePlanID := flags.String("capture-plan-id", "", "disabled capture plan")
	reason := flags.String("reason", "", "reason for review")
	correlationID := flags.String("correlation-id", "", "change correlation id")
	execute := flags.Bool("execute", false, "persist pending approval request")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !*execute {
		return errors.New("refusing to mutate: pass --execute; this records pending approval only")
	}
	if strings.TrimSpace(*databaseURL) == "" || strings.TrimSpace(*capturePlanID) == "" || strings.TrimSpace(*reason) == "" || strings.TrimSpace(*correlationID) == "" {
		return errors.New("dedicated database URL, capture plan id, reason, and correlation id are required")
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
	request, err := repo.RequestSubscriberOptionsCaptureAuthorization(ctx, storage.SubscriberOptionsCaptureAuthorizationRequest{CapturePlanID: *capturePlanID, RequestedBy: "subscriber-options-capture", RequestReason: *reason, CorrelationID: *correlationID})
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(request)
}

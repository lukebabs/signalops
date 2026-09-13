// subscriber-global-catalog-seed performs the controlled S1 compatibility
// import. It is manual and refuses mutation unless --execute is supplied.
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
		fmt.Fprintln(os.Stderr, "subscriber global catalog seed failed:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("subscriber-global-catalog-seed", flag.ContinueOnError)
	databaseURL := flags.String("database-url", os.Getenv("SIGNALOPS_DATABASE_URL"), "SignalOps PostgreSQL URL")
	sourceTenantID := flags.String("source-tenant-id", "tenant-local", "current compatibility-universe tenant")
	actorIdentity := flags.String("actor", "subscriber-catalog-reference-sync", "controlled worker identity recorded in provenance")
	correlationID := flags.String("correlation-id", "", "optional trace correlation id")
	execute := flags.Bool("execute", false, "perform the additive shadow seed")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !*execute {
		return errors.New("refusing to mutate: pass --execute after migration and role preflight")
	}
	if strings.TrimSpace(*databaseURL) == "" {
		return errors.New("SIGNALOPS_DATABASE_URL or --database-url is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	repo, err := postgresstorage.Open(ctx, *databaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	result, err := repo.SeedSubscriberGlobalCatalogShadow(ctx, storage.SubscriberGlobalCatalogSeedRequest{
		SourceTenantID: *sourceTenantID,
		ActorIdentity:  *actorIdentity,
		CorrelationID:  *correlationID,
	})
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}

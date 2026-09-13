// subscriber-options-demand-shadow-planner persists a non-identifying S6
// shadow plan. It has no provider client, scheduler integration, or capture path.
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
	"github.com/lukebabs/signalops/internal/subscriber/optionsdemand"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "subscriber options demand shadow planner failed:", err)
		os.Exit(1)
	}
}
func run(args []string) error {
	flags := flag.NewFlagSet("subscriber-options-demand-shadow-planner", flag.ContinueOnError)
	databaseURL := flags.String("database-url", os.Getenv("SIGNALOPS_SUBSCRIBER_OPTIONS_DEMAND_DATABASE_URL"), "dedicated Options-demand database URL")
	sessionDate := flags.String("session-date", "", "completed market session YYYY-MM-DD")
	capacity := flags.Int("max-symbols", 50, "shadow capacity from 1 through 1000")
	actor := flags.String("actor", "subscriber-options-demand-planner", "required machine identity")
	correlationID := flags.String("correlation-id", "", "change correlation id")
	execute := flags.Bool("execute", false, "persist append-only shadow snapshot")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !*execute {
		return errors.New("refusing to mutate: pass --execute after migration and workload preflight")
	}
	if strings.TrimSpace(*databaseURL) == "" || strings.TrimSpace(*actor) != "subscriber-options-demand-planner" {
		return errors.New("dedicated database URL and subscriber-options-demand-planner identity are required")
	}
	date, err := time.Parse("2006-01-02", strings.TrimSpace(*sessionDate))
	if err != nil {
		return errors.New("session-date must be YYYY-MM-DD")
	}
	if *capacity < 1 || *capacity > 1000 {
		return errors.New("max-symbols must be 1 through 1000")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	repo, err := postgresstorage.Open(ctx, *databaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	if err := repo.AssumeSubscriberOptionsDemandRole(ctx); err != nil {
		return err
	}
	aggregates, err := repo.ListSubscriberOptionsDemandAggregates(ctx)
	if err != nil {
		return err
	}
	candidates := make([]optionsdemand.Candidate, 0, len(aggregates))
	sourceCount := 0
	for _, a := range aggregates {
		candidates = append(candidates, optionsdemand.Candidate{GlobalAssetID: a.GlobalAssetID, HighestTierRank: a.HighestTierRank, EligibleTenantCount: a.EligibleTenantCount, EligibleWatcherCount: a.EligibleWatcherCount, DeferredSessions: a.DeferredSessions})
		sourceCount += a.EligibleWatcherCount
	}
	plan, err := optionsdemand.BuildCandidates(optionsdemand.Config{MaxSymbols: *capacity}, candidates)
	if err != nil {
		return err
	}
	members := append(toMembers(plan.Selected, "selected", 0), toMembers(plan.Deferred, "deferred", len(plan.Selected))...)
	snapshot, err := repo.RecordSubscriberOptionsDemandShadowSnapshot(ctx, storage.SubscriberOptionsDemandSnapshot{SessionDate: date, MaxSymbols: *capacity, SourceDemandCount: sourceCount, Members: members, PlannedBy: *actor, CorrelationID: *correlationID})
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(snapshot)
}
func toMembers(candidates []optionsdemand.Candidate, state string, offset int) []storage.SubscriberOptionsDemandSnapshotMember {
	out := make([]storage.SubscriberOptionsDemandSnapshotMember, 0, len(candidates))
	for i, c := range candidates {
		out = append(out, storage.SubscriberOptionsDemandSnapshotMember{SubscriberOptionsDemandAggregate: storage.SubscriberOptionsDemandAggregate{GlobalAssetID: c.GlobalAssetID, HighestTierRank: c.HighestTierRank, EligibleTenantCount: c.EligibleTenantCount, EligibleWatcherCount: c.EligibleWatcherCount, DeferredSessions: c.DeferredSessions}, Priority: offset + i + 1, SelectionState: state})
	}
	return out
}

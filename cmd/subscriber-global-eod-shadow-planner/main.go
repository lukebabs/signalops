// subscriber-global-eod-shadow-planner creates an auditable S2 plan. It does
// not call providers, enqueue work, or change coverage execution_mode.
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
	"github.com/lukebabs/signalops/internal/subscriber/eodplanner"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "subscriber global EOD shadow planner failed:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("subscriber-global-eod-shadow-planner", flag.ContinueOnError)
	databaseURL := flags.String("database-url", os.Getenv("SIGNALOPS_DATABASE_URL"), "SignalOps PostgreSQL URL")
	capacity := flags.Int("capacity", eodplanner.MaximumHotSetCapacity, "approved hot-set maximum, from 1 through 1000")
	actor := flags.String("actor", "subscriber-global-eod-reconciler", "controlled planner identity")
	correlationID := flags.String("correlation-id", "", "optional trace correlation id")
	execute := flags.Bool("execute", false, "persist an additive shadow plan")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !*execute {
		return errors.New("refusing to mutate: pass --execute after migration and workload-identity preflight")
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
	candidates, err := repo.ListSubscriberGlobalEODHotSetCandidates(ctx, 10000)
	if err != nil {
		return err
	}
	input := make([]eodplanner.Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		input = append(input, eodplanner.Candidate{GlobalAssetID: candidate.GlobalAssetID, EligibilityStatus: candidate.EligibilityStatus, ActiveSourceRows: candidate.ActiveSourceRows, BestSourceRank: candidate.BestSourceRank})
	}
	plan, err := eodplanner.Build(input, *capacity)
	if err != nil {
		return err
	}
	persisted, err := repo.RecordSubscriberGlobalEODHotSetShadowPlan(ctx, storage.SubscriberGlobalEODHotSetPlan{
		PlannerVersion: eodplanner.PlannerVersion, Capacity: plan.Capacity, CandidateCount: plan.CandidateCount,
		EligibleCount: plan.EligibleCount, ExcludedCount: plan.ExcludedCount, ExcludedByReason: plan.ExcludedByReason,
		Members: toStorageMembers(plan.Members), PlannedBy: *actor, CorrelationID: *correlationID,
	})
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(persisted)
}

func toStorageMembers(members []eodplanner.Member) []storage.SubscriberGlobalEODHotSetMember {
	result := make([]storage.SubscriberGlobalEODHotSetMember, 0, len(members))
	for _, member := range members {
		result = append(result, storage.SubscriberGlobalEODHotSetMember{GlobalAssetID: member.GlobalAssetID, Priority: member.Priority, SourceRank: member.SourceRank})
	}
	return result
}

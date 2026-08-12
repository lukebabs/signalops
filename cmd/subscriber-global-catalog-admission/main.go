// subscriber-global-catalog-admission imports bounded Massive ticker-reference
// evidence. It does not fetch prices, activate coverage, or schedule work.
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

	"github.com/lukebabs/signalops/internal/adapters/marketdata/massive"
	"github.com/lukebabs/signalops/internal/storage"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
	"github.com/lukebabs/signalops/internal/subscriber/catalogadmission"
)

type repository interface {
	storage.SubscriberGlobalCatalogEligibilityRepository
}

type report struct {
	Candidates int      `json:"candidates"`
	Eligible   int      `json:"eligible"`
	Ineligible int      `json:"ineligible"`
	Failed     int      `json:"failed"`
	Failures   []string `json:"failures,omitempty"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "subscriber global catalog admission failed:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("subscriber-global-catalog-admission", flag.ContinueOnError)
	databaseURL := flags.String("database-url", os.Getenv("SIGNALOPS_DATABASE_URL"), "catalog-sync PostgreSQL URL")
	maxAssets := flags.Int("max-assets", 200, "maximum discovered assets to reference; 1 through 1000")
	delay := flags.Duration("request-delay", 300*time.Millisecond, "minimum delay between Massive reference requests")
	actor := flags.String("actor", "subscriber-catalog-reference-sync", "controlled catalog-sync identity")
	correlationID := flags.String("correlation-id", "", "optional trace correlation id")
	execute := flags.Bool("execute", false, "record immutable governed admission decisions")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !*execute {
		return errors.New("refusing to mutate: pass --execute after admission review")
	}
	if strings.TrimSpace(*databaseURL) == "" {
		return errors.New("SIGNALOPS_DATABASE_URL or --database-url is required")
	}
	if *maxAssets <= 0 || *maxAssets > 1000 {
		return errors.New("max-assets must be between 1 and 1000")
	}
	if *delay < 0 {
		return errors.New("request-delay cannot be negative")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	repo, err := postgresstorage.Open(ctx, *databaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	client, err := massive.NewClient(massive.LoadClientConfigFromEnv())
	if err != nil {
		return err
	}
	candidates, err := repo.ListSubscriberGlobalReferenceCandidates(ctx, *maxAssets)
	if err != nil {
		return err
	}
	out := report{Candidates: len(candidates), Failures: []string{}}
	for index, candidate := range candidates {
		if index > 0 && *delay > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(*delay):
			}
		}
		details, err := client.GetTickerDetails(ctx, candidate.ProviderSymbol)
		if err != nil {
			out.Failed++
			out.Failures = append(out.Failures, candidate.ProviderSymbol+":"+err.Error())
			now := time.Now().UTC()
			evidence, encodeErr := json.Marshal(map[string]any{"provider": "massive", "provider_symbol": candidate.ProviderSymbol, "lookup_error": err.Error()})
			if encodeErr != nil {
				return encodeErr
			}
			if _, recordErr := repo.RecordSubscriberGlobalAssetEligibilityDecision(ctx, storage.SubscriberGlobalAssetEligibilityDecision{
				GlobalAssetID: candidate.GlobalAssetID, Decision: "deferred", ReasonCode: "massive_reference_lookup_failed",
				ProviderReferenceAt: &now, EvidenceJSON: evidence, ProvenanceJSON: admissionProvenance(*correlationID),
				DecidedBy: *actor, DecidedAt: now,
			}); recordErr != nil {
				return recordErr
			}
			continue
		}
		decision := catalogadmission.Evaluate(details)
		evidence, err := json.Marshal(decision.Evidence)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		_, err = repo.RecordSubscriberGlobalAssetEligibilityDecision(ctx, storage.SubscriberGlobalAssetEligibilityDecision{
			GlobalAssetID: candidate.GlobalAssetID, Decision: decision.Decision, ReasonCode: decision.ReasonCode,
			ProviderReferenceAt: &now, EvidenceJSON: evidence,
			ProvenanceJSON: admissionProvenance(*correlationID),
			DecidedBy:      *actor, DecidedAt: now,
		})
		if err != nil {
			return err
		}
		if decision.Decision == "eligible" {
			out.Eligible++
		} else {
			out.Ineligible++
		}
	}
	return json.NewEncoder(os.Stdout).Encode(out)
}

func admissionProvenance(correlationID string) []byte {
	value, _ := json.Marshal(map[string]string{"source": "massive_ticker_reference", "import_version": "s2-admission-v1", "correlation_id": strings.TrimSpace(correlationID)})
	return value
}

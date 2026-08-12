// subscriber-s3-pilot-provisioner creates the initial S3 pilot preference
// records through the normal RLS-scoped repository. It never enables a flag,
// creates global assets, changes coverage, or invokes a provider.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
)

const defaultPilotTier = "subscriber-list-pilot-v1"

type config struct {
	DatabaseURL     string
	TenantID        string
	Actor           string
	DefaultListName string
	GlobalAssetIDs  []string
	CorrelationID   string
}

func main() {
	var rawAssets string
	cfg := config{}
	flag.StringVar(&cfg.DatabaseURL, "database-url", strings.TrimSpace(os.Getenv("SIGNALOPS_SUBSCRIBER_GATEWAY_DATABASE_URL")), "dedicated subscriber gateway database URL")
	flag.StringVar(&cfg.TenantID, "tenant-id", "", "pilot tenant ID")
	flag.StringVar(&cfg.Actor, "actor", "", "controlled provisioning actor identity")
	flag.StringVar(&cfg.DefaultListName, "default-list-name", "MarketOps Pilot Default", "tenant-default list name")
	flag.StringVar(&rawAssets, "global-asset-ids", "", "comma-separated governed global asset IDs")
	flag.StringVar(&cfg.CorrelationID, "correlation-id", "", "provisioning correlation ID")
	flag.Parse()
	cfg.GlobalAssetIDs = splitIDs(rawAssets)

	if strings.TrimSpace(cfg.DatabaseURL) == "" || strings.TrimSpace(cfg.TenantID) == "" || strings.TrimSpace(cfg.Actor) == "" || len(cfg.GlobalAssetIDs) == 0 {
		flag.Usage()
		os.Exit(2)
	}
	if cfg.CorrelationID == "" {
		cfg.CorrelationID = "subscriber-s3-pilot-" + strings.TrimSpace(cfg.TenantID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	repo, err := postgresstorage.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("open subscriber gateway database", "error", err)
		os.Exit(1)
	}
	defer repo.Close()

	result, err := provision(ctx, repo, cfg)
	if err != nil {
		slog.Error("provision S3 pilot", "tenant_id", cfg.TenantID, "error", err)
		os.Exit(1)
	}
	fmt.Printf("{\"tenant_id\":%q,\"product_tier\":%q,\"default_list_id\":%q,\"seeded_memberships\":%d,\"capabilities\":{\"catalog_search\":false,\"eod_activation\":false,\"options_demand\":false}}\n", result.TenantID, defaultPilotTier, result.DefaultListID, result.SeededMemberships)
}

type result struct {
	TenantID          string
	DefaultListID     string
	SeededMemberships int
}

type pilotStore interface {
	UpsertSubscriberEntitlement(context.Context, storage.SubscriberEntitlementRecord) (storage.SubscriberEntitlementRecord, error)
	ListSubscriberWatchlists(context.Context, string, string) ([]storage.SubscriberWatchlistRecord, error)
	CreateSubscriberTenantDefaultWatchlist(context.Context, storage.SubscriberWatchlistCreateRequest) (storage.SubscriberWatchlistRecord, error)
	AddSubscriberTenantDefaultWatchlistMembership(context.Context, storage.SubscriberWatchlistMembershipRequest) (storage.SubscriberWatchlistMembershipRecord, error)
}

func provision(ctx context.Context, store pilotStore, cfg config) (result, error) {
	cfg.TenantID, cfg.Actor, cfg.DefaultListName, cfg.CorrelationID = strings.TrimSpace(cfg.TenantID), strings.TrimSpace(cfg.Actor), strings.TrimSpace(cfg.DefaultListName), strings.TrimSpace(cfg.CorrelationID)
	if cfg.TenantID == "" || cfg.Actor == "" || cfg.DefaultListName == "" || len(cfg.GlobalAssetIDs) == 0 {
		return result{}, fmt.Errorf("tenant, actor, default list name, and global asset IDs are required")
	}
	entitlement, err := store.UpsertSubscriberEntitlement(ctx, storage.SubscriberEntitlementRecord{
		TenantID: cfg.TenantID, ProvisioningVersion: defaultPilotTier, ProductTier: defaultPilotTier,
		Status: storage.SubscriberEntitlementActive, ProvisionedBy: cfg.Actor, CorrelationID: cfg.CorrelationID,
		Capabilities: []storage.SubscriberEntitlementCapabilityRecord{
			{Capability: "catalog_search", Enabled: false, QuotaLimit: 0},
			{Capability: "eod_activation", Enabled: false, QuotaLimit: 0},
			{Capability: "options_demand", Enabled: false, QuotaLimit: 0},
		},
	})
	if err != nil {
		return result{}, fmt.Errorf("provision pilot entitlement: %w", err)
	}
	lists, err := store.ListSubscriberWatchlists(ctx, cfg.TenantID, cfg.Actor)
	if err != nil {
		return result{}, fmt.Errorf("list pilot default list: %w", err)
	}
	var defaultList storage.SubscriberWatchlistRecord
	for _, list := range lists {
		if list.ListKind == storage.SubscriberWatchlistKindTenantDefault {
			defaultList = list
			break
		}
	}
	if defaultList.ListID == "" {
		defaultList, err = store.CreateSubscriberTenantDefaultWatchlist(ctx, storage.SubscriberWatchlistCreateRequest{
			TenantID: cfg.TenantID, ListName: cfg.DefaultListName, ActorSubject: cfg.Actor, CorrelationID: cfg.CorrelationID,
		})
		if err != nil {
			return result{}, fmt.Errorf("create pilot default list: %w", err)
		}
	}
	for _, globalAssetID := range cfg.GlobalAssetIDs {
		if _, err := store.AddSubscriberTenantDefaultWatchlistMembership(ctx, storage.SubscriberWatchlistMembershipRequest{
			TenantID: cfg.TenantID, ListID: defaultList.ListID, GlobalAssetID: globalAssetID, ActorSubject: cfg.Actor, CorrelationID: cfg.CorrelationID,
		}); err != nil {
			return result{}, fmt.Errorf("seed global asset %s: %w", globalAssetID, err)
		}
	}
	return result{TenantID: entitlement.TenantID, DefaultListID: defaultList.ListID, SeededMemberships: len(cfg.GlobalAssetIDs)}, nil
}

func splitIDs(raw string) []string {
	seen := map[string]struct{}{}
	values := []string{}
	for _, value := range strings.Split(raw, ",") {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return values
}

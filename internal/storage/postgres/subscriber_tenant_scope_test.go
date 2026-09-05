package postgres

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestValidSubscriberTenantID(t *testing.T) {
	for _, testCase := range []struct {
		tenantID string
		want     bool
	}{
		{tenantID: "tenant-local", want: true},
		{tenantID: "tenant_123", want: true},
		{tenantID: " tenant-a ", want: true},
		{tenantID: "", want: false},
		{tenantID: "tenant/a", want: false},
		{tenantID: "tenant a", want: false},
		{tenantID: "-tenant", want: false},
	} {
		if got := validSubscriberTenantID(testCase.tenantID); got != testCase.want {
			t.Fatalf("validSubscriberTenantID(%q) = %t, want %t", testCase.tenantID, got, testCase.want)
		}
	}
}

func TestWithSubscriberTenantScopeAgainstPostgres(t *testing.T) {
	repo := subscriberIntegrationRepository(t)
	defer repo.Close()
	repo.db.SetMaxOpenConns(1)
	repo.db.SetMaxIdleConns(1)
	ctx := context.Background()

	if err := repo.WithSubscriberTenantScope(ctx, "tenant-scope-test", func(ctx context.Context, tx *sql.Tx) error {
		var scoped string
		if err := tx.QueryRowContext(ctx, "SELECT current_setting('signalops.tenant_id', true)").Scan(&scoped); err != nil {
			return err
		}
		if scoped != "tenant-scope-test" {
			t.Fatalf("scoped tenant = %q", scoped)
		}
		return nil
	}); err != nil {
		t.Fatalf("with subscriber tenant scope: %v", err)
	}

	var after string
	if err := repo.db.QueryRowContext(ctx, "SELECT COALESCE(current_setting('signalops.tenant_id', true), '')").Scan(&after); err != nil {
		t.Fatalf("read tenant context after commit: %v", err)
	}
	if after != "" {
		t.Fatalf("tenant context leaked after transaction: %q", after)
	}
}

func TestSubscriberEntitlementReservationAgainstPostgres(t *testing.T) {
	repo := subscriberIntegrationRepository(t)
	ctx := context.Background()
	tenantID := "subscriber-entitlement-it"
	defer func() {
		for _, table := range []string{
			"subscriber_entitlement_decision_audit",
			"subscriber_entitlement_provisioning_audit",
			"subscriber_quota_reservations",
			"subscriber_entitlement_capabilities",
			"subscriber_tenant_entitlements",
		} {
			if _, err := repo.db.ExecContext(ctx, "DELETE FROM "+table+" WHERE tenant_id=$1", tenantID); err != nil {
				t.Errorf("cleanup %s: %v", table, err)
			}
		}
		repo.Close()
	}()

	record, err := repo.UpsertSubscriberEntitlement(ctx, storage.SubscriberEntitlementRecord{
		TenantID: tenantID, ProvisioningVersion: "v1", ProductTier: "pilot",
		Status: storage.SubscriberEntitlementActive, ProvisionedBy: "test-admin",
		Capabilities: []storage.SubscriberEntitlementCapabilityRecord{
			{Capability: "options_demand", Enabled: true, QuotaLimit: 3},
		},
	})
	if err != nil {
		t.Fatalf("provision entitlement: %v", err)
	}
	if record.ProvisioningVersion != "v1" || len(record.Capabilities) != 1 {
		t.Fatalf("provisioned record = %+v", record)
	}

	request := storage.SubscriberQuotaReservationRequest{
		TenantID: tenantID, Subject: "test-subject", Capability: "options_demand",
		RequestedUnits: 2, IdempotencyKey: "reserve-1", CorrelationID: "corr-1",
		RequestedAt: time.Date(2026, 8, 12, 1, 0, 0, 0, time.UTC),
	}
	reservation, decision, err := repo.ReserveSubscriberQuota(ctx, request)
	if err != nil {
		t.Fatalf("reserve quota: %v", err)
	}
	if reservation.ReservationID == "" || decision.DecisionReason != "allowed" {
		t.Fatalf("reservation=%+v decision=%+v", reservation, decision)
	}

	retry, retryDecision, err := repo.ReserveSubscriberQuota(ctx, request)
	if err != nil {
		t.Fatalf("retry reserve quota: %v", err)
	}
	if retry.ReservationID != reservation.ReservationID || retryDecision.DecisionReason != "allowed" {
		t.Fatalf("idempotent retry reservation=%+v decision=%+v", retry, retryDecision)
	}

	request.IdempotencyKey = "reserve-2"
	request.RequestedUnits = 2
	_, deferred, err := repo.ReserveSubscriberQuota(ctx, request)
	if err != nil {
		t.Fatalf("reserve over quota: %v", err)
	}
	if deferred.DecisionReason != "deferred_quota" {
		t.Fatalf("over quota decision=%+v", deferred)
	}

	var auditCount int
	if err := repo.db.QueryRowContext(ctx, "SELECT count(*) FROM subscriber_entitlement_decision_audit WHERE tenant_id=$1", tenantID).Scan(&auditCount); err != nil {
		t.Fatalf("count decision audit: %v", err)
	}
	if auditCount != 3 {
		t.Fatalf("decision audit count = %d, want 3", auditCount)
	}
}

func subscriberIntegrationRepository(t *testing.T) *Repository {
	t.Helper()
	if os.Getenv("SIGNALOPS_SUBSCRIBER_RLS_INTEGRATION") != "1" {
		t.Skip("set SIGNALOPS_SUBSCRIBER_RLS_INTEGRATION=1 to run")
	}
	dsn := os.Getenv("SIGNALOPS_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://signalops:signalops@localhost:15432/signalops?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	repo, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	return repo
}

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestSubscriberQuotaLifecycleAgainstPostgres(t *testing.T) {
	repo := subscriberIntegrationRepository(t)
	defer repo.Close()
	ctx := context.Background()
	tenantID := "subscriber-quota-lifecycle-it"
	defer func() {
		for _, table := range []string{
			"subscriber_quota_reservation_audit",
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
	}()

	if _, err := repo.UpsertSubscriberEntitlement(ctx, storage.SubscriberEntitlementRecord{
		TenantID: tenantID, ProvisioningVersion: "v1", Status: storage.SubscriberEntitlementActive,
		ProvisionedBy: "test-admin", Capabilities: []storage.SubscriberEntitlementCapabilityRecord{
			{Capability: "catalog_search", Enabled: true, QuotaLimit: 5},
		},
	}); err != nil {
		t.Fatalf("provision entitlement: %v", err)
	}

	reserve := func(key string) storage.SubscriberQuotaReservationRecord {
		record, decision, err := repo.ReserveSubscriberQuota(ctx, storage.SubscriberQuotaReservationRequest{
			TenantID: tenantID, Subject: "test-subject", Capability: "catalog_search", RequestedUnits: 1,
			IdempotencyKey: key, RequestedAt: time.Date(2026, 8, 12, 2, 0, 0, 0, time.UTC),
		})
		if err != nil || decision.DecisionReason != "allowed" {
			t.Fatalf("reserve %s record=%+v decision=%+v err=%v", key, record, decision, err)
		}
		return record
	}

	released := reserve("release-1")
	released, err := repo.FinalizeSubscriberQuotaReservation(ctx, storage.SubscriberQuotaReservationLifecycleRequest{
		TenantID: tenantID, ReservationID: released.ReservationID, ActorSubject: "test-worker",
		Transition: storage.SubscriberQuotaReleased, OccurredAt: time.Date(2026, 8, 12, 2, 1, 0, 0, time.UTC),
	})
	if err != nil || released.Status != storage.SubscriberQuotaReleased || released.ReleasedAt == nil {
		t.Fatalf("release record=%+v err=%v", released, err)
	}
	if _, err := repo.FinalizeSubscriberQuotaReservation(ctx, storage.SubscriberQuotaReservationLifecycleRequest{
		TenantID: tenantID, ReservationID: released.ReservationID, ActorSubject: "test-worker",
		Transition: storage.SubscriberQuotaReleased,
	}); err != nil {
		t.Fatalf("idempotent release: %v", err)
	}

	reactivated := reserve("release-1")
	if reactivated.ReservationID != released.ReservationID || reactivated.Status != storage.SubscriberQuotaReserved {
		t.Fatalf("re-reserved record=%+v released=%+v", reactivated, released)
	}
	if _, err := repo.FinalizeSubscriberQuotaReservation(ctx, storage.SubscriberQuotaReservationLifecycleRequest{
		TenantID: tenantID, ReservationID: reactivated.ReservationID, ActorSubject: "test-worker",
		Transition: storage.SubscriberQuotaReleased,
	}); err != nil {
		t.Fatalf("release re-reserved quota: %v", err)
	}

	consumed := reserve("consume-1")
	consumed, err = repo.FinalizeSubscriberQuotaReservation(ctx, storage.SubscriberQuotaReservationLifecycleRequest{
		TenantID: tenantID, ReservationID: consumed.ReservationID, ActorSubject: "test-worker",
		Transition: storage.SubscriberQuotaConsumed,
	})
	if err != nil || consumed.Status != storage.SubscriberQuotaConsumed || consumed.ReleasedAt != nil {
		t.Fatalf("consume record=%+v err=%v", consumed, err)
	}

	var auditCount int
	if err := repo.db.QueryRowContext(ctx, "SELECT count(*) FROM subscriber_quota_reservation_audit WHERE tenant_id=$1", tenantID).Scan(&auditCount); err != nil {
		t.Fatalf("count lifecycle audit: %v", err)
	}
	if auditCount != 3 {
		t.Fatalf("lifecycle audit count=%d, want 3", auditCount)
	}
}

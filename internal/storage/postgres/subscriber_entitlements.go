package postgres

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
	"github.com/lukebabs/signalops/internal/subscriber/policy"
)

func (r *Repository) UpsertSubscriberEntitlement(ctx context.Context, record storage.SubscriberEntitlementRecord) (storage.SubscriberEntitlementRecord, error) {
	if err := validateSubscriberEntitlement(record); err != nil {
		return record, err
	}
	record.TenantID = strings.TrimSpace(record.TenantID)
	record.ProvisioningVersion = strings.TrimSpace(record.ProvisioningVersion)
	record.ProductTier = strings.TrimSpace(record.ProductTier)
	record.ProvisionedBy = strings.TrimSpace(record.ProvisionedBy)
	record.CorrelationID = strings.TrimSpace(record.CorrelationID)

	err := r.WithSubscriberTenantScope(ctx, record.TenantID, func(ctx context.Context, tx *sql.Tx) error {
		var before string
		err := tx.QueryRowContext(ctx, `SELECT jsonb_build_object('provisioning_version', provisioning_version, 'product_tier', product_tier, 'status', status)::text FROM subscriber_tenant_entitlements WHERE tenant_id=$1 FOR UPDATE`, record.TenantID).Scan(&before)
		mutation := "update"
		if errors.Is(err, sql.ErrNoRows) {
			before = "{}"
			mutation = "provision"
		} else if err != nil {
			return fmt.Errorf("read existing entitlement: %w", err)
		}
		if record.Status == storage.SubscriberEntitlementSuspended {
			mutation = "suspend"
		}

		if err := tx.QueryRowContext(ctx, `
INSERT INTO subscriber_tenant_entitlements (tenant_id, provisioning_version, product_tier, status, provisioned_by, correlation_id)
VALUES ($1,$2,$3,$4,$5,$6)
ON CONFLICT (tenant_id) DO UPDATE SET
  provisioning_version=EXCLUDED.provisioning_version,
  product_tier=EXCLUDED.product_tier,
  status=EXCLUDED.status,
  provisioned_by=EXCLUDED.provisioned_by,
  correlation_id=EXCLUDED.correlation_id,
  updated_at=now()
RETURNING created_at, updated_at`,
			record.TenantID, record.ProvisioningVersion, record.ProductTier, record.Status, record.ProvisionedBy, record.CorrelationID,
		).Scan(&record.CreatedAt, &record.UpdatedAt); err != nil {
			return fmt.Errorf("upsert entitlement: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM subscriber_entitlement_capabilities WHERE tenant_id=$1`, record.TenantID); err != nil {
			return fmt.Errorf("replace entitlement capabilities: %w", err)
		}
		for _, capability := range record.Capabilities {
			if _, err := tx.ExecContext(ctx, `INSERT INTO subscriber_entitlement_capabilities (tenant_id, capability, enabled, quota_limit) VALUES ($1,$2,$3,$4)`,
				record.TenantID, capability.Capability, capability.Enabled, capability.QuotaLimit,
			); err != nil {
				return fmt.Errorf("insert entitlement capability: %w", err)
			}
		}
		after, err := json.Marshal(map[string]any{
			"provisioning_version": record.ProvisioningVersion,
			"product_tier":         record.ProductTier,
			"status":               record.Status,
			"capabilities":         record.Capabilities,
		})
		if err != nil {
			return fmt.Errorf("encode entitlement audit: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO subscriber_entitlement_provisioning_audit
  (audit_id, tenant_id, actor_subject, mutation, before_value, after_value, correlation_id)
VALUES ($1,$2,$3,$4,$5::jsonb,$6::jsonb,$7)`,
			newSubscriberID("subentprov"), record.TenantID, record.ProvisionedBy, mutation, before, string(after), record.CorrelationID,
		); err != nil {
			return fmt.Errorf("audit entitlement provisioning: %w", err)
		}
		return nil
	})
	return record, err
}

func (r *Repository) GetSubscriberEntitlement(ctx context.Context, tenantID string) (storage.SubscriberEntitlementRecord, error) {
	tenantID = strings.TrimSpace(tenantID)
	var record storage.SubscriberEntitlementRecord
	err := r.WithSubscriberTenantScope(ctx, tenantID, func(ctx context.Context, tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx, `
SELECT tenant_id, provisioning_version, product_tier, status, provisioned_by, correlation_id, created_at, updated_at
FROM subscriber_tenant_entitlements WHERE tenant_id=$1`, tenantID,
		).Scan(&record.TenantID, &record.ProvisioningVersion, &record.ProductTier, &record.Status, &record.ProvisionedBy, &record.CorrelationID, &record.CreatedAt, &record.UpdatedAt)
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("get subscriber entitlement: %w", err)
		}
		rows, err := tx.QueryContext(ctx, `SELECT capability, enabled, quota_limit FROM subscriber_entitlement_capabilities WHERE tenant_id=$1 ORDER BY capability`, tenantID)
		if err != nil {
			return fmt.Errorf("list entitlement capabilities: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var capability storage.SubscriberEntitlementCapabilityRecord
			if err := rows.Scan(&capability.Capability, &capability.Enabled, &capability.QuotaLimit); err != nil {
				return err
			}
			record.Capabilities = append(record.Capabilities, capability)
		}
		return rows.Err()
	})
	return record, err
}

func (r *Repository) ReserveSubscriberQuota(ctx context.Context, request storage.SubscriberQuotaReservationRequest) (storage.SubscriberQuotaReservationRecord, storage.SubscriberEntitlementDecisionRecord, error) {
	var reservation storage.SubscriberQuotaReservationRecord
	var decision storage.SubscriberEntitlementDecisionRecord
	if err := validateSubscriberQuotaRequest(request); err != nil {
		return reservation, decision, err
	}
	request.TenantID = strings.TrimSpace(request.TenantID)
	request.Subject = strings.TrimSpace(request.Subject)
	request.Capability = strings.TrimSpace(request.Capability)
	request.IdempotencyKey = strings.TrimSpace(request.IdempotencyKey)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	if request.RequestedAt.IsZero() {
		request.RequestedAt = time.Now().UTC()
	}

	err := r.WithSubscriberTenantScope(ctx, request.TenantID, func(ctx context.Context, tx *sql.Tx) error {
		var entitlement storage.SubscriberEntitlementRecord
		err := tx.QueryRowContext(ctx, `
SELECT tenant_id, provisioning_version, product_tier, status, provisioned_by, correlation_id, created_at, updated_at
FROM subscriber_tenant_entitlements WHERE tenant_id=$1 FOR UPDATE`, request.TenantID,
		).Scan(&entitlement.TenantID, &entitlement.ProvisioningVersion, &entitlement.ProductTier, &entitlement.Status, &entitlement.ProvisionedBy, &entitlement.CorrelationID, &entitlement.CreatedAt, &entitlement.UpdatedAt)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("lock entitlement: %w", err)
		}

		if err == nil {
			if existing, found, err := subscriberQuotaReservationByKey(ctx, tx, request.TenantID, request.Capability, entitlement.ProvisioningVersion, request.IdempotencyKey); err != nil {
				return err
			} else if found {
				reservation = existing
				decision = subscriberDecisionRecord(policy.Decision{
					Allowed: true, Reason: policy.DecisionAllowed, TenantID: request.TenantID, Subject: request.Subject,
					Capability: policy.Capability(request.Capability), RequestedUnits: request.RequestedUnits,
					EntitlementVersion: entitlement.ProvisioningVersion, PolicyVersion: policy.DefaultPolicyVersion,
					CorrelationID: request.CorrelationID, DecidedAt: request.RequestedAt,
				}, existing.ReservationID, request.IdempotencyKey)
				return insertSubscriberDecisionAudit(ctx, tx, decision)
			}
		}

		entitlementPolicy := policy.Entitlement{}
		consumed := 0
		if err == nil && entitlement.Status == storage.SubscriberEntitlementActive {
			var capability storage.SubscriberEntitlementCapabilityRecord
			capErr := tx.QueryRowContext(ctx, `SELECT capability, enabled, quota_limit FROM subscriber_entitlement_capabilities WHERE tenant_id=$1 AND capability=$2 FOR UPDATE`, request.TenantID, request.Capability).Scan(&capability.Capability, &capability.Enabled, &capability.QuotaLimit)
			if capErr != nil && !errors.Is(capErr, sql.ErrNoRows) {
				return fmt.Errorf("lock entitlement capability: %w", capErr)
			}
			if capErr == nil {
				entitlementPolicy = policy.Entitlement{
					TenantID: request.TenantID, Version: entitlement.ProvisioningVersion,
					EnabledCapabilities: map[policy.Capability]bool{policy.Capability(request.Capability): capability.Enabled},
					QuotaLimits:         map[policy.Capability]int{policy.Capability(request.Capability): capability.QuotaLimit},
				}
				if err := tx.QueryRowContext(ctx, `SELECT COALESCE(sum(requested_units), 0) FROM subscriber_quota_reservations WHERE tenant_id=$1 AND capability=$2 AND provisioning_version=$3 AND status IN ('reserved','consumed')`, request.TenantID, request.Capability, entitlement.ProvisioningVersion).Scan(&consumed); err != nil {
					return fmt.Errorf("sum reserved quota: %w", err)
				}
			}
		}

		policyDecision := policy.Evaluate(entitlementPolicy, policy.Request{
			TenantID: request.TenantID, Subject: request.Subject, Capability: policy.Capability(request.Capability),
			RequestedUnits: request.RequestedUnits, CorrelationID: request.CorrelationID, RequestedAt: request.RequestedAt,
		}, consumed)
		reservationID := ""
		if policyDecision.Allowed {
			reservationID = newSubscriberID("subquota")
			reservation = storage.SubscriberQuotaReservationRecord{
				ReservationID: reservationID, TenantID: request.TenantID, Capability: request.Capability,
				ProvisioningVersion: entitlement.ProvisioningVersion, IdempotencyKey: request.IdempotencyKey,
				Subject: request.Subject, RequestedUnits: request.RequestedUnits, Status: storage.SubscriberQuotaReserved,
				PolicyVersion: policyDecision.PolicyVersion, CorrelationID: request.CorrelationID, ReservedAt: request.RequestedAt,
			}
			if _, err := tx.ExecContext(ctx, `
INSERT INTO subscriber_quota_reservations
  (reservation_id, tenant_id, capability, provisioning_version, idempotency_key, subject, requested_units, status, policy_version, correlation_id, reserved_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
				reservation.ReservationID, reservation.TenantID, reservation.Capability, reservation.ProvisioningVersion,
				reservation.IdempotencyKey, reservation.Subject, reservation.RequestedUnits, reservation.Status,
				reservation.PolicyVersion, reservation.CorrelationID, reservation.ReservedAt,
			); err != nil {
				return fmt.Errorf("reserve subscriber quota: %w", err)
			}
		}
		decision = subscriberDecisionRecord(policyDecision, reservationID, request.IdempotencyKey)
		if err := insertSubscriberDecisionAudit(ctx, tx, decision); err != nil {
			return err
		}
		return nil
	})
	return reservation, decision, err
}

func subscriberQuotaReservationByKey(ctx context.Context, tx *sql.Tx, tenantID, capability, version, key string) (storage.SubscriberQuotaReservationRecord, bool, error) {
	var record storage.SubscriberQuotaReservationRecord
	err := tx.QueryRowContext(ctx, `
SELECT reservation_id, tenant_id, capability, provisioning_version, idempotency_key, subject, requested_units, status, policy_version, correlation_id, reserved_at, released_at
FROM subscriber_quota_reservations
WHERE tenant_id=$1 AND capability=$2 AND provisioning_version=$3 AND idempotency_key=$4`,
		tenantID, capability, version, key,
	).Scan(&record.ReservationID, &record.TenantID, &record.Capability, &record.ProvisioningVersion, &record.IdempotencyKey, &record.Subject, &record.RequestedUnits, &record.Status, &record.PolicyVersion, &record.CorrelationID, &record.ReservedAt, &record.ReleasedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return record, false, nil
	}
	if err != nil {
		return record, false, fmt.Errorf("get quota reservation: %w", err)
	}
	return record, true, nil
}

func subscriberDecisionRecord(value policy.Decision, reservationID, idempotencyKey string) storage.SubscriberEntitlementDecisionRecord {
	provenance, _ := json.Marshal(map[string]string{"reservation_id": reservationID, "idempotency_key": idempotencyKey})
	return storage.SubscriberEntitlementDecisionRecord{
		DecisionID: newSubscriberID("subentdec"), TenantID: value.TenantID, Subject: value.Subject,
		Capability: string(value.Capability), DecisionReason: string(value.Reason), RequestedUnits: value.RequestedUnits,
		ConsumedUnits: value.ConsumedUnits, QuotaLimit: value.QuotaLimit, EntitlementVersion: value.EntitlementVersion,
		PolicyVersion: value.PolicyVersion, CorrelationID: value.CorrelationID, DecisionAt: value.DecidedAt,
		ProvenanceJSON: provenance,
	}
}

func insertSubscriberDecisionAudit(ctx context.Context, tx *sql.Tx, decision storage.SubscriberEntitlementDecisionRecord) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO subscriber_entitlement_decision_audit
  (decision_id, tenant_id, subject, capability, decision_reason, requested_units, consumed_units, quota_limit, entitlement_version, policy_version, correlation_id, decision_at, provenance)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13::jsonb)`,
		decision.DecisionID, decision.TenantID, decision.Subject, decision.Capability, decision.DecisionReason,
		decision.RequestedUnits, decision.ConsumedUnits, decision.QuotaLimit, decision.EntitlementVersion,
		decision.PolicyVersion, decision.CorrelationID, decision.DecisionAt, string(decision.ProvenanceJSON),
	)
	if err != nil {
		return fmt.Errorf("audit subscriber entitlement decision: %w", err)
	}
	return nil
}

func validateSubscriberEntitlement(record storage.SubscriberEntitlementRecord) error {
	if !validSubscriberTenantID(record.TenantID) || strings.TrimSpace(record.ProvisioningVersion) == "" || strings.TrimSpace(record.ProvisionedBy) == "" {
		return fmt.Errorf("invalid subscriber entitlement")
	}
	if record.Status != storage.SubscriberEntitlementActive && record.Status != storage.SubscriberEntitlementSuspended {
		return fmt.Errorf("invalid subscriber entitlement status")
	}
	seen := map[string]struct{}{}
	for _, capability := range record.Capabilities {
		if !validSubscriberCapability(capability.Capability) || capability.QuotaLimit < 0 {
			return fmt.Errorf("invalid subscriber entitlement capability")
		}
		if _, ok := seen[capability.Capability]; ok {
			return fmt.Errorf("duplicate subscriber entitlement capability")
		}
		seen[capability.Capability] = struct{}{}
	}
	return nil
}

func validateSubscriberQuotaRequest(request storage.SubscriberQuotaReservationRequest) error {
	if !validSubscriberTenantID(request.TenantID) || strings.TrimSpace(request.Subject) == "" ||
		!validSubscriberCapability(request.Capability) || request.RequestedUnits <= 0 || strings.TrimSpace(request.IdempotencyKey) == "" {
		return fmt.Errorf("invalid subscriber quota reservation request")
	}
	return nil
}

func validSubscriberCapability(capability string) bool {
	switch policy.Capability(strings.TrimSpace(capability)) {
	case policy.CapabilityCatalogSearch, policy.CapabilityEODActivation, policy.CapabilityOptionsDemand:
		return true
	default:
		return false
	}
}

func newSubscriberID(prefix string) string {
	var bytes [12]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return prefix + "_" + fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(bytes[:])
}

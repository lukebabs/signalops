package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) UpsertSubscriberSubjectSubscription(ctx context.Context, input storage.SubscriberSubjectSubscriptionMutation) error {
	if err := validSubscriberSubjectSubscriptionMutation(input); err != nil {
		return err
	}
	return r.WithSubscriberTenantScope(ctx, input.TenantID, func(ctx context.Context, tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
INSERT INTO subscriber_subject_subscriptions
  (subscription_id, tenant_id, subject, product_key, status, trial_ends_at, provisioned_by, correlation_id)
SELECT $1, $2, $3, $4, $5,
  CASE WHEN $5='trialing' THEN now() + make_interval(days => p.trial_days) ELSE NULL END,
  $6, $7
FROM subscriber_subscription_products p
WHERE p.product_key=$4 AND p.billing_scope='subject' AND p.active=true
ON CONFLICT (tenant_id, subject) DO UPDATE SET
  product_key=EXCLUDED.product_key, status=EXCLUDED.status,
  trial_ends_at=EXCLUDED.trial_ends_at, current_period_ends_at=NULL,
  grace_ends_at=NULL, canceled_at=CASE WHEN EXCLUDED.status='canceled' THEN now() ELSE NULL END,
  provisioned_by=EXCLUDED.provisioned_by, correlation_id=EXCLUDED.correlation_id, updated_at=now()`,
			newSubscriberID("subjsub"), input.TenantID, input.Subject, input.ProductKey, input.Status, input.ActorSubject, input.CorrelationID)
		if err != nil {
			return fmt.Errorf("upsert subject subscription: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rows != 1 {
			return storage.ErrNotFound
		}
		return insertSubscriberSubscriptionAudit(ctx, tx, input.TenantID, input.Subject, input.ActorSubject, "subject_subscription_upserted", input.CorrelationID,
			fmt.Sprintf(`{"product_key":%q,"status":%q}`, input.ProductKey, input.Status))
	})
}

func (r *Repository) UpsertSubscriberTenantSubscription(ctx context.Context, input storage.SubscriberTenantSubscriptionMutation) error {
	if err := validSubscriberTenantSubscriptionMutation(input); err != nil {
		return err
	}
	return r.WithSubscriberTenantScope(ctx, input.TenantID, func(ctx context.Context, tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
INSERT INTO subscriber_tenant_subscriptions
  (subscription_id, tenant_id, product_key, status, provisioned_by, correlation_id)
SELECT $1, $2, $3, $4, $5, $6
FROM subscriber_subscription_products p
WHERE p.product_key=$3 AND p.billing_scope='tenant' AND p.active=true
ON CONFLICT (tenant_id) DO UPDATE SET
  product_key=EXCLUDED.product_key, status=EXCLUDED.status,
  current_period_ends_at=NULL, grace_ends_at=NULL,
  canceled_at=CASE WHEN EXCLUDED.status='canceled' THEN now() ELSE NULL END,
  provisioned_by=EXCLUDED.provisioned_by, correlation_id=EXCLUDED.correlation_id, updated_at=now()`,
			newSubscriberID("tensub"), input.TenantID, input.ProductKey, input.Status, input.ActorSubject, input.CorrelationID)
		if err != nil {
			return fmt.Errorf("upsert tenant subscription: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rows != 1 {
			return storage.ErrNotFound
		}
		return insertSubscriberSubscriptionAudit(ctx, tx, input.TenantID, "", input.ActorSubject, "tenant_subscription_upserted", input.CorrelationID,
			fmt.Sprintf(`{"product_key":%q,"status":%q}`, input.ProductKey, input.Status))
	})
}

func (r *Repository) UpsertSubscriberSubscriptionSeat(ctx context.Context, input storage.SubscriberSubscriptionSeatMutation) error {
	if err := validSubscriberSubscriptionSeatMutation(input); err != nil {
		return err
	}
	return r.WithSubscriberTenantScope(ctx, input.TenantID, func(ctx context.Context, tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
INSERT INTO subscriber_subscription_seats
  (tenant_id, subject, tenant_subscription_id, seat_role, status, assigned_by, correlation_id, assigned_at, revoked_at)
SELECT $1, $2, s.subscription_id, $3, $4, $5, $6, now(), CASE WHEN $4='revoked' THEN now() ELSE NULL END
FROM subscriber_tenant_subscriptions s
JOIN subscriber_subscription_products p ON p.product_key=s.product_key
WHERE s.tenant_id=$1 AND p.billing_scope='tenant'
ON CONFLICT (tenant_id, subject) DO UPDATE SET
  tenant_subscription_id=EXCLUDED.tenant_subscription_id, seat_role=EXCLUDED.seat_role,
  status=EXCLUDED.status, assigned_by=EXCLUDED.assigned_by, correlation_id=EXCLUDED.correlation_id,
  assigned_at=CASE WHEN EXCLUDED.status='active' THEN now() ELSE subscriber_subscription_seats.assigned_at END,
  revoked_at=CASE WHEN EXCLUDED.status='revoked' THEN now() ELSE NULL END`,
			input.TenantID, input.Subject, input.SeatRole, input.Status, input.ActorSubject, input.CorrelationID)
		if err != nil {
			return fmt.Errorf("upsert subscription seat: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rows != 1 {
			return storage.ErrNotFound
		}
		return insertSubscriberSubscriptionAudit(ctx, tx, input.TenantID, input.Subject, input.ActorSubject, "tenant_subscription_seat_upserted", input.CorrelationID,
			fmt.Sprintf(`{"seat_role":%q,"status":%q}`, input.SeatRole, input.Status))
	})
}

func insertSubscriberSubscriptionAudit(ctx context.Context, tx *sql.Tx, tenantID, subject, actor, eventType, correlationID, afterState string) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO subscriber_subscription_audit_events
  (audit_id, tenant_id, subject, actor_subject, event_type, after_state, correlation_id)
VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7)`,
		newSubscriberID("subaudit"), tenantID, subject, actor, eventType, afterState, correlationID)
	if err != nil {
		return fmt.Errorf("insert subscription audit: %w", err)
	}
	return nil
}

func validSubscriberSubjectSubscriptionMutation(input storage.SubscriberSubjectSubscriptionMutation) error {
	if !validSubscriberTenantID(input.TenantID) || strings.TrimSpace(input.Subject) == "" || strings.TrimSpace(input.ActorSubject) == "" {
		return errors.New("invalid subject subscription scope")
	}
	if input.ProductKey != "explorer" && input.ProductKey != "professional" {
		return errors.New("subject subscription must use explorer or professional")
	}
	return validSubscriberSubscriptionStatus(input.Status)
}
func validSubscriberTenantSubscriptionMutation(input storage.SubscriberTenantSubscriptionMutation) error {
	if !validSubscriberTenantID(input.TenantID) || strings.TrimSpace(input.ActorSubject) == "" {
		return errors.New("invalid tenant subscription scope")
	}
	if input.ProductKey != "institutional" {
		return errors.New("tenant subscription must use institutional")
	}
	return validSubscriberSubscriptionStatus(input.Status)
}
func validSubscriberSubscriptionSeatMutation(input storage.SubscriberSubscriptionSeatMutation) error {
	if !validSubscriberTenantID(input.TenantID) || strings.TrimSpace(input.Subject) == "" || strings.TrimSpace(input.ActorSubject) == "" {
		return errors.New("invalid subscription seat scope")
	}
	if input.SeatRole != "member" && input.SeatRole != "tenant_admin" {
		return errors.New("invalid subscription seat role")
	}
	if input.Status != "active" && input.Status != "revoked" {
		return errors.New("invalid subscription seat status")
	}
	return nil
}
func validSubscriberSubscriptionStatus(status string) error {
	switch strings.TrimSpace(status) {
	case storage.SubscriberSubscriptionTrialing, storage.SubscriberSubscriptionActive, storage.SubscriberSubscriptionPastDue, storage.SubscriberSubscriptionSuspended, storage.SubscriberSubscriptionCanceled:
		return nil
	default:
		return errors.New("invalid subscription status")
	}
}

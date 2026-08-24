package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) ListSubscriberSubscriptionAdministration(ctx context.Context, filter storage.SubscriberSubscriptionAdministrationFilter) (storage.SubscriberSubscriptionAdministrationSnapshot, error) {
	tenantID := strings.TrimSpace(filter.TenantID)
	if !validSubscriberTenantID(tenantID) {
		return storage.SubscriberSubscriptionAdministrationSnapshot{}, errors.New("invalid subscription administration tenant")
	}
	var snapshot storage.SubscriberSubscriptionAdministrationSnapshot
	snapshot.TenantID = tenantID
	err := r.WithSubscriberTenantScope(ctx, tenantID, func(ctx context.Context, tx *sql.Tx) error {
		products, err := listSubscriberSubscriptionProductsTx(ctx, tx, false)
		if err != nil {
			return err
		}
		snapshot.Products = products
		subjects, err := listSubscriberSubjectSubscriptionsTx(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		snapshot.SubjectSubscriptions = subjects
		contracts, err := listSubscriberTenantSubscriptionsTx(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		snapshot.TenantSubscriptions = contracts
		seats, err := listSubscriberSubscriptionSeatsTx(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		snapshot.Seats = seats
		audit, err := listSubscriberSubscriptionAuditEventsTx(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		snapshot.AuditEvents = audit
		webhooks, err := listSubscriberBillingWebhookEventsTx(ctx, tx)
		if err != nil {
			return err
		}
		snapshot.BillingWebhookEvents = webhooks
		upgradeInteractions, err := listSubscriberUpgradeInteractionsTx(ctx, tx, tenantID, 100)
		if err != nil {
			return err
		}
		snapshot.UpgradeInteractions = upgradeInteractions
		return nil
	})
	return snapshot, err
}

func (r *Repository) UpdateSubscriberSubscriptionProduct(ctx context.Context, input storage.SubscriberSubscriptionProductMutation) error {
	if err := validSubscriberSubscriptionProductMutation(input); err != nil {
		return err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	auditTenantID := strings.TrimSpace(input.TenantID)
	if !validSubscriberTenantID(auditTenantID) {
		return errors.New("invalid subscription product audit tenant")
	}
	if _, err := tx.ExecContext(ctx, `SELECT set_config('signalops.tenant_id', $1, true)`, auditTenantID); err != nil {
		return fmt.Errorf("set subscription product audit scope: %w", err)
	}
	var before []byte
	if err := tx.QueryRowContext(ctx, `
SELECT to_jsonb(p)::text::jsonb
FROM subscriber_subscription_products p
WHERE product_key=$1`, input.ProductKey).Scan(&before); errors.Is(err, sql.ErrNoRows) {
		return storage.ErrNotFound
	} else if err != nil {
		return fmt.Errorf("read subscription product: %w", err)
	}
	result, err := tx.ExecContext(ctx, `
UPDATE subscriber_subscription_products
SET display_name=$2, is_free=$3, trial_days=$4,
  feature_policy=$5::jsonb, limit_policy=$6::jsonb, active=$7,
  changed_by=$8, revision=revision+1, updated_at=now()
WHERE product_key=$1`, input.ProductKey, input.DisplayName, input.IsFree, input.TrialDays, string(input.FeaturePolicyJSON), string(input.LimitPolicyJSON), input.Active, input.ActorSubject)
	if err != nil {
		return fmt.Errorf("update subscription product: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return storage.ErrNotFound
	}
	var after []byte
	if err := tx.QueryRowContext(ctx, `SELECT to_jsonb(p)::text::jsonb FROM subscriber_subscription_products p WHERE product_key=$1`, input.ProductKey).Scan(&after); err != nil {
		return fmt.Errorf("read updated subscription product: %w", err)
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO subscriber_subscription_audit_events
  (audit_id, tenant_id, subject, actor_subject, event_type, before_state, after_state, correlation_id)
VALUES ($1, $2, '', $3, 'subscription_product_policy_updated', $4::jsonb, $5::jsonb, $6)`,
		newSubscriberID("subaudit"), auditTenantID, input.ActorSubject, string(before), string(after), input.CorrelationID)
	if err != nil {
		return fmt.Errorf("insert subscription product audit: %w", err)
	}
	return tx.Commit()
}

func (r *Repository) UpdateSubscriberSubscriptionProductBilling(ctx context.Context, input storage.SubscriberSubscriptionProductBillingMutation) error {
	if err := validSubscriberSubscriptionProductBillingMutation(input); err != nil {
		return err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT set_config('signalops.tenant_id', $1, true)`, input.TenantID); err != nil {
		return fmt.Errorf("set subscription billing audit scope: %w", err)
	}
	var before []byte
	if err := tx.QueryRowContext(ctx, `SELECT to_jsonb(p)::text::jsonb FROM subscriber_subscription_products p WHERE product_key=$1`, input.ProductKey).Scan(&before); errors.Is(err, sql.ErrNoRows) {
		return storage.ErrNotFound
	} else if err != nil {
		return fmt.Errorf("read subscription product billing: %w", err)
	}
	result, err := tx.ExecContext(ctx, `
UPDATE subscriber_subscription_products
SET stripe_product_id=$2, stripe_monthly_price_id=$3, stripe_annual_price_id=$4,
  changed_by=$5, revision=revision+1, updated_at=now()
WHERE product_key=$1`, input.ProductKey, input.StripeProductID, input.StripeMonthlyPriceID, input.StripeAnnualPriceID, input.ActorSubject)
	if err != nil {
		return fmt.Errorf("update subscription product billing: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return storage.ErrNotFound
	}
	var after []byte
	if err := tx.QueryRowContext(ctx, `SELECT to_jsonb(p)::text::jsonb FROM subscriber_subscription_products p WHERE product_key=$1`, input.ProductKey).Scan(&after); err != nil {
		return fmt.Errorf("read updated subscription product billing: %w", err)
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO subscriber_subscription_audit_events
  (audit_id, tenant_id, subject, actor_subject, event_type, before_state, after_state, correlation_id)
VALUES ($1, $2, '', $3, 'subscription_product_billing_updated', $4::jsonb, $5::jsonb, $6)`,
		newSubscriberID("subaudit"), input.TenantID, input.ActorSubject, string(before), string(after), input.CorrelationID)
	if err != nil {
		return fmt.Errorf("insert subscription product billing audit: %w", err)
	}
	return tx.Commit()
}

func listSubscriberSubscriptionProductsTx(ctx context.Context, tx *sql.Tx, activeOnly bool) ([]storage.SubscriberSubscriptionProductRecord, error) {
	query := `
SELECT product_key, billing_scope, display_name, is_free, trial_days,
  stripe_product_id, stripe_monthly_price_id, stripe_annual_price_id,
  feature_policy, limit_policy, revision, active, changed_by, created_at, updated_at
FROM subscriber_subscription_products`
	if activeOnly {
		query += ` WHERE active=true`
	}
	query += ` ORDER BY CASE product_key WHEN 'explorer' THEN 1 WHEN 'professional' THEN 2 WHEN 'institutional' THEN 3 ELSE 99 END`
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list subscription products: %w", err)
	}
	defer rows.Close()
	products := []storage.SubscriberSubscriptionProductRecord{}
	for rows.Next() {
		var product storage.SubscriberSubscriptionProductRecord
		if err := scanSubscriberSubscriptionProduct(rows, &product); err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	return products, rows.Err()
}

func listSubscriberSubjectSubscriptionsTx(ctx context.Context, tx *sql.Tx, tenantID string) ([]storage.SubscriberSubjectSubscriptionRecord, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT s.tenant_id, s.subject, COALESCE(subject_identity.display_name, '') AS subject_display_name, COALESCE(subject_identity.email, '') AS subject_email,
  s.subscription_id, s.product_key, p.display_name, s.status,
  s.trial_ends_at, s.current_period_ends_at, s.grace_ends_at, s.canceled_at,
  s.stripe_customer_id, s.stripe_subscription_id,
  s.provisioned_by, s.correlation_id, s.created_at, s.updated_at
FROM subscriber_subject_subscriptions s
JOIN subscriber_subscription_products p ON p.product_key=s.product_key
LEFT JOIN subscriber_subscription_admin_identity_labels($1) subject_identity ON subject_identity.subject=s.subject
WHERE s.tenant_id=$1
ORDER BY s.updated_at DESC, s.subject`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list subject subscriptions: %w", err)
	}
	defer rows.Close()
	records := []storage.SubscriberSubjectSubscriptionRecord{}
	for rows.Next() {
		var record storage.SubscriberSubjectSubscriptionRecord
		if err := rows.Scan(&record.TenantID, &record.Subject, &record.SubjectDisplayName, &record.SubjectEmail, &record.SubscriptionID, &record.ProductKey, &record.DisplayName, &record.Status, &record.TrialEndsAt, &record.CurrentPeriodEndsAt, &record.GraceEndsAt, &record.CanceledAt, &record.StripeCustomerID, &record.StripeSubscriptionID, &record.ProvisionedBy, &record.CorrelationID, &record.CreatedAt, &record.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan subject subscription: %w", err)
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func listSubscriberTenantSubscriptionsTx(ctx context.Context, tx *sql.Tx, tenantID string) ([]storage.SubscriberTenantSubscriptionRecord, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT s.tenant_id, s.subscription_id, s.product_key, p.display_name, s.status,
  s.current_period_ends_at, s.grace_ends_at, s.canceled_at,
  s.stripe_customer_id, s.stripe_subscription_id,
  s.provisioned_by, s.correlation_id, s.created_at, s.updated_at
FROM subscriber_tenant_subscriptions s
JOIN subscriber_subscription_products p ON p.product_key=s.product_key
WHERE s.tenant_id=$1
ORDER BY s.updated_at DESC`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list tenant subscriptions: %w", err)
	}
	defer rows.Close()
	records := []storage.SubscriberTenantSubscriptionRecord{}
	for rows.Next() {
		var record storage.SubscriberTenantSubscriptionRecord
		if err := rows.Scan(&record.TenantID, &record.SubscriptionID, &record.ProductKey, &record.DisplayName, &record.Status, &record.CurrentPeriodEndsAt, &record.GraceEndsAt, &record.CanceledAt, &record.StripeCustomerID, &record.StripeSubscriptionID, &record.ProvisionedBy, &record.CorrelationID, &record.CreatedAt, &record.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan tenant subscription: %w", err)
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func listSubscriberSubscriptionSeatsTx(ctx context.Context, tx *sql.Tx, tenantID string) ([]storage.SubscriberSubscriptionSeatRecord, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT s.tenant_id, s.subject, COALESCE(subject_identity.display_name, '') AS subject_display_name, COALESCE(subject_identity.email, '') AS subject_email,
  s.tenant_subscription_id, s.seat_role, s.status, s.assigned_by,
  COALESCE(actor_identity.display_name, '') AS assigned_by_display_name, COALESCE(actor_identity.email, '') AS assigned_by_email,
  s.correlation_id, s.assigned_at, s.revoked_at
FROM subscriber_subscription_seats s
LEFT JOIN subscriber_subscription_admin_identity_labels($1) subject_identity ON subject_identity.subject=s.subject
LEFT JOIN subscriber_subscription_admin_identity_labels($1) actor_identity ON actor_identity.subject=s.assigned_by
WHERE s.tenant_id=$1
ORDER BY s.assigned_at DESC, s.subject`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list subscription seats: %w", err)
	}
	defer rows.Close()
	records := []storage.SubscriberSubscriptionSeatRecord{}
	for rows.Next() {
		var record storage.SubscriberSubscriptionSeatRecord
		if err := rows.Scan(&record.TenantID, &record.Subject, &record.SubjectDisplayName, &record.SubjectEmail, &record.TenantSubscriptionID, &record.SeatRole, &record.Status, &record.AssignedBy, &record.AssignedByDisplayName, &record.AssignedByEmail, &record.CorrelationID, &record.AssignedAt, &record.RevokedAt); err != nil {
			return nil, fmt.Errorf("scan subscription seat: %w", err)
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func listSubscriberSubscriptionAuditEventsTx(ctx context.Context, tx *sql.Tx, tenantID string) ([]storage.SubscriberSubscriptionAuditEventRecord, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT a.audit_id, a.tenant_id, a.subject, COALESCE(subject_identity.display_name, '') AS subject_display_name, COALESCE(subject_identity.email, '') AS subject_email,
  a.subscription_id, a.actor_subject, COALESCE(actor_identity.display_name, '') AS actor_display_name, COALESCE(actor_identity.email, '') AS actor_email,
  a.event_type, COALESCE(a.before_state, '{}'::jsonb), COALESCE(a.after_state, '{}'::jsonb), a.correlation_id, a.occurred_at
FROM subscriber_subscription_audit_events a
LEFT JOIN subscriber_subscription_admin_identity_labels($1) subject_identity ON subject_identity.subject=a.subject
LEFT JOIN subscriber_subscription_admin_identity_labels($1) actor_identity ON actor_identity.subject=a.actor_subject
WHERE a.tenant_id=$1
ORDER BY a.occurred_at DESC
LIMIT 100`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list subscription audit events: %w", err)
	}
	defer rows.Close()
	records := []storage.SubscriberSubscriptionAuditEventRecord{}
	for rows.Next() {
		var record storage.SubscriberSubscriptionAuditEventRecord
		if err := rows.Scan(&record.AuditID, &record.TenantID, &record.Subject, &record.SubjectDisplayName, &record.SubjectEmail, &record.SubscriptionID, &record.ActorSubject, &record.ActorDisplayName, &record.ActorEmail, &record.EventType, &record.BeforeStateJSON, &record.AfterStateJSON, &record.CorrelationID, &record.OccurredAt); err != nil {
			return nil, fmt.Errorf("scan subscription audit event: %w", err)
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func listSubscriberBillingWebhookEventsTx(ctx context.Context, tx *sql.Tx) ([]storage.SubscriberBillingWebhookEventRecord, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT provider_event_id, event_type, processing_status, error_message, received_at, processed_at
FROM subscriber_billing_webhook_events
ORDER BY received_at DESC
LIMIT 100`)
	if err != nil {
		return nil, fmt.Errorf("list billing webhook events: %w", err)
	}
	defer rows.Close()
	records := []storage.SubscriberBillingWebhookEventRecord{}
	for rows.Next() {
		var record storage.SubscriberBillingWebhookEventRecord
		if err := rows.Scan(&record.ProviderEventID, &record.EventType, &record.ProcessingStatus, &record.ErrorMessage, &record.ReceivedAt, &record.ProcessedAt); err != nil {
			return nil, fmt.Errorf("scan billing webhook event: %w", err)
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

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

func (r *Repository) UpdateSubscriberSubjectSubscriptionBilling(ctx context.Context, input storage.SubscriberSubjectSubscriptionBillingMutation) error {
	if err := validSubscriberSubjectSubscriptionBillingMutation(input); err != nil {
		return err
	}
	return r.WithSubscriberTenantScope(ctx, input.TenantID, func(ctx context.Context, tx *sql.Tx) error {
		var before []byte
		if err := tx.QueryRowContext(ctx, `SELECT to_jsonb(s)::text::jsonb FROM subscriber_subject_subscriptions s WHERE tenant_id=$1 AND subject=$2`, input.TenantID, input.Subject).Scan(&before); errors.Is(err, sql.ErrNoRows) {
			return storage.ErrNotFound
		} else if err != nil {
			return fmt.Errorf("read subject subscription billing: %w", err)
		}
		result, err := tx.ExecContext(ctx, `
UPDATE subscriber_subject_subscriptions
SET stripe_customer_id=$3, stripe_subscription_id=$4, status=$5,
  current_period_ends_at=$6, grace_ends_at=$7, canceled_at=$8,
  provisioned_by=$9, correlation_id=$10, updated_at=now()
WHERE tenant_id=$1 AND subject=$2`, input.TenantID, input.Subject, input.StripeCustomerID, input.StripeSubscriptionID, input.Status, input.CurrentPeriodEndsAt, input.GraceEndsAt, input.CanceledAt, input.ActorSubject, input.CorrelationID)
		if err != nil {
			return fmt.Errorf("update subject subscription billing: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rows != 1 {
			return storage.ErrNotFound
		}
		var after []byte
		if err := tx.QueryRowContext(ctx, `SELECT to_jsonb(s)::text::jsonb FROM subscriber_subject_subscriptions s WHERE tenant_id=$1 AND subject=$2`, input.TenantID, input.Subject).Scan(&after); err != nil {
			return fmt.Errorf("read updated subject subscription billing: %w", err)
		}
		_, err = tx.ExecContext(ctx, `
INSERT INTO subscriber_subscription_audit_events
  (audit_id, tenant_id, subject, subscription_id, actor_subject, event_type, before_state, after_state, correlation_id)
SELECT $1, tenant_id, subject, subscription_id, $4, 'subject_subscription_billing_updated', $5::jsonb, $6::jsonb, $7
FROM subscriber_subject_subscriptions WHERE tenant_id=$2 AND subject=$3`, newSubscriberID("subaudit"), input.TenantID, input.Subject, input.ActorSubject, string(before), string(after), input.CorrelationID)
		if err != nil {
			return fmt.Errorf("insert subject subscription billing audit: %w", err)
		}
		return nil
	})
}

func (r *Repository) UpdateSubscriberTenantSubscriptionBilling(ctx context.Context, input storage.SubscriberTenantSubscriptionBillingMutation) error {
	if err := validSubscriberTenantSubscriptionBillingMutation(input); err != nil {
		return err
	}
	return r.WithSubscriberTenantScope(ctx, input.TenantID, func(ctx context.Context, tx *sql.Tx) error {
		var before []byte
		if err := tx.QueryRowContext(ctx, `SELECT to_jsonb(s)::text::jsonb FROM subscriber_tenant_subscriptions s WHERE tenant_id=$1`, input.TenantID).Scan(&before); errors.Is(err, sql.ErrNoRows) {
			return storage.ErrNotFound
		} else if err != nil {
			return fmt.Errorf("read tenant subscription billing: %w", err)
		}
		result, err := tx.ExecContext(ctx, `
UPDATE subscriber_tenant_subscriptions
SET stripe_customer_id=$2, stripe_subscription_id=$3, status=$4,
  current_period_ends_at=$5, grace_ends_at=$6, canceled_at=$7,
  provisioned_by=$8, correlation_id=$9, updated_at=now()
WHERE tenant_id=$1`, input.TenantID, input.StripeCustomerID, input.StripeSubscriptionID, input.Status, input.CurrentPeriodEndsAt, input.GraceEndsAt, input.CanceledAt, input.ActorSubject, input.CorrelationID)
		if err != nil {
			return fmt.Errorf("update tenant subscription billing: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rows != 1 {
			return storage.ErrNotFound
		}
		var after []byte
		if err := tx.QueryRowContext(ctx, `SELECT to_jsonb(s)::text::jsonb FROM subscriber_tenant_subscriptions s WHERE tenant_id=$1`, input.TenantID).Scan(&after); err != nil {
			return fmt.Errorf("read updated tenant subscription billing: %w", err)
		}
		_, err = tx.ExecContext(ctx, `
INSERT INTO subscriber_subscription_audit_events
  (audit_id, tenant_id, subject, subscription_id, actor_subject, event_type, before_state, after_state, correlation_id)
SELECT $1, tenant_id, '', subscription_id, $3, 'tenant_subscription_billing_updated', $4::jsonb, $5::jsonb, $6
FROM subscriber_tenant_subscriptions WHERE tenant_id=$2`, newSubscriberID("subaudit"), input.TenantID, input.ActorSubject, string(before), string(after), input.CorrelationID)
		if err != nil {
			return fmt.Errorf("insert tenant subscription billing audit: %w", err)
		}
		return nil
	})
}

func (r *Repository) ProcessSubscriberStripeWebhook(ctx context.Context, input storage.SubscriberStripeWebhookMutation) (storage.SubscriberBillingWebhookEventRecord, error) {
	if err := validSubscriberStripeWebhookMutation(input); err != nil {
		return storage.SubscriberBillingWebhookEventRecord{}, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return storage.SubscriberBillingWebhookEventRecord{}, err
	}
	defer tx.Rollback()
	var existing storage.SubscriberBillingWebhookEventRecord
	if err := tx.QueryRowContext(ctx, `SELECT provider_event_id,event_type,processing_status,error_message,received_at,processed_at FROM subscriber_billing_webhook_events WHERE provider_event_id=$1`, input.ProviderEventID).Scan(&existing.ProviderEventID, &existing.EventType, &existing.ProcessingStatus, &existing.ErrorMessage, &existing.ReceivedAt, &existing.ProcessedAt); err == nil {
		return existing, tx.Commit()
	} else if !errors.Is(err, sql.ErrNoRows) {
		return storage.SubscriberBillingWebhookEventRecord{}, fmt.Errorf("read billing webhook event: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO subscriber_billing_webhook_events(provider_event_id,event_type,payload,processing_status) VALUES($1,$2,$3::jsonb,'received')`, input.ProviderEventID, input.EventType, string(input.PayloadJSON)); err != nil {
		return storage.SubscriberBillingWebhookEventRecord{}, fmt.Errorf("record billing webhook event: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `SELECT set_config('signalops.stripe_webhook_reconcile', 'true', true)`); err != nil {
		return storage.SubscriberBillingWebhookEventRecord{}, fmt.Errorf("set stripe webhook reconciliation scope: %w", err)
	}
	matched := 0
	if strings.TrimSpace(input.StripeSubscriptionID) != "" {
		result, err := tx.ExecContext(ctx, `
UPDATE subscriber_subject_subscriptions
SET status=$2, stripe_customer_id=COALESCE(NULLIF($3,''), stripe_customer_id),
  current_period_ends_at=$4, grace_ends_at=$5, canceled_at=$6,
  provisioned_by='stripe-webhook', correlation_id=$7, updated_at=now()
WHERE stripe_subscription_id=$1`, input.StripeSubscriptionID, input.Status, input.StripeCustomerID, input.CurrentPeriodEndsAt, input.GraceEndsAt, input.CanceledAt, input.ProviderEventID)
		if err != nil {
			return r.failSubscriberStripeWebhook(ctx, tx, input, fmt.Errorf("update subject stripe subscription: %w", err))
		}
		rows, _ := result.RowsAffected()
		matched += int(rows)
		if rows > 0 {
			if err := r.insertSubscriberStripeWebhookSubjectAudits(ctx, tx, input); err != nil {
				return r.failSubscriberStripeWebhook(ctx, tx, input, fmt.Errorf("insert subject stripe audit: %w", err))
			}
		}
		result, err = tx.ExecContext(ctx, `
UPDATE subscriber_tenant_subscriptions
SET status=$2, stripe_customer_id=COALESCE(NULLIF($3,''), stripe_customer_id),
  current_period_ends_at=$4, grace_ends_at=$5, canceled_at=$6,
  provisioned_by='stripe-webhook', correlation_id=$7, updated_at=now()
WHERE stripe_subscription_id=$1`, input.StripeSubscriptionID, input.Status, input.StripeCustomerID, input.CurrentPeriodEndsAt, input.GraceEndsAt, input.CanceledAt, input.ProviderEventID)
		if err != nil {
			return r.failSubscriberStripeWebhook(ctx, tx, input, fmt.Errorf("update tenant stripe subscription: %w", err))
		}
		rows, _ = result.RowsAffected()
		matched += int(rows)
		if rows > 0 {
			if err := r.insertSubscriberStripeWebhookTenantAudits(ctx, tx, input); err != nil {
				return r.failSubscriberStripeWebhook(ctx, tx, input, fmt.Errorf("insert tenant stripe audit: %w", err))
			}
		}
	}
	status := "processed"
	if matched == 0 {
		status = "unmatched"
	}
	if _, err := tx.ExecContext(ctx, `UPDATE subscriber_billing_webhook_events SET processing_status=$2, processed_at=now(), error_message='' WHERE provider_event_id=$1`, input.ProviderEventID, status); err != nil {
		return storage.SubscriberBillingWebhookEventRecord{}, fmt.Errorf("update billing webhook event status: %w", err)
	}
	var record storage.SubscriberBillingWebhookEventRecord
	if err := tx.QueryRowContext(ctx, `SELECT provider_event_id,event_type,processing_status,error_message,received_at,processed_at FROM subscriber_billing_webhook_events WHERE provider_event_id=$1`, input.ProviderEventID).Scan(&record.ProviderEventID, &record.EventType, &record.ProcessingStatus, &record.ErrorMessage, &record.ReceivedAt, &record.ProcessedAt); err != nil {
		return storage.SubscriberBillingWebhookEventRecord{}, fmt.Errorf("read processed billing webhook event: %w", err)
	}
	return record, tx.Commit()
}

func (r *Repository) failSubscriberStripeWebhook(ctx context.Context, tx *sql.Tx, input storage.SubscriberStripeWebhookMutation, cause error) (storage.SubscriberBillingWebhookEventRecord, error) {
	_, _ = tx.ExecContext(ctx, `UPDATE subscriber_billing_webhook_events SET processing_status='failed', processed_at=now(), error_message=$2 WHERE provider_event_id=$1`, input.ProviderEventID, cause.Error())
	return storage.SubscriberBillingWebhookEventRecord{}, cause
}

func (r *Repository) insertSubscriberStripeWebhookSubjectAudits(ctx context.Context, tx *sql.Tx, input storage.SubscriberStripeWebhookMutation) error {
	tenantIDs, err := subscriberStripeWebhookAuditTenantIDs(ctx, tx, `SELECT tenant_id FROM subscriber_subject_subscriptions WHERE stripe_subscription_id=$1`, input.StripeSubscriptionID)
	if err != nil {
		return err
	}
	for _, tenantID := range tenantIDs {
		if _, err := tx.ExecContext(ctx, `SELECT set_config('signalops.tenant_id', $1, true)`, tenantID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO subscriber_subscription_audit_events(audit_id, tenant_id, subject, subscription_id, actor_subject, event_type, after_state, correlation_id) SELECT $1, tenant_id, subject, subscription_id, 'stripe-webhook', 'stripe_subscription_reconciled', jsonb_build_object('event_type',$2::text,'status',$3::text,'stripe_subscription_id',$4::text), $5 FROM subscriber_subject_subscriptions WHERE stripe_subscription_id=$4::text AND tenant_id=$6::text`, newSubscriberID("subaudit"), input.EventType, input.Status, input.StripeSubscriptionID, input.ProviderEventID, tenantID); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) insertSubscriberStripeWebhookTenantAudits(ctx context.Context, tx *sql.Tx, input storage.SubscriberStripeWebhookMutation) error {
	tenantIDs, err := subscriberStripeWebhookAuditTenantIDs(ctx, tx, `SELECT tenant_id FROM subscriber_tenant_subscriptions WHERE stripe_subscription_id=$1`, input.StripeSubscriptionID)
	if err != nil {
		return err
	}
	for _, tenantID := range tenantIDs {
		if _, err := tx.ExecContext(ctx, `SELECT set_config('signalops.tenant_id', $1, true)`, tenantID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO subscriber_subscription_audit_events(audit_id, tenant_id, subject, subscription_id, actor_subject, event_type, after_state, correlation_id) SELECT $1, tenant_id, '', subscription_id, 'stripe-webhook', 'stripe_subscription_reconciled', jsonb_build_object('event_type',$2::text,'status',$3::text,'stripe_subscription_id',$4::text), $5 FROM subscriber_tenant_subscriptions WHERE stripe_subscription_id=$4::text AND tenant_id=$6::text`, newSubscriberID("subaudit"), input.EventType, input.Status, input.StripeSubscriptionID, input.ProviderEventID, tenantID); err != nil {
			return err
		}
	}
	return nil
}

func subscriberStripeWebhookAuditTenantIDs(ctx context.Context, tx *sql.Tx, query string, args ...any) ([]string, error) {
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tenantIDs := []string{}
	for rows.Next() {
		var tenantID string
		if err := rows.Scan(&tenantID); err != nil {
			return nil, err
		}
		tenantIDs = append(tenantIDs, tenantID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tenantIDs, nil
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

func validSubscriberSubscriptionProductBillingMutation(input storage.SubscriberSubscriptionProductBillingMutation) error {
	if !validSubscriberTenantID(input.TenantID) || strings.TrimSpace(input.ActorSubject) == "" {
		return errors.New("invalid subscription product billing scope")
	}
	if input.ProductKey != "explorer" && input.ProductKey != "professional" && input.ProductKey != "institutional" {
		return errors.New("invalid subscription product")
	}
	return nil
}

func validSubscriberSubjectSubscriptionBillingMutation(input storage.SubscriberSubjectSubscriptionBillingMutation) error {
	if !validSubscriberTenantID(input.TenantID) || strings.TrimSpace(input.Subject) == "" || strings.TrimSpace(input.ActorSubject) == "" {
		return errors.New("invalid subject subscription billing scope")
	}
	if strings.TrimSpace(input.StripeSubscriptionID) == "" || strings.TrimSpace(input.StripeCustomerID) == "" {
		return errors.New("stripe customer and subscription IDs are required")
	}
	return validSubscriberSubscriptionStatus(input.Status)
}

func validSubscriberTenantSubscriptionBillingMutation(input storage.SubscriberTenantSubscriptionBillingMutation) error {
	if !validSubscriberTenantID(input.TenantID) || strings.TrimSpace(input.ActorSubject) == "" {
		return errors.New("invalid tenant subscription billing scope")
	}
	if strings.TrimSpace(input.StripeSubscriptionID) == "" || strings.TrimSpace(input.StripeCustomerID) == "" {
		return errors.New("stripe customer and subscription IDs are required")
	}
	return validSubscriberSubscriptionStatus(input.Status)
}

func validSubscriberStripeWebhookMutation(input storage.SubscriberStripeWebhookMutation) error {
	if strings.TrimSpace(input.ProviderEventID) == "" || strings.TrimSpace(input.EventType) == "" || !jsonObject(input.PayloadJSON) {
		return errors.New("invalid stripe webhook event")
	}
	if strings.TrimSpace(input.StripeSubscriptionID) == "" {
		return nil
	}
	return validSubscriberSubscriptionStatus(input.Status)
}

func validSubscriberSubscriptionProductMutation(input storage.SubscriberSubscriptionProductMutation) error {
	if input.ProductKey != "explorer" && input.ProductKey != "professional" && input.ProductKey != "institutional" {
		return errors.New("invalid subscription product")
	}
	if strings.TrimSpace(input.DisplayName) == "" || strings.TrimSpace(input.ActorSubject) == "" {
		return errors.New("invalid subscription product mutation")
	}
	if input.TrialDays < 0 || input.TrialDays > 31 {
		return errors.New("invalid subscription trial days")
	}
	if !jsonObject(input.FeaturePolicyJSON) || !jsonObject(input.LimitPolicyJSON) {
		return errors.New("invalid subscription policy json")
	}
	return nil
}

func jsonObject(raw []byte) bool {
	var value map[string]any
	return json.Unmarshal(raw, &value) == nil
}

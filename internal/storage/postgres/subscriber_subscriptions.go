package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) ListSubscriberSubscriptionProducts(ctx context.Context) ([]storage.SubscriberSubscriptionProductRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT product_key, billing_scope, display_name, is_free, trial_days,
  stripe_product_id, stripe_monthly_price_id, stripe_annual_price_id, monthly_display_price, annual_display_price,
  feature_policy, limit_policy, revision, active, changed_by, created_at, updated_at
FROM subscriber_subscription_products
WHERE active=true
ORDER BY CASE product_key WHEN 'explorer' THEN 1 WHEN 'professional' THEN 2 WHEN 'institutional' THEN 3 ELSE 99 END`)
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

func (r *Repository) GetSubscriberEffectiveSubscription(ctx context.Context, tenantID, subject string) (storage.SubscriberEffectiveSubscriptionRecord, error) {
	tenantID, subject = strings.TrimSpace(tenantID), strings.TrimSpace(subject)
	if !validSubscriberTenantID(tenantID) || subject == "" {
		return storage.SubscriberEffectiveSubscriptionRecord{}, errors.New("invalid subscriber subscription scope")
	}
	var result storage.SubscriberEffectiveSubscriptionRecord
	err := r.WithSubscriberTenantScope(ctx, tenantID, func(ctx context.Context, tx *sql.Tx) error {
		row := tx.QueryRowContext(ctx, `
SELECT s.tenant_id, s.subject, s.subscription_id, s.status, 'subject' AS source, '' AS seat_role,
  s.trial_ends_at, s.current_period_ends_at, s.grace_ends_at, s.canceled_at,
  p.product_key, p.billing_scope, p.display_name, p.is_free, p.trial_days,
  p.stripe_product_id, p.stripe_monthly_price_id, p.stripe_annual_price_id, p.monthly_display_price, p.annual_display_price,
  p.feature_policy, p.limit_policy, p.revision, p.active, p.changed_by, p.created_at, p.updated_at
FROM subscriber_subject_subscriptions s
JOIN subscriber_subscription_products p ON p.product_key=s.product_key
WHERE s.tenant_id=$1 AND s.subject=$2
  AND (s.status IN ('active','trialing') OR (s.status='past_due' AND s.grace_ends_at > now()))
LIMIT 1`, tenantID, subject)
		if err := scanEffectiveSubscriberSubscription(row, &result); err == nil {
			return nil
		} else if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("read subject subscription: %w", err)
		}

		row = tx.QueryRowContext(ctx, `
SELECT s.tenant_id, seat.subject, s.subscription_id, s.status, 'tenant_seat' AS source, seat.seat_role,
  NULL::timestamptz AS trial_ends_at, s.current_period_ends_at, s.grace_ends_at, s.canceled_at,
  p.product_key, p.billing_scope, p.display_name, p.is_free, p.trial_days,
  p.stripe_product_id, p.stripe_monthly_price_id, p.stripe_annual_price_id, p.monthly_display_price, p.annual_display_price,
  p.feature_policy, p.limit_policy, p.revision, p.active, p.changed_by, p.created_at, p.updated_at
FROM subscriber_tenant_subscriptions s
JOIN subscriber_subscription_seats seat ON seat.tenant_subscription_id=s.subscription_id
JOIN subscriber_subscription_products p ON p.product_key=s.product_key
WHERE s.tenant_id=$1 AND seat.subject=$2 AND seat.status='active'
  AND (s.status IN ('active','trialing') OR (s.status='past_due' AND s.grace_ends_at > now()))
LIMIT 1`, tenantID, subject)
		if err := scanEffectiveSubscriberSubscription(row, &result); errors.Is(err, sql.ErrNoRows) {
			return storage.ErrNotFound
		} else if err != nil {
			return fmt.Errorf("read tenant subscription: %w", err)
		}
		return nil
	})
	return result, err
}

type subscriberSubscriptionScanner interface {
	Scan(...any) error
}

func scanEffectiveSubscriberSubscription(scanner subscriberSubscriptionScanner, result *storage.SubscriberEffectiveSubscriptionRecord) error {
	return scanner.Scan(
		&result.TenantID, &result.Subject, &result.SubscriptionID, &result.Status, &result.Source, &result.SeatRole,
		&result.TrialEndsAt, &result.CurrentPeriodEndsAt, &result.GraceEndsAt, &result.CanceledAt,
		&result.Product.ProductKey, &result.Product.BillingScope, &result.Product.DisplayName, &result.Product.IsFree, &result.Product.TrialDays,
		&result.Product.StripeProductID, &result.Product.StripeMonthlyPriceID, &result.Product.StripeAnnualPriceID, &result.Product.MonthlyDisplayPrice, &result.Product.AnnualDisplayPrice,
		&result.Product.FeaturePolicyJSON, &result.Product.LimitPolicyJSON, &result.Product.Revision, &result.Product.Active,
		&result.Product.ChangedBy, &result.Product.CreatedAt, &result.Product.UpdatedAt,
	)
}

func scanSubscriberSubscriptionProduct(scanner subscriberSubscriptionScanner, product *storage.SubscriberSubscriptionProductRecord) error {
	if err := scanner.Scan(
		&product.ProductKey, &product.BillingScope, &product.DisplayName, &product.IsFree, &product.TrialDays,
		&product.StripeProductID, &product.StripeMonthlyPriceID, &product.StripeAnnualPriceID, &product.MonthlyDisplayPrice, &product.AnnualDisplayPrice,
		&product.FeaturePolicyJSON, &product.LimitPolicyJSON, &product.Revision, &product.Active, &product.ChangedBy,
		&product.CreatedAt, &product.UpdatedAt,
	); err != nil {
		return fmt.Errorf("scan subscription product: %w", err)
	}
	return nil
}

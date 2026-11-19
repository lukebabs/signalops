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

const defaultSubscriberUpgradeInteractionLimit = 100

func (r *Repository) RecordSubscriberUpgradeInteraction(ctx context.Context, input storage.SubscriberUpgradeInteractionInput) error {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.Subject = strings.TrimSpace(input.Subject)
	if input.AppID == "" {
		input.AppID = "marketops"
	}
	input.InteractionType = strings.TrimSpace(input.InteractionType)
	input.SourceFeature = strings.TrimSpace(input.SourceFeature)
	input.SourceRoute = strings.TrimSpace(input.SourceRoute)
	input.SourceURL = strings.TrimSpace(input.SourceURL)
	input.AssetSymbol = strings.ToUpper(strings.TrimSpace(input.AssetSymbol))
	input.CurrentTier = strings.TrimSpace(input.CurrentTier)
	input.RequiredTier = strings.TrimSpace(input.RequiredTier)
	input.PromptVariant = firstNonEmptyUpgradeValue(strings.TrimSpace(input.PromptVariant), "contextual")
	input.CTALabel = strings.TrimSpace(input.CTALabel)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)
	if !validSubscriberTenantID(input.TenantID) {
		return errors.New("invalid subscriber upgrade interaction tenant")
	}
	if input.Subject == "" {
		return errors.New("subscriber upgrade interaction subject is required")
	}
	if input.AppID != "marketops" {
		return errors.New("subscriber upgrade interaction app must be marketops")
	}
	switch input.InteractionType {
	case "prompt_shown", "prompt_clicked", "checkout_started", "contact_sales_clicked":
	default:
		return errors.New("invalid subscriber upgrade interaction type")
	}
	switch input.RequiredTier {
	case "explorer", "professional", "institutional":
	default:
		return errors.New("subscriber upgrade interaction required_tier must be explorer, professional, or institutional")
	}
	metadata := input.MetadataJSON
	if len(metadata) == 0 {
		metadata = []byte(`{}`)
	}
	if !json.Valid(metadata) {
		return errors.New("subscriber upgrade interaction metadata must be valid JSON")
	}
	var metadataObject map[string]any
	if err := json.Unmarshal(metadata, &metadataObject); err != nil {
		return errors.New("subscriber upgrade interaction metadata must be a JSON object")
	}
	return r.WithSubscriberTenantScope(ctx, input.TenantID, func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
INSERT INTO subscriber_upgrade_interactions
  (interaction_id, tenant_id, subject, app_id, interaction_type, source_feature, source_route, source_url,
   asset_symbol, current_tier, required_tier, prompt_variant, cta_label, correlation_id, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15::jsonb)`,
			newSubscriberID("subupg"), input.TenantID, input.Subject, input.AppID, input.InteractionType, input.SourceFeature, input.SourceRoute, input.SourceURL,
			input.AssetSymbol, input.CurrentTier, input.RequiredTier, input.PromptVariant, input.CTALabel, input.CorrelationID, string(metadata))
		if err != nil {
			return fmt.Errorf("insert subscriber upgrade interaction: %w", err)
		}
		return nil
	})
}

func (r *Repository) CreateSubscriberCheckoutSession(ctx context.Context, input storage.SubscriberCheckoutSessionInput) error {
	input.CheckoutRef = strings.TrimSpace(input.CheckoutRef)
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.Subject = strings.TrimSpace(input.Subject)
	input.ProductKey = strings.ToLower(strings.TrimSpace(input.ProductKey))
	input.BillingPeriod = strings.ToLower(strings.TrimSpace(input.BillingPeriod))
	input.StripePriceID = strings.TrimSpace(input.StripePriceID)
	input.StripeSessionID = strings.TrimSpace(input.StripeSessionID)
	input.Status = strings.TrimSpace(input.Status)
	input.ActorSubject = strings.TrimSpace(input.ActorSubject)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)
	if input.CheckoutRef == "" || !validSubscriberTenantID(input.TenantID) || input.Subject == "" || input.ActorSubject == "" {
		return errors.New("invalid subscriber checkout session scope")
	}
	if input.ProductKey != "explorer" && input.ProductKey != "professional" {
		return errors.New("subscriber checkout supports explorer or professional")
	}
	if input.BillingPeriod != "monthly" && input.BillingPeriod != "annual" {
		return errors.New("subscriber checkout billing period must be monthly or annual")
	}
	if input.StripePriceID == "" {
		return errors.New("subscriber checkout requires Stripe price ID")
	}
	if input.Status == "checkout_started" && input.StripeSessionID == "" {
		return errors.New("subscriber checkout_started requires Stripe session ID")
	}
	if input.Status != "created" && input.Status != "checkout_started" && input.Status != "failed" {
		return errors.New("invalid subscriber checkout session status")
	}
	return r.WithSubscriberTenantScope(ctx, input.TenantID, func(ctx context.Context, tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
INSERT INTO subscriber_checkout_sessions
  (checkout_ref, tenant_id, subject, product_key, billing_period, stripe_price_id,
   stripe_session_id, status, checkout_url_returned, actor_subject, correlation_id)
SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
FROM subscriber_subscription_products p
WHERE p.product_key=$4 AND p.billing_scope='subject' AND p.active=true
ON CONFLICT (checkout_ref) DO UPDATE SET
  stripe_session_id=COALESCE(NULLIF(EXCLUDED.stripe_session_id,''), subscriber_checkout_sessions.stripe_session_id),
  status=EXCLUDED.status,
  checkout_url_returned=EXCLUDED.checkout_url_returned,
  correlation_id=EXCLUDED.correlation_id,
  updated_at=now()`,
			input.CheckoutRef, input.TenantID, input.Subject, input.ProductKey, input.BillingPeriod, input.StripePriceID,
			input.StripeSessionID, input.Status, input.CheckoutURLReturned, input.ActorSubject, input.CorrelationID)
		if err != nil {
			return fmt.Errorf("insert subscriber checkout session: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rows != 1 {
			return storage.ErrNotFound
		}
		return insertSubscriberSubscriptionAudit(ctx, tx, input.TenantID, input.Subject, input.ActorSubject, "stripe_checkout_started", input.CorrelationID,
			fmt.Sprintf(`{"checkout_ref":%q,"product_key":%q,"billing_period":%q,"stripe_session_id":%q}`, input.CheckoutRef, input.ProductKey, input.BillingPeriod, input.StripeSessionID))
	})
}

func listSubscriberUpgradeInteractionsTx(ctx context.Context, tx *sql.Tx, tenantID string, limit int) ([]storage.SubscriberUpgradeInteractionRecord, error) {
	if limit <= 0 {
		limit = defaultSubscriberUpgradeInteractionLimit
	}
	rows, err := tx.QueryContext(ctx, `
SELECT u.interaction_id, u.tenant_id, u.subject,
  COALESCE(identity.display_name, '') AS subject_display_name,
  COALESCE(identity.email, '') AS subject_email,
  u.app_id, u.interaction_type, u.source_feature, u.source_route, u.source_url,
  u.asset_symbol, u.current_tier, u.required_tier, u.prompt_variant, u.cta_label,
  u.correlation_id, COALESCE(u.metadata, '{}'::jsonb), u.occurred_at
FROM subscriber_upgrade_interactions u
LEFT JOIN subscriber_subscription_admin_identity_labels($1) identity ON identity.subject=u.subject
WHERE u.tenant_id=$1
ORDER BY u.occurred_at DESC, u.interaction_id
LIMIT $2`, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("list subscriber upgrade interactions: %w", err)
	}
	defer rows.Close()
	records := []storage.SubscriberUpgradeInteractionRecord{}
	for rows.Next() {
		var record storage.SubscriberUpgradeInteractionRecord
		if err := rows.Scan(&record.InteractionID, &record.TenantID, &record.Subject, &record.SubjectDisplayName, &record.SubjectEmail, &record.AppID, &record.InteractionType, &record.SourceFeature, &record.SourceRoute, &record.SourceURL, &record.AssetSymbol, &record.CurrentTier, &record.RequiredTier, &record.PromptVariant, &record.CTALabel, &record.CorrelationID, &record.MetadataJSON, &record.OccurredAt); err != nil {
			return nil, fmt.Errorf("scan subscriber upgrade interaction: %w", err)
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func firstNonEmptyUpgradeValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

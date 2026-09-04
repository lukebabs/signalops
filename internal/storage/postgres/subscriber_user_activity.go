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

const defaultSubscriberUserActivityLimit = 100
const maxSubscriberUserActivityLimit = 500

func (r *Repository) RecordSubscriberUserActivity(ctx context.Context, input storage.SubscriberUserActivityRecordInput) error {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.Subject = strings.TrimSpace(input.Subject)
	input.SubjectDisplayName = strings.TrimSpace(input.SubjectDisplayName)
	input.SubjectEmail = strings.ToLower(strings.TrimSpace(input.SubjectEmail))
	if input.AppID == "" {
		input.AppID = "marketops"
	}
	input.EventType = strings.TrimSpace(input.EventType)
	input.FeatureKey = strings.TrimSpace(input.FeatureKey)
	input.HTTPMethod = strings.ToUpper(strings.TrimSpace(input.HTTPMethod))
	input.RoutePath = strings.TrimSpace(input.RoutePath)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)
	if !validSubscriberTenantID(input.TenantID) {
		return errors.New("invalid subscriber activity tenant")
	}
	if input.Subject == "" {
		return errors.New("subscriber activity subject is required")
	}
	if input.AppID != "marketops" {
		return errors.New("subscriber activity app must be marketops")
	}
	switch input.EventType {
	case "login", "logout", "feature_view", "api_mutation":
	default:
		return errors.New("invalid subscriber activity event type")
	}
	if input.StatusCode < 0 || input.StatusCode > 599 {
		return errors.New("invalid subscriber activity status code")
	}
	metadata := input.MetadataJSON
	if len(metadata) == 0 {
		metadata = []byte(`{}`)
	}
	if !json.Valid(metadata) {
		return errors.New("subscriber activity metadata must be valid JSON")
	}
	var metadataObject map[string]any
	if err := json.Unmarshal(metadata, &metadataObject); err != nil {
		return errors.New("subscriber activity metadata must be a JSON object")
	}
	if input.SubjectDisplayName != "" {
		metadataObject["subject_display_name"] = input.SubjectDisplayName
	}
	if input.SubjectEmail != "" {
		metadataObject["subject_email"] = input.SubjectEmail
	}
	metadata, _ = json.Marshal(metadataObject)
	return r.WithSubscriberTenantScope(ctx, input.TenantID, func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
INSERT INTO subscriber_user_activity_events
  (activity_id, tenant_id, subject, app_id, event_type, feature_key, http_method, route_path, status_code, correlation_id, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb)`,
			newSubscriberID("subact"), input.TenantID, input.Subject, input.AppID, input.EventType, input.FeatureKey, input.HTTPMethod, input.RoutePath, input.StatusCode, input.CorrelationID, string(metadata))
		if err != nil {
			return fmt.Errorf("insert subscriber user activity: %w", err)
		}
		return nil
	})
}

func (r *Repository) ListSubscriberUserActivity(ctx context.Context, filter storage.SubscriberUserActivityFilter) (storage.SubscriberUserActivitySnapshot, error) {
	filter.TenantID = strings.TrimSpace(filter.TenantID)
	filter.Subject = strings.TrimSpace(filter.Subject)
	filter.Query = strings.TrimSpace(filter.Query)
	filter.EventType = strings.TrimSpace(filter.EventType)
	if !validSubscriberTenantID(filter.TenantID) {
		return storage.SubscriberUserActivitySnapshot{}, errors.New("invalid subscriber activity tenant")
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = defaultSubscriberUserActivityLimit
	}
	if limit > maxSubscriberUserActivityLimit {
		limit = maxSubscriberUserActivityLimit
	}
	var snapshot storage.SubscriberUserActivitySnapshot
	snapshot.TenantID = filter.TenantID
	err := r.WithSubscriberTenantScope(ctx, filter.TenantID, func(ctx context.Context, tx *sql.Tx) error {
		summaries, err := listSubscriberUserActivitySummariesTx(ctx, tx, filter)
		if err != nil {
			return err
		}
		events, err := listSubscriberUserActivityEventsTx(ctx, tx, filter, limit)
		if err != nil {
			return err
		}
		snapshot.Summaries = summaries
		snapshot.Events = events
		return nil
	})
	return snapshot, err
}

func listSubscriberUserActivitySummariesTx(ctx context.Context, tx *sql.Tx, filter storage.SubscriberUserActivityFilter) ([]storage.SubscriberUserActivitySummaryRecord, error) {
	rows, err := tx.QueryContext(ctx, `
WITH filtered AS (
  SELECT *
  FROM subscriber_user_activity_events
  WHERE tenant_id=$1
    AND ($2='' OR subject=$2)
    AND ($3='' OR event_type=$3)
    AND ($4='' OR (
      lower(subject) LIKE '%' || lower($4) || '%'
      OR lower(feature_key) LIKE '%' || lower($4) || '%'
      OR lower(route_path) LIKE '%' || lower($4) || '%'
      OR lower(http_method) LIKE '%' || lower($4) || '%'
      OR lower(event_type) LIKE '%' || lower($4) || '%'
    ))
    AND ($5::timestamptz IS NULL OR occurred_at >= $5)
    AND ($6::timestamptz IS NULL OR occurred_at <= $6)
),
activity_identity AS (
  SELECT DISTINCT ON (subject) subject,
    COALESCE(metadata->>'subject_display_name', '') AS display_name,
    COALESCE(metadata->>'subject_email', '') AS email
  FROM filtered
  WHERE COALESCE(metadata->>'subject_display_name', '') <> '' OR COALESCE(metadata->>'subject_email', '') <> ''
  ORDER BY subject, occurred_at DESC
),
top_features AS (
  SELECT DISTINCT ON (subject) subject, feature_key
  FROM (
    SELECT subject, feature_key, count(*) AS event_count, max(occurred_at) AS last_seen
    FROM filtered
    WHERE feature_key <> ''
    GROUP BY subject, feature_key
  ) ranked
  ORDER BY subject, event_count DESC, last_seen DESC, feature_key
)
SELECT f.subject,
  COALESCE(NULLIF(identity.display_name, ''), activity_identity.display_name, '') AS subject_display_name,
  COALESCE(NULLIF(identity.email, ''), activity_identity.email, '') AS subject_email,
  max(f.occurred_at) AS last_activity_at,
  max(f.occurred_at) FILTER (WHERE f.event_type='login') AS last_login_at,
  max(f.occurred_at) FILTER (WHERE f.event_type='logout') AS last_logout_at,
  count(*) FILTER (WHERE f.event_type='login') AS login_count,
  count(*) FILTER (WHERE f.event_type='feature_view') AS feature_view_count,
  count(*) FILTER (WHERE f.event_type='api_mutation') AS mutation_count,
  count(*) FILTER (WHERE f.event_type='api_mutation' AND f.status_code >= 400) AS failed_mutation_count,
  COALESCE(top_features.feature_key, '') AS top_feature_key
FROM filtered f
LEFT JOIN subscriber_subscription_admin_identity_labels($1) identity ON identity.subject=f.subject
LEFT JOIN activity_identity ON activity_identity.subject=f.subject
LEFT JOIN top_features ON top_features.subject=f.subject
GROUP BY f.subject, identity.display_name, identity.email, activity_identity.display_name, activity_identity.email, top_features.feature_key
ORDER BY max(f.occurred_at) DESC, f.subject
LIMIT 100`, filter.TenantID, filter.Subject, filter.EventType, filter.Query, filter.OccurredAtFrom, filter.OccurredAtTo)
	if err != nil {
		return nil, fmt.Errorf("list subscriber user activity summaries: %w", err)
	}
	defer rows.Close()
	records := []storage.SubscriberUserActivitySummaryRecord{}
	for rows.Next() {
		var record storage.SubscriberUserActivitySummaryRecord
		if err := rows.Scan(&record.Subject, &record.SubjectDisplayName, &record.SubjectEmail, &record.LastActivityAt, &record.LastLoginAt, &record.LastLogoutAt, &record.LoginCount, &record.FeatureViewCount, &record.MutationCount, &record.FailedMutationCount, &record.TopFeatureKey); err != nil {
			return nil, fmt.Errorf("scan subscriber user activity summary: %w", err)
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func listSubscriberUserActivityEventsTx(ctx context.Context, tx *sql.Tx, filter storage.SubscriberUserActivityFilter, limit int) ([]storage.SubscriberUserActivityEventRecord, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT e.activity_id, e.tenant_id, e.subject,
  COALESCE(NULLIF(identity.display_name, ''), activity_identity.display_name, '') AS subject_display_name,
  COALESCE(NULLIF(identity.email, ''), activity_identity.email, '') AS subject_email,
  e.app_id, e.event_type, e.feature_key, e.http_method, e.route_path,
  e.status_code, e.correlation_id, COALESCE(e.metadata, '{}'::jsonb), e.occurred_at
FROM subscriber_user_activity_events e
LEFT JOIN subscriber_subscription_admin_identity_labels($1) identity ON identity.subject=e.subject
LEFT JOIN LATERAL (
  SELECT COALESCE(metadata->>'subject_display_name', '') AS display_name,
    COALESCE(metadata->>'subject_email', '') AS email
  FROM subscriber_user_activity_events latest
  WHERE latest.tenant_id=$1 AND latest.subject=e.subject
    AND (COALESCE(latest.metadata->>'subject_display_name', '') <> '' OR COALESCE(latest.metadata->>'subject_email', '') <> '')
  ORDER BY latest.occurred_at DESC
  LIMIT 1
) activity_identity ON true
WHERE e.tenant_id=$1
  AND ($2='' OR e.subject=$2)
  AND ($3='' OR e.event_type=$3)
  AND ($4='' OR (
    lower(e.subject) LIKE '%' || lower($4) || '%'
    OR lower(COALESCE(NULLIF(identity.email, ''), activity_identity.email, '')) LIKE '%' || lower($4) || '%'
    OR lower(COALESCE(NULLIF(identity.display_name, ''), activity_identity.display_name, '')) LIKE '%' || lower($4) || '%'
    OR lower(e.feature_key) LIKE '%' || lower($4) || '%'
    OR lower(e.route_path) LIKE '%' || lower($4) || '%'
    OR lower(e.http_method) LIKE '%' || lower($4) || '%'
    OR lower(e.event_type) LIKE '%' || lower($4) || '%'
    OR e.status_code::text LIKE '%' || $4 || '%'
  ))
  AND ($5::timestamptz IS NULL OR e.occurred_at >= $5)
  AND ($6::timestamptz IS NULL OR e.occurred_at <= $6)
ORDER BY e.occurred_at DESC, e.activity_id
LIMIT $7`, filter.TenantID, filter.Subject, filter.EventType, filter.Query, filter.OccurredAtFrom, filter.OccurredAtTo, limit)
	if err != nil {
		return nil, fmt.Errorf("list subscriber user activity events: %w", err)
	}
	defer rows.Close()
	records := []storage.SubscriberUserActivityEventRecord{}
	for rows.Next() {
		var record storage.SubscriberUserActivityEventRecord
		if err := rows.Scan(&record.ActivityID, &record.TenantID, &record.Subject, &record.SubjectDisplayName, &record.SubjectEmail, &record.AppID, &record.EventType, &record.FeatureKey, &record.HTTPMethod, &record.RoutePath, &record.StatusCode, &record.CorrelationID, &record.MetadataJSON, &record.OccurredAt); err != nil {
			return nil, fmt.Errorf("scan subscriber user activity event: %w", err)
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

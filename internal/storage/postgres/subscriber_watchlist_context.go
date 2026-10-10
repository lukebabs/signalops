package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) GetSubscriberWatchlistContextPreference(ctx context.Context, tenantID, subject string) (storage.SubscriberWatchlistContextPreference, error) {
	tenantID, subject = strings.TrimSpace(tenantID), strings.TrimSpace(subject)
	var preference storage.SubscriberWatchlistContextPreference
	if !validSubscriberTenantID(tenantID) || subject == "" {
		return preference, errors.New("invalid subscriber watchlist context scope")
	}
	err := r.WithSubscriberTenantScope(ctx, tenantID, func(ctx context.Context, tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx, `SELECT tenant_id, subject, selection_mode, list_id, provenance, updated_at
FROM subscriber_watchlist_context_preferences WHERE tenant_id=$1 AND subject=$2`, tenantID, subject).
			Scan(&preference.TenantID, &preference.Subject, &preference.SelectionMode, &preference.ListID, &preference.ProvenanceJSON, &preference.UpdatedAt)
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ErrNotFound
		}
		return err
	})
	return preference, err
}

func (r *Repository) SetSubscriberWatchlistContextPreference(ctx context.Context, preference storage.SubscriberWatchlistContextPreference) (storage.SubscriberWatchlistContextPreference, error) {
	preference.TenantID, preference.Subject = strings.TrimSpace(preference.TenantID), strings.TrimSpace(preference.Subject)
	preference.SelectionMode, preference.ListID = strings.TrimSpace(preference.SelectionMode), strings.TrimSpace(preference.ListID)
	if !validSubscriberTenantID(preference.TenantID) || preference.Subject == "" ||
		(preference.SelectionMode != storage.SubscriberWatchlistContextModeAll && preference.SelectionMode != storage.SubscriberWatchlistContextModeList) ||
		(preference.SelectionMode == storage.SubscriberWatchlistContextModeAll && preference.ListID != "") ||
		(preference.SelectionMode == storage.SubscriberWatchlistContextModeList && preference.ListID == "") {
		return preference, errors.New("invalid subscriber watchlist context preference")
	}
	if len(preference.ProvenanceJSON) == 0 {
		preference.ProvenanceJSON = []byte(`{}`)
	}
	err := r.WithSubscriberTenantScope(ctx, preference.TenantID, func(ctx context.Context, tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, `INSERT INTO subscriber_watchlist_context_preferences
  (tenant_id, subject, selection_mode, list_id, provenance, updated_at)
VALUES ($1,$2,$3,$4,$5::jsonb,now())
ON CONFLICT (tenant_id, subject) DO UPDATE SET selection_mode=EXCLUDED.selection_mode, list_id=EXCLUDED.list_id, provenance=EXCLUDED.provenance, updated_at=now()
RETURNING tenant_id, subject, selection_mode, list_id, provenance, updated_at`, preference.TenantID, preference.Subject, preference.SelectionMode, preference.ListID, string(preference.ProvenanceJSON)).
			Scan(&preference.TenantID, &preference.Subject, &preference.SelectionMode, &preference.ListID, &preference.ProvenanceJSON, &preference.UpdatedAt)
	})
	return preference, err
}

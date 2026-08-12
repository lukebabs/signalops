package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) CreateSubscriberPrivateWatchlist(ctx context.Context, request storage.SubscriberWatchlistCreateRequest) (storage.SubscriberWatchlistRecord, error) {
	return r.createSubscriberWatchlist(ctx, request, storage.SubscriberWatchlistKindPrivate)
}

// CreateSubscriberTenantDefaultWatchlist persists the tenant's shared list.
// The API layer must require a tenant administrator before calling it.
func (r *Repository) CreateSubscriberTenantDefaultWatchlist(ctx context.Context, request storage.SubscriberWatchlistCreateRequest) (storage.SubscriberWatchlistRecord, error) {
	return r.createSubscriberWatchlist(ctx, request, storage.SubscriberWatchlistKindTenantDefault)
}

func (r *Repository) createSubscriberWatchlist(ctx context.Context, request storage.SubscriberWatchlistCreateRequest, kind string) (storage.SubscriberWatchlistRecord, error) {
	var record storage.SubscriberWatchlistRecord
	if err := normalizeSubscriberWatchlistCreate(&request, kind); err != nil {
		return record, err
	}
	if request.ListID == "" {
		request.ListID = newSubscriberID("sublist")
	}
	provenance := subscriberWatchlistProvenance(request.ProvenanceJSON, request.CorrelationID)
	err := r.WithSubscriberTenantScope(ctx, request.TenantID, func(ctx context.Context, tx *sql.Tx) error {
		owner := ""
		if kind == storage.SubscriberWatchlistKindPrivate {
			owner = request.ActorSubject
		}
		err := tx.QueryRowContext(ctx, `INSERT INTO subscriber_watchlists
  (list_id, tenant_id, list_kind, owner_subject, list_name, created_by_subject, updated_by_subject, provenance)
VALUES ($1,$2,$3,$4,$5,$6,$6,$7::jsonb)
RETURNING list_id, tenant_id, list_kind, owner_subject, list_name, created_by_subject, updated_by_subject, provenance, created_at, updated_at`,
			request.ListID, request.TenantID, kind, owner, request.ListName, request.ActorSubject, string(provenance),
		).Scan(&record.ListID, &record.TenantID, &record.ListKind, &record.OwnerSubject, &record.ListName, &record.CreatedBySubject, &record.UpdatedBySubject, &record.ProvenanceJSON, &record.CreatedAt, &record.UpdatedAt)
		if err != nil {
			if isSubscriberUniqueViolation(err) {
				return storage.ErrConflict
			}
			return fmt.Errorf("create subscriber watchlist: %w", err)
		}
		return insertSubscriberWatchlistAudit(ctx, tx, record.TenantID, record.ListID, request.ActorSubject, "create_list", "", nil, record, request.CorrelationID)
	})
	return record, err
}

// ListSubscriberWatchlists returns the tenant default plus private lists owned
// by the supplied immutable subject. It cannot enumerate another subject's lists.
func (r *Repository) ListSubscriberWatchlists(ctx context.Context, tenantID, subject string) ([]storage.SubscriberWatchlistRecord, error) {
	tenantID, subject = strings.TrimSpace(tenantID), strings.TrimSpace(subject)
	if !validSubscriberTenantID(tenantID) || subject == "" {
		return nil, errors.New("invalid subscriber watchlist scope")
	}
	records := []storage.SubscriberWatchlistRecord{}
	err := r.WithSubscriberTenantScope(ctx, tenantID, func(ctx context.Context, tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `SELECT list_id, tenant_id, list_kind, owner_subject, list_name, created_by_subject, updated_by_subject, provenance, created_at, updated_at
FROM subscriber_watchlists
WHERE tenant_id=$1 AND (list_kind='tenant_default' OR (list_kind='private' AND owner_subject=$2))
ORDER BY CASE WHEN list_kind='tenant_default' THEN 0 ELSE 1 END, lower(list_name), list_id`, tenantID, subject)
		if err != nil {
			return fmt.Errorf("list subscriber watchlists: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var record storage.SubscriberWatchlistRecord
			if err := rows.Scan(&record.ListID, &record.TenantID, &record.ListKind, &record.OwnerSubject, &record.ListName, &record.CreatedBySubject, &record.UpdatedBySubject, &record.ProvenanceJSON, &record.CreatedAt, &record.UpdatedAt); err != nil {
				return fmt.Errorf("scan subscriber watchlist: %w", err)
			}
			records = append(records, record)
		}
		return rows.Err()
	})
	return records, err
}

// ListSubscriberWatchlistMemberships only returns a list the subject can read:
// the tenant default or the subject's own private list.
func (r *Repository) ListSubscriberWatchlistMemberships(ctx context.Context, tenantID, subject, listID string) ([]storage.SubscriberWatchlistMembershipRecord, error) {
	tenantID, subject, listID = strings.TrimSpace(tenantID), strings.TrimSpace(subject), strings.TrimSpace(listID)
	if !validSubscriberTenantID(tenantID) || subject == "" || listID == "" {
		return nil, errors.New("invalid subscriber watchlist membership scope")
	}
	records := []storage.SubscriberWatchlistMembershipRecord{}
	err := r.WithSubscriberTenantScope(ctx, tenantID, func(ctx context.Context, tx *sql.Tx) error {
		if _, err := readableSubscriberWatchlist(ctx, tx, tenantID, subject, listID); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `SELECT tenant_id, list_id, global_asset_id, added_by_subject, provenance, added_at, updated_at
FROM subscriber_watchlist_memberships
WHERE tenant_id=$1 AND list_id=$2
ORDER BY added_at, global_asset_id`, tenantID, listID)
		if err != nil {
			return fmt.Errorf("list subscriber watchlist memberships: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var record storage.SubscriberWatchlistMembershipRecord
			if err := rows.Scan(&record.TenantID, &record.ListID, &record.GlobalAssetID, &record.AddedBySubject, &record.ProvenanceJSON, &record.AddedAt, &record.UpdatedAt); err != nil {
				return fmt.Errorf("scan subscriber watchlist membership: %w", err)
			}
			records = append(records, record)
		}
		return rows.Err()
	})
	return records, err
}

func (r *Repository) AddSubscriberPrivateWatchlistMembership(ctx context.Context, request storage.SubscriberWatchlistMembershipRequest) (storage.SubscriberWatchlistMembershipRecord, error) {
	return r.addSubscriberWatchlistMembership(ctx, request, storage.SubscriberWatchlistKindPrivate)
}

// AddSubscriberTenantDefaultWatchlistMembership requires a completed
// tenant-administrator authorization check in the API layer before invocation.
func (r *Repository) AddSubscriberTenantDefaultWatchlistMembership(ctx context.Context, request storage.SubscriberWatchlistMembershipRequest) (storage.SubscriberWatchlistMembershipRecord, error) {
	return r.addSubscriberWatchlistMembership(ctx, request, storage.SubscriberWatchlistKindTenantDefault)
}

func (r *Repository) addSubscriberWatchlistMembership(ctx context.Context, request storage.SubscriberWatchlistMembershipRequest, requiredKind string) (storage.SubscriberWatchlistMembershipRecord, error) {
	var record storage.SubscriberWatchlistMembershipRecord
	if err := normalizeSubscriberWatchlistMembership(&request); err != nil {
		return record, err
	}
	provenance := subscriberWatchlistProvenance(request.ProvenanceJSON, request.CorrelationID)
	err := r.WithSubscriberTenantScope(ctx, request.TenantID, func(ctx context.Context, tx *sql.Tx) error {
		list, err := writableSubscriberWatchlist(ctx, tx, request.TenantID, request.ActorSubject, request.ListID, requiredKind)
		if err != nil {
			return err
		}
		err = tx.QueryRowContext(ctx, `INSERT INTO subscriber_watchlist_memberships
  (tenant_id, list_id, global_asset_id, added_by_subject, provenance)
VALUES ($1,$2,$3,$4,$5::jsonb)
ON CONFLICT (list_id, global_asset_id) DO NOTHING
RETURNING tenant_id, list_id, global_asset_id, added_by_subject, provenance, added_at, updated_at`,
			request.TenantID, request.ListID, request.GlobalAssetID, request.ActorSubject, string(provenance),
		).Scan(&record.TenantID, &record.ListID, &record.GlobalAssetID, &record.AddedBySubject, &record.ProvenanceJSON, &record.AddedAt, &record.UpdatedAt)
		if errors.Is(err, sql.ErrNoRows) {
			err = tx.QueryRowContext(ctx, `SELECT tenant_id, list_id, global_asset_id, added_by_subject, provenance, added_at, updated_at
FROM subscriber_watchlist_memberships
WHERE tenant_id=$1 AND list_id=$2 AND global_asset_id=$3`,
				request.TenantID, request.ListID, request.GlobalAssetID,
			).Scan(&record.TenantID, &record.ListID, &record.GlobalAssetID, &record.AddedBySubject, &record.ProvenanceJSON, &record.AddedAt, &record.UpdatedAt)
			if errors.Is(err, sql.ErrNoRows) {
				return storage.ErrNotFound
			}
			if err != nil {
				return fmt.Errorf("read existing subscriber watchlist membership: %w", err)
			}
			return nil
		}
		if err != nil {
			return fmt.Errorf("add subscriber watchlist membership: %w", err)
		}
		return insertSubscriberWatchlistAudit(ctx, tx, list.TenantID, list.ListID, request.ActorSubject, "add_asset", request.GlobalAssetID, nil, record, request.CorrelationID)
	})
	return record, err
}

func (r *Repository) RemoveSubscriberPrivateWatchlistMembership(ctx context.Context, request storage.SubscriberWatchlistMembershipRequest) error {
	return r.removeSubscriberWatchlistMembership(ctx, request, storage.SubscriberWatchlistKindPrivate)
}

// RemoveSubscriberTenantDefaultWatchlistMembership requires a completed
// tenant-administrator authorization check in the API layer before invocation.
func (r *Repository) RemoveSubscriberTenantDefaultWatchlistMembership(ctx context.Context, request storage.SubscriberWatchlistMembershipRequest) error {
	return r.removeSubscriberWatchlistMembership(ctx, request, storage.SubscriberWatchlistKindTenantDefault)
}

func (r *Repository) removeSubscriberWatchlistMembership(ctx context.Context, request storage.SubscriberWatchlistMembershipRequest, requiredKind string) error {
	if err := normalizeSubscriberWatchlistMembership(&request); err != nil {
		return err
	}
	return r.WithSubscriberTenantScope(ctx, request.TenantID, func(ctx context.Context, tx *sql.Tx) error {
		list, err := writableSubscriberWatchlist(ctx, tx, request.TenantID, request.ActorSubject, request.ListID, requiredKind)
		if err != nil {
			return err
		}
		var removed storage.SubscriberWatchlistMembershipRecord
		err = tx.QueryRowContext(ctx, `DELETE FROM subscriber_watchlist_memberships
WHERE tenant_id=$1 AND list_id=$2 AND global_asset_id=$3
RETURNING tenant_id, list_id, global_asset_id, added_by_subject, provenance, added_at, updated_at`,
			request.TenantID, request.ListID, request.GlobalAssetID,
		).Scan(&removed.TenantID, &removed.ListID, &removed.GlobalAssetID, &removed.AddedBySubject, &removed.ProvenanceJSON, &removed.AddedAt, &removed.UpdatedAt)
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("remove subscriber watchlist membership: %w", err)
		}
		return insertSubscriberWatchlistAudit(ctx, tx, list.TenantID, list.ListID, request.ActorSubject, "remove_asset", request.GlobalAssetID, removed, nil, request.CorrelationID)
	})
}

func readableSubscriberWatchlist(ctx context.Context, tx *sql.Tx, tenantID, subject, listID string) (storage.SubscriberWatchlistRecord, error) {
	var record storage.SubscriberWatchlistRecord
	err := tx.QueryRowContext(ctx, `SELECT list_id, tenant_id, list_kind, owner_subject, list_name, created_by_subject, updated_by_subject, provenance, created_at, updated_at
FROM subscriber_watchlists
WHERE tenant_id=$1 AND list_id=$2
  AND (list_kind='tenant_default' OR (list_kind='private' AND owner_subject=$3))`,
		tenantID, listID, subject,
	).Scan(&record.ListID, &record.TenantID, &record.ListKind, &record.OwnerSubject, &record.ListName, &record.CreatedBySubject, &record.UpdatedBySubject, &record.ProvenanceJSON, &record.CreatedAt, &record.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return record, storage.ErrNotFound
	}
	if err != nil {
		return record, fmt.Errorf("read subscriber watchlist: %w", err)
	}
	return record, nil
}

func writableSubscriberWatchlist(ctx context.Context, tx *sql.Tx, tenantID, actorSubject, listID, requiredKind string) (storage.SubscriberWatchlistRecord, error) {
	var record storage.SubscriberWatchlistRecord
	query := `SELECT list_id, tenant_id, list_kind, owner_subject, list_name, created_by_subject, updated_by_subject, provenance, created_at, updated_at
FROM subscriber_watchlists WHERE tenant_id=$1 AND list_id=$2 AND list_kind=$3`
	args := []any{tenantID, listID, requiredKind}
	if requiredKind == storage.SubscriberWatchlistKindPrivate {
		query += " AND owner_subject=$4"
		args = append(args, actorSubject)
	}
	err := tx.QueryRowContext(ctx, query, args...).Scan(&record.ListID, &record.TenantID, &record.ListKind, &record.OwnerSubject, &record.ListName, &record.CreatedBySubject, &record.UpdatedBySubject, &record.ProvenanceJSON, &record.CreatedAt, &record.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return record, storage.ErrNotFound
	}
	if err != nil {
		return record, fmt.Errorf("read writable subscriber watchlist: %w", err)
	}
	return record, nil
}

func insertSubscriberWatchlistAudit(ctx context.Context, tx *sql.Tx, tenantID, listID, actor, mutation, globalAssetID string, before, after any, correlationID string) error {
	beforeJSON, err := json.Marshal(subscriberWatchlistAuditValue(before))
	if err != nil {
		return fmt.Errorf("encode subscriber watchlist audit before value: %w", err)
	}
	afterJSON, err := json.Marshal(subscriberWatchlistAuditValue(after))
	if err != nil {
		return fmt.Errorf("encode subscriber watchlist audit after value: %w", err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO subscriber_watchlist_audit
  (audit_id, tenant_id, list_id, actor_subject, mutation, global_asset_id, before_value, after_value, correlation_id, occurred_at)
VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8::jsonb,$9,$10)`,
		newSubscriberID("sublistaudit"), tenantID, listID, actor, mutation, globalAssetID, string(beforeJSON), string(afterJSON), correlationID, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert subscriber watchlist audit: %w", err)
	}
	return nil
}

func subscriberWatchlistAuditValue(value any) any {
	if value == nil {
		return map[string]any{}
	}
	return value
}

func subscriberWatchlistProvenance(raw []byte, correlationID string) []byte {
	value := map[string]any{"schema_version": "subscriber.watchlist.v1"}
	if len(raw) > 0 {
		var supplied map[string]any
		if err := json.Unmarshal(raw, &supplied); err == nil && supplied != nil {
			value = supplied
		}
	}
	value["correlation_id"] = correlationID
	result, _ := json.Marshal(value)
	return result
}

func normalizeSubscriberWatchlistCreate(request *storage.SubscriberWatchlistCreateRequest, kind string) error {
	request.ListID = strings.TrimSpace(request.ListID)
	request.TenantID = strings.TrimSpace(request.TenantID)
	request.ListName = strings.TrimSpace(request.ListName)
	request.ActorSubject = strings.TrimSpace(request.ActorSubject)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	if !validSubscriberTenantID(request.TenantID) || request.ActorSubject == "" || request.ListName == "" || utf8.RuneCountInString(request.ListName) > 120 {
		return errors.New("invalid subscriber watchlist create request")
	}
	if kind != storage.SubscriberWatchlistKindPrivate && kind != storage.SubscriberWatchlistKindTenantDefault {
		return errors.New("invalid subscriber watchlist kind")
	}
	return nil
}

func normalizeSubscriberWatchlistMembership(request *storage.SubscriberWatchlistMembershipRequest) error {
	request.TenantID = strings.TrimSpace(request.TenantID)
	request.ListID = strings.TrimSpace(request.ListID)
	request.GlobalAssetID = strings.TrimSpace(request.GlobalAssetID)
	request.ActorSubject = strings.TrimSpace(request.ActorSubject)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	if !validSubscriberTenantID(request.TenantID) || request.ListID == "" || request.GlobalAssetID == "" || request.ActorSubject == "" {
		return errors.New("invalid subscriber watchlist membership request")
	}
	return nil
}

func isSubscriberUniqueViolation(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "duplicate key")
}

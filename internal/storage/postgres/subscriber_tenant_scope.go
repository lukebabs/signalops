package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

var subscriberTenantIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$`)

// TenantScopedFunc executes a bounded subscriber-private storage operation.
// The callback must only use the supplied transaction; it must not issue
// tenant-private queries through Repository directly.
type TenantScopedFunc func(context.Context, *sql.Tx) error

// WithSubscriberTenantScope starts a transaction and sets the tenant context
// that future forced-RLS subscriber-private tables require. It is intentionally
// not applied to existing compatibility tables, which retain their current
// application-authorized model until a separately reviewed migration.
//
// PostgreSQL SET LOCAL is transaction-scoped, so the context is cleared before
// the connection is returned to the pool on both commit and rollback.
func (r *Repository) WithSubscriberTenantScope(ctx context.Context, tenantID string, fn TenantScopedFunc) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("subscriber tenant scope: repository is not initialized")
	}
	tenantID = strings.TrimSpace(tenantID)
	if !subscriberTenantIDPattern.MatchString(tenantID) {
		return fmt.Errorf("subscriber tenant scope: invalid tenant id")
	}
	if fn == nil {
		return fmt.Errorf("subscriber tenant scope: callback is required")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("subscriber tenant scope: begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `SELECT set_config('signalops.tenant_id', $1, true)`, tenantID); err != nil {
		return fmt.Errorf("subscriber tenant scope: set tenant context: %w", err)
	}
	if err := fn(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("subscriber tenant scope: commit transaction: %w", err)
	}
	return nil
}

func validSubscriberTenantID(tenantID string) bool {
	return subscriberTenantIDPattern.MatchString(strings.TrimSpace(tenantID))
}

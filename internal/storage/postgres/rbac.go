package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/appmeta"
	"github.com/lukebabs/signalops/internal/storage"
)

const accessColumns = `tenant_id,subject,display_name,email,app_id,permission,granted_by,granted_at,updated_at`

func isRegisteredUseCaseApp(appID string) bool {
	profile, ok := appmeta.ProfileByID(appID)
	return ok && profile.AppID != appmeta.AppConsole
}

func scanAccess(s interface{ Scan(...any) error }) (storage.TenantUserAccessRecord, error) {
	var x storage.TenantUserAccessRecord
	err := s.Scan(&x.TenantID, &x.Subject, &x.DisplayName, &x.Email, &x.AppID, &x.Permission, &x.GrantedBy, &x.GrantedAt, &x.UpdatedAt)
	return x, err
}
func (r *Repository) ListTenantUserAccess(ctx context.Context, tenant string) ([]storage.TenantUserAccessRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+accessColumns+` FROM tenant_user_access WHERE tenant_id=$1 ORDER BY display_name,subject,app_id`, strings.TrimSpace(tenant))
	if err != nil {
		return nil, fmt.Errorf("list tenant user access: %w", err)
	}
	defer rows.Close()
	out := []storage.TenantUserAccessRecord{}
	for rows.Next() {
		x, e := scanAccess(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *Repository) ListTenantUserAccessForSubject(ctx context.Context, tenant, subject string) ([]storage.TenantUserAccessRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+accessColumns+` FROM tenant_user_access WHERE tenant_id=$1 AND subject=$2 ORDER BY app_id`, strings.TrimSpace(tenant), strings.TrimSpace(subject))
	if err != nil {
		return nil, fmt.Errorf("list subject access: %w", err)
	}
	defer rows.Close()
	out := []storage.TenantUserAccessRecord{}
	for rows.Next() {
		x, e := scanAccess(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *Repository) UpsertTenantUserAccess(ctx context.Context, x storage.TenantUserAccessRecord, actorSubject, actorName string) (storage.TenantUserAccessRecord, error) {
	if strings.TrimSpace(x.TenantID) == "" || strings.TrimSpace(x.Subject) == "" || (!isRegisteredUseCaseApp(x.AppID)) || (x.Permission != "read" && x.Permission != "write") {
		return x, fmt.Errorf("invalid tenant user access")
	}
	tx, e := r.db.BeginTx(ctx, nil)
	if e != nil {
		return x, e
	}
	defer tx.Rollback()
	var before sql.NullString
	_ = tx.QueryRowContext(ctx, `SELECT row_to_json(t)::text FROM (SELECT `+accessColumns+` FROM tenant_user_access WHERE tenant_id=$1 AND subject=$2 AND app_id=$3) t`, x.TenantID, x.Subject, x.AppID).Scan(&before)
	mutation := "grant"
	if before.Valid {
		mutation = "update"
	}
	row := tx.QueryRowContext(ctx, `INSERT INTO tenant_user_access (`+accessColumns+`) VALUES ($1,$2,$3,$4,$5,$6,$7,now(),now()) ON CONFLICT (tenant_id,subject,app_id) DO UPDATE SET display_name=EXCLUDED.display_name,email=EXCLUDED.email,permission=EXCLUDED.permission,granted_by=EXCLUDED.granted_by,updated_at=now() RETURNING `+accessColumns, x.TenantID, x.Subject, x.DisplayName, x.Email, x.AppID, x.Permission, x.GrantedBy)
	out, e := scanAccess(row)
	if e != nil {
		return x, e
	}
	after := `{"permission":"` + out.Permission + `"}`
	if _, e = tx.ExecContext(ctx, `INSERT INTO tenant_user_access_audit (tenant_id,subject,app_id,mutation,actor_subject,actor_display_name,before_value,after_value) VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8::jsonb)`, x.TenantID, x.Subject, x.AppID, mutation, actorSubject, actorName, accessAuditJSON(before.String), after); e != nil {
		return x, e
	}
	return out, tx.Commit()
}
func (r *Repository) CreateInitialTenantUserAccess(ctx context.Context, x storage.TenantUserAccessRecord, actorSubject, actorName string) (storage.TenantUserAccessRecord, bool, error) {
	if strings.TrimSpace(x.TenantID) == "" || strings.TrimSpace(x.Subject) == "" || !isRegisteredUseCaseApp(x.AppID) || (x.Permission != "read" && x.Permission != "write") {
		return x, false, fmt.Errorf("invalid tenant user access")
	}
	tx, e := r.db.BeginTx(ctx, nil)
	if e != nil {
		return x, false, e
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, x.TenantID); e != nil {
		return x, false, e
	}
	var provisioned bool
	if e = tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM tenant_user_access WHERE tenant_id=$1) OR EXISTS (SELECT 1 FROM tenant_user_access_audit WHERE tenant_id=$1)`, x.TenantID).Scan(&provisioned); e != nil {
		return x, false, e
	}
	if provisioned {
		return x, false, nil
	}
	row := tx.QueryRowContext(ctx, `INSERT INTO tenant_user_access (`+accessColumns+`) VALUES ($1,$2,$3,$4,$5,$6,$7,now(),now()) RETURNING `+accessColumns, x.TenantID, x.Subject, x.DisplayName, x.Email, x.AppID, x.Permission, x.GrantedBy)
	out, e := scanAccess(row)
	if e != nil {
		return x, false, e
	}
	after := `{"permission":"` + out.Permission + `"}`
	if _, e = tx.ExecContext(ctx, `INSERT INTO tenant_user_access_audit (tenant_id,subject,app_id,mutation,actor_subject,actor_display_name,before_value,after_value) VALUES ($1,$2,$3,'grant',$4,$5,'{}'::jsonb,$6::jsonb)`, x.TenantID, x.Subject, x.AppID, actorSubject, actorName, after); e != nil {
		return x, false, e
	}
	if e = tx.Commit(); e != nil {
		return x, false, e
	}
	return out, true, nil
}

func accessAuditJSON(v string) string {
	if strings.TrimSpace(v) == "" {
		return "{}"
	}
	return v
}
func (r *Repository) DeleteTenantUserAccess(ctx context.Context, tenant, subject, app, actorSubject, actorName string) error {
	tx, e := r.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var before string
	e = tx.QueryRowContext(ctx, `DELETE FROM tenant_user_access WHERE tenant_id=$1 AND subject=$2 AND app_id=$3 RETURNING row_to_json(tenant_user_access)::text`, tenant, subject, app).Scan(&before)
	if e == sql.ErrNoRows {
		return storage.ErrNotFound
	}
	if e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO tenant_user_access_audit (tenant_id,subject,app_id,mutation,actor_subject,actor_display_name,before_value) VALUES ($1,$2,$3,'revoke',$4,$5,$6::jsonb)`, tenant, subject, app, actorSubject, actorName, before)
	if e != nil {
		return e
	}
	return tx.Commit()
}
func (r *Repository) ListTenantUserAccessAudit(ctx context.Context, tenant, subject string, limit int) ([]storage.TenantUserAccessAuditRecord, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	q := `SELECT audit_id,tenant_id,subject,app_id,mutation,actor_subject,actor_display_name,before_value::text,after_value::text,occurred_at FROM tenant_user_access_audit WHERE tenant_id=$1`
	args := []any{tenant}
	if subject != "" {
		q += ` AND subject=$2`
		args = append(args, subject)
	}
	q += ` ORDER BY occurred_at DESC LIMIT $` + fmt.Sprint(len(args)+1)
	args = append(args, limit)
	rows, e := r.db.QueryContext(ctx, q, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []storage.TenantUserAccessAuditRecord{}
	for rows.Next() {
		var x storage.TenantUserAccessAuditRecord
		if e = rows.Scan(&x.AuditID, &x.TenantID, &x.Subject, &x.AppID, &x.Mutation, &x.ActorSubject, &x.ActorDisplayName, &x.BeforeJSON, &x.AfterJSON, &x.OccurredAt); e != nil {
			return nil, e
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) UpsertAdministrationNotification(ctx context.Context, x storage.AdministrationNotificationRecord) (storage.AdministrationNotificationRecord, error) {
	row := r.db.QueryRowContext(ctx, `
INSERT INTO administration_notifications(
  notification_id,tenant_id,source,category,severity,title,summary,dedupe_key,state,
  occurrence_count,first_occurred_at,last_occurred_at,evidence
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'active',1,$9,$10,$11)
ON CONFLICT(tenant_id,dedupe_key,state) DO UPDATE SET
  severity = CASE
    WHEN administration_notifications.severity = 'critical' OR EXCLUDED.severity = 'critical' THEN 'critical'
    WHEN administration_notifications.occurrence_count + 1 >= 3 AND EXCLUDED.category = 'routine_job_failure' THEN 'critical'
    ELSE EXCLUDED.severity
  END,
  title=EXCLUDED.title,
  summary=EXCLUDED.summary,
  evidence=EXCLUDED.evidence,
  occurrence_count=administration_notifications.occurrence_count+1,
  last_occurred_at=EXCLUDED.last_occurred_at
RETURNING notification_id,tenant_id,source,category,severity,title,summary,dedupe_key,state,
  occurrence_count,first_occurred_at,last_occurred_at,resolved_at,evidence`,
		x.NotificationID, x.TenantID, x.Source, x.Category, x.Severity, x.Title, x.Summary,
		x.DedupeKey, x.FirstOccurredAt, x.LastOccurredAt, jsonOrEmpty(x.EvidenceJSON))
	return scanAdministrationNotification(row, x)
}

func (r *Repository) ListAdministrationNotifications(ctx context.Context, f storage.AdministrationNotificationFilter) ([]storage.AdministrationNotificationRecord, error) {
	q := `SELECT n.notification_id,n.tenant_id,n.source,n.category,n.severity,n.title,n.summary,n.dedupe_key,n.state,n.occurrence_count,n.first_occurred_at,n.last_occurred_at,n.resolved_at,n.evidence,s.read_at,s.archived_at
FROM administration_notifications n
LEFT JOIN administration_notification_inbox_state s ON s.notification_id=n.notification_id AND s.subject=$2
WHERE n.tenant_id=$1`
	a := []any{f.TenantID, f.Subject}
	if !f.IncludeArchived {
		q += ` AND s.archived_at IS NULL`
	}
	if value := strings.TrimSpace(f.Severity); value != "" {
		a = append(a, value)
		q += fmt.Sprintf(" AND n.severity=$%d", len(a))
	}
	if value := strings.TrimSpace(f.State); value != "" {
		a = append(a, value)
		q += fmt.Sprintf(" AND n.state=$%d", len(a))
	}
	q += ` ORDER BY n.last_occurred_at DESC LIMIT ` + fmt.Sprint(clampLimit(f.Limit))
	rows, err := r.db.QueryContext(ctx, q, a...)
	if err != nil {
		return nil, fmt.Errorf("list administration notifications: %w", err)
	}
	defer rows.Close()
	out := []storage.AdministrationNotificationRecord{}
	for rows.Next() {
		var x storage.AdministrationNotificationRecord
		var resolved, read, archived sql.NullTime
		if err := rows.Scan(&x.NotificationID, &x.TenantID, &x.Source, &x.Category, &x.Severity, &x.Title, &x.Summary, &x.DedupeKey, &x.State, &x.OccurrenceCount, &x.FirstOccurredAt, &x.LastOccurredAt, &resolved, &x.EvidenceJSON, &read, &archived); err != nil {
			return nil, err
		}
		if resolved.Valid {
			x.ResolvedAt = &resolved.Time
		}
		if read.Valid {
			x.ReadAt = &read.Time
		}
		if archived.Valid {
			x.ArchivedAt = &archived.Time
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

type administrationNotificationRow interface{ Scan(...any) error }

func scanAdministrationNotification(row administrationNotificationRow, x storage.AdministrationNotificationRecord) (storage.AdministrationNotificationRecord, error) {
	var resolved sql.NullTime
	if err := row.Scan(&x.NotificationID, &x.TenantID, &x.Source, &x.Category, &x.Severity, &x.Title, &x.Summary, &x.DedupeKey, &x.State, &x.OccurrenceCount, &x.FirstOccurredAt, &x.LastOccurredAt, &resolved, &x.EvidenceJSON); err != nil {
		return x, fmt.Errorf("upsert administration notification: %w", err)
	}
	if resolved.Valid {
		x.ResolvedAt = &resolved.Time
	}
	return x, nil
}

func (r *Repository) SetAdministrationNotificationInboxState(ctx context.Context, x storage.AdministrationNotificationInboxState) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO administration_notification_inbox_state(notification_id,subject,read_at,archived_at)
VALUES($1,$2,$3,$4)
ON CONFLICT(notification_id,subject) DO UPDATE SET read_at=EXCLUDED.read_at,archived_at=EXCLUDED.archived_at`,
		x.NotificationID, x.Subject, x.ReadAt, x.ArchivedAt)
	return err
}

func (r *Repository) GetAdministrationSMTPSettings(ctx context.Context, tenant string) (storage.AdministrationSMTPSettings, error) {
	var x storage.AdministrationSMTPSettings
	err := r.db.QueryRowContext(ctx, `
SELECT tenant_id,host,port,COALESCE(username,''),COALESCE(password_ciphertext,''::bytea),
  use_starttls,use_ssl,from_email,from_name,COALESCE(reply_to,''),updated_by,updated_at
FROM administration_smtp_settings WHERE tenant_id=$1`, tenant).Scan(
		&x.TenantID, &x.Host, &x.Port, &x.Username, &x.PasswordCiphertext, &x.UseStartTLS, &x.UseSSL,
		&x.FromEmail, &x.FromName, &x.ReplyTo, &x.UpdatedBy, &x.UpdatedAt)
	x.HasPassword = len(x.PasswordCiphertext) > 0
	if err == sql.ErrNoRows {
		return x, storage.ErrNotFound
	}
	return x, err
}

func (r *Repository) UpsertAdministrationSMTPSettings(ctx context.Context, x storage.AdministrationSMTPSettings) (storage.AdministrationSMTPSettings, error) {
	err := r.db.QueryRowContext(ctx, `
INSERT INTO administration_smtp_settings(tenant_id,host,port,username,password_ciphertext,use_starttls,use_ssl,from_email,from_name,reply_to,updated_by)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT(tenant_id) DO UPDATE SET
  host=EXCLUDED.host,port=EXCLUDED.port,username=EXCLUDED.username,
  password_ciphertext=COALESCE(NULLIF(EXCLUDED.password_ciphertext,''::bytea),administration_smtp_settings.password_ciphertext),
  use_starttls=EXCLUDED.use_starttls,use_ssl=EXCLUDED.use_ssl,from_email=EXCLUDED.from_email,
  from_name=EXCLUDED.from_name,reply_to=EXCLUDED.reply_to,updated_by=EXCLUDED.updated_by,updated_at=now()
RETURNING updated_at`, x.TenantID, x.Host, x.Port, x.Username, x.PasswordCiphertext, x.UseStartTLS, x.UseSSL, x.FromEmail, x.FromName, x.ReplyTo, x.UpdatedBy).Scan(&x.UpdatedAt)
	x.HasPassword = len(x.PasswordCiphertext) > 0
	return x, err
}

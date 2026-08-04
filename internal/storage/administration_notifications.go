package storage

import (
	"context"
	"time"
)

type AdministrationNotificationRecord struct {
	NotificationID  string     `json:"notification_id"`
	TenantID        string     `json:"tenant_id"`
	Source          string     `json:"source"`
	Category        string     `json:"category"`
	Severity        string     `json:"severity"`
	Title           string     `json:"title"`
	Summary         string     `json:"summary"`
	DedupeKey       string     `json:"dedupe_key"`
	State           string     `json:"state"`
	OccurrenceCount int        `json:"occurrence_count"`
	FirstOccurredAt time.Time  `json:"first_occurred_at"`
	LastOccurredAt  time.Time  `json:"last_occurred_at"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
	EvidenceJSON    []byte     `json:"evidence"`
	ReadAt          *time.Time `json:"read_at,omitempty"`
	ArchivedAt      *time.Time `json:"archived_at,omitempty"`
}

type AdministrationNotificationFilter struct {
	TenantID, Subject, Severity, State string
	IncludeArchived                    bool
	Limit                              int
}

type AdministrationNotificationInboxState struct {
	NotificationID, Subject string
	ReadAt, ArchivedAt      *time.Time
}

type AdministrationSMTPSettings struct {
	TenantID           string    `json:"tenant_id"`
	Host               string    `json:"host"`
	Port               int       `json:"port"`
	Username           string    `json:"username"`
	PasswordCiphertext []byte    `json:"-"`
	HasPassword        bool      `json:"has_password"`
	UseStartTLS        bool      `json:"use_starttls"`
	UseSSL             bool      `json:"use_ssl"`
	FromEmail          string    `json:"from_email"`
	FromName           string    `json:"from_name"`
	ReplyTo            string    `json:"reply_to"`
	UpdatedBy          string    `json:"updated_by"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type AdministrationNotificationRepository interface {
	UpsertAdministrationNotification(context.Context, AdministrationNotificationRecord) (AdministrationNotificationRecord, error)
	ListAdministrationNotifications(context.Context, AdministrationNotificationFilter) ([]AdministrationNotificationRecord, error)
	SetAdministrationNotificationInboxState(context.Context, AdministrationNotificationInboxState) error
	GetAdministrationSMTPSettings(context.Context, string) (AdministrationSMTPSettings, error)
	UpsertAdministrationSMTPSettings(context.Context, AdministrationSMTPSettings) (AdministrationSMTPSettings, error)
}

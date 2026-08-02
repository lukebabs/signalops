package storage

import (
	"context"
	"time"
)

type TenantUserAccessRecord struct {
	TenantID    string    `json:"tenant_id"`
	Subject     string    `json:"subject"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email"`
	AppID       string    `json:"app_id"`
	Permission  string    `json:"permission"`
	GrantedBy   string    `json:"granted_by"`
	GrantedAt   time.Time `json:"granted_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
type TenantUserAccessAuditRecord struct {
	AuditID                                                            int64
	TenantID, Subject, AppID, Mutation, ActorSubject, ActorDisplayName string
	BeforeJSON, AfterJSON                                              string
	OccurredAt                                                         time.Time
}
type TenantUserAccessRepository interface {
	ListTenantUserAccess(context.Context, string) ([]TenantUserAccessRecord, error)
	ListTenantUserAccessForSubject(context.Context, string, string) ([]TenantUserAccessRecord, error)
	UpsertTenantUserAccess(context.Context, TenantUserAccessRecord, string, string) (TenantUserAccessRecord, error)
	DeleteTenantUserAccess(context.Context, string, string, string, string, string) error
	ListTenantUserAccessAudit(context.Context, string, string, int) ([]TenantUserAccessAuditRecord, error)
}

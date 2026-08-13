package storage

import (
	"context"
	"time"
)

const (
	SubscriberWatchlistKindTenantDefault = "tenant_default"
	SubscriberWatchlistKindPrivate       = "private"
	SubscriberWatchlistContextModeList   = "list"
	SubscriberWatchlistContextModeAll    = "all"
)

type SubscriberWatchlistRecord struct {
	ListID           string
	TenantID         string
	ListKind         string
	OwnerSubject     string
	ListName         string
	CreatedBySubject string
	UpdatedBySubject string
	ProvenanceJSON   []byte
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type SubscriberWatchlistMembershipRecord struct {
	TenantID       string
	ListID         string
	GlobalAssetID  string
	AddedBySubject string
	ProvenanceJSON []byte
	AddedAt        time.Time
	UpdatedAt      time.Time
}

type SubscriberWatchlistItemRecord struct {
	TenantID          string
	ListID            string
	ListKind          string
	ListName          string
	GlobalAssetID     string
	Ticker            string
	CompanyName       string
	AssetType         string
	Exchange          string
	Sector            string
	EligibilityStatus string
	CoverageState     string
	CoverageMode      string
	AddedAt           time.Time
}

// SubscriberWatchlistContextPreference is a subject-owned UX preference. It
// contains no market data and is always resolved against the caller's current
// authorized list set before it can be used by a MarketOps read.
type SubscriberWatchlistContextPreference struct {
	TenantID       string
	Subject        string
	SelectionMode  string
	ListID         string
	UpdatedAt      time.Time
	ProvenanceJSON []byte
}

type SubscriberWatchlistCreateRequest struct {
	ListID         string
	TenantID       string
	ListName       string
	ActorSubject   string
	CorrelationID  string
	ProvenanceJSON []byte
}

type SubscriberWatchlistMembershipRequest struct {
	TenantID       string
	ListID         string
	GlobalAssetID  string
	ActorSubject   string
	CorrelationID  string
	ProvenanceJSON []byte
}

// SubscriberWatchlistRepository is the storage boundary for S3 list
// preferences. Tenant-default mutations must be invoked only after the API
// tenant-administrator guard succeeds; private mutations require the owner
// subject and are constrained by the repository query itself.
type SubscriberWatchlistRepository interface {
	CreateSubscriberPrivateWatchlist(context.Context, SubscriberWatchlistCreateRequest) (SubscriberWatchlistRecord, error)
	CreateSubscriberTenantDefaultWatchlist(context.Context, SubscriberWatchlistCreateRequest) (SubscriberWatchlistRecord, error)
	ListSubscriberWatchlists(context.Context, string, string) ([]SubscriberWatchlistRecord, error)
	GetSubscriberWatchlistContextPreference(context.Context, string, string) (SubscriberWatchlistContextPreference, error)
	SetSubscriberWatchlistContextPreference(context.Context, SubscriberWatchlistContextPreference) (SubscriberWatchlistContextPreference, error)
	ListSubscriberWatchlistMemberships(context.Context, string, string, string) ([]SubscriberWatchlistMembershipRecord, error)
	ListSubscriberWatchlistItems(context.Context, string, string, string) ([]SubscriberWatchlistItemRecord, error)
	AddSubscriberPrivateWatchlistMembership(context.Context, SubscriberWatchlistMembershipRequest) (SubscriberWatchlistMembershipRecord, error)
	AddSubscriberTenantDefaultWatchlistMembership(context.Context, SubscriberWatchlistMembershipRequest) (SubscriberWatchlistMembershipRecord, error)
	RemoveSubscriberPrivateWatchlistMembership(context.Context, SubscriberWatchlistMembershipRequest) error
	RemoveSubscriberTenantDefaultWatchlistMembership(context.Context, SubscriberWatchlistMembershipRequest) error
}

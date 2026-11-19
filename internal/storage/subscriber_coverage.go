package storage

import "context"

type SubscriberAdminWarmCatalogRecord struct {
	GlobalAssetID       string
	Ticker              string
	CompanyName         string
	AssetType           string
	Exchange            string
	Sector              string
	EligibilityStatus   string
	CoverageState       string
	CoverageMode        string
	WarmRank            int
	MarketCapRank       int
	TenantDefaultMember bool
	LegacyProtected     bool
}

type SubscriberCoverageRepository interface {
	ListSubscriberAdminWarmCatalog(context.Context, string, string, string, string, int, int) ([]SubscriberAdminWarmCatalogRecord, error)
	AddSubscriberTenantDefaultCatalogMembership(context.Context, SubscriberWatchlistMembershipRequest) (SubscriberCatalogMembershipResult, error)
}

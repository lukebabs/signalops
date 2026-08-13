package storage

import "context"

type SubscriberCatalogProjectionRecord struct {
	GlobalAssetID     string
	Ticker            string
	CompanyName       string
	AssetType         string
	Exchange          string
	Sector            string
	EligibilityStatus string
	CoverageState     string
	CoverageMode      string
}

type SubscriberCatalogProjectionRepository interface {
	SearchSubscriberCatalog(context.Context, string, string, int) ([]SubscriberCatalogProjectionRecord, error)
}

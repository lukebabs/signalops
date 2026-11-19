package postgres

import "testing"

func TestSubscriberGlobalAssetIDIsStableAndSourceScoped(t *testing.T) {
	first := subscriberGlobalAssetID("src-massive", " aapl ")
	if first != subscriberGlobalAssetID("SRC-MASSIVE", "AAPL") {
		t.Fatal("global asset identity is not normalized and stable")
	}
	if first == subscriberGlobalAssetID("other-source", "AAPL") {
		t.Fatal("global asset identity must retain source provenance")
	}
}

func TestSubscriberCatalogFingerprintChangesWithReference(t *testing.T) {
	asset := subscriberCatalogSourceAsset{TenantID: "tenant-local", SourceID: "src-massive", UniverseGroup: "top", Rank: 1, Ticker: "AAPL", Company: "Apple", IsActive: true}
	first := subscriberCatalogFingerprint(asset)
	asset.Company = "Apple Inc."
	if first == subscriberCatalogFingerprint(asset) {
		t.Fatal("reference fingerprint must change when reference evidence changes")
	}
}

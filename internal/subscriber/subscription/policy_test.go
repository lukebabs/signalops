package subscription

import "testing"

func TestTierDepthIsMonotonic(t *testing.T) {
	features := []Feature{
		FeatureMarketDashboards,
		FeatureSectorRotationDiscovery,
		FeatureValueIntelligence,
		FeatureResearchReports,
		FeatureSignalAssuranceAnalytics,
		FeatureAPI,
	}
	for _, feature := range features {
		if Allows(TierExplorer, feature) && !Allows(TierProfessional, feature) {
			t.Fatalf("professional unexpectedly loses %s", feature)
		}
		if Allows(TierProfessional, feature) && !Allows(TierInstitutional, feature) {
			t.Fatalf("institutional unexpectedly loses %s", feature)
		}
	}
}

func TestExplorerIsDiscoveryOnly(t *testing.T) {
	if !Allows(TierExplorer, FeatureMarketDashboards) || !Allows(TierExplorer, FeaturePublicSignals) {
		t.Fatal("explorer must retain its discovery surfaces")
	}
	for _, feature := range []Feature{FeatureValueIntelligence, FeatureOptionsSignals, FeatureEarningsCalendar, FeatureResearchReports} {
		if Allows(TierExplorer, feature) {
			t.Fatalf("explorer must not receive %s", feature)
		}
	}
	if got, _ := Limit(TierExplorer, LimitPrivateWatchlists); got != 3 {
		t.Fatalf("explorer watchlist limit = %d, want 3", got)
	}
	if got, _ := Limit(TierExplorer, LimitAssetsPerWatchlist); got != 25 {
		t.Fatalf("explorer asset limit = %d, want 25", got)
	}
}

func TestProfessionalAndInstitutionalDepth(t *testing.T) {
	for _, feature := range []Feature{FeatureValueIntelligence, FeatureDistressedOpportunityIntelligence, FeatureEarningsOpportunityIntelligence, FeatureSectorRotationDetail, FeatureOptionsSignals, FeatureEarningsCalendar, FeatureResearchReports} {
		if !Allows(TierProfessional, feature) {
			t.Fatalf("professional must receive %s", feature)
		}
	}
	for _, feature := range []Feature{FeatureSignalAssuranceAnalytics, FeaturePortfolioAnalysis, FeatureBatchScreening, FeatureHistoricalReplay, FeatureStrategyValidation, FeatureCustomUniverses, FeatureAPI, FeatureWhiteLabel} {
		if !Allows(TierInstitutional, feature) {
			t.Fatalf("institutional must receive %s", feature)
		}
		if Allows(TierProfessional, feature) {
			t.Fatalf("professional must not receive institutional feature %s", feature)
		}
	}
	if got, _ := Limit(TierInstitutional, LimitPrivateWatchlists); got != -1 {
		t.Fatalf("institutional watchlist limit = %d, want governed fair-use marker", got)
	}
}

func TestPolicyCopiesCannotMutateDefinitions(t *testing.T) {
	policy, ok := PolicyFor(TierProfessional)
	if !ok {
		t.Fatal("professional policy missing")
	}
	policy.Features[FeatureAPI] = true
	policy.Limits[LimitPrivateWatchlists] = 1
	if Allows(TierProfessional, FeatureAPI) {
		t.Fatal("caller must not mutate canonical feature policy")
	}
	if got, _ := Limit(TierProfessional, LimitPrivateWatchlists); got != 20 {
		t.Fatalf("caller must not mutate canonical limit, got %d", got)
	}
}

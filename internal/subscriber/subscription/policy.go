// Package subscription defines the commercial feature policy for MarketOps.
//
// It is intentionally independent from provider-demand entitlements. A paid
// plan controls what analysis a subscriber may consume; it never authorizes a
// browser request to poll a provider or create a tenant-local market-data copy.
package subscription

import "strings"

type Tier string

const (
	TierExplorer      Tier = "explorer"
	TierProfessional  Tier = "professional"
	TierInstitutional Tier = "institutional"
)

type Feature string

const (
	FeatureMarketDashboards                  Feature = "market_dashboards"
	FeaturePublicSignals                     Feature = "public_signals"
	FeatureSectorRotationDiscovery           Feature = "sector_rotation_discovery"
	FeatureValueIntelligence                 Feature = "value_intelligence"
	FeatureDistressedOpportunityIntelligence Feature = "distressed_opportunity_intelligence"
	FeatureEarningsOpportunityIntelligence   Feature = "earnings_opportunity_intelligence"
	FeatureSectorRotationDetail              Feature = "sector_rotation_detail"
	FeatureOptionsSignals                    Feature = "options_signals"
	FeatureEarningsCalendar                  Feature = "earnings_calendar"
	FeatureResearchReports                   Feature = "research_reports"
	FeatureSignalAssuranceAnalytics          Feature = "signal_assurance_analytics"
	FeaturePortfolioAnalysis                 Feature = "portfolio_analysis"
	FeatureBatchScreening                    Feature = "batch_screening"
	FeatureHistoricalReplay                  Feature = "historical_replay"
	FeatureStrategyValidation                Feature = "strategy_validation"
	FeatureCustomUniverses                   Feature = "custom_universes"
	FeatureAPI                               Feature = "api"
	FeatureWhiteLabel                        Feature = "white_label"
)

const (
	LimitPrivateWatchlists  = "private_watchlists"
	LimitAssetsPerWatchlist = "assets_per_watchlist"
)

// Policy is a versioned, immutable-in-process plan definition. Billing stores
// plan revisions separately; callers must not infer provider collection rights
// from any feature below.
type Policy struct {
	Tier     Tier
	Features map[Feature]bool
	Limits   map[string]int // -1 means governed fair-use, not an unbounded resource.
}

var policies = map[Tier]Policy{
	TierExplorer: {
		Tier: TierExplorer,
		Features: featureSet(
			FeatureMarketDashboards,
			FeaturePublicSignals,
			FeatureSectorRotationDiscovery,
		),
		Limits: map[string]int{LimitPrivateWatchlists: 3, LimitAssetsPerWatchlist: 25},
	},
	TierProfessional: {
		Tier: TierProfessional,
		Features: featureSet(
			FeatureMarketDashboards,
			FeaturePublicSignals,
			FeatureSectorRotationDiscovery,
			FeatureValueIntelligence,
			FeatureDistressedOpportunityIntelligence,
			FeatureEarningsOpportunityIntelligence,
			FeatureSectorRotationDetail,
			FeatureOptionsSignals,
			FeatureEarningsCalendar,
			FeatureResearchReports,
		),
		Limits: map[string]int{LimitPrivateWatchlists: 20, LimitAssetsPerWatchlist: 100},
	},
	TierInstitutional: {
		Tier: TierInstitutional,
		Features: featureSet(
			FeatureMarketDashboards,
			FeaturePublicSignals,
			FeatureSectorRotationDiscovery,
			FeatureValueIntelligence,
			FeatureDistressedOpportunityIntelligence,
			FeatureEarningsOpportunityIntelligence,
			FeatureSectorRotationDetail,
			FeatureOptionsSignals,
			FeatureEarningsCalendar,
			FeatureResearchReports,
			FeatureSignalAssuranceAnalytics,
			FeaturePortfolioAnalysis,
			FeatureBatchScreening,
			FeatureHistoricalReplay,
			FeatureStrategyValidation,
			FeatureCustomUniverses,
			FeatureAPI,
			FeatureWhiteLabel,
		),
		Limits: map[string]int{LimitPrivateWatchlists: -1, LimitAssetsPerWatchlist: -1},
	},
}

func featureSet(features ...Feature) map[Feature]bool {
	result := make(map[Feature]bool, len(features))
	for _, feature := range features {
		result[feature] = true
	}
	return result
}

func ParseTier(value string) (Tier, bool) {
	tier := Tier(strings.TrimSpace(value))
	_, ok := policies[tier]
	return tier, ok
}

func PolicyFor(tier Tier) (Policy, bool) {
	policy, ok := policies[tier]
	if !ok {
		return Policy{}, false
	}
	return Policy{Tier: policy.Tier, Features: copyFeatures(policy.Features), Limits: copyLimits(policy.Limits)}, true
}

func Allows(tier Tier, feature Feature) bool {
	policy, ok := policies[tier]
	return ok && policy.Features[feature]
}

func Limit(tier Tier, name string) (int, bool) {
	policy, ok := policies[tier]
	if !ok {
		return 0, false
	}
	limit, exists := policy.Limits[name]
	return limit, exists
}

func copyFeatures(source map[Feature]bool) map[Feature]bool {
	result := make(map[Feature]bool, len(source))
	for feature, enabled := range source {
		result[feature] = enabled
	}
	return result
}

func copyLimits(source map[string]int) map[string]int {
	result := make(map[string]int, len(source))
	for name, limit := range source {
		result[name] = limit
	}
	return result
}

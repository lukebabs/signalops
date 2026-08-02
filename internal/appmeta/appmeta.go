package appmeta

import "strings"

const (
	DefaultAppID   = "console"
	DefaultUseCase = "general"

	AppConsole   = "console"
	AppMarketOps = "marketops"
	AppCyberOps  = "cyberops"
)

type Metadata struct {
	AppID   string
	Domain  string
	UseCase string
}

type Profile struct {
	AppID            string   `json:"app_id"`
	Label            string   `json:"label"`
	DefaultRoute     string   `json:"default_route"`
	Domains          []string `json:"domains"`
	EnabledModules   []string `json:"enabled_modules"`
	DashboardProfile string   `json:"dashboard_profile"`
	LandingSummary   string   `json:"landing_summary"`
	RoutePrefix      string   `json:"route_prefix"`
}

var Profiles = []Profile{
	{
		AppID:        AppConsole,
		Label:        "Administration",
		DefaultRoute: "/admin/dashboard",
		Domains:      []string{"market_data", "crm", "security", "operations", "iot", "procurement", "custom"},
		EnabledModules: []string{
			"dashboard", "event_explorer", "timeline", "correlation", "insights",
			"pipelines", "rules", "sources", "health", "replay", "administration", "settings",
		},
		DashboardProfile: "console.default",
		LandingSummary:   "Platform administration and governed operations.",
		RoutePrefix:      "/admin",
	},
	{
		AppID:            AppMarketOps,
		Label:            "MarketOps",
		DefaultRoute:     "/marketops/dashboard",
		Domains:          []string{"market_data"},
		EnabledModules:   []string{"dashboard", "symbols", "option_contracts", "signals", "alerts", "replay", "providers", "pipelines", "health"},
		DashboardProfile: "marketdata.default",
		LandingSummary:   "Strategic financial context and daily market evidence for disciplined analyst review.",
		RoutePrefix:      "/marketops",
	},
	{
		AppID:            AppCyberOps,
		Label:            "CyberOps",
		DefaultRoute:     "/cyberops/dashboard",
		Domains:          []string{"security"},
		EnabledModules:   []string{"dashboard", "anomalies", "signals", "alerts", "insights", "settings"},
		DashboardProfile: "security.default",
		LandingSummary:   "Firewall evidence, deterministic detections, and focused security investigation.",
		RoutePrefix:      "/cyberops",
	},
}

func Normalize(appID, domain, useCase, fallbackDomain string) Metadata {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		appID = DefaultAppID
	}
	domain = strings.TrimSpace(domain)
	if domain == "" {
		domain = strings.TrimSpace(fallbackDomain)
	}
	useCase = strings.TrimSpace(useCase)
	if useCase == "" {
		useCase = DefaultUseCase
	}
	return Metadata{AppID: appID, Domain: domain, UseCase: useCase}
}

// ProfileByID returns the canonical registered application profile.
func ProfileByID(appID string) (Profile, bool) {
	for _, profile := range Profiles {
		if profile.AppID == strings.TrimSpace(appID) {
			return profile, true
		}
	}
	return Profile{}, false
}

// UseCaseProfileForPath resolves a registered use case from its browser/API
// route prefix. Console is deliberately excluded because it is super-admin-only.
func UseCaseProfileForPath(path string) (Profile, bool) {
	for _, profile := range Profiles {
		if profile.AppID == AppConsole || profile.RoutePrefix == "" {
			continue
		}
		if path == profile.RoutePrefix || strings.HasPrefix(path, profile.RoutePrefix+"/") || strings.Contains(path, profile.RoutePrefix+"/") {
			return profile, true
		}
	}
	return Profile{}, false
}

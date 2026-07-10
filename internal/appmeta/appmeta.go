package appmeta

import "strings"

const (
	DefaultAppID   = "console"
	DefaultUseCase = "general"

	AppConsole   = "console"
	AppMarketOps = "marketops"
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
}

var Profiles = []Profile{
	{
		AppID:        AppConsole,
		Label:        "SignalOps Console",
		DefaultRoute: "/dashboard",
		Domains:      []string{"market_data", "crm", "security", "operations", "iot", "procurement", "custom"},
		EnabledModules: []string{
			"dashboard", "event_explorer", "timeline", "correlation", "insights",
			"pipelines", "rules", "sources", "health", "replay", "administration", "settings",
		},
		DashboardProfile: "console.default",
	},
	{
		AppID:            AppMarketOps,
		Label:            "MarketOps",
		DefaultRoute:     "/marketops/dashboard",
		Domains:          []string{"market_data"},
		EnabledModules:   []string{"dashboard", "symbols", "option_contracts", "signals", "alerts", "replay", "providers", "pipelines", "health"},
		DashboardProfile: "marketdata.default",
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

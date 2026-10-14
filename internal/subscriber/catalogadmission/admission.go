// Package catalogadmission maps retained Massive reference data to the strict
// S2 US-common-stock eligibility evidence contract.
package catalogadmission

import (
	"strings"

	"github.com/lukebabs/signalops/internal/adapters/marketdata/massive"
)

type Decision struct {
	Decision   string
	ReasonCode string
	Evidence   map[string]any
}

func Evaluate(details massive.TickerDetails) Decision {
	countryCode := ""
	if strings.EqualFold(strings.TrimSpace(details.Locale), "us") {
		countryCode = "US"
	}
	securityType := "other"
	if strings.EqualFold(strings.TrimSpace(details.Market), "stocks") && strings.EqualFold(strings.TrimSpace(details.Type), "cs") {
		securityType = "common_stock"
	}
	exchangeListed := strings.TrimSpace(details.Exchange) != ""
	providerEligible := strings.TrimSpace(details.Ticker) != ""
	evidence := map[string]any{
		"provider":          "massive",
		"provider_ticker":   details.Ticker,
		"provider_name":     details.Name,
		"locale":            details.Locale,
		"market":            details.Market,
		"provider_type":     details.Type,
		"primary_exchange":  details.Exchange,
		"country_code":      countryCode,
		"security_type":     securityType,
		"exchange_listed":   exchangeListed,
		"provider_eligible": providerEligible,
		"is_active":         details.Active,
	}
	if countryCode == "US" && securityType == "common_stock" && exchangeListed && providerEligible && details.Active {
		return Decision{Decision: "eligible", ReasonCode: "massive_us_common_stock_active", Evidence: evidence}
	}
	return Decision{Decision: "ineligible", ReasonCode: "massive_reference_outside_us_common_stock_policy", Evidence: evidence}
}

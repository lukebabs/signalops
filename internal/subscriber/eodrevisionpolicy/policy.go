// Package eodrevisionpolicy centralizes deterministic EOD observation choices.
package eodrevisionpolicy

import "fmt"

type UsageContext string
type ObservationRole string

const (
	HistoricalAssurance  UsageContext = "historical_assurance"
	CurrentMarketContext UsageContext = "current_market_context"

	InitialCapture      ObservationRole = "initial_tenant_local_capture"
	LatestVerifiedValue ObservationRole = "global_reobservation"
)

// Selection is the immutable consumer contract: the context determines which
// version is selected, and the returned as-of time must accompany the result.
type Selection struct {
	UsageContext            UsageContext
	SelectedObservationRole ObservationRole
	PolicyVersion           string
}

func SelectionFor(context UsageContext) (Selection, error) {
	switch context {
	case HistoricalAssurance:
		return Selection{UsageContext: context, SelectedObservationRole: InitialCapture, PolicyVersion: "s4-as-of-selection-v1"}, nil
	case CurrentMarketContext:
		return Selection{UsageContext: context, SelectedObservationRole: LatestVerifiedValue, PolicyVersion: "s4-as-of-selection-v1"}, nil
	default:
		return Selection{}, fmt.Errorf("unsupported EOD revision usage context %q", context)
	}
}

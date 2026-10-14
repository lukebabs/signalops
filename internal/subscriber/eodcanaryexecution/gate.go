// Package eodcanaryexecution defines the fail-closed S4 canary execution gate.
// It constructs evidence expectations only; it has no provider dependency.
package eodcanaryexecution

import (
	"errors"
	"sort"
	"strings"
)

const (
	Version                 = "s4-canary-execution-gate-v1"
	MaximumProviderRequests = 2
	ExpectedWorkerIdentity  = "subscriber-global-eod-reconciler"
)

type Member struct {
	GlobalAssetID string
	Ticker        string
	Priority      int
}

type ExecutionMember struct {
	GlobalAssetID  string
	Ticker         string
	RequestOrdinal int
}

// Freeze validates the controlled identity and turns the canary membership
// into exactly bounded request slots. Calling it cannot enable a provider.
func Freeze(identity string, members []Member, maxProviderRequests int) ([]ExecutionMember, error) {
	if strings.TrimSpace(identity) != ExpectedWorkerIdentity {
		return nil, errors.New("unexpected global EOD worker identity")
	}
	if maxProviderRequests <= 0 || maxProviderRequests > MaximumProviderRequests {
		return nil, errors.New("provider request budget must be between 1 and 2")
	}
	if len(members) != maxProviderRequests {
		return nil, errors.New("frozen canary membership must exactly match provider request budget")
	}
	ordered := append([]Member(nil), members...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Priority == ordered[j].Priority {
			return ordered[i].GlobalAssetID < ordered[j].GlobalAssetID
		}
		return ordered[i].Priority < ordered[j].Priority
	})
	seenIDs, seenTickers := map[string]bool{}, map[string]bool{}
	result := make([]ExecutionMember, 0, len(ordered))
	for index, member := range ordered {
		member.GlobalAssetID = strings.TrimSpace(member.GlobalAssetID)
		member.Ticker = strings.ToUpper(strings.TrimSpace(member.Ticker))
		if member.GlobalAssetID == "" || member.Ticker == "" || member.Priority <= 0 || seenIDs[member.GlobalAssetID] || seenTickers[member.Ticker] {
			return nil, errors.New("invalid or duplicate frozen canary member")
		}
		seenIDs[member.GlobalAssetID], seenTickers[member.Ticker] = true, true
		result = append(result, ExecutionMember{GlobalAssetID: member.GlobalAssetID, Ticker: member.Ticker, RequestOrdinal: index + 1})
	}
	return result, nil
}

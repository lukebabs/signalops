// Package eodcanary deterministically derives a small, frozen S4 canary from
// an existing S2 shadow plan. It is side-effect free and never starts work.
package eodcanary

import (
	"errors"
	"sort"
	"strings"
)

const (
	Version           = "s4-shared-eod-canary-v1"
	MaximumCanarySize = 50
)

type Member struct {
	GlobalAssetID string
	Priority      int
	SourceRank    int
}

func Select(members []Member, maxSymbols int) ([]Member, error) {
	if maxSymbols <= 0 || maxSymbols > MaximumCanarySize {
		return nil, errors.New("canary size must be between 1 and 50")
	}
	if len(members) == 0 {
		return nil, errors.New("canary requires at least one shadow-plan member")
	}
	selected := append([]Member(nil), members...)
	seenIDs := make(map[string]struct{}, len(selected))
	seenPriorities := make(map[int]struct{}, len(selected))
	for index := range selected {
		selected[index].GlobalAssetID = strings.TrimSpace(selected[index].GlobalAssetID)
		if selected[index].GlobalAssetID == "" || selected[index].Priority <= 0 {
			return nil, errors.New("invalid shadow-plan member")
		}
		if _, exists := seenIDs[selected[index].GlobalAssetID]; exists {
			return nil, errors.New("duplicate shadow-plan global asset")
		}
		if _, exists := seenPriorities[selected[index].Priority]; exists {
			return nil, errors.New("duplicate shadow-plan priority")
		}
		seenIDs[selected[index].GlobalAssetID] = struct{}{}
		seenPriorities[selected[index].Priority] = struct{}{}
	}
	sort.Slice(selected, func(left, right int) bool {
		if selected[left].Priority != selected[right].Priority {
			return selected[left].Priority < selected[right].Priority
		}
		return selected[left].GlobalAssetID < selected[right].GlobalAssetID
	})
	if len(selected) > maxSymbols {
		selected = selected[:maxSymbols]
	}
	return selected, nil
}

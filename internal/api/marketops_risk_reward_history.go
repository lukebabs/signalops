package api

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const riskRewardMinimumUsableInputs = 5

type riskRewardOverviewCandidate struct {
	member   signalOverviewMember
	date     string
	eligible bool
	usable   int
	created  time.Time
}

func signalOverviewRiskRewardBest(results []storage.AlgorithmResultRecord, active map[string]struct{}, days int) []map[string]any {
	candidates := make([]riskRewardOverviewCandidate, 0, len(results))
	for _, result := range results {
		payload := map[string]any{}
		if json.Unmarshal(result.ResultPayloadJSON, &payload) != nil {
			continue
		}
		candidate, ok := riskRewardCandidateFromPayload(payload, result.CreatedAt, active)
		if ok {
			candidates = append(candidates, candidate)
		}
	}
	return riskRewardOverviewPoints(candidates, active, days)
}

func signalOverviewRiskRewardSnapshots(snapshots []storage.MarketOpsRiskRewardSnapshotRecord, active map[string]struct{}, days int) []map[string]any {
	candidates := make([]riskRewardOverviewCandidate, 0, len(snapshots))
	for _, snapshot := range snapshots {
		payload := map[string]any{}
		if json.Unmarshal(snapshot.ResultPayloadJSON, &payload) != nil {
			continue
		}
		candidate, ok := riskRewardCandidateFromPayload(payload, snapshot.CreatedAt, active)
		if !ok {
			continue
		}
		candidate.usable = snapshot.UsableInputCount
		candidate.eligible = snapshot.Eligible && snapshot.UsableInputCount >= riskRewardMinimumUsableInputs
		candidates = append(candidates, candidate)
	}
	return riskRewardOverviewPoints(candidates, active, days)
}

func riskRewardCandidateFromPayload(payload map[string]any, created time.Time, active map[string]struct{}) (riskRewardOverviewCandidate, bool) {
	symbol := strings.ToUpper(stringAny(payload["symbol"]))
	if _, ok := active[symbol]; !ok {
		return riskRewardOverviewCandidate{}, false
	}
	date := observationDate(payload)
	score, ok := numberAny(payload["technical_score"])
	if date == "" || !ok {
		return riskRewardOverviewCandidate{}, false
	}
	direction := strings.ToLower(stringAny(payload["technical_direction"]))
	if direction != "bullish" && direction != "bearish" {
		direction = "neutral"
	}
	usable := 0
	if basis, ok := payload["confidence_basis"].(map[string]any); ok {
		if value, ok := numberAny(basis["usable_technical_inputs"]); ok {
			usable = int(value)
		}
	}
	return riskRewardOverviewCandidate{member: signalOverviewMember{Ticker: symbol, Label: direction, Score: &score, AsOf: date}, date: date, eligible: usable >= riskRewardMinimumUsableInputs, usable: usable, created: created}, true
}

func riskRewardOverviewPoints(candidates []riskRewardOverviewCandidate, active map[string]struct{}, days int) []map[string]any {
	byDate := map[string]map[string]riskRewardOverviewCandidate{}
	for _, candidate := range candidates {
		if byDate[candidate.date] == nil {
			byDate[candidate.date] = map[string]riskRewardOverviewCandidate{}
		}
		if current, exists := byDate[candidate.date][candidate.member.Ticker]; !exists || betterRiskRewardCandidate(candidate, current) {
			byDate[candidate.date][candidate.member.Ticker] = candidate
		}
	}
	dates := make([]string, 0, len(byDate))
	for date := range byDate {
		dates = append(dates, date)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))
	if len(dates) > days {
		dates = dates[:days]
	}
	sort.Strings(dates)
	points := make([]map[string]any, 0, len(dates))
	for _, date := range dates {
		categories := map[string][]signalOverviewMember{"bullish": {}, "neutral": {}, "bearish": {}}
		eligible, insufficient := 0, 0
		for _, candidate := range byDate[date] {
			if !candidate.eligible {
				insufficient++
				continue
			}
			eligible++
			categories[candidate.member.Label] = append(categories[candidate.member.Label], candidate.member)
		}
		entries := make([]map[string]any, 0, 3)
		for _, category := range []string{"bullish", "neutral", "bearish"} {
			members := categories[category]
			sort.Slice(members, func(i, j int) bool { return members[i].Ticker < members[j].Ticker })
			entries = append(entries, map[string]any{"key": category, "count": len(members), "members": members})
		}
		points = append(points, map[string]any{"trade_date": date, "categories": entries, "coverage": map[string]int{"eligible": eligible, "insufficient_inputs": insufficient, "unprocessed": max(0, len(active)-eligible-insufficient)}})
	}
	return points
}

func betterRiskRewardCandidate(candidate, current riskRewardOverviewCandidate) bool {
	if candidate.eligible != current.eligible {
		return candidate.eligible
	}
	if candidate.usable != current.usable {
		return candidate.usable > current.usable
	}
	return candidate.created.After(current.created)
}

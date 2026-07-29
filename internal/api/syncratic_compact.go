package api

import (
	"context"
	"sort"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const (
	maxSyncraticSignals        = 12
	maxSyncraticAlerts         = 12
	maxSyncraticArtifacts      = 12
	maxSyncraticProposals      = 12
	maxSyncraticLabels         = 12
	maxSyncraticTransitions    = 12
	maxSyncraticMarketEvidence = 12
	maxSyncraticEventIDs       = 50
)

type syncraticContextLedger struct {
	Signals []storage.SignalLedgerRecord
	Alerts  []storage.AlertLedgerRecord
}

func loadSyncraticContextLedger(ctx context.Context, repo storage.QueryRepository, tenantID string, windowStart, windowEnd time.Time, signalLimit, alertLimit int) (syncraticContextLedger, error) {
	filters := storage.SignalLedgerFilter{TenantID: tenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", WindowStart: windowStart.UTC(), WindowEnd: windowEnd.UTC(), Limit: signalLimitOrDefault(signalLimit)}
	signals, err := repo.ListSignalLedger(ctx, filters)
	if err != nil {
		return syncraticContextLedger{}, err
	}
	alerts, err := repo.ListAlertLedger(ctx, storage.AlertLedgerFilter{TenantID: tenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", WindowStart: windowStart.UTC(), WindowEnd: windowEnd.UTC(), Limit: alertLimitOrDefault(alertLimit)})
	if err != nil {
		return syncraticContextLedger{}, err
	}
	return syncraticContextLedger{Signals: signals, Alerts: alerts}, nil
}

func compactSyncraticSignals(subject string, records []storage.SignalLedgerRecord, windowStart, windowEnd time.Time, allowedTypes map[string]struct{}) ([]storage.SignalLedgerRecord, int) {
	eligible := make([]storage.SignalLedgerRecord, 0, len(records))
	for _, record := range records {
		if !timeInWindow(record.SignalTime, windowStart, windowEnd) || !recordEvidenceMatchesSymbol(subject, record.EntitiesJSON, record.EventJSON, record.SemanticEvidenceJSON, record.EvidenceJSON) {
			continue
		}
		if len(allowedTypes) > 0 {
			if _, ok := allowedTypes[record.SignalType]; !ok {
				continue
			}
		}
		eligible = append(eligible, record)
	}
	sort.Slice(eligible, func(i, j int) bool {
		if severityRank(eligible[i].Severity) != severityRank(eligible[j].Severity) {
			return severityRank(eligible[i].Severity) > severityRank(eligible[j].Severity)
		}
		if eligible[i].Confidence != eligible[j].Confidence {
			return eligible[i].Confidence > eligible[j].Confidence
		}
		if !eligible[i].SignalTime.Equal(eligible[j].SignalTime) {
			return eligible[i].SignalTime.After(eligible[j].SignalTime)
		}
		return eligible[i].SignalID < eligible[j].SignalID
	})
	return limitSyncraticSignals(eligible), len(eligible)
}

func compactSyncraticAlerts(subject string, records []storage.AlertLedgerRecord, windowStart, windowEnd time.Time) ([]storage.AlertLedgerRecord, int) {
	eligible := make([]storage.AlertLedgerRecord, 0, len(records))
	for _, record := range records {
		if timeInWindow(record.LastObservedAt, windowStart, windowEnd) && recordEvidenceMatchesSymbol(subject, record.EntitiesJSON, record.EvidenceJSON) {
			eligible = append(eligible, record)
		}
	}
	sort.Slice(eligible, func(i, j int) bool {
		if severityRank(eligible[i].Severity) != severityRank(eligible[j].Severity) {
			return severityRank(eligible[i].Severity) > severityRank(eligible[j].Severity)
		}
		if !eligible[i].LastObservedAt.Equal(eligible[j].LastObservedAt) {
			return eligible[i].LastObservedAt.After(eligible[j].LastObservedAt)
		}
		return eligible[i].AlertID < eligible[j].AlertID
	})
	return limitSyncraticAlerts(eligible), len(eligible)
}

func limitSyncraticSignals(records []storage.SignalLedgerRecord) []storage.SignalLedgerRecord {
	if len(records) <= maxSyncraticSignals {
		return records
	}
	return records[:maxSyncraticSignals]
}

func limitSyncraticAlerts(records []storage.AlertLedgerRecord) []storage.AlertLedgerRecord {
	if len(records) <= maxSyncraticAlerts {
		return records
	}
	return records[:maxSyncraticAlerts]
}

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/marketops/signalassurance"
	"github.com/lukebabs/signalops/internal/storage"
	"github.com/lukebabs/signalops/pkg/broker"
)

type signalAssurancePriceSource interface {
	ListMarketOpsBacktestNormalizedEvents(context.Context, storage.MarketOpsBacktestEventFilter) ([]storage.NormalizedEventLedgerRecord, error)
}

func publishResearchEligibleFromMaterialization(ctx context.Context, cfg RouterConfig, proposal storage.AlgorithmSignalProposalRecord, result storage.AlgorithmResultRecord, materialization storage.AlgorithmSignalMaterializationRecord, signal storage.SignalLedgerRecord) error {
	if cfg.Publisher == nil {
		return fmt.Errorf("SAF publisher is not configured")
	}
	source, ok := cfg.QueryRepository.(signalAssurancePriceSource)
	if !ok {
		return fmt.Errorf("SAF canonical price source is unavailable")
	}
	var proposalPayload struct {
		Symbol string `json:"symbol"`
	}
	_ = json.Unmarshal(proposal.ProposalPayloadJSON, &proposalPayload)
	symbol := strings.ToUpper(strings.TrimSpace(proposalPayload.Symbol))
	if symbol == "" {
		return fmt.Errorf("materialization proposal symbol is missing")
	}
	events, err := source.ListMarketOpsBacktestNormalizedEvents(ctx, storage.MarketOpsBacktestEventFilter{TenantID: signal.TenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", Dataset: "equity_eod_prices", Symbols: []string{symbol, "SPY"}, WindowStart: signal.SignalTime.AddDate(0, 0, -10), WindowEnd: signal.SignalTime.AddDate(0, 0, 1), Limit: 100})
	if err != nil {
		return err
	}
	event, err := signalassurance.EligibleFromMaterialization(materialization, proposal, result, signal, events, "SPY")
	if err != nil {
		return err
	}
	value, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = cfg.Publisher.Publish(ctx, broker.Message{Topic: broker.TopicName("", broker.MarketOpsSignalAssuranceEligibleTopic), Key: event.EligibleEventID, Value: value, CorrelationID: event.EligibleEventID, CausationID: materialization.MaterializationID, TraceID: signal.TraceID, PublishedAt: event.EventAvailableAt})
	return err
}

package storage

import (
	"context"
	"time"
)

const (
	MarketOpsOutcomeSourceHypothesisEvaluation = "hypothesis_evaluation"
	MarketOpsOutcomeSourceOpportunity          = "opportunity"
	MarketOpsOutcomeSourceSignal               = "signal"

	MarketOpsOutcomePending      = "pending"
	MarketOpsOutcomeMatured      = "matured"
	MarketOpsOutcomeMissingPrice = "missing_price"
)

type MarketOpsSignalOutcomeRecord struct {
	OutcomeID             string
	TenantID              string
	AppID                 string
	SourceType            string
	SourceID              string
	HypothesisKey         string
	HypothesisVersion     string
	AssetID               string
	Symbol                string
	Direction             string
	OriginSessionDate     time.Time
	HorizonSessions       int
	MaturedSessionDate    *time.Time
	OutcomeStatus         string
	ForwardReturn         *float64
	MaxFavorableExcursion *float64
	MaxAdverseExcursion   *float64
	MaximumDrawdown       *float64
	RealizedVolChange     *float64
	DirectionalHit        *bool
	ThresholdHit          *bool
	DaysToThreshold       *int
	OriginEventID         string
	OutcomeEventIDs       []string
	OutcomePayloadJSON    []byte
	CalculationVersion    string
	CalculationRunID      string
	DeterministicKey      string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type MarketOpsSignalOutcomeFilter struct {
	TenantID          string
	AppID             string
	SourceType        string
	SourceID          string
	HypothesisKey     string
	HypothesisVersion string
	Symbol            string
	Direction         string
	OutcomeStatus     string
	HorizonSessions   int
	OriginStart       time.Time
	OriginEnd         time.Time
	Limit             int
}

type MarketOpsOutcomeWriteRepository interface {
	UpsertMarketOpsSignalOutcome(context.Context, MarketOpsSignalOutcomeRecord) error
}

type MarketOpsOutcomeQueryRepository interface {
	ListMarketOpsSignalOutcomes(context.Context, MarketOpsSignalOutcomeFilter) ([]MarketOpsSignalOutcomeRecord, error)
	GetMarketOpsSignalOutcome(context.Context, string, string) (MarketOpsSignalOutcomeRecord, error)
}

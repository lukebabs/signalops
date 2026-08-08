package signalassurance

import (
	"encoding/json"
	"fmt"

	"github.com/lukebabs/signalops/internal/storage"
)

// CompatibleOutcomeProjection maps a materialized LIVE assertion into the
// pre-existing immutable MarketOps forward-outcome ledger only when its actual
// session horizon is one of that ledger's supported fixed horizons. SAF never
// creates a parallel outcome for other horizons.
func CompatibleOutcomeProjection(assertion storage.SignalAssertionRecord, evaluation storage.SignalAssertionEvaluationRecord, nextState string) *storage.MarketOpsSignalOutcomeRecord {
	if assertion.EvaluationMode != storage.SignalAssuranceModeLive || nextState != storage.SignalAssertionMaterialized || !compatibleHorizon(evaluation.TradingDaysActive) || evaluation.AbsoluteReturn == nil {
		return nil
	}
	direction := "upside"
	if assertion.SignalDirection == "bearish" {
		direction = "downside"
	}
	payload, _ := json.Marshal(map[string]any{"projected_by": "signal_assurance.v1.1", "assertion_id": assertion.AssertionID, "evaluation_id": evaluation.EvaluationID, "input_completeness": evaluation.InputCompleteness})
	key := fmt.Sprintf("saf-projection|%s|%s|%d|signal_assurance.v1.1", assertion.TenantID, assertion.SourceLedgerSignalID, evaluation.TradingDaysActive)
	return &storage.MarketOpsSignalOutcomeRecord{OutcomeID: deterministicID("moutcome", key), TenantID: assertion.TenantID, AppID: "marketops", SourceType: storage.MarketOpsOutcomeSourceSignal, SourceID: assertion.SourceLedgerSignalID, AssetID: assertion.AssetID, Symbol: assertion.Symbol, Direction: direction, OriginSessionDate: assertion.ConfirmedAt, HorizonSessions: evaluation.TradingDaysActive, MaturedSessionDate: &evaluation.EvaluationSessionDate, OutcomeStatus: storage.MarketOpsOutcomeMatured, ForwardReturn: evaluation.AbsoluteReturn, MaxFavorableExcursion: evaluation.MFE, MaxAdverseExcursion: evaluation.MAE, DirectionalHit: boolPtr(true), ThresholdHit: boolPtr(true), OriginEventID: assertion.SourceLedgerSignalID, OutcomePayloadJSON: payload, CalculationVersion: "signal_assurance.v1.1", CalculationRunID: evaluation.EvaluationID, DeterministicKey: key}
}

func boolPtr(value bool) *bool { return &value }

func compatibleHorizon(horizon int) bool {
	return horizon == 1 || horizon == 5 || horizon == 10 || horizon == 20
}

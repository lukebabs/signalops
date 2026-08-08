package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) UpsertSignalValidationContract(ctx context.Context, x storage.SignalValidationContractRecord) error {
	if err := validateSignalValidationContract(x); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO signal_validation_contracts (contract_id,signal_type,contract_version,algorithm,algorithm_version,direction,primary_metric,comparison_operator,threshold,evaluation_windows,max_horizon_trading_days,materialization_policy,invalidation_policy,config,active,contract_scope_key,created_at) VALUES ($1,$2,$3,NULLIF($4,''),NULLIF($5,''),$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17) ON CONFLICT (contract_id) DO NOTHING`, x.ContractID, x.SignalType, x.ContractVersion, x.Algorithm, x.AlgorithmVersion, x.Direction, x.PrimaryMetric, x.ComparisonOperator, x.Threshold, jsonOrEmpty(x.EvaluationWindowsJSON), x.MaxHorizonTradingDays, x.MaterializationPolicy, x.InvalidationPolicy, jsonOrEmpty(x.ConfigJSON), x.Active, x.ContractScopeKey, timeOrNow(x.CreatedAt))
	if err != nil {
		return fmt.Errorf("insert signal validation contract: %w", err)
	}
	return nil
}

func (r *Repository) GetSignalValidationContract(ctx context.Context, contractID string) (storage.SignalValidationContractRecord, error) {
	return scanSignalValidationContract(r.db.QueryRowContext(ctx, signalValidationContractSelect+` WHERE contract_id=$1`, strings.TrimSpace(contractID)))
}

func (r *Repository) RegisterSignalAssuranceAssertion(ctx context.Context, registration storage.SignalAssuranceRegistration) (storage.SignalAssertionRecord, bool, error) {
	if err := validateSignalAssuranceRegistration(registration); err != nil {
		return storage.SignalAssertionRecord{}, false, err
	}
	ledger, err := r.GetSignalLedger(ctx, registration.Event.SignalLedgerID)
	if err != nil {
		return storage.SignalAssertionRecord{}, false, fmt.Errorf("load source ledger signal: %w", err)
	}
	if ledger.TenantID != registration.Event.TenantID || ledger.SignalID != registration.Event.SignalID {
		return storage.SignalAssertionRecord{}, false, errors.New("eligible event must reference an existing same-tenant signal ledger record")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return storage.SignalAssertionRecord{}, false, err
	}
	defer func() { _ = tx.Rollback() }()
	var existingID string
	err = tx.QueryRowContext(ctx, `SELECT assertion_id FROM signal_assertions WHERE tenant_id=$1 AND registration_idempotency_key=$2`, registration.Assertion.TenantID, registration.Assertion.RegistrationIdempotencyKey).Scan(&existingID)
	if err == nil {
		assertion, scanErr := scanSignalAssertion(tx.QueryRowContext(ctx, signalAssertionSelect+` WHERE assertion_id=$1`, existingID))
		if scanErr != nil {
			return storage.SignalAssertionRecord{}, false, scanErr
		}
		if _, execErr := tx.ExecContext(ctx, `INSERT INTO signal_assurance_registration_inbox (tenant_id,eligible_event_id,signal_ledger_id,assertion_id,payload) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (tenant_id,eligible_event_id) DO NOTHING`, registration.Event.TenantID, registration.Event.EligibleEventID, registration.Event.SignalLedgerID, assertion.AssertionID, jsonOrEmpty(registration.PayloadJSON)); execErr != nil {
			return storage.SignalAssertionRecord{}, false, execErr
		}
		if err := tx.Commit(); err != nil {
			return storage.SignalAssertionRecord{}, false, err
		}
		return assertion, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return storage.SignalAssertionRecord{}, false, err
	}
	x := registration.Assertion
	_, err = tx.ExecContext(ctx, `INSERT INTO signal_assertions (assertion_id,tenant_id,asset_id,symbol,signal_id,source_ledger_signal_id,signal_type,signal_direction,signal_score,algorithm,algorithm_version,confirmed_at,state,evaluation_mode,evaluation_run_id,registration_idempotency_key,validation_contract_id,validation_contract_version,validation_contract,evaluation_engine_version,baseline_snapshot,baseline_provenance,transition_sequence) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,NULLIF($15,''),$16,$17,$18,$19,$20,$21,$22,$23)`, x.AssertionID, x.TenantID, x.AssetID, strings.ToUpper(x.Symbol), x.SignalID, x.SourceLedgerSignalID, x.SignalType, x.SignalDirection, x.SignalScore, x.Algorithm, x.AlgorithmVersion, x.ConfirmedAt, x.State, x.EvaluationMode, x.EvaluationRunID, x.RegistrationIdempotencyKey, x.ValidationContractID, x.ValidationContractVersion, jsonOrEmpty(x.ValidationContractJSON), x.EvaluationEngineVersion, jsonOrEmpty(x.BaselineSnapshotJSON), jsonOrEmpty(x.BaselineProvenanceJSON), x.TransitionSequence)
	if err != nil {
		return storage.SignalAssertionRecord{}, false, fmt.Errorf("insert signal assertion: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO signal_assurance_registration_inbox (tenant_id,eligible_event_id,signal_ledger_id,assertion_id,payload) VALUES ($1,$2,$3,$4,$5)`, registration.Event.TenantID, registration.Event.EligibleEventID, registration.Event.SignalLedgerID, x.AssertionID, jsonOrEmpty(registration.PayloadJSON)); err != nil {
		return storage.SignalAssertionRecord{}, false, err
	}
	eventID := "saf_event_" + x.AssertionID + "_1_created"
	_, err = tx.ExecContext(ctx, `INSERT INTO signal_assertion_events (event_id,assertion_id,event_type,previous_state,new_state,occurred_at,transition_sequence,evaluation_mode,evaluation_run_id,idempotency_key) VALUES ($1,$2,'ASSERTION_CREATED','',$3,$4,1,$5,NULLIF($6,''),$7)`, eventID, x.AssertionID, x.State, x.ConfirmedAt, x.EvaluationMode, x.EvaluationRunID, "created|"+x.AssertionID)
	if err != nil {
		return storage.SignalAssertionRecord{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return storage.SignalAssertionRecord{}, false, err
	}
	assertion, err := r.GetSignalAssuranceAssertion(ctx, x.TenantID, x.AssertionID)
	return assertion, true, err
}

func (r *Repository) PersistSignalAssuranceEvaluation(ctx context.Context, persistence storage.SignalAssuranceEvaluationPersistence) (storage.SignalAssertionEvaluationRecord, bool, error) {
	if err := validateSignalAssuranceEvaluation(persistence.Evaluation); err != nil {
		return storage.SignalAssertionEvaluationRecord{}, false, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return storage.SignalAssertionEvaluationRecord{}, false, err
	}
	defer func() { _ = tx.Rollback() }()
	assertion, err := scanSignalAssertion(tx.QueryRowContext(ctx, signalAssertionSelect+` WHERE assertion_id=$1 FOR UPDATE`, persistence.Evaluation.AssertionID))
	if err != nil {
		return storage.SignalAssertionEvaluationRecord{}, false, err
	}
	if assertion.State != persistence.PreviousState {
		return storage.SignalAssertionEvaluationRecord{}, false, storage.ErrConflict
	}
	existing, existingErr := scanSignalAssuranceEvaluation(tx.QueryRowContext(ctx, signalAssuranceEvaluationSelect+` WHERE assertion_id=$1 AND evaluation_session_date=$2 AND evaluation_version=$3`, persistence.Evaluation.AssertionID, persistence.Evaluation.EvaluationSessionDate, persistence.Evaluation.EvaluationVersion))
	if existingErr == nil {
		if err := tx.Commit(); err != nil {
			return storage.SignalAssertionEvaluationRecord{}, false, err
		}
		return existing, false, nil
	}
	if !errors.Is(existingErr, sql.ErrNoRows) {
		return storage.SignalAssertionEvaluationRecord{}, false, existingErr
	}
	x := persistence.Evaluation
	transitionSequence := assertion.TransitionSequence
	if persistence.NextState != persistence.PreviousState {
		transitionSequence++
	}
	x.TransitionSequence = transitionSequence
	_, err = tx.ExecContext(ctx, `INSERT INTO signal_assertion_evaluations (evaluation_id,assertion_id,evaluated_at,evaluation_session_date,evaluation_mode,evaluation_run_id,input_snapshot,input_completeness,transition_sequence,trading_days_active,calendar_days_active,asset_price,benchmark_price,absolute_return,benchmark_return,benchmark_relative_return,mfe,mae,materialization_condition_met,invalidation_condition_met,evaluation_version) VALUES ($1,$2,$3,$4,$5,NULLIF($6,''),$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)`, x.EvaluationID, x.AssertionID, x.EvaluatedAt, x.EvaluationSessionDate, x.EvaluationMode, x.EvaluationRunID, jsonOrEmpty(x.InputSnapshotJSON), x.InputCompleteness, x.TransitionSequence, x.TradingDaysActive, x.CalendarDaysActive, x.AssetPrice, x.BenchmarkPrice, x.AbsoluteReturn, x.BenchmarkReturn, x.BenchmarkRelativeReturn, x.MFE, x.MAE, x.MaterializationConditionMet, x.InvalidationConditionMet, x.EvaluationVersion)
	if err != nil {
		return storage.SignalAssertionEvaluationRecord{}, false, fmt.Errorf("insert assertion evaluation: %w", err)
	}
	if persistence.NextState != persistence.PreviousState {
		column := terminalTimestampColumn(persistence.NextState)
		if column == "" {
			return storage.SignalAssertionEvaluationRecord{}, false, errors.New("invalid assertion lifecycle transition")
		}
		query := `UPDATE signal_assertions SET state=$1, transition_sequence=$2, updated_at=now(), ` + column + `=$3 WHERE assertion_id=$4`
		if _, err = tx.ExecContext(ctx, query, persistence.NextState, transitionSequence, x.EvaluatedAt, x.AssertionID); err != nil {
			return storage.SignalAssertionEvaluationRecord{}, false, err
		}
		eventType := "ASSERTION_" + persistence.NextState
		eventID := fmt.Sprintf("saf_event_%s_%d_%s", x.AssertionID, transitionSequence, strings.ToLower(persistence.NextState))
		_, err = tx.ExecContext(ctx, `INSERT INTO signal_assertion_events (event_id,assertion_id,event_type,previous_state,new_state,reason_code,details,occurred_at,transition_sequence,evaluation_id,evaluation_mode,evaluation_run_id,idempotency_key) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NULLIF($12,''),$13)`, eventID, x.AssertionID, eventType, persistence.PreviousState, persistence.NextState, persistence.ReasonCode, jsonOrEmpty(persistence.EventDetailsJSON), x.EvaluatedAt, transitionSequence, x.EvaluationID, x.EvaluationMode, x.EvaluationRunID, fmt.Sprintf("%s|%d|%s", x.AssertionID, transitionSequence, eventType))
		if err != nil {
			return storage.SignalAssertionEvaluationRecord{}, false, err
		}
	}
	if err = tx.Commit(); err != nil {
		return storage.SignalAssertionEvaluationRecord{}, false, err
	}
	return x, true, nil
}

func (r *Repository) ListSignalAssuranceAssertions(ctx context.Context, f storage.SignalAssuranceAssertionFilter) ([]storage.SignalAssertionRecord, error) {
	rows, err := r.db.QueryContext(ctx, signalAssertionSelect+` WHERE ($1='' OR tenant_id=$1) AND ($2='' OR state=$2) AND ($3='' OR evaluation_mode=$3) AND ($4='' OR symbol=$4) ORDER BY confirmed_at DESC LIMIT $5`, strings.TrimSpace(f.TenantID), strings.TrimSpace(f.State), strings.TrimSpace(f.EvaluationMode), strings.ToUpper(strings.TrimSpace(f.Symbol)), clampLimit(f.Limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []storage.SignalAssertionRecord{}
	for rows.Next() {
		x, scanErr := scanSignalAssertion(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *Repository) GetSignalAssuranceAssertion(ctx context.Context, tenantID, assertionID string) (storage.SignalAssertionRecord, error) {
	return scanSignalAssertion(r.db.QueryRowContext(ctx, signalAssertionSelect+` WHERE tenant_id=$1 AND assertion_id=$2`, strings.TrimSpace(tenantID), strings.TrimSpace(assertionID)))
}
func (r *Repository) ListSignalAssuranceEvaluations(ctx context.Context, f storage.SignalAssuranceEvaluationFilter) ([]storage.SignalAssertionEvaluationRecord, error) {
	rows, err := r.db.QueryContext(ctx, signalAssuranceEvaluationSelect+` WHERE assertion_id=$1 ORDER BY evaluation_session_date DESC LIMIT $2`, strings.TrimSpace(f.AssertionID), clampLimit(f.Limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []storage.SignalAssertionEvaluationRecord{}
	for rows.Next() {
		x, e := scanSignalAssuranceEvaluation(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

const signalValidationContractSelect = `SELECT contract_id,signal_type,contract_version,COALESCE(algorithm,''),COALESCE(algorithm_version,''),direction,primary_metric,comparison_operator,threshold,evaluation_windows,max_horizon_trading_days,materialization_policy,invalidation_policy,config,active,contract_scope_key,created_at FROM signal_validation_contracts`
const signalAssertionSelect = `SELECT assertion_id,tenant_id,asset_id,symbol,signal_id,source_ledger_signal_id,signal_type,signal_direction,signal_score,algorithm,algorithm_version,confirmed_at,state,evaluation_mode,COALESCE(evaluation_run_id,''),registration_idempotency_key,validation_contract_id,validation_contract_version,validation_contract,evaluation_engine_version,baseline_snapshot,baseline_provenance,materialized_at,invalidated_at,superseded_at,expired_at,closed_at,transition_sequence,created_at,updated_at FROM signal_assertions`
const signalAssuranceEvaluationSelect = `SELECT evaluation_id,assertion_id,evaluated_at,evaluation_session_date,evaluation_mode,COALESCE(evaluation_run_id,''),input_snapshot,input_completeness,transition_sequence,trading_days_active,calendar_days_active,asset_price,benchmark_price,absolute_return,benchmark_return,benchmark_relative_return,mfe,mae,materialization_condition_met,invalidation_condition_met,evaluation_version,created_at FROM signal_assertion_evaluations`

type signalAssuranceScanner interface{ Scan(...any) error }

func scanSignalValidationContract(s signalAssuranceScanner) (storage.SignalValidationContractRecord, error) {
	var x storage.SignalValidationContractRecord
	err := s.Scan(&x.ContractID, &x.SignalType, &x.ContractVersion, &x.Algorithm, &x.AlgorithmVersion, &x.Direction, &x.PrimaryMetric, &x.ComparisonOperator, &x.Threshold, &x.EvaluationWindowsJSON, &x.MaxHorizonTradingDays, &x.MaterializationPolicy, &x.InvalidationPolicy, &x.ConfigJSON, &x.Active, &x.ContractScopeKey, &x.CreatedAt)
	if err != nil {
		return x, mapScanError("scan signal validation contract", err)
	}
	return x, nil
}
func scanSignalAssertion(s signalAssuranceScanner) (storage.SignalAssertionRecord, error) {
	var x storage.SignalAssertionRecord
	err := s.Scan(&x.AssertionID, &x.TenantID, &x.AssetID, &x.Symbol, &x.SignalID, &x.SourceLedgerSignalID, &x.SignalType, &x.SignalDirection, &x.SignalScore, &x.Algorithm, &x.AlgorithmVersion, &x.ConfirmedAt, &x.State, &x.EvaluationMode, &x.EvaluationRunID, &x.RegistrationIdempotencyKey, &x.ValidationContractID, &x.ValidationContractVersion, &x.ValidationContractJSON, &x.EvaluationEngineVersion, &x.BaselineSnapshotJSON, &x.BaselineProvenanceJSON, &x.MaterializedAt, &x.InvalidatedAt, &x.SupersededAt, &x.ExpiredAt, &x.ClosedAt, &x.TransitionSequence, &x.CreatedAt, &x.UpdatedAt)
	if err != nil {
		return x, mapScanError("scan signal assertion", err)
	}
	return x, nil
}
func scanSignalAssuranceEvaluation(s signalAssuranceScanner) (storage.SignalAssertionEvaluationRecord, error) {
	var x storage.SignalAssertionEvaluationRecord
	err := s.Scan(&x.EvaluationID, &x.AssertionID, &x.EvaluatedAt, &x.EvaluationSessionDate, &x.EvaluationMode, &x.EvaluationRunID, &x.InputSnapshotJSON, &x.InputCompleteness, &x.TransitionSequence, &x.TradingDaysActive, &x.CalendarDaysActive, &x.AssetPrice, &x.BenchmarkPrice, &x.AbsoluteReturn, &x.BenchmarkReturn, &x.BenchmarkRelativeReturn, &x.MFE, &x.MAE, &x.MaterializationConditionMet, &x.InvalidationConditionMet, &x.EvaluationVersion, &x.CreatedAt)
	if err != nil {
		return x, mapScanError("scan signal assurance evaluation", err)
	}
	return x, nil
}
func validateSignalValidationContract(x storage.SignalValidationContractRecord) error {
	if strings.TrimSpace(x.ContractID) == "" || strings.TrimSpace(x.SignalType) == "" || strings.TrimSpace(x.ContractVersion) == "" || strings.TrimSpace(x.Direction) == "" || strings.TrimSpace(x.PrimaryMetric) == "" || strings.TrimSpace(x.ComparisonOperator) == "" || x.MaxHorizonTradingDays <= 0 {
		return errors.New("signal validation contract has required fields missing")
	}
	if !json.Valid(jsonOrEmpty(x.EvaluationWindowsJSON)) || !json.Valid(jsonOrEmpty(x.ConfigJSON)) {
		return errors.New("signal validation contract JSON is invalid")
	}
	return nil
}
func validateSignalAssuranceRegistration(x storage.SignalAssuranceRegistration) error {
	if strings.TrimSpace(x.Event.TenantID) == "" || strings.TrimSpace(x.Event.EligibleEventID) == "" || strings.TrimSpace(x.Assertion.AssertionID) == "" || strings.TrimSpace(x.Assertion.RegistrationIdempotencyKey) == "" || x.Event.TenantID != x.Assertion.TenantID {
		return errors.New("signal assurance registration is invalid")
	}
	return validateSignalValidationContract(x.Contract)
}
func validateSignalAssuranceEvaluation(x storage.SignalAssertionEvaluationRecord) error {
	if strings.TrimSpace(x.EvaluationID) == "" || strings.TrimSpace(x.AssertionID) == "" || x.EvaluationSessionDate.IsZero() || strings.TrimSpace(x.EvaluationVersion) == "" || x.TradingDaysActive < 0 || x.CalendarDaysActive < 0 {
		return errors.New("signal assurance evaluation is invalid")
	}
	return nil
}
func terminalTimestampColumn(state string) string {
	switch state {
	case storage.SignalAssertionMaterialized:
		return "materialized_at"
	case storage.SignalAssertionInvalidated:
		return "invalidated_at"
	case storage.SignalAssertionSuperseded:
		return "superseded_at"
	case storage.SignalAssertionExpired:
		return "expired_at"
	}
	return ""
}
func timeOrNow(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value.UTC()
}

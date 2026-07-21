package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

const marketOpsOptionsCaptureSelect = `
SELECT capture_id, tenant_id, symbol, session_date, provider, source_id, run_id, status,
 analytics_ready, contract_count, usable_iv_count, usable_greeks_count, open_interest_count,
 required_surface_cells, quality_reasons, metrics, error_message, attempt_count,
 started_at, completed_at, created_at, updated_at
FROM marketops_options_capture_sessions`

func (r *Repository) UpsertMarketOpsOptionsCapture(ctx context.Context, record storage.MarketOpsOptionsCaptureRecord) error {
	if err := validateMarketOpsOptionsCapture(record); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO marketops_options_capture_sessions (
 capture_id, tenant_id, symbol, session_date, provider, source_id, run_id, status,
 analytics_ready, contract_count, usable_iv_count, usable_greeks_count, open_interest_count,
 required_surface_cells, quality_reasons, metrics, error_message, attempt_count, started_at, completed_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,now())
ON CONFLICT (tenant_id, symbol, session_date, provider) DO UPDATE SET
 capture_id=EXCLUDED.capture_id, source_id=EXCLUDED.source_id, run_id=EXCLUDED.run_id,
 status=EXCLUDED.status, analytics_ready=EXCLUDED.analytics_ready, contract_count=EXCLUDED.contract_count,
 usable_iv_count=EXCLUDED.usable_iv_count, usable_greeks_count=EXCLUDED.usable_greeks_count,
 open_interest_count=EXCLUDED.open_interest_count, required_surface_cells=EXCLUDED.required_surface_cells,
 quality_reasons=EXCLUDED.quality_reasons, metrics=EXCLUDED.metrics, error_message=EXCLUDED.error_message,
 attempt_count=marketops_options_capture_sessions.attempt_count+1, started_at=EXCLUDED.started_at,
 completed_at=EXCLUDED.completed_at, updated_at=now()`,
		record.CaptureID, strings.TrimSpace(record.TenantID), strings.ToUpper(strings.TrimSpace(record.Symbol)), dayOnly(record.SessionDate),
		firstNonEmptyString(record.Provider, "massive"), strings.TrimSpace(record.SourceID), strings.TrimSpace(record.RunID), record.Status,
		record.AnalyticsReady, record.ContractCount, record.UsableIVCount, record.UsableGreeksCount, record.OpenInterestCount,
		record.RequiredSurfaceCells, jsonOrEmptyArray(record.QualityReasonsJSON), jsonOrEmpty(record.MetricsJSON), record.ErrorMessage,
		maxInt(record.AttemptCount, 1), record.StartedAt.UTC(), record.CompletedAt.UTC())
	if err != nil {
		return fmt.Errorf("upsert marketops options capture: %w", err)
	}
	return nil
}

func (r *Repository) ListMarketOpsOptionsCaptures(ctx context.Context, filter storage.MarketOpsOptionsCaptureFilter) ([]storage.MarketOpsOptionsCaptureRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsOptionsCaptureSelect+`
WHERE tenant_id=$1
 AND ($2='' OR upper(symbol)=upper($2))
 AND ($3::date IS NULL OR session_date >= $3)
 AND ($4::date IS NULL OR session_date < $4)
 AND ($5='' OR status=$5)
 AND ($6::boolean IS NULL OR analytics_ready=$6)
ORDER BY session_date DESC, symbol ASC LIMIT $7`, strings.TrimSpace(filter.TenantID), strings.ToUpper(strings.TrimSpace(filter.Symbol)), nullTime(filter.SessionStart), nullTime(filter.SessionEnd), strings.TrimSpace(filter.Status), filter.Ready, clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops options captures: %w", err)
	}
	defer rows.Close()
	records := []storage.MarketOpsOptionsCaptureRecord{}
	for rows.Next() {
		record, err := scanMarketOpsOptionsCapture(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list marketops options captures rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetMarketOpsOptionsCapture(ctx context.Context, tenantID string, captureID string) (storage.MarketOpsOptionsCaptureRecord, error) {
	record, err := scanMarketOpsOptionsCapture(r.db.QueryRowContext(ctx, marketOpsOptionsCaptureSelect+` WHERE tenant_id=$1 AND capture_id=$2`, strings.TrimSpace(tenantID), strings.TrimSpace(captureID)))
	if err != nil {
		return storage.MarketOpsOptionsCaptureRecord{}, err
	}
	return record, nil
}

func scanMarketOpsOptionsCapture(scanner interface{ Scan(...any) error }) (storage.MarketOpsOptionsCaptureRecord, error) {
	var record storage.MarketOpsOptionsCaptureRecord
	if err := scanner.Scan(&record.CaptureID, &record.TenantID, &record.Symbol, &record.SessionDate, &record.Provider, &record.SourceID,
		&record.RunID, &record.Status, &record.AnalyticsReady, &record.ContractCount, &record.UsableIVCount, &record.UsableGreeksCount,
		&record.OpenInterestCount, &record.RequiredSurfaceCells, &record.QualityReasonsJSON, &record.MetricsJSON, &record.ErrorMessage,
		&record.AttemptCount, &record.StartedAt, &record.CompletedAt, &record.CreatedAt, &record.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return storage.MarketOpsOptionsCaptureRecord{}, storage.ErrNotFound
		}
		return storage.MarketOpsOptionsCaptureRecord{}, fmt.Errorf("scan marketops options capture: %w", err)
	}
	return record, nil
}

func validateMarketOpsOptionsCapture(record storage.MarketOpsOptionsCaptureRecord) error {
	if strings.TrimSpace(record.CaptureID) == "" || strings.TrimSpace(record.TenantID) == "" || strings.TrimSpace(record.Symbol) == "" || record.SessionDate.IsZero() || strings.TrimSpace(record.RunID) == "" {
		return fmt.Errorf("marketops options capture id, tenant_id, symbol, session_date, and run_id are required")
	}
	validStatus := record.Status == storage.MarketOpsOptionsCaptureAnalyticsReady || record.Status == storage.MarketOpsOptionsCapturePartial || record.Status == storage.MarketOpsOptionsCaptureNoData || record.Status == storage.MarketOpsOptionsCaptureFailed
	if !validStatus {
		return fmt.Errorf("marketops options capture status is invalid")
	}
	if record.AnalyticsReady != (record.Status == storage.MarketOpsOptionsCaptureAnalyticsReady) {
		return fmt.Errorf("marketops options capture readiness and status must agree")
	}
	if record.ContractCount < 0 || record.UsableIVCount < 0 || record.UsableGreeksCount < 0 || record.OpenInterestCount < 0 || record.RequiredSurfaceCells < 0 || record.RequiredSurfaceCells > 7 {
		return fmt.Errorf("marketops options capture counts are invalid")
	}
	if record.StartedAt.IsZero() || record.CompletedAt.IsZero() || record.CompletedAt.Before(record.StartedAt) {
		return fmt.Errorf("marketops options capture timestamps are invalid")
	}
	return nil
}

func jsonOrEmptyArray(value []byte) []byte {
	if len(value) == 0 || string(value) == "null" {
		return []byte(`[]`)
	}
	return value
}

func maxInt(value, minimum int) int {
	if value < minimum {
		return minimum
	}
	return value
}

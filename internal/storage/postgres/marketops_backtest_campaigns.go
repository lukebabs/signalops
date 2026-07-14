package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) UpsertMarketOpsBacktestCampaign(ctx context.Context, record storage.MarketOpsBacktestCampaignRecord) error {
	if strings.TrimSpace(record.CampaignID) == "" || strings.TrimSpace(record.TenantID) == "" || strings.TrimSpace(record.DetectorID) == "" {
		return fmt.Errorf("marketops backtest campaign_id, tenant_id, and detector_id are required")
	}
	startedAt := record.StartedAt.UTC()
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	status := strings.TrimSpace(record.Status)
	if status == "" {
		status = storage.RunStatusStarted
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO marketops_backtest_campaigns (
 campaign_id, tenant_id, app_id, domain, use_case, source_id, source_adapter, detector_id, detector_version,
 requested_by, universe_group, dataset_scope, symbols, window_start, window_end, window_step_days, max_symbols,
 max_windows, max_runs, max_records, batch_size, status, child_run_ids, metrics, error_message, started_at,
 completed_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,now())
ON CONFLICT (campaign_id) DO UPDATE SET
 tenant_id=EXCLUDED.tenant_id, app_id=EXCLUDED.app_id, domain=EXCLUDED.domain, use_case=EXCLUDED.use_case,
 source_id=EXCLUDED.source_id, source_adapter=EXCLUDED.source_adapter, detector_id=EXCLUDED.detector_id,
 detector_version=EXCLUDED.detector_version, requested_by=EXCLUDED.requested_by, universe_group=EXCLUDED.universe_group,
 dataset_scope=EXCLUDED.dataset_scope, symbols=EXCLUDED.symbols, window_start=EXCLUDED.window_start,
 window_end=EXCLUDED.window_end, window_step_days=EXCLUDED.window_step_days, max_symbols=EXCLUDED.max_symbols,
 max_windows=EXCLUDED.max_windows, max_runs=EXCLUDED.max_runs, max_records=EXCLUDED.max_records,
 batch_size=EXCLUDED.batch_size, status=EXCLUDED.status, child_run_ids=EXCLUDED.child_run_ids,
 metrics=EXCLUDED.metrics, error_message=EXCLUDED.error_message, started_at=EXCLUDED.started_at,
 completed_at=EXCLUDED.completed_at, updated_at=now()`, record.CampaignID, strings.TrimSpace(record.TenantID), recordAppID(record.AppID), recordDomain(record.Domain), recordUseCase(record.UseCase), strings.TrimSpace(record.SourceID), firstNonEmptyString(record.SourceAdapter, "market_data.massive"), strings.TrimSpace(record.DetectorID), strings.TrimSpace(record.DetectorVersion), firstNonEmptyString(record.RequestedBy, "operator-local"), strings.TrimSpace(record.UniverseGroup), pqArray(record.DatasetScope), pqArray(record.Symbols), record.WindowStart.UTC(), record.WindowEnd.UTC(), record.WindowStepDays, record.MaxSymbols, record.MaxWindows, record.MaxRuns, record.MaxRecords, record.BatchSize, status, pqArray(record.ChildRunIDs), jsonOrEmpty(record.MetricsJSON), strings.TrimSpace(record.ErrorMessage), startedAt, record.CompletedAt)
	if err != nil {
		return fmt.Errorf("upsert marketops backtest campaign: %w", err)
	}
	return nil
}

func (r *Repository) ListMarketOpsBacktestCampaigns(ctx context.Context, filter storage.MarketOpsBacktestCampaignFilter) ([]storage.MarketOpsBacktestCampaignRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsBacktestCampaignSelect+`
WHERE ($1='' OR tenant_id=$1) AND ($2='' OR app_id=$2) AND ($3='' OR domain=$3) AND ($4='' OR use_case=$4)
 AND ($5='' OR source_id=$5) AND ($6='' OR detector_id=$6) AND ($7='' OR universe_group=$7) AND ($8='' OR status=$8)
ORDER BY started_at DESC LIMIT $9`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AppID), strings.TrimSpace(filter.Domain), strings.TrimSpace(filter.UseCase), strings.TrimSpace(filter.SourceID), strings.TrimSpace(filter.DetectorID), strings.TrimSpace(filter.UniverseGroup), strings.TrimSpace(filter.Status), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops backtest campaigns: %w", err)
	}
	defer rows.Close()
	records := []storage.MarketOpsBacktestCampaignRecord{}
	for rows.Next() {
		rec, err := scanMarketOpsBacktestCampaign(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list marketops backtest campaigns rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetMarketOpsBacktestCampaign(ctx context.Context, campaignID string) (storage.MarketOpsBacktestCampaignRecord, error) {
	record, err := scanMarketOpsBacktestCampaign(r.db.QueryRowContext(ctx, marketOpsBacktestCampaignSelect+` WHERE campaign_id=$1`, strings.TrimSpace(campaignID)))
	if err != nil {
		return storage.MarketOpsBacktestCampaignRecord{}, err
	}
	return record, nil
}

const marketOpsBacktestCampaignSelect = `SELECT campaign_id, tenant_id, app_id, domain, use_case, source_id, source_adapter, detector_id,
 detector_version, requested_by, universe_group, COALESCE(array_to_json(dataset_scope), '[]'::json)::text,
 COALESCE(array_to_json(symbols), '[]'::json)::text, window_start, window_end, window_step_days, max_symbols,
 max_windows, max_runs, max_records, batch_size, status, COALESCE(array_to_json(child_run_ids), '[]'::json)::text,
 metrics, error_message, started_at, completed_at, created_at, updated_at FROM marketops_backtest_campaigns`

type marketOpsBacktestCampaignScanner interface{ Scan(dest ...any) error }

func scanMarketOpsBacktestCampaign(scanner marketOpsBacktestCampaignScanner) (storage.MarketOpsBacktestCampaignRecord, error) {
	var record storage.MarketOpsBacktestCampaignRecord
	var datasetScopeJSON, symbolsJSON, childRunIDsJSON string
	var completedAt sql.NullTime
	var errorMessage sql.NullString
	if err := scanner.Scan(&record.CampaignID, &record.TenantID, &record.AppID, &record.Domain, &record.UseCase, &record.SourceID, &record.SourceAdapter, &record.DetectorID, &record.DetectorVersion, &record.RequestedBy, &record.UniverseGroup, &datasetScopeJSON, &symbolsJSON, &record.WindowStart, &record.WindowEnd, &record.WindowStepDays, &record.MaxSymbols, &record.MaxWindows, &record.MaxRuns, &record.MaxRecords, &record.BatchSize, &record.Status, &childRunIDsJSON, &record.MetricsJSON, &errorMessage, &record.StartedAt, &completedAt, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return storage.MarketOpsBacktestCampaignRecord{}, mapScanError("scan marketops backtest campaign", err)
	}
	if err := json.Unmarshal([]byte(datasetScopeJSON), &record.DatasetScope); err != nil {
		return storage.MarketOpsBacktestCampaignRecord{}, err
	}
	if err := json.Unmarshal([]byte(symbolsJSON), &record.Symbols); err != nil {
		return storage.MarketOpsBacktestCampaignRecord{}, err
	}
	if err := json.Unmarshal([]byte(childRunIDsJSON), &record.ChildRunIDs); err != nil {
		return storage.MarketOpsBacktestCampaignRecord{}, err
	}
	if completedAt.Valid {
		record.CompletedAt = &completedAt.Time
	}
	record.ErrorMessage = errorMessage.String
	return record, nil
}

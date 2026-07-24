package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) ListMarketOpsSignalOverviewInputs(ctx context.Context, filter storage.MarketOpsSignalOverviewFilter) (storage.MarketOpsSignalOverviewInputs, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT tenant_id, app_id, domain, use_case, source_id, universe_group, rank, ticker, ticker_key,
  company, company_key, display_name, display_sector, asset_type, exchange, sector, sector_key, industry, industry_key,
  is_active, metadata, created_at, updated_at
FROM marketops_asset_universe
WHERE tenant_id=$1 AND is_active=true AND ($2='' OR universe_group=$2)
ORDER BY universe_group ASC, rank ASC
LIMIT 200`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.UniverseGroup))
	if err != nil {
		return storage.MarketOpsSignalOverviewInputs{}, fmt.Errorf("list signal overview assets: %w", err)
	}
	defer rows.Close()
	assets := []storage.MarketOpsAssetRecord{}
	for rows.Next() {
		asset, scanErr := scanMarketOpsAsset(rows)
		if scanErr != nil {
			return storage.MarketOpsSignalOverviewInputs{}, scanErr
		}
		assets = append(assets, asset)
	}
	if err := rows.Err(); err != nil {
		return storage.MarketOpsSignalOverviewInputs{}, fmt.Errorf("list signal overview asset rows: %w", err)
	}
	symbols := make([]string, 0, len(assets))
	for _, asset := range assets {
		symbols = append(symbols, strings.ToUpper(strings.TrimSpace(asset.Ticker)))
	}
	inputs := storage.MarketOpsSignalOverviewInputs{Assets: assets}
	if len(symbols) == 0 {
		return inputs, nil
	}

	resultRows, err := r.db.QueryContext(ctx, algorithmResultSelect+`
WHERE tenant_id=$1 AND algorithm_id='signalops.algorithms.risk_reward_temporal_v1'
  AND upper(COALESCE(result_payload->>'symbol','')) = ANY($2)
  AND COALESCE(result_payload->>'observation_time','') >= $3
ORDER BY created_at DESC`, strings.TrimSpace(filter.TenantID), pqArray(symbols), filter.SessionStart.UTC().Format("2006-01-02T15:04:05Z"))
	if err != nil {
		return storage.MarketOpsSignalOverviewInputs{}, fmt.Errorf("list signal overview risk reward results: %w", err)
	}
	defer resultRows.Close()
	for resultRows.Next() {
		record, scanErr := scanAlgorithmResult(resultRows)
		if scanErr != nil {
			return storage.MarketOpsSignalOverviewInputs{}, scanErr
		}
		inputs.AlgorithmResults = append(inputs.AlgorithmResults, record)
	}
	if err := resultRows.Err(); err != nil {
		return storage.MarketOpsSignalOverviewInputs{}, fmt.Errorf("list signal overview risk reward rows: %w", err)
	}

	evaluationRows, err := r.db.QueryContext(ctx, marketOpsHypothesisEvaluationSelect+`
WHERE tenant_id=$1 AND triggered=true AND invalidated=false AND upper(symbol) = ANY($2)
  AND session_date >= $3::date
ORDER BY session_date DESC, hypothesis_key`, strings.TrimSpace(filter.TenantID), pqArray(symbols), filter.SessionStart.UTC())
	if err != nil {
		return storage.MarketOpsSignalOverviewInputs{}, fmt.Errorf("list signal overview hypothesis evaluations: %w", err)
	}
	defer evaluationRows.Close()
	for evaluationRows.Next() {
		record, scanErr := scanMarketOpsHypothesisEvaluation(evaluationRows)
		if scanErr != nil {
			return storage.MarketOpsSignalOverviewInputs{}, scanErr
		}
		inputs.HypothesisEvaluations = append(inputs.HypothesisEvaluations, record)
	}
	if err := evaluationRows.Err(); err != nil {
		return storage.MarketOpsSignalOverviewInputs{}, fmt.Errorf("list signal overview hypothesis rows: %w", err)
	}

	inputs.HypothesisDefinitions, err = r.ListMarketOpsHypothesisDefinitions(ctx, storage.MarketOpsHypothesisDefinitionFilter{TenantID: filter.TenantID, Limit: 200})
	if err != nil {
		return storage.MarketOpsSignalOverviewInputs{}, err
	}
	inputs.IntradayConditionSnaps, err = r.ListMarketOpsIntradayConditionSnapshots(ctx, storage.MarketOpsIntradayConditionSnapshotFilter{TenantID: filter.TenantID, UniverseGroup: filter.UniverseGroup, Limit: 200})
	if err != nil {
		return storage.MarketOpsSignalOverviewInputs{}, err
	}
	return inputs, nil
}

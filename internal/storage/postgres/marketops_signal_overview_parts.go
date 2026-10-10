package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) listSignalOverviewOptions(ctx context.Context, tenant string, symbols []string, sessionStart time.Time) ([]storage.MarketOpsOptionsDistributionRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsOptionsDistributionSelect+`
WHERE tenant_id=$1 AND upper(symbol) = ANY($2) AND window_name='10_trade_days'
  AND trade_date >= $3::date
ORDER BY trade_date DESC, symbol ASC`, tenant, pqArray(symbols), sessionStart.UTC())
	if err != nil {
		return nil, fmt.Errorf("list signal overview options distributions: %w", err)
	}
	defer rows.Close()
	items := []storage.MarketOpsOptionsDistributionRecord{}
	for rows.Next() {
		item, scanErr := scanMarketOpsOptionsDistribution(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list signal overview options distribution rows: %w", err)
	}
	return items, nil
}

func (r *Repository) listSignalOverviewAlgorithmResults(ctx context.Context, tenant string, symbols []string, sessionStart time.Time) ([]storage.AlgorithmResultRecord, error) {
	rows, err := r.db.QueryContext(ctx, algorithmResultSelect+`
WHERE tenant_id=$1 AND algorithm_id='signalops.algorithms.risk_reward_temporal_v1'
  AND upper(COALESCE(result_payload->>'symbol','')) = ANY($2)
  AND COALESCE(result_payload->>'observation_time','') >= $3
ORDER BY created_at DESC`, tenant, pqArray(symbols), sessionStart.UTC().Format("2006-01-02T15:04:05Z"))
	if err != nil {
		return nil, fmt.Errorf("list signal overview risk reward results: %w", err)
	}
	defer rows.Close()
	items := []storage.AlgorithmResultRecord{}
	for rows.Next() {
		item, scanErr := scanAlgorithmResult(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list signal overview risk reward rows: %w", err)
	}
	return items, nil
}

func (r *Repository) listSignalOverviewEvaluations(ctx context.Context, tenant string, symbols []string, sessionStart time.Time) ([]storage.MarketOpsHypothesisEvaluationRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsHypothesisEvaluationSelect+`
WHERE tenant_id=$1 AND triggered=true AND invalidated=false AND upper(symbol) = ANY($2)
  AND session_date >= $3::date
ORDER BY session_date DESC, hypothesis_key`, tenant, pqArray(symbols), sessionStart.UTC())
	if err != nil {
		return nil, fmt.Errorf("list signal overview hypothesis evaluations: %w", err)
	}
	defer rows.Close()
	items := []storage.MarketOpsHypothesisEvaluationRecord{}
	for rows.Next() {
		item, scanErr := scanMarketOpsHypothesisEvaluation(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list signal overview hypothesis rows: %w", err)
	}
	return items, nil
}

func (r *Repository) listSignalOverviewDefinitions(ctx context.Context, tenant string) ([]storage.MarketOpsHypothesisDefinitionRecord, error) {
	return r.ListMarketOpsHypothesisDefinitions(ctx, storage.MarketOpsHypothesisDefinitionFilter{TenantID: tenant, Limit: 200})
}

func (r *Repository) listSignalOverviewIntraday(ctx context.Context, filter storage.MarketOpsSignalOverviewFilter) ([]storage.MarketOpsIntradayConditionSnapshotRecord, error) {
	return r.ListMarketOpsIntradayConditionSnapshots(ctx, storage.MarketOpsIntradayConditionSnapshotFilter{TenantID: filter.TenantID, UniverseGroup: filter.UniverseGroup, Limit: 200})
}

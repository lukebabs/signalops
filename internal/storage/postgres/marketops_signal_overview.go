package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
	"golang.org/x/sync/errgroup"
)

func (r *Repository) ListMarketOpsSignalOverviewInputs(ctx context.Context, filter storage.MarketOpsSignalOverviewFilter) (storage.MarketOpsSignalOverviewInputs, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT tenant_id, app_id, domain, use_case, source_id, universe_group, rank, ticker, ticker_key,
  company, company_key, display_name, display_sector, asset_type, exchange, sector, sector_key, industry, industry_key,
  is_active, metadata, created_at, updated_at
FROM marketops_universal_assets
WHERE tenant_id=$1 AND is_active=true AND ($2='' OR $2='all_active' OR universe_group=$2)
ORDER BY rank ASC
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

	// These reads are independent after the authorized asset symbols are known.
	// Run them concurrently so the Dashboard latency is bounded by the slowest
	// projection instead of the sum of every historical query.
	group, groupCtx := errgroup.WithContext(ctx)
	var options []storage.MarketOpsOptionsDistributionRecord
	var results []storage.AlgorithmResultRecord
	var evaluations []storage.MarketOpsHypothesisEvaluationRecord
	var definitions []storage.MarketOpsHypothesisDefinitionRecord
	var intraday []storage.MarketOpsIntradayConditionSnapshotRecord
	group.Go(func() error {
		var queryErr error
		options, queryErr = r.listSignalOverviewOptions(groupCtx, strings.TrimSpace(filter.TenantID), symbols, filter.SessionStart)
		return queryErr
	})
	group.Go(func() error {
		var queryErr error
		results, queryErr = r.listSignalOverviewAlgorithmResults(groupCtx, strings.TrimSpace(filter.TenantID), symbols, filter.SessionStart)
		return queryErr
	})
	group.Go(func() error {
		var queryErr error
		evaluations, queryErr = r.listSignalOverviewEvaluations(groupCtx, strings.TrimSpace(filter.TenantID), symbols, filter.SessionStart)
		return queryErr
	})
	group.Go(func() error {
		var queryErr error
		definitions, queryErr = r.listSignalOverviewDefinitions(groupCtx, strings.TrimSpace(filter.TenantID))
		return queryErr
	})
	group.Go(func() error {
		var queryErr error
		intraday, queryErr = r.listSignalOverviewIntraday(groupCtx, filter)
		return queryErr
	})
	if err := group.Wait(); err != nil {
		return storage.MarketOpsSignalOverviewInputs{}, err
	}
	inputs.OptionsDistributions = options
	inputs.AlgorithmResults = results
	inputs.HypothesisEvaluations = evaluations
	inputs.HypothesisDefinitions = definitions
	inputs.IntradayConditionSnaps = intraday
	return inputs, nil
}

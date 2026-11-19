package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

// ListSubscriberGlobalOptionsDistributions reads the platform-owned Options
// distribution projection for already-authorized watchlist symbols.
func (r *Repository) ListSubscriberGlobalOptionsDistributions(ctx context.Context, symbols []string, sessionStart time.Time, limit int) ([]storage.MarketOpsOptionsDistributionRecord, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT 'platform-global', symbol, trade_date, window_name, source_id, provider, trade_days, contract_count, call_contract_count, put_contract_count, total_call_open_interest, total_put_open_interest, total_call_volume, total_put_volume, missing_open_interest_count, call_put_open_interest_ratio, call_put_volume_ratio, ratio_delta, ratio_change_pct, ratio_zscore, change_point_score, confidence, moneyness_distribution, expiration_distribution, metrics, provenance, source_trade_dates, observed_at, observed_at FROM subscriber_gateway_global_options_distributions WHERE upper(symbol) = ANY($1) AND trade_date >= $2::date ORDER BY trade_date DESC, symbol ASC LIMIT $3", pqArray(symbols), sessionStart.UTC(), riskRewardSnapshotLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list subscriber global options distributions: %w", err)
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
		return nil, fmt.Errorf("list subscriber global options distribution rows: %w", err)
	}
	return items, nil
}

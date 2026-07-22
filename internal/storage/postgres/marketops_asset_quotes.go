package postgres

import (
	"context"
	"fmt"
	"github.com/lukebabs/signalops/internal/storage"
	"strings"
)

func (r *Repository) UpsertMarketOpsAssetQuote(ctx context.Context, x storage.MarketOpsAssetQuoteRecord) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO marketops_asset_quote_cache (tenant_id,universe_group,ticker,price,quote_timestamp,market_status,stale,previous_close,change_value,change_percent,week52_low,week52_high,refreshed_at,provider) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) ON CONFLICT (tenant_id,universe_group,ticker) DO UPDATE SET price=EXCLUDED.price,quote_timestamp=EXCLUDED.quote_timestamp,market_status=EXCLUDED.market_status,stale=EXCLUDED.stale,previous_close=EXCLUDED.previous_close,change_value=EXCLUDED.change_value,change_percent=EXCLUDED.change_percent,week52_low=EXCLUDED.week52_low,week52_high=EXCLUDED.week52_high,refreshed_at=EXCLUDED.refreshed_at,provider=EXCLUDED.provider`, x.TenantID, x.UniverseGroup, strings.ToUpper(x.Ticker), x.Price, x.QuoteTimestamp.UTC(), x.MarketStatus, x.Stale, x.PreviousClose, x.Change, x.ChangePercent, x.Week52Low, x.Week52High, x.RefreshedAt.UTC(), x.Provider)
	if err != nil {
		return fmt.Errorf("upsert asset quote cache: %w", err)
	}
	return nil
}
func (r *Repository) ListMarketOpsAssetQuotes(ctx context.Context, f storage.MarketOpsAssetQuoteFilter) ([]storage.MarketOpsAssetQuoteRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT tenant_id,universe_group,ticker,price,quote_timestamp,market_status,stale,previous_close,change_value,change_percent,week52_low,week52_high,refreshed_at,provider FROM marketops_asset_quote_cache WHERE tenant_id=$1 AND ($2='' OR universe_group=$2) ORDER BY ticker LIMIT $3`, f.TenantID, f.UniverseGroup, clampLimit(f.Limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []storage.MarketOpsAssetQuoteRecord{}
	for rows.Next() {
		var x storage.MarketOpsAssetQuoteRecord
		if err := rows.Scan(&x.TenantID, &x.UniverseGroup, &x.Ticker, &x.Price, &x.QuoteTimestamp, &x.MarketStatus, &x.Stale, &x.PreviousClose, &x.Change, &x.ChangePercent, &x.Week52Low, &x.Week52High, &x.RefreshedAt, &x.Provider); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

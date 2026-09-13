package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) UpsertMarketOpsSRIETFHoldingsSnapshot(ctx context.Context, x storage.MarketOpsSRIETFHoldingsSnapshotRecord) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO sri_etf_holdings_snapshots (snapshot_id,tenant_id,etf_symbol,fund_name,effective_date,retrieved_at,source,source_url,content_hash,holdings_count,total_weight,top_ten_weight) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) ON CONFLICT (tenant_id,etf_symbol,effective_date,source,content_hash) DO NOTHING", x.SnapshotID, x.TenantID, x.ETFSymbol, x.FundName, x.EffectiveDate.UTC(), x.RetrievedAt.UTC(), x.Source, x.SourceURL, x.ContentHash, x.HoldingsCount, x.TotalWeight, x.TopTenWeight)
	if err != nil {
		return fmt.Errorf("upsert sri etf holdings snapshot: %w", err)
	}
	return nil
}

func (r *Repository) UpsertMarketOpsSRIETFHolding(ctx context.Context, x storage.MarketOpsSRIETFHoldingRecord) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO sri_etf_holdings (snapshot_id,holding_key,holding_rank,ticker,name,identifier,sedol,sector,currency,weight,shares_held) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) ON CONFLICT (snapshot_id,holding_key) DO NOTHING", x.SnapshotID, x.HoldingKey, x.HoldingRank, x.Ticker, x.Name, x.Identifier, x.SEDOL, x.Sector, x.Currency, x.Weight, x.SharesHeld)
	if err != nil {
		return fmt.Errorf("upsert sri etf holding: %w", err)
	}
	return nil
}

func (r *Repository) GetLatestMarketOpsSRIETFHoldingsSnapshot(ctx context.Context, tenantID, etfSymbol string) (storage.MarketOpsSRIETFHoldingsSnapshotRecord, bool, error) {
	var x storage.MarketOpsSRIETFHoldingsSnapshotRecord
	err := r.db.QueryRowContext(ctx, "SELECT snapshot_id,tenant_id,etf_symbol,fund_name,effective_date,retrieved_at,source,source_url,content_hash,holdings_count,total_weight,top_ten_weight FROM sri_etf_holdings_snapshots WHERE tenant_id=$1 AND etf_symbol=$2 ORDER BY effective_date DESC,retrieved_at DESC LIMIT 1", strings.TrimSpace(tenantID), strings.ToUpper(strings.TrimSpace(etfSymbol))).Scan(&x.SnapshotID, &x.TenantID, &x.ETFSymbol, &x.FundName, &x.EffectiveDate, &x.RetrievedAt, &x.Source, &x.SourceURL, &x.ContentHash, &x.HoldingsCount, &x.TotalWeight, &x.TopTenWeight)
	if err == sql.ErrNoRows {
		return storage.MarketOpsSRIETFHoldingsSnapshotRecord{}, false, nil
	}
	if err != nil {
		return storage.MarketOpsSRIETFHoldingsSnapshotRecord{}, false, fmt.Errorf("get latest sri etf holdings snapshot: %w", err)
	}
	return x, true, nil
}

func (r *Repository) ListMarketOpsSRIETFHoldings(ctx context.Context, snapshotID string, limit int) ([]storage.MarketOpsSRIETFHoldingRecord, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT snapshot_id,holding_key,holding_rank,ticker,name,identifier,sedol,sector,currency,weight,shares_held FROM sri_etf_holdings WHERE snapshot_id=$1 ORDER BY holding_rank LIMIT $2", strings.TrimSpace(snapshotID), clampLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list sri etf holdings: %w", err)
	}
	defer rows.Close()
	out := []storage.MarketOpsSRIETFHoldingRecord{}
	for rows.Next() {
		var x storage.MarketOpsSRIETFHoldingRecord
		if err := rows.Scan(&x.SnapshotID, &x.HoldingKey, &x.HoldingRank, &x.Ticker, &x.Name, &x.Identifier, &x.SEDOL, &x.Sector, &x.Currency, &x.Weight, &x.SharesHeld); err != nil {
			return nil, fmt.Errorf("scan sri etf holding: %w", err)
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

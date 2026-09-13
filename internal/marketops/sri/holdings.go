package sri

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/adapters/marketdata/statestreet"
	"github.com/lukebabs/signalops/internal/storage"
)

type HoldingsRefreshResult struct {
	ETFs, Snapshots, Holdings, Unsupported int
}

// RefreshStateStreetHoldings stores current issuer-published ETF composition snapshots.
// These snapshots are representational context only and are deliberately excluded from
// SRI calculation inputs and historical reconstruction.
func RefreshStateStreetHoldings(ctx context.Context, repo storage.MarketOpsSRIRepository, client statestreet.Client, tenantID string) (HoldingsRefreshResult, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return HoldingsRefreshResult{}, fmt.Errorf("tenant id is required")
	}
	registry, err := repo.ListMarketOpsSRIETFRegistry(ctx, tenantID, "")
	if err != nil {
		return HoldingsRefreshResult{}, fmt.Errorf("list SRI ETF registry: %w", err)
	}
	symbols := make([]string, 0, len(registry))
	seen := map[string]bool{}
	result := HoldingsRefreshResult{}
	for _, item := range registry {
		symbol := strings.ToUpper(strings.TrimSpace(item.ETFSymbol))
		if !item.Active || !strings.EqualFold(item.Role, "primary") || seen[symbol] {
			continue
		}
		seen[symbol] = true
		if !statestreet.Supports(symbol) {
			result.Unsupported++
			continue
		}
		symbols = append(symbols, symbol)
	}
	sort.Strings(symbols)
	for _, symbol := range symbols {
		current, err := client.DownloadCurrent(ctx, symbol)
		if err != nil {
			return result, fmt.Errorf("refresh %s State Street holdings: %w", symbol, err)
		}
		snapshotID := "sri_holdings_" + stable(strings.Join([]string{
			tenantID, symbol, current.EffectiveDate.Format("2006-01-02"), statestreet.Source, current.ContentHash,
		}, "|"))
		snapshot := storage.MarketOpsSRIETFHoldingsSnapshotRecord{
			SnapshotID: snapshotID, TenantID: tenantID, ETFSymbol: symbol, FundName: current.FundName,
			EffectiveDate: current.EffectiveDate, RetrievedAt: current.RetrievedAt,
			Source: statestreet.Source, SourceURL: current.SourceURL, ContentHash: current.ContentHash,
			HoldingsCount: len(current.Holdings), TotalWeight: current.TotalWeight, TopTenWeight: current.TopTenWeight,
		}
		if err := repo.UpsertMarketOpsSRIETFHoldingsSnapshot(ctx, snapshot); err != nil {
			return result, err
		}
		result.ETFs++
		result.Snapshots++
		for _, holding := range current.Holdings {
			key := stable(strings.Join([]string{snapshotID, fmt.Sprintf("%d", holding.Rank), holding.Ticker, holding.Identifier, holding.Name}, "|"))
			record := storage.MarketOpsSRIETFHoldingRecord{
				SnapshotID: snapshotID, HoldingKey: key, HoldingRank: holding.Rank, Ticker: holding.Ticker,
				Name: holding.Name, Identifier: holding.Identifier, SEDOL: holding.SEDOL, Sector: holding.Sector,
				Currency: holding.Currency, Weight: holding.Weight, SharesHeld: holding.SharesHeld,
			}
			if err := repo.UpsertMarketOpsSRIETFHolding(ctx, record); err != nil {
				return result, err
			}
			result.Holdings++
		}
	}
	return result, nil
}

func stateStreetSnapshotID(tenantID, symbol, contentHash string, effectiveDate time.Time) string {
	return "sri_holdings_" + stable(strings.Join([]string{tenantID, symbol, effectiveDate.Format("2006-01-02"), statestreet.Source, contentHash}, "|"))
}

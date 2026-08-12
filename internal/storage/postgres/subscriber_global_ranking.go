package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) ImportSubscriberGlobalRankingSnapshot(ctx context.Context, in storage.SubscriberGlobalRankingSnapshotImport) (storage.SubscriberGlobalRankingSnapshotImport, error) {
	in.RankingSnapshotID, in.SourceLabel, in.SourceSHA256, in.ImportedBy = strings.TrimSpace(in.RankingSnapshotID), strings.TrimSpace(in.SourceLabel), strings.TrimSpace(in.SourceSHA256), strings.TrimSpace(in.ImportedBy)
	if in.RankingSnapshotID == "" {
		in.RankingSnapshotID = newSubscriberID("subrank")
	}
	if in.AsOfDate.IsZero() || in.SourceLabel == "" || in.SourceSHA256 == "" || in.ImportedBy == "" || in.RequestedCapacity <= 0 || in.RequestedCapacity > 1000 || len(in.Entries) != in.RequestedCapacity {
		return in, errors.New("invalid global ranking snapshot import")
	}
	if len(in.ProvenanceJSON) == 0 {
		in.ProvenanceJSON = []byte("{}")
	}
	if !json.Valid(in.ProvenanceJSON) {
		return in, errors.New("ranking snapshot provenance must be JSON")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return in, fmt.Errorf("begin ranking snapshot import: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `UPDATE subscriber_global_ranking_snapshots SET is_current=false WHERE is_current`); err != nil {
		return in, fmt.Errorf("clear current ranking snapshot: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO subscriber_global_ranking_snapshots (ranking_snapshot_id,source_label,source_sha256,as_of_date,requested_capacity,source_rows_examined,distinct_symbols_selected,duplicate_symbols_skipped,imported_by,provenance,is_current) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10::jsonb,true)`, in.RankingSnapshotID, in.SourceLabel, in.SourceSHA256, in.AsOfDate, in.RequestedCapacity, in.SourceRowsExamined, len(in.Entries), in.DuplicateSymbolsSkipped, in.ImportedBy, string(in.ProvenanceJSON)); err != nil {
		return in, fmt.Errorf("insert ranking snapshot: %w", err)
	}
	for _, entry := range in.Entries {
		entry.ProviderSymbol = strings.ToUpper(strings.TrimSpace(entry.ProviderSymbol))
		if entry.SelectionRank <= 0 || entry.SourceRank <= 0 || entry.ProviderSymbol == "" || strings.TrimSpace(entry.SourceRowSHA256) == "" {
			return in, errors.New("invalid ranking snapshot entry")
		}
		id := subscriberGlobalAssetID("src-massive", entry.ProviderSymbol)
		provenance, _ := json.Marshal(map[string]any{"source": "companies.csv", "source_rank": entry.SourceRank, "ranking_snapshot_id": in.RankingSnapshotID, "source_row_sha256": entry.SourceRowSHA256})
		if _, err = tx.ExecContext(ctx, `INSERT INTO subscriber_global_assets (global_asset_id,source_id,provider_symbol,canonical_symbol,company_name,eligibility_status,reference_provenance,first_seen_at,last_seen_at) VALUES ($1,'src-massive',$2,$2,$3,'discovered',$4::jsonb,now(),now()) ON CONFLICT (source_id,provider_symbol) DO UPDATE SET company_name=EXCLUDED.company_name,last_seen_at=now(),updated_at=now()`, id, entry.ProviderSymbol, strings.TrimSpace(entry.CompanyName), string(provenance)); err != nil {
			return in, fmt.Errorf("upsert ranked global asset %s: %w", entry.ProviderSymbol, err)
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO subscriber_global_ranking_snapshot_entries (ranking_snapshot_id,selection_rank,source_rank,global_asset_id,provider_symbol,company_name,market_cap_raw,revenue_raw,source_row_sha256) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, in.RankingSnapshotID, entry.SelectionRank, entry.SourceRank, id, entry.ProviderSymbol, entry.CompanyName, entry.MarketCapRaw, entry.RevenueRaw, entry.SourceRowSHA256); err != nil {
			return in, fmt.Errorf("insert ranked entry %s: %w", entry.ProviderSymbol, err)
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO subscriber_global_asset_identity_resolutions (source_global_asset_id,canonical_global_asset_id,resolution_version,resolution_reason) SELECT global_asset_id,first_value(global_asset_id) OVER (PARTITION BY canonical_symbol ORDER BY CASE WHEN source_id='src-massive' THEN 0 ELSE 1 END,source_id,global_asset_id),'s2-canonical-security-v1','canonical_symbol_provider_source_resolution' FROM subscriber_global_assets ON CONFLICT (source_global_asset_id) DO UPDATE SET canonical_global_asset_id=EXCLUDED.canonical_global_asset_id,resolution_version=EXCLUDED.resolution_version,resolution_reason=EXCLUDED.resolution_reason,resolved_at=now()`); err != nil {
		return in, fmt.Errorf("refresh global identity resolution: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return in, fmt.Errorf("commit ranking snapshot import: %w", err)
	}
	return in, nil
}

func rankingRowSHA256(values ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(values, "\x00")))
	return hex.EncodeToString(sum[:])
}

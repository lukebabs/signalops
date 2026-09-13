package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

// These readers deliberately query security-barrier views rather than raw SRI
// tables. The views contain only platform-global SRI evidence.
func (r *Repository) ListSubscriberGlobalSRISegments(ctx context.Context, activeOnly bool, limit int) ([]storage.MarketOpsSRISegmentRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT tenant_id,segment_id,segment_key,name,segment_type,parent_segment_key,active,registry_version,metadata FROM subscriber_gateway_global_sri_segments WHERE ($1=false OR active=true) ORDER BY segment_type,name LIMIT $2`, activeOnly, clampLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list subscriber global sri segments: %w", err)
	}
	defer rows.Close()
	out := []storage.MarketOpsSRISegmentRecord{}
	for rows.Next() {
		var x storage.MarketOpsSRISegmentRecord
		if err := rows.Scan(&x.TenantID, &x.SegmentID, &x.SegmentKey, &x.Name, &x.SegmentType, &x.ParentSegmentKey, &x.Active, &x.RegistryVersion, &x.MetadataJSON); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *Repository) ListSubscriberGlobalSRIETFRegistry(ctx context.Context, segmentID string) ([]storage.MarketOpsSRIETFRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT tenant_id,etf_symbol,segment_id,role,benchmark_priority,active,registry_version,config FROM subscriber_gateway_global_sri_etf_registry WHERE ($1='' OR segment_id=$1) AND active=true ORDER BY segment_id,benchmark_priority,etf_symbol`, strings.TrimSpace(segmentID))
	if err != nil {
		return nil, fmt.Errorf("list subscriber global sri registry: %w", err)
	}
	defer rows.Close()
	out := []storage.MarketOpsSRIETFRecord{}
	for rows.Next() {
		var x storage.MarketOpsSRIETFRecord
		if err := rows.Scan(&x.TenantID, &x.ETFSymbol, &x.SegmentID, &x.Role, &x.BenchmarkPriority, &x.Active, &x.RegistryVersion, &x.ConfigJSON); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *Repository) ListSubscriberGlobalSRISnapshots(ctx context.Context, f storage.MarketOpsSRISnapshotFilter) ([]storage.MarketOpsSRISnapshotRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT q.snapshot_id,q.tenant_id,q.segment_id,q.session_date,q.as_of_time,q.state,q.composite_score,q.relative_strength_score,q.momentum_score,q.momentum_acceleration,q.rank,q.rank_change_5d,q.evidence_quality,q.quality_state,q.quality_flags,q.components,q.input_provenance,q.algorithm_version,q.configuration_version,q.calculation_run_id,q.deterministic_key FROM subscriber_gateway_global_sri_snapshots q JOIN subscriber_gateway_global_sri_segments s ON s.segment_id=q.segment_id WHERE ($1='' OR q.segment_id=$1) AND ($2='' OR s.segment_type=$2) AND ($3='' OR q.state=$3) AND ($4='' OR q.quality_state=$4) AND ($5::timestamptz IS NULL OR q.session_date >= $5::date) AND ($6::timestamptz IS NULL OR q.session_date <= $6::date) ORDER BY q.session_date DESC,q.rank NULLS LAST LIMIT $7`, f.SegmentID, f.SegmentType, f.State, f.QualityState, nullTime(f.SessionStart), nullTime(f.SessionEnd), clampLimit(f.Limit))
	if err != nil {
		return nil, fmt.Errorf("list subscriber global sri snapshots: %w", err)
	}
	defer rows.Close()
	out := []storage.MarketOpsSRISnapshotRecord{}
	for rows.Next() {
		var x storage.MarketOpsSRISnapshotRecord
		if err := rows.Scan(&x.SnapshotID, &x.TenantID, &x.SegmentID, &x.SessionDate, &x.AsOfTime, &x.State, &x.CompositeScore, &x.RelativeStrengthScore, &x.MomentumScore, &x.MomentumAcceleration, &x.Rank, &x.RankChange5D, &x.EvidenceQuality, &x.QualityState, &x.QualityFlagsJSON, &x.ComponentsJSON, &x.InputProvenanceJSON, &x.AlgorithmVersion, &x.ConfigurationVersion, &x.CalculationRunID, &x.DeterministicKey); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *Repository) GetLatestSubscriberGlobalSRIETFHoldingsSnapshot(ctx context.Context, etf string) (storage.MarketOpsSRIETFHoldingsSnapshotRecord, bool, error) {
	var x storage.MarketOpsSRIETFHoldingsSnapshotRecord
	err := r.db.QueryRowContext(ctx, `SELECT snapshot_id,tenant_id,etf_symbol,fund_name,effective_date,retrieved_at,source,source_url,content_hash,holdings_count,total_weight,top_ten_weight FROM subscriber_gateway_global_sri_etf_holdings_snapshots WHERE etf_symbol=$1 ORDER BY effective_date DESC,retrieved_at DESC LIMIT 1`, strings.ToUpper(strings.TrimSpace(etf))).Scan(&x.SnapshotID, &x.TenantID, &x.ETFSymbol, &x.FundName, &x.EffectiveDate, &x.RetrievedAt, &x.Source, &x.SourceURL, &x.ContentHash, &x.HoldingsCount, &x.TotalWeight, &x.TopTenWeight)
	if err == sql.ErrNoRows {
		return storage.MarketOpsSRIETFHoldingsSnapshotRecord{}, false, nil
	}
	if err != nil {
		return storage.MarketOpsSRIETFHoldingsSnapshotRecord{}, false, fmt.Errorf("get subscriber global sri holdings snapshot: %w", err)
	}
	return x, true, nil
}
func (r *Repository) ListSubscriberGlobalSRIETFHoldings(ctx context.Context, snapshotID string, limit int) ([]storage.MarketOpsSRIETFHoldingRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT snapshot_id,holding_key,holding_rank,ticker,name,identifier,sedol,sector,currency,weight,shares_held FROM subscriber_gateway_global_sri_etf_holdings WHERE snapshot_id=$1 ORDER BY holding_rank LIMIT $2`, strings.TrimSpace(snapshotID), clampLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list subscriber global sri holdings: %w", err)
	}
	defer rows.Close()
	out := []storage.MarketOpsSRIETFHoldingRecord{}
	for rows.Next() {
		var x storage.MarketOpsSRIETFHoldingRecord
		if err := rows.Scan(&x.SnapshotID, &x.HoldingKey, &x.HoldingRank, &x.Ticker, &x.Name, &x.Identifier, &x.SEDOL, &x.Sector, &x.Currency, &x.Weight, &x.SharesHeld); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

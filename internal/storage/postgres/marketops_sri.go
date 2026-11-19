package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) UpsertMarketOpsSRISegment(ctx context.Context, x storage.MarketOpsSRISegmentRecord) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO sri_segments (tenant_id,segment_id,segment_key,name,segment_type,parent_segment_key,active,registry_version,metadata) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT (tenant_id,segment_id) DO UPDATE SET segment_key=EXCLUDED.segment_key,name=EXCLUDED.name,segment_type=EXCLUDED.segment_type,parent_segment_key=EXCLUDED.parent_segment_key,active=EXCLUDED.active,registry_version=EXCLUDED.registry_version,metadata=EXCLUDED.metadata,updated_at=now()", x.TenantID, x.SegmentID, x.SegmentKey, x.Name, x.SegmentType, x.ParentSegmentKey, x.Active, x.RegistryVersion, jsonOrEmpty(x.MetadataJSON))
	if err != nil {
		return fmt.Errorf("upsert sri segment: %w", err)
	}
	return nil
}
func (r *Repository) UpsertMarketOpsSRIETF(ctx context.Context, x storage.MarketOpsSRIETFRecord) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO sri_etf_registry (tenant_id,etf_symbol,segment_id,role,benchmark_priority,active,registry_version,config) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (tenant_id,etf_symbol,segment_id,registry_version) DO UPDATE SET role=EXCLUDED.role,benchmark_priority=EXCLUDED.benchmark_priority,active=EXCLUDED.active,config=EXCLUDED.config,updated_at=now()", x.TenantID, strings.ToUpper(x.ETFSymbol), x.SegmentID, x.Role, x.BenchmarkPriority, x.Active, x.RegistryVersion, jsonOrEmpty(x.ConfigJSON))
	if err != nil {
		return fmt.Errorf("upsert sri etf: %w", err)
	}
	return nil
}
func (r *Repository) UpsertMarketOpsSRISnapshot(ctx context.Context, x storage.MarketOpsSRISnapshotRecord) error {
	// Partial snapshots are incomplete computations. Only a later usable result may supersede one; usable rankings remain immutable.
	_, err := r.db.ExecContext(ctx, "INSERT INTO sri_segment_snapshots (snapshot_id,tenant_id,segment_id,session_date,as_of_time,state,composite_score,relative_strength_score,momentum_score,momentum_acceleration,rank,rank_change_5d,evidence_quality,quality_state,quality_flags,components,input_provenance,algorithm_version,configuration_version,calculation_run_id,deterministic_key) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21) ON CONFLICT (tenant_id,deterministic_key) DO UPDATE SET snapshot_id=EXCLUDED.snapshot_id, as_of_time=EXCLUDED.as_of_time, state=EXCLUDED.state, composite_score=EXCLUDED.composite_score, relative_strength_score=EXCLUDED.relative_strength_score, momentum_score=EXCLUDED.momentum_score, momentum_acceleration=EXCLUDED.momentum_acceleration, rank=EXCLUDED.rank, rank_change_5d=EXCLUDED.rank_change_5d, evidence_quality=EXCLUDED.evidence_quality, quality_state=EXCLUDED.quality_state, quality_flags=EXCLUDED.quality_flags, components=EXCLUDED.components, input_provenance=EXCLUDED.input_provenance, configuration_version=EXCLUDED.configuration_version, calculation_run_id=EXCLUDED.calculation_run_id WHERE sri_segment_snapshots.quality_state='partial' AND EXCLUDED.quality_state='usable'", x.SnapshotID, x.TenantID, x.SegmentID, x.SessionDate.UTC(), x.AsOfTime.UTC(), x.State, x.CompositeScore, x.RelativeStrengthScore, x.MomentumScore, x.MomentumAcceleration, x.Rank, x.RankChange5D, x.EvidenceQuality, x.QualityState, jsonOrEmpty(x.QualityFlagsJSON), jsonOrEmpty(x.ComponentsJSON), jsonOrEmpty(x.InputProvenanceJSON), x.AlgorithmVersion, x.ConfigurationVersion, x.CalculationRunID, x.DeterministicKey)
	if err != nil {
		return fmt.Errorf("upsert sri snapshot: %w", err)
	}
	return nil
}
func (r *Repository) ListMarketOpsSRISegments(ctx context.Context, tenantID string, activeOnly bool, limit int) ([]storage.MarketOpsSRISegmentRecord, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT tenant_id,segment_id,segment_key,name,segment_type,parent_segment_key,active,registry_version,metadata FROM sri_segments WHERE tenant_id=$1 AND ($2=false OR active=true) ORDER BY segment_type,name LIMIT $3", strings.TrimSpace(tenantID), activeOnly, clampLimit(limit))
	if err != nil {
		return nil, err
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
func (r *Repository) ListMarketOpsSRIETFRegistry(ctx context.Context, tenantID, segmentID string) ([]storage.MarketOpsSRIETFRecord, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT tenant_id,etf_symbol,segment_id,role,benchmark_priority,active,registry_version,config FROM sri_etf_registry WHERE tenant_id=$1 AND ($2='' OR segment_id=$2) AND active=true ORDER BY segment_id,benchmark_priority,etf_symbol", strings.TrimSpace(tenantID), strings.TrimSpace(segmentID))
	if err != nil {
		return nil, err
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
func (r *Repository) ListMarketOpsSRISnapshots(ctx context.Context, f storage.MarketOpsSRISnapshotFilter) ([]storage.MarketOpsSRISnapshotRecord, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT q.snapshot_id,q.tenant_id,q.segment_id,q.session_date,q.as_of_time,q.state,q.composite_score,q.relative_strength_score,q.momentum_score,q.momentum_acceleration,q.rank,q.rank_change_5d,q.evidence_quality,q.quality_state,q.quality_flags,q.components,q.input_provenance,q.algorithm_version,q.configuration_version,q.calculation_run_id,q.deterministic_key FROM sri_segment_snapshots q JOIN sri_segments s ON s.tenant_id=q.tenant_id AND s.segment_id=q.segment_id WHERE q.tenant_id=$1 AND ($2='' OR q.segment_id=$2) AND ($3='' OR s.segment_type=$3) AND ($4='' OR q.state=$4) AND ($5='' OR q.quality_state=$5) AND ($6::timestamptz IS NULL OR q.session_date >= $6::date) AND ($7::timestamptz IS NULL OR q.session_date <= $7::date) ORDER BY q.session_date DESC,q.rank NULLS LAST LIMIT $8", f.TenantID, f.SegmentID, f.SegmentType, f.State, f.QualityState, nullTime(f.SessionStart), nullTime(f.SessionEnd), clampLimit(f.Limit))
	if err != nil {
		return nil, err
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

var _ = time.Now

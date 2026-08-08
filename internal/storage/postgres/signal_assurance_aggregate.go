package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

func (r *Repository) RefreshSignalAssuranceEffectiveness(ctx context.Context, tenantID, mode string, asOf time.Time, engineVersion string) error {
	rows, err := r.db.QueryContext(ctx, `SELECT signal_type,count(*),count(*) FILTER (WHERE state='MATERIALIZED'),count(*) FILTER (WHERE state IN ('ACTIVE')),count(*) FILTER (WHERE state IN ('INVALIDATED','SUPERSEDED','EXPIRED')) FROM signal_assertions WHERE tenant_id=$1 AND evaluation_mode=$2 GROUP BY signal_type`, tenantID, mode)
	if err != nil {
		return fmt.Errorf("aggregate signal assurance: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var signalType string
		var sample, materialized, censored, excluded int
		if err := rows.Scan(&signalType, &sample, &materialized, &censored, &excluded); err != nil {
			return err
		}
		metrics, _ := json.Marshal(map[string]any{"materialization_rate": ratio(materialized, sample), "scope": "signal_type"})
		id := aggregateID(tenantID, mode, "signal_type", signalType, engineVersion)
		if _, err := r.db.ExecContext(ctx, `INSERT INTO signal_effectiveness_snapshots (snapshot_id,tenant_id,evaluation_mode,dimension_key,dimension_value,sample_size,materialized_count,censored_count,excluded_count,metrics,as_of,evaluation_engine_version) VALUES ($1,$2,$3,'signal_type',$4,$5,$6,$7,$8,$9,$10,$11) ON CONFLICT (tenant_id,evaluation_mode,dimension_key,dimension_value,evaluation_engine_version) DO UPDATE SET sample_size=EXCLUDED.sample_size,materialized_count=EXCLUDED.materialized_count,censored_count=EXCLUDED.censored_count,excluded_count=EXCLUDED.excluded_count,metrics=EXCLUDED.metrics,as_of=EXCLUDED.as_of,created_at=now()`, id, tenantID, mode, signalType, sample, materialized, censored, excluded, metrics, asOf.UTC(), engineVersion); err != nil {
			return err
		}
	}
	return rows.Err()
}
func ratio(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d)
}
func aggregateID(parts ...string) string {
	h := sha256.Sum256([]byte(fmt.Sprint(parts)))
	return "saf_effectiveness_" + hex.EncodeToString(h[:])[:24]
}

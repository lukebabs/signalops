package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) InsertMarketOpsOpportunityDisposition(ctx context.Context, record storage.MarketOpsOpportunityDispositionRecord) error {
	if err := validateMarketOpsOpportunityDisposition(record); err != nil {
		return err
	}
	result, err := r.db.ExecContext(ctx, `
INSERT INTO marketops_opportunity_dispositions (
  disposition_id, tenant_id, opportunity_id, disposition, actor, note, metadata, created_at
)
SELECT $1, $2, $3, $4, $5, $6, $7, $8
WHERE EXISTS (
  SELECT 1 FROM marketops_opportunities WHERE tenant_id=$2 AND opportunity_id=$3
)
ON CONFLICT (disposition_id) DO NOTHING`,
		strings.TrimSpace(record.DispositionID), strings.TrimSpace(record.TenantID),
		strings.TrimSpace(record.OpportunityID), strings.TrimSpace(record.Disposition),
		strings.TrimSpace(record.Actor), strings.TrimSpace(record.Note),
		jsonOrEmpty(record.MetadataJSON), record.CreatedAt.UTC())
	if err != nil {
		return fmt.Errorf("insert marketops opportunity disposition: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("insert marketops opportunity disposition rows affected: %w", err)
	}
	if rows == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (r *Repository) ListMarketOpsOpportunityDispositions(ctx context.Context, filter storage.MarketOpsOpportunityDispositionFilter) ([]storage.MarketOpsOpportunityDispositionRecord, error) {
	if strings.TrimSpace(filter.TenantID) == "" {
		return nil, fmt.Errorf("marketops opportunity disposition tenant_id is required")
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT disposition_id, tenant_id, opportunity_id, disposition, actor, note, metadata, created_at
FROM marketops_opportunity_dispositions
WHERE tenant_id=$1 AND ($2='' OR opportunity_id=$2) AND ($3='' OR disposition=$3)
ORDER BY created_at DESC, disposition_id DESC
LIMIT $4`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.OpportunityID),
		strings.TrimSpace(filter.Disposition), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops opportunity dispositions: %w", err)
	}
	defer rows.Close()
	out := []storage.MarketOpsOpportunityDispositionRecord{}
	for rows.Next() {
		var record storage.MarketOpsOpportunityDispositionRecord
		if err := rows.Scan(&record.DispositionID, &record.TenantID, &record.OpportunityID,
			&record.Disposition, &record.Actor, &record.Note, &record.MetadataJSON, &record.CreatedAt); err != nil {
			return nil, mapScanError("scan marketops opportunity disposition", err)
		}
		out = append(out, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list marketops opportunity dispositions rows: %w", err)
	}
	return out, nil
}

func validateMarketOpsOpportunityDisposition(record storage.MarketOpsOpportunityDispositionRecord) error {
	if err := requireMarketOpsFields("opportunity disposition", map[string]string{
		"disposition_id": record.DispositionID, "tenant_id": record.TenantID,
		"opportunity_id": record.OpportunityID, "disposition": record.Disposition, "actor": record.Actor,
	}); err != nil {
		return err
	}
	if !oneOf(record.Disposition, storage.MarketOpsOpportunityDispositionWatch,
		storage.MarketOpsOpportunityDispositionAdvance,
		storage.MarketOpsOpportunityDispositionNeedsMoreEvidence,
		storage.MarketOpsOpportunityDispositionDismiss,
		storage.MarketOpsOpportunityDispositionResolved) {
		return fmt.Errorf("marketops opportunity disposition is invalid")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("marketops opportunity disposition created_at is required")
	}
	return validateJSONObject("marketops opportunity disposition metadata", jsonOrEmpty(record.MetadataJSON))
}

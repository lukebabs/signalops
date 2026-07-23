WITH ranked AS (
  SELECT snapshot_id,
         row_number() OVER (
           PARTITION BY tenant_id, universe_group, symbol
           ORDER BY as_of_time DESC, created_at DESC, snapshot_id DESC
         ) AS row_number
  FROM marketops_intraday_condition_snapshots
)
DELETE FROM marketops_intraday_condition_snapshots snapshots
USING ranked
WHERE snapshots.snapshot_id = ranked.snapshot_id
  AND ranked.row_number > 1;

CREATE UNIQUE INDEX IF NOT EXISTS marketops_intraday_condition_snapshots_current_idx
  ON marketops_intraday_condition_snapshots (tenant_id, universe_group, symbol);

-- The reconciliation queue originally covered the static Top 50 universe.
-- MarketOps now reconciles the active, centrally governed catalog in bounded
-- batches (up to the hot-universe capacity), so queue priority must support
-- that range without weakening its positive-rank invariant.
ALTER TABLE marketops_equity_reconciliation_tasks
  DROP CONSTRAINT IF EXISTS marketops_equity_reconciliation_tasks_universe_rank_check;

ALTER TABLE marketops_equity_reconciliation_tasks
  ADD CONSTRAINT marketops_equity_reconciliation_tasks_universe_rank_check
  CHECK (universe_rank > 0 AND universe_rank <= 1000);

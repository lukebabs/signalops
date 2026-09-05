ALTER TABLE marketops_equity_reconciliation_tasks
  DROP CONSTRAINT IF EXISTS marketops_equity_reconciliation_tasks_universe_rank_check;

ALTER TABLE marketops_equity_reconciliation_tasks
  ADD CONSTRAINT marketops_equity_reconciliation_tasks_universe_rank_check
  CHECK (universe_rank > 0 AND universe_rank <= 50);

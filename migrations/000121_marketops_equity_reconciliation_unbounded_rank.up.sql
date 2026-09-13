-- Queue priority is an ordering attribute, not a universe-size policy.
-- Provider capacity is governed by the run configuration; reconciliation rank has
-- no fixed upper bound so catalog growth cannot make queued work invalid.
ALTER TABLE marketops_equity_reconciliation_tasks
  DROP CONSTRAINT IF EXISTS marketops_equity_reconciliation_tasks_universe_rank_check;

ALTER TABLE marketops_equity_reconciliation_tasks
  ADD CONSTRAINT marketops_equity_reconciliation_tasks_universe_rank_check
  CHECK (universe_rank > 0);

-- Retain a ranked candidate pool larger than the 1,000-asset warm capacity.
-- The planner still selects at most 1,000 verified US common stocks.
ALTER TABLE subscriber_global_ranking_snapshots
  DROP CONSTRAINT subscriber_global_ranking_snapshots_requested_capacity_check;
ALTER TABLE subscriber_global_ranking_snapshots
  ADD CONSTRAINT subscriber_global_ranking_snapshots_requested_capacity_check
  CHECK (requested_capacity > 0 AND requested_capacity <= 10000);

ALTER TABLE subscriber_global_ranking_snapshot_entries
  DROP CONSTRAINT subscriber_global_ranking_snapshot_entries_selection_rank_check;
ALTER TABLE subscriber_global_ranking_snapshot_entries
  ADD CONSTRAINT subscriber_global_ranking_snapshot_entries_selection_rank_check
  CHECK (selection_rank > 0 AND selection_rank <= 10000);

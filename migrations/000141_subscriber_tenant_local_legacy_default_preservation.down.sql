-- This preservation migration intentionally has no destructive rollback.
-- Disable a future tenant-local subscriber feature flag before any separately
-- approved list-policy change; retain the list, memberships, audit, and legacy
-- source records for reconciliation and rollback evidence.
SELECT 1;

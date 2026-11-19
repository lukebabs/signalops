-- Parity manifests and any later global evidence are immutable operational
-- records. Revert by restoring a pre-migration backup; do not remove the
-- evidence kind or source view while it may be referenced by a manifest.
SELECT 1;

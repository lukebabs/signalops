# S5 Canonical Catalog Projection — 2026-08-13

The global catalog retains source-level records for provenance. A single listed security may therefore have multiple source rows, such as the retained S&P Global and Massive records for AAPL.

Migration `000100_subscriber_catalog_canonical_projection` changes only the subscriber search projection. It resolves each eligible source row through `subscriber_global_asset_identity_resolutions`, selects the canonical global asset, and returns one result per governed security. The deterministic resolver prefers the Massive record when available.

No source record, reference observation, coverage evidence, historical membership, or identity-resolution audit is deleted or rewritten. The database test confirmed one canonical AAPL result after the migration and two raw-source results after rollback.

Private-list membership mutations remain independently authorized and continue to use the returned canonical asset ID from the catalog surface. The existing S5 quota policy is unchanged.

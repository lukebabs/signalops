# S5 Catalog Membership Repair — 2026-08-13

The pilot HAR established that authentication, tenant binding, private-list creation, catalog search, and canonical asset selection worked. The catalog-membership POST then returned `400 invalid_subscriber_membership` despite a valid private-list ID and governed global asset ID.

Root cause: migration `000099` used a data-modifying SQL CTE to write an audit mutation named `add_catalog_asset`. That mutation is outside the established immutable audit taxonomy, and the CTE form also interacted poorly with forced RLS. The error was normalized too broadly by the API as an invalid membership request.

Migration `000101_subscriber_catalog_function_rls` replaces only that function with an equivalent scoped PL/pgSQL implementation. It validates the transaction tenant, private-list ownership, immutable subject, and eligible global asset; writes a standard `add_asset` audit; and creates only the existing deduplicated central activation request. It does not change provider access, list ownership, tenant-default policy, or cross-tenant visibility.

The retry repair re-evaluates a previously released quota reservation before re-reserving it. This permits a valid retry after the failed pre-repair request while preserving the ten-request pilot quota.

Transactional verification produced one queued result for the pilot user and zero rows for the same list when scoped to `tenant-local`; all probe mutations were rolled back.

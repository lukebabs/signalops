# S3 Pilot Tenant, Tier, and Default-List Decision

Date: 2026-08-12 UTC

## Decision

The first S3 pilot tenant is `tenant-pilot-b`. It already has the isolated OIDC tenant claim and established MarketOps read grant required to test the subscriber boundary.

Its provisioned tier is `subscriber-list-pilot-v1`:

| Capability | Enabled | Quota | Pilot meaning |
| --- | ---: | ---: | --- |
| `catalog_search` | No | 0 | No global catalog search or projection is enabled. |
| `eod_activation` | No | 0 | A list membership cannot warm a cold asset or change shared EOD coverage. |
| `options_demand` | No | 0 | A list membership cannot contribute to options collection. |

The entitlement is active because S3 list preferences are being piloted. The capabilities remain explicit default-deny, so activation of the list API does not expand provider usage, coverage, or data product scope.

## Default list policy

The tenant has exactly one shared default list:

- ID: `sublist_3273f605844d5859f55e8117`
- Name: **MarketOps Pilot Default**
- Kind: `tenant_default`
- Provisioning actor: `subscriber-pilot-provisioner`
- Correlation ID: `subscriber-s3-pilot-seed-2026-08-12`

It is seeded from the current governed S2 ranked hot-set shadow plan `subeodplan_bde2b62388cf4a7861c92734`, not from a tenant-local watchlist. The resulting central membership references are:

| Rank | Symbol |
| ---: | --- |
| 1 | NVDA |
| 2 | AAPL |
| 3 | GOOGL |
| 4 | MSFT |
| 5 | AMZN |
| 6 | AVGO |
| 7 | SPCX |
| 8 | META |
| 9 | TSLA |
| 10 | BRK.B |

The seed created one list-audit record and ten membership-audit records. It created no global assets, coverage row, activation request, provider call, scheduled job, or legacy MarketOps mutation.

## Controlled provisioning

The reusable command is `cmd/subscriber-s3-pilot-provisioner`. It writes through the dedicated subscriber gateway database login and normal RLS-scoped repository methods. It provisions the default-deny entitlement, reuses an existing tenant-default list when present, and idempotently adds the supplied global asset IDs.

The command does not enable `SIGNALOPS_SUBSCRIBER_LISTS_ENABLED`. Local configuration names only `tenant-pilot-b` as the prospective pilot tenant; the actual feature flag remains false.

## Remaining enablement evidence

Before setting the flag true for this one tenant, retain:

1. output from the S3 pilot readiness preflight using the deployment-secret-managed gateway login;
2. deployment evidence that the flag is true only for `tenant-pilot-b`;
3. browser evidence from a tenant administrator, two pilot users, and a user in another tenant; and
4. a rollback check that turns the flag off without deleting the entitlement, list, memberships, or audit.

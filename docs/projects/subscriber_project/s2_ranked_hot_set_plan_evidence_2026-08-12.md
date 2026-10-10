# S2 Ranked Hot-Set Plan Evidence

Date: 2026-08-12 UTC

## Ranked input

The user-supplied `companies.csv` ranking was imported without committing the source file. Its retained SHA-256 is `3eeabf25c0fedc02bceb0435da301631ddc92f64cf38f68d772fc8623939b272` and its as-of date is 2026-08-12.

The immutable current snapshot is `subrank_a4fd5bd500262f7f2193b48c`. It examined 1,001 source rows to select 1,000 distinct symbols, skipping one duplicate (`WRB`). Each selected entry retains its raw source rank, selected rank, market-cap/revenue text, row checksum, and source checksum.

## Governed selection result

The final shadow plan is `subeodplan_bde2b62388cf4a7861c92734`, produced by `s2-eod-hot-set-shadow-v1` in `shadow` mode with a capacity of 1,000.

| Measure | Result |
| --- | ---: |
| Ranked candidates | 1,000 |
| Eligible canonical assets selected | 881 |
| Excluded candidates | 119 |
| Global catalog assets currently eligible | 985 |
| Global catalog assets currently ineligible | 119 |
| Global catalog assets still discovered | 1 |

The one discovered ranked symbol is `UMBF.O`, selected at rank 904 (source rank 905). Massive reference lookup returned no usable record, so the catalog retains an immutable `massive_reference_lookup_failed` deferred decision. The system does not infer a replacement symbol, promote the asset, or schedule collection from the company-name match.

## Safety result

- No coverage record is outside `shadow` mode.
- No open global coverage-activation request exists.
- The browser gateway has no read privilege on either plan runs or ranking entries.
- No provider collection, scheduler, MarketOps universe mutation, or tenant projection was enabled.

S2 is complete as a governed catalog and EOD-planner shadow. The activation-request schema is ready, but its writer correctly remains deferred until S3 introduces authenticated tenant-default and subject-owned private list membership. This preserves the rule that no cold asset can be warmed from an unauthorized browser action.

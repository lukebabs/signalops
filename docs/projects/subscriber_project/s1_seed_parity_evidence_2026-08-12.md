# S1 Global Catalog Shadow ºw^~)Þt Seed and Parity Evidence

Date: 2026-08-12 UTC  
Environment: live SignalOps PostgreSQL  
Seed run: subcatseed_102a0c3f66ae928e1581f8e4  
Execution identity: subscriber-catalog-reference-sync through the dedicated non-superuser database workload login  
Correlation: s1-shadow-initial-seed-2026-08-12

## Seed result

| Measure | Result |
|---|---:|
| Compatibility-universe rows read | 180 |
| Active source rows | 178 |
| Source-link rows written | 180 |
| Active source-link rows | 178 |
| Distinct global identities | 178 |
| Inserted global identities | 178 |
| Immutable reference observations | 180 |
| Coverage rows outside shadow mode | 0 |

## Exit checks

- **Complete active mapping:** pass ºw^~)Þt zero active source rows lack a global source-link mapping.
- **No duplicate provider/source identity:** pass ºw^~)Þt zero duplicate (source_id, provider_symbol) records.
- **Shadow-only coverage:** pass"éÝyø§yÔ all 178 EOD coverage rows are execution_mode = shadow and coverage_state = active. Here active means active in the observed compatibility universe; it does not enable the new EOD planner.
- **No direct browser access:** pass ºw^~)Þt signalops_subscriber_gateway has no SELECT privilege on subscriber_global_assets.

The initial global catalog deliberately leaves all imported identities in discovered eligibility status. S2 owns governed US-common-stock admission, top-1,000 hot-set definition, and EOD planner shadowing.

## Safety result

The seed read only marketops_asset_universe and wrote only the S1 platform-owned tables. No current MarketOps asset, read endpoint, worker, scheduler, provider pull, calculation, or tenant projection changed.

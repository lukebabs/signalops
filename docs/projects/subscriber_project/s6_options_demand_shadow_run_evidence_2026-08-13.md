# S6 Options-Demand Shadow Run Evidence — 2026-08-13

Status: first dedicated-login shadow snapshot completed; no Options demand was eligible.

The planner authenticated as `signalops_subscriber_options_demand_runner`, explicitly assumed only `signalops_subscriber_options_demand`, and persisted:

| Field | Observed value |
| --- | --- |
| Snapshot | `suboptdemand_be5667e0b05ed529f6dbbab3` |
| Session | 2026-08-12 |
| Execution mode | `shadow` |
| Source demand / candidates / selected / deferred | `0 / 0 / 0 / 0` |
| Planner identity | `subscriber-options-demand-planner` |
| Provider execution | `false` |
| Capture execution | `false` |

The empty result is expected because all tenant `options_demand` capabilities remain default-deny. The run did not contact a provider, read a direct membership or Options table, enqueue capture, mutate an entitlement, or start a scheduler.

This validates the dedicated runtime path. A non-zero shadow plan requires a separate, explicit product-policy decision to enable a bounded Options-demand entitlement; it still would not authorize capture.

# G105 Reviewed Label Batch Inventory And Sync Readiness

Status: live-validated inventory and sync-readiness gate.

## Purpose

G105 executes the first non-decision step from G104: inventory available graph proposal review candidates, sync any already-reviewed graph proposal decisions into evaluation labels, and determine whether the first `25` matched-label milestone can be reached without synthetic labels or automatic review decisions.

This gate does not make semantic graph proposal decisions. Accept/reject/supersede decisions require operator judgment.

## Current Label State

Before and after the idempotent G084 sync:

- Persisted evaluation labels: `7`.
- Distinct graph facts: `7`.
- Label counts: `positive=7`.
- Decision status counts: `accepted=7`.
- Subject symbols: `SPY=5`, `AAPL=2`.
- Rejected labels: `0`.
- Superseded labels: `0`.

G084 sync result:

- Endpoint: `POST /v1/marketops/backtest-evaluation-labels/sync`.
- Request scope: `tenant-local`, `marketops`, `market_data`, `daily_market_surveillance`, `limit=200`.
- Response: `synced=7`, `positive=7`.
- Persisted label count after sync remained `7`, confirming no duplicate count inflation.

## Proposal Inventory

Canonical source: `GET /v1/marketops/dsm/graph-proposals`.

Status inventory with `limit=50` per status:

- `proposed`: `50` rows, `39` distinct graph facts.
- `accepted`: `7` rows, `7` distinct graph facts.
- `rejected`: `0` rows.
- `superseded`: `0` rows.

Available proposed coverage:

- Symbols: `AAPL`, `NVDA`, `SPY`.
- Candidate types: `node_candidate`, `relationship_candidate`.
- Signal types: `marketops.dsm.accumulation`, `marketops.dsm.hedging_pressure`, `marketops.dsm.pinning_risk`, `marketops.dsm.speculative_call_pressure`, `marketops.dsm.speculative_put_pressure`, `marketops.dsm.volatility_expansion`.

## First Review Batch

The first G104 milestone is `25` matched reviewed labels. Current count is `7`, so the next batch needs at least `18` additional reviewed distinct graph facts.

G105 identified the following bounded `18`-fact proposed review queue. These are candidates for operator review only; no decision was applied by this gate.

| # | Proposal ID | Symbol | Signal Type | Candidate Type |
|---|---|---|---|---|
| 1 | `graphprop_marketops_dsm_v1_72581d337915a2400280bfd5` | `NVDA` | `marketops.dsm.volatility_expansion` | `node_candidate` |
| 2 | `graphprop_marketops_dsm_v1_20f09f1d62e22d5081b52f68` | `NVDA` | `marketops.dsm.volatility_expansion` | `node_candidate` |
| 3 | `graphprop_marketops_dsm_v1_c6f605309eadadaae0259916` | `NVDA` | `marketops.dsm.volatility_expansion` | `relationship_candidate` |
| 4 | `graphprop_marketops_dsm_v1_c2d79239ea6de007070450c2` | `NVDA` | `marketops.dsm.volatility_expansion` | `relationship_candidate` |
| 5 | `graphprop_marketops_dsm_v1_ecb888a0573881a06c2aff32` | `NVDA` | `marketops.dsm.volatility_expansion` | `node_candidate` |
| 6 | `graphprop_marketops_dsm_v1_5a5c72998b0def424fdc4ddd` | `SPY` | `marketops.dsm.speculative_call_pressure` | `node_candidate` |
| 7 | `graphprop_marketops_dsm_v1_d5d5fa302ab64a7bc8323ff0` | `SPY` | `marketops.dsm.speculative_call_pressure` | `relationship_candidate` |
| 8 | `graphprop_marketops_dsm_v1_dda9ebe99ed9687dd2fac10a` | `SPY` | `marketops.dsm.speculative_call_pressure` | `relationship_candidate` |
| 9 | `graphprop_marketops_dsm_v1_837b957de464b337ff9ae0f9` | `SPY` | `marketops.dsm.speculative_call_pressure` | `node_candidate` |
| 10 | `graphprop_marketops_dsm_v1_c0a718917726c768f344bfa4` | `SPY` | `marketops.dsm.speculative_call_pressure` | `node_candidate` |
| 11 | `graphprop_marketops_dsm_v1_2e915a8cfaa35765e515ba80` | `SPY` | `marketops.dsm.hedging_pressure` | `node_candidate` |
| 12 | `graphprop_marketops_dsm_v1_66383261bedc6da586186259` | `SPY` | `marketops.dsm.hedging_pressure` | `relationship_candidate` |
| 13 | `graphprop_marketops_dsm_v1_8c663020d77ef5450fadaf25` | `SPY` | `marketops.dsm.hedging_pressure` | `relationship_candidate` |
| 14 | `graphprop_marketops_dsm_v1_a71a7236f722c8d843df0130` | `SPY` | `marketops.dsm.hedging_pressure` | `node_candidate` |
| 15 | `graphprop_marketops_dsm_v1_09cba78191845c8f442895cc` | `AAPL` | `marketops.dsm.pinning_risk` | `relationship_candidate` |
| 16 | `graphprop_marketops_dsm_v1_d3480a4bf5a86cd708926778` | `AAPL` | `marketops.dsm.pinning_risk` | `relationship_candidate` |
| 17 | `graphprop_marketops_dsm_v1_7bee18e3812cb5c7c5cad4e5` | `AAPL` | `marketops.dsm.pinning_risk` | `node_candidate` |
| 18 | `graphprop_marketops_dsm_v1_e3879ae5d74e992081bb579a` | `AAPL` | `marketops.dsm.pinning_risk` | `node_candidate` |

## Result

G105 confirms the first reviewed-label milestone is operationally reachable from current proposed evidence: `39` distinct proposed graph facts exist, and a bounded `18`-fact queue is available to move from `7` to at least `25` matched reviewed labels if operators review all candidates.

No G085 evaluation or G094 readiness re-check was rerun because the reviewed label set did not change. The next execution step is operator review of the queued proposals, followed by G084 sync, G085 evaluation, and G094 readiness re-check.

## Boundary

No accept/reject/supersede decisions, synthetic labels, threshold relaxation, detector mutation, policy deployment, graph writeback, frontend change, or ingestion expansion was performed.

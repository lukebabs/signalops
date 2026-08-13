# S4 Canary Execution Gate

Status: implemented as a disabled-by-default control plane. It does not make a provider request, schedule a job, update global coverage, or alter existing MarketOps scheduling.

## Goal

The prepared S4 cohort is exactly two frozen canonical assets for the completed 2026-08-12 session:

| Request ordinal | Symbol | Canary member |
|---:|---|---|
| 1 | NVDA | `subglobal_b9dd5a39559ce4ea0ebed63f1574411e` |
| 2 | AAPL | `subglobal_c0a811ecc7ae886f86a517dda3c4d398` |

Migration `000102_subscriber_global_eod_canary_execution_gate` creates a separate append-only execution-plan record. It cannot alter the original prepared canary. Its database constraints require all of the following:

- exactly one execution plan per frozen canary;
- a maximum of two request slots, each tied to one frozen canonical asset;
- `provider_execution_enabled=false`;
- `scheduled_execution_enabled=false`;
- `kill_switch_engaged=true`; and
- `execution_state='disabled'`.

The migration also creates an immutable per-symbol evidence ledger. A provider request evidence event is unique per frozen symbol; because the plan has only two numbered slots, the plan has a hard two-request ceiling. No browser credential can create the records.

## Evidence and parity contract

Creating the disabled gate writes one `execution_planned` event per symbol. It captures the canonical symbol, ordinal, expected algorithm version `subscriber-global-eod-baseline-v1`, validation-contract reference, frozen plan/canary provenance, and `parity_status=not_started`.

A separately reviewed future execution release must only append evidence in this sequence: `provider_request_started`, `provider_response_received`, `normalization_completed`, then `parity_compared`. The parity event must compare the same symbol/session with tenant-local output and record algorithm version, normalized result fingerprint, quality/coverage state, source timestamp, baseline/provenance, and a precise mismatch reason when not equal. Absence of a parity event is an incomplete canary, never a passing result.

## Workload preflight

The gate is restricted to the `subscriber-global-eod-reconciler` machine identity and its dedicated `signalops_subscriber_global_eod` database group. Before persisting a gate, run:

```bash
SIGNALOPS_SUBSCRIBER_WORKLOAD_IDENTITY=subscriber-global-eod-reconciler \
  bash ./scripts/subscriber_project_global_eod_canary_workload_preflight.sh
```

This verifies a non-superuser, non-CREATEROLE, NOBYPASSRLS workload login; required global EOD records; no mutation access to frozen/evidence records; and no access to subscriber lists, entitlements, or legacy MarketOps asset ownership. It validates a deployment identity declaration, not a browser token, and makes no provider request.

After migration and a passing preflight, an operator may persist only the disabled plan:

```bash
go run ./cmd/subscriber-global-eod-canary-execution-gate \
  --execute \
  --acknowledge-provider-disabled \
  --canary-run-id subeodcanary_c44b065fb587d29470db5119 \
  --correlation-id subscriber-s4-gate-aapl-nvda-2026-08-13
```

The command does not import a market-data client and cannot make a network/provider request. It fails unless the canary is still prepared, parity-required, disabled, and contains exactly two members.

## Kill and recovery

The kill switch begins engaged and cannot be changed by this release. Restarting or redeploying the worker therefore leaves collection disabled. Existing evidence is retained for audit; rollback drops only this additive control-plane schema after an operator confirms no later execution-release tables depend on it. Existing MarketOps jobs and the S2 shadow planner are unaffected.

Actual provider execution remains a later explicit authorization gate. It requires a dedicated short-lived workload credential, a separately reviewed code release that preserves the two-request ceiling, an explicit named session approval, live parity/reporting verification, and the existing backup/restore readiness work. It must not be enabled by a browser action or by this command.

## Live execution release

Migration `000103_subscriber_global_eod_canary_live_execution` adds a one-time authorization record and separate append-only live-run ledger. An authorization is restricted to the existing frozen execution plan, `subscriber-global-eod-reconciler`, Massive, the original session, and exactly two members. It cannot enable a scheduler. A request-intent row is committed for both frozen symbols before either external call; each symbol has no retry path and only one intent row. If the process stops after intent, it fails closed rather than risking an unrecorded repeat call.

The live worker writes a central global EOD baseline record and immutable provider response/normalization evidence for each symbol. Its canonical payload is the established Massive EOD OHLCV contract and its algorithm version is `subscriber-global-eod-baseline-v1`. It does not accept tenant, symbol, or execution-plan input from a browser.

The separate parity reporter makes no provider call. It compares each global canonical payload with the `tenant-local` `src-massive` normalized EOD ledger for the same symbol/session. It records `matched`, `mismatched`, or `missing`; only two `matched` rows constitute a successful canary. A missing comparison is an incomplete canary, never a pass.

The previously deployed gate remains intact and disabled. The live authorization is a new, separately auditable, exact-two-request exception; it does not weaken or update the original gate.

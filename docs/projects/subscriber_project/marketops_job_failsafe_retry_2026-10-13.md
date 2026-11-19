# MarketOps job fail-safe and retry policy — 2026-10-13

## Policy

MarketOps scheduled jobs use Kubernetes `concurrencyPolicy: Forbid` and a bounded
`backoffLimit`. Deterministic jobs may retry once at the pod level. Provider-backed
capture jobs remain single-attempt so a transient failure cannot duplicate vendor
requests or consume another capture budget; they are retried by an explicit,
dependency-aware recovery run.

The operations monitor reconciles any `running` status or run older than two hours
to `recovery_needed` with exit code `42` before evaluating health. This prevents a
lost pod from permanently blocking task recovery. The monitor excludes its own
previous failed status from the failure count, avoiding a self-sustaining failure
loop.

## Retried deterministically

- SAF benchmark materialization
- Operations monitor
- Task-retry controller
- Syncratic Intelligence materialization
- Post-close recovery
- Risk/Reward materialization

These jobs retry at most once (`backoffLimit: 1`) and remain protected by
`Forbid` concurrency and their existing dependency gates.

## Provider-backed single attempt

Intraday, warm-EOD, daily post-close, SRI capture/holdings, and FMP continuation
remain `backoffLimit: 0`. Recovery for these jobs is explicit and dependency-aware,
so a failed pod cannot silently issue duplicate provider requests.

## Operational evidence

The monitor prints reconciled stale leases and all health counters. A successful
cycle requires zero unresolved failed jobs (excluding the monitor's own prior
status), zero stale running leases, zero mutating dry-runs, and fresh intraday
evidence during market hours.

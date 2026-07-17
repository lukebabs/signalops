# G131 Quality-Aware Algorithm Proposals

Status: implemented backend/proposal gate

Date: 2026-07-17

## Purpose

G131 closes the immediate follow-up from G130: algorithm results for options distribution features can still be written for audit and diagnostics, but low-quality call/put open-interest ratio evidence must not become reviewable signal proposals.

This keeps the generic SignalOps algorithm substrate intact while adding a narrow evidence-quality gate at the proposal boundary.

## Scope

- Propagate normalized options-distribution quality metadata into `algorithm_results.result_payload`.
- Gate proposal generation for `options_distribution_daily` + `call_put_open_interest_ratio`.
- Allow proposals only when `call_put_oi_ratio_quality=usable`.
- Preserve proposal behavior for other datasets and features.
- Add explicit `quality_gate` metadata to generated proposal payloads.

## Behavior

The algorithm runner copies these quality fields from normalized event payloads into result payloads when present:

- `open_interest_quality`
- `open_interest_zero_count`
- `open_interest_positive_count`
- `open_interest_zero_rate`
- `call_put_oi_denominator_is_zero`
- `call_put_oi_ratio_quality`

The proposal generator applies policy `g131.options_distribution_quality.v1` only to:

- `dataset=options_distribution_daily`
- `feature=call_put_open_interest_ratio`

For that scoped feature, proposal generation requires:

- `call_put_oi_ratio_quality=usable`

Non-usable states such as `all_zero`, `denominator_zero`, and `partial_zero` are skipped. They remain in `algorithm_results` for investigation, but do not enter the operator proposal queue.

## Validation

Automated validation:

- Focused Go tests passed for:
  - `./internal/algorithms`
  - `./internal/algorithms/proposals`
  - `./cmd/algorithm-runner`
  - `./cmd/algorithm-proposal-generator`
- Docker target builds passed for:
  - `algorithm-runner`
  - `algorithm-proposal-generator`

Live NVDA validation over execution request `algexec_9b5c5859ecb0d78233495268`:

- Algorithm results written: 27.
- Result quality breakdown:
  - `usable=9`
  - `all_zero=10`
  - `denominator_zero=6`
  - `partial_zero=2`
- Signal proposals generated: 9.
- Usable proposals: 9.
- Non-usable proposals: 0.

## Result

G131 makes options call/put ratio proposals quality-aware without deleting or hiding algorithm outputs. Analysts only review candidates backed by usable ratio evidence, while data-quality artifacts remain available for diagnostics and future UI surfacing.

## Deferred

- UI surfacing of ratio quality badges inside asset options and algorithm proposal views.
- Alternative downgrade behavior for `partial_zero` evidence.
- Top 50 batch proposal generation over broader options history.

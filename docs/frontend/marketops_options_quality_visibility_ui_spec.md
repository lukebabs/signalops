# MarketOps Options Quality Visibility UI Specification

Status: proposed for frontend-agent implementation
Gate: G132 frontend follow-up
Author: Codex
Date: 2026-07-17
Backend baseline: G130-G131, latest backend commit `1367148`

## Purpose

Add clear quality visibility for MarketOps options distribution evidence so analysts can understand why an options call/put ratio is usable, degraded, or blocked from becoming an algorithm signal proposal.

G130 added persisted quality metadata. G131 uses that metadata to block low-quality options ratio results from the proposal queue. G132 should make those decisions visible in the existing UI without adding ingestion controls, provider calls, scheduler controls, or algorithm execution changes.

## User Outcome

An analyst reviewing NVDA or another asset should be able to answer quickly:

- Is the current call/put open-interest ratio based on usable evidence?
- Are zero open-interest values affecting interpretation?
- Did a result become a proposal because it passed the quality gate?
- If proposal counts are lower than algorithm-result counts, is that because non-usable ratio evidence was skipped?

## Scope

In scope:

- Extend the existing `/marketops/assets` options panel from G128.
- Extend the existing `/marketops/algorithms` result/proposal views from G109/G114/G116/G123.
- Display options distribution quality fields already returned by existing APIs.
- Add compact badges, summary rows, table columns, and detail disclosures that make quality status interpretable.
- Preserve existing authenticated API conventions and React Query patterns.
- Add frontend type guards/tests for the quality fields.

Out of scope:

- No provider ingestion trigger.
- No Massive live-preview trigger.
- No Top 50 batch controls.
- No new algorithm execution controls.
- No materialization policy changes.
- No backend API changes unless implementation proves a required field is unavailable.
- No attempt to show skipped proposal rows from the backend, because skipped rows are not currently persisted as proposals.

## Backend Data Sources

Use existing endpoints only.

Asset options coverage/distribution/chain:

```http
GET /v1/tenants/{tenant_id}/marketops/assets/{symbol}/options/coverage
GET /v1/tenants/{tenant_id}/marketops/assets/{symbol}/options/distribution?window=10_trade_days&limit=10
GET /v1/tenants/{tenant_id}/marketops/assets/{symbol}/options/chain?trade_date=YYYY-MM-DD&contract_type=call&limit=500
```

Algorithm results and proposals:

```http
GET /v1/algorithms/results?tenant_id={tenant_id}&execution_request_id={execution_request_id}&algorithm_id={algorithm_id}&result_type={result_type}&limit=50
GET /v1/algorithms/results/{algorithm_result_id}?tenant_id={tenant_id}
GET /v1/algorithms/signal-proposals?tenant_id={tenant_id}&execution_request_id={execution_request_id}&algorithm_result_id={algorithm_result_id}&status={status}&limit=50
GET /v1/algorithms/signal-proposals/{proposal_id}?tenant_id={tenant_id}
GET /v1/algorithms/signal-proposals/summary?tenant_id={tenant_id}&execution_request_id={execution_request_id}&status={status}
```

## Quality Fields

Distribution snapshots may expose quality fields directly or under `metrics`, depending on the current DTO shape. The UI should read both locations defensively.

Preferred direct fields:

```json
{
  "open_interest_quality": "partial_zero",
  "open_interest_zero_count": 21,
  "open_interest_positive_count": 12,
  "open_interest_zero_rate": 0.636364,
  "call_put_oi_denominator_is_zero": false,
  "call_put_oi_ratio_quality": "usable"
}
```

Algorithm result payload fields from G131:

```json
{
  "dataset": "options_distribution_daily",
  "feature": "call_put_open_interest_ratio",
  "symbol": "NVDA",
  "open_interest_quality": "partial_zero",
  "open_interest_zero_count": 21,
  "open_interest_positive_count": 12,
  "open_interest_zero_rate": 0.636364,
  "call_put_oi_denominator_is_zero": false,
  "call_put_oi_ratio_quality": "usable"
}
```

Proposal payload fields from G131:

```json
{
  "quality_gate": {
    "passed": true,
    "policy": "g131.options_distribution_quality.v1"
  },
  "algorithm_result": {
    "payload": {
      "dataset": "options_distribution_daily",
      "feature": "call_put_open_interest_ratio",
      "call_put_oi_ratio_quality": "usable"
    }
  }
}
```

Known quality values:

- `usable`: call/put OI ratio is acceptable for proposal review.
- `partial_zero`: some OI values are zero; ratio should be treated as degraded evidence and currently does not pass G131 proposal gating for call/put OI ratio results.
- `all_zero`: all relevant OI values are zero; do not treat ratio as market signal evidence.
- `denominator_zero`: put-side denominator is zero; call/put OI ratio is not interpretable.
- Missing/unknown: show as `unknown`, not as usable.

## UX Requirements

### 1. Asset Options Panel

In `/marketops/assets`, when an asset is selected and the options panel is visible:

- Add a compact quality summary near the call/put ratio chart or snapshot header.
- Show `call_put_oi_ratio_quality` as a badge.
- Show `open_interest_quality` as a badge or secondary status.
- Show zero-rate as a percentage when present.
- Show zero/positive OI counts when present.
- Show denominator-zero as a warning state when true.
- Keep the visual density appropriate for an analyst workflow; avoid a large explanatory card.

Recommended badge semantics:

- `usable`: neutral/success badge.
- `partial_zero`: warning badge.
- `all_zero` and `denominator_zero`: blocked/error badge.
- `unknown`: muted badge.

For distribution rows, add a quality column or compact row adornment. The analyst should be able to scan historical rows and see whether the ratio quality changes over time.

For latest snapshot details, add a small disclosure named `Quality details` that includes:

- ratio quality;
- open-interest quality;
- zero count;
- positive count;
- zero rate;
- denominator-zero flag;
- provider/source metadata already displayed by G128.

### 2. Algorithm Results View

In `/marketops/algorithms`, when result payloads include `dataset=options_distribution_daily` and `feature=call_put_open_interest_ratio`:

- Show a quality badge in the result list/detail summary, not only inside raw JSON.
- Show `call_put_oi_ratio_quality` and `open_interest_quality` in the selected result detail.
- If quality is not `usable`, show a concise warning such as `Not eligible for G131 proposal gate`.
- Do not imply that every algorithm result should have a proposal; make the distinction explicit.

If the result is for another dataset/feature, do not show an options quality warning. Avoid generic false alarms.

### 3. Signal Proposal View

In the existing signal proposal list/detail UI:

- Show `quality_gate.policy` when present.
- Show `quality_gate.passed=true` as a compact passed-gate badge.
- Show the nested algorithm-result ratio quality when present.
- In proposal detail, include a small `Evidence quality` section with:
  - gate policy;
  - gate pass state;
  - ratio quality;
  - open-interest quality;
  - zero-rate/counts when present.

Because G131 skips non-usable options ratio results before proposal insert, do not expect blocked proposal rows. The UI should explain this only where it has enough context, for example in result detail for a non-usable result.

### 4. Empty And Mixed States

- If an asset has no options distribution rows, keep the existing empty state.
- If quality fields are missing on older rows, render `unknown` without breaking the chart/table.
- If an algorithm execution has 27 results but only 9 proposals, the UI may show this as `Proposal queue contains only quality-gated candidates` when the selected result/proposal context is options ratio evidence.
- Do not add a synthetic skipped-count metric unless it can be derived from data currently loaded in the UI. If deriving locally, label it as `loaded rows only`.

## Implementation Guidance

### Types

Extend existing frontend types with optional fields:

```ts
export type MarketOpsOptionsRatioQuality =
  | 'usable'
  | 'partial_zero'
  | 'all_zero'
  | 'denominator_zero'
  | 'unknown';

export interface MarketOpsOptionsQualityFields {
  open_interest_quality?: string;
  open_interest_zero_count?: number;
  open_interest_positive_count?: number;
  open_interest_zero_rate?: number;
  call_put_oi_denominator_is_zero?: boolean;
  call_put_oi_ratio_quality?: string;
}
```

Add parser/helpers near existing MarketOps/algorithm summarizers rather than scattering `as any` casts through route components.

Recommended helpers:

```ts
getOptionsQualityFields(value: unknown): MarketOpsOptionsQualityFields
normalizeRatioQuality(value: unknown): MarketOpsOptionsRatioQuality
formatZeroRate(value: unknown): string
isOptionsRatioAlgorithmPayload(value: unknown): boolean
```

### Components

Prefer small reusable presentation helpers:

- `OptionsQualityBadge`
- `OptionsQualitySummary`
- `AlgorithmResultQualitySummary`
- `ProposalQualityGateSummary`

Use existing app badge/table/detail styles. Do not introduce a new design language.

## Tests

Add or update frontend tests for:

- API/type parsing does not break when quality fields are missing.
- Asset options distribution summary renders `usable`, `partial_zero`, `all_zero`, and `denominator_zero` states.
- Algorithm result summarizer extracts quality fields from `result_payload`.
- Proposal summarizer extracts `quality_gate.policy` and nested ratio quality from `proposal_payload`.
- Non-options algorithm results do not show options-specific quality warnings.
- Build/typecheck remains green.

Recommended commands:

```bash
npm test -- --run
npm run build
```

## Acceptance Criteria

- `/marketops/assets` shows options ratio/open-interest quality for persisted distribution snapshots when fields are present.
- `/marketops/algorithms` shows quality metadata for options ratio algorithm results without requiring raw JSON inspection.
- Signal proposal detail shows G131 gate metadata for proposals generated after G131.
- Missing fields are handled as `unknown`; no crashes on older rows.
- No new backend routes, ingestion triggers, live preview triggers, or materialization policy changes are added.
- Frontend tests and build pass.

## Handoff Notes

Use G131 live validation as the reference scenario:

- execution request: `algexec_9b5c5859ecb0d78233495268`;
- result breakdown: `usable=9`, `all_zero=10`, `denominator_zero=6`, `partial_zero=2`;
- proposal count: 9;
- non-usable proposal count: 0.

The UI should help an analyst understand this outcome without suggesting that the skipped rows disappeared. They remain algorithm results; they are simply not eligible for review as signal proposals under `g131.options_distribution_quality.v1`.

# G141 Historical Coverage And Outcome Population

Status: implemented backend and live coverage validation; strict outcome population blocked by historical options quality

Date: 2026-07-20

## Purpose

G141 provides the bounded historical execution path needed to turn the G137-G140 research layers into a point-in-time calibration dataset. It expands canonical AAPL equity EOD coverage, computes trailing transition statistics without look-ahead, preflights options analytics coverage, and coordinates state, hypothesis, opportunity, and outcome calculation through one deterministic command.

G141 does not lower the quality policy merely to produce triggers. A run with enough equity history but insufficient required IV-surface coverage must stop before writes. Sparse open-interest, premium, or distribution history remains explicit and blocks only the hypotheses that require it.

## Implemented Scope

- Extended signalops-massive-puller with exact Top 50 symbol selection, inclusive date ranges, an explicit maximum observation-day count, and global provider/event budgets.
- Weekends are skipped while exchange-holiday failures remain in the report.
- Added point-in-time same-direction persistence, trailing z-score, and trailing empirical percentile to G137 transitions.
- Rarity statistics require 20 prior observations and use at most the most recent 60 prior observations.
- Added signalops-marketops-history-runner and a Docker target.
- The runner reads only persisted Massive normalized equity events and persisted option chain/distribution rows.
- It coordinates existing state, hypothesis, opportunity, and outcome engines in memory, then persists in dependency order with existing deterministic identities.
- Strict preflight requires at least 60 AAPL equity source sessions and 20 analytics-ready option sessions. Distribution coverage is reported as a hypothesis-specific warning rather than a global blocker.
- An analytics-ready option session has all five required 30/60/90-DTE ATM and 30-DTE 25-delta put/call cells with usable IV, delta, and underlying price.
- The insufficient-coverage override is restricted to dry runs.
- Maximum scope remains AAPL and 200 source sessions.
- The coordinator makes no provider calls and creates no production signals, proposals, alerts, insights, graph writes, trades, promotions, or policies.
- After G142, only chain rows from an analytics-ready capture's exact ingestion run are eligible for G141 coverage and state construction; pre-G142 and partial captures cannot satisfy the gate.

## Point-In-Time Rules

Transition persistence, z-score, and percentile use only earlier transitions for the same feature and canonical dimensions. The first 20 changes have null rarity statistics. State construction uses evidence from the source session or earlier. Outcome calculation uses persisted equity sessions no later than the explicit as-of date. Missing analytics remain missing and do not become zero.

## Live Validation

A bounded Massive acquisition ran for AAPL equity EOD only from 2026-01-02 through 2026-07-17:

- 141 weekday candidates;
- 147 provider requests;
- 135 events built and published;
- six exchange-holiday dates with no bar;
- 135 distinct normalized AAPL equity sessions persisted;
- no credential replacement and no direct database insertion.

The strict historical runner reported:

- 135 equity source sessions against the 60-session minimum;
- 2 option-chain sessions;
- 0 analytics-ready option sessions;
- 2 option-distribution sessions, reported as a positioning-hypothesis warning;
- status blocked_insufficient_coverage due to analytics-ready option coverage;
- zero ledger writes.

A dry-run diagnostic over the same window produced:

- 24 feature definitions;
- 3,375 feature observations;
- 135 states;
- 1,102 transitions;
- 134 evidence records;
- 540 hypothesis evaluations;
- 0 eligible evaluations;
- 0 triggered evaluations;
- 0 opportunities;
- 0 outcomes.

The dominant rejection reasons are missing or unusable historical IV-surface, premium, and bucketed OI evidence. This is a truthful source-data result, not an algorithm failure.

Synthetic deterministic integration tests provide five analytics-ready option sessions plus 31 equity sessions. That fixture triggers H004, builds an opportunity, produces matured outcomes, and verifies every persisted layer count. It proves the pipeline is functional without inserting synthetic records into live ledgers.

## Provider Boundary

The current Massive chain snapshot endpoint provides IV, Greeks, and open interest for the current snapshot. The reference-contract as_of parameter identifies historical contract metadata; it is not a historical IV/Greeks/open-interest snapshot. Historical aggregate bars and quotes can provide price/volume and bid/ask evidence, but do not by themselves reconstruct provider open interest or historical snapshot Greeks.

G141 does not relabel reference contract rows as analytics-ready history. The remaining valid paths are prospective daily snapshot retention or a separately approved historical analytics dataset with explicit point-in-time lineage.

Massive documentation:

- https://massive.com/docs/rest/options/snapshots/option-chain-snapshot
- https://massive.com/docs/rest/options/contracts/contract-overview
- https://massive.com/docs/rest/options/aggregates/custom-bars

## Safety Boundary

G141 does not add unbounded acquisition, direct normalized-ledger insertion, synthetic analytics, automatic scheduling, Top 50 fanout, partial writes after failed preflight, automatic calibration/promotion, or frontend changes.

## Acceptance Result

The G141 execution substrate is implemented and validated. The equity-history requirement is satisfied live. Strict outcome population remains intentionally blocked until at least 20 analytics-ready option sessions exist. Calibration summaries remain premature while the live triggered and matured sample is zero.

## Links

- Operations: ../operations/historical_research_pipeline.md
- G137 state materialization: G137_aapl_market_state_vertical_slice.md
- G138 hypotheses: G138_research_hypothesis_evaluator.md
- G139 opportunities: G139_opportunity_layer.md
- G140 outcomes: G140_forward_outcome_evaluation.md
- Architecture evaluation: ../architecture/market_state_intelligence_evaluation.md

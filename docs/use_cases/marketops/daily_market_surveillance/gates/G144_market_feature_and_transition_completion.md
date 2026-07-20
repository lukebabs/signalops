# G144 Market Feature And Transition Completion

Status: implemented backend feature, transition, and bounded cohort contract

Date: 2026-07-20

## Purpose

G144 completes the longitudinal inputs required to study the initial H001/H004/H006/H007 hypothesis pack without widening G143 provider acquisition or selected-evidence storage. It adds realized volatility, normalized option-cell changes, curve state, point-in-time earnings context, richer transition windows, and explicit bounded multi-symbol state execution.

The gate does not claim that live longitudinal options coverage now exists. New lookback observations remain explicitly missing until genuine eligible sessions accumulate.

## Implemented Scope

- Advances the canonical state shape to `marketops.market_state.v2`.
- Extends the feature catalog from 29 to 44 definitions and each state from 39 to 69 deterministic observation slots.
- Retains the original 39 G137/G143 hypothesis-critical slots as the required completeness denominator. The 30 new longitudinal/context slots are auditable and mature independently, so an early 60-session or event-calendar gap cannot inflate or distort required quality.
- Adds annualized log-return realized volatility for 10, 20, and 60 eligible equity sessions plus 5-session change in 20-session realized volatility.
- Adds 30-DTE ATM IV less 20-session realized volatility and the corresponding IV/RV ratio.
- Adds normalized 1- and 5-eligible-option-session IV changes for all seven G143 surface cells.
- Adds 1- and 5-eligible-option-session midpoint premium changes and 5-session open-interest changes for 30-DTE 25-delta put/call cells.
- Classifies usable 30/60/90-DTE ATM curves as `contango`, `backwardation`, `flat`, or `mixed`.
- Adds point-in-time `days_to_earnings`, `days_since_earnings`, and `earnings_window_state` observations from normalized `market_event_calendar` records.
- Adds absolute-difference transitions at 3, 5, 10, and 20 eligible state sessions while preserving the existing 1-session transition contract and point-in-time rarity/persistence statistics.
- Adds second-difference acceleration for realized volatility, IV, curve slope, premium, and normalized OI-change features.
- Persists classified term-structure regime changes without coercing text states into numeric values.
- Makes a known pre/day/post earnings window invalidate H001. Missing event context remains explicit but optional under the current `record_when_available` hypothesis policy.
- Generalizes the state builder to any explicit symbol and the CLI to comma-separated explicit cohorts.
- Enforces `--max-symbols` between 1 and 10, rejects cohorts larger than the declared cap, derives isolated per-symbol calculation run IDs, and reports per-symbol plus aggregate metrics.
- Preserves `--symbol` as a single-symbol compatibility alias.
- Reads the event calendar within a bounded historical/future window; no provider call occurs in state calculation.

## Point-In-Time And Quality Semantics

Realized volatility uses only current and prior eligible equity sessions. Option changes use the nth prior persisted option session, not calendar-day subtraction, and retain both selected-contract artifact references. Quotes must be positive, non-crossed, timestamped, and from their respective sessions for premium-change calculations. Missing IV, quotes, OI, prior sessions, or event knowledge never become zero.

An earnings event is eligible only when its `known_at` value, or the normalized event processing time when no explicit value is supplied, is at or before the materialized session end. A retrospectively learned event cannot enter an earlier state.

The state quality summary separately reports:

- required feature slots and usable/blocked required slots;
- total feature slots and usable total slots;
- quality counts across all 69 observations.

## Storage And Execution Boundary

G144 uses the existing generic feature, state, transition, and evidence ledgers. No migration is required. Existing G143 selected option rows remain the only contract-level analytical evidence; no full chain is persisted by this gate.

One invocation processes only the symbols explicitly supplied by the operator. It does not resolve a universe group, schedule work, continue hidden fanout, call Massive, synthesize history, or relax G141's genuine-session threshold.

## Validation

Focused tests prove:

- 44 definitions and 69 stable observation identities per state;
- required completeness remains based on the original 39 hypothesis-critical slots;
- 10/20/60-session realized volatility and 5-session realized-volatility change mature only with sufficient equity history;
- 1/5-session IV and premium changes and 5-session OI changes use eligible prior option sessions with exact artifact lineage;
- term-structure classification and regime-transition persistence;
- 3/5/10/20-session transitions and selected-feature acceleration;
- point-in-time event knowledge excludes later-learned earnings dates;
- known earnings windows invalidate H001 while unavailable context does not;
- explicit non-AAPL state builds;
- normalized/deduplicated cohort order, per-symbol run isolation, dry-run no-write behavior, and rejection above the hard symbol cap;
- G143 provider-shaped H001/H004/H006/H007 compatibility remains intact.

The focused state, hypothesis, and state-materializer CLI packages pass. Full repository validation is recorded in the build journal and gate audit.

## Non-Goals

G144 does not add an earnings-calendar ingestion provider, scheduler, Top 50 universe expansion, historical option synthesis, full-chain persistence, hypothesis backtesting/calibration, proposal generation, production signal materialization, opportunity disposition, graph mutation, Syncratic context, or frontend behavior.

Live research remains blocked until genuine analytics-ready option sessions accumulate. Event observations remain missing in environments that do not publish point-in-time `market_event_calendar` records.

## Next Gate

G145 implemented the hypothesis evaluation/outcome adapter, comparison and walk-forward modes, regime/event segmentation, sample warnings, and versioned calibration summaries. See `G145_hypothesis_backtest_and_calibration.md`.

G146 is now next: reviewed hypothesis proposal and opportunity governance without automatic promotion.

## Links

- Canonical architecture: `../../../../design/syncratic_marketops_market_state_intelligence_architecture_v1.md`
- Reconciliation: `../architecture/market_state_intelligence_evaluation.md`
- G143 selected evidence: `G143_options_surface_evidence_v1.md`
- State feature implementation: `../../../../../internal/marketops/state/g144_features.go`
- Transition implementation: `../../../../../internal/marketops/state/g144_transitions.go`
- Bounded CLI: `../../../../../cmd/marketops-state-materializer/main.go`

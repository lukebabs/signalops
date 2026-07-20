# G148 Graph, Ask, And Cohort Rollout

Status: proposed - backend implementation specification ready

Date: 2026-07-20

## Objective

Complete the remaining governed Market State Intelligence v1 integration in three ordered, independently fail-closed slices:

1. map persisted market-state intelligence records into the existing review-controlled graph-proposal workflow;
2. build bounded, evidence-pure Market State Syncratic contexts and operator-triggered Ask explanations;
3. stage explicit asset cohorts through existing state/evaluation/opportunity controls with durable per-symbol readiness reporting.

G148 is an integration and controlled-rollout gate. It is not permission for automatic provider fanout, scheduled Ask, automatic graph mutation, hypothesis promotion, threshold relaxation, or production hypothesis signal materialization.

## Current Truth

The deployed system already has strong foundations, but they are narrower than G148 requires:

- the graph proposal table and mapper require a DSM artifact and production signal;
- all current graph proposals are signal-artifact sourced;
- Syncratic context windows contain signal, alert, artifact, graph-proposal, and label references;
- the current Ask prompt fetches bounded signal details, not market state, transition, hypothesis, opportunity, outcome, or calibration records;
- Syncratic Ask is operator-triggered, server-side, prompt-capped, idempotent, and data-quality aware;
- the state materializer supports explicit cohorts capped at 10 symbols;
- hypothesis evaluation, opportunity building, outcome materialization, and hypothesis proposal generation remain AAPL-only;
- no operational endpoint returns per-symbol Market State rollout readiness.

Live reconciliation on 2026-07-20 found:

- 120 graph proposals: 7 accepted and 113 proposed, all using the legacy signal-artifact source model;
- 9 Syncratic context windows, all using `symbol_signal_cluster_5d`;
- 6 AAPL market states with average completeness `0.140`, all `partial` or `missing`;
- 24 AAPL hypothesis evaluations, with 0 eligible and 0 triggered;
- no persisted Market State opportunities;
- no non-AAPL market states or hypothesis evaluations.

These facts are acceptance fixtures. G148 must expose them truthfully and must not manufacture broader readiness.

## Required Sequence

G148 MUST be implemented and validated in this order:

1. **G148-A: source-aware market-intelligence graph proposals**
2. **G148-B: Market State context and Ask v2**
3. **G148-C: explicit cohort execution and readiness**

Each slice must keep its own tests and live validation. G148-C must not be described as a broad rollout until G148-A and G148-B contracts are stable and the cohort dry-run proves quality blocking and partial-failure behavior.

## G148-A: Source-Aware Market-Intelligence Graph Proposals

### Storage Evolution

Generalize the existing `marketops_dsm_graph_proposals` review ledger instead of creating an unrelated graph decision system.

Migration `000035` should:

- add `proposal_source`, defaulting existing rows to `dsm_signal`;
- add `source_record_type`, `source_record_id`, and `source_record_version`;
- add bounded `source_refs` and `lineage_refs` JSON objects;
- allow legacy `artifact_id`, `signal_id`, `signal_type`, and `detector_id` to be absent for non-signal sources;
- allow legacy severity/confidence to be nullable rather than inventing values for state records;
- retain all existing proposal IDs, decisions, foreign keys where values are present, and decision audit fields;
- enforce that `dsm_signal` rows retain their existing signal/artifact identity;
- enforce that non-signal rows have a source record type and ID;
- add source/type/status and source-record indexes;
- provide a reversible down migration that refuses lossy rollback when non-signal rows exist, or explicitly documents the required cleanup precondition.

Allowed sources:

- `dsm_signal`
- `market_state`
- `state_transition`
- `hypothesis_definition`
- `hypothesis_evaluation`
- `opportunity`
- `outcome`

### Canonical API

Add a source-aware canonical API while preserving the existing DSM routes as compatible legacy aliases:

```text
GET  /v1/marketops/graph-proposals
GET  /v1/marketops/graph-proposals/{proposal_id}
POST /v1/marketops/graph-proposals/{proposal_id}/decision
```

List filters:

- tenant/app/domain/use-case;
- proposal source;
- source record type and ID;
- subject symbol;
- candidate type;
- status;
- bounded limit.

The legacy `/v1/marketops/dsm/graph-proposals` list must implicitly filter `proposal_source=dsm_signal`. Existing clients and tests must remain green.

### Deterministic Mapper

Add an explicit operator CLI:

```text
signalops-marketops-intelligence-graph-mapper
```

Required flags:

- `--tenant-id`
- `--symbol`
- `--session-start`
- `--session-end`
- `--source-types`
- `--max-source-records`
- `--max-proposals`
- `--dry-run`
- an explicit write acknowledgement consistent with repository CLI conventions.

The mapper must:

- read only persisted records;
- emit deterministic proposal IDs from source type, source ID/version, and candidate identity;
- be idempotent;
- stop at explicit record and proposal caps;
- report source counts, candidate counts, skipped reasons, quality counts, and proposal IDs;
- never accept proposals or write graph state.

Initial mappings:

- asset node;
- market-state node and `STATE_OF` relationship to the asset;
- material state-transition node and relationship to its current state;
- exact hypothesis-definition node;
- hypothesis-evaluation node with `EVALUATES_STATE` and `INSTANCE_OF` relationships;
- opportunity node with relationships to contributing evaluations;
- matured outcome node with `OUTCOME_OF` relationship to its exact source.

Do not create graph nodes for every feature observation or evidence row. Their IDs belong in bounded lineage properties unless a later reviewed mapping requires first-class nodes.

Candidate properties must contain only stable analytical identity, dates, schema/version, direction, lifecycle/quality state, and lineage IDs. Do not copy full provider payloads, prompts, raw evaluation payloads, or generated narratives into graph proposals.

### Graph Acceptance

G148-A passes only when:

- legacy DSM graph proposal behavior and decisions remain unchanged;
- non-signal candidates require no fabricated signal or artifact;
- repeated dry-run/write runs produce stable identities and no duplicates;
- accepted/rejected decisions preserve the existing actor-attributed decision audit fields;
- no graph database mutation or automatic acceptance exists;
- an AAPL bounded smoke produces inspectable candidates or a truthful zero-candidate result with reasons.

## G148-B: Market State Context And Ask v2

### Context Storage

Migration `000036` should extend `syncratic_context_windows` additively with:

- `context_payload_version`;
- `market_state_ids`;
- `state_transition_ids`;
- `marketops_evidence_ids`;
- `hypothesis_evaluation_ids`;
- `opportunity_ids`;
- `outcome_ids`;
- `calibration_summary_ids`;
- bounded `quality_warnings` JSON;
- bounded `lineage_refs` JSON.

Existing v1 signal contexts remain valid and readable.

### Context Strategy

Add `market_state_session_v1`.

Creating this strategy requires one persisted `market_state_id`. The server derives symbol, asset, session, schema version, and time bounds from that state rather than trusting duplicated client fields.

The deterministic bundle may include:

- selected state summary and completeness/quality;
- the seven persisted surface cells and explicit missingness;
- material transitions for the same current state;
- exact-version hypothesis evaluations for the state;
- linked evidence;
- compatible opportunities;
- matured or pending outcomes linked to included evaluations/opportunities;
- one exact-version calibration summary per included hypothesis version;
- source lineage identifiers;
- quality and truncation warnings.

Hard default caps:

- 1 market state;
- 25 feature summaries, prioritizing the canonical surface and required features;
- 50 material transitions;
- 8 hypothesis evaluations;
- 20 evidence rows;
- 10 opportunities;
- 20 outcomes;
- 8 exact-version calibration summaries;
- 12,000 serialized prompt bytes.

The stored context must record returned/available counts and every truncation decision.

### Evidence Purity

The builder must reject or quality-block:

- different tenant, asset, or symbol;
- records from a different market state or session unless explicitly marked as prior-session comparison;
- incompatible state schema versions;
- hypothesis evaluation versions that do not match their definitions or calibration report;
- opportunities whose contributions are outside the selected context;
- outcomes whose source record is absent;
- unresolved lineage IDs;
- evidence containing a conflicting known ticker.

A data-quality-blocked context may explain the quality problem, but Ask must not produce market-direction inference from it.

### Prompt Contract

Add `marketops.syncratic.ask_prompt.v2` while retaining v1 for legacy contexts.

The v2 prompt must label each claim category:

- observed fact;
- calculated feature;
- statistical rarity;
- hypothesis inference;
- historical association;
- governance state;
- unknown future outcome.

The generated response must:

- cite persisted record IDs from the supplied context;
- distinguish missing/blocked evidence from neutral or zero;
- keep calibration sample warnings visible;
- state that historical association is not a guaranteed future outcome;
- avoid trade, order, or portfolio instructions;
- avoid claiming lifecycle promotion, proposal acceptance, or production materialization.

Continue to use server-side Syncratic Ask with direct reasoning, graph/search/KEE disabled, one bounded external-context item, sanitized errors, prompt/evidence digests, and unchanged rerun skipping. Ask remains an explicit operator action.

### API And Frontend Boundary

Extend the existing context-window create route to accept `market_state_session_v1` plus `market_state_id`. Do not let the browser assemble trusted context JSON.

The Market State frontend follow-up may add:

- `Build bounded Ask context`;
- navigation to the existing Syncratic detail;
- Ask state badges and citations.

It must not call the external Syncratic facade directly, trigger Ask automatically, or combine deterministic state with generated prose without labels.

Write the frontend-agent specification only after the backend response shapes and live context fixture are validated.

### Ask Acceptance

G148-B passes only when:

- legacy v1 contexts and Ask remain compatible;
- context ID and evidence digest are deterministic;
- wrong-symbol/session/version fixtures are rejected or quality-blocked;
- exact-version calibration and warnings survive prompt construction;
- prompt caps and truncation metadata are tested;
- unchanged Ask reruns do not call the upstream client;
- Ask failure cannot corrupt deterministic context or insight rows;
- one bounded AAPL state-context smoke is validated without graph writes or lifecycle mutation.

## G148-C: Explicit Cohort Execution And Readiness

### Execution Boundary

Generalize the existing AAPL-only analytical CLIs through shared internal services and one explicit cohort runner. Do not orchestrate provider acquisition.

Recommended command:

```text
signalops-marketops-intelligence-cohort-runner
```

Required controls:

- `--tenant-id`
- either explicit `--symbols` or `--universe-group`;
- `--max-symbols`, hard maximum 10 for G148;
- inclusive session start/end;
- explicit stages;
- `--continue-on-error`;
- `--dry-run`;
- explicit write acknowledgement;
- run ID and structured JSON summary.

Allowed stages:

- preflight;
- state materialization;
- hypothesis evaluation;
- opportunity build;
- outcome materialization;
- hypothesis proposal generation.

Graph mapping and Syncratic Ask must not run automatically as cohort stages. They remain separately reviewed/operator-triggered actions.

The runner must use persisted equity/options/event inputs only. It must not call Massive, create schedules, relax thresholds, synthesize history, promote hypotheses, accept proposals, or materialize unsupported hypothesis signals.

### Durable Run Records

Migration `000037` should add:

- `marketops_intelligence_cohort_runs`;
- `marketops_intelligence_cohort_symbol_results`.

Run records capture scope, requested/resolved symbols, stage list, caps, dry-run/write mode, status, aggregate metrics, errors, actor, and timestamps.

Symbol results capture:

- per-stage status and error;
- input coverage counts;
- latest state date/schema/quality/completeness;
- required/surface coverage;
- evaluation/eligible/triggered counts and rejection reasons;
- opportunity counts;
- pending/matured outcome counts;
- proposal counts by governance status;
- exact-version calibration availability and minimum-sample state;
- readiness dimensions and reasons.

Partial symbol failure must not erase successful symbol results. Reruns with the same run ID and scope must be rejected or idempotent; they must never duplicate deterministic analytical rows.

### Readiness Semantics

Do not collapse all readiness into one production-looking boolean.

Return independent dimensions:

- `coverage_state`: unavailable, incomplete, usable;
- `evaluation_state`: not_run, blocked, evaluated_no_trigger, triggered;
- `governance_state`: research_only, candidate, approved, proposal_pending, reviewed;
- `calibration_state`: unavailable, below_minimum, available;
- `outcome_state`: unavailable, pending, matured.

A derived rollout status may be:

- `not_observed`;
- `inspection_ready`;
- `research_evaluation_ready`;
- `review_ready`;
- `blocked`.

G148 must not return `production_ready`. Hypothesis materialization remains unsupported and fail-closed.

### Read API

Add bounded read endpoints:

```text
GET /v1/marketops/intelligence/cohort-runs
GET /v1/marketops/intelligence/cohort-runs/{run_id}
GET /v1/marketops/intelligence/readiness
```

Readiness filters:

- tenant;
- universe group;
- explicit symbols;
- latest session date;
- rollout status;
- bounded limit.

The readiness response must be aggregate-first and include per-symbol rows in one request so the frontend never performs Top 50 N+1 state/evaluation/calibration reads.

### Operational Frontend

After backend validation, extend the existing MarketOps Assets experience with a read-only `Intelligence readiness` view:

- symbol and universe membership;
- latest state date, quality, schema, completeness, and surface coverage;
- evaluation, governance, calibration, and outcome dimensions;
- explicit reasons;
- last cohort run and per-stage status;
- link to Market State when a state exists.

Do not put provider acquisition, cohort execution, proposal review, graph decisions, Ask execution, or hypothesis promotion on this dashboard.

### Cohort Acceptance

G148-C passes only when:

- explicit and universe-resolved cohorts enforce the 10-symbol cap;
- dry-run performs no writes;
- a failing symbol does not invalidate successful symbols;
- all per-symbol counts/reasons are persisted and reproducible;
- evaluator, opportunity, outcome, and proposal logic no longer contains an AAPL-only validation when called through the bounded cohort path;
- lifecycle, quality, and exact-version checks remain unchanged;
- readiness is aggregate-first and never manufactures production readiness;
- a live bounded dry-run includes AAPL plus at least one symbol with no state and reports both truthfully;
- no provider request, scheduler, automatic Ask, automatic graph decision, or direct signal write occurs.

## Security, Privacy, And Governance

- Use existing bearer authentication and actor derivation.
- Never accept actor identity from the request body.
- Never place credentials, tokens, raw prompts, provider payloads, or privacy reveal output in graph/context/cohort records.
- Keep Syncratic calls server-side.
- Bound all lists, payloads, prompts, cohorts, sessions, and candidate counts.
- Treat generated explanation as generated synthesis, never deterministic evidence.
- Treat graph acceptance, proposal review, analyst disposition, and hypothesis lifecycle as separate decisions.
- Preserve research-only and fail-closed defaults.

## Tests And Validation

Required automated coverage:

- migrations apply, down, and re-apply in isolation;
- legacy graph proposal compatibility;
- generic graph source validation and stable IDs;
- graph mapper idempotency, caps, and zero-result paths;
- legacy Syncratic v1 compatibility;
- state-context exact identity and evidence purity;
- prompt category, cap, digest, skip, and failure behavior;
- cohort cap, universe resolution, dry-run, partial failure, stage isolation, and rerun rules;
- aggregate readiness semantics and no-N+1 response;
- AAPL-only guard removal only through bounded explicit scope;
- full Go, Python, schema, and frontend regression suites where touched.

Required live validation:

1. inspect pre-migration graph/context counts;
2. apply migrations and rebuild the gateway/CLIs;
3. run one bounded graph dry-run and one idempotent write smoke if candidates are valid;
4. create one AAPL `market_state_session_v1` context;
5. validate prompt metadata without forcing Ask unless credentials/budget are explicitly approved;
6. run a 2-3 symbol cohort dry-run containing AAPL and at least one missing-data symbol;
7. prove no provider calls, automatic Ask, graph decisions, signal writes, or lifecycle changes occurred;
8. verify gateway health and read endpoints;
9. record empirical limitations separately from structural effectiveness.

## Implementation Touch Points

Expected backend files include:

- migrations `000035` through `000037`;
- `internal/storage/storage.go` and MarketOps-specific storage interfaces;
- PostgreSQL graph, Syncratic context, and cohort repositories;
- `internal/api/router.go`, graph proposal APIs, Syncratic APIs, and cohort APIs;
- a new `internal/marketops/graph` deterministic mapper;
- shared internal state/evaluation/opportunity/outcome/proposal services extracted from CLI-only code where needed;
- new graph-mapper and cohort-runner commands;
- focused repository/API/CLI tests;
- Dockerfile/Compose targets for explicit operator execution;
- API and operations documentation.

Expected frontend work comes after backend contracts are validated:

- source-aware graph proposal review updates;
- Market State to Syncratic context handoff;
- read-only cohort readiness on Assets;
- type/client/query/helper tests and deployed responsive validation.

## Explicitly Out Of Scope

- automatic or scheduled Top 50 provider collection;
- cohorts larger than 10 in G148;
- full-chain persistence expansion;
- synthetic historical options or event data;
- threshold relaxation to create triggers;
- automatic hypothesis promotion;
- automatic proposal acceptance or graph mutation;
- graph-database writeback;
- automatic or batch Syncratic Ask;
- Syncratic Search, KEE, or external corpus ingestion;
- direct hypothesis signal materialization;
- trade, order, portfolio, or execution controls;
- claiming empirical effectiveness from sparse AAPL fixtures.

## Gate Result

This specification is ready for a backend agent to implement G148-A first. G148-B and G148-C remain ordered follow-on slices inside the gate, not parallel workstreams. A frontend-agent specification is intentionally deferred until the new source-aware graph, state-context, and aggregate readiness API contracts are implemented and live-validated.

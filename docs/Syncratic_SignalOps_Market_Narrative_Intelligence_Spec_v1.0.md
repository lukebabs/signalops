# Syncratic SignalOps --- Market Narrative Intelligence Specification

**Version:** 1.0\
**Status:** Architecture / Implementation Specification\
**Platform:** Syncratic SignalOps\
**Subsystem**
**Date:** September 2026

## 1. Vision

SignalOps Market Narrative Intelligence transforms heterogeneous market
information into **time-indexed, evidence-backed narratives** consumable
by Syncratic analytics, valuation, risk, and decision-support
algorithms.

The objective is not merely to collect financial news, transcribe
television, or calculate sentiment. It is to determine:

> **What narratives are forming around market events, how rapidly are
> they changing, what evidence supports them, and does observable market
> behavior confirm or contradict them?**

The architecture separates three information classes:

1.  **Market observations --- what happened.**
2.  **Reported information --- what is being said about what happened.**
3.  **Narrative intelligence --- the explanations, hypotheses, and
    consensus forming around what happened.**

SignalOps must never treat a repeated media explanation as ground truth.
Narratives are evidence-backed hypotheses whose strength evolves as new
information and market telemetry arrive.

The resulting **Market Narrative Bus** is a machine-consumable stream of
evolving narratives, catalysts, entities, relationships, evidence,
confidence, velocity, novelty, disagreement, and market confirmation.

## 2. Strategic Outcomes

The subsystem should enable SignalOps to answer:

-   What is moving the market?
-   What explanations are currently dominant?
-   Which narratives are newly emerging, accelerating, fading, or
    resurfacing?
-   Are independent sources converging on the same explanation?
-   Does observable market behavior support the prevailing narrative?
-   Where does market behavior contradict media consensus?
-   Which assets, sectors, factors, and macro variables are associated
    with a narrative?
-   What information was available when a narrative first formed?
-   How did the explanation for a market event evolve?
-   Which narratives historically accompanied similar regimes?

The intended progression is:

**event detection → reporting → narrative formation → evidence
validation → market confirmation → insight generation**

## 3. Core Architectural Principle

The ingestion mechanism is not the product.

Scheduled CNBC/Bloomberg capture and local speech-to-text remains a
useful future sensor because broadcast media captures interviews,
analyst interpretation, and real-time conversational context. However,
transcription is not the differentiated capability.

The core intellectual asset is the **narrative inference, temporal
analysis, provenance, and evidence-validation layer**.

Another component of sourcing narrative is existing Financial
Modeling Prep (FMP) access and expands through RSS/Atom, primary
government sources, filings, company communications, earnings
transcripts, and licensed feeds.

Broadcast transcriptions will be from another subsystem (streamer-recorder) that will include a transcriber.

## 4. High-Level Architecture

``` text
FMP / RSS / SEC / Fed / Treasury / BLS / BEA / Company IR
Earnings Transcripts / Licensed News / Optional TV-ASR
                         │
                         ▼
                SignalOps Ingestion
                 normalize/dedupe
                         │
                         ▼
                 MARKET EVENT BUS
                         │
        ┌────────────────┼────────────────┐
        ▼                ▼                ▼
   Market Events     News Events      Macro Events
        └────────────────┼────────────────┘
                         ▼
               MARKET NARRATIVE BUS
                         │
              cluster/correlate/infer
                         ▼
                NARRATIVE ARTIFACTS
                         │
          Entities / Narratives / Catalysts
                         │
                         ▼
                  Narrative Graph
                         │
                ┌────────┴────────┐
                ▼                 ▼
           Market Regime       DOSM / VC
                └────────┬────────┘
                         ▼
                 Insight Candidates
```

## 5. Event Bus vs. Narrative Bus

The **Market Event Bus** represents observable or reported occurrences:
price changes, yield moves, economic releases, guidance, news
publications, primary-source releases, or attributed analyst claims. It
answers: **What happened or what was reported?**

The **Market Narrative Bus** represents evolving explanatory hypotheses
constructed from clusters of events and evidence: rates-driven growth
repricing, delayed Fed easing, AI spending concerns, oil-driven
inflation concerns, and similar explanatory structures. It answers:
**What explanation is forming around the observed events?**

These buses MUST remain logically distinct.

## 6. Source Strategy

**Phase 1 --- FMP anchor:** Use existing FMP access for available stock
news, general market news, company press releases, company-specific
news, and relevant market observations. Normalize everything before
narrative processing.

**Phase 2 --- RSS/Atom:** Add Federal Reserve, Treasury, SEC, BLS, BEA,
company investor relations, exchanges, central banks, economic agencies,
financial publications, and industry publications.

**Phase 3 --- Specialized sources:** Add SEC filings, earnings
transcripts, company presentations, central-bank speeches, economic
releases, licensed premium feeds, and sector-specific sources.

**Phase 4 --- Broadcast sensor:** Add scheduled transcribed context from stream-recorder:

``` text
Broadcast → Scheduled Capture → Audio Extraction → Local ASR
          → Timestamped Transcript Events → Entity/Claim Extraction
          → Market Event Bus
```

Broadcast becomes another sensor; downstream logic remains
source-agnostic.

## 7. Canonical Market Information Event

``` json
{
  "event_id": "uuid",
  "event_type": "market_news",
  "source": "fmp",
  "source_type": "news_api",
  "published_at": "2026-09-18T09:35:12-04:00",
  "ingested_at": "2026-09-18T09:35:16-04:00",
  "headline": "...",
  "body": "...",
  "source_url": "...",
  "entities": ["NVDA", "QQQ"],
  "topics": ["semiconductors", "rates"],
  "content_hash": "...",
  "language": "en",
  "provenance": {
    "provider": "fmp",
    "provider_event_id": "...",
    "retrieved_at": "..."
  }
}
```

Each adapter should implement acquisition, timestamp normalization,
attribution, hashing, duplicate and near-duplicate detection, entity
resolution, topic classification, claim extraction, provenance
preservation, and event publication.

**No narrative inference should occur inside a connector. Connectors
collect and normalize evidence.**

## 8. Temporal Narrative Formation

Narratives should normally be inferred from **clusters**, not isolated
articles.

Recommended rolling windows:

-   5 minutes --- emerging/high-frequency narrative
-   15 minutes --- short-term narrative formation
-   30 minutes --- intraday confirmation
-   60 minutes --- persistent intraday narrative
-   session/multi-hour --- regime context
-   multi-day --- persistent themes

Example:

``` text
09:31  NVDA falls 2%
09:33  Semiconductor stocks under pressure
09:35  Treasury yields move higher
09:37  Investors reassess Fed expectations
09:39  Technology shares lead market lower
09:42  Rate-sensitive growth stocks decline
09:46  10Y yield reaches session high
09:48  Nasdaq extends losses
```

The cluster may support a narrative that no individual event proves
independently.

## 9. Narrative Artifact

``` json
{
  "narrative_id": "rates_growth_repricing_20260918",
  "version": 4,
  "narrative": "Higher Treasury yields are associated with a repricing of rate-sensitive growth and semiconductor equities.",
  "entities": ["UST10Y", "NASDAQ", "QQQ", "NVDA", "AMD", "AVGO"],
  "relationships": [{
    "subject": "long_term_treasury_yields",
    "predicate": "associated_with",
    "object": "growth_valuation_pressure"
  }],
  "metrics": {
    "confidence": 0.82,
    "source_consensus": 0.74,
    "narrative_velocity": 0.91,
    "narrative_acceleration": 0.67,
    "novelty": 0.37,
    "market_confirmation": 0.88
  },
  "evidence": ["event-id-1", "event-id-2"]
}
```

## 10. Narrative Metrics

-   **Confidence:** composite evidentiary strength based on independent
    sources, source quality, primary evidence, semantic consistency,
    temporal coherence, market confirmation, and contradictions.
-   **Source consensus:** convergence across genuinely independent
    sources. Syndicated copies MUST NOT count as independent
    confirmation.
-   **Narrative intensity:** current prominence of a narrative.
-   **Narrative velocity:** `NV(n) = ΔI(n) / Δt`; rate of change in
    intensity.
-   **Narrative acceleration:** `NA(n) = ΔNV(n) / Δt`; identifies
    narratives moving rapidly toward dominance.
-   **Novelty:** distinguishes genuinely new narratives from recurring
    ones.
-   **Persistence:** duration of material narrative activity.
-   **Market confirmation:** consistency with observable market
    behavior.
-   **Contradiction:** material evidence opposing the narrative.
-   **Narrative divergence:** disagreement between information consensus
    and observable market behavior.

## 11. Narrative Lifecycle

``` text
EMERGING → FORMING → ESTABLISHED
                         ├─→ ACCELERATING
                         ├─→ CONTESTED
                         ├─→ DIVERGING
                         ▼
                       FADING → DORMANT → RESURGING
```

Narrative memory should distinguish new formation, continuation,
acceleration, recurrence, mutation, contradiction, and collapse.

## 12. Narrative Memory and Graph

SignalOps should retain durable narrative identity. Titans or equivalent
stateful memory can maintain narrative context while the Knowledge Graph
maintains explicit entities, relationships, temporal structure, and
evidence.

``` text
Narrative: US_RECESSION_CONCERN

First observed: 2026-03-11
Last observed:  2026-09-18

Mar ████████
Apr █████
May ██
Jun █
Jul ██
Aug ███
Sep ███████
```

Narratives should also exist as graph objects:

``` text
Higher Inflation
       │
       ▼
Fed Easing Expectations
       │ delayed/repriced
       ▼
Treasury Yields
       │
       ▼
Growth Discount Rates
       │
   ┌───┴──────────┐
   ▼              ▼
NASDAQ       Semiconductors
                  │
             NVDA/AMD/AVGO
```

Edges must distinguish observed relationships, reported claims, inferred
associations, proposed causal hypotheses, and verified factual
relationships.

## 13. Claims Are Not Facts

This is a foundational Knowledge Assurance requirement.

If an analyst says higher yields caused an NVDA decline, store an
attributed **CLAIM** with subject, relation, object, source, timestamp,
and evidence. Do not silently promote it to a fact.

SignalOps must preserve distinctions among:

-   observation
-   source assertion
-   consensus assertion
-   statistical association
-   inferred hypothesis
-   verified fact

## 14. Market Confirmation

Narrative claims should be tested against telemetry:

``` text
             "rates repricing"
                    │
             SignalOps hypothesis
                    │
       ┌────────────┼────────────┐
       ▼            ▼            ▼
    UST10Y ↑      QQQ ↓        IWM ?
       ✓            ✓            ✗
       └────────────┼────────────┘
                    ▼
          Evidence consistency
                   0.78
```

Confirmation may consider direction, magnitude, timing, cross-asset
relationships, sectors, factors, volatility, rates, commodities,
currencies, and options activity.

## 15. Narrative Divergence

A high-value SignalOps output occurs when information consensus and
market behavior disagree.

``` text
MEDIA CONSENSUS
"AI spending slowdown" → HIGH

MARKET BEHAVIOR
NVDA +2.1%
AVGO +1.7%
AMD  +2.8%
SOXX +1.9%

SIGNALOPS
NARRATIVE DIVERGENCE DETECTED
Market confirmation: LOW
Narrative confidence: 0.31
```

SignalOps should not invent the reason for disagreement. It should emit
an Insight Candidate such as:

> Observable market behavior is inconsistent with the currently dominant
> AI-spending-slowdown narrative.

## 16. Optional Broadcast Windows

If licensed broadcast feeds are introduced, targeted capture is
preferable to indiscriminate 24/7 recording:

``` text
07:00–09:30   Premarket narrative formation
09:30–11:00   Price discovery / narrative reaction
13:30–14:30   Afternoon positioning
15:00–16:15   Closing narrative
16:15–17:00   Post-close / earnings reaction
```

Special triggers may include FOMC, CPI, PPI, payrolls, GDP, Treasury
auctions, major earnings, central-bank speeches, and geopolitical
shocks.

Local ASR preserves the sovereign-deployment model. Existing RTX
5090-class infrastructure can support this optional workload subject to
scheduling with other GPU services.

## 17. Provenance: No Narrative Without Evidence

Every Narrative Artifact MUST retain its evidence chain:

``` text
Narrative Artifact
       │
       ├── FMP news event
       ├── primary-source release
       ├── independent news event
       ├── UST10Y observation
       ├── QQQ observation
       └── NVDA observation
```

Core rule:

> **No narrative without evidence.**

An LLM may formulate, summarize, cluster, or propose a narrative. It
must not manufacture the evidence supporting it. The evidence graph
determines whether a narrative survives, strengthens, becomes contested,
diverges, or collapses.

## 18. Historical Narrative Replay

Persisted events, narrative versions, evidence, and lifecycle states
should allow reconstruction of the system's knowledge **as of a
historical timestamp**, avoiding hindsight contamination.

Example query:

> What narrative was forming during the first 90 minutes of the selloff,
> what evidence existed at that moment, and when did the dominant
> explanation change?

This provides the basis for backtesting narrative signals against
subsequent market behavior.

## 19. DOSM / VC Integration

Narrative Intelligence becomes another evidence channel rather than a
replacement for quantitative models.

Potential features:

-   narrative regime
-   intensity
-   velocity
-   acceleration
-   novelty
-   persistence
-   market confirmation
-   divergence
-   entity exposure
-   catalyst type
-   source consensus
-   contradiction level

``` text
Market Telemetry ───────┐
Fundamentals ───────────┤
Valuation ──────────────┤
Options / Volatility ───┤
Narrative Intelligence ─┤
                        ▼
                   DOSM / VC
                        │
                        ▼
                Insight Candidate
```

Narrative metrics should initially be explanatory/contextual features
rather than automatic trading instructions. Predictive value should be
established empirically through replay and backtesting.

## 20. SignalOps Compatibility

The subsystem maps directly to the established SignalOps pattern:

``` text
Source Observation
      ↓
Event Artifact
      ↓
Normalization / Classification
      ↓
Correlation
      ↓
Narrative Artifact
      ↓
KG Update Proposal
      ↓
Evidence / Confirmation
      ↓
Insight Candidate
      ↓
DOSM / VC / downstream consumer
```

It preserves SignalOps design goals: API-first, event-driven,
asynchronous, idempotent, replayable, multi-tenant, observable,
horizontally scalable, model-agnostic, and suitable for
sovereign/customer-managed deployment.

## 21. Recommended Implementation Sequence

### P0 --- Narrative Foundation

-   FMP connector
-   canonical `MarketInformationEvent`
-   content hashing and deduplication
-   entity/ticker resolution
-   event persistence
-   Market Event Bus
-   provenance model

### P1 --- Narrative Formation

-   rolling temporal windows
-   semantic clustering
-   topic and claim extraction
-   `NarrativeArtifact`
-   confidence and source-consensus metrics
-   Market Narrative Bus

### P2 --- Market Validation

-   price/rate/index telemetry correlation
-   market-confirmation score
-   contradiction detection
-   narrative divergence
-   Insight Candidate generation

### P3 --- Narrative Dynamics

-   intensity
-   velocity
-   acceleration
-   novelty
-   persistence
-   lifecycle state
-   narrative memory

### P4 --- Knowledge Integration

-   Narrative Graph
-   KG update proposals
-   Titans/stateful narrative memory
-   historical replay
-   narrative query APIs

### P5 --- Source Expansion

-   authoritative RSS/Atom
-   filings and earnings
-   economic releases
-   premium/licensed feeds
-   optional broadcast ASR

### P6 --- Quantitative Integration

-   DOSM/VC feature integration
-   historical backtesting
-   narrative-regime analysis
-   divergence studies
-   feature-ablation testing
-   predictive-value qualification

## 22. Non-Goals for Initial Release

The initial implementation should NOT attempt to:

-   produce autonomous trading instructions;
-   assume media consensus is causally correct;
-   infer causality solely from textual co-occurrence;
-   ingest every available source;
-   require 24/7 television recording;
-   make an LLM the system of record;
-   discard raw evidence after summarization;
-   collapse claims and facts into one representation;
-   optimize predictive models before sufficient replay history exists.

## 23. Success Criteria

The first production-worthy release should demonstrate that SignalOps
can:

1.  ingest FMP news continuously and reliably;
2.  normalize and deduplicate events;
3.  resolve major market entities;
4.  cluster related events temporally and semantically;
5.  generate versioned Narrative Artifacts;
6.  retain source-level provenance;
7.  calculate narrative confidence and consensus;
8.  correlate narratives with market telemetry;
9.  detect material narrative/market divergence;
10. reconstruct the evidence available at a prior point in time;
11. expose narrative events to existing SignalOps consumers;
12. generate evidence-backed Insight Candidates without presenting
    unsupported causal claims as facts.

## 24. Longer-Term Vision

The Market Narrative Bus should evolve into a generalized **Narrative
Intelligence capability** within SignalOps.

The pattern is not limited to equities. The same architecture can be
applied to:

-   cybersecurity operations;
-   geopolitical intelligence;
-   supply-chain intelligence;
-   corporate/competitive intelligence;
-   defense and ambient intelligence;
-   operational telemetry;
-   risk and crisis management.

The generalized problem becomes:

> **Detect not only events, but the explanations, competing hypotheses,
> consensus formation, contradictions, and narrative shifts surrounding
> those events.**

For financial markets specifically:

> **Price tells SignalOps what happened. Narrative Intelligence
> characterizes the explanations forming around why participants believe
> it happened. Evidence and market telemetry determine how much
> confidence those explanations deserve.**

This is materially different from conventional sentiment analysis.

## 25. Desired Product Outcome

The finished subsystem should make it possible for SignalOps to produce
outputs such as:

``` text
MARKET NARRATIVE UPDATE — 10:42 ET

Dominant narrative:
Higher-than-expected inflation is driving a repricing
of near-term Fed easing expectations and increasing
pressure on rate-sensitive growth equities.

Status: ACCELERATING
Confidence: 0.84
Source consensus: 0.79
Market confirmation: 0.88
Novelty: 0.42
Narrative velocity: HIGH

Supporting observations:
- Treasury yields rising
- QQQ declining
- semiconductors underperforming
- multiple independent reports referencing delayed easing

Contradictory evidence:
- small-cap response remains limited

Knowledge status:
Evidence-backed hypothesis; causal relationship not established.

SignalOps action:
Publish Narrative Artifact
Propose KG updates
Generate Insight Candidate
Expose features to DOSM / VC
```

The strategic outcome is a **provenance-preserving market reasoning
layer** that converts fragmented information into structured, temporal,
testable narratives while maintaining the epistemic discipline required
by Syncratic Knowledge Assurance.

------------------------------------------------------------------------

## 26. Guiding Principles

1.  **No narrative without evidence.**
2.  **Claims are not facts.**
3.  **Correlation is not causation.**
4.  **Source repetition is not independent consensus.**
5.  **Narratives evolve and must be versioned.**
6.  **Market behavior can confirm or contradict a narrative.**
7.  **Historical replay must avoid hindsight contamination.**
8.  **The LLM is an inference component, not the system of record.**
9.  **All sources normalize into common SignalOps artifacts.**
10. **The Market Narrative Bus is the product; individual feeds are
    sensors.**
11. **Narrative Intelligence augments quantitative models rather than
    replacing them.**
12. **The architecture should remain model-agnostic and
    sovereign-deployment compatible.**

------------------------------------------------------------------------

**End of Specification**


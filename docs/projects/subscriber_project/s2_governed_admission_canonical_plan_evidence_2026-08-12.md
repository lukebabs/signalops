# S2 Governed Admission and Canonical Hot-Set Evidence

Date: 2026-08-12 UTC
Environment: live SignalOps PostgreSQL
Admission correlation: s2-initial-massive-admission-2026-08-12
Canonical plan: subeodplan_cd72633656bd183310c4a618
Planner correlation: s2-canonicalized-shadow-plan-2026-08-12

## Governed Massive admission

The bounded ticker-reference import persisted Massive evidence for the seeded source identities. It admits only active US stocks / CS records with a primary exchange.

| Result | Count |
|---|---:|
| Eligible source identities | 172 decisions |
| Ineligible source identities | 7 |
| Canonical eligible securities | 125 |
| Canonical ineligible securities | 7 |

The ineligible records are ADR/OTC classifications rather than US common stocks: ARM, ASML, BABA, HSBC, NVS, TCEHY, and TSM. They remain in the catalog with their source evidence but are excluded from the hot EOD cohort.

## Canonicalization

The compatibility seed contained 178 source identities. Forty-six symbols appeared under both src-massive and src-spglobal. Migration 000093_subscriber_global_canonical_identity retains every original source row and immutable reference observation, while resolving the records to 132 canonical securities. src-massive is selected as the canonical head where it exists.

The resulting 125 selected plan members are distinct canonical global IDs. A source alias cannot produce a second planned EOD symbol.

## Current EOD shadow plan

| Measure | Result |
|---|---:|
| Mode | shadow |
| Capacity | 1,000 |
| Canonical candidates | 132 |
| Eligible | 125 |
| Selected | 125 |
| Excluded | 7 (not_eligible) |
| Coverage rows outside shadow mode | 0 |
| Activation requests | 0 |
| Legacy MarketOps universe rows changed by S2 | 0 |

No price, options, intraday, historical, or provider collection job was invoked. No browser or tenant projection path was enabled.

## Top-1,000 breadth gap

Massiveºw^~)Þus All Tickers endpoint supports filtering active US common stocks, but a live probe rejected sort=market_cap as an invalid sort field. It therefore cannot, by itself, produce an authoritative market-cap-ranked top 1,000 in a single bounded catalogue call.

The current 125-security plan is a governed compatibility cohort, not a claim that the full top 1,000 has been defined. Before S2 can claim that capacity is populated, the platform needs an approved, auditable ranking source and snapshot policzéÝyø§yÔfor example a licensed market-cap/liquidity ranking feed or an explicitly governed constituent/ranking dataset. The source must yield a point-in-time ranked list without expanding continuous collection or guessing rank from ticker order.

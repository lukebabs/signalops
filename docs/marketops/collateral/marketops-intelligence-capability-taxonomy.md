# MarketOps Intelligence Capability Taxonomy

MarketOps presents analyst-facing capabilities separately from the stable
technical models that produce their evidence. Capability names describe the
research question; model names, versions, and algorithm IDs remain visible in
calculation traces and provenance.

| Capability | Current technical lineage | Analyst purpose |
|---|---|---|
| Value Intelligence | Valuation Composite (VC) | Explain relative valuation from price, financial, and peer-context evidence. |
| Distressed Opportunity Intelligence | Distressed Opportunity Scoring Model (DOSM) | Prioritize deeper research using valuation, operating quality, cash generation, balance-sheet, and bounded technical context. |
| Earnings Opportunity Intelligence | Earnings Event Opportunity Model (EEOM) | Assess deterministic pre-earnings setup quality from persisted evidence; it is not an earnings forecast or recommendation. |
| Sector Rotation Intelligence | SRI | Provide research-only, price-led sector and industry relative-strength context. |
| Market Structure Intelligence | Planned | A future capability. It does not rename or alter Market State, Risk/Reward, EROC, tactical posture, or intraday monitoring. |

The taxonomy does not change routes, API payloads, database fields, model IDs,
scheduler names, retained evidence, or historical provenance.

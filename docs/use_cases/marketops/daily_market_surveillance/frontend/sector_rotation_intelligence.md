# Sector Rotation Intelligence Frontend

Route: /marketops/sectors

Sector Intelligence is a mobile-first MarketOps view for reading price-led sector and industry context. It is available from the MarketOps navigation as Sector Intelligence.

## Card semantics

Each card shows the segment ID, composite score, context state, rank, evidence quality, relative strength, momentum, acceleration, completed session date, and expandable method/input provenance.

The screen never treats a card as a trade instruction. Its visible evidence note states that the Foundation does not claim rotation, breadth, diffusion, flows, or a recommendation.

## ETF progression

The **ETF progression** tab lists the 16 scored primary ETFs in the same dense, sortable table pattern as Assets. Its columns expose rank, ETF and segment identity, composite score, state, relative strength, momentum, acceleration, quality, and completed session. Select a row to open its inline chart without leaving the page. The chart defaults to composite score and can switch to relative strength, momentum, or acceleration. Each point retains its session date, state, rank, and quality in the tooltip.

The history endpoint returns up to 60 usable, common completed market sessions, so incomplete snapshots do not displace a scored session in the chart. On a narrow viewport, the table supports horizontal swipe and selecting a row scrolls its inline chart into view.

The selected-row panel also has an **ETF makeup** tab. It shows a current, issuer-published holdings snapshot, its effective date, reported total and top-ten weights, a source link, and a compact scrollable constituent table. The makeup is explicitly representational: it does not alter SRI scores and does not claim to recreate historical holdings.

State Street daily holdings files provide this free snapshot for 12 of the 16 current SRI primary ETFs: KRE and the eleven Select Sector SPDR ETFs. IBB, IGV, OIH, and SMH remain transparently unavailable until a separately governed public source is added.

For source coverage, provenance, API behavior, schedule, and troubleshooting, see [SRI Current ETF Makeup](../operations/sri_etf_makeup.md).

## Color language

Color reinforces, but never replaces, the text state:

| Color | State or score meaning |
|---|---|
| Emerald | LEADING or strong composite score (75+) |
| Sky | IMPROVING or positive score (60+) |
| Slate | NEUTRAL |
| Amber | WEAKENING, partial quality, or a low/watch score |
| Rose | LAGGING or unavailable/unusable quality |

Cards carry the state color through the left accent border, state badge, rank emphasis, and score tone. Quality has its own labelled badge: usable is emerald, partial is amber, and other unusable values are rose. The legend remains visible so users do not need to rely on color alone.

## Operator validation

1. Sign in and select MarketOps.
2. Open /marketops/sectors.
3. Confirm the context legend, segment/type filters, evidence note, and responsive card grid render.
4. Confirm card state, score, rank, and quality colors agree with their text labels.
5. Test a narrow mobile viewport: filters wrap, cards remain one column, and method/input details remain expandable without horizontal page scrolling.
6. Change segment and state filters. The displayed values must remain research-only context.

If no usable snapshots exist, the empty state identifies the 61-session price requirement. A later partial seed must not mask a previous usable ranking.

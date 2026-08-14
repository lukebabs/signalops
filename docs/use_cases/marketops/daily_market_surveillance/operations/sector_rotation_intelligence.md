# Sector Rotation Intelligence Foundation Operations

Status: implemented research-only foundation.

Sector Rotation Intelligence (SRI) supplies a deterministic, price-led cross-sectional view of covered sectors and industries. It is an analyst context surface, not a signal generator, rotation claim, market forecast, or trade recommendation.

## Implemented scope

The versioned foundation is:

- algorithm: sri.foundation.v1
- registry: sri.registry.v1
- storage migration: 000086_marketops_sector_rotation_intelligence
- active registry: three context benchmarks, 11 Select Sector ETFs, and five industry-pair contexts
- scored output: 16 non-benchmark sector and industry snapshots

The benchmark basket is SPY, QQQ, and RSP. Sector primary ETFs are XLB, XLC, XLE, XLF, XLI, XLK, XLP, XLRE, XLU, XLV, and XLY. Industry contexts use IGV/SKYY, SMH/SOXX, IBB/XBI, KRE/KBE, and OIH/XOP; the first ETF is the current scoring primary and the second is preserved as a registry comparison mapping.

## Inputs and readiness

SRI reads only canonical normalized equity EOD prices from the existing normalized event ledger. It does not call a provider while scoring and does not infer prices, fills, holdings, flows, options activity, breadth, or constituent data.

A segment is usable only when its primary ETF and every benchmark have at least 61 price sessions. Otherwise SRI persists a partial NEUTRAL snapshot with INSUFFICIENT_PRICE_HISTORY and no rank. Missing data is never converted to a zero score.

Inspect readiness with:

~~~sql
SELECT normalized_payload->>'symbol' AS symbol,
       count(DISTINCT observation_time::date) AS sessions,
       max(observation_time::date) AS latest_session
FROM normalized_event_ledger
WHERE tenant_id = 'tenant-local'
  AND dataset = 'equity_eod_prices'
  AND normalized_payload->>'symbol' IN
      ('SPY','QQQ','RSP','XLB','XLC','XLE','XLF','XLI','XLK','XLP',
       'XLRE','XLU','XLV','XLY','IGV','SKYY','SMH','SOXX','IBB','XBI',
       'KRE','KBE','OIH','XOP')
GROUP BY 1
ORDER BY 1;
~~~

For a historical remediation, use bounded Massive equity pulls over explicit date ranges, explicit symbols, maximum observation days, provider-request caps, event caps, a request delay, and continue-on-error. Publish through the existing raw-event and normalizer path; do not write SRI tables directly or synthesize missing sessions. Recheck the canonical ledger before calculating SRI.

## Calculation

For each eligible primary ETF, SRI calculates 5-, 20-, and 60-session returns, then subtracts the mean return of SPY, QQQ, and RSP at the same horizon. It ranks the eligible cross-section into percentiles:

- Relative strength: 10% 5-session, 40% 20-session, 50% 60-session relative-return percentile.
- Momentum: 15% 5-session, 35% 20-session, 50% 60-session absolute-return percentile.
- Acceleration: percentile of 5-session return minus the preceding 20-session return.
- Composite: 55% relative strength, 30% momentum, 15% acceleration.

States are bounded labels for this price-led context:

| State | Rule |
|---|---|
| LEADING | composite at least 75 |
| IMPROVING | composite at least 60 and acceleration at least 55 |
| NEUTRAL | no other state rule matched |
| WEAKENING | composite below 40 and acceleration below 45 |
| LAGGING | composite below 25 |

The state does not assert capital flow, constituent breadth, rotation in/out, future return, or a recommended action.

## Daily execution

SRI is independent of the active-asset post-close workflow. A dedicated weekday 20:07 America/New_York job first reconciles the 24 versioned registry ETFs and benchmarks to one completed EOD session, waits for canonical normalization, and only then invokes the runner. The registry is seeded idempotently and the runner upserts deterministic snapshots. A missing source ETF fails the job closed; it never publishes a stale session as current.

Manual execution:

~~~bash
scripts/marketops_sri_refresh.sh --date YYYY-MM-DD
~~~

Expected output is JSON with segments, snapshots, and partial. A successful execution writes one deterministic snapshot per scored segment for the applicable market session and algorithm version.

Apply storage before the first run:

~~~bash
docker compose --profile storage run --rm postgres-migrate
~~~

## Historical progression materialization

The ETF Progression view reads only canonical EOD events and displays up to the last 60 common, usable market sessions. It makes no provider call. Initialize or refresh that history with:

~~~bash
docker compose --profile marketops-daily run --rm marketops-sri-runner --tenant-id tenant-local --as-of YYYY-MM-DD --backfill-sessions 60 --run-id sri-progression-60-session-YYYYMMDD
~~~

The runner uses only sessions shared by every configured primary ETF and benchmark. It reports the session bounds, snapshot count, and partial count as JSON; incomplete historical records remain excluded from the progression API.


## Current issuer holdings snapshots

For complete source, coverage, provenance, API, UI, schedule, verification, and troubleshooting guidance, see [SRI Current ETF Makeup](sri_etf_makeup.md).

Migration 000087_marketops_sri_etf_holdings adds append-only current ETF makeup snapshots and their constituents. The free State Street adapter downloads the issuer daily XLSX holdings files for KRE and XLB, XLC, XLE, XLF, XLI, XLK, XLP, XLRE, XLU, XLV, and XLY. It records source URL, issuer effective date, retrieval time, content hash, and reported constituent weights.

This is a present-time representation layer only. It is not an SRI input, cannot change historical scores, and does not infer unavailable holdings. The remaining primary ETFs (IBB, IGV, OIH, and SMH) return an explicit unavailable state in the UI.

Run a collection manually with:

    docker compose --profile marketops-daily run --rm marketops-sri-holdings-runner --tenant-id tenant-local

The independent weekday marketops-sri-holdings-refresh timer runs at 20:20 America/New_York, after SRI scoring. A source failure is reported as a failed governed job; the prior immutable snapshot remains visible rather than being replaced with fabricated data.

## Verification and troubleshooting

1. Confirm migrations 000086 and 000087 are applied.
2. Confirm all primary ETFs and benchmarks satisfy the 61-session readiness query.
3. Run the SRI command for a completed market session.
4. Confirm 16 latest usable snapshots, ranks, and quality_state=usable.
5. Run the issuer holdings command and confirm current snapshots for the 12 State Street-covered primary ETFs.
6. Open /marketops/sectors under an authenticated MarketOps session.
7. If a later partial seed exists, the API intentionally selects the latest usable snapshot for rankings and asset context; it does not let a partial placeholder hide a usable prior session.

Useful relational check:

~~~sql
SELECT s.name, s.segment_type, x.state, x.composite_score, x.rank,
       x.session_date, x.quality_state
FROM sri_segment_snapshots x
JOIN sri_segments s
  ON s.tenant_id = x.tenant_id AND s.segment_id = x.segment_id
WHERE x.tenant_id = 'tenant-local'
  AND x.quality_state = 'usable'
  AND x.session_date = (
    SELECT max(session_date)
    FROM sri_segment_snapshots
    WHERE tenant_id = 'tenant-local' AND quality_state = 'usable'
  )
ORDER BY x.rank, s.name;
~~~

## Explicit boundaries and next work

SRI Foundation does not publish MarketOps signals, SAF assertions, alerts, opportunities, or Syncratic knowledge changes. It does not mutate VC, DOSM, EEOM, Tactical Market Posture, or Exhaustive Reversal.

Historical holdings, constituent breadth, diffusion, flows, options context, pairwise rotation, state-change event publication, backtesting, and SAF validation are separate future work. Any addition must preserve point-in-time inputs, versioned contracts, immutable provenance, and the research-only boundary until separately governed.

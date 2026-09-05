# SRI Current ETF Makeup: Data, Operations, and Analyst Guide

Status: implemented, research-only representation layer.

This guide documents the current ETF makeup capability shown in MarketOps Sector Intelligence. It explains exactly what the feature represents, where the data comes from, how it is stored and refreshed, how to use it in the UI, and—equally important—what it must not be used to infer.

The feature supplements the price-led Sector Rotation Intelligence (SRI) Foundation. It does not change the SRI algorithm, produce a signal, validate an assurance assertion, or reconstruct historical ETF composition.

Related documents:

- [SRI Foundation operations](sector_rotation_intelligence.md) covers price readiness, calculation, scoring, progression history, and the research boundary.
- [SRI frontend guide](../frontend/sector_rotation_intelligence.md) covers the Sector Intelligence route and display conventions.

## Purpose and decision boundary

An SRI score tells an analyst how the primary ETF performed relative to the configured benchmark basket over the SRI lookbacks. A current makeup snapshot helps the analyst understand what that ETF represents today—for example, its leading constituents and concentration.

These are separate questions:

| Question | SRI Foundation answer | ETF makeup answer |
|---|---|---|
| What is the current relative-strength and momentum context? | Yes, from canonical EOD prices. | No. |
| What companies or instruments are represented by this ETF today? | No. | Yes, when the issuer publishes a supported holdings file. |
| What was in the ETF on a past SRI session? | No. | No; historical reconstruction is deliberately out of scope. |
| Does a constituent cause the SRI score or state? | No causal claim. | No causal claim. |
| Is this a trading recommendation or signal confirmation? | No. | No. |

The displayed data is an issuer-published, point-in-time representation. It should be read alongside the effective date and source link, not as a causal explanation of an SRI state.

## Coverage

SRI has 16 scored primary ETFs. State Street provides free daily holdings files for 12 of them. The collector only requests the supported symbols; it never tries to substitute a similar ETF or scrape a non-governed source.

| SRI primary ETF | Context | Current makeup status | Source |
|---|---|---|---|
| XLB | Materials | Available | State Street daily holdings |
| XLC | Communication Services | Available | State Street daily holdings |
| XLE | Energy | Available | State Street daily holdings |
| XLF | Financials | Available | State Street daily holdings |
| XLI | Industrials | Available | State Street daily holdings |
| XLK | Technology | Available | State Street daily holdings |
| XLP | Consumer Staples | Available | State Street daily holdings |
| XLRE | Real Estate | Available | State Street daily holdings |
| XLU | Utilities | Available | State Street daily holdings |
| XLV | Health Care | Available | State Street daily holdings |
| XLY | Consumer Discretionary | Available | State Street daily holdings |
| KRE | Regional Banks industry context | Available | State Street daily holdings |
| IBB | Biotechnology industry context | Unavailable | No governed free issuer adapter |
| IGV | Software industry context | Unavailable | No governed free issuer adapter |
| OIH | Oil Services industry context | Unavailable | No governed free issuer adapter |
| SMH | Semiconductors industry context | Unavailable | No governed free issuer adapter |

An unavailable result is intentional transparency. The API and UI say that no current issuer snapshot is available; they do not backfill, estimate, approximate, or borrow holdings from an alternative fund.

At first collection on 2026-08-11 UTC, the issuer files supplied an effective date of 2026-08-07 for all 12 supported ETFs. That issuer effective date is retained on every displayed snapshot. The retrieval timestamp is retained separately, so a stale issuer file is distinguishable from the time SignalOps downloaded it.

## Source and adapter contract

The adapter uses the public State Street daily holdings XLSX URL pattern:

~~~
https://www.ssga.com/library-content/products/fund-data/etfs/us/holdings-daily-us-en-{symbol}.xlsx
~~~

The symbol is allow-listed and normalized to upper case before the source URL is constructed. Supported symbols are KRE, XLB, XLC, XLE, XLF, XLI, XLK, XLP, XLRE, XLU, XLV, and XLY.

For every allowed symbol, the adapter:

1. Performs an HTTPS GET with a 30-second client timeout.
2. Rejects a non-2xx response.
3. Reads no more than 10 MiB from the source file.
4. Parses XLSX shared strings and the first holdings worksheet.
5. Requires issuer metadata for Fund Name and Holdings As of date.
6. Requires a holdings header row containing Name, Weight, and Shares Held.
7. Captures the optional Ticker, Identifier, SEDOL, Sector, and Local Currency columns when present.
8. Retains constituents with a non-empty name and valid numeric weight in issuer workbook order.
9. Calculates the reported total constituent weight and top-ten weight from the parsed issuer values.
10. Hashes the complete downloaded workbook with SHA-256.

Weights are stored and displayed in the issuer's percentage units. A stored value of 14.467241 is shown as 14.47%, not multiplied again as a ratio.

The source URL is stored with the snapshot and presented as the Issuer source link in the UI. Analysts should open that link when they need the original issuer file or need to validate the source's own disclosure terms.

## Data model and provenance

Migration 000087_marketops_sri_etf_holdings introduces two tables.

### sri_etf_holdings_snapshots

One row represents one retrieved issuer file for one tenant, ETF, effective date, source, and content hash.

| Field | Meaning |
|---|---|
| snapshot_id | Deterministic SignalOps identifier derived from tenant, ETF, effective date, source, and file hash. |
| tenant_id | Tenant scope. |
| etf_symbol | SRI primary ETF symbol. |
| fund_name | Fund name supplied by the issuer workbook. |
| effective_date | Issuer Holdings As of date, not the collector run date. |
| retrieved_at | UTC time SignalOps successfully read the workbook. |
| source | state_street. |
| source_url | Exact public XLSX address retrieved. |
| content_hash | SHA-256 digest of the complete workbook. |
| holdings_count | Number of stored constituents. |
| total_weight | Sum of parsed issuer holding weights. |
| top_ten_weight | Sum of the first ten stored constituents' weights. |
| created_at | Database insert timestamp. |

The uniqueness constraint is tenant_id, etf_symbol, effective_date, source, content_hash. If the issuer republishes changed content for the same effective date, the changed hash is preserved as a distinct immutable snapshot. If the content is identical, the collector is idempotent.

### sri_etf_holdings

One row represents one constituent from a stored snapshot.

| Field | Meaning |
|---|---|
| snapshot_id | Parent issuer snapshot; deletion is restricted. |
| holding_key | Deterministic key derived from the snapshot, rank, ticker, identifier, and name. |
| holding_rank | Issuer workbook order, starting at one. |
| ticker, name | Issuer-supplied constituent identity where available. |
| identifier, sedol | Issuer-supplied identifiers where present. |
| sector, currency | Optional issuer-supplied classification and local currency. |
| weight | Issuer-reported percentage weight. |
| shares_held | Issuer-reported shares held. |

Snapshot rows are append-only. The system does not update an old snapshot to reflect a newer issuer workbook, and it does not erase an older snapshot when a new one is collected.

## Collection flow

The collection command is the marketops-sri-holdings-runner. It reads the active SRI ETF registry for the requested tenant and processes primary ETFs only.

~~~
docker compose --profile marketops-daily run --rm marketops-sri-holdings-runner --tenant-id tenant-local
~~~

The runner returns a compact JSON report:

~~~
{"etfs":12,"snapshots":12,"holdings":700,"unsupported_primary_etfs":4}
~~~

Report fields:

| Field | Meaning |
|---|---|
| etfs | Supported primary ETFs successfully downloaded and stored during this run. |
| snapshots | Snapshot records accepted for storage. |
| holdings | Constituent records accepted for storage. |
| unsupported_primary_etfs | Active primary SRI ETFs without a State Street adapter. |

The first successful local collection stored 12 issuer snapshots containing 700 holdings. Counts, effective dates, and concentration change as issuers rebalance; do not treat this initial count as a permanent coverage contract.

If a supported source request or workbook parse fails, the runner exits with an error. It does not publish a partial replacement snapshot. Previously stored immutable snapshots remain queryable. A source failure is visible through the governed scheduled-job status and is eligible for administrator notification.

## Automated schedule

The user-level systemd timer signalops-marketops-sri-holdings-refresh.timer is enabled and calls the governed job marketops-sri-holdings-refresh.

| Job | Schedule | Time zone | Purpose |
|---|---|---|---|
| marketops-sri-refresh | Weekdays 20:07 | America/New_York | Reconcile SRI prices and calculate the price-led foundation. |
| marketops-sri-holdings-refresh | Weekdays 20:20 | America/New_York | Refresh supported current issuer ETF makeup snapshots. |

The 13-minute ordering is intentional. Holdings collection is independent of price scoring, but runs after it so the analyst experience sees a completed SRI score before a matching current representation layer is refreshed.

Install or refresh all MarketOps user timers with:

~~~
./scripts/install_marketops_daily_user_timer.sh
~~~

Verify the timer:

~~~
systemctl --user status signalops-marketops-sri-holdings-refresh.timer --no-pager
systemctl --user list-timers 'signalops-*' --no-pager
~~~

The timer records runtime state through the governed scheduled-job wrapper. The operational source is the dedicated MarketOps database tables `marketops_scheduled_job_statuses` and `marketops_scheduled_job_runs`; `runtime/scheduled-jobs/` is fallback/debug output only. Do not edit runtime files by hand.

## API contract

The read-only makeup endpoint is:

~~~
GET /v1/marketops/sectors/{segment_id}/makeup?tenant_id={tenant_id}&limit={limit}
~~~

It requires the same authenticated MarketOps session and tenant scope as the other Sector Intelligence endpoints. The limit defaults to 25 and applies to holdings returned in ascending issuer rank.

For a supported ETF with a stored snapshot, the response has this shape:

~~~
{
  "segment_id": "sri_sector_technology",
  "etf_symbol": "XLK",
  "availability": "available",
  "snapshot": {
    "snapshot_id": "sri_holdings_...",
    "fund_name": "Technology Select Sector SPDR Fund",
    "effective_date": "2026-08-07",
    "retrieved_at": "2026-08-11T06:...",
    "source": "state_street",
    "source_url": "https://www.ssga.com/...",
    "holdings_count": 76,
    "total_weight": 99.98,
    "top_ten_weight": 61.29
  },
  "holdings": [
    {
      "rank": 1,
      "ticker": "NVDA",
      "name": "NVIDIA CORP",
      "weight": 14.47,
      "sector": "Information Technology"
    }
  ],
  "research_only": true,
  "evidence_note": "Current issuer-published ETF composition for representation only. It does not affect SRI scores or reconstruct historical holdings."
}
~~~

For a primary ETF without a governed current-source adapter, the endpoint returns HTTP 200 with availability unavailable, an empty holdings list, a reason, and research_only true. This is not a 404 because the SRI segment exists; its composition simply is not available under the current source policy.

For a segment with no primary ETF, the endpoint returns availability not_configured. An underlying query error remains a 500.

## Analyst workflow in the UI

1. Sign in to SignalOps and open MarketOps.
2. Navigate to /marketops/sectors.
3. Choose ETF progression.
4. Select a row in the ETF table. The row expands in place and scrolls into view on narrow displays.
5. Use Progression to inspect up to 60 common, usable historical SRI sessions. Switch among composite, relative-strength, momentum, and acceleration metrics.
6. Choose ETF makeup to inspect the current issuer snapshot.
7. Read the fund name, issuer effective date, holdings count, reported total weight, and top-ten weight before interpreting the rows.
8. Use Issuer source to open the original public holdings file when a deeper audit is needed.
9. If the tab says unavailable, treat that as a known coverage gap—not a zero exposure, empty portfolio, or weak SRI score.

The expanded makeup table is horizontally scrollable within its own container on mobile. It does not require scrolling past the whole ETF table to find the selected ETF's detail.

## Verification and diagnostics

Apply migration 000087 before executing the collector:

~~~
docker compose --profile storage run --rm postgres-migrate
~~~

Confirm the latest stored snapshots:

~~~
SELECT etf_symbol,
       effective_date,
       retrieved_at,
       holdings_count,
       round(total_weight::numeric, 2) AS total_weight,
       round(top_ten_weight::numeric, 2) AS top_ten_weight,
       source
FROM sri_etf_holdings_snapshots
WHERE tenant_id = 'tenant-local'
ORDER BY etf_symbol, effective_date DESC, retrieved_at DESC;
~~~

Confirm a constituent table is populated:

~~~
SELECT h.holding_rank,
       h.ticker,
       h.name,
       round(h.weight::numeric, 4) AS weight,
       h.sector
FROM sri_etf_holdings h
JOIN sri_etf_holdings_snapshots s ON s.snapshot_id = h.snapshot_id
WHERE s.tenant_id = 'tenant-local'
  AND s.etf_symbol = 'XLK'
ORDER BY s.effective_date DESC, s.retrieved_at DESC, h.holding_rank
LIMIT 25;
~~~

Confirm that only supported current sources have snapshots:

~~~
SELECT r.etf_symbol,
       r.segment_id,
       CASE WHEN s.snapshot_id IS NULL THEN 'unavailable' ELSE 'available' END AS makeup_status
FROM sri_etf_registry r
LEFT JOIN LATERAL (
  SELECT snapshot_id
  FROM sri_etf_holdings_snapshots
  WHERE tenant_id = r.tenant_id
    AND etf_symbol = r.etf_symbol
  ORDER BY effective_date DESC, retrieved_at DESC
  LIMIT 1
) s ON true
WHERE r.tenant_id = 'tenant-local'
  AND r.role = 'primary'
  AND r.active = true
ORDER BY r.etf_symbol;
~~~

Expected present state is 12 available and four unavailable primary ETFs. If the count changes, verify the registry, source allow-list, and job report before changing documentation or treating the difference as an error.

## Troubleshooting

| Symptom | Likely cause | Safe response |
|---|---|---|
| Job reports a non-2xx source error | Issuer file URL or public availability changed. | Keep the prior snapshot; investigate the issuer file manually; update the allow-listed adapter only through review. |
| Workbook parse error | Issuer XLSX structure changed or is malformed. | Preserve the downloaded error context in job logs; adapt the parser and add a fixture test before re-running. |
| No snapshot in an otherwise valid SRI row | ETF is one of IBB, IGV, OIH, or SMH, or the first collection has not run. | Show unavailable; do not infer composition. |
| Total weight is near, but not exactly, 100% | Issuer rounding, cash, derivatives, or disclosure conventions. | Preserve the issuer value; do not normalize weights silently. |
| UI shows an older effective date | The issuer has not published a newer file or the refresh failed. | Compare retrieved_at and source URL; inspect job status; do not label an old file as current market composition. |
| Progression and makeup seem inconsistent | They answer different questions and have different time bases. | Use the SRI session date for score history and the issuer effective date for makeup. Do not imply causality. |
| Need composition for an uncovered ETF | No governed free adapter is active. | Propose a source and adapter with provenance, licensing, availability, and parser tests before enabling it. |

## Governance constraints and future extensions

The following are deliberately not implemented:

- Historical holdings reconstruction or point-in-time constituent backfills.
- Weight-based re-scoring, attribution, causal explanations, or a claim that a constituent drove a state.
- Holdings-derived breadth, diffusion, flows, factor exposures, options activity, or rotation signals.
- A substitute source for IBB, IGV, OIH, or SMH without a separately reviewed adapter.
- SAF assertions, automated alerts, opportunity generation, or trade recommendations based on makeup data.

Any extension must preserve the same minimum controls: source licensing and availability review, tenant scoping, retained source URL and effective date, immutable content provenance, deterministic IDs, parser tests, explicit coverage states, and a documented separation from the SRI scoring algorithm.

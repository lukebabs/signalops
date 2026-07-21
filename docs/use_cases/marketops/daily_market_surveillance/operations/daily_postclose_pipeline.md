# Daily Post-Close Pipeline

Use `scripts/marketops_daily_postclose.sh` for one governed MarketOps session. The workflow is a single bounded batch, not one scheduler per asset. It runs equity acquisition, waits for canonical normalization, captures point-in-time options evidence, and executes research intelligence in cohorts of at most ten symbols.

## Safety Contract

- Same-session writes are blocked before 18:00 in `America/New_York`.
- `--write` requires `MARKETOPS_DAILY_ACKNOWLEDGE_WRITES=true`.
- A non-blocking file lock rejects overlapping runs.
- Equity acquisition attempts the Top 50 with exactly 50 provider requests, no retries, and at most 50 built/published events.
- The normalization barrier requires all 50 distinct same-session equity symbols before options acquisition.
- Options acquisition uses the same explicit 50-symbol supported list, at most two 250-record pages and 500 transient candidates per symbol, no automatic retries, and at most seven persisted evidence contracts per symbol.
- Intelligence runs in five explicit batches of at most ten symbols. Graph decisions, Syncratic Ask, hypothesis promotion, and trading remain outside the workflow.
- Existing complete equity coverage and successful deterministic cohort batches are skipped. A non-success cohort rerun requires operator inspection.
- Provider no-data, partial options surfaces, missing features, zero triggers, and zero outcomes remain explicit.

The 2026-07-21 universe revision removes `2222.SR`, `005930.KS`, `000660.KS`, `601939.SS`, and `RO.SW`. It adds the next five provider-supported megacaps not already retained: Philip Morris International (`PM`), Royal Bank of Canada (`RY`), Alibaba (`BABA`), Novartis (`NVS`), and Palo Alto Networks (`PANW`). Each replacement returned one real 2026-07-20 equity aggregate from Massive before activation.

Selection used the [current global market-cap ordering](https://companiesmarketcap.com/) and skipped additional unsupported primary-exchange listings rather than recreating the original data gap. Migration `000039` keeps the database asset universe aligned with the embedded acquisition seed.

## Plan And Dry Run

Render the exact commands without provider or storage access:

```bash
scripts/marketops_daily_postclose.sh --date 2026-07-21 --write --plan
```

Run a no-write provider validation after the session has settled:

```bash
scripts/marketops_daily_postclose.sh --date 2026-07-21 --dry-run
```

A dry-run stops after equity when persisted same-session coverage is below the normalization threshold, because dry-run events are intentionally not published.

## Manual Write

```bash
MARKETOPS_DAILY_ACKNOWLEDGE_WRITES=true \
  scripts/marketops_daily_postclose.sh --date 2026-07-21 --write
```

## Timer Installation

Install the user timer:

```bash
scripts/install_marketops_daily_user_timer.sh
```

The installer builds the three pinned workflow images before enabling the timer. The timer runs at exactly 18:01:55 `America/New_York`, Monday through Friday, with one-second timer accuracy and no randomized delay. `Persistent=true` starts a missed invocation when the user service manager resumes; before 18:00 the script resolves the most recent completed weekday rather than using an incomplete current session. For execution while the user is logged out, an administrator must enable linger:

```bash
loginctl enable-linger "$USER"
```

Inspect status and logs:

```bash
systemctl --user list-timers signalops-marketops-daily.timer
systemctl --user status signalops-marketops-daily.service
journalctl --user -u signalops-marketops-daily.service
```

## Failure Handling

The service exits non-zero when dependencies are down, the credential preflight fails, normalization does not reach the explicit threshold, capture rows are incomplete, a non-success deterministic cohort run already exists, or a child stage exits unsuccessfully. Inspect the provider report and persisted capture/cohort ledgers before manually rerunning the same session.

Do not use the legacy `massive-scheduler` interval loop for this workflow. It does not provide the normalization, G142 options, or ten-symbol cohort barriers.

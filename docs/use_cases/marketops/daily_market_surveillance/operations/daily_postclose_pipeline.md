# Daily Post-Close Pipeline

Use `scripts/marketops_daily_postclose.sh` for one governed MarketOps session. The workflow is a single bounded batch, not one scheduler per asset. It runs equity acquisition, waits for canonical normalization, captures point-in-time options evidence, and executes research intelligence in cohorts of at most ten symbols.

## Safety Contract

- Same-session writes are blocked before 18:00 in `America/New_York`.
- `--write` requires `MARKETOPS_DAILY_ACKNOWLEDGE_WRITES=true`.
- A non-blocking file lock rejects overlapping runs.
- Equity acquisition attempts the Top 50 with exactly 50 provider requests, no retries, and at most 50 built/published events.
- A durable reconciliation stage compares the active database universe with exact normalized equity rows, enqueues only missing symbols, and processes one symbol at a time in rank order.
- Reconciliation makes at most two additional provider calls per queued symbol, uses 30-second then two-minute bounded backoffs, caps the run at 100 provider requests and 15 minutes, and replays an existing raw event once before calling the provider. Exhausted tasks remain failed until an operator uses `--requeue-failed`.
- The normalization barrier requires all 50 active same-session equity symbols from `src-massive` before options acquisition.
- Options acquisition uses the same explicit 50-symbol supported list, at most two 250-record pages and 500 transient candidates per symbol, no automatic retries, and at most seven persisted evidence contracts per symbol.
- Intelligence runs in five explicit batches of at most ten symbols. Graph decisions, Syncratic Ask, hypothesis promotion, and trading remain outside the workflow.
- Existing complete equity coverage and successful deterministic cohort batches are skipped. A non-success cohort rerun requires operator inspection.
- Provider no-data, partial options surfaces, missing features, zero triggers, and zero outcomes remain explicit.

The 2026-07-21 universe revision removes `2222.SR`, `005930.KS`, `000660.KS`, `601939.SS`, and `RO.SW`. It adds the next five provider-supported megacaps not already retained: Philip Morris International (`PM`), Royal Bank of Canada (`RY`), Alibaba (`BABA`), Novartis (`NVS`), and Palo Alto Networks (`PANW`). Each replacement returned one real 2026-07-20 equity aggregate from Massive before activation.

Selection used the [current global market-cap ordering](https://companiesmarketcap.com/) and skipped additional unsupported primary-exchange listings rather than recreating the original data gap. Migration `000039` keeps the database asset universe aligned with the embedded acquisition seed.

## Equity Reconciliation

Migration `000040` adds `marketops_equity_reconciliation_tasks`. Its unique tenant/source/universe/dataset/date/symbol key makes discovery idempotent, while explicit queued, running, awaiting-normalization, succeeded, and failed states preserve recovery evidence across process restarts. Expired running leases are returned to the queue. A normalized row always short-circuits provider access; an existing raw row is republished once through the supported replay envelope.

Inspect without writes or provider calls:

```bash
docker compose --profile massive-pull run --rm massive-puller --mode reconcile-equity --date 2026-07-20 --dry-run=true
```

Repair an incomplete session:

```bash
docker compose --profile massive-pull run --rm massive-puller --mode reconcile-equity --date 2026-07-20 --max-attempts 2 --deadline 15m --retry-backoffs 30s,2m --normalization-poll 5s --max-provider-requests 100 --dry-run=false --acknowledge-writes
```

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

Reconciliation reports are bounded JSON in the service journal. Inspect `marketops_equity_reconciliation_tasks` for per-symbol attempts, replay count, last error, lease, and terminal status. Failed tasks are intentionally not reset by an ordinary direct rerun; use `--requeue-failed` only after inspecting the provider or normalization failure.

Live acceptance on 2026-07-21 repaired the 2026-07-20 session from 45 to 50 exact normalized equity symbols. The queue processed `PM`, `RY`, `BABA`, `NVS`, and `PANW` sequentially with five provider requests and five successful terminal tasks. The final ledger contained exactly one row per active symbol, included canonical `GOOGL`, and did not require `GOOG`.

Do not use the legacy `massive-scheduler` interval loop for this workflow. It does not provide the normalization, G142 options, or ten-symbol cohort barriers.

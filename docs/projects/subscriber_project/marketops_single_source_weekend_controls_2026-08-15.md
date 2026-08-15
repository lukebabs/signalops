# MarketOps single-source and non-trading-day controls

Status: source controls complete; live read-switch verification is required.

## Observed drift

The running Gateway read the shared SignalOps database while MarketOps batch
work wrote to the dedicated MarketOps database. The shared store's newest
Risk/Reward snapshot and options distribution were for 2026-08-13; the
dedicated store had the corresponding 2026-08-14 rows. This was a source
selection split, not a browser-cache or client-rendering failure.

## Single-source policy

MarketOps-only Gateway routes must read the dedicated MarketOps primary and
temporal stores. The shared databases remain the authority for non-MarketOps
platform data and retain the historic MarketOps copy only as rollback evidence;
they are not a second live MarketOps source after cutover. No shared database
or historical rows are deleted by this control.

The root cutover launcher now parses only the two required boundary credentials
as literal data from a root-owned, non-group/world-readable file. It rejects
unexpected or duplicate keys and does not execute the environment file. After
replacement it verifies that the Gateway has both dedicated MarketOps settings
and emitted its dedicated-read startup evidence.

## Weekend policy

2026-08-15 is a Saturday. EOD, intraday, post-close recovery, tactical retry,
SRI refresh, and financial continuation jobs are skipped on non-trading days.
The scheduler records a successful `skipped` status with reason
`non_trading_day`; it does not invoke the job command. Only explicit
maintenance controls—operations monitoring, storage monitoring, and retention
governance—may run on weekends. The FMP continuation timer is now weekdays
only.

This is a calendar-day guard, not a market-holiday calendar. Market-holiday
handling remains a separate admission requirement before re-enabling any
scheduled MarketOps workload.

## Required live verification

1. Redeploy only the Gateway with the protected production environment and the
   read-cutover overlay.
2. Confirm its dedicated-read startup evidence and both dedicated settings
   without printing credentials.
3. Verify protected MarketOps asset and options views advance to the dedicated
   store's latest completed session, and verify one non-MarketOps route.
4. Keep EOD and intraday schedulers disabled this weekend; inspect their status
   artifacts for `skipped/non_trading_day` if they are manually invoked.
5. Retain the shared copy until rollback and route-validation evidence are
   recorded; do not delete shared platform databases.

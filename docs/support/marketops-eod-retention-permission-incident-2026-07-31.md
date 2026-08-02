# MarketOps EOD Retention Permission Incident — 2026-07-31

## Summary

The governed MarketOps post-close user timer started normally at 18:01:55 ET on 2026-07-31. It completed acquisition, normalization, options, cohorts, risk/reward execution, and algorithm corroboration, then stopped at the risk-reward retention step. Financial retention, the daily Tactical Posture runner, the universal completion gate, and Syncratic intelligence did not run.

## Root cause

`scripts/marketops_risk_reward_retention.sh` had mode `0664` rather than an executable mode. The post-close workflow invoked it directly, so Bash returned `Permission denied` and systemd marked the service failed with exit status `126`.

This was not a timer failure. `signalops-marketops-daily.timer` remained enabled, and `signalops-marketops-daily.service` recorded the failed invocation.

## Evidence

- Timer invocation: 2026-07-31 22:01:55 UTC / 18:01:55 ET.
- Failure: 2026-07-31 22:10:21 UTC at `marketops_daily_postclose.sh` line 391.
- Service result: `status=126`.
- Error: `./scripts/marketops_risk_reward_retention.sh: Permission denied`.

## Permanent fix

The post-close workflow invokes both retention scripts through `bash`, rather than executing them directly. A missing executable bit can no longer block EOD processing. Script files retain executable mode for direct operator use.

## Recovery procedure

1. Inspect the failed session journal:
   `journalctl --user -u signalops-marketops-daily.service --since '2026-07-31 22:00:00 UTC'`.
2. Confirm the required upstream work completed and note the session date.
3. Re-run the governed workflow only with the explicit write acknowledgement:
   `MARKETOPS_DAILY_ACKNOWLEDGE_WRITES=true scripts/marketops_daily_postclose.sh --date YYYY-MM-DD --write`.
4. Confirm the workflow reaches the universal completion gate and that `tactical_market_posture_v1` results are present for assets with complete technical inputs.

## Verification

- Run `bash -n scripts/marketops_daily_postclose.sh`.
- Confirm both retention scripts can be invoked with `bash`.
- On the next scheduled weekday run, confirm `systemctl --user status signalops-marketops-daily.service` reports success and the journal includes the tactical runner after retention.

# Dashboard performance guardrails

Status: implemented and verified 2026-09-15.

The Dashboard renders the Market Intelligence reel independently of the aggregate signal-overview request. The aggregate request is allowed to complete in the background, and the last successful projection remains visible for 60 seconds while a refresh is in flight. Independent signal-overview inputs are read concurrently by the gateway.

The Playwright guardrail requires the reel to become visible within three seconds and verifies that the signal-overview request still occurs. It does not impose a brittle fixed latency target on the database-backed aggregate, but preserves the key user-facing contract: the Dashboard shell must not be blank while aggregate analytics load.

Run:

```bash
scripts/run_marketops_dashboard_performance_ui_smoke.sh
```

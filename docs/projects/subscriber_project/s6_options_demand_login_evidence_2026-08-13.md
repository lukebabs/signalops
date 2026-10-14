# S6 Options-Demand Login Evidence — 2026-08-13

Status: dedicated workload login provisioned and preflight passed. Options demand remains default-deny; no snapshot, capture, provider call, or scheduler is enabled.

The database principal `signalops_subscriber_options_demand_runner` was created with `LOGIN NOINHERIT`, no administrative attributes, and membership only in `signalops_subscriber_options_demand`.

Its real password-authenticated preflight passed from an ephemeral PostgreSQL client container. The session used the explicit role boundary:

```text
PGOPTIONS=-c role=signalops_subscriber_options_demand
```

The preflight uses that role selection because the login is intentionally `NOINHERIT`. The planner command itself performs the same fixed role transition after authenticating, so its deployment secret needs only the runner DSN. It must not share a login, credential, or role session with the gateway, global-EOD reconciler, or Options-capture worker.

The preflight verified aggregate-only input, append-only shadow-storage access, and absence of direct subscriber/list/Options-capture-table privileges. It made no provider request and wrote no snapshot.

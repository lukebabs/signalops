# S6 Options Capture Canary Gate

Status: disabled-by-default control-plane implementation. It does not create an Options capture, contact Massive, or schedule work.

Migration `000113` allows `subscriber-options-capture` to freeze exactly one asset: priority one from an existing non-zero S6 shadow snapshot. The plan requires a provider budget of one, `provider_execution_enabled=false`, `scheduled_execution_enabled=false`, and an engaged kill switch. The database allows only `capture_planned` evidence while disabled.

The capture identity can read only the aggregate S6 snapshot and global asset identity required to freeze this member. It has no entitlement, list, membership, gateway, or existing Options-evidence access.

A later execution release must create a distinct named authorization and worker that retains request intent before an external call, prospective readiness policy, response and normalization evidence, one-request ceiling, and rollback controls. This gate is not that release.

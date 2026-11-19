# S6 Options Capture Gate Evidence — 2026-08-13

Status: disabled control plane deployed; no Options capture has occurred.

Migration `000113_subscriber_options_capture_canary_gate` is applied. A separate `LOGIN NOINHERIT` principal, `signalops_subscriber_options_capture_runner`, was created with membership only in `signalops_subscriber_options_capture`. Its password-authenticated preflight passed through the explicit restricted role.

The capture gate frozen from the pilot shadow plan is:

| Field | Value |
| --- | --- |
| Gate | `suboptcapturegate_e9a7fb2c2551ff26b087b7a5` |
| Source shadow plan | `suboptdemand_a35104af4805f1578bf3d697` |
| Frozen asset | NVDA, request ordinal 1 |
| Provider budget | 1 |
| State | `disabled` |
| Provider and scheduler execution | `false` / `false` |
| Kill switch | engaged |
| Readiness policy | `subscriber-options-prospective-readiness-v1` |
| Evidence ledger | one `capture_planned`; no provider evidence |

The capture-worker identity has no direct access to lists, memberships, entitlements, or existing Options evidence. The one-time generated credential is stored outside the repository at `/tmp/signalops-s6-options-capture-runner-credential` with mode `600`; move it to the secret manager and delete the file.

The next release, if approved, must be a separate named one-request authorization and capture worker. This gate cannot be turned on by browser action, a schedule, or a restart.

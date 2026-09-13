# S6 Options Capture Named-Approval Evidence — 2026-08-13

Migration `000115_subscriber_options_capture_named_approval` is applied. The following immutable attestation is now attached to the pending NVDA review request:

| Field | Value |
| --- | --- |
| Approval | `suboptcaptureapprovalattestation_20260813_nvda` |
| Approver | `luke@strategiclabs.io` |
| State | `approved_pending_recovery` |
| Provider / request limit / retries | Massive / 1 / 0 |
| Recovery gate | `blocked` |

The retained approval statement is: “I, luke@strategiclabs.io, approve one Massive Options capture for the frozen NVDA canary, with no retries.”

This is not an execution authorization. The provider gate remains disabled and the kill switch remains engaged. Completion of encrypted off-host backup/WAL configuration and an isolated restore rehearsal is required before a separately reviewed execution release may consume this approval.

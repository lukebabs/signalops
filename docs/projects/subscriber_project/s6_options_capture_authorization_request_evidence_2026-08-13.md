# S6 Options Capture Authorization-Request Evidence — 2026-08-13

Migration `000114_subscriber_options_capture_authorization_request` is applied. The dedicated capture worker recorded the following review request against the unchanged disabled NVDA gate:

| Field | Value |
| --- | --- |
| Request | `suboptcaptureapproval_8ee12a1dcd9b69db1434f93f` |
| Gate | `suboptcapturegate_e9a7fb2c2551ff26b087b7a5` |
| State | `pending_approval` |
| Requested worker / provider | `subscriber-options-capture` / Massive |
| Provider budget | 1 |
| Scope | NVDA prospective Options-capture canary |

This is intentionally a request for human review, not approval. It does not modify the disabled gate, kill switch, scheduler state, feature flags, provider credentials, or existing Options data, and it made no provider request.

An explicit future authorization must name the approver and capture execution release, retain provider intent before the one external request, and satisfy the separate recovery and rollback gates.

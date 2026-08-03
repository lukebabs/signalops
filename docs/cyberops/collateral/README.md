# SignalOps CyberOps Collateral

This folder contains externally safe, evidence-led CyberOps positioning. It describes the implemented firewall and IoT-network-observability capability within SignalOps as of August 2026.

## Documents

- [CyberOps Datasheet](cyberops-datasheet.md): concise product, workflow, data, and governance overview.
- [Executive Value Brief](cyberops-executive-value-brief.md): business problems, differentiated value, and operating outcomes.
- [Analyst Capability Guide](cyberops-analyst-capability-guide.md): how an analyst uses traffic, anomaly, signal, insight, alert, and settings views.
- [Detection and Evidence Reference](cyberops-detection-evidence-reference.md): detection semantics, learning conditions, lifecycle policy, and limitations.

## Positioning guardrails

CyberOps is deterministic, evidence-driven security operations support. It does not block traffic, alter firewall policy, perform automated remediation, make threat-attribution claims, or replace a SIEM, SOAR, or managed detection-and-response service. A firewall allow is not automatically safe, and a firewall deny is not proof of malicious activity.

The operational source of truth is `docs/use_cases/cyberops/README.md`, the lifecycle-noise-control specification, and its runbook. Retention claims are governed by `docs/support/retention-governance.md`.

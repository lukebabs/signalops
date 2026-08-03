# CyberOps Analyst Capability Guide

## Start with evidence, then decide whether a condition warrants investigation

CyberOps is built for security operators who need a clear picture of network activity without treating every firewall event as an incident. Its workflow starts with allowed traffic and feed quality, then adds internal-device behavior, explicit service configuration, durable Signals, and lifecycle-governed work items.

## Use the traffic dashboard to understand the environment

The CyberOps dashboard is the starting point for current network behavior. Choose a one-hour, 24-hour, or seven-day window to inspect:

- total received logs, parsed allowed events, and unparsed logs;
- top source addresses and destinations;
- common protocols and destination ports;
- source-to-destination flows, including first/last seen and count;
- a streamed, rolling live-firewall inflow timeline.

Use the chart and ranked panels to select an entity or flow, then inspect the filtered flow list. A sudden fall in all received logs may be an ingestion or source issue; a rise in unparsed logs may be parser quality—not necessarily a security event.

The dashboard intentionally emphasizes allowed traffic. A deny is useful evidence, but it is not by itself proof of malicious activity or a reason to interrupt an analyst.

## Configure the interpretation boundary in Settings

CyberOps Settings provides two important tenant-scoped controls.

### Internal network CIDRs

Enter one internal CIDR per line and select **Apply CIDRs**. This identifies the addresses whose behavior will be treated as internal-device activity in the IoT/network-behavior workflow. The configuration is explicit: without internal CIDRs, CyberOps cannot responsibly claim a device-level baseline for the environment.

### Approved public services

Add an approved service using destination IP, TCP/UDP protocol, destination port, and an optional reason. This is an allow-list for interpretation, not a firewall rule and not an authorization to expose a service. It provides context when public-source allowed traffic reaches a destination service.

Only authorized administrators can change this configuration. The platform records the actor and before/after state for policy audit.

## Read the behavior and anomaly panel

The IoT/network-behavior panel uses the configured internal CIDRs to show device-to-peer traffic, direction, protocols, ports, count, and first/last seen. It also displays:

- **Learning progress:** active baseline hours compared with the required minimum.
- **Current assessment:** the observed metric, the historical mean/stddev when eligible, and any withheld reason.
- **Status:** learning, waiting for a completed hour, baseline ready, or threshold met.
- **Novel service observations:** device/peer/protocol/port combinations that provide an investigation lead.

The current deterministic anomaly worker evaluates two completed-hour metrics per internal device:

1. allowed firewall-log count; and
2. number of distinct peers.

It requires at least 24 active baseline hours and uses a seven-day lookback. Only an absolute z-score of at least 3σ is emitted. A high score is a deviation from that device’s observed recent pattern; it does not diagnose a cause or prove maliciousness.

## Distinguish Signals, Insights, and Alerts

| View | What you are seeing | How to use it |
|---|---|---|
| Signals | Immutable detector observations | Review the evidence, signal type, source, and lifecycle decision before escalating |
| Insights | Grouped, reviewable work items | Assess the episode, fingerprint, counts, first/last observed times, and approval context |
| Alerts | Policy-approved urgent conditions | Acknowledge or resolve according to your response process; examine linked evidence first |

CyberOps does not make these three terms interchangeable. A valid Signal may remain record-only. A repeated signal should update a bounded episode instead of creating many items. A lifecycle decision is designed to answer why the disposition occurred.

## Interpret current lifecycle behavior correctly

The deployed CyberOps v1 policy is in shadow mode for the local environment. In shadow mode, the system retains signals, grouped episodes, and immutable projected decisions but does not create new policy-driven Insights or Alerts. This is deliberate: it allows the organization to assess projected volume and evidence quality before enabling interruption workflows.

The baseline policy semantics are:

- External firewall deny: record evidence only.
- Approved public-service exposure: record evidence only.
- Unapproved public-service exposure: projected low-severity Insight, deduplicated by destination IP/protocol/port.
- Public-source port scan: projected high-severity Alert and linked Insight when one source reaches ten distinct denied ports on one destination in five minutes.

## Investigation checklist

1. Confirm that the live feed is healthy and parsing successfully.
2. Verify the selected time window and whether the event is allowed or denied.
3. For device behavior, confirm the CIDR configuration and baseline-learning status.
4. Treat a z-score as a deviation, then inspect the underlying flow, peers, protocol, and port for context.
5. Check whether a destination service is explicitly approved; never infer approval from repeated prior traffic.
6. Review the policy reason, fingerprint, episode count, and linked Signals before treating an Insight or Alert as an incident.
7. Perform threat investigation, asset context, vulnerability review, and response using the organization’s approved security process; CyberOps does not make those determinations automatically.

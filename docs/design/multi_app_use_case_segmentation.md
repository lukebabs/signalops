# Multi-App Use Case Segmentation

Timestamp: `2026-07-10T06:45:00Z`

## Purpose

SignalOps must support multiple independent user-facing workstreams without
forking the core engine. Market data, security, IoT, data quality, and future
tenant-specific workflows may each need their own app experience, terminology,
navigation, dashboards, source views, detector packs, and operational defaults.

The scalable model is a unified SignalOps core platform with independently
composable domain packs and app profiles.

## Design Principle

SignalOps should not create separate backends per use case.

Use cases should be represented as app profiles and domain packs that compose
the same durable engine:

```text
SignalOps Core
  -> Domain Packs
  -> App Profiles
  -> User-Facing Apps
```

This preserves one reliability model, one replay model, one auth model, one
observability surface, and one canonical storage strategy while allowing each
use case to feel like a focused product.

## Layer Model

### Core Platform Layer

The core platform remains domain-neutral and owns:

- raw event ingestion
- normalized event contracts
- Redpanda topic boundaries
- normalizer services
- detector worker boundaries
- signal, alert, and insight ledgers
- retry, DLQ, replay, and idempotency behavior
- PostgreSQL and TimescaleDB persistence
- auth, RBAC, and tenant isolation
- source, pipeline, rule, detector, and replay registries
- public and internal APIs
- health and observability

The core platform must not hardcode one use case as the primary product model.

### Domain Pack Layer

A domain pack defines a reusable use-case family.

Examples:

- `marketdata`
- `security`
- `iot`
- `data_quality`
- `business_operations`

A domain pack may include:

- source adapters
- source schemas
- normalization policies
- detector packs
- feature definitions
- rule packs
- alert and insight templates
- dashboard/widget presets
- replay policies
- retention defaults
- terminology defaults

Example structure:

```text
domains/
  marketdata/
    sources/
    schemas/
    detectors/
    features/
    dashboards/
    rules/
  security/
  iot/
  data_quality/
```

Domain packs must not bypass core idempotency, replay, tenant isolation,
canonical persistence, or lifecycle controls.

### App Profile Layer

An app profile is a user-facing composition of core capabilities and domain
packs.

An app profile owns:

- app identity
- route set
- navigation
- default dashboard layout
- default filters
- enabled modules
- labels and terminology
- role and permission mapping
- widget composition
- default source and pipeline views

An app profile does not own:

- canonical ingestion
- durable broker topics
- detector execution semantics
- replay mechanics
- signal persistence
- alert/insight lifecycle state
- cross-tenant isolation

## Product Representation

SignalOps should keep a unified operator console:

```text
SignalOps Console
  Dashboard
  Event Explorer
  Timeline
  Correlation
  Insights
  Pipelines
  Rules
  Sources
  Health
  Replay
  Administration
  Settings
```

Specialized apps should present domain-specific navigation on top of the same
engine. The first specialized app profile should be `marketops`.

```text
MarketOps
  Market Dashboard
  Symbols
  Option Contracts
  Signals
  Alerts
  Replay
  Providers
  Pipelines
  Health
```

Future examples:

```text
SecOps
  Security Dashboard
  Events
  Incidents
  Entities
  Detections
  Alerts
  Replay
  Health
```

```text
IoTOps
  Fleet Dashboard
  Devices
  Telemetry
  Signals
  Alerts
  Replay
  Health
```

## First-Class Metadata

The next architecture gate should introduce the following metadata as
first-class routing, filtering, authorization, and presentation fields:

- `app_id`
- `domain`
- `use_case`

These should augment the already established source, dataset, pipeline,
detector, signal, and lifecycle metadata.

Recommended event/API shape:

```json
{
  "tenant_id": "tenant-local",
  "app_id": "marketops",
  "domain": "market_data",
  "use_case": "daily_market_surveillance",
  "source_id": "massive",
  "source_domain": "market_data",
  "source_adapter": "massive",
  "dataset": "eod_prices",
  "pipeline_id": "marketdata.eod.v1",
  "detector_id": "marketdata.price_gap.v1"
}
```

## Backend Accommodation

Backend implementation should treat `app_id`, `domain`, and `use_case` as
additive metadata at first.

The initial gate should:

- extend relevant request DTOs, normalized event records, signal records, alert
  records, insight records, and query filters where appropriate
- preserve backward compatibility when these fields are omitted
- default existing local data to the platform console context unless a specific
  app profile is supplied
- ensure broker messages carry the metadata through normalization, detection,
  signal publication, and persistence
- index/filter these fields where operator queries need them
- document the metadata contract in API and event documentation

The first implementation does not need a full app registry table unless the
metadata wiring exposes a concrete need. A static app profile definition for
`console` and `marketops` is sufficient for the first gate.

## Frontend Accommodation

The frontend should evolve into a multi-app shell:

```text
web/src/
  core/
    auth/
    api/
    layout/
    stream-client/
    widgets/
  apps/
    console/
    marketops/
    securityops/
    iotops/
  domain-packs/
    marketdata/
    security/
    iot/
```

The first app profile should be configuration-driven:

```ts
{
  appId: "marketops",
  label: "MarketOps",
  defaultRoute: "/marketops/dashboard",
  enabledModules: ["dashboard", "signals", "alerts", "sources", "replay", "health"],
  dashboardProfile: "marketdata.default",
  terminology: {
    source: "Provider",
    event: "Market Event",
    signal: "Market Signal"
  }
}
```

The current SignalOps UI should remain the platform console and should not be
replaced by MarketOps. MarketOps should be introduced as the first specialized
profile that composes existing APIs and widgets with market-data terminology
and filters.

## Architecture Gate

Next gate:

- Introduce `app_id`, `domain`, and `use_case` as first-class routing and
  presentation metadata.
- Define the first app profile as `marketops`.
- Keep the current SignalOps UI as the platform console.

Acceptance criteria:

- Existing ingestion, normalization, detection, persistence, replay, and UI
  paths continue to work without requiring the new fields.
- New metadata can flow from raw ingestion through normalized events and
  signals.
- API list/detail filters can include the new metadata where relevant.
- Documentation defines the metadata semantics and defaults.
- `marketops` is defined as the first specialized app profile.
- SignalOps Console remains available as the domain-neutral operator app.

## Non-Goals

- Do not fork the backend per app.
- Do not create a separate database per app profile in the first gate.
- Do not replace the SignalOps Console with MarketOps.
- Do not introduce tenant-provided executable plugins until the core app/domain
  metadata contract is stable.
- Do not require all existing historical records to be backfilled before the
  additive metadata path is accepted.

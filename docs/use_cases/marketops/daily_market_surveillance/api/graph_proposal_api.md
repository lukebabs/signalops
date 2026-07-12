# Graph Proposal API

Status: implemented in G079; used by G080 operator review workflow
Use case: MarketOps Daily Market Surveillance

## Purpose

This note documents the G079 API boundary for reviewing graph target candidates that are embedded in persisted DSM artifacts. G080 uses the decision endpoint from the DSM Workbench for bounded operator review.

The API exposes graph proposals as first-class review records. It does not mutate an external graph database.

## Endpoints

Use existing `/v1/*` gateway conventions and authentication.

Implemented endpoints:

- `GET /v1/marketops/dsm/graph-proposals`
- `GET /v1/marketops/dsm/graph-proposals/{proposal_id}`
- `POST /v1/marketops/dsm/graph-proposals/{proposal_id}/decision`

## List Filters

The list endpoint supports bounded, index-friendly filters:

- `tenant_id` from the current request pattern
- `app_id`, default `marketops`
- `domain`, default `market_data`
- `use_case`, default `daily_market_surveillance`
- `artifact_id`
- `signal_id`
- `subject_symbol`
- `signal_type`
- `candidate_type`
- `status`
- `limit`, capped to the existing API convention

## Response Shape

List response:

```json
{
  "graph_proposals": [
    {
      "proposal_id": "graphprop_marketops_dsm_v1_...",
      "tenant_id": "default",
      "app_id": "marketops",
      "domain": "market_data",
      "use_case": "daily_market_surveillance",
      "artifact_id": "artifact_marketops_dsm_v1_...",
      "signal_id": "sig_marketops_dsm_taxonomy_v1_...",
      "candidate_type": "relationship_candidate",
      "node_id": "",
      "from_node": "ticker:AAPL",
      "relationship": "EXHIBITS_SIGNAL",
      "to_node": "signal_type:marketops.dsm.pinning_risk",
      "labels": [],
      "properties": { "severity": "high" },
      "confidence": 0.84,
      "severity": "high",
      "status": "proposed",
      "created_at": "2026-07-11T00:00:00Z",
      "updated_at": "2026-07-11T00:00:00Z"
    }
  ]
}
```

Detail response:

```json
{
  "graph_proposal": {
    "proposal_id": "graphprop_marketops_dsm_v1_...",
    "artifact_id": "artifact_marketops_dsm_v1_...",
    "signal_id": "sig_marketops_dsm_taxonomy_v1_...",
    "candidate_type": "node_candidate",
    "node_id": "artifact:artifact_marketops_dsm_v1_...",
    "labels": ["DSMArtifact"],
    "properties": { "artifact_id": "artifact_marketops_dsm_v1_...", "severity": "high" },
    "status": "proposed"
  }
}
```

Decision request:

```json
{
  "status": "accepted",
  "note": "Approved for downstream graph materialization."
}
```

Decision response should return the updated `graph_proposal` envelope.

## Validation Rules

The API should reject:

- unknown statuses
- decisions on missing proposal ids
- malformed or empty notes if notes become required by policy
- attempts to change immutable candidate identity fields through the decision endpoint

The API should allow idempotent status writes. Repeating the same decision with the same status should not create duplicate records.

## Auth And Tenancy

G079 should follow the same gateway authentication posture as `/v1/marketops/dsm/artifacts`.

Unauthenticated requests should return `401 unauthorized` when auth is enabled. Authenticated requests must be tenant-scoped consistently with existing signal, artifact, alert, and insight APIs.


## G080 Frontend Usage

The DSM Workbench uses `POST /v1/marketops/dsm/graph-proposals/{proposal_id}/decision` to update review state only. The request body contains the target `status` and optional `note`. Auth-enabled requests derive the actor from the bearer token; auth-disabled local development requests may use the existing `X-SignalOps-Actor` fallback.

The frontend must not use this endpoint to imply graph materialization. Accepted proposals remain review records until a later gate introduces explicit production graph-write semantics.

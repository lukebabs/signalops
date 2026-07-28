BEGIN;

-- New immutable policy definition for normalization-quality evaluation. Existing
-- published definitions are not rewritten; this policy is selected explicitly by
-- the Massive ingestion runtime while the compatibility flag is enabled.
INSERT INTO platform_primitive_definitions (
  tenant_id, primitive_type, definition_id, definition_key, version, app_id, domain, use_case,
  status, implementation_ref, point_in_time_required, contract, metadata
) VALUES (
  'tenant-local',
  'policy',
  'policy_signalops_normalized_event_quality_v1',
  'signalops.normalized_event_quality',
  '1.0.0',
  '',
  '',
  '',
  'active',
  'policy/normalized_event_quality_gate:v1',
  true,
  '{
    "policy_kind":"quality_state_gate",
    "scope":"normalized_event",
    "allowed_states":["usable","degraded","partial","stale","missing","invalid","contradictory","not_applicable","suppressed"],
    "default_state":"usable",
    "failure_action":"dlq",
    "fails_closed":true
  }'::jsonb,
  '{"registry_seed":"normalized_event_quality_policy_v1"}'::jsonb
)
ON CONFLICT (tenant_id, primitive_type, definition_key, version) DO NOTHING;

COMMIT;

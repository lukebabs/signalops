DELETE FROM platform_primitive_definition_audit_events
WHERE tenant_id = 'tenant-local'
  AND definition_id = 'policy_signalops_normalized_event_quality_v1';

DELETE FROM platform_primitive_definitions
WHERE tenant_id = 'tenant-local'
  AND primitive_type = 'policy'
  AND definition_id = 'policy_signalops_normalized_event_quality_v1'
  AND definition_key = 'signalops.normalized_event_quality'
  AND version = '1.0.0';

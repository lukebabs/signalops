-- NarrativeOps is a first-class SignalOps application profile and must be
-- registered before tenant-scoped access grants can reference it.
INSERT INTO registered_use_case_profiles (app_id)
VALUES ('narrativeops')
ON CONFLICT (app_id) DO NOTHING;

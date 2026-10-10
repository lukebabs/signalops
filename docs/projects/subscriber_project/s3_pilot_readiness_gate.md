# S3 Pilot Readiness Gate

The S3 API remains disabled until a named pilot tenant passes the following preflight:

    SIGNALOPS_SUBSCRIBER_LISTS_ENABLED=false \
    SIGNALOPS_SUBSCRIBER_LISTS_PILOT_TENANTS=<pilot-tenant> \
    bash ./scripts/subscriber_project_s3_pilot_preflight.sh --tenant-id <pilot-tenant>

The script refuses to run if the feature is already enabled. It verifies:

1. the tenant is explicitly named in the pilot configuration;
2. the dedicated gateway login passes its least-privilege and forced-RLS preflight;
3. all S3 list tables exist;
4. the pilot tenant has an active Subscriber entitlement; and
5. exactly one tenant-default list is provisioned.

The final two requirements are intentionally unresolved until product ownership selects the pilot entitlement/tier and initial tenant-default-list policy. A preflight failure is the expected safe result until that controlled provisioning occurs.

After a passing preflight, the deployment owner may enable the flag for only the named tenant, then run the browser ownership, administrator, and cross-tenant tests. No collection, global EOD coverage, or catalog provider pull is enabled by this gate.

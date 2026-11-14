import os
import copy
import time
from datetime import date, datetime
from zoneinfo import ZoneInfo
from typing import Any

from dagster import (
    Config,
    DagsterRunStatus,
    Definitions,
    Failure,
    OpExecutionContext,
    Out,
    RunRequest,
    SkipReason,
    job,
    op,
    run_status_sensor,
    schedule,
)


class SessionConfig(Config):
    session_date: str | None = None


REQUIRED_FEATURES = (
    "range_position_252d",
    "rsi_14",
    "return_5d",
    "volume_ratio_10d",
    "distance_sma_50_pct",
    "distance_sma_200_pct",
    "sma_50_slope_20d_pct",
    "atr_14_pct",
)


def _session_date(config: SessionConfig) -> str:
    return config.session_date or os.getenv("MARKETOPS_SESSION_DATE") or date.today().isoformat()


def _scheduled_session_date(scheduled_execution_time: datetime | None) -> str:
    """Return the ET trading session represented by a Dagster schedule tick.

    The child worker must use the scheduler's session, not its own wall clock.
    This is especially important for UTC ticks after midnight and for manual
    reconciliation runs launched from the Dagster UI.
    """

    if scheduled_execution_time is None:
        return date.today().isoformat()
    return scheduled_execution_time.astimezone(ZoneInfo("America/New_York")).date().isoformat()


def _marketops_database_url() -> str | None:
    direct = os.getenv("MARKETOPS_DATABASE_URL") or os.getenv("SIGNALOPS_MARKETOPS_DATABASE_URL")
    if direct:
        return direct
    injected = "/vault/secrets/marketops-runtime.env"
    try:
        for raw_line in open(injected, encoding="utf-8"):
            line = raw_line.strip()
            if line.startswith("export MARKETOPS_DATABASE_URL="):
                return line.split("=", 1)[1].strip().strip("'\"")
    except FileNotFoundError:
        pass
    return None


@op(out=Out(dict))
def observe_marketops_dependency_watermark(
    context: OpExecutionContext, config: SessionConfig
) -> dict[str, Any]:
    """Read-only completeness gate for the current MarketOps asset-list revision."""

    database_url = _marketops_database_url()
    session_date = _session_date(config)
    if not database_url:
        result = {"status": "blocked", "reason": "marketops_database_url_missing", "session_date": session_date}
        context.log.warning(result)
        return result

    import psycopg

    with psycopg.connect(database_url) as connection:
        with connection.cursor() as cursor:
            cursor.execute(
                """
                SELECT lists.current_revision, revisions.membership_fingerprint, count(members.global_asset_id)::int
                FROM marketops_asset_lists lists
                JOIN marketops_asset_list_revisions revisions
                  ON revisions.list_id = lists.list_id
                 AND revisions.revision = lists.current_revision
                JOIN marketops_asset_list_members members
                  ON members.list_id = revisions.list_id
                 AND members.revision = revisions.revision
                WHERE lists.list_id = 'marketops-primary-tenant-local'
                GROUP BY lists.current_revision, revisions.membership_fingerprint
                """
            )
            list_revision, list_fingerprint, expected = cursor.fetchone()
            cursor.execute(
                """
                SELECT count(*)::int
                FROM (
                  SELECT o.symbol
                  FROM marketops_feature_observations o
                  WHERE o.tenant_id = 'tenant-local'
                    AND o.app_id = 'marketops'
                    AND o.session_date = %s::date
                    AND o.feature_key = ANY(%s)
                    AND o.quality_state IN ('usable', 'usable_with_warning')
                    AND o.numeric_value IS NOT NULL
                  GROUP BY o.symbol
                  HAVING count(DISTINCT o.feature_key) >= 4
                ) ready
                """,
                (session_date, list(REQUIRED_FEATURES)),
            )
            ready = cursor.fetchone()[0]
            cursor.execute(
                """
                SELECT count(*)::int
                FROM marketops_risk_reward_snapshots
                WHERE session_date = %s::date
                """,
                (session_date,),
            )
            snapshot_count = cursor.fetchone()[0]
            cursor.execute(
                """
                SELECT count(*)::int
                FROM subscriber_gateway_global_risk_reward_snapshots
                WHERE session_date = %s::date
                  AND eligible = true
                  AND usable_input_count >= 4
                """,
                (session_date,),
            )
            projected = cursor.fetchone()[0]

    # Projection is the downstream consumer of this watermark.  Requiring the
    # projected count to be complete here creates a circular gate (projection
    # cannot run until the watermark is ready, while the watermark cannot be
    # ready until projection has run).  Upstream readiness is therefore the
    # admission criterion; projection records its own completion afterward.
    status = "ready" if expected == ready else ("degraded" if snapshot_count >= expected else "blocked")
    reason = "" if status == "ready" else (
        "unavailable_assets_excluded_from_usable_cohort" if status == "degraded"
        else "dependency_watermark_incomplete"
    )
    result = {
        "status": status,
        "session_date": session_date,
        "asset_list_id": "marketops-primary-tenant-local",
        "asset_list_revision": int(list_revision),
        "asset_list_fingerprint": list_fingerprint,
        "expected_assets": expected,
        "ready_assets": ready,
        "eligible_projected_assets": projected,
        "snapshot_assets": snapshot_count,
        "required_minimum_inputs": 4,
        "reason": reason,
    }
    with psycopg.connect(database_url) as connection:
        with connection.cursor() as cursor:
            cursor.execute(
                """
                SELECT marketops_record_dependency_watermark(
                  %s, %s, %s::date, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s
                )
                """,
                (
                    f"dagster-watermark:tenant-local:{session_date}:{list_revision}",
                    "tenant-local",
                    session_date,
                    "marketops-primary-tenant-local",
                    list_revision,
                    list_fingerprint,
                    expected,
                    ready,
                    projected,
                    4,
                    status,
                    reason,
                    f"dagster-shadow-{context.run_id}",
                ),
            )
    context.log.info(result)
    return result


@job
def marketops_dependency_shadow_graph() -> None:
    observe_marketops_dependency_watermark()


@schedule(
    name="dagster_marketops_dependency_watermark_schedule",
    job=marketops_dependency_shadow_graph,
    cron_schedule="50 18 * * 1-5",
    execution_timezone="America/New_York",
)
def marketops_dependency_watermark_schedule(context):
    scheduled = context.scheduled_execution_time.isoformat() if context.scheduled_execution_time else "manual"
    session_date = _scheduled_session_date(context.scheduled_execution_time)
    return RunRequest(
        run_key=f"dependency-watermark:{scheduled}",
        run_config={"ops": {"observe_marketops_dependency_watermark": {"config": {"session_date": session_date}}}},
        tags={"marketops_scheduler": "dagster", "marketops_session_date": session_date},
    )


MARKETOPS_CRONJOBS: dict[str, tuple[str, str]] = {
    "marketops-intraday": ("*/15 9-20 * * 1-5", "America/New_York"),
    "marketops-sri-refresh": ("0 18 * * 1-5", "America/New_York"),
    "marketops-sri-holdings-refresh": ("20 18 * * 1-5", "America/New_York"),
    "marketops-fmp-annual-financial": ("30 2 * * 6", "America/New_York"),
    "marketops-saf-benchmark": ("15 18 * * 1-5", "America/New_York"),
    "marketops-saf-evaluation": ("45 18 * * 1-5", "America/New_York"),
    "marketops-operations-monitor": ("0 * * * *", "UTC"),
    "marketops-retention-governance": ("30 2 * * *", "America/New_York"),
    "marketops-task-retry": ("*/15 16-20 * * 1-5", "America/New_York"),
    "marketops-warm-eod": ("20 16 * * 1-5", "America/New_York"),
    "marketops-daily-postclose": ("35 17 * * 1-5", "America/New_York"),
    "marketops-syncratic-intelligence": ("45 16 * * 1-5", "America/New_York"),
    "marketops-postclose-recovery": ("*/15 16-20 * * 1-5", "America/New_York"),
    "marketops-risk-reward": ("45 17 * * 1-5", "America/New_York"),
    "marketops-fmp-continuation": ("0 4 * * 1-5", "America/New_York"),
    "marketops-global-dashboard-projection": ("0 19 * * 1-5", "America/New_York"),
}


class LaunchConfig(Config):
    job_id: str
    timeout_seconds: int = 3600
    session_date: str | None = None


@op
def launch_marketops_cronjob(context: OpExecutionContext, config: LaunchConfig) -> None:
    """Launch and supervise one existing MarketOps CronJob template.

    The CronJob remains the immutable workload template during migration; Dagster
    becomes the sole scheduler and creates one child Job per scheduled run.
    """
    from kubernetes import client, config as kube_config

    namespace = os.getenv("MARKETOPS_CRONJOB_NAMESPACE", "signalops-marketops")
    kube_config.load_incluster_config()
    batch = client.BatchV1Api()
    cronjob = batch.read_namespaced_cron_job(config.job_id, namespace)
    template = cronjob.spec.job_template
    run_name = f"dagster-{config.job_id}-{context.run_id[:12].lower()}"
    job_spec = copy.deepcopy(template.spec)
    metadata = client.V1ObjectMeta(
        name=run_name,
        namespace=namespace,
        labels={
            "app.kubernetes.io/name": "signalops-marketops-dagster-runner",
            "signalops.syncratic.io/orchestrator": "dagster",
            "signalops.syncratic.io/job-id": config.job_id,
            "dagster.io/run-id": context.run_id,
        },
    )
    # Mark child workloads as Dagster-authorized.  This is consumed by the
    # post-close recovery guard and avoids treating the CronJob template's
    # legacy approval gate as an active scheduler boundary.
    for container in job_spec.template.spec.containers:
        container.env = list(container.env or [])
        container.env.append(client.V1EnvVar(name="MARKETOPS_SCHEDULER_AUTHORITY", value="dagster"))
        session_date = config.session_date or context.dagster_run.tags.get("marketops_session_date")
        if session_date:
            container.env.append(client.V1EnvVar(name="MARKETOPS_SESSION_DATE", value=session_date))
        if config.job_id in {"marketops-daily-postclose", "marketops-saf-evaluation"}:
            container.env.append(client.V1EnvVar(name="MARKETOPS_DEFER_GLOBAL_PROJECTION", value="true"))
    batch.create_namespaced_job(namespace, client.V1Job(metadata=metadata, spec=job_spec))
    context.log.info("launched Kubernetes Job %s for %s", run_name, config.job_id)

    deadline = time.monotonic() + config.timeout_seconds
    while time.monotonic() < deadline:
        current = batch.read_namespaced_job(run_name, namespace).status
        if (current.succeeded or 0) > 0:
            context.log.info("Kubernetes Job %s succeeded", run_name)
            return
        if (current.failed or 0) > 0:
            raise Failure(description=f"Kubernetes Job {run_name} failed")
        time.sleep(5)
    raise Failure(description=f"Kubernetes Job {run_name} exceeded {config.timeout_seconds}s")


def _build_marketops_job(job_id: str):
    @job(name=f"dagster_{job_id.replace('-', '_')}")
    def marketops_job() -> None:
        launch_marketops_cronjob()

    return marketops_job


def _build_marketops_schedule(job_id: str, cron: str, timezone: str):
    target_job = _build_marketops_job(job_id)

    @schedule(
        name=f"dagster_{job_id.replace('-', '_')}_schedule",
        job=target_job,
        cron_schedule=cron,
        execution_timezone=timezone,
    )
    def marketops_schedule(context):
        scheduled = context.scheduled_execution_time.isoformat() if context.scheduled_execution_time else "manual"
        session_date = _scheduled_session_date(context.scheduled_execution_time)
        return RunRequest(
            run_key=f"{job_id}:{scheduled}",
            run_config={"ops": {"launch_marketops_cronjob": {"config": {"job_id": job_id}}}},
            tags={
                "marketops_job_id": job_id,
                "marketops_scheduler": "dagster",
                "marketops_session_date": session_date,
                "marketops_reconciliation_attempt": "0",
            },
        )

    return target_job, marketops_schedule


_marketops_targets = [_build_marketops_schedule(job_id, cron, timezone) for job_id, (cron, timezone) in MARKETOPS_CRONJOBS.items()]
_marketops_jobs = [target for target, _ in _marketops_targets]
_marketops_schedules = [target for _, target in _marketops_targets]


@job(name="marketops_reconciliation_dispatch")
def marketops_reconciliation_dispatch() -> None:
    """Retry one failed MarketOps child Job under Dagster supervision."""

    launch_marketops_cronjob()


@run_status_sensor(
    name="dagster_marketops_failed_run_reconciliation",
    run_status=DagsterRunStatus.FAILURE,
    request_job=marketops_reconciliation_dispatch,
    minimum_interval_seconds=60,
)
def marketops_failed_run_reconciliation_sensor(context):
    """Bounded automatic retry for failed scheduled MarketOps runs.

    The child Kubernetes Job remains the source of workload execution. This
    sensor only retries a failed Dagster launch once, preserving the existing
    worker dependency guards and avoiding an unbounded retry loop.
    """

    failed_run = context.dagster_run
    job_id = failed_run.tags.get("marketops_job_id", "")
    if not job_id or not job_id.startswith("marketops-"):
        return SkipReason("not a MarketOps scheduled run")
    try:
        attempt = int(failed_run.tags.get("marketops_reconciliation_attempt", "0"))
    except ValueError:
        attempt = 0
    if attempt >= 1:
        return SkipReason(f"reconciliation retry already attempted for {job_id}")
    session_date = failed_run.tags.get("marketops_session_date") or _scheduled_session_date(None)
    return RunRequest(
        run_key=f"reconcile:{failed_run.run_id}",
        run_config={
            "ops": {
                "launch_marketops_cronjob": {
                    "config": {"job_id": job_id, "session_date": session_date}
                }
            }
        },
        tags={
            "marketops_job_id": job_id,
            "marketops_scheduler": "dagster",
            "marketops_session_date": session_date,
            "marketops_reconciliation_attempt": "1",
            "marketops_reconciliation_source_run": failed_run.run_id,
        },
    )


defs = Definitions(
    jobs=[marketops_dependency_shadow_graph, marketops_reconciliation_dispatch, *_marketops_jobs],
    schedules=[marketops_dependency_watermark_schedule, *_marketops_schedules],
    sensors=[marketops_failed_run_reconciliation_sensor],
)

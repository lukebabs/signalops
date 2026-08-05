import { useEffect, useState } from "react";
import { useAdministrationNotifications, useAdministrationSMTPSettings, useMutateAdministrationSMTPSettings, useHealthz, useReadyz, useRuns, useReplayStatus, useScheduledJobs, useMarketOpsTasks, useStorageOverview, useMutateAdministrationNotificationState } from '../api/queries';
import { useUi } from '../store/ui';
import { MetricTile } from '../components/MetricTile';
import { RefreshButton } from '../components/RefreshButton';
import { StatusBadge } from '../components/StatusBadge';
import { ErrorState, EmptyState } from '../components/States';
import { isApiError } from '../api/client';
import { formatUtc } from '../lib/format';
import { replayJobCount, worstReplayWorkerHealth, latestReplayWorkerSeenAt } from '../lib/replayStatus';
import { useTenant } from '../auth/session';
import { useAppProfile } from '../apps/AppProfileContext';

const BASE_URL =
  (import.meta.env.VITE_SIGNALOPS_API_BASE_URL ?? '').replace(/\/+$/, '') ||
  '(same-origin via dev proxy)';

export function SystemRoute() {
  const isMarketops = useAppProfile().currentAppId === 'marketops';
  const healthz = useHealthz();
  const readyz = useReadyz();
  const probe = useRuns(1); // storage availability probe: 200 = available, 503 = unavailable
  const lastRefresh = useUi((s) => s.lastRefresh);
  const lastStreamEventAt = useUi((s) => s.lastStreamEventAt);
  const streamConnected = useUi((s) => s.streamConnected);
  const streamError = useUi((s) => s.streamError);
  const restFallback = useUi((s) => s.streamMode) === 'rest_fallback';
  const setLastRefresh = useUi((s) => s.setLastRefresh);

  const TENANT_ID = useTenant();
  const replayStatus = useReplayStatus({ tenant_id: TENANT_ID, limit: 10 });
  const scheduledJobs = useScheduledJobs();
  const marketOpsTasks = useMarketOpsTasks(TENANT_ID);
  const storage = useStorageOverview();
  const notifications = useAdministrationNotifications(TENANT_ID);
  const mutateNotification = useMutateAdministrationNotificationState(TENANT_ID);

  const storageAvailable = probe.isSuccess;
  const storageUnavailable =
    probe.isError && isApiError(probe.error) && probe.error.status === 503;

  const replayStatusOk = replayStatus.data?.replay_status;
  const replayWorkers = replayStatusOk?.workers ?? [];
  const replayWorstHealth = worstReplayWorkerHealth(replayWorkers);
  const replayLastSeen = latestReplayWorkerSeenAt(replayWorkers);

  function refreshAll() {
    healthz.refetch();
    readyz.refetch();
    probe.refetch();
    replayStatus.refetch();
    scheduledJobs.refetch();
    storage.refetch();
    notifications.refetch();
    setLastRefresh(new Date().toISOString());
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-semibold">{isMarketops ? 'Health' : 'System'}</h1>
        <RefreshButton
          onClick={refreshAll}
          loading={healthz.isFetching || readyz.isFetching || probe.isFetching || replayStatus.isFetching}
        />
      </div>
      <div className="grid grid-cols-2 gap-2 md:grid-cols-3">
        <MetricTile
          label="Gateway Health (/healthz)"
          value={healthz.data?.status ?? (healthz.isError ? 'unreachable' : '…')}
          hint={healthz.data?.time}
        />
        <MetricTile
          label="Gateway Ready (/readyz)"
          value={readyz.data?.status ?? (readyz.isError ? 'unreachable' : '…')}
          hint={readyz.data?.time}
        />
        <MetricTile
          label="Storage Query"
          value={storageAvailable ? 'available' : storageUnavailable ? 'unavailable' : 'checking'}
          hint={
            storageUnavailable
              ? '503 storage_unavailable — check SIGNALOPS_DATABASE_URL and Postgres'
              : undefined
          }
        />
        <MetricTile label="API Base URL" value={<code className="text-sm">{BASE_URL}</code>} />
        <MetricTile label="Last Refresh" value={formatUtc(lastRefresh ?? undefined)} />
        <MetricTile
          label="Dashboard Stream"
          value={
            restFallback
              ? 'REST refresh'
              : streamConnected
                ? 'connected'
                : streamError
                  ? 'reconnecting'
                  : 'checking'
          }
          hint={restFallback ? 'SSE disabled under auth; REST polling active' : (streamError ?? undefined)}
        />
        <MetricTile label="Last Stream Event" value={formatUtc(lastStreamEventAt ?? undefined)} />
      </div>

      <StorageMonitoring stores={storage.data?.stores ?? []} loading={storage.isLoading} error={storage.isError ? storage.error : null} />
      <AdministrationInbox tenantId={TENANT_ID} data={notifications.data} loading={notifications.isLoading} error={notifications.isError ? notifications.error : null} mutate={mutateNotification} />
      <NotificationEmailSettings tenantId={TENANT_ID} />
      <h2 className="text-sm font-semibold text-gray-900">Scheduled Jobs</h2><div className="overflow-x-auto rounded border border-gray-200 bg-white"><table className="min-w-full divide-y divide-gray-200 text-xs"><thead className="bg-gray-50 text-left text-gray-500"><tr><th className="px-2 py-1">Job</th><th className="px-2 py-1">Schedule</th><th className="px-2 py-1">Status</th><th className="px-2 py-1">Started</th><th className="px-2 py-1">Completed</th><th className="px-2 py-1">Exit</th></tr></thead><tbody className="divide-y divide-gray-100">{scheduledJobs.data?.jobs.map(job => <tr key={job.job_id}><td className="px-2 py-1 font-medium">{job.label}</td><td className="px-2 py-1 text-gray-600">{job.schedule} · {job.timezone}</td><td className="px-2 py-1"><StatusBadge status={job.status} /></td><td className="px-2 py-1 text-gray-600">{formatUtc(job.started_at)}</td><td className="px-2 py-1 text-gray-600">{formatUtc(job.completed_at)}</td><td className="px-2 py-1">{job.exit_code ?? "—"}</td></tr>)}</tbody></table></div>{scheduledJobs.isError ? <ErrorState error={scheduledJobs.error} /> : null}<h2 className="text-sm font-semibold text-gray-900">MarketOps task control</h2><div className="overflow-x-auto rounded border border-gray-200 bg-white"><table className="min-w-full divide-y divide-gray-200 text-xs"><thead className="bg-gray-50 text-left text-gray-500"><tr><th className="px-2 py-1">Task</th><th className="px-2 py-1">Asset</th><th className="px-2 py-1">State</th><th className="px-2 py-1">Reason</th><th className="px-2 py-1">Retry</th></tr></thead><tbody className="divide-y divide-gray-100">{marketOpsTasks.data?.tasks.filter((task:any)=>task.status!=="succeeded").map((task:any)=><tr key={task.task_id}><td className="px-2 py-1">{task.task_type}</td><td className="px-2 py-1 font-mono">{task.symbol||"—"}</td><td className="px-2 py-1"><StatusBadge status={task.status}/></td><td className="px-2 py-1 text-gray-600">{task.failure_class||task.error_message||"—"}</td><td className="px-2 py-1 text-gray-600">{task.status==="retry_scheduled"?formatUtc(task.next_attempt_at):"—"}</td></tr>)}</tbody></table>{marketOpsTasks.data?.tasks.filter((task:any)=>task.status!=="succeeded").length===0?<div className="p-2 text-xs text-gray-500">No incomplete MarketOps tasks.</div>:null}</div>{marketOpsTasks.isError?<ErrorState error={marketOpsTasks.error}/>:null}<h2 className="text-sm font-semibold text-gray-900">Replay Operations</h2>
      <div className="grid grid-cols-2 gap-2 md:grid-cols-3">
        <MetricTile
          label="Replay Worker"
          value={replayStatus.isError ? 'unreachable' : replayWorstHealth}
        />
        <MetricTile
          label="Replay Queue"
          value={replayJobCount(replayStatusOk, 'queued')}
          hint={replayStatus.isError ? 'status unavailable' : undefined}
        />
        <MetricTile label="Replay Running" value={replayJobCount(replayStatusOk, 'running')} />
        <MetricTile label="Replay Failed" value={replayJobCount(replayStatusOk, 'failed')} />
        <MetricTile label="Replay Last Seen" value={formatUtc(replayLastSeen)} />
      </div>
      {replayStatus.isError ? (
        <ErrorState error={replayStatus.error} />
      ) : replayStatus.isLoading ? (
        <div className="text-sm text-gray-500">Loading replay worker status…</div>
      ) : replayWorkers.length ? (
        <div className="overflow-x-auto rounded border border-gray-200 bg-white">
          <table className="min-w-full divide-y divide-gray-200 text-xs">
            <thead className="bg-gray-50 text-left text-gray-500">
              <tr>
                <th className="px-2 py-1">Worker ID</th>
                <th className="px-2 py-1">Health</th>
                <th className="px-2 py-1">Status</th>
                <th className="px-2 py-1">Last seen</th>
                <th className="px-2 py-1">Last claimed</th>
                <th className="px-2 py-1">Last completed</th>
                <th className="px-2 py-1">Last error</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {replayWorkers.map((w) => (
                <tr key={w.worker_id}>
                  <td className="break-all px-2 py-1 font-mono">{w.worker_id}</td>
                  <td className="px-2 py-1"><StatusBadge status={w.health} /></td>
                  <td className="px-2 py-1"><StatusBadge status={w.status} /></td>
                  <td className="px-2 py-1 text-gray-600">{formatUtc(w.last_seen_at)}</td>
                  <td className="px-2 py-1">
                    {w.last_claimed_replay_job_id ? (
                      <>
                        <code className="break-all font-mono">{w.last_claimed_replay_job_id}</code>
                        <div className="text-gray-500">{formatUtc(w.last_claimed_at)}</div>
                      </>
                    ) : (
                      '—'
                    )}
                  </td>
                  <td className="px-2 py-1">
                    {w.last_completed_replay_job_id ? (
                      <>
                        <code className="break-all font-mono">{w.last_completed_replay_job_id}</code>
                        <div className="text-gray-500">{formatUtc(w.last_completed_at)}</div>
                      </>
                    ) : (
                      '—'
                    )}
                  </td>
                  <td className="px-2 py-1 text-red-700">
                    {w.last_error_message ? (
                      <>
                        <div className="break-all">{w.last_error_message}</div>
                        <div className="text-gray-500">{formatUtc(w.last_error_at)}</div>
                      </>
                    ) : (
                      '—'
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <EmptyState message="No replay worker heartbeat recorded." />
      )}
    </div>
  );
}

function bytes(value?: number) { if (value == null) return '—'; const units=['B','KB','MB','GB','TB']; let n=value,i=0; while(n>=1024&&i<units.length-1){n/=1024;i++} return `${n.toFixed(i ? 1 : 0)} ${units[i]}`; }
function StorageMonitoring({stores,loading,error}:{stores:any[];loading:boolean;error:unknown}) { return <section className="space-y-2"><div><h2 className="text-sm font-semibold text-gray-900">Persistent Storage</h2><p className="text-xs text-gray-500">SignalOps-owned PostgreSQL, TimescaleDB, and Redpanda volumes. Docker images, container layers, and host logs are excluded.</p></div>{loading?<div className="text-xs text-gray-500">Collecting storage snapshots…</div>:error?<ErrorState error={error}/>:<div className="grid gap-2 md:grid-cols-3">{stores.map((s:any)=><div key={s.store_id} className="rounded border border-gray-200 bg-white p-3"><div className="flex justify-between gap-2"><strong className="text-sm capitalize">{s.store_id}</strong><StatusBadge status={s.status}/></div>{s.used_bytes != null?<><div className="mt-2 text-lg font-semibold">{bytes(s.used_bytes)}</div><div className="text-xs text-gray-500">of {bytes(s.capacity_bytes)} · {Number(s.usage_percent ?? 0).toFixed(1)}% used</div><div className="mt-2 h-2 overflow-hidden rounded bg-gray-100"><div className={s.status==='critical'?'h-full bg-red-500':s.status==='warning'?'h-full bg-amber-400':'h-full bg-brand-600'} style={{width:`${Math.min(100,Number(s.usage_percent ?? 0))}%`}} /></div><div className="mt-2 text-[11px] text-gray-500">Free {bytes(s.free_bytes)} · {formatUtc(s.observed_at)}</div></>:<div className="mt-2 text-xs text-gray-500">{s.message ?? 'No snapshot recorded yet.'}</div>}</div>)}</div>}</section> }

function AdministrationInbox({ tenantId, data, loading, error, mutate }: { tenantId: string; data: any; loading: boolean; error: unknown; mutate: any }) {
  const notices = data?.notifications ?? [];
  return <section className="space-y-2"><div className="flex items-baseline justify-between gap-3"><div><h2 className="text-sm font-semibold text-gray-900 dark:text-gray-100">Administrator Inbox</h2><p className="text-xs text-gray-500 dark:text-gray-400">Governed job outcomes, repeated job incidents, and platform threshold events. This is separate from analyst alerts.</p></div><span className="rounded-full bg-brand-100 px-2 py-0.5 text-xs font-medium text-brand-800 dark:bg-brand-900/50 dark:text-brand-200">{data?.unread_count ?? 0} unread</span></div>{loading ? <div className="text-xs text-gray-500">Loading inbox…</div> : error ? <ErrorState error={error} /> : notices.length === 0 ? <EmptyState message="No administrator notifications yet." /> : <div className="overflow-x-auto rounded border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-900"><table className="min-w-full divide-y divide-gray-200 text-xs dark:divide-gray-700"><thead className="bg-gray-50 text-left text-gray-500 dark:bg-gray-800 dark:text-gray-300"><tr><th className="px-2 py-1">Severity</th><th className="px-2 py-1">Event</th><th className="px-2 py-1">Source</th><th className="px-2 py-1">Occurrences</th><th className="px-2 py-1">Last observed</th><th className="px-2 py-1" aria-label="Inbox actions" /></tr></thead><tbody className="divide-y divide-gray-100 dark:divide-gray-800">{notices.map((notice: any) => <tr key={notice.notification_id} className={notice.read_at ? 'text-gray-500 dark:text-gray-400' : 'bg-brand-50/40 dark:bg-brand-950/20'}><td className="px-2 py-2"><span className={notice.severity === 'critical' ? 'font-medium text-red-700 dark:text-red-300' : notice.severity === 'warning' ? 'font-medium text-amber-700 dark:text-amber-300' : 'font-medium text-brand-700 dark:text-brand-300'}>{notice.severity}</span></td><td className="max-w-xl px-2 py-2"><div className="font-medium text-gray-900 dark:text-gray-100">{notice.title}</div><div className="mt-0.5 text-gray-600 dark:text-gray-400">{notice.summary}</div></td><td className="px-2 py-2 font-mono text-gray-600 dark:text-gray-400">{notice.source}</td><td className="px-2 py-2">{notice.occurrence_count}</td><td className="whitespace-nowrap px-2 py-2">{formatUtc(notice.last_occurred_at)}</td><td className="whitespace-nowrap px-2 py-2"><button type="button" disabled={mutate.isPending} onClick={() => mutate.mutate({ id: notice.notification_id, read: true, archived: !notice.archived_at })} className="rounded border border-gray-300 bg-white px-2 py-1 text-[11px] font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-200 dark:hover:bg-gray-700">{notice.archived_at ? 'Restore' : 'Archive'}</button></td></tr>)}</tbody></table></div>}</section>;
}

function NotificationEmailSettings({ tenantId }: { tenantId: string }) {
  const settings = useAdministrationSMTPSettings(tenantId);
  const save = useMutateAdministrationSMTPSettings(tenantId);
  const [form, setForm] = useState({ host: '', port: 587, username: '', password: '', use_starttls: true, use_ssl: false, from_email: '', from_name: 'SignalOps', reply_to: '' });
  useEffect(() => { const x = settings.data?.settings; if (x) setForm({ host:x.host, port:x.port, username:x.username, password:'', use_starttls:x.use_starttls, use_ssl:x.use_ssl, from_email:x.from_email, from_name:x.from_name, reply_to:x.reply_to }); }, [settings.data]);
  return <section className="space-y-2 rounded border border-gray-200 bg-white p-3 dark:border-gray-700 dark:bg-gray-900"><div><h2 className="text-sm font-semibold text-gray-900 dark:text-gray-100">Notification email</h2><p className="text-xs text-gray-500 dark:text-gray-400">Credentials are encrypted by the gateway before storage. Leave password blank to retain the existing secret. Delivery activation follows configuration and recipient policy setup.</p></div>{settings.isError ? <ErrorState error={settings.error} /> : <form onSubmit={(event) => { event.preventDefault(); save.mutate({ tenant_id:tenantId, ...form }); }} className="flex flex-wrap items-end gap-2"><label className="text-xs text-gray-700 dark:text-gray-300">SMTP host<input required value={form.host} onChange={e=>setForm({...form,host:e.target.value})} className="mt-1 block w-48 rounded border border-gray-300 bg-white px-2 py-1 text-sm dark:border-gray-600 dark:bg-gray-800" /></label><label className="text-xs text-gray-700 dark:text-gray-300">Port<input required type="number" min="1" max="65535" value={form.port} onChange={e=>setForm({...form,port:Number(e.target.value)})} className="mt-1 block w-20 rounded border border-gray-300 bg-white px-2 py-1 text-sm dark:border-gray-600 dark:bg-gray-800" /></label><label className="text-xs text-gray-700 dark:text-gray-300">Username<input value={form.username} onChange={e=>setForm({...form,username:e.target.value})} className="mt-1 block w-40 rounded border border-gray-300 bg-white px-2 py-1 text-sm dark:border-gray-600 dark:bg-gray-800" /></label><label className="text-xs text-gray-700 dark:text-gray-300">Password {settings.data?.settings?.has_password ? '(saved)' : ''}<input type="password" value={form.password} onChange={e=>setForm({...form,password:e.target.value})} className="mt-1 block w-40 rounded border border-gray-300 bg-white px-2 py-1 text-sm dark:border-gray-600 dark:bg-gray-800" /></label><label className="text-xs text-gray-700 dark:text-gray-300">From email<input required type="email" value={form.from_email} onChange={e=>setForm({...form,from_email:e.target.value})} className="mt-1 block w-48 rounded border border-gray-300 bg-white px-2 py-1 text-sm dark:border-gray-600 dark:bg-gray-800" /></label><label className="text-xs text-gray-700 dark:text-gray-300">From name<input value={form.from_name} onChange={e=>setForm({...form,from_name:e.target.value})} className="mt-1 block w-32 rounded border border-gray-300 bg-white px-2 py-1 text-sm dark:border-gray-600 dark:bg-gray-800" /></label><label className="inline-flex items-center gap-1 text-xs text-gray-700 dark:text-gray-300"><input type="checkbox" checked={form.use_starttls} onChange={e=>setForm({...form,use_starttls:e.target.checked,use_ssl:e.target.checked?false:form.use_ssl})} />STARTTLS</label><label className="inline-flex items-center gap-1 text-xs text-gray-700 dark:text-gray-300"><input type="checkbox" checked={form.use_ssl} onChange={e=>setForm({...form,use_ssl:e.target.checked,use_starttls:e.target.checked?false:form.use_starttls})} />SSL</label><button disabled={save.isPending} className="rounded bg-brand-700 px-3 py-1.5 text-xs font-medium text-white disabled:opacity-50">{save.isPending ? 'Saving…' : 'Save email settings'}</button>{save.isError ? <span className="text-xs text-red-700">Unable to save configuration.</span> : save.isSuccess ? <span className="text-xs text-green-700">Saved.</span> : null}</form>}</section>;
}

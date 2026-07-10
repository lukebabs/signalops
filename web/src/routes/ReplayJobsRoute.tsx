import { useState } from 'react';
import { History, Plus, Ban } from 'lucide-react';
import { useReplayJobs, useReplayJob, useCreateReplayJob, useCancelReplayJob } from '../api/queries';
import { isApiError } from '../api/client';
import { LoadingState, ErrorState, EmptyState } from '../components/States';
import { MetricTile } from '../components/MetricTile';
import { StatusBadge } from '../components/StatusBadge';
import { CopyButton } from '../components/CopyButton';
import { JsonViewer } from '../components/JsonViewer';
import { RefreshButton } from '../components/RefreshButton';
import { formatUtc, duration, orDash, toRfc3339Utc, toDatetimeLocal } from '../lib/format';
import { isCancelableStatus, cancellationOf, replayRecords } from '../lib/replay';
import type { ReplayJob, ReplaySourceKind, ReplayMode, ReplayJobStatus } from '../types';
import { useTenant, useActor } from '../auth/session';

const SOURCE_KINDS: ReplaySourceKind[] = ['raw_events', 'normalized_events', 'signals'];
const REPLAY_MODES: ReplayMode[] = ['original', 'latest_compatible', 'explicit'];
const STATUSES: ReplayJobStatus[] = ['queued', 'running', 'succeeded', 'failed', 'canceled'];
const LIMITS = [25, 50, 100, 200];

// result is raw JSON; pull a numeric counter defensively (number or numeric string).
function numField(obj: unknown, key: string): number | undefined {
  if (!obj || typeof obj !== 'object') return undefined;
  const v = (obj as Record<string, unknown>)[key];
  if (typeof v === 'number' && Number.isFinite(v)) return v;
  if (typeof v === 'string' && v !== '') {
    const n = Number(v);
    if (Number.isFinite(n)) return n;
  }
  return undefined;
}

function createdLabel(status: string): string {
  // Creating a job only queues it; only say "Started" if the backend reports running.
  return status === 'running' ? 'Started replay' : 'Queued';
}

export function ReplayJobsRoute() {
  const TENANT_ID = useTenant();
  const actor = useActor();

  // List filters.
  const [status, setStatus] = useState<ReplayJobStatus | ''>('');
  const [sourceKind, setSourceKind] = useState<ReplaySourceKind | ''>('');
  const [sourceId, setSourceId] = useState('');
  const [dataset, setDataset] = useState('');
  const [limit, setLimit] = useState(50);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const list = useReplayJobs({
    tenant_id: TENANT_ID,
    source_id: sourceId || undefined,
    dataset: dataset || undefined,
    source_kind: sourceKind || undefined,
    status: status || undefined,
    limit,
  });
  const detail = useReplayJob(selectedId ?? undefined);
  const create = useCreateReplayJob();
  const data = list.data?.replay_jobs ?? [];

  const queued = data.filter((j) => j.status === 'queued').length;
  const running = data.filter((j) => j.status === 'running').length;
  const failed = data.filter((j) => j.status === 'failed').length;
  const canceled = data.filter((j) => j.status === 'canceled').length;
  const published = data.reduce((n, j) => n + (numField(j.result, 'published') ?? 0), 0);

  // Create-form state. Windows default to the last 24h (UTC wall-clock).
  const [fSourceKind, setFSourceKind] = useState<ReplaySourceKind>('raw_events');
  const [fMode, setFMode] = useState<ReplayMode>('original');
  const [fStart, setFStart] = useState(() => {
    const d = new Date();
    d.setDate(d.getDate() - 1);
    return toDatetimeLocal(d.toISOString());
  });
  const [fEnd, setFEnd] = useState(() => toDatetimeLocal(new Date().toISOString()));
  const [fSourceId, setFSourceId] = useState('');
  const [fDataset, setFDataset] = useState('');
  const [fMax, setFMax] = useState(10);
  const [errors, setErrors] = useState<Record<string, string>>({});

  // Clear stale create result/error and the touched field's error on edit.
  function touch(key?: string) {
    create.reset();
    setErrors((e) => (key ? { ...e, [key]: '' } : {}));
  }

  function validate(): Record<string, string> {
    const e: Record<string, string> = {};
    if (!fStart.trim()) e.start = 'Required';
    if (!fEnd.trim()) e.end = 'Required';
    if (!e.start && !e.end) {
      const st = new Date(toRfc3339Utc(fStart)).getTime();
      const et = new Date(toRfc3339Utc(fEnd)).getTime();
      if (isNaN(st) || isNaN(et)) {
        e.end = 'Invalid datetime';
      } else if (et <= st) {
        e.end = 'window_end must be after window_start';
      }
    }
    if (!Number.isInteger(fMax) || fMax < 1 || fMax > 200) e.max = '1–200';
    return e;
  }

  function onSubmit(ev: React.FormEvent) {
    ev.preventDefault();
    const e = validate();
    setErrors(e);
    if (Object.values(e).some(Boolean)) return;
    create.mutate(
      {
        tenant_id: TENANT_ID,
        source_kind: fSourceKind,
        replay_mode: fMode,
        requested_by: actor,
        window_start: toRfc3339Utc(fStart),
        window_end: toRfc3339Utc(fEnd),
        source_id: fSourceId || undefined,
        dataset: fDataset || undefined,
        filters: {},
        options: { max_records: fMax },
      },
      { onSuccess: (d) => setSelectedId(d.replay_job.replay_job_id) },
    );
  }

  const inputCls = 'rounded border border-gray-300 px-2 py-1 text-sm';
  const labelCls = 'text-xs text-gray-500';

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <History size={18} className="text-brand-700" />
        <h1 className="text-lg font-semibold">Replay Jobs</h1>
      </div>

      <div className="grid grid-cols-2 gap-2 md:grid-cols-6">
        <MetricTile label="Replay Jobs" value={data.length} />
        <MetricTile label="Queued" value={queued} />
        <MetricTile label="Running" value={running} />
        <MetricTile label="Failed" value={failed} />
        <MetricTile label="Canceled" value={canceled} />
        <MetricTile label="Published" value={published} hint={list.isError ? 'unreachable' : undefined} />
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <select
          value={status}
          onChange={(e) => setStatus(e.target.value as ReplayJobStatus | '')}
          className={inputCls}
          aria-label="Filter by status"
        >
          <option value="">any status</option>
          {STATUSES.map((s) => (
            <option key={s} value={s}>{s}</option>
          ))}
        </select>
        <select
          value={sourceKind}
          onChange={(e) => setSourceKind(e.target.value as ReplaySourceKind | '')}
          className={inputCls}
          aria-label="Filter by source kind"
        >
          <option value="">any source kind</option>
          {SOURCE_KINDS.map((s) => (
            <option key={s} value={s}>{s}</option>
          ))}
        </select>
        <input
          placeholder="source id"
          value={sourceId}
          onChange={(e) => setSourceId(e.target.value)}
          className={inputCls}
        />
        <input
          placeholder="dataset"
          value={dataset}
          onChange={(e) => setDataset(e.target.value)}
          className={inputCls}
        />
        <select
          value={limit}
          onChange={(e) => setLimit(Number(e.target.value))}
          className={inputCls}
          aria-label="Page limit"
        >
          {LIMITS.map((n) => (
            <option key={n} value={n}>{n}</option>
          ))}
        </select>
        <RefreshButton onClick={() => list.refetch()} loading={list.isFetching} />
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        {/* Table — left/span-2 on desktop; below the form on mobile. */}
        <div className="order-2 lg:order-none lg:col-span-2">
          {list.isLoading ? (
            <LoadingState />
          ) : list.isError ? (
            <ErrorState error={list.error} />
          ) : data.length ? (
            <div className="overflow-x-auto rounded border border-gray-200 bg-white">
              <table className="min-w-full divide-y divide-gray-200 text-sm">
                <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500">
                  <tr>
                    <th className="px-3 py-2">Job</th>
                    <th className="px-3 py-2">Source</th>
                    <th className="px-3 py-2">Window</th>
                    <th className="px-3 py-2">Mode</th>
                    <th className="px-3 py-2">Status</th>
                    <th className="px-3 py-2">Result</th>
                    <th className="px-3 py-2">Updated</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {data.map((j) => {
                    const scanned = numField(j.result, 'scanned');
                    const pub = numField(j.result, 'published');
                    const max = numField(j.options, 'max_records');
                    return (
                      <tr
                        key={j.replay_job_id}
                        onClick={() => setSelectedId(j.replay_job_id)}
                        className={`cursor-pointer align-top hover:bg-gray-50 ${selectedId === j.replay_job_id ? 'bg-brand-50' : ''}`}
                      >
                        <td className="px-3 py-2">
                          <div className="font-mono text-xs text-gray-800">{j.replay_job_id}</div>
                          <div className="text-xs text-gray-500">{j.requested_by || '—'}</div>
                          <div className="text-xs text-gray-500">{formatUtc(j.created_at)}</div>
                        </td>
                        <td className="px-3 py-2 text-xs">
                          <div className="font-mono">{j.source_kind}</div>
                          <div className="font-mono text-gray-600">{j.source_id || '—'}</div>
                          <div className="text-gray-600">{j.dataset || '—'}</div>
                        </td>
                        <td className="px-3 py-2 text-xs text-gray-600">
                          <div>{formatUtc(j.window_start)}</div>
                          <div>{formatUtc(j.window_end)}</div>
                        </td>
                        <td className="px-3 py-2 text-xs font-mono">{j.replay_mode}</td>
                        <td className="px-3 py-2"><StatusBadge status={j.status} /></td>
                        <td className="px-3 py-2 text-xs text-gray-600">
                          {scanned !== undefined || pub !== undefined ? (
                            <span>{orDash(scanned)}/{orDash(pub)}{max !== undefined ? ` · max ${max}` : ''}</span>
                          ) : (
                            <span className="text-gray-400">—</span>
                          )}
                        </td>
                        <td className="px-3 py-2 text-xs text-gray-600">{formatUtc(j.updated_at)}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          ) : (
            <EmptyState message="No replay jobs found." />
          )}
        </div>

        {/* Right column: create form + detail. Above the table on mobile. */}
        <div className="order-1 space-y-4 lg:order-none lg:col-span-1">
          <form
            onSubmit={onSubmit}
            className="space-y-2 rounded border border-gray-200 bg-white p-3"
            aria-label="Create replay job"
          >
            <div className="flex items-center gap-1 text-sm font-semibold text-gray-900">
              <Plus size={14} /> New Replay Job
            </div>

            <div className="grid grid-cols-2 gap-2">
              <label className="block">
                <span className={labelCls}>Source kind</span>
                <select
                  value={fSourceKind}
                  onChange={(e) => { touch(); setFSourceKind(e.target.value as ReplaySourceKind); }}
                  className={`${inputCls} mt-0.5 w-full`}
                >
                  {SOURCE_KINDS.map((s) => (
                    <option key={s} value={s}>{s}</option>
                  ))}
                </select>
              </label>
              <label className="block">
                <span className={labelCls}>Replay mode</span>
                <select
                  value={fMode}
                  onChange={(e) => { touch(); setFMode(e.target.value as ReplayMode); }}
                  className={`${inputCls} mt-0.5 w-full`}
                >
                  {REPLAY_MODES.map((m) => (
                    <option key={m} value={m}>{m}</option>
                  ))}
                </select>
              </label>
            </div>

            <div className="grid grid-cols-2 gap-2">
              <label className="block">
                <span className={labelCls}>Window start <span className="text-gray-400">(UTC)</span></span>
                <input
                  type="datetime-local"
                  value={fStart}
                  onChange={(e) => { touch('start'); setFStart(e.target.value); }}
                  className={`${inputCls} mt-0.5 w-full`}
                  aria-invalid={Boolean(errors.start)}
                />
                {errors.start && <span className="text-xs text-red-700">{errors.start}</span>}
              </label>
              <label className="block">
                <span className={labelCls}>Window end <span className="text-gray-400">(UTC)</span></span>
                <input
                  type="datetime-local"
                  value={fEnd}
                  onChange={(e) => { touch('end'); setFEnd(e.target.value); }}
                  className={`${inputCls} mt-0.5 w-full`}
                  aria-invalid={Boolean(errors.end)}
                />
                {errors.end && <span className="text-xs text-red-700">{errors.end}</span>}
              </label>
            </div>

            <div className="grid grid-cols-2 gap-2">
              <label className="block">
                <span className={labelCls}>Source id <span className="text-gray-400">(optional)</span></span>
                <input
                  value={fSourceId}
                  onChange={(e) => { touch(); setFSourceId(e.target.value); }}
                  placeholder="src-…"
                  className={`${inputCls} mt-0.5 w-full`}
                />
              </label>
              <label className="block">
                <span className={labelCls}>Dataset <span className="text-gray-400">(optional)</span></span>
                <input
                  value={fDataset}
                  onChange={(e) => { touch(); setFDataset(e.target.value); }}
                  placeholder="dataset"
                  className={`${inputCls} mt-0.5 w-full`}
                />
              </label>
            </div>

            <label className="block">
              <span className={labelCls}>Max records (1–200)</span>
              <input
                type="number"
                min={1}
                max={200}
                value={fMax}
                onChange={(e) => { touch('max'); setFMax(Number(e.target.value)); }}
                className={`${inputCls} mt-0.5 w-full`}
                aria-invalid={Boolean(errors.max)}
              />
              {errors.max && <span className="text-xs text-red-700">{errors.max}</span>}
            </label>

            <button
              type="submit"
              disabled={create.isPending}
              className="inline-flex w-full items-center justify-center gap-1 rounded bg-brand-500 px-3 py-1.5 text-sm text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-50"
            >
              <Plus size={14} /> {create.isPending ? 'Queuing…' : 'Queue replay job'}
            </button>

            {create.isSuccess && create.data && (
              <div className="rounded border border-green-200 bg-green-50 p-2 text-xs text-green-800">
                <div className="flex flex-wrap items-center gap-2">
                  <StatusBadge status={create.data.replay_job.status} />
                  <span>{createdLabel(create.data.replay_job.status)}</span>
                  <code className="font-mono">{create.data.replay_job.replay_job_id}</code>
                </div>
                <p className="mt-1 text-green-700">The worker runs separately; status updates on refresh.</p>
              </div>
            )}
            {create.isError && (
              <p className="text-xs text-red-700" role="alert">
                Create failed: {isApiError(create.error) ? create.error.message : 'unknown error'}. List and detail preserved.
              </p>
            )}
          </form>

          <div className="rounded border border-gray-200 bg-white p-3">
            {!selectedId ? (
              <EmptyState message="Select a replay job to inspect details." />
            ) : detail.isLoading ? (
              <LoadingState />
            ) : detail.isError ? (
              <ErrorState error={detail.error} />
            ) : detail.data ? (
              <ReplayJobDetail key={detail.data.replay_job.replay_job_id} job={detail.data.replay_job} />
            ) : null}
          </div>
        </div>
      </div>
    </div>
  );
}

function ReplayJobDetail({ job }: { job: ReplayJob }) {
  const inFlight = job.status === 'queued' || job.status === 'running';
  const cancelable = isCancelableStatus(job.status);
  const cancel = useCancelReplayJob();
  const [confirming, setConfirming] = useState(false);
  const [reason, setReason] = useState('operator canceled from Replay UI');
  const cancelInFlight = cancel.isPending && cancel.variables?.replayJobId === job.replay_job_id;

  const scanned = numField(job.result, 'scanned');
  const published = numField(job.result, 'published');
  const failed = numField(job.result, 'failed');
  const batches = numField(job.result, 'batches');
  const batchSize = numField(job.result, 'batch_size');
  const maxRecords = numField(job.options, 'max_records');
  const cancelMeta = cancellationOf(job.result);
  const records = replayRecords(job.result);

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <StatusBadge status={job.status} />
        <span className="text-xs font-mono text-gray-600">{job.replay_mode}</span>
        <code className="break-all text-xs text-gray-700">{job.replay_job_id}</code>
        <CopyButton value={job.replay_job_id} />
      </div>

      {cancelable && (
        <div className="rounded border border-gray-200 bg-gray-50 p-2">
          {!confirming ? (
            <button
              type="button"
              onClick={() => setConfirming(true)}
              className="inline-flex items-center gap-1 rounded border border-red-300 bg-white px-2 py-1 text-xs text-red-700 hover:bg-red-50"
            >
              <Ban size={14} /> Cancel
            </button>
          ) : (
            <div className="space-y-2">
              <div className="text-xs text-gray-700">Cancel replay job?</div>
              <input
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                placeholder="reason"
                className="w-full rounded border border-gray-300 px-2 py-1 text-xs"
              />
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  disabled={cancelInFlight}
                  onClick={() =>
                    cancel.mutate({ replayJobId: job.replay_job_id, reason: reason.trim() || undefined })
                  }
                  className="inline-flex items-center gap-1 rounded border border-red-300 bg-red-100 px-2 py-1 text-xs text-red-800 hover:bg-red-200 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <Ban size={14} /> {cancelInFlight ? 'Canceling…' : 'Confirm cancel'}
                </button>
                <button
                  type="button"
                  onClick={() => { setConfirming(false); cancel.reset(); }}
                  className="inline-flex items-center gap-1 rounded border border-gray-300 bg-white px-2 py-1 text-xs text-gray-700 hover:bg-gray-50"
                >
                  Dismiss
                </button>
              </div>
              {cancel.isError && (
                <p className="text-xs text-red-700" role="alert">
                  Cancel failed: {isApiError(cancel.error) ? cancel.error.message : 'unknown error'}. Status preserved.
                </p>
              )}
            </div>
          )}
        </div>
      )}

      {inFlight && (
        <p className="text-xs text-gray-500">Replay in progress — detail refreshes every few seconds.</p>
      )}
      {job.error_message && (
        <div className="rounded border border-red-200 bg-red-50 p-2 text-xs text-red-800" role="alert">
          <span className="font-medium">Error:</span> {job.error_message}
        </div>
      )}

      <div className="grid grid-cols-2 gap-2 text-sm">
        <div><div className="text-xs text-gray-500">Tenant</div><div className="text-xs font-mono">{job.tenant_id}</div></div>
        <div><div className="text-xs text-gray-500">Source kind</div><div className="text-xs font-mono">{job.source_kind}</div></div>
        <div><div className="text-xs text-gray-500">Source id</div><div className="text-xs font-mono">{job.source_id || '—'}</div></div>
        <div><div className="text-xs text-gray-500">Dataset</div><div className="text-xs">{job.dataset || '—'}</div></div>
        <div><div className="text-xs text-gray-500">Requested by</div><div className="text-xs">{job.requested_by || '—'}</div></div>
        <div><div className="text-xs text-gray-500">Mode</div><div className="text-xs font-mono">{job.replay_mode}</div></div>
        <div><div className="text-xs text-gray-500">Window start</div><div className="text-xs">{formatUtc(job.window_start)}</div></div>
        <div><div className="text-xs text-gray-500">Window end</div><div className="text-xs">{formatUtc(job.window_end)}</div></div>
        <div><div className="text-xs text-gray-500">Created</div><div className="text-xs">{formatUtc(job.created_at)}</div></div>
        <div><div className="text-xs text-gray-500">Updated</div><div className="text-xs">{formatUtc(job.updated_at)}</div></div>
        <div><div className="text-xs text-gray-500">Started</div><div className="text-xs">{formatUtc(job.started_at)}</div></div>
        <div><div className="text-xs text-gray-500">Completed</div><div className="text-xs">{formatUtc(job.completed_at)}</div></div>
      </div>

      {job.started_at && job.completed_at && (
        <div className="text-xs text-gray-500">Duration: {duration(job.started_at, job.completed_at)}</div>
      )}

      <div className="grid grid-cols-3 gap-2 text-sm">
        <div><div className="text-xs text-gray-500">Scanned</div><div>{orDash(scanned)}</div></div>
        <div><div className="text-xs text-gray-500">Published</div><div>{orDash(published)}</div></div>
        <div><div className="text-xs text-gray-500">Failed</div><div>{orDash(failed)}</div></div>
        <div><div className="text-xs text-gray-500">Batches</div><div>{orDash(batches)}</div></div>
        <div><div className="text-xs text-gray-500">Batch size</div><div>{orDash(batchSize)}</div></div>
        <div><div className="text-xs text-gray-500">Max records</div><div>{orDash(maxRecords)}</div></div>
      </div>

      {(job.status === 'canceled' || cancelMeta) && (
        <div className="rounded border border-gray-200 bg-gray-50 p-2 text-xs">
          <div className="mb-1 text-gray-600">Cancellation</div>
          <div className="grid grid-cols-3 gap-2">
            <div><div className="text-gray-400">Actor</div><div className="break-words">{cancelMeta?.actor || '—'}</div></div>
            <div><div className="text-gray-400">Reason</div><div className="break-words">{cancelMeta?.reason || '—'}</div></div>
            <div><div className="text-gray-400">Canceled at</div><div>{formatUtc(cancelMeta?.canceled_at)}</div></div>
          </div>
        </div>
      )}

      {records.length > 0 && (
        <div>
          <div className="mb-1 text-xs font-medium text-gray-600">Records ({records.length})</div>
          <div className="max-h-64 overflow-auto rounded border border-gray-200">
            <table className="min-w-full divide-y divide-gray-200 text-xs">
              <thead className="sticky top-0 bg-gray-50 text-left text-gray-500">
                <tr>
                  <th className="px-2 py-1">Status</th>
                  <th className="px-2 py-1">Source ID</th>
                  <th className="px-2 py-1">Key</th>
                  <th className="px-2 py-1">Attempts</th>
                  <th className="px-2 py-1">Topic</th>
                  <th className="px-2 py-1">Partition</th>
                  <th className="px-2 py-1">Offset</th>
                  <th className="px-2 py-1">Error</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {records.map((rec, i) => (
                  <tr key={rec.key || rec.source_id || i}>
                    <td className="px-2 py-1"><StatusBadge status={rec.status} /></td>
                    <td className="break-all px-2 py-1 font-mono">{rec.source_id || '—'}</td>
                    <td className="max-w-[10rem] break-all px-2 py-1 font-mono">{rec.key || '—'}</td>
                    <td className="px-2 py-1">{orDash(rec.attempts)}</td>
                    <td className="break-all px-2 py-1 font-mono">{rec.topic || '—'}</td>
                    <td className="px-2 py-1">{orDash(rec.partition)}</td>
                    <td className="px-2 py-1">{orDash(rec.offset)}</td>
                    <td className="break-all px-2 py-1 text-red-700">{rec.error || '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      <JsonViewer label="Filters" value={job.filters} />
      <JsonViewer label="Options" value={job.options} />
      <JsonViewer label="Result" value={job.result} />
    </div>
  );
}

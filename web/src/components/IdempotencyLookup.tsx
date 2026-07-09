import { useState, type FormEvent } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { useIdempotency } from '../api/queries';
import { isApiError } from '../api/client';
import { useUi } from '../store/ui';
import { MetricTile } from './MetricTile';
import { JsonViewer } from './JsonViewer';
import { CopyButton } from './CopyButton';
import { LoadingState, ErrorState } from './States';
import { formatUtc, orDash } from '../lib/format';
import { useTenant } from '../auth/session';

export function IdempotencyLookup() {
  const navigate = useNavigate();
  const setSelectedEventId = useUi((s) => s.setSelectedEventId);
  const tenant = useTenant();
  const [tenantId, setTenantId] = useState(tenant);
  const [sourceId, setSourceId] = useState('src-massive');
  const [key, setKey] = useState('');
  const [submitted, setSubmitted] = useState(false);

  const canSubmit = !!tenantId.trim() && !!sourceId.trim() && !!key.trim();
  // Manual lookup: query is disabled; fetch only on submit via refetch.
  const q = useIdempotency(tenantId.trim(), sourceId.trim(), key.trim(), false);

  function lookup(e: FormEvent) {
    e.preventDefault();
    if (!canSubmit) return;
    setSubmitted(true);
    q.refetch();
  }

  function reset() {
    setSubmitted(false);
  }

  function viewRawEvent() {
    if (q.data) {
      setSelectedEventId(q.data.idempotency.event_id);
      navigate({ to: '/raw-events' });
    }
  }

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-semibold">Idempotency Lookup</h1>
      <form onSubmit={lookup} className="flex flex-wrap items-end gap-2">
        <label className="text-xs text-gray-600">
          Tenant ID
          <input
            value={tenantId}
            onChange={(e) => {
              setTenantId(e.target.value);
              reset();
            }}
            className="ml-1 block rounded border border-gray-300 px-2 py-1 text-sm"
          />
        </label>
        <label className="text-xs text-gray-600">
          Source ID
          <input
            value={sourceId}
            onChange={(e) => {
              setSourceId(e.target.value);
              reset();
            }}
            className="ml-1 block rounded border border-gray-300 px-2 py-1 text-sm"
          />
        </label>
        <label className="text-xs text-gray-600">
          Idempotency Key
          <input
            value={key}
            onChange={(e) => {
              setKey(e.target.value);
              reset();
            }}
            placeholder="idem_..."
            className="ml-1 block rounded border border-gray-300 px-2 py-1 text-sm"
          />
        </label>
        <button
          type="submit"
          disabled={!canSubmit}
          className="rounded bg-brand-500 px-3 py-1 text-sm text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-50"
        >
          Lookup
        </button>
      </form>

      <div>
        {submitted && q.isFetching && <LoadingState label="Looking up…" />}
        {submitted && q.isError && !q.isFetching && (
          isApiError(q.error) && q.error.status === 404 ? (
            <div className="rounded border border-gray-200 bg-gray-50 p-4 text-sm text-gray-600">
              No idempotency record found.
            </div>
          ) : (
            <ErrorState error={q.error} />
          )
        )}
        {submitted && q.data && !q.isFetching && (
          <div className="space-y-3">
            <div className="grid grid-cols-2 gap-2 md:grid-cols-3">
              <MetricTile label="Status" value={q.data.idempotency.status} />
              <MetricTile label="Event ID" value={q.data.idempotency.event_id} />
              <MetricTile label="Dataset" value={q.data.idempotency.dataset} />
              <MetricTile label="Topic" value={orDash(q.data.idempotency.topic)} />
              <MetricTile label="Partition" value={orDash(q.data.idempotency.partition)} />
              <MetricTile label="Offset" value={orDash(q.data.idempotency.offset)} />
              <MetricTile label="First Seen" value={formatUtc(q.data.idempotency.first_seen_at)} />
              <MetricTile label="Last Seen" value={formatUtc(q.data.idempotency.last_seen_at)} />
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-xs font-medium text-gray-600">Payload Hash</span>
              <code className="break-all text-xs text-gray-700">
                {orDash(q.data.idempotency.payload_hash)}
              </code>
              {q.data.idempotency.payload_hash && (
                <CopyButton value={q.data.idempotency.payload_hash} />
              )}
            </div>
            <JsonViewer label="Metadata" value={q.data.idempotency.metadata} />
            <button
              type="button"
              onClick={viewRawEvent}
              className="rounded border border-brand-500 px-3 py-1 text-sm text-brand-700 hover:bg-brand-50"
            >
              View raw event
            </button>
          </div>
        )}
        {!submitted && (
          <div className="p-4 text-sm text-gray-500">
            Enter tenant, source, and idempotency key, then Lookup.
          </div>
        )}
      </div>
    </div>
  );
}

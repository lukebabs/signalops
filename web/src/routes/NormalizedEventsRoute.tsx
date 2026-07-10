import { useState } from 'react';
import { FileCheck2 } from 'lucide-react';
import { useNormalizedEvents, useNormalizedEvent } from '../api/queries';
import { useAppProfile } from '../apps/AppProfileContext';
import { LoadingState, ErrorState, EmptyState } from '../components/States';
import { MetricTile } from '../components/MetricTile';
import { CopyButton } from '../components/CopyButton';
import { JsonViewer } from '../components/JsonViewer';
import { formatUtc } from '../lib/format';
import type { NormalizedEvent } from '../types';
import { useTenant } from '../auth/session';

export function NormalizedEventsRoute() {
  const TENANT_ID = useTenant();
  const { metadataFilter } = useAppProfile();
  const [sourceId, setSourceId] = useState('');
  const [dataset, setDataset] = useState('');
  const [limit, setLimit] = useState(50);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const list = useNormalizedEvents({
    tenant_id: TENANT_ID,
    source_id: sourceId || undefined,
    dataset: dataset || undefined,
    limit,
    ...metadataFilter,
  });
  const detail = useNormalizedEvent(selectedId);
  const data = list.data?.normalized_events ?? [];

  const distinctSources = new Set(data.map((e) => e.source_id)).size;
  const distinctDatasets = new Set(data.map((e) => e.dataset)).size;
  const avgConfidence = data.length ? data.reduce((n, e) => n + e.confidence, 0) / data.length : 0;

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h1 className="text-lg font-semibold">Normalized Events</h1>
        <div className="flex flex-wrap items-center gap-2">
          <input
            placeholder="source id"
            value={sourceId}
            onChange={(e) => setSourceId(e.target.value)}
            className="rounded border border-gray-300 px-2 py-1 text-sm"
          />
          <input
            placeholder="dataset"
            value={dataset}
            onChange={(e) => setDataset(e.target.value)}
            className="rounded border border-gray-300 px-2 py-1 text-sm"
          />
          <select
            value={limit}
            onChange={(e) => setLimit(Number(e.target.value))}
            className="rounded border border-gray-300 px-2 py-1 text-sm"
          >
            {[25, 50, 100, 200].map((n) => (
              <option key={n} value={n}>{n}</option>
            ))}
          </select>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-2 md:grid-cols-4">
        <MetricTile label="Normalized Events" value={data.length} />
        <MetricTile label="Distinct Sources" value={distinctSources} />
        <MetricTile label="Distinct Datasets" value={distinctDatasets} />
        <MetricTile label="Avg Confidence" value={avgConfidence.toFixed(2)} />
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div className="lg:col-span-2">
          {list.isLoading ? (
            <LoadingState />
          ) : list.isError ? (
            <ErrorState error={list.error} />
          ) : data.length ? (
            <div className="overflow-x-auto rounded border border-gray-200 bg-white">
              <table className="min-w-full divide-y divide-gray-200 text-sm">
                <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500">
                  <tr>
                    <th className="px-3 py-2">Event</th>
                    <th className="px-3 py-2">Source</th>
                    <th className="px-3 py-2">Dataset</th>
                    <th className="px-3 py-2">Observed</th>
                    <th className="px-3 py-2">Confidence</th>
                    <th className="px-3 py-2">Raw Broker</th>
                    <th className="px-3 py-2">Normalized Broker</th>
                    <th className="px-3 py-2">Created</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {data.map((e) => (
                    <tr
                      key={e.event_id}
                      onClick={() => setSelectedId(e.event_id)}
                      className={`cursor-pointer align-top hover:bg-gray-50 ${selectedId === e.event_id ? 'bg-brand-50' : ''}`}
                    >
                      <td className="px-3 py-2">
                        <div className="flex items-start gap-2">
                          <FileCheck2 size={16} className="mt-0.5 text-brand-700" />
                          <div>
                            <div className="font-mono text-xs text-gray-700">{e.event_id}</div>
                            <div className="text-xs text-gray-500">{e.schema_id} v{e.schema_version}</div>
                          </div>
                        </div>
                      </td>
                      <td className="px-3 py-2 text-xs">
                        <div className="font-mono">{e.source_adapter}</div>
                        <div className="text-gray-500">{e.source_id}</div>
                      </td>
                      <td className="px-3 py-2 text-xs">{e.dataset}</td>
                      <td className="px-3 py-2 text-xs text-gray-600">{formatUtc(e.observation_time)}</td>
                      <td className="px-3 py-2 text-xs">{e.confidence.toFixed(2)}</td>
                      <td className="px-3 py-2 text-xs font-mono">{e.raw_partition}/{e.raw_offset}</td>
                      <td className="px-3 py-2 text-xs font-mono">{e.normalized_partition}/{e.normalized_offset}</td>
                      <td className="px-3 py-2 text-xs text-gray-600">{formatUtc(e.created_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <EmptyState message="No normalized events found." />
          )}
        </div>

        <div className="rounded border border-gray-200 bg-white p-3">
          {!selectedId ? (
            <EmptyState message="Select a normalized event to inspect details." />
          ) : detail.isLoading ? (
            <LoadingState />
          ) : detail.isError ? (
            <ErrorState error={detail.error} />
          ) : detail.data ? (
            <NormalizedDetailBody ev={detail.data.normalized_event} />
          ) : null}
        </div>
      </div>
    </div>
  );
}

function NormalizedDetailBody({ ev }: { ev: NormalizedEvent }) {
  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <code className="break-all text-xs text-gray-700">{ev.event_id}</code>
        <CopyButton value={ev.event_id} />
      </div>
      <div className="grid grid-cols-2 gap-2 text-sm">
        <div><div className="text-xs text-gray-500">Schema</div><div className="text-xs">{ev.schema_id} v{ev.schema_version}</div></div>
        <div><div className="text-xs text-gray-500">Confidence</div><div>{ev.confidence.toFixed(2)}</div></div>
        <div><div className="text-xs text-gray-500">Observed</div><div className="text-xs">{formatUtc(ev.observation_time)}</div></div>
        <div><div className="text-xs text-gray-500">Processed</div><div className="text-xs">{formatUtc(ev.processing_time)}</div></div>
        <div><div className="text-xs text-gray-500">Source</div><div className="text-xs font-mono">{ev.source_id}</div></div>
        <div><div className="text-xs text-gray-500">Dataset</div><div className="text-xs">{ev.dataset}</div></div>
        <div><div className="text-xs text-gray-500">Raw Broker</div><div className="text-xs font-mono">{ev.raw_partition}/{ev.raw_offset}</div></div>
        <div><div className="text-xs text-gray-500">Normalized Broker</div><div className="text-xs font-mono">{ev.normalized_partition}/{ev.normalized_offset}</div></div>
      </div>
      <JsonViewer label="Entities" value={ev.entities} />
      <JsonViewer label="Evidence" value={ev.evidence} />
      <JsonViewer label="Metadata" value={ev.metadata} />
      <JsonViewer label="Full Event" value={ev.event} />
    </div>
  );
}

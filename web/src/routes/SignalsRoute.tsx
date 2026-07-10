import { useState } from 'react';
import { Link } from '@tanstack/react-router';
import { Radar } from 'lucide-react';
import { useSignals, useSignal } from '../api/queries';
import { useAppProfile } from '../apps/AppProfileContext';
import { LoadingState, ErrorState, EmptyState } from '../components/States';
import { MetricTile } from '../components/MetricTile';
import { CopyButton } from '../components/CopyButton';
import { JsonViewer } from '../components/JsonViewer';
import { formatUtc } from '../lib/format';
import type { SignalRecord } from '../types';
import { useTenant } from '../auth/session';

const SEVERITY_STYLES: Record<string, string> = {
  critical: 'text-red-700',
  high: 'text-orange-700',
  medium: 'text-amber-700',
  low: 'text-gray-700',
  info: 'text-gray-500',
};

function SeverityLabel({ severity }: { severity: string }) {
  return (
    <span className={`text-xs font-medium ${SEVERITY_STYLES[severity] ?? 'text-gray-600'}`}>{severity}</span>
  );
}

export function SignalsRoute() {
  const TENANT_ID = useTenant();
  const { metadataFilter } = useAppProfile();
  const [sourceId, setSourceId] = useState('');
  const [dataset, setDataset] = useState('');
  const [detectorId, setDetectorId] = useState('');
  const [severity, setSeverity] = useState('');
  const [limit, setLimit] = useState(50);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const list = useSignals({
    tenant_id: TENANT_ID,
    source_id: sourceId || undefined,
    dataset: dataset || undefined,
    detector_id: detectorId || undefined,
    severity: severity || undefined,
    limit,
    ...metadataFilter,
  });
  const detail = useSignal(selectedId);
  const data = list.data?.signals ?? [];

  const detectors = new Set(data.map((s) => s.detector_id)).size;
  const highCritical = data.filter((s) => s.severity === 'high' || s.severity === 'critical').length;
  const avgConfidence = data.length ? data.reduce((n, s) => n + s.confidence, 0) / data.length : 0;

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h1 className="text-lg font-semibold">Signals</h1>
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
          <input
            placeholder="detector id"
            value={detectorId}
            onChange={(e) => setDetectorId(e.target.value)}
            className="rounded border border-gray-300 px-2 py-1 text-sm"
          />
          <select
            value={severity}
            onChange={(e) => setSeverity(e.target.value)}
            className="rounded border border-gray-300 px-2 py-1 text-sm"
          >
            <option value="">any severity</option>
            {['info', 'low', 'medium', 'high', 'critical'].map((s) => (
              <option key={s} value={s}>{s}</option>
            ))}
          </select>
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
        <MetricTile label="Signals" value={data.length} />
        <MetricTile label="Detectors" value={detectors} />
        <MetricTile label="High/Critical" value={highCritical} />
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
                    <th className="px-3 py-2">Signal</th>
                    <th className="px-3 py-2">Detector</th>
                    <th className="px-3 py-2">Model</th>
                    <th className="px-3 py-2">Source/Dataset</th>
                    <th className="px-3 py-2">Severity</th>
                    <th className="px-3 py-2">Confidence</th>
                    <th className="px-3 py-2">Events</th>
                    <th className="px-3 py-2">Window</th>
                    <th className="px-3 py-2">Broker</th>
                    <th className="px-3 py-2">Created</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {data.map((s) => (
                    <tr
                      key={s.signal_id}
                      onClick={() => setSelectedId(s.signal_id)}
                      className={`cursor-pointer align-top hover:bg-gray-50 ${selectedId === s.signal_id ? 'bg-brand-50' : ''}`}
                    >
                      <td className="px-3 py-2">
                        <div className="flex items-start gap-2">
                          <Radar size={16} className="mt-0.5 text-brand-700" />
                          <div>
                            <div className="font-mono text-xs text-gray-700">{s.signal_id}</div>
                            <div className="text-xs text-gray-500">{s.signal_type}</div>
                          </div>
                        </div>
                      </td>
                      <td className="px-3 py-2 text-xs">
                        <div className="font-mono">{s.detector_id}</div>
                        <div className="text-gray-500">v{s.detector_version}</div>
                      </td>
                      <td className="px-3 py-2 text-xs font-mono">{s.model_version}</td>
                      <td className="px-3 py-2 text-xs">
                        <div className="font-mono">{s.source_id}</div>
                        <div>{s.dataset}</div>
                      </td>
                      <td className="px-3 py-2"><SeverityLabel severity={s.severity} /></td>
                      <td className="px-3 py-2 text-xs">{s.confidence.toFixed(2)}</td>
                      <td className="px-3 py-2 text-xs">{s.event_ids.length}</td>
                      <td className="px-3 py-2 text-xs text-gray-600">{formatUtc(s.window_start)}</td>
                      <td className="px-3 py-2 text-xs font-mono">{s.broker_partition}/{s.broker_offset}</td>
                      <td className="px-3 py-2 text-xs text-gray-600">{formatUtc(s.created_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <EmptyState message="No signals found." />
          )}
        </div>

        <div className="rounded border border-gray-200 bg-white p-3">
          {!selectedId ? (
            <EmptyState message="Select a signal to inspect details." />
          ) : detail.isLoading ? (
            <LoadingState />
          ) : detail.isError ? (
            <ErrorState error={detail.error} />
          ) : detail.data ? (
            <SignalDetailBody signal={detail.data.signal} />
          ) : null}
        </div>
      </div>
    </div>
  );
}

function SignalDetailBody({ signal }: { signal: SignalRecord }) {
  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <SeverityLabel severity={signal.severity} />
        <code className="break-all text-xs text-gray-700">{signal.signal_id}</code>
        <CopyButton value={signal.signal_id} />
      </div>
      <div className="grid grid-cols-2 gap-2 text-sm">
        <div><div className="text-xs text-gray-500">Type</div><div className="text-xs">{signal.signal_type}</div></div>
        <div><div className="text-xs text-gray-500">Confidence</div><div>{signal.confidence.toFixed(2)}</div></div>
        <div><div className="text-xs text-gray-500">Detector</div><div className="text-xs font-mono">{signal.detector_id} v{signal.detector_version}</div></div>
        <div><div className="text-xs text-gray-500">Model</div><div className="text-xs font-mono">{signal.model_version}</div></div>
        <div><div className="text-xs text-gray-500">Source</div><div className="text-xs font-mono">{signal.source_id}</div></div>
        <div><div className="text-xs text-gray-500">Dataset</div><div className="text-xs">{signal.dataset}</div></div>
        <div><div className="text-xs text-gray-500">Window</div><div className="text-xs">{formatUtc(signal.window_start)} → {formatUtc(signal.window_end)}</div></div>
        <div><div className="text-xs text-gray-500">Broker</div><div className="text-xs font-mono">{signal.broker_partition}/{signal.broker_offset}</div></div>
      </div>
      <div>
        <div className="mb-1 text-xs font-medium text-gray-600">Event IDs</div>
        {signal.event_ids.length ? (
          <Link to="/normalized-events" className="break-all text-xs text-brand-700 hover:underline">
            {signal.event_ids.join(', ')}
          </Link>
        ) : (
          <span className="text-xs text-gray-400">—</span>
        )}
      </div>
      <JsonViewer label="Entities" value={signal.entities} />
      <JsonViewer label="Supporting Metrics" value={signal.supporting_metrics} />
      <JsonViewer label="Graph Targets" value={signal.graph_targets} />
      <JsonViewer label="Semantic Evidence" value={signal.semantic_evidence} />
      <JsonViewer label="Evidence" value={signal.evidence} />
      <JsonViewer label="Recommendation" value={signal.recommendation} />
      <JsonViewer label="Full Signal Event" value={signal.event} />
    </div>
  );
}

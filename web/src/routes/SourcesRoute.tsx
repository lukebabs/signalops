import { DatabaseZap } from 'lucide-react';
import { useCatalogSources } from '../api/queries';
import { EmptyState, ErrorState, LoadingState } from '../components/States';
import { StatusBadge } from '../components/StatusBadge';
import { MetricTile } from '../components/MetricTile';
import { JsonViewer } from '../components/JsonViewer';
import { formatUtc } from '../lib/format';

const TENANT_ID = 'tenant-local';

export function SourcesRoute() {
  const sources = useCatalogSources(TENANT_ID, 50);
  const data = sources.data?.sources ?? [];
  const active = data.filter((source) => source.status === 'active').length;
  const datasets = new Set(data.flatMap((source) => source.datasets));

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-lg font-semibold">Sources</h1>
          <p className="text-xs text-gray-500">Tenant {TENANT_ID}</p>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-2 md:grid-cols-3">
        <MetricTile label="Registered Sources" value={data.length} />
        <MetricTile label="Active Sources" value={active} />
        <MetricTile label="Datasets" value={datasets.size} />
      </div>

      {sources.isLoading ? (
        <LoadingState />
      ) : sources.isError ? (
        <ErrorState error={sources.error} />
      ) : data.length ? (
        <div className="overflow-hidden rounded border border-gray-200 bg-white">
          <table className="min-w-full divide-y divide-gray-200 text-sm">
            <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500">
              <tr>
                <th className="px-3 py-2">Source</th>
                <th className="px-3 py-2">Domain</th>
                <th className="px-3 py-2">Adapter</th>
                <th className="px-3 py-2">Modes</th>
                <th className="px-3 py-2">Datasets</th>
                <th className="px-3 py-2">Status</th>
                <th className="px-3 py-2">Updated</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {data.map((source) => (
                <tr key={`${source.tenant_id}:${source.source_id}`} className="align-top">
                  <td className="px-3 py-2">
                    <div className="flex items-start gap-2">
                      <DatabaseZap size={16} className="mt-0.5 text-brand-700" />
                      <div>
                        <div className="font-medium text-gray-900">{source.display_name}</div>
                        <div className="font-mono text-xs text-gray-500">{source.source_id}</div>
                        <div className="mt-1 max-w-md text-xs text-gray-600">{source.description}</div>
                      </div>
                    </div>
                  </td>
                  <td className="px-3 py-2 font-mono text-xs">{source.source_domain}</td>
                  <td className="px-3 py-2 font-mono text-xs">{source.source_adapter}</td>
                  <td className="px-3 py-2 text-xs">{source.ingestion_modes.join(', ') || '—'}</td>
                  <td className="px-3 py-2 text-xs">{source.datasets.join(', ') || '—'}</td>
                  <td className="px-3 py-2"><StatusBadge status={source.status} /></td>
                  <td className="px-3 py-2 text-xs text-gray-600">{formatUtc(source.updated_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <EmptyState message="No catalog sources registered for this tenant." />
      )}

      {data.length > 0 && (
        <div className="rounded border border-gray-200 bg-white p-3">
          <h2 className="mb-2 text-sm font-semibold">Source Metadata</h2>
          <JsonViewer value={data.map((source) => ({ source_id: source.source_id, metadata: source.metadata }))} />
        </div>
      )}
    </div>
  );
}

import { Workflow } from 'lucide-react';
import { useCatalogPipelines } from '../api/queries';
import { EmptyState, ErrorState, LoadingState } from '../components/States';
import { StatusBadge } from '../components/StatusBadge';
import { MetricTile } from '../components/MetricTile';
import { JsonViewer } from '../components/JsonViewer';
import { formatUtc } from '../lib/format';
import { useTenant } from '../auth/session';

export function PipelinesRoute() {
  const TENANT_ID = useTenant();
  const pipelines = useCatalogPipelines(TENANT_ID, 50);
  const data = pipelines.data?.pipelines ?? [];
  const active = data.filter((pipeline) => pipeline.status === 'active').length;
  const stages = new Set(data.flatMap((pipeline) => pipeline.stages));
  const outputTopics = new Set(data.flatMap((pipeline) => pipeline.output_topics));

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-lg font-semibold">Pipelines</h1>
          <p className="text-xs text-gray-500">Tenant {TENANT_ID}</p>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-2 md:grid-cols-4">
        <MetricTile label="Registered Pipelines" value={data.length} />
        <MetricTile label="Active Pipelines" value={active} />
        <MetricTile label="Stages" value={stages.size} />
        <MetricTile label="Output Topics" value={outputTopics.size} />
      </div>

      {pipelines.isLoading ? (
        <LoadingState />
      ) : pipelines.isError ? (
        <ErrorState error={pipelines.error} />
      ) : data.length ? (
        <div className="overflow-hidden rounded border border-gray-200 bg-white">
          <table className="min-w-full divide-y divide-gray-200 text-sm">
            <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500">
              <tr>
                <th className="px-3 py-2">Pipeline</th>
                <th className="px-3 py-2">Source</th>
                <th className="px-3 py-2">Stages</th>
                <th className="px-3 py-2">Inputs</th>
                <th className="px-3 py-2">Outputs</th>
                <th className="px-3 py-2">Status</th>
                <th className="px-3 py-2">Updated</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {data.map((pipeline) => (
                <tr key={`${pipeline.tenant_id}:${pipeline.pipeline_id}`} className="align-top">
                  <td className="px-3 py-2">
                    <div className="flex items-start gap-2">
                      <Workflow size={16} className="mt-0.5 text-brand-700" />
                      <div>
                        <div className="font-medium text-gray-900">{pipeline.pipeline_name}</div>
                        <div className="font-mono text-xs text-gray-500">{pipeline.pipeline_id}</div>
                        <div className="mt-1 max-w-md text-xs text-gray-600">{pipeline.description}</div>
                      </div>
                    </div>
                  </td>
                  <td className="px-3 py-2 text-xs">
                    <div className="font-mono text-gray-900">{pipeline.source_id}</div>
                    <div className="font-mono text-gray-500">{pipeline.source_domain}</div>
                  </td>
                  <td className="px-3 py-2 text-xs">{pipeline.stages.join(' -> ') || '-'}</td>
                  <td className="px-3 py-2 text-xs">{pipeline.input_datasets.join(', ') || '-'}</td>
                  <td className="px-3 py-2 text-xs">{pipeline.output_topics.join(', ') || '-'}</td>
                  <td className="px-3 py-2"><StatusBadge status={pipeline.status} /></td>
                  <td className="px-3 py-2 text-xs text-gray-600">{formatUtc(pipeline.updated_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <EmptyState message="No catalog pipelines registered for this tenant." />
      )}

      {data.length > 0 && (
        <div className="rounded border border-gray-200 bg-white p-3">
          <h2 className="mb-2 text-sm font-semibold">Pipeline Metadata</h2>
          <JsonViewer value={data.map((pipeline) => ({ pipeline_id: pipeline.pipeline_id, metadata: pipeline.metadata }))} />
        </div>
      )}
    </div>
  );
}

import { ShieldCheck } from 'lucide-react';
import { useCatalogRules } from '../api/queries';
import { EmptyState, ErrorState, LoadingState } from '../components/States';
import { StatusBadge } from '../components/StatusBadge';
import { MetricTile } from '../components/MetricTile';
import { JsonViewer } from '../components/JsonViewer';
import { formatUtc, orDash } from '../lib/format';
import { useTenant } from '../auth/session';

// Severity has no shared badge component; render as restrained colored text
// (no new color-heavy visual system), consistent with Sources/Pipelines.
const SEVERITY_STYLES: Record<string, string> = {
  critical: 'text-red-700',
  high: 'text-orange-700',
  medium: 'text-amber-700',
  low: 'text-gray-700',
  info: 'text-gray-500',
};

function SeverityLabel({ severity }: { severity: string }) {
  return (
    <span className={`text-xs font-medium ${SEVERITY_STYLES[severity] ?? 'text-gray-600'}`}>
      {severity}
    </span>
  );
}

export function RulesRoute() {
  const TENANT_ID = useTenant();
  const rules = useCatalogRules(TENANT_ID, 50);
  const data = rules.data?.rules ?? [];
  const active = data.filter((rule) => rule.status === 'active').length;
  const ruleTypes = new Set(data.map((rule) => rule.rule_type));
  const criticalOrHigh = data.filter(
    (rule) => rule.severity === 'critical' || rule.severity === 'high',
  ).length;

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-lg font-semibold">Rules</h1>
          <p className="text-xs text-gray-500">Tenant {TENANT_ID}</p>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-2 md:grid-cols-4">
        <MetricTile label="Registered Rules" value={data.length} />
        <MetricTile label="Active Rules" value={active} />
        <MetricTile label="Rule Types" value={ruleTypes.size} />
        <MetricTile label="Critical/High" value={criticalOrHigh} />
      </div>

      {rules.isLoading ? (
        <LoadingState />
      ) : rules.isError ? (
        <ErrorState error={rules.error} />
      ) : data.length ? (
        <div className="overflow-x-auto rounded border border-gray-200 bg-white">
          <table className="min-w-full divide-y divide-gray-200 text-sm">
            <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500">
              <tr>
                <th className="px-3 py-2">Rule</th>
                <th className="px-3 py-2">Type</th>
                <th className="px-3 py-2">Severity</th>
                <th className="px-3 py-2">Scope</th>
                <th className="px-3 py-2">Actions</th>
                <th className="px-3 py-2">Status</th>
                <th className="px-3 py-2">Updated</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {data.map((rule) => (
                <tr key={`${rule.tenant_id}:${rule.rule_id}`} className="align-top">
                  <td className="px-3 py-2">
                    <div className="flex items-start gap-2">
                      <ShieldCheck size={16} className="mt-0.5 text-brand-700" />
                      <div>
                        <div className="font-medium text-gray-900">
                          {rule.rule_name}{' '}
                          <span className="font-normal text-gray-400">v{rule.version}</span>
                        </div>
                        <div className="font-mono text-xs text-gray-500">{rule.rule_id}</div>
                        <div className="mt-1 max-w-md text-xs text-gray-600">{rule.description}</div>
                      </div>
                    </div>
                  </td>
                  <td className="px-3 py-2 font-mono text-xs">{rule.rule_type}</td>
                  <td className="px-3 py-2">
                    <SeverityLabel severity={rule.severity} />
                  </td>
                  <td className="px-3 py-2 text-xs">
                    <div className="font-mono text-gray-900">{orDash(rule.source_id)}</div>
                    <div className="font-mono text-gray-500">{orDash(rule.pipeline_id)}</div>
                    <div className="text-gray-600">{rule.dataset_scope.join(', ') || '—'}</div>
                    <div className="text-gray-500">{rule.entity_scope.join(', ') || '—'}</div>
                  </td>
                  <td className="px-3 py-2 text-xs">{rule.actions.join(', ') || '—'}</td>
                  <td className="px-3 py-2">
                    <StatusBadge status={rule.status} />
                  </td>
                  <td className="px-3 py-2 text-xs text-gray-600">{formatUtc(rule.updated_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <EmptyState message="No catalog rules registered for this tenant." />
      )}

      {data.length > 0 && (
        <div className="space-y-3">
          <div className="rounded border border-gray-200 bg-white p-3">
            <h2 className="mb-2 text-sm font-semibold">Rule Expressions</h2>
            <JsonViewer
              value={data.map((rule) => ({ rule_id: rule.rule_id, expression: rule.expression }))}
            />
          </div>
          <div className="rounded border border-gray-200 bg-white p-3">
            <h2 className="mb-2 text-sm font-semibold">Rule Metadata</h2>
            <JsonViewer
              value={data.map((rule) => ({ rule_id: rule.rule_id, metadata: rule.metadata }))}
            />
          </div>
        </div>
      )}
    </div>
  );
}

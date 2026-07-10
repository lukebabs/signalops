import { CircleDollarSign } from 'lucide-react';
import { useMarketOpsAssets } from '../api/queries';
import { EmptyState, ErrorState, LoadingState } from '../components/States';
import { StatusBadge } from '../components/StatusBadge';
import { MetricTile } from '../components/MetricTile';
import { JsonViewer } from '../components/JsonViewer';
import { formatUtc } from '../lib/format';
import { useTenant } from '../auth/session';

// Read-only MarketOps asset universe (G071 frontend). Mirrors the dense
// table layout of Sources/Pipelines/Rules; renders backend data only.
export function MarketOpsAssetsRoute() {
  const TENANT_ID = useTenant();
  const query = useMarketOpsAssets({
    tenant_id: TENANT_ID,
    universe_group: 'top50_megacap',
    active_only: true,
    limit: 50,
  });

  // Sort defensively by rank so the displayed order is stable regardless of
  // backend ordering; slice() avoids mutating the cached response.
  const data = (query.data?.assets ?? []).slice().sort((a, b) => a.rank - b.rank);
  const active = data.filter((a) => a.is_active).length;
  const sectors = new Set(data.map((a) => a.sector_key || a.sector).filter(Boolean)).size;
  const industries = new Set(data.map((a) => a.industry_key || a.industry).filter(Boolean)).size;
  const sourceIds = new Set(data.map((a) => a.source_id));
  const sourceLabel = sourceIds.size === 1 ? [...sourceIds][0] : `${sourceIds.size} sources`;

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-lg font-semibold">Assets</h1>
          <p className="text-xs text-gray-500">Tenant {TENANT_ID} · top50_megacap</p>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-2 md:grid-cols-5">
        <MetricTile label="Universe Assets" value={data.length} />
        <MetricTile label="Active Assets" value={active} />
        <MetricTile label="Sectors" value={sectors} />
        <MetricTile label="Industries" value={industries} />
        <MetricTile label="Source" value={sourceLabel} />
      </div>

      {query.isLoading ? (
        <LoadingState />
      ) : query.isError ? (
        <ErrorState error={query.error} />
      ) : data.length ? (
        <div className="overflow-hidden rounded border border-gray-200 bg-white">
          <table className="min-w-full divide-y divide-gray-200 text-sm">
            <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500">
              <tr>
                <th className="px-3 py-2">Rank</th>
                <th className="px-3 py-2">Asset</th>
                <th className="px-3 py-2">Sector</th>
                <th className="px-3 py-2">Industry</th>
                <th className="px-3 py-2">Source</th>
                <th className="px-3 py-2">Status</th>
                <th className="px-3 py-2">Updated</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {data.map((a) => (
                <tr key={`${a.tenant_id}:${a.ticker_key}`} className="align-top">
                  <td className="px-3 py-2 text-xs text-gray-500">{a.rank}</td>
                  <td className="px-3 py-2">
                    <div className="flex items-start gap-2">
                      <CircleDollarSign size={16} className="mt-0.5 text-brand-700" />
                      <div>
                        <div className="font-medium text-gray-900">{a.company || '—'}</div>
                        <div className="font-mono text-xs text-gray-500">{a.ticker}</div>
                        <div className="text-xs text-gray-500">{a.asset_type}</div>
                      </div>
                    </div>
                  </td>
                  <td className="px-3 py-2 text-xs">{a.sector || '—'}</td>
                  <td className="px-3 py-2 text-xs">{a.industry || '—'}</td>
                  <td className="px-3 py-2 font-mono text-xs">{a.source_id}</td>
                  <td className="px-3 py-2"><StatusBadge status={a.is_active ? 'active' : 'inactive'} /></td>
                  <td className="px-3 py-2 text-xs text-gray-600">{formatUtc(a.updated_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <EmptyState message="No MarketOps assets found for this tenant." />
      )}

      {data.length > 0 && (
        <div className="rounded border border-gray-200 bg-white p-3">
          <h2 className="mb-2 text-sm font-semibold">Asset Metadata</h2>
          <JsonViewer value={data.map((a) => ({ ticker: a.ticker, metadata: a.metadata }))} />
        </div>
      )}
    </div>
  );
}

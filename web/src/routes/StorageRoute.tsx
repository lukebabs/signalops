import { useMemo, useState } from "react";
import { useRetentionGovernance, useStorageAnalysis, useStorageOverview } from "../api/queries";
import { ThemedEChart as ReactECharts } from "../components/ThemedEChart";
import { ErrorState, EmptyState } from "../components/States";
import { RefreshButton } from "../components/RefreshButton";
import { StatusBadge } from "../components/StatusBadge";
import { formatUtc } from "../lib/format";

function bytes(value?: number) {
  if (value == null) return "—";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let n = value,
    i = 0;
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024;
    i++;
  }
  return `${n.toFixed(i ? 1 : 0)} ${units[i]}`;
}
const ownerLabel = (app: string, domain: string) =>
  app === "platform"
    ? domain === "platform"
      ? "Platform / shared"
      : "Unattributed"
    : `${app === "marketops" ? "MarketOps" : app === "cyberops" ? "CyberOps" : app} · ${domain}`;
const ownerLabelFromKey = (key: string) => {
  const parts = key.split("|");
  return ownerLabel(parts[0] ?? "platform", parts[1] ?? "platform");
};

export function StorageRoute() {
  const [window, setWindow] = useState("90d");
  const [owner, setOwner] = useState("all");
  const overview = useStorageOverview();
  const analysis = useStorageAnalysis(window);
  const governance = useRetentionGovernance();
  const components = useMemo(
    () =>
      (analysis.data?.components ?? []).filter(
        (c) => owner === "all" || `${c.app_id}|${c.domain}` === owner,
      ),
    [analysis.data, owner],
  );
  const history = useMemo(() => analysis.data?.history ?? [], [analysis.data]);
  const chart = useMemo(() => {
    const owners = [
      ...new Set(history.map((p) => `${p.app_id}|${p.domain}`)),
    ].filter((x) => owner === "all" || x === owner);
    const dates = [...new Set(history.map((p) => p.date))];
    return {
      tooltip: { trigger: "axis" },
      legend: { bottom: 0 },
      grid: { left: 48, right: 16, top: 24, bottom: 44 },
      xAxis: { type: "category", data: dates },
      yAxis: {
        type: "value",
        axisLabel: { formatter: (v: number) => bytes(v) },
      },
      series: owners.map((key) => ({
        name: ownerLabelFromKey(key),
        type: "line",
        smooth: true,
        showSymbol: false,
        data: dates.map(
          (date) =>
            history.find(
              (p) => p.date === date && `${p.app_id}|${p.domain}` === key,
            )?.attributed_bytes ?? 0,
        ),
      })),
    };
  }, [history, owner]);
  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-lg font-semibold">Storage intelligence</h1>
          <p className="text-sm text-gray-500">
            Capacity, ownership, and growth across SignalOps data stores.
            Component inventory is recorded daily at 02:00 ET and retained for
            90 days.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <select
            value={window}
            onChange={(e) => setWindow(e.target.value)}
            className="rounded border border-gray-300 bg-white px-2 py-1 text-xs"
          >
            <option value="7d">7 days</option>
            <option value="30d">30 days</option>
            <option value="90d">90 days</option>
          </select>
          <RefreshButton
            onClick={() => {
              overview.refetch();
              analysis.refetch();
            }}
            loading={overview.isFetching || analysis.isFetching}
          />
        </div>
      </div>
      <div className="grid gap-2 md:grid-cols-3">
        {(overview.data?.stores ?? []).map((s) => (
          <div
            key={s.store_id}
            className="rounded border border-gray-200 bg-white p-3"
          >
            <div className="flex justify-between gap-2">
              <strong className="capitalize">{s.store_id}</strong>
              <StatusBadge status={s.status} />
            </div>
            {s.used_bytes != null ? (
              <>
                <div className="mt-2 text-xl font-semibold">
                  {bytes(s.used_bytes)}
                </div>
                <div className="text-xs text-gray-500">
                  of {bytes(s.capacity_bytes)} ·{" "}
                  {Number(s.usage_percent ?? 0).toFixed(1)}% used
                </div>
                <div className="mt-2 h-2 overflow-hidden rounded bg-gray-100">
                  <div
                    className={
                      s.status === "critical"
                        ? "h-full bg-red-500"
                        : s.status === "warning"
                          ? "h-full bg-amber-400"
                          : "h-full bg-brand-600"
                    }
                    style={{
                      width: `${Math.min(100, Number(s.usage_percent ?? 0))}%`,
                    }}
                  />
                </div>
                <div className="mt-2 text-[11px] text-gray-500">
                  Free {bytes(s.free_bytes)} · {formatUtc(s.observed_at)}
                </div>
              </>
            ) : (
              <p className="mt-2 text-xs text-gray-500">
                {s.message ?? "No snapshot recorded."}
              </p>
            )}
          </div>
        ))}
      </div>
      {analysis.isError ? (
        <ErrorState error={analysis.error} />
      ) : analysis.isLoading ? (
        <div className="text-sm text-gray-500">Loading storage inventory…</div>
      ) : !analysis.data?.components.length ? (
        <EmptyState message="No detailed storage inventory has been recorded yet. The monitor captures it on its next daily run at 02:00 ET." />
      ) : (
        <>
          <section className="rounded border border-gray-200 bg-white p-3">
            <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
              <div>
                <h2 className="text-sm font-semibold">Use-case ownership</h2>
                <p className="text-xs text-gray-500">
                  Exact for dedicated tables and topics; shared ledgers are
                  allocated by row proportion.
                </p>
              </div>
              <select
                value={owner}
                onChange={(e) => setOwner(e.target.value)}
                className="rounded border border-gray-300 bg-white px-2 py-1 text-xs"
              >
                <option value="all">All owners</option>
                {(analysis.data.ownership_totals ?? []).map((x) => (
                  <option
                    key={`${x.app_id}|${x.domain}`}
                    value={`${x.app_id}|${x.domain}`}
                  >
                    {ownerLabel(x.app_id, x.domain)}
                  </option>
                ))}
              </select>
            </div>
            <div className="grid gap-2 md:grid-cols-4">
              {analysis.data.ownership_totals.map((x) => (
                <div
                  key={`${x.app_id}|${x.domain}`}
                  className="rounded border border-gray-200 bg-gray-50 p-2"
                >
                  <div className="text-xs text-gray-500">
                    {ownerLabel(x.app_id, x.domain)}
                  </div>
                  <div className="text-lg font-semibold">
                    {bytes(x.attributed_bytes)}
                  </div>
                </div>
              ))}
            </div>
          </section>
          <section className="rounded border border-gray-200 bg-white p-3">
            <h2 className="text-sm font-semibold">Storage evolution</h2>
            <p className="mb-2 text-xs text-gray-500">
              Daily attributed storage trend for the selected window.
            </p>
            <ReactECharts
              option={chart}
              style={{ height: 300 }}
              notMerge
              lazyUpdate
            />
          </section>
          <section className="rounded border border-gray-200 bg-white">
            <div className="border-b border-gray-200 p-3">
              <h2 className="text-sm font-semibold">
                Largest persisted components
              </h2>
              <p className="text-xs text-gray-500">
                Physical size includes indexes and table overhead. “Estimated”
                is attribution only, not measured disk size.
              </p>
            </div>
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200 text-xs">
                <thead className="bg-gray-50 text-left text-gray-500">
                  <tr>
                    <th className="px-3 py-2">Component</th>
                    <th className="px-3 py-2">Store</th>
                    <th className="px-3 py-2">Owner</th>
                    <th className="px-3 py-2">Attribution</th>
                    <th className="px-3 py-2 text-right">Physical</th>
                    <th className="px-3 py-2 text-right">Attributed</th>
                    <th className="px-3 py-2">Observed</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {[...components]
                    .sort((a, b) => b.attributed_bytes - a.attributed_bytes)
                    .map((c) => (
                      <tr
                        key={`${c.store_id}:${c.component_name}:${c.app_id}:${c.domain}`}
                      >
                        <td className="px-3 py-2">
                          <code>{c.component_name}</code>
                          <span className="ml-2 text-gray-500">
                            {c.component_kind}
                          </span>
                        </td>
                        <td className="px-3 py-2 capitalize">{c.store_id}</td>
                        <td className="px-3 py-2">
                          {ownerLabel(c.app_id, c.domain)}
                        </td>
                        <td className="px-3 py-2">
                          <span
                            className={
                              c.attribution_method === "estimated"
                                ? "rounded bg-amber-100 px-1.5 py-0.5 text-amber-800"
                                : "rounded bg-brand-50 px-1.5 py-0.5 text-brand-700"
                            }
                          >
                            {c.attribution_method}
                          </span>
                        </td>
                        <td className="px-3 py-2 text-right">
                          {bytes(c.physical_bytes)}
                        </td>
                        <td className="px-3 py-2 text-right">
                          {bytes(c.attributed_bytes)}
                        </td>
                        <td className="px-3 py-2 text-gray-500">
                          {formatUtc(c.observed_at)}
                        </td>
                      </tr>
                    ))}
                </tbody>
              </table>
            </div>
          </section>
        </>
      )}
      <section className="rounded border border-gray-200 bg-white">
        <div className="border-b border-gray-200 p-3"><h2 className="text-sm font-semibold">Retention governance</h2><p className="text-xs text-gray-500">Metadata-first policies. All policies remain dry-run until explicitly enforced.</p></div>
        {governance.isLoading ? <div className="p-3 text-xs text-gray-500">Loading policy status…</div> : governance.isError ? <div className="p-3"><ErrorState error={governance.error} /></div> : <div className="overflow-x-auto"><table className="min-w-full divide-y divide-gray-200 text-xs"><thead className="bg-gray-50 text-left text-gray-500"><tr><th className="px-3 py-2">Policy</th><th className="px-3 py-2">Scope</th><th className="px-3 py-2">Retention</th><th className="px-3 py-2">Preservation</th><th className="px-3 py-2">Mode</th><th className="px-3 py-2">Latest run</th></tr></thead><tbody className="divide-y divide-gray-100">{(governance.data?.policies ?? []).map(p => <tr key={p.policy_id}><td className="px-3 py-2"><code>{p.policy_id}</code><div className="mt-0.5 text-gray-500">{p.description}</div></td><td className="px-3 py-2">{ownerLabel(p.app_id,p.domain)}</td><td className="px-3 py-2">{p.retention_days} days</td><td className="px-3 py-2">{p.preservation_rule || "—"}</td><td className="px-3 py-2"><StatusBadge status={p.mode} /></td><td className="px-3 py-2">{p.last_run ? <><StatusBadge status={p.last_run.status}/><div className="mt-1 text-gray-600">{p.last_run.candidate_rows.toLocaleString()} candidates · {p.last_run.affected_rows.toLocaleString()} removed</div><div className="text-gray-500">{formatUtc(p.last_run.completed_at)}</div></> : <span className="text-gray-500">Awaiting first run</span>}</td></tr>)}</tbody></table></div>}
      </section>
    </div>
  );
}

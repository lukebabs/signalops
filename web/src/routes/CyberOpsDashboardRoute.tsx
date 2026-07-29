import { useEffect, useState, type ReactNode } from "react";
import ReactECharts from "echarts-for-react";
import { useTenant } from "../auth/session";
import { useCyberOpsTrafficOverview } from "../api/queries";
import { subscribeCyberOpsLiveTraffic, type CyberOpsLiveTrafficStatus } from "../api/stream";
import { CyberOpsIoTPanel } from "./CyberOpsIoTPanel";
import { EmptyState, ErrorState, LoadingState } from "../components/States";
import { formatUtc, OPERATING_TIME_ZONE } from "../lib/format";
import type { CyberOpsLiveTrafficSnapshot, CyberOpsTrafficCount, CyberOpsTrafficFlow, CyberOpsTrafficWindow } from "../types";

const windows: CyberOpsTrafficWindow[] = ["1h", "24h", "7d"];

type LiveTrafficState = { snapshot?: CyberOpsLiveTrafficSnapshot; status: CyberOpsLiveTrafficStatus; error?: Error };

export function CyberOpsDashboardRoute() {
  const tenant = useTenant();
  const [window, setWindow] = useState<CyberOpsTrafficWindow>("24h");
  const [flows, setFlows] = useState<CyberOpsTrafficFlow[] | null>(null);
  const query = useCyberOpsTrafficOverview(tenant, window);
  const live = useCyberOpsLiveTraffic(tenant);
  const data = query.data;
  const filter = (kind: "source" | "destination" | "protocol" | "port", key: string) => {
    if (!data) return;
    setFlows(data.flows.filter((flow) => kind === "source" ? flow.source_ip === key : kind === "destination" ? flow.destination_ip === key : kind === "protocol" ? flow.protocol === key : flow.protocol + "/" + flow.destination_port === key));
  };

  return <div className="space-y-3">
    <div className="flex flex-wrap items-center justify-between gap-2">
      <div>
        <h1 className="text-lg font-semibold">CyberOps traffic dashboard</h1>
        <p className="text-xs text-gray-500">Allowed firewall traffic and live ingestion quality. Counts are log observations, not sessions or bytes.</p>
      </div>
      <select aria-label="Traffic window" value={window} onChange={(event) => { setWindow(event.target.value as CyberOpsTrafficWindow); setFlows(null); }} className="rounded border border-gray-300 px-2 py-1 text-xs">
        {windows.map((value) => <option key={value} value={value}>Last {value}</option>)}
      </select>
    </div>
    <LiveTrafficPanel live={live} />
    {query.isLoading && !data ? <LoadingState label="Loading allowed firewall traffic..." /> : null}
    {query.isError && !data ? <ErrorState error={query.error} /> : null}
    {!query.isLoading && !query.isError && (!data || !data.allowed_events) ? <EmptyState message="No explicit allowed firewall traffic is available for this window." /> : null}
    {data && data.allowed_events ? <>
      <div className="grid gap-2 sm:grid-cols-3 xl:grid-cols-6">
        <Metric label="Allowed logs" value={data.allowed_events} />
        <Metric label="Sources" value={data.unique_sources} />
        <Metric label="Destinations" value={data.unique_destinations} />
        <Metric label="Services" value={data.unique_services} />
        <Metric label="Unparsed logs" value={data.unparsed_logs} />
        <Metric label="Updated" value={formatUtc(data.generated_at)} />
      </div>
      <div className="grid gap-3 xl:grid-cols-2">
        <Timeline points={data.timeline} />
        <Ranking title="Protocols" values={data.protocols} onClick={(key) => filter("protocol", key)} />
        <Ranking title="Top source IPs" values={data.top_sources} onClick={(key) => filter("source", key)} />
        <Ranking title="Top destinations" values={data.top_destinations} onClick={(key) => filter("destination", key)} />
        <Ranking title="Destination services" values={data.destination_ports} onClick={(key) => filter("port", key)} />
        <FlowChart flows={data.flows} onClick={setFlows} />
      </div>
      <FlowTable flows={flows ?? data.flows} filtered={flows !== null} onClose={() => setFlows(null)} />
    </> : null}
    <CyberOpsIoTPanel />
  </div>;
}

function useCyberOpsLiveTraffic(tenantId: string): LiveTrafficState {
  const [live, setLive] = useState<LiveTrafficState>({ status: "connecting" });
  useEffect(() => {
    let active = true;
    setLive({ status: "connecting" });
    const subscription = subscribeCyberOpsLiveTraffic({
      tenantId,
      onSnapshot: (snapshot) => { if (active) setLive({ snapshot, status: "open" }); },
      onStatus: (status) => { if (active) setLive((current) => ({ ...current, status })); },
      onError: (error) => { if (active) setLive((current) => ({ ...current, status: "reconnecting", error })); },
    });
    return () => { active = false; subscription.close(); };
  }, [tenantId]);
  return live;
}

function LiveTrafficPanel({ live }: { live: LiveTrafficState }) {
  const points = live.snapshot?.points ?? [];
  const latest = points[points.length - 1];
  const hasLogs = points.some((point) => point.received_logs > 0);
  const unparsed = points.reduce((total, point) => total + point.unparsed_logs, 0);
  const historicalWindow = live.snapshot?.last_observed_at ? Date.now() - new Date(live.snapshot.last_observed_at).getTime() > 30 * 60 * 1000 : false;
  const option = {
    animationDurationUpdate: 250,
    tooltip: { trigger: "axis" },
    legend: { data: ["Received logs", "Allowed traffic", "Unparsed logs"], top: 0, textStyle: { fontSize: 10 } },
    grid: { left: 42, right: 16, top: 48, bottom: 58, containLabel: true },
    xAxis: { type: "category", data: points.map((point) => formatLiveTime(point.time)), axisLabel: { fontSize: 9, interval: 4, rotate: 35, margin: 12, hideOverlap: true } },
    yAxis: { type: "value", minInterval: 1, axisLabel: { fontSize: 9 } },
    series: [
      { name: "Received logs", type: "line", showSymbol: false, smooth: true, data: points.map((point) => point.received_logs), lineStyle: { width: 2, color: "#2563eb" }, areaStyle: { color: "#dbeafe" } },
      { name: "Allowed traffic", type: "line", showSymbol: false, smooth: true, data: points.map((point) => point.allowed_events), lineStyle: { width: 2, color: "#059669" } },
      { name: "Unparsed logs", type: "line", showSymbol: false, smooth: true, data: points.map((point) => point.unparsed_logs), lineStyle: { width: 2, color: "#d97706" } },
    ],
  };
  const status = live.status === "open" ? "Live" : live.status === "reconnecting" ? "Reconnecting" : live.status === "closed" ? "Disconnected" : "Connecting";
  const statusTone = live.status === "open" ? "text-emerald-700" : live.status === "reconnecting" ? "text-amber-700" : "text-gray-500";
  return <section className="rounded border border-blue-200 bg-white p-3">
    <div className="mb-2 flex flex-wrap items-start justify-between gap-2">
      <div>
        <div className="text-xs font-semibold text-gray-800">Live firewall inflow</div>
        <p className="text-[11px] text-gray-500">{historicalWindow ? "No current traffic is available; showing the latest observed 30-minute event window." : "Rolling 30 minutes, delivered as a secure live stream and grouped into one-minute buckets."}</p>
      </div>
      <div className={"text-xs font-medium " + statusTone}>{status}</div>
    </div>
    <div className="mb-2 grid gap-2 sm:grid-cols-3">
      <Metric label="Latest received" value={latest?.received_logs ?? "—"} />
      <Metric label="Latest allowed" value={latest?.allowed_events ?? "—"} />
      <Metric label="Last observed" value={live.snapshot?.last_observed_at ? formatUtc(live.snapshot.last_observed_at) : "No logs yet"} />
    </div>
    {points.length ? <ReactECharts option={option} notMerge lazyUpdate style={{ height: 280 }} /> : <div className="py-12 text-xs text-gray-500">Connecting to the live firewall stream…</div>}
    {!hasLogs && points.length ? <p className="mt-1 text-[11px] text-gray-500">No CyberOps firewall logs were observed in this rolling window.</p> : null}
    {unparsed > 0 ? <p className="mt-1 text-[11px] text-amber-700">{unparsed} unparsed log observation{unparsed === 1 ? "" : "s"} in this window. Review firewall parser compatibility; these are not detections.</p> : null}
    {live.error ? <p className="mt-1 text-[11px] text-amber-700">The live stream will retry automatically. {live.error.message}</p> : null}
  </section>;
}

function Metric({ label, value }: { label: string; value: string | number }) {
  return <div className="rounded border border-gray-200 bg-white p-2"><div className="text-[11px] text-gray-500">{label}</div><div className="text-sm font-semibold text-gray-900">{value}</div></div>;
}

function Timeline({ points }: { points: { time: string; count: number }[] }) {
  const option = { tooltip: { trigger: "axis" }, xAxis: { type: "category", data: points.map((point) => formatUtc(point.time)), axisLabel: { fontSize: 9 } }, yAxis: { type: "value", minInterval: 1 }, series: [{ type: "line", smooth: true, data: points.map((point) => point.count), areaStyle: { color: "#dbeafe" }, itemStyle: { color: "#2563eb" } }] };
  return <Panel title="Allowed traffic over time"><ReactECharts option={option} style={{ height: 250 }} /></Panel>;
}

function Ranking({ title, values, onClick }: { title: string; values: CyberOpsTrafficCount[]; onClick: (key: string) => void }) {
  const option = { tooltip: { trigger: "axis" }, xAxis: { type: "value" }, yAxis: { type: "category", data: values.map((value) => value.key).reverse(), axisLabel: { fontSize: 9 } }, series: [{ type: "bar", data: values.map((value) => value.count).reverse(), itemStyle: { color: "#0f766e" } }] };
  return <Panel title={title}><ReactECharts option={option} onEvents={{ click: (event: { dataIndex?: number }) => { const item = values[values.length - 1 - (event.dataIndex ?? -1)]; if (item) onClick(item.key); } }} style={{ height: 250 }} /></Panel>;
}

function FlowChart({ flows, onClick }: { flows: CyberOpsTrafficFlow[]; onClick: (flows: CyberOpsTrafficFlow[]) => void }) {
  const visible = flows.slice(0, 10);
  const option = { tooltip: { trigger: "axis" }, xAxis: { type: "value" }, yAxis: { type: "category", data: visible.map((flow) => flow.source_ip + " → " + flow.destination_ip + ":" + flow.destination_port).reverse(), axisLabel: { fontSize: 8 } }, series: [{ type: "bar", data: visible.map((flow) => flow.count).reverse(), itemStyle: { color: "#7c3aed" } }] };
  return <Panel title="Top source → destination flows"><ReactECharts option={option} onEvents={{ click: (event: { dataIndex?: number }) => { const flow = visible[visible.length - 1 - (event.dataIndex ?? -1)]; if (flow) onClick([flow]); } }} style={{ height: 250 }} /></Panel>;
}

function Panel({ title, children }: { title: string; children: ReactNode }) {
  return <section className="rounded border border-gray-200 bg-white p-3"><div className="mb-1 text-xs font-semibold text-gray-800">{title}</div>{children}</section>;
}

function FlowTable({ flows, filtered, onClose }: { flows: CyberOpsTrafficFlow[]; filtered: boolean; onClose: () => void }) {
  return <section className="rounded border border-gray-200 bg-white p-3"><div className="mb-2 flex justify-between"><div><div className="text-xs font-semibold">{filtered ? "Matching flows" : "Top flows"}</div><p className="text-[11px] text-gray-500">Source, destination, service, count, and observed range.</p></div>{filtered ? <button className="text-xs text-brand-700 underline" onClick={onClose}>Clear</button> : null}</div><div className="overflow-x-auto"><table className="w-full text-left text-xs"><thead><tr className="border-b text-gray-500"><th>Source</th><th>Destination</th><th>Service</th><th>Count</th><th>First seen</th><th>Last seen</th></tr></thead><tbody>{flows.map((flow) => <tr key={flow.source_ip + "-" + flow.destination_ip + "-" + flow.protocol + "-" + flow.destination_port} className="border-b"><td className="font-mono">{flow.source_ip}</td><td className="font-mono">{flow.destination_ip}</td><td>{flow.protocol.toUpperCase()}/{flow.destination_port}</td><td>{flow.count}</td><td>{formatUtc(flow.first_seen)}</td><td>{formatUtc(flow.last_seen)}</td></tr>)}</tbody></table></div></section>;
}

function formatLiveTime(value: string): string {
  const date = new Date(value);
  return date.toLocaleTimeString("en-GB", { timeZone: OPERATING_TIME_ZONE, hour: "2-digit", minute: "2-digit", timeZoneName: "short" });
}

import { useQuery } from "@tanstack/react-query";
import { Fragment, useState } from "react";
import { Link } from "@tanstack/react-router";
import { api } from "../api/client";
import { useTenant } from "../auth/session";
import { MarketOpsWatchlistSelector } from "../components/MarketOpsWatchlistContext";
import { EmptyState, ErrorState, LoadingState } from "../components/States";
import {
  EEOMAssessmentGuide,
  EEOMSelectedSignals,
} from "../components/EEOMAssessmentDetails";
export function MarketOpsEEOMRoute() {
  const tenantId = useTenant();
  const q = useQuery({
    queryKey: ["marketops-eeom", tenantId],
    queryFn: () => api.getMarketOpsEEOM(tenantId),
    staleTime: 60_000,
  });
  const rows = ((q.data?.results ?? []) as any[])
    .slice()
    .sort(
      (a, b) =>
        String(a.earnings_date).localeCompare(String(b.earnings_date)) ||
        String(a.ticker).localeCompare(String(b.ticker)),
    );
  const [selected, setSelected] = useState<string | null>(null);
  const selectedRow = rows.find((row) => row.result_id === selected) ?? null;
  const historyQ = useQuery({
    queryKey: ["marketops-eeom-history", tenantId, selectedRow?.ticker],
    queryFn: () =>
      api.getMarketOpsEEOM(tenantId, {
        symbol: selectedRow?.ticker,
        includeHistory: true,
      }),
    enabled: Boolean(selectedRow?.ticker),
    staleTime: 60_000,
  });
  const historyRows = ((historyQ.data?.results ?? []) as any[])
    .slice()
    .sort(
      (a, b) =>
        String(b.trade_date).localeCompare(String(a.trade_date)) ||
        String(a.earnings_date).localeCompare(String(b.earnings_date)),
    );
  return (
    <div className="space-y-3">
      <div>
        <h1 className="text-lg font-semibold">
          Earnings Opportunity Intelligence
        </h1>
        <p className="text-xs text-gray-500">
          Earnings Opportunity Intelligence (EEOM): deterministic pre-earnings
          setup quality from persisted MarketOps evidence. It is not an earnings
          forecast, price target, or recommendation.
        </p>
      </div>
      <MarketOpsWatchlistSelector />
      <div className="rounded border border-brand-100 bg-brand-50 p-3 text-xs text-brand-800">
        Earnings Opportunity Intelligence (EEOM) scores setup quality from technical, options, Risk/Reward,
        Value Intelligence and Distressed Opportunity Intelligence, earnings materiality. Unavailable inputs are
        shown and reweighted; posture describes evidence balance separately.
      </div>
      <EEOMAssessmentGuide />
      {q.isLoading ? (
        <LoadingState label="Loading earnings opportunities..." />
      ) : q.isError ? (
        <ErrorState error={q.error} />
      ) : !rows.length ? (
        <EmptyState message="No point-in-time-known earnings events have eligible Earnings Opportunity Intelligence evidence in the next 30 days." />
      ) : (
        <div data-testid="marketops-eeom-current-table" className="overflow-x-auto rounded border border-gray-200 bg-white">
          <table className="min-w-full divide-y divide-gray-200 text-sm">
            <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500">
              <tr>
                <th className="px-3 py-2">Asset</th>
                <th className="px-3 py-2">Earnings</th>
                <th className="px-3 py-2">Setup quality</th>
                <th className="px-3 py-2">Posture</th>
                <th className="px-3 py-2">Assessment</th>
                <th className="px-3 py-2">Evidence</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {rows.map((x) => (
                <Fragment key={x.result_id}>
                  <tr
                    data-testid={`marketops-eeom-row-${x.ticker}`}
                    onClick={() =>
                      setSelected(selected === x.result_id ? null : x.result_id)
                    }
                    className={
                      selected === x.result_id
                        ? "cursor-pointer bg-brand-50"
                        : "cursor-pointer hover:bg-gray-50"
                    }
                  >
                    <td className="px-3 py-2 font-mono text-xs font-semibold">
                      <Link
                        to="/marketops/state"
                        search={{ symbol: x.ticker }}
                        onClick={(event) => event.stopPropagation()}
                        className="text-brand-700 underline"
                      >
                        {x.ticker}
                      </Link>
                    </td>
                    <td className="px-3 py-2 text-xs">
                      {x.earnings_date}
                      {x.event?.days_to_event != null ? (
                        <div className="text-[11px] text-gray-500">
                          {x.event.days_to_event} days
                        </div>
                      ) : null}
                    </td>
                    <td className="px-3 py-2 font-semibold">
                      {Number(x.score).toFixed(1)} / 10
                    </td>
                    <td className="px-3 py-2 text-xs capitalize">
                      {String(x.posture).replace("_", " ")}
                    </td>
                    <td className="px-3 py-2 text-xs capitalize">
                      {String(x.classification).replace(/_/g, " ")}
                    </td>
                    <td className="px-3 py-2 text-xs capitalize">
                      {String(x.evidence_quality)}
                    </td>
                  </tr>
                  {selected === x.result_id ? (
                    <tr key={x.result_id + "-detail"} className="bg-brand-50">
                      <td colSpan={6} className="p-3">
                        <div className="space-y-3">
                          <EEOMSelectedSignals row={x} />
                          <EEOMEvolutionHistory
                            rows={historyRows}
                            loading={historyQ.isFetching}
                            error={historyQ.isError ? historyQ.error : null}
                          />
                        </div>
                      </td>
                    </tr>
                  ) : null}
                </Fragment>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function EEOMEvolutionHistory({
  rows,
  loading,
  error,
}: {
  rows: any[];
  loading: boolean;
  error: unknown;
}) {
  const hasRows = rows.length > 0;
  return (
    <div data-testid="marketops-eeom-evolution-history" className="rounded border border-gray-200 bg-white p-3 text-xs shadow-sm dark:border-slate-700 dark:bg-slate-900">
      <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100">
            Earnings setup evolution
          </h3>
          <p className="text-[11px] text-gray-600 dark:text-gray-300">
            Historical point-in-time EEOM rows are preserved here. The main table remains current-only to avoid conflicting default signals.
          </p>
        </div>
        <span className="rounded-full bg-gray-100 px-2 py-1 text-[11px] font-medium text-gray-700 dark:bg-slate-800 dark:text-gray-200">
          {loading ? "Loading history" : `${rows.length} historical rows`}
        </span>
      </div>
      {error ? (
        <p className="mt-3 rounded border border-red-200 bg-red-50 p-2 text-red-700 dark:border-red-900/60 dark:bg-red-950/40 dark:text-red-200">
          Could not load EEOM evolution history.
        </p>
      ) : !hasRows && !loading ? (
        <p className="mt-3 text-gray-600 dark:text-gray-300">
          No historical EEOM rows are available for this asset in the current earnings window.
        </p>
      ) : (
        <div className="mt-3 overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200 text-left dark:divide-slate-700">
            <thead className="text-[11px] uppercase tracking-wide text-gray-500 dark:text-gray-400">
              <tr>
                <th className="py-2 pr-3">Trade date</th>
                <th className="px-3 py-2">Earnings</th>
                <th className="px-3 py-2">Posture</th>
                <th className="px-3 py-2">Score</th>
                <th className="px-3 py-2">Evidence</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100 dark:divide-slate-800">
              {rows.slice(0, 10).map((row) => (
                <tr key={row.result_id}>
                  <td className="py-2 pr-3 font-mono text-[11px] text-gray-700 dark:text-gray-200">
                    {row.trade_date}
                  </td>
                  <td className="px-3 py-2 text-gray-700 dark:text-gray-200">
                    {row.earnings_date}
                  </td>
                  <td className="px-3 py-2 capitalize text-gray-700 dark:text-gray-200">
                    {String(row.posture || "unknown").replace(/_/g, " ")}
                  </td>
                  <td className="px-3 py-2 font-semibold text-gray-900 dark:text-gray-100">
                    {Number(row.score || 0).toFixed(1)} / 10
                  </td>
                  <td className="px-3 py-2 capitalize text-gray-700 dark:text-gray-200">
                    {String(row.evidence_quality || "unknown")}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

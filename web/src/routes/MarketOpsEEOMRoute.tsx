import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
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
  return (
    <div className="space-y-3">
      <div>
        <h1 className="text-lg font-semibold">
          Earnings Event Opportunity Model (EEOM)
        </h1>
        <p className="text-xs text-gray-500">
          Earnings Event Opportunity Model (EEOM): deterministic pre-earnings
          setup quality from persisted MarketOps evidence. It is not an earnings
          forecast, price target, or recommendation.
        </p>
      </div>
      <MarketOpsWatchlistSelector />
      <div className="rounded border border-brand-100 bg-brand-50 p-3 text-xs text-brand-800">
        EEOM scores setup quality from technical, options, Risk/Reward,
        Valuation Composite, DOSM, earnings materiality. Unavailable inputs are
        shown and reweighted; posture describes evidence balance separately.
      </div>
      <EEOMAssessmentGuide />
      {q.isLoading ? (
        <LoadingState label="Loading earnings opportunities..." />
      ) : q.isError ? (
        <ErrorState error={q.error} />
      ) : !rows.length ? (
        <EmptyState message="No point-in-time-known earnings events have eligible EEOM evidence in the next 30 days." />
      ) : (
        <div className="overflow-x-auto rounded border border-gray-200 bg-white">
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
                <>
                  <tr
                    key={x.result_id}
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
                        <EEOMSelectedSignals row={x} />
                      </td>
                    </tr>
                  ) : null}
                </>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

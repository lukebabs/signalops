import { createContext, useContext, useMemo, type ReactNode } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, isApiError } from "../api/client";
import { useTenant } from "../auth/session";
import type { SubscriberWatchlistContext, SubscriberWatchlistContextMode } from "../types";

type MarketOpsWatchlistContextValue = {
  available: boolean;
  loading: boolean;
  context?: SubscriberWatchlistContext;
  selectedKey: string;
  tickerSet: Set<string>;
  setSelection: (mode: SubscriberWatchlistContextMode, listId?: string) => Promise<void>;
};

const MarketOpsWatchlistContext = createContext<MarketOpsWatchlistContextValue | null>(null);

export function MarketOpsWatchlistContextProvider({ children }: { children: ReactNode }) {
  const tenantId = useTenant();
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: ["subscriber-watchlist-context", tenantId],
    queryFn: () => api.getSubscriberWatchlistContext(tenantId),
    retry: false,
    staleTime: 15_000,
  });
  const mutation = useMutation({
    mutationFn: ({ mode, listId }: { mode: SubscriberWatchlistContextMode; listId?: string }) =>
      api.setSubscriberWatchlistContext(tenantId, { selection_mode: mode, list_id: mode === "all" ? "" : listId, provenance: { surface: "marketops.shared_watchlist_selector" } }),
    onSuccess: async (context) => {
      queryClient.setQueryData(["subscriber-watchlist-context", tenantId], context);
      await queryClient.invalidateQueries({ predicate: (entry) => Array.isArray(entry.queryKey) && entry.queryKey[0] === "marketops" });
    },
  });
  const unavailable = query.isError && isApiError(query.error) && query.error.status === 404;
  const value = useMemo<MarketOpsWatchlistContextValue>(() => ({
    available: !unavailable && !!query.data,
    loading: query.isLoading,
    context: query.data,
    selectedKey: query.data?.selection_mode === "all" ? "all" : query.data?.list_id ?? "",
    tickerSet: new Set((query.data?.items ?? []).map(item => item.ticker.toUpperCase())),
    setSelection: async (mode, listId) => { await mutation.mutateAsync({ mode, listId }); },
  }), [unavailable, query.isLoading, query.data, mutation]);
  return <MarketOpsWatchlistContext.Provider value={value}>{children}</MarketOpsWatchlistContext.Provider>;
}

export function useMarketOpsWatchlistContext() {
  const context = useContext(MarketOpsWatchlistContext);
  if (!context) throw new Error("MarketOpsWatchlistContextProvider is required");
  return context;
}

export function MarketOpsWatchlistSelector() {
  const { available, loading, context, selectedKey, setSelection } = useMarketOpsWatchlistContext();
  if (loading || !available || !context) return null;
  return <label className="inline-flex items-center gap-2 text-xs text-gray-600">
    <span className="font-medium text-gray-700">Watchlist</span>
    <select value={selectedKey} onChange={event => {
      const value = event.target.value;
      void setSelection(value === "all" ? "all" : "list", value === "all" ? undefined : value);
    }} className="rounded border border-gray-300 bg-white px-2 py-1.5 text-xs text-gray-800">
      <option value="all">All my watchlists · {context.member_count}</option>
      {context.lists.map(list => <option key={list.list_id} value={list.list_id}>{list.list_name}{list.list_kind === "tenant_default" ? " · Tenant default" : ""}</option>)}
    </select>
  </label>;
}

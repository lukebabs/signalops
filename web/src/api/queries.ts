import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from './client';
import type {
  RawEventFilter,
  NormalizedEventFilter,
  SignalFilter,
  AlertFilter,
  InsightFilter,
  AlertLifecycleMutationOptions,
  InsightLifecycleMutationOptions,
} from '../types';

export const queryKeys = {
  healthz: ['healthz'] as const,
  readyz: ['readyz'] as const,
  runs: (limit: number) => ['runs', limit] as const,
  run: (runId: string) => ['run', runId] as const,
  providerUsage: (runId: string | undefined, limit: number) =>
    ['provider-usage', runId, limit] as const,
  rawEvents: (filter: RawEventFilter) => ['raw-events', filter] as const,
  rawEvent: (eventId: string) => ['raw-event', eventId] as const,
  idempotency: (tenantId: string, sourceId: string, key: string) =>
    ['idempotency', tenantId, sourceId, key] as const,
  catalogSources: (tenantId: string, limit: number) => ['catalog-sources', tenantId, limit] as const,
  catalogPipelines: (tenantId: string, limit: number) => ['catalog-pipelines', tenantId, limit] as const,
  catalogRules: (tenantId: string, limit: number) => ['catalog-rules', tenantId, limit] as const,
  normalizedEvents: (filter: NormalizedEventFilter) => ['normalized-events', filter] as const,
  normalizedEvent: (eventId: string) => ['normalized-event', eventId] as const,
  signals: (filter: SignalFilter) => ['signals', filter] as const,
  signal: (signalId: string) => ['signal', signalId] as const,
  alerts: (filter: AlertFilter) => ['alerts', filter] as const,
  alert: (alertId: string) => ['alert', alertId] as const,
  insights: (filter: InsightFilter) => ['insights', filter] as const,
  insight: (insightId: string) => ['insight', insightId] as const,
};

export function useHealthz() {
  return useQuery({ queryKey: queryKeys.healthz, queryFn: api.healthz, refetchInterval: 15000 });
}

export function useReadyz() {
  return useQuery({ queryKey: queryKeys.readyz, queryFn: api.readyz, refetchInterval: 15000 });
}

export function useRuns(limit: number) {
  return useQuery({ queryKey: queryKeys.runs(limit), queryFn: () => api.listRuns(limit) });
}

export function useRun(runId: string | null) {
  return useQuery({
    queryKey: queryKeys.run(runId ?? ''),
    queryFn: () => api.getRun(runId!),
    enabled: !!runId,
  });
}

export function useProviderUsage(runId: string | null, limit = 50) {
  return useQuery({
    queryKey: queryKeys.providerUsage(runId ?? undefined, limit),
    queryFn: () => api.listProviderUsage(runId ?? undefined, limit),
    enabled: !!runId,
  });
}

// Unfiltered recent provider usage for the Dashboard (no run_id gating).
// Reuses the same query key prefix so DashboardStreamBridge invalidations refresh it.
export function useRecentProviderUsage(limit = 50) {
  return useQuery({
    queryKey: queryKeys.providerUsage(undefined, limit),
    queryFn: () => api.listProviderUsage(undefined, limit),
  });
}

export function useRawEvents(filter: RawEventFilter) {
  return useQuery({
    queryKey: queryKeys.rawEvents(filter),
    queryFn: () => api.listRawEvents(filter),
  });
}

export function useRawEvent(eventId: string | null) {
  return useQuery({
    queryKey: queryKeys.rawEvent(eventId ?? ''),
    queryFn: () => api.getRawEvent(eventId!),
    enabled: !!eventId,
  });
}

// Idempotency lookup is user-triggered; disable retries so a 404 surfaces immediately.
export function useIdempotency(
  tenantId: string,
  sourceId: string,
  key: string,
  enabled: boolean,
) {
  return useQuery({
    queryKey: queryKeys.idempotency(tenantId, sourceId, key),
    queryFn: () => api.getIdempotency(tenantId, sourceId, key),
    enabled: enabled && !!tenantId && !!sourceId && !!key,
    retry: false,
  });
}

export function useCatalogSources(tenantId = 'tenant-local', limit = 50) {
  return useQuery({
    queryKey: queryKeys.catalogSources(tenantId, limit),
    queryFn: () => api.listCatalogSources(tenantId, limit),
  });
}

export function useCatalogPipelines(tenantId = 'tenant-local', limit = 50) {
  return useQuery({
    queryKey: queryKeys.catalogPipelines(tenantId, limit),
    queryFn: () => api.listCatalogPipelines(tenantId, limit),
  });
}

export function useCatalogRules(tenantId = 'tenant-local', limit = 50) {
  return useQuery({
    queryKey: queryKeys.catalogRules(tenantId, limit),
    queryFn: () => api.listCatalogRules(tenantId, limit),
  });
}

export function useNormalizedEvents(filter: NormalizedEventFilter) {
  return useQuery({
    queryKey: queryKeys.normalizedEvents(filter),
    queryFn: () => api.listNormalizedEvents(filter),
  });
}

export function useNormalizedEvent(eventId: string | null) {
  return useQuery({
    queryKey: queryKeys.normalizedEvent(eventId ?? ''),
    queryFn: () => api.getNormalizedEvent(eventId!),
    enabled: !!eventId,
  });
}

export function useSignals(filter: SignalFilter) {
  return useQuery({
    queryKey: queryKeys.signals(filter),
    queryFn: () => api.listSignals(filter),
  });
}

export function useSignal(signalId: string | null) {
  return useQuery({
    queryKey: queryKeys.signal(signalId ?? ''),
    queryFn: () => api.getSignal(signalId!),
    enabled: !!signalId,
  });
}

export function useAlerts(filter: AlertFilter) {
  return useQuery({
    queryKey: queryKeys.alerts(filter),
    queryFn: () => api.listAlerts(filter),
  });
}

export function useAlert(alertId: string | null) {
  return useQuery({
    queryKey: queryKeys.alert(alertId ?? ''),
    queryFn: () => api.getAlert(alertId!),
    enabled: !!alertId,
  });
}

export function useInsights(filter: InsightFilter) {
  return useQuery({
    queryKey: queryKeys.insights(filter),
    queryFn: () => api.listInsights(filter),
  });
}

export function useInsight(insightId: string | null) {
  return useQuery({
    queryKey: queryKeys.insight(insightId ?? ''),
    queryFn: () => api.getInsight(insightId!),
    enabled: !!insightId,
  });
}

// Lifecycle mutations: on success, write the returned record into the detail cache
// (instant update) and invalidate the list prefix so filtered tables + Dashboard
// summaries (which sit under ['alerts']/['insights']) refetch. Actor is the
// placeholder operator-local until real auth lands.
export function useMutateAlertLifecycle() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (options: AlertLifecycleMutationOptions) => api.mutateAlertLifecycle(options),
    onSuccess: (data, variables) => {
      queryClient.setQueryData(queryKeys.alert(variables.alertId), data);
      queryClient.invalidateQueries({ queryKey: ['alerts'] });
    },
  });
}

export function useMutateInsightLifecycle() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (options: InsightLifecycleMutationOptions) => api.mutateInsightLifecycle(options),
    onSuccess: (data, variables) => {
      queryClient.setQueryData(queryKeys.insight(variables.insightId), data);
      queryClient.invalidateQueries({ queryKey: ['insights'] });
    },
  });
}

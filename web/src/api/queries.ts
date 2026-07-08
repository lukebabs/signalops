import { useQuery } from '@tanstack/react-query';
import { api } from './client';
import type { RawEventFilter } from '../types';

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

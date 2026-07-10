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
  ReplayJobFilter,
  ReplayJobCreateRequest,
  ReplayJob,
  MarketOpsAssetFilter,
  MarketOpsDSMArtifactFilter,
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
  replayJobs: (filter: ReplayJobFilter) => ['replay-jobs', filter] as const,
  replayJob: (replayJobId: string) => ['replay-job', replayJobId] as const,
  replayStatus: (tenantId: string, limit?: number) => ['replay-status', tenantId, limit] as const,
  appProfiles: ['app-profiles'] as const,
  marketOpsAssets: (filter: MarketOpsAssetFilter) => ['marketops-assets', filter] as const,
  marketOpsDSMArtifacts: (filter: MarketOpsDSMArtifactFilter) => ['marketops-dsm-artifacts', filter] as const,
  marketOpsDSMArtifact: (artifactId: string) => ['marketops-dsm-artifact', artifactId] as const,
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

// Replay execution is asynchronous: creating a job only queues it. The list
// polls so newly queued/running jobs refresh without a second SSE stream.
export function useReplayJobs(filter: ReplayJobFilter = { tenant_id: 'tenant-local', limit: 50 }) {
  return useQuery({
    queryKey: queryKeys.replayJobs(filter),
    queryFn: () => api.listReplayJobs(filter),
    refetchInterval: 5000,
  });
}

// Detail polls only while a job is still in flight; once terminal it stops.
export function useReplayJob(replayJobId?: string) {
  return useQuery({
    queryKey: queryKeys.replayJob(replayJobId ?? ''),
    queryFn: () => api.getReplayJob(replayJobId!),
    enabled: Boolean(replayJobId),
    refetchInterval: (query) => {
      const status = query.state.data?.replay_job.status;
      return status === 'queued' || status === 'running' ? 3000 : false;
    },
  });
}

// On create, seed the detail cache with the returned (queued) job and
// invalidate the list prefix so filtered tables + Dashboard summary refetch.
export function useCreateReplayJob() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: ReplayJobCreateRequest) => api.createReplayJob(body),
    onSuccess: (data) => {
      queryClient.setQueryData(queryKeys.replayJob(data.replay_job.replay_job_id), data);
      queryClient.invalidateQueries({ queryKey: ['replay-jobs'] });
    },
  });
}

// Replay operations observability (G064): worker health + job counts. REST
// polling at the replay list cadence (5s); participates in manual refresh.
export function useReplayStatus({ tenant_id, limit }: { tenant_id: string; limit?: number }) {
  return useQuery({
    queryKey: queryKeys.replayStatus(tenant_id, limit),
    queryFn: () => api.getReplayStatus({ tenant_id, limit }),
    refetchInterval: 5000,
  });
}

// G066 app profiles (console + marketops). Static backend data; cache 5 min.
export function useAppProfiles() {
  return useQuery({
    queryKey: queryKeys.appProfiles,
    queryFn: api.getAppProfiles,
    staleTime: 5 * 60 * 1000,
  });
}

// G071 MarketOps asset universe (read-only). The seed changes slowly; cache 5 min.
export function useMarketOpsAssets(
  filter: MarketOpsAssetFilter = { tenant_id: 'tenant-local', universe_group: 'top50_megacap', active_only: true, limit: 50 },
) {
  return useQuery({
    queryKey: queryKeys.marketOpsAssets(filter),
    queryFn: () => api.listMarketOpsAssets(filter),
    staleTime: 5 * 60 * 1000,
  });
}

// G078 first-class MarketOps DSM artifact ledger. Artifacts are materialized
// from signal semantic evidence by the backend, so a short stale time keeps the
// workbench responsive while avoiding unnecessary refetch churn.
export function useMarketOpsDSMArtifacts(filter: MarketOpsDSMArtifactFilter = { tenant_id: 'tenant-local', limit: 50 }) {
  return useQuery({
    queryKey: queryKeys.marketOpsDSMArtifacts(filter),
    queryFn: () => api.listMarketOpsDSMArtifacts(filter),
    staleTime: 60 * 1000,
  });
}

export function useMarketOpsDSMArtifact(artifactId: string | null) {
  return useQuery({
    queryKey: queryKeys.marketOpsDSMArtifact(artifactId ?? ''),
    queryFn: () => api.getMarketOpsDSMArtifact(artifactId!),
    enabled: !!artifactId,
    staleTime: 60 * 1000,
  });
}

// Cancel with an optimistic update: mark the matching queued/running job as
// `canceled` in every list cache immediately, roll back on error, then write
// the authoritative returned job into the detail cache and refetch.
type ReplayJobsCache = { replay_jobs: ReplayJob[] };

export function useCancelReplayJob() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (vars: { replayJobId: string; reason?: string; note?: string }) =>
      api.cancelReplayJob(vars.replayJobId, { reason: vars.reason, note: vars.note }),
    onMutate: async ({ replayJobId }) => {
      // Stop in-flight list refetches from clobbering the optimistic update.
      await queryClient.cancelQueries({ queryKey: ['replay-jobs'] });
      const previous = queryClient.getQueriesData<ReplayJobsCache>({ queryKey: ['replay-jobs'] });
      queryClient.setQueriesData<ReplayJobsCache>({ queryKey: ['replay-jobs'] }, (old) => {
        if (!old) return old;
        return {
          replay_jobs: old.replay_jobs.map((j) =>
            j.replay_job_id === replayJobId && (j.status === 'queued' || j.status === 'running')
              ? { ...j, status: 'canceled' }
              : j,
          ),
        };
      });
      return { previous };
    },
    onError: (_err, _vars, context) => {
      // Restore the pre-mutation list caches.
      context?.previous?.forEach(([key, data]) => {
        if (data !== undefined) queryClient.setQueryData(key, data);
      });
    },
    onSuccess: (data, { replayJobId }) => {
      queryClient.setQueryData(queryKeys.replayJob(replayJobId), data);
    },
    onSettled: async (_data, _err, { replayJobId }) => {
      await queryClient.invalidateQueries({ queryKey: ['replay-jobs'] });
      queryClient.invalidateQueries({ queryKey: ['replay-job', replayJobId] });
    },
  });
}

// Lifecycle mutations: on success, write the returned record into the detail cache
// (instant update) and invalidate the list prefix so filtered tables + Dashboard
// summaries (which sit under ['alerts']/['insights']) refetch. When auth is enabled
// the gateway derives the actor from the token; the operator-local placeholder is
// only sent in auth-disabled (local dev) mode — see api/client.ts.
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

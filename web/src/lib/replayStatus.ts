// Pure helpers over the G064 replay operations status response. No React, no
// DOM — fully unit-testable. Tolerate missing job-count keys and no workers.
import type { ReplayOperationsStatus, ReplayWorkerStatusRecord, ReplayWorkerHealth } from '../types';

// Read a status count defensively; the backend 0-fills all five statuses, but
// historical/forward shapes may omit keys.
export function replayJobCount(status: ReplayOperationsStatus | undefined, key: string): number {
  if (!status || !status.job_counts) return 0;
  const v = status.job_counts[key];
  return typeof v === 'number' && Number.isFinite(v) ? v : 0;
}

// Worst worker health across the fleet: error > stale > online. Returns
// 'unknown' when there are no heartbeats.
export function worstReplayWorkerHealth(
  workers: ReplayWorkerStatusRecord[],
): ReplayWorkerHealth | 'unknown' {
  if (!workers || workers.length === 0) return 'unknown';
  const rank = (h: string) => (h === 'error' ? 3 : h === 'stale' ? 2 : h === 'online' ? 1 : 0);
  let worst: ReplayWorkerHealth = workers[0].health;
  for (const w of workers) {
    if (rank(w.health) > rank(worst)) worst = w.health;
  }
  return worst;
}

// Most recent last_seen_at across workers (ISO string compares lexicographically
// for RFC3339). Undefined when there are no workers.
export function latestReplayWorkerSeenAt(workers: ReplayWorkerStatusRecord[]): string | undefined {
  if (!workers || workers.length === 0) return undefined;
  let latest: string | undefined;
  for (const w of workers) {
    if (w.last_seen_at && (latest === undefined || w.last_seen_at > latest)) {
      latest = w.last_seen_at;
    }
  }
  return latest;
}

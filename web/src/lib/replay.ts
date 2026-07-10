// Pure helpers over replay job `result` JSON (raw from the backend). No React,
// no DOM — fully unit-testable. Tolerate both the older G059 result shape
// (scanned/published/max_records/completed_at only) and the G061 shape
// (failed/batches/batch_size/canceled/records).
import type { ReplayResult, ReplayCancellationResult, ReplayRecordResult } from '../types';

// A replay result is arbitrary JSON (json.RawMessage on the backend). Treat any
// non-null, non-array object as a ReplayResult; null/strings/numbers parse to
// undefined so callers fall back to "-".
export function parseReplayResult(result: unknown): ReplayResult | undefined {
  if (!result || typeof result !== 'object' || Array.isArray(result)) return undefined;
  return result as ReplayResult;
}

// Cancellation metadata lives in result.canceled as an object (written by
// CancelReplayJob: {actor, reason, canceled_at}). A boolean `canceled` is the
// normal-completion marker and carries no metadata.
export function cancellationOf(result: unknown): ReplayCancellationResult | undefined {
  const r = parseReplayResult(result);
  const c = r?.canceled;
  if (c && typeof c === 'object' && !Array.isArray(c)) {
    return c as ReplayCancellationResult;
  }
  return undefined;
}

// Only queued/running jobs may be canceled; terminal statuses are not cancelable.
export function isCancelableStatus(status: string): boolean {
  return status === 'queued' || status === 'running';
}

export function replayRecords(result: unknown): ReplayRecordResult[] {
  const r = parseReplayResult(result);
  if (!r || !Array.isArray(r.records)) return [];
  return r.records as ReplayRecordResult[];
}

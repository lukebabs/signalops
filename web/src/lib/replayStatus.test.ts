import { describe, expect, it } from 'vitest';
import { replayJobCount, worstReplayWorkerHealth, latestReplayWorkerSeenAt } from './replayStatus';
import type { ReplayOperationsStatus, ReplayWorkerStatusRecord } from '../types';

function worker(over: Partial<ReplayWorkerStatusRecord> = {}): ReplayWorkerStatusRecord {
  return {
    worker_id: 'w',
    status: 'idle',
    health: 'online',
    process_started_at: '2026-07-10T06:01:36Z',
    last_seen_at: '2026-07-10T06:09:57Z',
    metadata: {},
    created_at: '2026-07-10T06:01:36Z',
    updated_at: '2026-07-10T06:09:57Z',
    ...over,
  };
}

describe('replay status helpers (G065)', () => {
  it('worstReplayWorkerHealth ranks error > stale > online and unknown when empty', () => {
    expect(worstReplayWorkerHealth([])).toBe('unknown');
    expect(worstReplayWorkerHealth([worker({ health: 'online' })])).toBe('online');
    expect(worstReplayWorkerHealth([worker({ health: 'online' }), worker({ health: 'stale' })])).toBe('stale');
    expect(worstReplayWorkerHealth([worker({ health: 'stale' }), worker({ health: 'online' })])).toBe('stale');
    expect(worstReplayWorkerHealth([worker({ health: 'online' }), worker({ health: 'error' })])).toBe('error');
    expect(worstReplayWorkerHealth([worker({ health: 'error' }), worker({ health: 'stale' })])).toBe('error');
  });

  it('replayJobCount returns counts and zero for missing keys/status', () => {
    const status = { job_counts: { queued: 2, running: 1, succeeded: 3 } } as unknown as ReplayOperationsStatus;
    expect(replayJobCount(status, 'queued')).toBe(2);
    expect(replayJobCount(status, 'failed')).toBe(0); // missing key
    expect(replayJobCount(undefined, 'queued')).toBe(0);
  });

  it('latestReplayWorkerSeenAt returns the most recent last_seen_at', () => {
    expect(latestReplayWorkerSeenAt([])).toBeUndefined();
    const ws = [
      worker({ last_seen_at: '2026-07-10T06:09:57Z' }),
      worker({ last_seen_at: '2026-07-10T07:00:00Z' }),
    ];
    expect(latestReplayWorkerSeenAt(ws)).toBe('2026-07-10T07:00:00Z');
  });
});

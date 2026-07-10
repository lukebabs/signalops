import { describe, expect, it } from 'vitest';
import { parseReplayResult, cancellationOf, isCancelableStatus, replayRecords } from './replay';

// Older G059 shape: scanned/published/max_records/completed_at only.
const G059_RESULT = {
  replay_job_id: 'replay-g059-raw',
  source_kind: 'raw_events',
  scanned: 1,
  published: 1,
  max_records: 1,
  completed_at: '2026-07-10T03:00:13.877805066Z',
};

// G061 shape: counters + per-record accounting, canceled: false (bool).
const G061_RESULT = {
  replay_job_id: 'replay-test',
  source_kind: 'raw_events',
  scanned: 3,
  published: 2,
  failed: 1,
  batches: 2,
  max_records: 3,
  batch_size: 2,
  canceled: false,
  started_at: '2026-07-10T04:00:00Z',
  completed_at: '2026-07-10T04:00:03Z',
  records: [
    { source_id: 'event-1', key: 'k1', status: 'published', topic: 'signalops.local.raw.v1', partition: 0, offset: 101, attempts: 1 },
    { source_id: 'event-2', key: 'k2', status: 'failed', attempts: 3, error: 'publish failed' },
  ],
};

// Canceled shape: result.canceled is an object written by CancelReplayJob.
const CANCELED_RESULT = {
  replay_job_id: 'replay-test',
  source_kind: 'raw_events',
  scanned: 1,
  published: 1,
  failed: 0,
  batches: 1,
  max_records: 3,
  batch_size: 1,
  completed_at: '2026-07-10T04:14:46Z',
  canceled: { actor: 'operator-local', reason: 'operator canceled from Replay UI', canceled_at: '2026-07-10T04:14:46Z' },
};

describe('replay result helpers (G062)', () => {
  it('parses object results and rejects non-objects', () => {
    expect(parseReplayResult(G059_RESULT)?.scanned).toBe(1);
    expect(parseReplayResult({})?.scanned).toBeUndefined();
    expect(parseReplayResult(null)).toBeUndefined();
    expect(parseReplayResult('nope')).toBeUndefined();
    expect(parseReplayResult(undefined)).toBeUndefined();
  });

  it('tolerates the older G059 result shape', () => {
    const r = parseReplayResult(G059_RESULT);
    expect(r?.failed).toBeUndefined();
    expect(r?.batches).toBeUndefined();
    expect(r?.records).toBeUndefined();
    expect(replayRecords(G059_RESULT)).toEqual([]);
    expect(cancellationOf(G059_RESULT)).toBeUndefined();
  });

  it('reads G061 counters and records', () => {
    const r = parseReplayResult(G061_RESULT);
    expect(r?.failed).toBe(1);
    expect(r?.batches).toBe(2);
    expect(r?.batch_size).toBe(2);
    expect(r?.canceled).toBe(false);
    const recs = replayRecords(G061_RESULT);
    expect(recs).toHaveLength(2);
    expect(recs[0].status).toBe('published');
    expect(recs[1].error).toBe('publish failed');
  });

  it('extracts cancellation metadata only when canceled is an object', () => {
    expect(cancellationOf(G061_RESULT)).toBeUndefined(); // canceled: false (bool)
    const c = cancellationOf(CANCELED_RESULT);
    expect(c?.actor).toBe('operator-local');
    expect(c?.reason).toBe('operator canceled from Replay UI');
    expect(c?.canceled_at).toBe('2026-07-10T04:14:46Z');
  });

  it('flags only queued/running as cancelable', () => {
    expect(isCancelableStatus('queued')).toBe(true);
    expect(isCancelableStatus('running')).toBe(true);
    expect(isCancelableStatus('succeeded')).toBe(false);
    expect(isCancelableStatus('failed')).toBe(false);
    expect(isCancelableStatus('canceled')).toBe(false);
  });
});

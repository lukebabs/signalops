import { create } from 'zustand';
import type { RawEventFilter } from '../types';

const DEFAULT_RAW_FILTER: RawEventFilter = {
  tenant_id: '',
  source_id: '',
  dataset: '',
  limit: 50,
};

interface UiState {
  selectedRunId: string | null;
  selectedEventId: string | null;
  runsLimit: number;
  rawFilter: RawEventFilter;
  lastRefresh: string | null;
  streamConnected: boolean;
  lastStreamEventAt: string | null;
  streamError: string | null;
  setSelectedRunId: (id: string | null) => void;
  setSelectedEventId: (id: string | null) => void;
  setRunsLimit: (n: number) => void;
  setRawFilter: (f: Partial<RawEventFilter>) => void;
  setLastRefresh: (t: string) => void;
  setStreamConnected: (connected: boolean) => void;
  recordStreamEvent: () => void;
  setStreamError: (message: string | null) => void;
}

export const useUi = create<UiState>((set) => ({
  selectedRunId: null,
  selectedEventId: null,
  runsLimit: 50,
  rawFilter: DEFAULT_RAW_FILTER,
  lastRefresh: null,
  streamConnected: false,
  lastStreamEventAt: null,
  streamError: null,
  setSelectedRunId: (id) => set({ selectedRunId: id }),
  setSelectedEventId: (id) => set({ selectedEventId: id }),
  setRunsLimit: (n) => set({ runsLimit: n }),
  setRawFilter: (f) => set((s) => ({ rawFilter: { ...s.rawFilter, ...f } })),
  setLastRefresh: (t) => set({ lastRefresh: t }),
  setStreamConnected: (connected) =>
    set({ streamConnected: connected, streamError: connected ? null : null }),
  recordStreamEvent: () => set({ lastStreamEventAt: new Date().toISOString() }),
  setStreamError: (message) => set({ streamError: message, streamConnected: false }),
}));

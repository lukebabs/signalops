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
  setSelectedRunId: (id: string | null) => void;
  setSelectedEventId: (id: string | null) => void;
  setRunsLimit: (n: number) => void;
  setRawFilter: (f: Partial<RawEventFilter>) => void;
  setLastRefresh: (t: string) => void;
}

export const useUi = create<UiState>((set) => ({
  selectedRunId: null,
  selectedEventId: null,
  runsLimit: 50,
  rawFilter: DEFAULT_RAW_FILTER,
  lastRefresh: null,
  setSelectedRunId: (id) => set({ selectedRunId: id }),
  setSelectedEventId: (id) => set({ selectedEventId: id }),
  setRunsLimit: (n) => set({ runsLimit: n }),
  setRawFilter: (f) => set((s) => ({ rawFilter: { ...s.rawFilter, ...f } })),
  setLastRefresh: (t) => set({ lastRefresh: t }),
}));

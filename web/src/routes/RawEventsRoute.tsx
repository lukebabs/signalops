import { useRawEvents } from '../api/queries';
import { useUi } from '../store/ui';
import { RawEventTable } from '../components/RawEventTable';
import { RawEventDetail } from '../components/RawEventDetail';
import { LoadingState, ErrorState, EmptyState } from '../components/States';
import { RefreshButton } from '../components/RefreshButton';

export function RawEventsRoute() {
  const rawFilter = useUi((s) => s.rawFilter);
  const setRawFilter = useUi((s) => s.setRawFilter);
  const selectedEventId = useUi((s) => s.selectedEventId);
  const setSelectedEventId = useUi((s) => s.setSelectedEventId);
  const setLastRefresh = useUi((s) => s.setLastRefresh);
  const events = useRawEvents(rawFilter);
  const data = events.data?.raw_events ?? [];

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h1 className="text-lg font-semibold">Raw Events</h1>
        <div className="flex flex-wrap items-center gap-2">
          <input
            placeholder="tenant id"
            value={rawFilter.tenant_id ?? ''}
            onChange={(e) => setRawFilter({ tenant_id: e.target.value })}
            className="rounded border border-gray-300 px-2 py-1 text-sm"
          />
          <input
            placeholder="source id"
            value={rawFilter.source_id ?? ''}
            onChange={(e) => setRawFilter({ source_id: e.target.value })}
            className="rounded border border-gray-300 px-2 py-1 text-sm"
          />
          <input
            placeholder="dataset"
            value={rawFilter.dataset ?? ''}
            onChange={(e) => setRawFilter({ dataset: e.target.value })}
            className="rounded border border-gray-300 px-2 py-1 text-sm"
          />
          <select
            value={rawFilter.limit ?? 50}
            onChange={(e) => setRawFilter({ limit: Number(e.target.value) })}
            className="rounded border border-gray-300 px-2 py-1 text-sm"
          >
            {[25, 50, 100, 200].map((n) => (
              <option key={n} value={n}>
                {n}
              </option>
            ))}
          </select>
          <RefreshButton
            onClick={() => {
              events.refetch();
              setLastRefresh(new Date().toISOString());
            }}
            loading={events.isFetching}
          />
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div className="lg:col-span-2">
          {events.isLoading ? (
            <LoadingState />
          ) : events.isError ? (
            <ErrorState error={events.error} />
          ) : data.length ? (
            <RawEventTable events={data} onSelect={setSelectedEventId} />
          ) : (
            <EmptyState message="No raw events found." />
          )}
        </div>
        <div className="rounded border border-gray-200 bg-white p-3">
          <RawEventDetail eventId={selectedEventId} />
        </div>
      </div>
    </div>
  );
}

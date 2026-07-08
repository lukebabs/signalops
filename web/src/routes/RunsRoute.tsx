import { useRuns } from '../api/queries';
import { useUi } from '../store/ui';
import { RunTable } from '../components/RunTable';
import { RunDetail } from '../components/RunDetail';
import { RunsBarChart } from '../components/RunsBarChart';
import { LoadingState, ErrorState, EmptyState } from '../components/States';
import { RefreshButton } from '../components/RefreshButton';

export function RunsRoute() {
  const runsLimit = useUi((s) => s.runsLimit);
  const setRunsLimit = useUi((s) => s.setRunsLimit);
  const selectedRunId = useUi((s) => s.selectedRunId);
  const setSelectedRunId = useUi((s) => s.setSelectedRunId);
  const setLastRefresh = useUi((s) => s.setLastRefresh);
  const runs = useRuns(runsLimit);
  const data = runs.data?.runs ?? [];

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-semibold">Scheduler Runs</h1>
        <div className="flex items-center gap-2">
          <label className="text-xs text-gray-600">
            Limit
            <select
              value={runsLimit}
              onChange={(e) => setRunsLimit(Number(e.target.value))}
              className="ml-1 rounded border border-gray-300 px-2 py-1 text-sm"
            >
              {[25, 50, 100, 200].map((n) => (
                <option key={n} value={n}>
                  {n}
                </option>
              ))}
            </select>
          </label>
          <RefreshButton
            onClick={() => {
              runs.refetch();
              setLastRefresh(new Date().toISOString());
            }}
            loading={runs.isFetching}
          />
        </div>
      </div>

      <RunsBarChart runs={data} />

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div className="lg:col-span-2">
          {runs.isLoading ? (
            <LoadingState />
          ) : runs.isError ? (
            <ErrorState error={runs.error} />
          ) : data.length ? (
            <RunTable runs={data} onSelect={setSelectedRunId} />
          ) : (
            <EmptyState message="No scheduler runs found." />
          )}
        </div>
        <div className="rounded border border-gray-200 bg-white p-3">
          <RunDetail runId={selectedRunId} />
        </div>
      </div>
    </div>
  );
}

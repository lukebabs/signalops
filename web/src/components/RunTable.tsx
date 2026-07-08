import { useMemo } from 'react';
import { AgGridReact } from 'ag-grid-react';
import type { ColDef } from 'ag-grid-community';
import 'ag-grid-community/styles/ag-grid.css';
import 'ag-grid-community/styles/ag-theme-quartz.css';
import type { SchedulerRun } from '../types';
import { StatusBadge, DryRunBadge } from './StatusBadge';
import { formatUtc, duration, truncate } from '../lib/format';

export function RunTable({
  runs,
  onSelect,
}: {
  runs: SchedulerRun[];
  onSelect: (id: string) => void;
}) {
  const columnDefs = useMemo<ColDef[]>(
    () => [
      {
        headerName: 'Status',
        field: 'status',
        cellRenderer: (p: { value: string }) => <StatusBadge status={p.value} />,
        width: 120,
        sortable: true,
      },
      {
        headerName: 'Started',
        field: 'started_at',
        valueFormatter: (p: { value: string }) => formatUtc(p.value),
        width: 210,
        sortable: true,
      },
      { headerName: 'Source', field: 'source_id', width: 140, sortable: true },
      {
        headerName: 'Datasets',
        field: 'datasets',
        valueFormatter: (p: { value: string[] }) => (Array.isArray(p.value) ? p.value.join(', ') : ''),
        width: 180,
      },
      {
        headerName: 'Dry',
        field: 'dry_run',
        cellRenderer: (p: { value: boolean }) => (p.value ? <DryRunBadge dryRun /> : null),
        width: 80,
      },
      { headerName: 'Built', field: 'events_built', width: 80, sortable: true },
      { headerName: 'Pub', field: 'events_published', width: 80, sortable: true },
      { headerName: 'Req', field: 'provider_requests', width: 80, sortable: true },
      { headerName: 'Fail', field: 'failures', width: 80, sortable: true },
      {
        headerName: 'Duration',
        valueGetter: (p: { data: SchedulerRun }) => duration(p.data?.started_at, p.data?.completed_at),
        width: 100,
      },
      {
        headerName: 'Run ID',
        field: 'run_id',
        valueFormatter: (p: { value: string }) => truncate(p.value ?? '', 36),
        flex: 1,
      },
    ],
    [],
  );

  return (
    <div className="ag-theme-quartz h-[560px] w-full">
      <AgGridReact
        rowData={runs}
        columnDefs={columnDefs}
        defaultColDef={{ resizable: true }}
        getRowId={(p: { data: SchedulerRun }) => p.data.run_id}
        onRowClicked={(e: { data?: SchedulerRun }) => {
          if (e.data) onSelect(e.data.run_id);
        }}
      />
    </div>
  );
}

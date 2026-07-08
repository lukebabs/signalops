import { useMemo } from 'react';
import { AgGridReact } from 'ag-grid-react';
import type { ColDef } from 'ag-grid-community';
import 'ag-grid-community/styles/ag-grid.css';
import 'ag-grid-community/styles/ag-theme-quartz.css';
import type { RawEvent } from '../types';
import { formatUtc, orDash, truncate } from '../lib/format';

export function RawEventTable({
  events,
  onSelect,
}: {
  events: RawEvent[];
  onSelect: (id: string) => void;
}) {
  const columnDefs = useMemo<ColDef[]>(
    () => [
      {
        headerName: 'Event ID',
        field: 'event_id',
        valueFormatter: (p: { value: string }) => truncate(p.value ?? '', 30),
        flex: 1,
      },
      { headerName: 'Dataset', field: 'dataset', width: 180, sortable: true },
      {
        headerName: 'Observation',
        field: 'observation_time',
        valueFormatter: (p: { value: string }) => formatUtc(p.value),
        width: 210,
        sortable: true,
      },
      {
        headerName: 'Processing',
        field: 'processing_time',
        valueFormatter: (p: { value: string }) => formatUtc(p.value),
        width: 210,
        sortable: true,
      },
      {
        headerName: 'Topic',
        field: 'broker_topic',
        valueFormatter: (p: { value?: string }) => orDash(p.value),
        width: 200,
      },
      {
        headerName: 'Part',
        field: 'broker_partition',
        valueFormatter: (p: { value?: number }) => orDash(p.value),
        width: 80,
      },
      {
        headerName: 'Offset',
        field: 'broker_offset',
        valueFormatter: (p: { value?: number }) => orDash(p.value),
        width: 90,
      },
      {
        headerName: 'Idempotency Key',
        field: 'idempotency_key',
        valueFormatter: (p: { value: string }) => truncate(p.value ?? '', 28),
        width: 220,
      },
    ],
    [],
  );

  return (
    <div className="ag-theme-quartz h-[560px] w-full">
      <AgGridReact
        rowData={events}
        columnDefs={columnDefs}
        defaultColDef={{ resizable: true }}
        getRowId={(p: { data: RawEvent }) => p.data.event_id}
        onRowClicked={(e: { data?: RawEvent }) => {
          if (e.data) onSelect(e.data.event_id);
        }}
      />
    </div>
  );
}

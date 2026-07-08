import { useRawEvent } from '../api/queries';
import { JsonViewer } from './JsonViewer';
import { CopyButton } from './CopyButton';
import { MetricTile } from './MetricTile';
import { LoadingState, ErrorState, EmptyState } from './States';
import { formatUtc, orDash } from '../lib/format';

export function RawEventDetail({ eventId }: { eventId: string | null }) {
  const event = useRawEvent(eventId);

  if (!eventId) return <EmptyState message="Select a raw event to inspect details." />;
  if (event.isLoading) return <LoadingState />;
  if (event.isError) return <ErrorState error={event.error} />;
  if (!event.data) return <EmptyState message="No raw event data." />;
  const e = event.data.raw_event;

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <code className="break-all text-xs text-gray-700">{e.event_id}</code>
        <CopyButton value={e.event_id} />
      </div>
      <div className="grid grid-cols-2 gap-2 md:grid-cols-3">
        <MetricTile label="Dataset" value={e.dataset} />
        <MetricTile label="Source" value={e.source_id} />
        <MetricTile label="Topic" value={orDash(e.broker_topic)} />
        <MetricTile label="Partition" value={orDash(e.broker_partition)} />
        <MetricTile label="Offset" value={orDash(e.broker_offset)} />
        <MetricTile label="Observation" value={formatUtc(e.observation_time)} />
        <MetricTile label="Processing" value={formatUtc(e.processing_time)} />
        <MetricTile label="Created" value={formatUtc(e.created_at)} />
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-xs font-medium text-gray-600">Idempotency Key</span>
        <code className="break-all text-xs text-gray-700">{e.idempotency_key}</code>
        <CopyButton value={e.idempotency_key} />
      </div>
      <JsonViewer label="Entity Hints" value={e.entity_hints} />
      <JsonViewer label="Payload" value={e.payload} />
    </div>
  );
}

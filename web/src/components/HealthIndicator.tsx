import { useHealthz, useReadyz } from '../api/queries';
import { useUi } from '../store/ui';

export function HealthIndicator() {
  const healthz = useHealthz();
  const readyz = useReadyz();
  const streamConnected = useUi((s) => s.streamConnected);
  const streamError = useUi((s) => s.streamError);
  const restFallback = useUi((s) => s.streamMode) === 'rest_fallback';

  const ok = healthz.isSuccess && healthz.data?.status === 'ok';
  const ready = readyz.isSuccess && readyz.data?.status === 'ready';
  const unreachable = healthz.isError || readyz.isError;
  // Under auth REST fallback the dashboard stream is intentionally off; health is gated on
  // gateway health alone (not SSE), so the dot stays green when the gateway is healthy.
  const live = ok && ready && (restFallback || streamConnected);

  const dot = live ? 'bg-green-500' : unreachable ? 'bg-red-500' : 'bg-amber-500';
  const label = live
    ? restFallback
      ? 'healthy · REST'
      : 'healthy live'
    : unreachable
      ? 'unreachable'
      : restFallback
        ? 'REST refresh'
        : streamError
          ? 'stream reconnecting'
          : 'checking';

  return (
    <div className="flex items-center gap-2 text-xs text-gray-600">
      <span className={`inline-block h-2.5 w-2.5 rounded-full ${dot}`} />
      <span>{label}</span>
    </div>
  );
}

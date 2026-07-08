import { useHealthz, useReadyz } from '../api/queries';

export function HealthIndicator() {
  const healthz = useHealthz();
  const readyz = useReadyz();

  const ok = healthz.isSuccess && healthz.data?.status === 'ok';
  const ready = readyz.isSuccess && readyz.data?.status === 'ready';
  const unreachable = healthz.isError || readyz.isError;

  const dot = ok && ready ? 'bg-green-500' : unreachable ? 'bg-red-500' : 'bg-amber-500';
  const label = ok && ready ? 'healthy' : unreachable ? 'unreachable' : 'checking';

  return (
    <div className="flex items-center gap-2 text-xs text-gray-600">
      <span className={`inline-block h-2.5 w-2.5 rounded-full ${dot}`} />
      <span>{label}</span>
    </div>
  );
}

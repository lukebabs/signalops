import { useEffect, useState } from 'react';
import { getAccessToken } from '../auth/session';
import { LoadingState, ErrorState } from '../components/States';

export function DagsterAdministrationRoute() {
  const [error, setError] = useState<string | null>(null);
  const [ready, setReady] = useState(false);
  useEffect(() => {
    let cancelled = false;
    const bootstrap = async () => {
      const token = getAccessToken();
      if (!token) { if (!cancelled) setError('Your administrator session has expired. Sign in again to open orchestration.'); return; }
      try {
        const response = await fetch('/v1/administration/orchestration/dagster/session', { method: 'POST', headers: { Authorization: `Bearer ${token}` } });
        if (!response.ok) throw new Error(`Dagster access request failed (${response.status})`);
        if (!cancelled) setReady(true);
      } catch (cause) { if (!cancelled) setError(String((cause as Error)?.message ?? cause)); }
    };
    void bootstrap();
    return () => { cancelled = true; };
  }, []);
  if (error) return <ErrorState error={error} />;
  if (!ready) return <LoadingState label="Opening orchestration observability…" />;
  return (
    <section className="-m-2 flex min-h-[calc(100vh-8rem)] flex-col overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-900">
      <div className="flex items-center justify-between gap-3 border-b border-gray-200 px-4 py-3 dark:border-gray-700">
        <div>
          <h1 className="text-sm font-semibold text-gray-950 dark:text-gray-50">Orchestration</h1>
          <p className="text-xs text-gray-500 dark:text-gray-400">Dagster read-only observability for SignalOps administrators.</p>
        </div>
        <span className="rounded-full border border-emerald-300 bg-emerald-50 px-2 py-1 text-[11px] font-medium text-emerald-700 dark:border-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-300">Read-only</span>
      </div>
      <iframe
        title="Dagster orchestration observability"
        src="/admin/orchestration/dagster/"
        className="min-h-[calc(100vh-11rem)] w-full flex-1 border-0 bg-white dark:bg-gray-950"
        referrerPolicy="same-origin"
      />
    </section>
  );
}

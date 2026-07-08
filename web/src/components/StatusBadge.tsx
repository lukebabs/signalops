const STATUS_STYLES: Record<string, string> = {
  succeeded: 'bg-green-100 text-green-800 border-green-300',
  failed: 'bg-red-100 text-red-800 border-red-300',
  started: 'bg-blue-100 text-blue-800 border-blue-300',
  canceled: 'bg-gray-200 text-gray-700 border-gray-300',
};

export function StatusBadge({ status }: { status: string }) {
  const cls = STATUS_STYLES[status] ?? 'bg-gray-100 text-gray-700 border-gray-300';
  return (
    <span
      className={`inline-flex items-center rounded border px-2 py-0.5 text-xs font-medium ${cls}`}
    >
      {status}
    </span>
  );
}

export function DryRunBadge({ dryRun }: { dryRun: boolean }) {
  if (!dryRun) return null;
  return (
    <span className="inline-flex items-center rounded border border-amber-300 bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-800">
      dry-run
    </span>
  );
}

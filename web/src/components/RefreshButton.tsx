import { RefreshCw } from 'lucide-react';

export function RefreshButton({
  onClick,
  loading,
}: {
  onClick: () => void;
  loading?: boolean;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="inline-flex items-center gap-1 rounded border border-gray-300 bg-white px-2 py-1 text-sm text-gray-700 hover:bg-gray-50"
    >
      <RefreshCw size={14} className={loading ? 'animate-spin' : ''} /> Refresh
    </button>
  );
}

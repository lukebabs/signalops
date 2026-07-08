import { AlertCircle } from 'lucide-react';
import { isApiError } from '../api/client';

export function LoadingState({ label = 'Loading…' }: { label?: string }) {
  return <div className="p-4 text-sm text-gray-500">{label}</div>;
}

export function EmptyState({ message }: { message: string }) {
  return <div className="p-4 text-sm text-gray-500">{message}</div>;
}

export function ErrorState({ error }: { error: unknown }) {
  if (isApiError(error)) {
    return (
      <div className="m-2 rounded border border-red-200 bg-red-50 p-3 text-sm text-red-800">
        <div className="flex items-center gap-2 font-medium">
          <AlertCircle size={14} /> {error.code} ({error.status || 'network'})
        </div>
        <div className="mt-1 break-all text-xs text-red-700">{error.message}</div>
        <div className="mt-1 break-all text-xs text-red-500">{error.endpoint}</div>
      </div>
    );
  }
  return (
    <div className="m-2 rounded border border-red-200 bg-red-50 p-3 text-sm text-red-800">
      <div className="flex items-center gap-2 font-medium">
        <AlertCircle size={14} /> Request failed
      </div>
      <div className="mt-1 text-xs">{String(error)}</div>
    </div>
  );
}

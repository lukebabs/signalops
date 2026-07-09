import { useEffect, useState } from 'react';
import { Activity, LogIn } from 'lucide-react';
import { LoadingState, ErrorState } from '../components/States';
import { useAuth } from './session';

// Compact sign-in screen reusing the shell visual language. One primary action (redirect to IdP).
export function LoginScreen({
  loading,
  error,
  onSignIn,
}: {
  loading?: boolean;
  error?: string | null;
  onSignIn?: () => void;
}) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 p-4">
      <div className="w-full max-w-sm space-y-4 rounded border border-gray-200 bg-white p-6">
        <div className="flex items-center gap-2">
          <Activity size={20} className="text-brand-700" />
          <span className="text-base font-semibold">SignalOps</span>
        </div>
        {loading ? (
          <LoadingState label="Starting session…" />
        ) : (
          <>
            <p className="text-sm text-gray-600">
              Sign in with your Syncratic identity to continue.
            </p>
            {error && <ErrorState error={error} />}
            <button
              type="button"
              onClick={onSignIn}
              className="inline-flex w-full items-center justify-center gap-2 rounded bg-brand-500 px-3 py-2 text-sm text-white hover:bg-brand-700"
            >
              <LogIn size={16} /> Sign in
            </button>
          </>
        )}
      </div>
    </div>
  );
}

// Processes the IdP redirect at /auth/callback, then navigates to the restored path.
export function AuthCallbackProcessor() {
  const { finishCallback } = useAuth();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const path = await finishCallback();
        if (cancelled) return;
        window.location.replace(path || '/');
      } catch (e) {
        if (!cancelled) setError(String((e as Error)?.message ?? e));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [finishCallback]);

  return <LoginScreen loading={error === null} error={error} />;
}

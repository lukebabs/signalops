import { useEffect, useState } from 'react';
import { LogIn, UserPlus } from 'lucide-react';
import syncraticPortalLogo from '../assets/syncratic-portal-logo.svg';
import { LoadingState, ErrorState } from '../components/States';
import { useAuth } from './session';
import { getUserManager, rememberRedirectPath, sanitizeRedirectPath } from './oidc';

// Compact sign-in screen reusing the shell visual language. One primary action (redirect to IdP).
export function LoginScreen({
  loading,
  error,
  onSignIn,
  onSignUp,
}: {
  loading?: boolean;
  error?: string | null;
  onSignIn?: () => void;
  onSignUp?: () => void;
}) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 p-4">
      <div className="w-full max-w-sm space-y-4 rounded border border-gray-200 bg-white p-6">
        <div className="flex items-center gap-2">
          <img src={syncraticPortalLogo} alt="Syncratic" className="h-8 w-auto" />
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
            {onSignUp && (
              <button
                type="button"
                onClick={onSignUp}
                className="inline-flex w-full items-center justify-center gap-2 rounded border border-brand-200 bg-white px-3 py-2 text-sm text-brand-700 hover:bg-brand-50"
              >
                <UserPlus size={16} /> Create account
              </button>
            )}
          </>
        )}
      </div>
    </div>
  );
}


export function AuthLoginRedirectProcessor({ authenticated }: { authenticated: boolean }) {
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const params = new URLSearchParams(window.location.search);
        const destination = sanitizeRedirectPath(params.get('redirect') ?? '/marketops/dashboard');
        if (authenticated) {
          window.location.replace(destination);
          return;
        }
        rememberRedirectPath(destination);
        await getUserManager().signinRedirect();
      } catch (e) {
        // eslint-disable-next-line no-console
        console.error('[signalops] auth login facade failed:', e);
        if (!cancelled) setError(String((e as Error)?.message ?? e));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [authenticated]);

  return (
    <LoginScreen
      loading={error === null}
      error={error}
      onSignIn={error ? () => window.location.assign('/') : undefined}
    />
  );
}

// Processes the IdP redirect at /auth/callback, then navigates to the restored path.
// On failure the underlying oidc-client-ts error is logged and shown on screen so a
// PKCE/state problem is diagnosable instead of silently bouncing to the login screen.
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
        // eslint-disable-next-line no-console
        console.error('[signalops] signinRedirectCallback failed:', e);
        if (!cancelled) setError(String((e as Error)?.message ?? e));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [finishCallback]);

  return (
    <LoginScreen
      loading={error === null}
      error={error}
      onSignIn={error ? () => window.location.assign('/') : undefined}
    />
  );
}

// Used only inside oidc-client-ts's hidden iframe when the IdP does not issue
// a refresh token. The callback returns the authorization response to its parent.
export function SilentRenewProcessor() {
  useEffect(() => {
    void getUserManager().signinSilentCallback();
  }, []);
  return <div aria-live="polite" className="sr-only">Refreshing session…</div>;
}

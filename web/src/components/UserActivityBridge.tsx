import { useEffect, useRef, useState } from 'react';
import { appIdFromPathname, navForApp } from '../apps/appRouting';
import { getAccessToken, useAuth } from '../auth/session';

function currentPathname(): string {
  return typeof window === 'undefined' ? '/' : window.location.pathname;
}

function featureKeyForPath(pathname: string): string {
  const appId = appIdFromPathname(pathname);
  const match = navForApp(appId).find((item) => item.to === pathname);
  if (match) return match.module;
  const parts = pathname.split('/').filter(Boolean);
  if (parts[0] === 'marketops') return parts[1] || 'dashboard';
  return parts[0] || 'console';
}

export function UserActivityBridge() {
  const { authEnabled, authenticated } = useAuth();
  const [pathname, setPathname] = useState(currentPathname);
  const lastRecordedRef = useRef('');

  useEffect(() => {
    if (typeof window === 'undefined') return;
    const update = () => setPathname(window.location.pathname);
    const originalPushState = window.history.pushState;
    const originalReplaceState = window.history.replaceState;
    window.history.pushState = function patchedPushState(...args) {
      const result = originalPushState.apply(this, args);
      update();
      return result;
    };
    window.history.replaceState = function patchedReplaceState(...args) {
      const result = originalReplaceState.apply(this, args);
      update();
      return result;
    };
    window.addEventListener('popstate', update);
    window.addEventListener('hashchange', update);
    update();
    return () => {
      window.history.pushState = originalPushState;
      window.history.replaceState = originalReplaceState;
      window.removeEventListener('popstate', update);
      window.removeEventListener('hashchange', update);
    };
  }, []);

  useEffect(() => {
    const appId = appIdFromPathname(pathname);
    if (!authEnabled || !authenticated || appId !== 'marketops') return;
    const token = getAccessToken();
    if (!token) return;
    const featureKey = featureKeyForPath(pathname);
    const key = `${pathname}:${featureKey}`;
    if (lastRecordedRef.current === key) return;
    lastRecordedRef.current = key;
    void fetch('/v1/session/activity', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({
        event_type: 'feature_view',
        app_id: 'marketops',
        feature_key: featureKey,
        route_path: pathname,
        correlation_id: `feature-view-${Date.now()}`,
      }),
    }).catch(() => undefined);
  }, [authEnabled, authenticated, pathname]);

  return null;
}

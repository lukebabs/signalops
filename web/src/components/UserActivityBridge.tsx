import { useEffect, useMemo, useRef } from 'react';
import { useLocation } from '@tanstack/react-router';
import { appIdFromPathname, navForApp } from '../apps/appRouting';
import { getAccessToken, useAuth } from '../auth/session';

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
  const location = useLocation();
  const pathname = location.pathname;
  const lastRecordedRef = useRef('');
  const appId = useMemo(() => appIdFromPathname(pathname), [pathname]);

  useEffect(() => {
    if (!authEnabled || !authenticated || appId !== 'marketops') return;
    const token = getAccessToken();
    if (!token) return;
    const key = `${pathname}:${featureKeyForPath(pathname)}`;
    if (lastRecordedRef.current === key) return;
    lastRecordedRef.current = key;
    void fetch('/v1/session/activity', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({
        event_type: 'feature_view',
        app_id: 'marketops',
        feature_key: featureKeyForPath(pathname),
        route_path: pathname,
        correlation_id: `feature-view-${Date.now()}`,
      }),
    }).catch(() => undefined);
  }, [appId, authEnabled, authenticated, pathname]);

  return null;
}

import { createContext, useContext, useMemo, type ReactNode } from 'react';
import { useLocation } from '@tanstack/react-router';
import { useAppProfiles } from '../api/queries';
import { CONSOLE_PROFILE, CYBEROPS_PROFILE, MARKETOPS_PROFILE } from './appProfiles';
import {
  appIdFromPathname,
  metadataFilterForApp,
  navForApp,
  type MetadataFilter,
  type NavItem,
} from './appRouting';
import type { AppProfile } from '../types';

interface AppProfileContextValue {
  profiles: AppProfile[];
  currentApp: AppProfile;
  currentAppId: string;
  metadataFilter: MetadataFilter;
  nav: NavItem[];
}

const AppProfileContext = createContext<AppProfileContextValue | null>(null);

export function AppProfileProvider({ children }: { children: ReactNode }) {
  const { data, isError } = useAppProfiles();
  const location = useLocation();

  // Static profiles keep their entry points visible while the app-profile
  // request resolves or a gateway rollout is serving an older profile list.
  // Backend values override these defaults and can add future apps.
  const profiles = useMemo<AppProfile[]>(
    () => {
      const defaults = [CONSOLE_PROFILE, MARKETOPS_PROFILE, CYBEROPS_PROFILE];
      const received = !isError ? data?.app_profiles ?? [] : [];
      return [
        ...defaults.map((profile) => received.find((candidate) => candidate.app_id === profile.app_id) ?? profile),
        ...received.filter((profile) => !defaults.some((fallback) => fallback.app_id === profile.app_id)),
      ];
    },
    [data, isError],
  );

  const appId = appIdFromPathname(location.pathname);

  const value = useMemo<AppProfileContextValue>(() => {
    const currentApp = profiles.find((p) => p.app_id === appId) ?? profiles[0] ?? CONSOLE_PROFILE;
    return {
      profiles,
      currentApp,
      currentAppId: currentApp.app_id,
      metadataFilter: metadataFilterForApp(currentApp.app_id),
      nav: navForApp(currentApp.app_id),
    };
  }, [profiles, appId]);

  return <AppProfileContext.Provider value={value}>{children}</AppProfileContext.Provider>;
}

export function useAppProfile(): AppProfileContextValue {
  const ctx = useContext(AppProfileContext);
  if (!ctx) throw new Error('useAppProfile must be used within AppProfileProvider');
  return ctx;
}

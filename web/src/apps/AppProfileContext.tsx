import { createContext, useContext, useMemo, type ReactNode } from 'react';
import { useLocation } from '@tanstack/react-router';
import { useAppProfiles } from '../api/queries';
import { CONSOLE_PROFILE, MARKETOPS_PROFILE } from './appProfiles';
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

  // Fall back to both static profiles if the request fails or is empty. The
  // console profile is the must-have fallback (keeps the default UI usable);
  // including marketops too means /marketops/* routes scope correctly before the
  // request resolves, instead of flashing unscoped data.
  const profiles = useMemo<AppProfile[]>(
    () => (!isError && data?.app_profiles?.length ? data.app_profiles : [CONSOLE_PROFILE, MARKETOPS_PROFILE]),
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

import { createContext, useContext, useMemo, type ReactNode } from 'react';
import { useLocation } from '@tanstack/react-router';
import { useSessionExperience } from '../api/queries';
import { CONSOLE_PROFILE, CYBEROPS_PROFILE, MARKETOPS_PROFILE } from './appProfiles';
import { useAuth } from '../auth/session';
import {
  appIdFromPathname,
  metadataFilterForApp,
  navForApp,
  type MetadataFilter,
  type NavItem,
} from './appRouting';
import type { AppProfile } from '../types';

interface AppProfileContextValue {
  profiles: Array<AppProfile & { permission?: string }>;
  currentApp: AppProfile;
  currentAppId: string;
  metadataFilter: MetadataFilter;
  nav: NavItem[];
  loading: boolean;
  superAdmin: boolean;
}

const AppProfileContext = createContext<AppProfileContextValue | null>(null);

export function AppProfileProvider({ children }: { children: ReactNode }) {
  const experience = useSessionExperience();
  const { authEnabled } = useAuth();
  const location = useLocation();
  const profiles = useMemo<Array<AppProfile & { permission?: string }>>(() => {
    if (!authEnabled) return [CONSOLE_PROFILE, MARKETOPS_PROFILE, CYBEROPS_PROFILE];
    return experience.data?.app_profiles ?? [];
  }, [authEnabled, experience.data]);
  const appId = appIdFromPathname(location.pathname);
  const superAdmin = !authEnabled || Boolean(experience.data?.super_admin);
  const value = useMemo<AppProfileContextValue>(() => {
    const isAllowed = appId === 'console' ? superAdmin : profiles.some((profile) => profile.app_id === appId);
    const currentApp = profiles.find((profile) => profile.app_id === appId) ?? CONSOLE_PROFILE;
    return {
      profiles,
      currentApp,
      currentAppId: appId,
      metadataFilter: metadataFilterForApp(appId),
      nav: isAllowed ? navForApp(appId) : [],
      loading: authEnabled && experience.isLoading,
      superAdmin,
    };
  }, [profiles, appId, superAdmin, authEnabled, experience.isLoading]);
  return <AppProfileContext.Provider value={value}>{children}</AppProfileContext.Provider>;
}

export function useAppProfile(): AppProfileContextValue {
  const ctx = useContext(AppProfileContext);
  if (!ctx) throw new Error('useAppProfile must be used within AppProfileProvider');
  return ctx;
}

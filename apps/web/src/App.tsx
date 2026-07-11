import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useCallback, useEffect, useMemo, useRef } from 'react';
import { fetchProfile, fetchSession, isMaintenanceModeError, type LayoutMode, type Locale, type ThemeMode } from './lib/api';
import { LeotimeMark } from './lib/leotimeLogo';
import { translate } from './lib/i18n';
import { AuthScreen } from './lib/authUi';
import { DashboardShell } from './features/shell/DashboardShell';
import { usePersistentState } from './lib/persistentState';
import {
  getExperiencePresetDimensions,
  inferExperiencePreset,
  readExperiencePreset,
  readNavigationMode,
  type ExperiencePreset,
  type NamedExperiencePreset,
  type NavigationMode,
} from './lib/experience';
import { useExperienceEffect } from './lib/themeUi';

export function App() {
  const queryClient = useQueryClient();
  const [locale, setLocale] = usePersistentState<Locale>('leotime.locale', 'es');
  const [layoutMode, setLayoutMode] = usePersistentState<LayoutMode>('leotime.layout', 'solid');
  const [themeMode, setThemeMode] = usePersistentState<ThemeMode>('leotime.theme', 'solid');
  const [navigationMode, setNavigationMode] = usePersistentState<NavigationMode>('leotime.nav', readNavigationMode());
  const [preset, setPreset] = usePersistentState<ExperiencePreset>('leotime.preset', readExperiencePreset());
  const profileHydratedRef = useRef(false);
  const preferencesTouchedRef = useRef(false);
  useExperienceEffect({ themeMode, layoutMode, navigationMode, preset });
  const sessionQuery = useQuery({ queryKey: ['session'], queryFn: fetchSession, retry: 1 });
  const profileQuery = useQuery({
    queryKey: ['profile'],
    queryFn: fetchProfile,
    enabled: sessionQuery.data?.authenticated === true,
  });

  const applyLocale = useCallback(
    (value: Locale) => {
      preferencesTouchedRef.current = true;
      setLocale(value);
    },
    [setLocale],
  );
  const applyLayoutMode = useCallback(
    (value: LayoutMode) => {
      preferencesTouchedRef.current = true;
      if (value !== layoutMode) {
        setPreset('custom');
      }
      setLayoutMode(value);
    },
    [layoutMode, setLayoutMode, setPreset],
  );
  const applyNavigationMode = useCallback(
    (value: NavigationMode) => {
      preferencesTouchedRef.current = true;
      if (value !== navigationMode) {
        setPreset('custom');
      }
      setNavigationMode(value);
    },
    [navigationMode, setNavigationMode, setPreset],
  );
  const applyExperiencePreset = useCallback(
    (value: NamedExperiencePreset) => {
      preferencesTouchedRef.current = true;
      const dimensions = getExperiencePresetDimensions(value);
      setThemeMode(dimensions.themeMode);
      setLayoutMode(dimensions.layoutMode);
      setNavigationMode(dimensions.navigationMode);
      setPreset(value);
    },
    [setLayoutMode, setNavigationMode, setPreset, setThemeMode],
  );
  const applyThemeMode = useCallback(
    (value: ThemeMode) => {
      preferencesTouchedRef.current = true;
      if (value !== themeMode) {
        setPreset('custom');
      }
      setThemeMode(value);
    },
    [setPreset, setThemeMode, themeMode],
  );

  useEffect(() => {
    profileHydratedRef.current = false;
    preferencesTouchedRef.current = false;
  }, [sessionQuery.data?.user?.id]);

  useEffect(() => {
    if (!profileQuery.data || profileHydratedRef.current) {
      return;
    }
    profileHydratedRef.current = true;
    if (preferencesTouchedRef.current) {
      return;
    }
    setLocale(profileQuery.data.locale);
    setLayoutMode(profileQuery.data.layoutMode);
    setThemeMode(profileQuery.data.settings.themeMode);
    setPreset(
      inferExperiencePreset({
        themeMode: profileQuery.data.settings.themeMode,
        layoutMode: profileQuery.data.layoutMode,
        navigationMode,
      }),
    );
  }, [navigationMode, profileQuery.data, setLayoutMode, setLocale, setPreset, setThemeMode]);

  const t = useMemo(() => (key: Parameters<typeof translate>[1]) => translate(locale, key), [locale]);

  if (sessionQuery.isLoading) {
    return (
      <main className="boot-screen">
        <LeotimeMark className="boot-logo" size={36} title="leotime" />
        <span>{t('appName')}</span>
      </main>
    );
  }

  if (sessionQuery.isError) {
    const maintenance = isMaintenanceModeError(sessionQuery.error);
    return (
      <main className="boot-screen">
        <LeotimeMark className="boot-logo" size={36} title="leotime" />
        <p>{maintenance ? t('maintenanceModeMessage') : t('sessionLoadFailed')}</p>
        <button
          type="button"
          onClick={() => (maintenance ? window.location.reload() : void sessionQuery.refetch())}
        >
          {maintenance ? t('reloadApp') : t('retry')}
        </button>
      </main>
    );
  }

  if (!sessionQuery.data?.authenticated || !sessionQuery.data.user) {
    return (
      <AuthScreen
        locale={locale}
        onAuthenticated={() => {
          void queryClient.invalidateQueries({ queryKey: ['session'] });
        }}
        setLocale={applyLocale}
        t={t}
      />
    );
  }

  return (
    <DashboardShell
      layoutMode={layoutMode}
      locale={locale}
      navigationMode={navigationMode}
      onApplyExperiencePreset={applyExperiencePreset}
      preset={preset}
      setLayoutMode={applyLayoutMode}
      setLocale={applyLocale}
      setNavigationMode={applyNavigationMode}
      setThemeMode={applyThemeMode}
      themeMode={themeMode}
      t={t}
      user={sessionQuery.data.user}
      userName={sessionQuery.data.user.name}
    />
  );
}

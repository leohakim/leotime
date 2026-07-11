import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useCallback, useEffect, useMemo, useRef } from 'react';
import { fetchProfile, fetchSession, isMaintenanceModeError, type LayoutMode, type Locale, type ThemeMode } from './lib/api';
import { LeotimeMark } from './lib/leotimeLogo';
import { translate } from './lib/i18n';
import { AuthScreen } from './lib/authUi';
import { DashboardShell } from './features/shell/DashboardShell';
import { usePersistentState } from './lib/persistentState';
import { useThemeEffect } from './lib/themeUi';

export function App() {
  const queryClient = useQueryClient();
  const [locale, setLocale] = usePersistentState<Locale>('leotime.locale', 'es');
  const [layoutMode, setLayoutMode] = usePersistentState<LayoutMode>('leotime.layout', 'solid');
  const [themeMode, setThemeMode] = usePersistentState<ThemeMode>('leotime.theme', 'solid');
  const profileHydratedRef = useRef(false);
  const preferencesTouchedRef = useRef(false);
  useThemeEffect(themeMode);
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
      setLayoutMode(value);
    },
    [setLayoutMode],
  );
  const applyThemeMode = useCallback(
    (value: ThemeMode) => {
      preferencesTouchedRef.current = true;
      setThemeMode(value);
    },
    [setThemeMode],
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
  }, [profileQuery.data, setLayoutMode, setLocale, setThemeMode]);

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
      setLayoutMode={applyLayoutMode}
      setLocale={applyLocale}
      setThemeMode={applyThemeMode}
      themeMode={themeMode}
      t={t}
      user={sessionQuery.data.user}
      userName={sessionQuery.data.user.name}
    />
  );
}


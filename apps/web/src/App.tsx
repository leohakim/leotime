import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useMemo } from 'react';
import { fetchSession, type LayoutMode, type Locale, type ThemeMode } from './lib/api';
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
  useThemeEffect(themeMode);
  const sessionQuery = useQuery({ queryKey: ['session'], queryFn: fetchSession, retry: 1 });

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
    return (
      <main className="boot-screen">
        <LeotimeMark className="boot-logo" size={36} title="leotime" />
        <p>{t('sessionLoadFailed')}</p>
        <button type="button" onClick={() => void sessionQuery.refetch()}>
          {t('retry')}
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
        setLocale={setLocale}
        t={t}
      />
    );
  }

  return (
    <DashboardShell
      layoutMode={layoutMode}
      locale={locale}
      setLayoutMode={setLayoutMode}
      setLocale={setLocale}
      setThemeMode={setThemeMode}
      themeMode={themeMode}
      t={t}
      user={sessionQuery.data.user}
      userName={sessionQuery.data.user.name}
    />
  );
}


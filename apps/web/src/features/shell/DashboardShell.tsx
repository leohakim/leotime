import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useEffect, useMemo } from 'react';
import {
  fetchClients,
  fetchProfile,
  fetchProjects,
  fetchTags,
  fetchTasks,
  fetchTimeEntries,
  fetchTimers,
  logout,
  type LayoutMode,
  type Locale,
  type ThemeMode,
  type User,
} from '../../lib/api';
import { ClientPanel } from '../clients/ClientPanel';
import { ProjectPanel } from '../projects/ProjectPanel';
import { TagPanel } from '../tags/TagPanel';
import { TaskPanel } from '../tasks/TaskPanel';
import {
  AppRoute,
  routeShowsTimerBar,
  routeUsesTimesheetEntries,
  useAppRoute,
} from '../../lib/appRoutes';
import { ProfileSettingsPanel } from '../../lib/profileSettingsUi';
import { BackupSettingsPanel } from '../../lib/backupSettingsUi';
import { QueryErrorBanner } from '../../lib/crudFormUi';
import {
  patchTimeEntriesCache,
  refreshOverviewIfOnline,
  removeTimerFromCache,
} from '../../lib/offline/cache';
import { useOfflineStatus } from '../../lib/offline/offlineContext';
import { resetOfflineStorage } from '../../lib/offline/db';
import { isLocalId, stopTimer } from '../../lib/offline/mutations';
import { CalendarPanel } from '../../lib/calendarUi';
import { DashboardPanel } from '../../lib/dashboardUi';
import { ImportExportPanel } from '../../lib/importExportUi';
import { InvoicePanel } from '../../lib/invoiceUi';
import { addMonths, endOfMonth, startOfMonth, toMonthQueryFrom, toMonthQueryTo } from '../../lib/calendarMonth';
import { TimeReportPanel } from '../../lib/reportUi';
import { ManualTimeEntryPanel, TimeEntriesList } from '../../lib/timeEntryUi';
import { TimerCommandRow } from '../../lib/timerUi';
import {
  addWeeks,
  MANUAL_ENTRY_DIRECTORY_DAYS,
  manualEntryDirectoryRange,
  startOfWeek,
  toWeekQueryFrom,
  toWeekQueryTo,
} from '../../lib/timesheetWeek';
import { usePersistentState } from '../../lib/persistentState';
import type { ExperiencePreset, NamedExperiencePreset, NavigationMode } from '../../lib/experience';
import { MobileBottomNav } from './MobileBottomNav';
import { ShellSidebar } from './ShellSidebar';
import { ShellTopbar } from './ShellTopbar';
import { PlaceholderPage } from '../../lib/placeholderPageUi';
import type { Translator } from '../../lib/translator';
import { toastMutationSuccess, useToast } from '../../lib/toast';

type DashboardShellProps = {
  layoutMode: LayoutMode;
  locale: Locale;
  navigationMode: NavigationMode;
  onApplyExperiencePreset: (preset: NamedExperiencePreset) => void;
  preset: ExperiencePreset;
  setLayoutMode: (layoutMode: LayoutMode) => void;
  setLocale: (locale: Locale) => void;
  setNavigationMode: (navigationMode: NavigationMode) => void;
  setThemeMode: (themeMode: ThemeMode) => void;
  themeMode: ThemeMode;
  t: Translator;
  user: User;
  userName: string;
};

type TimeView = 'timesheet' | 'calendar';

function routePageTitle(route: AppRoute, t: Translator): string {
  switch (route) {
    case 'dashboard':
      return t('dashboard');
    case 'timesheet':
    case 'manual-time-entry':
      return t('timeTracker');
    case 'calendar':
      return t('calendar');
    case 'overview':
      return t('reporting');
    case 'detailed':
      return t('detailed');
    case 'shared':
      return t('shared');
    case 'projects':
      return t('projects');
    case 'tasks':
      return t('tasks');
    case 'clients':
      return t('clients');
    case 'tags':
      return t('tags');
    case 'import-export':
      return t('importExport');
    case 'invoices':
      return t('invoices');
    case 'settings':
      return t('settings');
    case 'profile':
      return t('profileSettings');
    default:
      return t('timeTracker');
  }
}

export function DashboardShell({
  layoutMode,
  locale,
  navigationMode,
  onApplyExperiencePreset,
  preset,
  setLayoutMode,
  setLocale,
  setNavigationMode,
  setThemeMode,
  themeMode,
  t,
  user,
  userName,
}: DashboardShellProps) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const { refreshPendingCount } = useOfflineStatus();
  const [route, navigate] = useAppRoute();
  const [timeView, setTimeView] = usePersistentState<TimeView>('leotime.timeView', 'timesheet');
  const [weekAnchorIso, setWeekAnchorIso] = usePersistentState('leotime.timesheetWeek', new Date().toISOString().slice(0, 10));
  const [monthAnchorIso, setMonthAnchorIso] = usePersistentState(
    'leotime.calendarMonth',
    `${new Date().getFullYear()}-${String(new Date().getMonth() + 1).padStart(2, '0')}-01`,
  );
  const [selectedCalendarDay, setSelectedCalendarDay] = usePersistentState<string>('leotime.calendarDay', '');
  const weekAnchor = useMemo(() => new Date(`${weekAnchorIso}T12:00:00`), [weekAnchorIso]);
  const monthAnchor = useMemo(() => new Date(`${monthAnchorIso}T12:00:00`), [monthAnchorIso]);
  const weekStart = useMemo(() => startOfWeek(weekAnchor), [weekAnchor]);
  const weekEnd = useMemo(() => {
    const end = new Date(weekStart);
    end.setDate(end.getDate() + 6);
    return end;
  }, [weekStart]);
  const monthStart = useMemo(() => startOfMonth(monthAnchor), [monthAnchor]);
  const monthEnd = useMemo(() => endOfMonth(monthStart), [monthStart]);
  const weekQueryKey = weekStart.toISOString().slice(0, 10);
  const monthQueryKey = `${monthStart.getFullYear()}-${monthStart.getMonth()}`;

  useEffect(() => {
    if (route === 'calendar') {
      setTimeView('calendar');
    } else if (route === 'timesheet') {
      setTimeView('timesheet');
    }
  }, [route, setTimeView]);

  const needsTimesheetEntries = routeUsesTimesheetEntries(route);
  const activeTimeView: TimeView = route === 'calendar' ? 'calendar' : 'timesheet';
  const manualDirectoryRange = useMemo(() => manualEntryDirectoryRange(), []);

  const clientsQuery = useQuery({
    queryKey: ['clients'],
    queryFn: () => fetchClients({ includeArchived: true }),
  });
  const profileQuery = useQuery({
    queryKey: ['profile'],
    queryFn: fetchProfile,
    retry: 1,
  });
  const taskProjectRequired = profileQuery.data?.settings.taskProjectRequired ?? false;
  const projectsQuery = useQuery({
    queryKey: ['projects'],
    queryFn: () => fetchProjects({ includeArchived: true }),
  });
  const tasksQuery = useQuery({
    queryKey: ['tasks'],
    queryFn: () => fetchTasks({ includeArchived: true }),
  });
  const tagsQuery = useQuery({
    queryKey: ['tags'],
    queryFn: () => fetchTags({ includeArchived: true }),
  });
  const timeEntriesQuery = useQuery({
    queryKey: ['time-entries', activeTimeView, activeTimeView === 'timesheet' ? weekQueryKey : monthQueryKey],
    queryFn: () =>
      activeTimeView === 'timesheet'
        ? fetchTimeEntries({
            from: toWeekQueryFrom(weekStart),
            to: toWeekQueryTo(weekEnd),
          })
        : fetchTimeEntries({
            from: toMonthQueryFrom(monthStart),
            to: toMonthQueryTo(monthEnd),
          }),
    enabled: needsTimesheetEntries,
  });
  const manualDirectoryQuery = useQuery({
    queryKey: ['time-entries', 'manual-directory', manualDirectoryRange.from.slice(0, 10)],
    queryFn: () => fetchTimeEntries(manualDirectoryRange),
    enabled: route === 'manual-time-entry',
  });
  const timersQuery = useQuery({
    queryKey: ['timers'],
    queryFn: fetchTimers,
    refetchInterval: (query) => ((query.state.data?.timers?.length ?? 0) > 0 ? 30_000 : false),
  });
  const openTimers = timersQuery.data?.timers ?? [];
  const activeTimer = openTimers[0] ?? null;
  const stopTimerMutation = useMutation({
    mutationFn: (timeEntryId: string) => {
      const timer = openTimers.find((item) => item.id === timeEntryId);
      if (!timer) {
        throw new Error('timer not found');
      }
      return stopTimer(timeEntryId, timer);
    },
    onSuccess: (entry) => {
      removeTimerFromCache(queryClient, entry.id);
      patchTimeEntriesCache(queryClient, entry);
      void refreshPendingCount();
      void refreshOverviewIfOnline(queryClient);
      if (!isLocalId(entry.id)) {
        queryClient.invalidateQueries({ queryKey: ['timers'] });
        queryClient.invalidateQueries({ queryKey: ['time-entries'] });
      }
      toastMutationSuccess(toast, t, 'timerStopped', entry.id);
    },
    onError: () => toast.error(t('timerStopFailed')),
  });
  const logoutMutation = useMutation({
    mutationFn: logout,
    onSuccess: async () => {
      await resetOfflineStorage();
      queryClient.clear();
      await queryClient.invalidateQueries({ queryKey: ['session'] });
    },
    onError: () => toast.error(t('logoutFailed')),
  });

  return (
    <div className={`app-shell layout-${layoutMode} shell-nav-${navigationMode}`}>
      <ShellSidebar
        activeTimer={activeTimer}
        locale={locale}
        navigate={navigate}
        onStop={(timeEntryId) => stopTimerMutation.mutate(timeEntryId)}
        route={route}
        setLocale={setLocale}
        stoppingTimerId={stopTimerMutation.isPending ? (stopTimerMutation.variables ?? null) : null}
        t={t}
        userName={userName}
      />

      <main className="workspace shell-workspace">
        <ShellTopbar
          layoutMode={layoutMode}
          navigationMode={navigationMode}
          onApplyExperiencePreset={onApplyExperiencePreset}
          onLogout={() => logoutMutation.mutate()}
          pageTitle={routePageTitle(route, t)}
          preset={preset}
          setLayoutMode={setLayoutMode}
          setNavigationMode={setNavigationMode}
          setThemeMode={setThemeMode}
          themeMode={themeMode}
          t={t}
        />

        <div className="page-content">
          {routeShowsTimerBar(route) ? (
            <TimerCommandRow
              clients={clientsQuery.data?.clients ?? []}
              onStop={(timeEntryId) => stopTimerMutation.mutate(timeEntryId)}
              projects={projectsQuery.data?.projects ?? []}
              stoppingTimerId={stopTimerMutation.isPending ? (stopTimerMutation.variables ?? null) : null}
              tags={tagsQuery.data?.tags ?? []}
              taskProjectRequired={taskProjectRequired}
              tasks={tasksQuery.data?.tasks ?? []}
              timers={openTimers}
              t={t}
            />
          ) : null}

          {route === 'dashboard' ? (
            <DashboardPanel
              clients={clientsQuery.data?.clients ?? []}
              locale={locale}
              projects={projectsQuery.data?.projects ?? []}
              tags={tagsQuery.data?.tags ?? []}
              tasks={tasksQuery.data?.tasks ?? []}
              t={t}
            />
          ) : null}

          {route === 'timesheet' || route === 'calendar' ? (
            <TimeViewSwitcher navigate={navigate} t={t} timeView={activeTimeView} />
          ) : null}

          {route === 'timesheet' ? (
            <>
              <QueryErrorBanner error={timeEntriesQuery.error} onRetry={() => void timeEntriesQuery.refetch()} t={t} />
              <TimeEntriesList
                entries={timeEntriesQuery.data?.timeEntries ?? []}
                isLoading={timeEntriesQuery.isLoading}
                locale={locale}
                onNextWeek={() => setWeekAnchorIso(addWeeks(weekAnchor, 1).toISOString().slice(0, 10))}
                onPreviousWeek={() => setWeekAnchorIso(addWeeks(weekAnchor, -1).toISOString().slice(0, 10))}
                onTodayWeek={() => setWeekAnchorIso(new Date().toISOString().slice(0, 10))}
                projects={projectsQuery.data?.projects ?? []}
                taskProjectRequired={taskProjectRequired}
                tasks={tasksQuery.data?.tasks ?? []}
                t={t}
                weekAnchor={weekAnchor}
              />
            </>
          ) : null}

          {route === 'calendar' ? (
            <CalendarPanel
              entries={timeEntriesQuery.data?.timeEntries ?? []}
              isLoading={timeEntriesQuery.isLoading}
              locale={locale}
              monthAnchor={monthAnchor}
              onNextMonth={() => {
                const next = addMonths(monthStart, 1);
                setMonthAnchorIso(`${next.getFullYear()}-${String(next.getMonth() + 1).padStart(2, '0')}-01`);
              }}
              onPreviousMonth={() => {
                const previous = addMonths(monthStart, -1);
                setMonthAnchorIso(`${previous.getFullYear()}-${String(previous.getMonth() + 1).padStart(2, '0')}-01`);
              }}
              onSelectDay={setSelectedCalendarDay}
              onTodayMonth={() => {
                const today = new Date();
                setMonthAnchorIso(`${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}-01`);
              }}
              projects={projectsQuery.data?.projects ?? []}
              selectedDay={selectedCalendarDay}
              tasks={tasksQuery.data?.tasks ?? []}
              t={t}
            />
          ) : null}

          {route === 'overview' ? <TimeReportPanel locale={locale} t={t} /> : null}
          {route === 'detailed' ? <TimeReportPanel detailed locale={locale} t={t} /> : null}
          {route === 'shared' ? <PlaceholderPage titleKey="shared" t={t} /> : null}

          {route === 'invoices' ? <InvoicePanel clients={clientsQuery.data?.clients ?? []} locale={locale} t={t} userName={userName} /> : null}

          {route === 'manual-time-entry' ? (
            <>
              <QueryErrorBanner
                error={manualDirectoryQuery.error}
                onRetry={() => void manualDirectoryQuery.refetch()}
                t={t}
              />
              <ManualTimeEntryPanel
                clients={clientsQuery.data?.clients ?? []}
                directoryDays={MANUAL_ENTRY_DIRECTORY_DAYS}
                isLoading={manualDirectoryQuery.isLoading}
                locale={locale}
                projects={projectsQuery.data?.projects ?? []}
                tags={tagsQuery.data?.tags ?? []}
                taskProjectRequired={taskProjectRequired}
                tasks={tasksQuery.data?.tasks ?? []}
                t={t}
                timeEntries={manualDirectoryQuery.data?.timeEntries ?? []}
              />
            </>
          ) : null}

          {route === 'clients' ? (
            <>
              <QueryErrorBanner error={clientsQuery.error} onRetry={() => void clientsQuery.refetch()} t={t} />
              <ClientPanel clients={clientsQuery.data?.clients ?? []} isLoading={clientsQuery.isLoading} t={t} />
            </>
          ) : null}

          {route === 'projects' ? (
            <>
              <QueryErrorBanner error={projectsQuery.error} onRetry={() => void projectsQuery.refetch()} t={t} />
              <ProjectPanel
                clients={clientsQuery.data?.clients ?? []}
                isLoading={projectsQuery.isLoading}
                projects={projectsQuery.data?.projects ?? []}
                t={t}
              />
            </>
          ) : null}

          {route === 'tasks' ? (
            <>
              <QueryErrorBanner error={tasksQuery.error} onRetry={() => void tasksQuery.refetch()} t={t} />
              <TaskPanel
                isLoading={tasksQuery.isLoading}
                projects={projectsQuery.data?.projects ?? []}
                taskProjectRequired={taskProjectRequired}
                tasks={tasksQuery.data?.tasks ?? []}
                t={t}
              />
            </>
          ) : null}

          {route === 'tags' ? (
            <>
              <QueryErrorBanner error={tagsQuery.error} onRetry={() => void tagsQuery.refetch()} t={t} />
              <TagPanel isLoading={tagsQuery.isLoading} tags={tagsQuery.data?.tags ?? []} t={t} />
            </>
          ) : null}

          {route === 'import-export' ? <ImportExportPanel t={t} /> : null}

          {route === 'settings' || route === 'profile' ? (
            <>
              <ProfileSettingsPanel
                focusSection={route === 'settings' ? 'settings' : undefined}
                layoutMode={layoutMode}
                navigationMode={navigationMode}
                onApplyExperiencePreset={onApplyExperiencePreset}
                preset={preset}
                setLayoutMode={setLayoutMode}
                setLocale={setLocale}
                setNavigationMode={setNavigationMode}
                setThemeMode={setThemeMode}
                t={t}
                themeMode={themeMode}
                user={user}
              />
              <BackupSettingsPanel t={t} />
            </>
          ) : null}
        </div>
      </main>

      <MobileBottomNav locale={locale} navigate={navigate} route={route} setLocale={setLocale} t={t} userName={userName} />
    </div>
  );
}

function TimeViewSwitcher({
  timeView,
  navigate,
  t,
}: {
  timeView: TimeView;
  navigate: (route: AppRoute) => void;
  t: Translator;
}) {
  return (
    <div className="time-view-switcher" role="tablist" aria-label={t('time')}>
      <div className="segmented-control">
        <button
          aria-selected={timeView === 'timesheet'}
          className={timeView === 'timesheet' ? 'selected' : undefined}
          onClick={() => navigate('timesheet')}
          role="tab"
          type="button"
        >
          {t('timesheet')}
        </button>
        <button
          aria-selected={timeView === 'calendar'}
          className={timeView === 'calendar' ? 'selected' : undefined}
          onClick={() => navigate('calendar')}
          role="tab"
          type="button"
        >
          {t('calendar')}
        </button>
      </div>
    </div>
  );
}

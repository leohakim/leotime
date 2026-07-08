import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  BarChart3,
  Building2,
  CalendarDays,
  ChevronDown,
  Clock3,
  Columns3,
  FileText,
  FolderKanban,
  Import,
  Languages,
  LayoutDashboard,
  ListTodo,
  LogOut,
  Minimize2,
  PanelLeft,
  Settings,
  Tags,
} from 'lucide-react';
import { useEffect, useMemo } from 'react';
import {
  fetchClients,
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
import { LeotimeMark } from '../../lib/leotimeLogo';
import { AppRoute, routeHref, routeShowsTimerBar, routeUsesTimeEntries, useAppRoute } from '../../lib/appRoutes';
import { ProfileSettingsPanel } from '../../lib/profileSettingsUi';
import { BackupSettingsPanel } from '../../lib/backupSettingsUi';
import { initials } from '../../lib/crudFormUi';
import {
  patchTimeEntriesCache,
  refreshOverviewIfOnline,
  removeTimerFromCache,
} from '../../lib/offline/cache';
import { useOfflineStatus } from '../../lib/offline/offlineContext';
import { OfflineStatusPill } from '../../lib/offline/offlineStatusUi';
import { resetOfflineStorage } from '../../lib/offline/db';
import { isLocalId, stopTimer } from '../../lib/offline/mutations';
import { CalendarPanel } from '../../lib/calendarUi';
import { DashboardPanel } from '../../lib/dashboardUi';
import { ImportExportPanel } from '../../lib/importExportUi';
import { InvoicePanel } from '../../lib/invoiceUi';
import { addMonths, endOfMonth, startOfMonth, toMonthQueryFrom, toMonthQueryTo } from '../../lib/calendarMonth';
import { TimeReportPanel } from '../../lib/reportUi';
import { ManualTimeEntryPanel, TimeEntriesList } from '../../lib/timeEntryUi';
import { SidebarTimer, TimerCommandRow } from '../../lib/timerUi';
import { addWeeks, startOfWeek, toWeekQueryFrom, toWeekQueryTo } from '../../lib/timesheetWeek';
import { usePersistentState } from '../../lib/persistentState';
import { ThemeSwitcher } from '../../lib/themeUi';
import { PlaceholderPage } from '../../lib/placeholderPageUi';
import type { Translator } from '../../lib/translator';
import { toastMutationSuccess, useToast } from '../../lib/toast';

type DashboardShellProps = {
  layoutMode: LayoutMode;
  locale: Locale;
  setLayoutMode: (layoutMode: LayoutMode) => void;
  setLocale: (locale: Locale) => void;
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
      return t('overview');
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

function isReportingRoute(route: AppRoute): boolean {
  return route === 'overview' || route === 'detailed' || route === 'shared';
}

export function DashboardShell({ layoutMode, locale, setLayoutMode, setLocale, setThemeMode, themeMode, t, user, userName }: DashboardShellProps) {
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

  const needsTimeEntries = routeUsesTimeEntries(route);
  const activeTimeView: TimeView = route === 'calendar' ? 'calendar' : 'timesheet';

  const clientsQuery = useQuery({
    queryKey: ['clients'],
    queryFn: () => fetchClients({ includeArchived: true }),
  });
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
    enabled: needsTimeEntries,
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
    <div className={`app-shell layout-${layoutMode}`}>
      <aside className="sidebar" aria-label="Primary">
        <div className="org-switcher">
          <LeotimeMark className="org-avatar-logo" size={30} title="leotime" />
          <span>{t('organizationName')}</span>
          <ChevronDown aria-hidden="true" />
        </div>

        <SidebarTimer
          activeTimer={activeTimer}
          onStop={(timeEntryId) => stopTimerMutation.mutate(timeEntryId)}
          stoppingTimerId={stopTimerMutation.isPending ? (stopTimerMutation.variables ?? null) : null}
          t={t}
        />

        <nav className="sidebar-nav">
          <a className={route === 'dashboard' ? 'active' : ''} href={routeHref('dashboard')} onClick={(event) => { event.preventDefault(); navigate('dashboard'); }}>
            <LayoutDashboard aria-hidden="true" />
            {t('dashboard')}
          </a>
          <a className={route === 'timesheet' ? 'active' : ''} href={routeHref('timesheet')} onClick={(event) => { event.preventDefault(); navigate('timesheet'); }}>
            <Clock3 aria-hidden="true" />
            {t('time')}
          </a>
          <a className={route === 'calendar' ? 'active' : ''} href={routeHref('calendar')} onClick={(event) => { event.preventDefault(); navigate('calendar'); }}>
            <CalendarDays aria-hidden="true" />
            {t('calendar')}
          </a>
          <a className={`nav-parent${isReportingRoute(route) ? ' active' : ''}`} href={routeHref('overview')} onClick={(event) => { event.preventDefault(); navigate('overview'); }}>
            <BarChart3 aria-hidden="true" />
            {t('reporting')}
            <ChevronDown aria-hidden="true" />
          </a>
          <div className="nav-children" aria-label={t('reporting')}>
            <a className={route === 'overview' ? 'active' : ''} href={routeHref('overview')} onClick={(event) => { event.preventDefault(); navigate('overview'); }}>
              {t('overview')}
            </a>
            <a className={route === 'detailed' ? 'active' : ''} href={routeHref('detailed')} onClick={(event) => { event.preventDefault(); navigate('detailed'); }}>
              {t('detailed')}
            </a>
            <a className={route === 'shared' ? 'active' : ''} href={routeHref('shared')} onClick={(event) => { event.preventDefault(); navigate('shared'); }}>
              {t('shared')}
            </a>
          </div>

          <span className="nav-section-label">{t('manage')}</span>
          <a className={route === 'projects' ? 'active' : ''} href={routeHref('projects')} onClick={(event) => { event.preventDefault(); navigate('projects'); }}>
            <FolderKanban aria-hidden="true" />
            {t('projects')}
          </a>
          <a className={route === 'tasks' ? 'active' : ''} href={routeHref('tasks')} onClick={(event) => { event.preventDefault(); navigate('tasks'); }}>
            <ListTodo aria-hidden="true" />
            {t('tasks')}
          </a>
          <a className={route === 'clients' ? 'active' : ''} href={routeHref('clients')} onClick={(event) => { event.preventDefault(); navigate('clients'); }}>
            <Building2 aria-hidden="true" />
            {t('clients')}
          </a>
          <a className={route === 'tags' ? 'active' : ''} href={routeHref('tags')} onClick={(event) => { event.preventDefault(); navigate('tags'); }}>
            <Tags aria-hidden="true" />
            {t('tags')}
          </a>

          <span className="nav-section-label">{t('admin')}</span>
          <a className={route === 'import-export' ? 'active' : ''} href={routeHref('import-export')} onClick={(event) => { event.preventDefault(); navigate('import-export'); }}>
            <Import aria-hidden="true" />
            {t('importExport')}
          </a>
          <a className={route === 'invoices' ? 'active' : ''} href={routeHref('invoices')} onClick={(event) => { event.preventDefault(); navigate('invoices'); }}>
            <FileText aria-hidden="true" />
            {t('invoices')}
          </a>
          <a className={route === 'settings' ? 'active' : ''} href={routeHref('settings')} onClick={(event) => { event.preventDefault(); navigate('settings'); }}>
            <Settings aria-hidden="true" />
            {t('settings')}
          </a>
        </nav>

        <div className="sidebar-footer">
          <button type="button" title={t('language')} onClick={() => setLocale(locale === 'es' ? 'en' : 'es')}>
            <Languages aria-hidden="true" />
          </button>
          <a className={route === 'profile' ? 'active' : ''} href={routeHref('profile')} onClick={(event) => { event.preventDefault(); navigate('profile'); }}>
            <Settings aria-hidden="true" />
            {t('profileSettings')}
          </a>
          <div className="profile-avatar" aria-hidden="true">
            {initials(userName)}
          </div>
        </div>
      </aside>

      <main className="workspace">
        <header className="tracker-topbar">
          <div className="tracker-title">
            <LeotimeMark size={18} />
            <h1>{routePageTitle(route, t)}</h1>
          </div>
          <div className="toolbar">
            <OfflineStatusPill t={t} />
            <ThemeSwitcher setThemeMode={setThemeMode} themeMode={themeMode} t={t} />
            <LayoutSwitcher layoutMode={layoutMode} setLayoutMode={setLayoutMode} t={t} />
            <button type="button" title={t('logout')} onClick={() => logoutMutation.mutate()}>
              <LogOut aria-hidden="true" />
            </button>
          </div>
        </header>

        <div className="page-content">
          {routeShowsTimerBar(route) ? (
            <TimerCommandRow
              onStop={(timeEntryId) => stopTimerMutation.mutate(timeEntryId)}
              projects={projectsQuery.data?.projects ?? []}
              stoppingTimerId={stopTimerMutation.isPending ? (stopTimerMutation.variables ?? null) : null}
              tags={tagsQuery.data?.tags ?? []}
              tasks={tasksQuery.data?.tasks ?? []}
              timers={openTimers}
              t={t}
            />
          ) : null}

          {route === 'dashboard' ? <DashboardPanel locale={locale} t={t} /> : null}

          {route === 'timesheet' || route === 'calendar' ? (
            <TimeViewSwitcher navigate={navigate} t={t} timeView={activeTimeView} />
          ) : null}

          {route === 'timesheet' ? (
            <TimeEntriesList
              entries={timeEntriesQuery.data?.timeEntries ?? []}
              isLoading={timeEntriesQuery.isLoading}
              locale={locale}
              onNextWeek={() => setWeekAnchorIso(addWeeks(weekAnchor, 1).toISOString().slice(0, 10))}
              onPreviousWeek={() => setWeekAnchorIso(addWeeks(weekAnchor, -1).toISOString().slice(0, 10))}
              onTodayWeek={() => setWeekAnchorIso(new Date().toISOString().slice(0, 10))}
              projects={projectsQuery.data?.projects ?? []}
              tasks={tasksQuery.data?.tasks ?? []}
              t={t}
              weekAnchor={weekAnchor}
            />
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
            <ManualTimeEntryPanel
              clients={clientsQuery.data?.clients ?? []}
              isLoading={timeEntriesQuery.isLoading}
              locale={locale}
              projects={projectsQuery.data?.projects ?? []}
              tags={tagsQuery.data?.tags ?? []}
              tasks={tasksQuery.data?.tasks ?? []}
              t={t}
              timeEntries={timeEntriesQuery.data?.timeEntries ?? []}
            />
          ) : null}

          {route === 'clients' ? <ClientPanel clients={clientsQuery.data?.clients ?? []} isLoading={clientsQuery.isLoading} t={t} /> : null}

          {route === 'projects' ? (
            <ProjectPanel
              clients={clientsQuery.data?.clients ?? []}
              isLoading={projectsQuery.isLoading}
              projects={projectsQuery.data?.projects ?? []}
              t={t}
            />
          ) : null}

          {route === 'tasks' ? (
            <TaskPanel isLoading={tasksQuery.isLoading} projects={projectsQuery.data?.projects ?? []} tasks={tasksQuery.data?.tasks ?? []} t={t} />
          ) : null}

          {route === 'tags' ? <TagPanel isLoading={tagsQuery.isLoading} tags={tagsQuery.data?.tags ?? []} t={t} /> : null}

          {route === 'import-export' ? <ImportExportPanel t={t} /> : null}

          {route === 'settings' || route === 'profile' ? (
            <>
              <ProfileSettingsPanel
                focusSection={route === 'settings' ? 'settings' : undefined}
                setLayoutMode={setLayoutMode}
                setLocale={setLocale}
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

function LayoutSwitcher({
  layoutMode,
  setLayoutMode,
  t,
}: {
  layoutMode: LayoutMode;
  setLayoutMode: (layoutMode: LayoutMode) => void;
  t: Translator;
}) {
  const options: Array<{ value: LayoutMode; label: string; icon: typeof PanelLeft }> = [
    { value: 'solid', label: t('solid'), icon: PanelLeft },
    { value: 'minimal', label: t('minimal'), icon: Minimize2 },
    { value: 'compact', label: t('compact'), icon: Columns3 },
  ];

  return (
    <div className="segmented-control" aria-label={t('layout')}>
      {options.map((option) => {
        const Icon = option.icon;
        return (
          <button
            className={layoutMode === option.value ? 'selected' : ''}
            key={option.value}
            onClick={() => setLayoutMode(option.value)}
            title={option.label}
            type="button"
          >
            <Icon aria-hidden="true" />
          </button>
        );
      })}
    </div>
  );
}

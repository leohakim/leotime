import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  BadgeDollarSign,
  BarChart3,
  CalendarDays,
  Building2,
  ChevronDown,
  Clock3,
  CircleAlert,
  CircleCheck,
  Columns3,
  DollarSign,
  EllipsisVertical,
  FileText,
  Pencil,
  FolderKanban,
  Import,
  Languages,
  LayoutDashboard,
  ListTodo,
  LogOut,
  Mail,
  Minimize2,
  PanelLeft,
  Play,
  Plus,
  RotateCcw,
  Save,
  Settings,
  Tag,
  Tags,
  Trash2,
  X,
} from 'lucide-react';
import { FormEvent, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  archiveClient,
  archiveProject,
  archiveTag,
  archiveTask,
  fetchClients,
  fetchProjects,
  fetchSession,
  fetchTags,
  fetchTasks,
  fetchTimeEntries,
  fetchTimers,
  login,
  logout,
  restoreClient,
  restoreProject,
  restoreTag,
  restoreTask,
  updateClient,
  updateProject,
  updateTag,
  updateTask,
  type Client,
  type ClientInput,
  type LayoutMode,
  type Locale,
  type ThemeMode,
  type Project,
  type ProjectInput,
  type Tag as TagRecord,
  type TagInput,
  type Task,
  type TaskInput,
  type User,
} from './lib/api';
import { LeotimeLogo, LeotimeMark } from './lib/leotimeLogo';
import { translate } from './lib/i18n';
import { AppRoute, routeHref, routeShowsTimerBar, routeUsesTimeEntries, useAppRoute } from './lib/appRoutes';
import { PlaceholderPage } from './lib/placeholderPageUi';
import { ProfileSettingsPanel } from './lib/profileSettingsUi';
import {
  patchClientsCache,
  patchProjectsCache,
  patchTagsCache,
  patchTasksCache,
  patchTimeEntriesCache,
  refreshOverviewIfOnline,
  removeTimerFromCache,
} from './lib/offline/cache';
import { useOfflineStatus } from './lib/offline/offlineContext';
import { OfflineStatusPill } from './lib/offline/offlineStatusUi';
import {
  createClient,
  createProject,
  createTag,
  createTask,
  isLocalId,
  stopTimer,
} from './lib/offline/mutations';
import { sortTasksByNewest } from './lib/taskSort';
import { ProjectBadge } from './lib/projectBadgeUi';
import { CalendarPanel } from './lib/calendarUi';
import { DashboardPanel } from './lib/dashboardUi';
import { ImportExportPanel } from './lib/importExportUi';
import { InvoicePanel } from './lib/invoiceUi';
import { addMonths, endOfMonth, startOfMonth, toMonthQueryFrom, toMonthQueryTo } from './lib/calendarMonth';
import { TimeReportPanel } from './lib/reportUi';
import { ManualTimeEntryPanel, TimeEntriesList } from './lib/timeEntryUi';
import { SidebarTimer, TimerCommandRow } from './lib/timerUi';
import { addWeeks, startOfWeek, toWeekQueryFrom, toWeekQueryTo } from './lib/timesheetWeek';
import { usePersistentState } from './lib/persistentState';
import { ThemeSwitcher, useThemeEffect } from './lib/themeUi';
import { toastMutationSuccess, useToast } from './lib/toast';

export function App() {
  const [locale, setLocale] = usePersistentState<Locale>('leotime.locale', 'es');
  const [layoutMode, setLayoutMode] = usePersistentState<LayoutMode>('leotime.layout', 'solid');
  const [themeMode, setThemeMode] = usePersistentState<ThemeMode>('leotime.theme', 'solid');
  useThemeEffect(themeMode);
  const sessionQuery = useQuery({ queryKey: ['session'], queryFn: fetchSession });

  const t = useMemo(() => (key: Parameters<typeof translate>[1]) => translate(locale, key), [locale]);

  if (sessionQuery.isLoading) {
    return (
      <main className="boot-screen">
        <LeotimeMark className="boot-logo" size={36} title="leotime" />
        <span>{t('appName')}</span>
      </main>
    );
  }

  if (!sessionQuery.data?.authenticated || !sessionQuery.data.user) {
    return <LoginScreen locale={locale} setLocale={setLocale} t={t} />;
  }

  return (
    <Dashboard
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

type Translator = (key: Parameters<typeof translate>[1]) => string;

type LoginScreenProps = {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: Translator;
};

function LoginScreen({ locale, setLocale, t }: LoginScreenProps) {
  const queryClient = useQueryClient();
  const [email, setEmail] = useState('admin@example.com');
  const [password, setPassword] = useState('change-me-now');
  const loginMutation = useMutation({
    mutationFn: () => login(email, password),
    onSuccess: (session) => {
      queryClient.setQueryData(['session'], session);
    },
  });

  function onSubmit(event: FormEvent) {
    event.preventDefault();
    loginMutation.mutate();
  }

  return (
    <main className="login-screen">
      <section className="login-panel" aria-labelledby="login-title">
        <LeotimeLogo className="brand-row" markSize={28} />
        <h1 id="login-title">{t('welcome')}</h1>
        <form onSubmit={onSubmit} className="login-form">
          <label>
            {t('email')}
            <input value={email} onChange={(event) => setEmail(event.target.value)} type="email" />
          </label>
          <label>
            {t('password')}
            <input value={password} onChange={(event) => setPassword(event.target.value)} type="password" />
          </label>
          <button type="submit" disabled={loginMutation.isPending}>
            <Play aria-hidden="true" />
            {t('login')}
          </button>
        </form>
        <button className="ghost-button" type="button" onClick={() => setLocale(locale === 'es' ? 'en' : 'es')}>
          <Languages aria-hidden="true" />
          {t('language')}
        </button>
      </section>
    </main>
  );
}

type DashboardProps = {
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

function Dashboard({ layoutMode, locale, setLayoutMode, setLocale, setThemeMode, themeMode, t, user, userName }: DashboardProps) {
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
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['session'] }),
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
            <ProfileSettingsPanel
              focusSection={route === 'settings' ? 'settings' : undefined}
              setLayoutMode={setLayoutMode}
              setLocale={setLocale}
              setThemeMode={setThemeMode}
              t={t}
              themeMode={themeMode}
              user={user}
            />
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

type ClientFormState = Omit<ClientInput, 'defaultHourlyRateMinor'> & {
  hourlyRate: string;
  active: boolean;
};

type ClientFormErrors = Partial<Record<keyof ClientFormState | 'form', string>>;

const emptyClientForm: ClientFormState = {
  name: '',
  email: '',
  taxId: '',
  billingAddress: '',
  defaultCurrency: 'EUR',
  hourlyRate: '',
  active: true,
};

function ClientPanel({ clients, isLoading, t }: { clients: Client[]; isLoading: boolean; t: Translator }) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const { refreshPendingCount } = useOfflineStatus();
  const [editingClientId, setEditingClientId] = useState<string | null>(null);
  const [form, setForm] = useState<ClientFormState>(emptyClientForm);
  const [errors, setErrors] = useState<ClientFormErrors>({});

  const createMutation = useMutation({
    mutationFn: createClient,
    onSuccess: (client) => {
      setForm(emptyClientForm);
      setErrors({});
      patchClientsCache(queryClient, client);
      void refreshPendingCount();
      void refreshOverviewIfOnline(queryClient);
      if (!isLocalId(client.id)) {
        queryClient.invalidateQueries({ queryKey: ['clients'] });
      }
      toastMutationSuccess(toast, t, 'clientCreated', client.id);
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('clientSaveFailed') }));
      toast.error(t('clientSaveFailed'));
    },
  });

  const updateMutation = useMutation({
    mutationFn: async ({
      clientId,
      input,
      active,
      wasActive,
    }: {
      clientId: string;
      input: ClientInput;
      active: boolean;
      wasActive: boolean;
    }) => {
      const updated = await updateClient(clientId, input);
      if (active && !wasActive) {
        await restoreClient(clientId);
      } else if (!active && wasActive) {
        await archiveClient(clientId);
      }
      return updated;
    },
    onSuccess: () => {
      setEditingClientId(null);
      setForm(emptyClientForm);
      setErrors({});
      queryClient.invalidateQueries({ queryKey: ['clients'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
      toast.success(t('clientUpdated'));
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('clientSaveFailed') }));
      toast.error(t('clientSaveFailed'));
    },
  });

  const archiveMutation = useMutation({
    mutationFn: archiveClient,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['clients'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
      toast.success(t('clientArchived'));
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('clientArchiveFailed') }));
      toast.error(t('clientArchiveFailed'));
    },
  });

  const restoreMutation = useMutation({
    mutationFn: restoreClient,
    onSuccess: () => {
      setEditingClientId(null);
      setForm(emptyClientForm);
      setErrors({});
      queryClient.invalidateQueries({ queryKey: ['clients'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
      toast.success(t('clientRestored'));
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('clientSaveFailed') }));
      toast.error(t('clientSaveFailed'));
    },
  });

  function submitClient(event: FormEvent) {
    event.preventDefault();
    const validation = validateClientForm(form, t);
    setErrors(validation);
    if (hasErrors(validation)) {
      return;
    }

    const input = clientFormToInput(form);
    if (editingClientId) {
      const client = clients.find((item) => item.id === editingClientId);
      updateMutation.mutate({
        clientId: editingClientId,
        input,
        active: form.active,
        wasActive: !client?.archivedAt,
      });
      return;
    }
    createMutation.mutate(input);
  }

  function updateField<K extends keyof ClientFormState>(field: K, value: ClientFormState[K]) {
    const next = { ...form, [field]: value };
    setForm(next);
    if (hasErrors(errors)) {
      setErrors(validateClientForm(next, t));
    }
  }

  function startEditing(client: Client) {
    setEditingClientId(client.id);
    setErrors({});
    setForm({
      name: client.name,
      email: client.email,
      taxId: client.taxId,
      billingAddress: client.billingAddress,
      defaultCurrency: client.defaultCurrency,
      hourlyRate: formatRateInput(client.defaultHourlyRateMinor),
      active: !client.archivedAt,
    });
  }

  function cancelEditing() {
    setEditingClientId(null);
    setForm(emptyClientForm);
    setErrors({});
  }

  const isSaving = createMutation.isPending || updateMutation.isPending;
  const activeClientCount = clients.filter((client) => !client.archivedAt).length;
  const activeClients = clients.filter((client) => !client.archivedAt);
  const inactiveClients = clients.filter((client) => client.archivedAt);

  function renderClientRow(client: Client, isActive: boolean) {
    return (
      <article
        className={editingClientId === client.id ? 'client-row selected' : isActive ? 'client-row' : 'client-row archived'}
        key={client.id}
      >
        <div className="client-row-main">
          <div className="client-avatar" aria-hidden="true">
            {client.name.slice(0, 1).toUpperCase()}
          </div>
          <div className="client-row-copy">
            <div className="client-row-title">
              <strong>{client.name}</strong>
              <span className={isActive ? 'status-pill' : 'status-pill warning-pill'}>
                {isActive ? <CircleCheck aria-hidden="true" /> : null}
                {isActive ? t('active') : t('inactive')}
              </span>
            </div>
            <span className="client-contact">
              <Mail aria-hidden="true" />
              {client.email || t('noContact')}
            </span>
          </div>
        </div>
        <div className="client-row-meta">
          <span className="rate-pill">
            <BadgeDollarSign aria-hidden="true" />
            {client.defaultCurrency} {formatMinor(client.defaultHourlyRateMinor)}/h
          </span>
          {client.taxId ? <span>{client.taxId}</span> : null}
        </div>
        <div className="client-row-actions">
          <button
            className="secondary-button icon-button"
            type="button"
            onClick={() => startEditing(client)}
            title={t('edit')}
          >
            <Pencil aria-hidden="true" />
          </button>
          {isActive ? (
            <button
              className="secondary-button icon-button danger-button"
              type="button"
              onClick={() => archiveMutation.mutate(client.id)}
              title={t('archive')}
            >
              <Trash2 aria-hidden="true" />
            </button>
          ) : (
            <button
              className="secondary-button icon-button"
              type="button"
              onClick={() => restoreMutation.mutate(client.id)}
              title={t('reactivate')}
            >
              <RotateCcw aria-hidden="true" />
            </button>
          )}
        </div>
      </article>
    );
  }

  return (
    <section className="clients-section" id="clients" aria-labelledby="clients-title">
      <div className="clients-heading">
        <div className="section-title-group">
          <span className="section-kicker">
            <Building2 aria-hidden="true" />
            {t('clients')}
          </span>
          <h2 id="clients-title">{t('clientDirectory')}</h2>
          <p>{t('clientPanelSubtitle')}</p>
        </div>
        <button className="secondary-button" type="button" onClick={cancelEditing}>
          <Plus aria-hidden="true" />
          {t('newClient')}
        </button>
      </div>

      <div className="clients-workbench">
        <div className="client-directory">
          <div className="directory-toolbar">
            <div>
              <span>{t('activeClients')}</span>
              <strong>{activeClientCount}</strong>
            </div>
            {isLoading ? (
              <span className="sync-pill">{t('loading')}</span>
            ) : (
              <span className="sync-pill">{t('synced')}</span>
            )}
          </div>

          <div className="client-list" aria-busy={isLoading}>
            {clients.length === 0 ? (
              <div className="empty-state">
                <Building2 aria-hidden="true" />
                <p>{t('noClients')}</p>
              </div>
            ) : null}
            {activeClients.map((client) => renderClientRow(client, true))}
          </div>

          <DirectoryInactiveHeading count={inactiveClients.length} t={t} />
          {inactiveClients.length > 0 ? (
            <div className="client-list client-list-inactive" aria-label={t('inactiveDirectory')}>
              {inactiveClients.map((client) => renderClientRow(client, false))}
            </div>
          ) : null}
        </div>

        <form className="client-editor" noValidate onSubmit={submitClient}>
          <div className="editor-header">
            <div>
              <span>{editingClientId ? t('editingClient') : t('newClient')}</span>
              <h3>{editingClientId ? t('clientFormEdit') : t('clientFormCreate')}</h3>
            </div>
            {editingClientId ? (
              <button className="ghost-button icon-button" type="button" onClick={cancelEditing} title={t('cancel')}>
                <X aria-hidden="true" />
              </button>
            ) : null}
          </div>

          {errors.form ? (
            <div className="form-alert" role="alert">
              <CircleAlert aria-hidden="true" />
              {errors.form}
            </div>
          ) : null}

          <div className="client-form-grid">
            <label className={fieldClass(errors.name)} htmlFor="client-name">
              <span>
                {t('name')} <em>{t('required')}</em>
              </span>
              <input
                aria-describedby={errors.name ? 'client-name-error' : undefined}
                aria-invalid={Boolean(errors.name)}
                id="client-name"
                onChange={(event) => updateField('name', event.target.value)}
                placeholder={t('clientNamePlaceholder')}
                value={form.name}
              />
              <FieldError id="client-name-error" message={errors.name} />
            </label>

            <label className={fieldClass(errors.email)} htmlFor="client-email">
              <span>{t('email')}</span>
              <input
                aria-describedby={errors.email ? 'client-email-error' : undefined}
                aria-invalid={Boolean(errors.email)}
                id="client-email"
                onChange={(event) => updateField('email', event.target.value)}
                placeholder={t('clientEmailPlaceholder')}
                type="email"
                value={form.email}
              />
              <FieldError id="client-email-error" message={errors.email} />
            </label>

            <label className={fieldClass(errors.defaultCurrency)} htmlFor="client-currency">
              <span>
                {t('defaultCurrency')} <em>{t('required')}</em>
              </span>
              <input
                aria-describedby={errors.defaultCurrency ? 'client-currency-error' : undefined}
                aria-invalid={Boolean(errors.defaultCurrency)}
                id="client-currency"
                maxLength={3}
                onChange={(event) => updateField('defaultCurrency', event.target.value.toUpperCase())}
                placeholder={t('clientCurrencyPlaceholder')}
                value={form.defaultCurrency}
              />
              <FieldError id="client-currency-error" message={errors.defaultCurrency} />
            </label>

            <label className={fieldClass(errors.hourlyRate)} htmlFor="client-rate">
              <span>{t('hourlyRate')}</span>
              <input
                aria-describedby={errors.hourlyRate ? 'client-rate-error' : undefined}
                aria-invalid={Boolean(errors.hourlyRate)}
                id="client-rate"
                inputMode="decimal"
                min="0"
                onChange={(event) => updateField('hourlyRate', event.target.value)}
                placeholder={t('clientRatePlaceholder')}
                type="text"
                value={form.hourlyRate}
              />
              <FieldError id="client-rate-error" message={errors.hourlyRate} />
            </label>

            <label className={fieldClass(errors.taxId)} htmlFor="client-tax-id">
              <span>{t('taxId')}</span>
              <input
                id="client-tax-id"
                onChange={(event) => updateField('taxId', event.target.value)}
                placeholder={t('clientTaxPlaceholder')}
                value={form.taxId}
              />
              <FieldError id="client-tax-id-error" message={errors.taxId} />
            </label>

            <label className={fieldClass(errors.billingAddress) + ' client-address-field'} htmlFor="client-address">
              <span>{t('billingAddress')}</span>
              <input
                id="client-address"
                onChange={(event) => updateField('billingAddress', event.target.value)}
                placeholder={t('clientAddressPlaceholder')}
                value={form.billingAddress}
              />
              <FieldError id="client-address-error" message={errors.billingAddress} />
            </label>

            {editingClientId ? (
              <label className="client-active-field" htmlFor="client-active">
                <input
                  checked={form.active}
                  id="client-active"
                  onChange={(event) => updateField('active', event.target.checked)}
                  type="checkbox"
                />
                <span>{t('clientActive')}</span>
              </label>
            ) : null}
          </div>

          <div className="client-form-actions">
            <button type="submit" disabled={isSaving}>
              {editingClientId ? <Save aria-hidden="true" /> : <Plus aria-hidden="true" />}
              {editingClientId ? t('updateClient') : t('createClient')}
            </button>
            <button className="secondary-button" type="button" onClick={cancelEditing}>
              <X aria-hidden="true" />
              {t('cleanForm')}
            </button>
          </div>
        </form>
      </div>
    </section>
  );
}

type ProjectFormState = Omit<ProjectInput, 'defaultHourlyRateMinor'> & {
  hourlyRate: string;
  active: boolean;
};

type ProjectFormErrors = Partial<Record<keyof ProjectFormState | 'form', string>>;

const emptyProjectForm: ProjectFormState = {
  clientId: '',
  name: '',
  color: '#2563eb',
  hourlyRate: '',
  active: true,
};

function ProjectPanel({
  clients,
  isLoading,
  projects,
  t,
}: {
  clients: Client[];
  isLoading: boolean;
  projects: Project[];
  t: Translator;
}) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const { refreshPendingCount } = useOfflineStatus();
  const [editingProjectId, setEditingProjectId] = useState<string | null>(null);
  const [form, setForm] = useState<ProjectFormState>(emptyProjectForm);
  const [errors, setErrors] = useState<ProjectFormErrors>({});

  const createMutation = useMutation({
    mutationFn: (input: ProjectInput) => createProject(input, { clients }),
    onSuccess: (project) => {
      setForm(emptyProjectForm);
      setErrors({});
      patchProjectsCache(queryClient, project);
      void refreshPendingCount();
      void refreshOverviewIfOnline(queryClient);
      if (!isLocalId(project.id)) {
        queryClient.invalidateQueries({ queryKey: ['projects'] });
      }
      toastMutationSuccess(toast, t, 'projectCreated', project.id);
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('projectSaveFailed') }));
      toast.error(t('projectSaveFailed'));
    },
  });

  const updateMutation = useMutation({
    mutationFn: async ({
      projectId,
      input,
      active,
      wasActive,
    }: {
      projectId: string;
      input: ProjectInput;
      active: boolean;
      wasActive: boolean;
    }) => {
      const updated = await updateProject(projectId, input);
      if (active && !wasActive) {
        await restoreProject(projectId);
      } else if (!active && wasActive) {
        await archiveProject(projectId);
      }
      return updated;
    },
    onSuccess: () => {
      setEditingProjectId(null);
      setForm(emptyProjectForm);
      setErrors({});
      queryClient.invalidateQueries({ queryKey: ['projects'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
      toast.success(t('projectUpdated'));
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('projectSaveFailed') }));
      toast.error(t('projectSaveFailed'));
    },
  });

  const archiveMutation = useMutation({
    mutationFn: archiveProject,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
      toast.success(t('projectArchived'));
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('projectArchiveFailed') }));
      toast.error(t('projectArchiveFailed'));
    },
  });

  const restoreMutation = useMutation({
    mutationFn: restoreProject,
    onSuccess: () => {
      setEditingProjectId(null);
      setForm(emptyProjectForm);
      setErrors({});
      queryClient.invalidateQueries({ queryKey: ['projects'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
      toast.success(t('projectRestored'));
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('projectSaveFailed') }));
      toast.error(t('projectSaveFailed'));
    },
  });

  function submitProject(event: FormEvent) {
    event.preventDefault();
    const validation = validateProjectForm(form, t);
    setErrors(validation);
    if (hasErrors(validation)) {
      return;
    }

    const input = projectFormToInput(form);
    if (editingProjectId) {
      const project = projects.find((item) => item.id === editingProjectId);
      updateMutation.mutate({
        projectId: editingProjectId,
        input,
        active: form.active,
        wasActive: !project?.archivedAt,
      });
      return;
    }
    createMutation.mutate(input);
  }

  function updateField<K extends keyof ProjectFormState>(field: K, value: ProjectFormState[K]) {
    const next = { ...form, [field]: value };
    setForm(next);
    if (hasErrors(errors)) {
      setErrors(validateProjectForm(next, t));
    }
  }

  function startEditing(project: Project) {
    setEditingProjectId(project.id);
    setErrors({});
    setForm({
      clientId: project.clientId,
      name: project.name,
      color: project.color,
      hourlyRate:
        project.defaultHourlyRateMinor === null || project.defaultHourlyRateMinor === undefined
          ? ''
          : formatRateInput(project.defaultHourlyRateMinor),
      active: !project.archivedAt,
    });
  }

  function cancelEditing() {
    setEditingProjectId(null);
    setForm(emptyProjectForm);
    setErrors({});
  }

  const isSaving = createMutation.isPending || updateMutation.isPending;
  const activeProjectCount = projects.filter((project) => !project.archivedAt).length;
  const activeProjects = projects.filter((project) => !project.archivedAt);
  const inactiveProjects = projects.filter((project) => project.archivedAt);

  function renderProjectRow(project: Project, isActive: boolean) {
    return (
      <article
        className={editingProjectId === project.id ? 'client-row selected' : isActive ? 'client-row' : 'client-row archived'}
        key={project.id}
      >
        <div className="client-row-main">
          <div className="project-color-dot" style={{ backgroundColor: project.color }} aria-hidden="true" />
          <div className="client-row-copy">
            <div className="client-row-title">
              <strong>{project.name}</strong>
              <span className={isActive ? 'status-pill' : 'status-pill warning-pill'}>
                {isActive ? <CircleCheck aria-hidden="true" /> : null}
                {isActive ? t('active') : t('inactive')}
              </span>
            </div>
            <span className="client-contact">
              <Building2 aria-hidden="true" />
              {project.clientName || t('projectClientOptional')}
            </span>
          </div>
        </div>
        <div className="client-row-meta">
          {project.defaultHourlyRateMinor === null ? null : (
            <span className="rate-pill">
              <BadgeDollarSign aria-hidden="true" />
              {formatMinor(project.defaultHourlyRateMinor)}/h
            </span>
          )}
          <span>{project.color}</span>
        </div>
        <div className="client-row-actions">
          <button
            className="secondary-button icon-button"
            type="button"
            onClick={() => startEditing(project)}
            title={t('edit')}
          >
            <Pencil aria-hidden="true" />
          </button>
          {isActive ? (
            <button
              className="secondary-button icon-button danger-button"
              type="button"
              onClick={() => archiveMutation.mutate(project.id)}
              title={t('archive')}
            >
              <Trash2 aria-hidden="true" />
            </button>
          ) : (
            <button
              className="secondary-button icon-button"
              type="button"
              onClick={() => restoreMutation.mutate(project.id)}
              title={t('reactivate')}
            >
              <RotateCcw aria-hidden="true" />
            </button>
          )}
        </div>
      </article>
    );
  }

  return (
    <section className="clients-section projects-section" id="projects" aria-labelledby="projects-title">
      <div className="clients-heading">
        <div className="section-title-group">
          <span className="section-kicker">
            <FolderKanban aria-hidden="true" />
            {t('projects')}
          </span>
          <h2 id="projects-title">{t('projectDirectory')}</h2>
          <p>{t('projectPanelSubtitle')}</p>
        </div>
        <button className="secondary-button" type="button" onClick={cancelEditing}>
          <Plus aria-hidden="true" />
          {t('newProject')}
        </button>
      </div>

      <div className="clients-workbench">
        <div className="client-directory">
          <div className="directory-toolbar">
            <div>
              <span>{t('activeProjects')}</span>
              <strong>{activeProjectCount}</strong>
            </div>
            {isLoading ? (
              <span className="sync-pill">{t('loading')}</span>
            ) : (
              <span className="sync-pill">{t('synced')}</span>
            )}
          </div>

          <div className="client-list" aria-busy={isLoading}>
            {projects.length === 0 ? (
              <div className="empty-state">
                <FolderKanban aria-hidden="true" />
                <p>{t('noProjects')}</p>
              </div>
            ) : null}
            {activeProjects.map((project) => renderProjectRow(project, true))}
          </div>

          <DirectoryInactiveHeading count={inactiveProjects.length} t={t} />
          {inactiveProjects.length > 0 ? (
            <div className="client-list client-list-inactive" aria-label={t('inactiveDirectory')}>
              {inactiveProjects.map((project) => renderProjectRow(project, false))}
            </div>
          ) : null}
        </div>

        <form className="client-editor" noValidate onSubmit={submitProject}>
          <div className="editor-header">
            <div>
              <span>{editingProjectId ? t('editingProject') : t('newProject')}</span>
              <h3>{editingProjectId ? t('projectFormEdit') : t('projectFormCreate')}</h3>
            </div>
            {editingProjectId ? (
              <button className="ghost-button icon-button" type="button" onClick={cancelEditing} title={t('cancel')}>
                <X aria-hidden="true" />
              </button>
            ) : null}
          </div>

          {errors.form ? (
            <div className="form-alert" role="alert">
              <CircleAlert aria-hidden="true" />
              {errors.form}
            </div>
          ) : null}

          <div className="client-form-grid">
            <label className={fieldClass(errors.name)} htmlFor="project-name">
              <span>
                {t('name')} <em>{t('required')}</em>
              </span>
              <input
                aria-describedby={errors.name ? 'project-name-error' : undefined}
                aria-invalid={Boolean(errors.name)}
                id="project-name"
                onChange={(event) => updateField('name', event.target.value)}
                placeholder={t('projectNamePlaceholder')}
                value={form.name}
              />
              <FieldError id="project-name-error" message={errors.name} />
            </label>

            <label className="form-field" htmlFor="project-client">
              <span>{t('projectClient')}</span>
              <select
                id="project-client"
                onChange={(event) => updateField('clientId', event.target.value)}
                value={form.clientId}
              >
                <option value="">{t('projectClientOptional')}</option>
                {clients.filter((client) => !client.archivedAt).map((client) => (
                  <option key={client.id} value={client.id}>
                    {client.name}
                  </option>
                ))}
              </select>
            </label>

            <label className={fieldClass(errors.color)} htmlFor="project-color">
              <span>
                {t('projectColor')} <em>{t('required')}</em>
              </span>
              <div className="color-input-row">
                <input
                  aria-label={t('projectColor')}
                  onChange={(event) => updateField('color', event.target.value)}
                  type="color"
                  value={form.color}
                />
                <input
                  aria-describedby={errors.color ? 'project-color-error' : undefined}
                  aria-invalid={Boolean(errors.color)}
                  id="project-color"
                  onChange={(event) => updateField('color', event.target.value)}
                  placeholder={t('projectColorPlaceholder')}
                  value={form.color}
                />
              </div>
              <FieldError id="project-color-error" message={errors.color} />
            </label>

            <label className={fieldClass(errors.hourlyRate)} htmlFor="project-rate">
              <span>{t('hourlyRate')}</span>
              <input
                aria-describedby={errors.hourlyRate ? 'project-rate-error' : undefined}
                aria-invalid={Boolean(errors.hourlyRate)}
                id="project-rate"
                inputMode="decimal"
                min="0"
                onChange={(event) => updateField('hourlyRate', event.target.value)}
                placeholder={t('clientRatePlaceholder')}
                type="text"
                value={form.hourlyRate}
              />
              <FieldError id="project-rate-error" message={errors.hourlyRate} />
            </label>

            {editingProjectId ? (
              <label className="client-active-field" htmlFor="project-active">
                <input
                  checked={form.active}
                  id="project-active"
                  onChange={(event) => updateField('active', event.target.checked)}
                  type="checkbox"
                />
                <span>{t('projectActive')}</span>
              </label>
            ) : null}
          </div>

          <div className="client-form-actions">
            <button type="submit" disabled={isSaving}>
              {editingProjectId ? <Save aria-hidden="true" /> : <Plus aria-hidden="true" />}
              {editingProjectId ? t('updateProject') : t('createProject')}
            </button>
            <button className="secondary-button" type="button" onClick={cancelEditing}>
              <X aria-hidden="true" />
              {t('cleanForm')}
            </button>
          </div>
        </form>
      </div>
    </section>
  );
}

type TaskFormState = {
  projectId: string;
  name: string;
  billable: boolean;
  active: boolean;
};

type TaskFormErrors = Partial<Record<keyof TaskFormState | 'form', string>>;

const emptyTaskForm: TaskFormState = {
  projectId: '',
  name: '',
  billable: true,
  active: true,
};

function TaskPanel({
  isLoading,
  projects,
  tasks,
  t,
}: {
  isLoading: boolean;
  projects: Project[];
  tasks: Task[];
  t: Translator;
}) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const { refreshPendingCount } = useOfflineStatus();
  const [editingTaskId, setEditingTaskId] = useState<string | null>(null);
  const [form, setForm] = useState<TaskFormState>(emptyTaskForm);
  const [errors, setErrors] = useState<TaskFormErrors>({});
  const sortedTasks = useMemo(() => sortTasksByNewest(tasks), [tasks]);

  const createMutation = useMutation({
    mutationFn: (input: TaskInput) => createTask(input, { projects }),
    onSuccess: (created) => {
      patchTasksCache(queryClient, created);
      setForm(emptyTaskForm);
      setErrors({});
      void refreshPendingCount();
      void refreshOverviewIfOnline(queryClient);
      if (!isLocalId(created.id)) {
        queryClient.invalidateQueries({ queryKey: ['tasks'] });
      }
      toastMutationSuccess(toast, t, 'taskCreated', created.id);
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('taskSaveFailed') }));
      toast.error(t('taskSaveFailed'));
    },
  });

  const updateMutation = useMutation({
    mutationFn: async ({
      taskId,
      input,
      active,
      wasActive,
    }: {
      taskId: string;
      input: TaskInput;
      active: boolean;
      wasActive: boolean;
    }) => {
      const updated = await updateTask(taskId, input);
      if (active && !wasActive) {
        await restoreTask(taskId);
      } else if (!active && wasActive) {
        await archiveTask(taskId);
      }
      return updated;
    },
    onSuccess: () => {
      setEditingTaskId(null);
      setForm(emptyTaskForm);
      setErrors({});
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
      toast.success(t('taskUpdated'));
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('taskSaveFailed') }));
      toast.error(t('taskSaveFailed'));
    },
  });

  const inlineUpdateMutation = useMutation({
    mutationFn: ({ taskId, input }: { taskId: string; input: TaskInput }) => updateTask(taskId, input),
    onSuccess: (updated) => {
      queryClient.setQueryData(['tasks'], (current: { tasks: Task[] } | undefined) => {
        if (!current) {
          return current;
        }
        return {
          tasks: sortTasksByNewest(current.tasks.map((item) => (item.id === updated.id ? updated : item))),
        };
      });
      if (editingTaskId === updated.id) {
        setForm((current) => ({ ...current, name: updated.name }));
      }
    },
    onError: () => toast.error(t('taskSaveFailed')),
  });

  const saveInlineTaskName = useCallback(
    (taskId: string, name: string) => {
      const task = tasks.find((item) => item.id === taskId);
      if (!task) {
        return;
      }
      inlineUpdateMutation.mutate({
        taskId,
        input: {
          projectId: task.projectId,
          name,
          billable: task.billable,
        },
      });
    },
    [inlineUpdateMutation, tasks],
  );

  const archiveMutation = useMutation({
    mutationFn: archiveTask,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
      toast.success(t('taskArchived'));
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('taskArchiveFailed') }));
      toast.error(t('taskArchiveFailed'));
    },
  });

  const restoreMutation = useMutation({
    mutationFn: restoreTask,
    onSuccess: () => {
      setEditingTaskId(null);
      setForm(emptyTaskForm);
      setErrors({});
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
      toast.success(t('taskRestored'));
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('taskSaveFailed') }));
      toast.error(t('taskSaveFailed'));
    },
  });

  function submitTask(event: FormEvent) {
    event.preventDefault();
    const validation = validateTaskForm(form, t);
    setErrors(validation);
    if (hasErrors(validation)) {
      return;
    }

    const input = taskFormToInput(form);
    if (editingTaskId) {
      const task = tasks.find((item) => item.id === editingTaskId);
      updateMutation.mutate({
        taskId: editingTaskId,
        input,
        active: form.active,
        wasActive: !task?.archivedAt,
      });
      return;
    }
    createMutation.mutate(input);
  }

  function updateField<K extends keyof TaskFormState>(field: K, value: TaskFormState[K]) {
    const next = { ...form, [field]: value };
    setForm(next);
    if (hasErrors(errors)) {
      setErrors(validateTaskForm(next, t));
    }
  }

  function startEditing(task: Task) {
    setEditingTaskId(task.id);
    setErrors({});
    setForm({
      projectId: task.projectId,
      name: task.name,
      billable: task.billable,
      active: !task.archivedAt,
    });
  }

  function cancelEditing() {
    setEditingTaskId(null);
    setForm(emptyTaskForm);
    setErrors({});
  }

  const isSaving = createMutation.isPending || updateMutation.isPending;
  const activeTaskCount = sortedTasks.filter((task) => !task.archivedAt).length;
  const activeTasks = sortedTasks.filter((task) => !task.archivedAt);
  const inactiveTasks = sortedTasks.filter((task) => task.archivedAt);

  function renderTaskRow(task: Task, isActive: boolean) {
    return (
      <article
        className={editingTaskId === task.id ? 'client-row selected' : isActive ? 'client-row' : 'client-row archived'}
        key={task.id}
      >
        <div className="client-row-main">
          <div className="client-row-copy">
            <div className="client-row-title">
              <TaskInlineNameInput onSave={saveInlineTaskName} t={t} task={task} />
              <span className={isActive ? 'status-pill' : 'status-pill warning-pill'}>
                {isActive ? <CircleCheck aria-hidden="true" /> : null}
                {isActive ? t('active') : t('inactive')}
              </span>
            </div>
            <ProjectBadge color={task.projectColor} emptyLabel={t('taskProjectOptional')} name={task.projectName} />
          </div>
        </div>
        <div className="client-row-meta">
          <span className={task.billable ? 'rate-pill billable-on' : 'rate-pill'}>
            <DollarSign aria-hidden="true" />
            {task.billable ? t('billable') : t('nonBillable')}
          </span>
        </div>
        <div className="client-row-actions">
          <button
            className="secondary-button icon-button"
            type="button"
            onClick={() => startEditing(task)}
            title={t('edit')}
          >
            <Pencil aria-hidden="true" />
          </button>
          {isActive ? (
            <button
              className="secondary-button icon-button danger-button"
              type="button"
              onClick={() => archiveMutation.mutate(task.id)}
              title={t('archive')}
            >
              <Trash2 aria-hidden="true" />
            </button>
          ) : (
            <button
              className="secondary-button icon-button"
              type="button"
              onClick={() => restoreMutation.mutate(task.id)}
              title={t('reactivate')}
            >
              <RotateCcw aria-hidden="true" />
            </button>
          )}
        </div>
      </article>
    );
  }

  return (
    <section className="clients-section tasks-section" id="tasks" aria-labelledby="tasks-title">
      <div className="clients-heading">
        <div className="section-title-group">
          <span className="section-kicker">
            <ListTodo aria-hidden="true" />
            {t('tasks')}
          </span>
          <h2 id="tasks-title">{t('taskDirectory')}</h2>
          <p>{t('taskPanelSubtitle')}</p>
        </div>
        <button className="secondary-button" type="button" onClick={cancelEditing}>
          <Plus aria-hidden="true" />
          {t('newTask')}
        </button>
      </div>

      <div className="clients-workbench">
        <div className="client-directory">
          <div className="directory-toolbar">
            <div>
              <span>{t('activeTasks')}</span>
              <strong>{activeTaskCount}</strong>
            </div>
            {isLoading ? (
              <span className="sync-pill">{t('loading')}</span>
            ) : (
              <span className="sync-pill">{t('synced')}</span>
            )}
          </div>

          <div className="client-list" aria-busy={isLoading}>
            {sortedTasks.length === 0 ? (
              <div className="empty-state">
                <ListTodo aria-hidden="true" />
                <p>{t('noTasks')}</p>
              </div>
            ) : null}
            {activeTasks.map((task) => renderTaskRow(task, true))}
          </div>

          <DirectoryInactiveHeading count={inactiveTasks.length} t={t} />
          {inactiveTasks.length > 0 ? (
            <div className="client-list client-list-inactive" aria-label={t('inactiveDirectory')}>
              {inactiveTasks.map((task) => renderTaskRow(task, false))}
            </div>
          ) : null}
        </div>

        <form className="client-editor" noValidate onSubmit={submitTask}>
          <div className="editor-header">
            <div>
              <span>{editingTaskId ? t('editingTask') : t('newTask')}</span>
              <h3>{editingTaskId ? t('taskFormEdit') : t('taskFormCreate')}</h3>
            </div>
            {editingTaskId ? (
              <button className="ghost-button icon-button" type="button" onClick={cancelEditing} title={t('cancel')}>
                <X aria-hidden="true" />
              </button>
            ) : null}
          </div>

          {errors.form ? (
            <div className="form-alert" role="alert">
              <CircleAlert aria-hidden="true" />
              {errors.form}
            </div>
          ) : null}

          <div className="client-form-grid">
            <label className={fieldClass(errors.name)} htmlFor="task-name">
              <span>
                {t('name')} <em>{t('required')}</em>
              </span>
              <input
                aria-describedby={errors.name ? 'task-name-error' : undefined}
                aria-invalid={Boolean(errors.name)}
                id="task-name"
                onChange={(event) => updateField('name', event.target.value)}
                placeholder={t('taskNamePlaceholder')}
                value={form.name}
              />
              <FieldError id="task-name-error" message={errors.name} />
            </label>

            <label className="form-field" htmlFor="task-project">
              <span>{t('taskProject')}</span>
              <select
                id="task-project"
                onChange={(event) => updateField('projectId', event.target.value)}
                value={form.projectId}
              >
                <option value="">{t('taskProjectOptional')}</option>
                {projects.filter((project) => !project.archivedAt).map((project) => (
                  <option key={project.id} value={project.id}>
                    {project.name}
                  </option>
                ))}
              </select>
            </label>

            <label className="form-field checkbox-field" htmlFor="task-billable">
              <span>{t('billable')}</span>
              <input
                checked={form.billable}
                id="task-billable"
                onChange={(event) => updateField('billable', event.target.checked)}
                type="checkbox"
              />
            </label>

            {editingTaskId ? (
              <label className="client-active-field" htmlFor="task-active">
                <input
                  checked={form.active}
                  id="task-active"
                  onChange={(event) => updateField('active', event.target.checked)}
                  type="checkbox"
                />
                <span>{t('taskActive')}</span>
              </label>
            ) : null}
          </div>

          <div className="client-form-actions">
            <button type="submit" disabled={isSaving}>
              {editingTaskId ? <Save aria-hidden="true" /> : <Plus aria-hidden="true" />}
              {editingTaskId ? t('updateTask') : t('createTask')}
            </button>
            <button className="secondary-button" type="button" onClick={cancelEditing}>
              <X aria-hidden="true" />
              {t('cleanForm')}
            </button>
          </div>
        </form>
      </div>
    </section>
  );
}

type TagFormState = {
  name: string;
  color: string;
  active: boolean;
};

type TagFormErrors = Partial<Record<keyof TagFormState | 'form', string>>;

const emptyTagForm: TagFormState = {
  name: '',
  color: '#64748b',
  active: true,
};

function TagPanel({ isLoading, tags, t }: { isLoading: boolean; tags: TagRecord[]; t: Translator }) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const { refreshPendingCount } = useOfflineStatus();
  const [editingTagId, setEditingTagId] = useState<string | null>(null);
  const [form, setForm] = useState<TagFormState>(emptyTagForm);
  const [errors, setErrors] = useState<TagFormErrors>({});

  const createMutation = useMutation({
    mutationFn: createTag,
    onSuccess: (tag) => {
      setForm(emptyTagForm);
      setErrors({});
      patchTagsCache(queryClient, tag);
      void refreshPendingCount();
      void refreshOverviewIfOnline(queryClient);
      if (!isLocalId(tag.id)) {
        queryClient.invalidateQueries({ queryKey: ['tags'] });
      }
      toastMutationSuccess(toast, t, 'tagCreated', tag.id);
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('tagSaveFailed') }));
      toast.error(t('tagSaveFailed'));
    },
  });

  const updateMutation = useMutation({
    mutationFn: async ({
      tagId,
      input,
      active,
      wasActive,
    }: {
      tagId: string;
      input: TagInput;
      active: boolean;
      wasActive: boolean;
    }) => {
      const updated = await updateTag(tagId, input);
      if (active && !wasActive) {
        await restoreTag(tagId);
      } else if (!active && wasActive) {
        await archiveTag(tagId);
      }
      return updated;
    },
    onSuccess: () => {
      setEditingTagId(null);
      setForm(emptyTagForm);
      setErrors({});
      queryClient.invalidateQueries({ queryKey: ['tags'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
      toast.success(t('tagUpdated'));
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('tagSaveFailed') }));
      toast.error(t('tagSaveFailed'));
    },
  });

  const archiveMutation = useMutation({
    mutationFn: archiveTag,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tags'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
      toast.success(t('tagArchived'));
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('tagArchiveFailed') }));
      toast.error(t('tagArchiveFailed'));
    },
  });

  const restoreMutation = useMutation({
    mutationFn: restoreTag,
    onSuccess: () => {
      setEditingTagId(null);
      setForm(emptyTagForm);
      setErrors({});
      queryClient.invalidateQueries({ queryKey: ['tags'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
      toast.success(t('tagRestored'));
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('tagSaveFailed') }));
      toast.error(t('tagSaveFailed'));
    },
  });

  function submitTag(event: FormEvent) {
    event.preventDefault();
    const validation = validateTagForm(form, t);
    setErrors(validation);
    if (hasErrors(validation)) {
      return;
    }

    const input = tagFormToInput(form);
    if (editingTagId) {
      const tag = tags.find((item) => item.id === editingTagId);
      updateMutation.mutate({
        tagId: editingTagId,
        input,
        active: form.active,
        wasActive: !tag?.archivedAt,
      });
      return;
    }
    createMutation.mutate(input);
  }

  function updateField<K extends keyof TagFormState>(field: K, value: TagFormState[K]) {
    const next = { ...form, [field]: value };
    setForm(next);
    if (hasErrors(errors)) {
      setErrors(validateTagForm(next, t));
    }
  }

  function startEditing(tag: TagRecord) {
    setEditingTagId(tag.id);
    setErrors({});
    setForm({
      name: tag.name,
      color: tag.color,
      active: !tag.archivedAt,
    });
  }

  function cancelEditing() {
    setEditingTagId(null);
    setForm(emptyTagForm);
    setErrors({});
  }

  const isSaving = createMutation.isPending || updateMutation.isPending;
  const activeTagCount = tags.filter((tag) => !tag.archivedAt).length;
  const activeTags = tags.filter((tag) => !tag.archivedAt);
  const inactiveTags = tags.filter((tag) => tag.archivedAt);

  function renderTagRow(tag: TagRecord, isActive: boolean) {
    return (
      <article
        className={editingTagId === tag.id ? 'client-row selected' : isActive ? 'client-row' : 'client-row archived'}
        key={tag.id}
      >
        <div className="client-row-main">
          <div className="project-color-dot" style={{ backgroundColor: tag.color }} aria-hidden="true" />
          <div className="client-row-copy">
            <div className="client-row-title">
              <strong>{tag.name}</strong>
              <span className={isActive ? 'status-pill' : 'status-pill warning-pill'}>
                {isActive ? <CircleCheck aria-hidden="true" /> : null}
                {isActive ? t('active') : t('inactive')}
              </span>
            </div>
            <span className="client-contact">
              <Tag aria-hidden="true" />
              {tag.color}
            </span>
          </div>
        </div>
        <div className="client-row-actions">
          <button
            className="secondary-button icon-button"
            type="button"
            onClick={() => startEditing(tag)}
            title={t('edit')}
          >
            <Pencil aria-hidden="true" />
          </button>
          {isActive ? (
            <button
              className="secondary-button icon-button danger-button"
              type="button"
              onClick={() => archiveMutation.mutate(tag.id)}
              title={t('archive')}
            >
              <Trash2 aria-hidden="true" />
            </button>
          ) : (
            <button
              className="secondary-button icon-button"
              type="button"
              onClick={() => restoreMutation.mutate(tag.id)}
              title={t('reactivate')}
            >
              <RotateCcw aria-hidden="true" />
            </button>
          )}
        </div>
      </article>
    );
  }

  return (
    <section className="clients-section tags-section" id="tags" aria-labelledby="tags-title">
      <div className="clients-heading">
        <div className="section-title-group">
          <span className="section-kicker">
            <Tags aria-hidden="true" />
            {t('tags')}
          </span>
          <h2 id="tags-title">{t('tagDirectory')}</h2>
          <p>{t('tagPanelSubtitle')}</p>
        </div>
        <button className="secondary-button" type="button" onClick={cancelEditing}>
          <Plus aria-hidden="true" />
          {t('newTag')}
        </button>
      </div>

      <div className="clients-workbench">
        <div className="client-directory">
          <div className="directory-toolbar">
            <div>
              <span>{t('activeTags')}</span>
              <strong>{activeTagCount}</strong>
            </div>
            {isLoading ? (
              <span className="sync-pill">{t('loading')}</span>
            ) : (
              <span className="sync-pill">{t('synced')}</span>
            )}
          </div>

          <div className="client-list" aria-busy={isLoading}>
            {tags.length === 0 ? (
              <div className="empty-state">
                <Tags aria-hidden="true" />
                <p>{t('noTags')}</p>
              </div>
            ) : null}
            {activeTags.map((tag) => renderTagRow(tag, true))}
          </div>

          <DirectoryInactiveHeading count={inactiveTags.length} t={t} />
          {inactiveTags.length > 0 ? (
            <div className="client-list client-list-inactive" aria-label={t('inactiveDirectory')}>
              {inactiveTags.map((tag) => renderTagRow(tag, false))}
            </div>
          ) : null}
        </div>

        <form className="client-editor" noValidate onSubmit={submitTag}>
          <div className="editor-header">
            <div>
              <span>{editingTagId ? t('editingTag') : t('createTag')}</span>
              <h3>{editingTagId ? t('tagFormEdit') : t('tagFormCreate')}</h3>
            </div>
            {editingTagId ? (
              <button className="ghost-button icon-button" type="button" onClick={cancelEditing} title={t('cancel')}>
                <X aria-hidden="true" />
              </button>
            ) : null}
          </div>

          {errors.form ? (
            <div className="form-alert" role="alert">
              <CircleAlert aria-hidden="true" />
              {errors.form}
            </div>
          ) : null}

          <div className="client-form-grid">
            <label className={fieldClass(errors.name)} htmlFor="tag-name">
              <span>
                {t('name')} <em>{t('required')}</em>
              </span>
              <input
                aria-describedby={errors.name ? 'tag-name-error' : undefined}
                aria-invalid={Boolean(errors.name)}
                id="tag-name"
                onChange={(event) => updateField('name', event.target.value)}
                placeholder={t('tagNamePlaceholder')}
                value={form.name}
              />
              <FieldError id="tag-name-error" message={errors.name} />
            </label>

            <label className={fieldClass(errors.color)} htmlFor="tag-color">
              <span>
                {t('tagColor')} <em>{t('required')}</em>
              </span>
              <div className="color-input-row">
                <input
                  aria-label={t('tagColor')}
                  onChange={(event) => updateField('color', event.target.value)}
                  type="color"
                  value={form.color}
                />
                <input
                  aria-describedby={errors.color ? 'tag-color-error' : undefined}
                  aria-invalid={Boolean(errors.color)}
                  id="tag-color"
                  onChange={(event) => updateField('color', event.target.value)}
                  placeholder={t('tagColorPlaceholder')}
                  value={form.color}
                />
              </div>
              <FieldError id="tag-color-error" message={errors.color} />
            </label>

            {editingTagId ? (
              <label className="client-active-field" htmlFor="tag-active">
                <input
                  checked={form.active}
                  id="tag-active"
                  onChange={(event) => updateField('active', event.target.checked)}
                  type="checkbox"
                />
                <span>{t('tagActive')}</span>
              </label>
            ) : null}
          </div>

          <div className="client-form-actions">
            <button type="submit" disabled={isSaving}>
              {editingTagId ? <Save aria-hidden="true" /> : <Plus aria-hidden="true" />}
              {editingTagId ? t('updateTag') : t('createTag')}
            </button>
            <button className="secondary-button" type="button" onClick={cancelEditing}>
              <X aria-hidden="true" />
              {t('cleanForm')}
            </button>
          </div>
        </form>
      </div>
    </section>
  );
}

function TaskInlineNameInput({
  onSave,
  t,
  task,
}: {
  onSave: (taskId: string, name: string) => void;
  t: Translator;
  task: Task;
}) {
  const [liveName, setLiveName] = useState(task.name);
  const [inlineError, setInlineError] = useState('');
  const skipSaveRef = useRef(true);
  const taskRef = useRef(task);
  taskRef.current = task;

  useEffect(() => {
    skipSaveRef.current = true;
    setLiveName(task.name);
    setInlineError('');
  }, [task.id]);

  useEffect(() => {
    const trimmed = liveName.trim();
    if (skipSaveRef.current) {
      skipSaveRef.current = false;
      return;
    }
    if (trimmed === taskRef.current.name) {
      setInlineError('');
      return;
    }
    if (!trimmed) {
      setInlineError(t('taskNameRequired'));
      return;
    }
    if (trimmed.length < 2) {
      setInlineError(t('taskNameTooShort'));
      return;
    }

    setInlineError('');
    const handle = window.setTimeout(() => {
      onSave(taskRef.current.id, trimmed);
    }, 400);

    return () => window.clearTimeout(handle);
  }, [liveName, onSave, t]);

  function handleBlur() {
    const trimmed = liveName.trim();
    if (!trimmed || trimmed.length < 2) {
      setLiveName(taskRef.current.name);
      setInlineError('');
    }
  }

  return (
    <label className="client-row-inline-field">
      <span className="visually-hidden">{t('taskName')}</span>
      <input
        aria-describedby={inlineError ? `task-inline-error-${task.id}` : undefined}
        aria-invalid={Boolean(inlineError)}
        aria-label={`${t('taskName')}: ${task.name}`}
        className={inlineError ? 'client-row-inline-input invalid' : 'client-row-inline-input'}
        onBlur={handleBlur}
        onChange={(event) => setLiveName(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === 'Enter') {
            event.currentTarget.blur();
          }
        }}
        value={liveName}
      />
      {inlineError ? (
        <span className="client-row-inline-error" id={`task-inline-error-${task.id}`} role="alert">
          {inlineError}
        </span>
      ) : null}
    </label>
  );
}

function fieldClass(error?: string) {
  return error ? 'form-field has-error' : 'form-field';
}

function FieldError({ id, message }: { id: string; message?: string }) {
  if (!message) {
    return null;
  }
  return (
    <span className="field-message" id={id}>
      {message}
    </span>
  );
}

function DirectoryInactiveHeading({ count, t }: { count: number; t: Translator }) {
  if (count === 0) {
    return null;
  }

  return (
    <div className="directory-inactive-heading">
      <span>{t('inactiveDirectory')}</span>
      <strong>{count}</strong>
    </div>
  );
}

function validateClientForm(form: ClientFormState, t: Translator): ClientFormErrors {
  const errors: ClientFormErrors = {};
  const name = form.name.trim();
  const email = form.email.trim();
  const currency = form.defaultCurrency.trim().toUpperCase();
  const rate = form.hourlyRate.trim().replace(',', '.');

  if (!name) {
    errors.name = t('clientNameRequired');
  } else if (name.length < 2) {
    errors.name = t('clientNameTooShort');
  }

  if (email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
    errors.email = t('clientEmailInvalid');
  }

  if (!/^[A-Z]{3}$/.test(currency)) {
    errors.defaultCurrency = t('clientCurrencyInvalid');
  }

  if (rate && (!/^\d+(\.\d{1,2})?$/.test(rate) || Number(rate) < 0)) {
    errors.hourlyRate = t('clientRateInvalid');
  }

  return errors;
}

function validateProjectForm(form: ProjectFormState, t: Translator): ProjectFormErrors {
  const errors: ProjectFormErrors = {};
  const name = form.name.trim();
  const color = form.color.trim();
  const rate = form.hourlyRate.trim().replace(',', '.');

  if (!name) {
    errors.name = t('projectNameRequired');
  } else if (name.length < 2) {
    errors.name = t('projectNameTooShort');
  }

  if (!/^#[0-9a-fA-F]{6}$/.test(color)) {
    errors.color = t('projectColorInvalid');
  }

  if (rate && (!/^\d+(\.\d{1,2})?$/.test(rate) || Number(rate) < 0)) {
    errors.hourlyRate = t('projectRateInvalid');
  }

  return errors;
}

function validateTaskForm(form: TaskFormState, t: Translator): TaskFormErrors {
  const errors: TaskFormErrors = {};
  const name = form.name.trim();

  if (!name) {
    errors.name = t('taskNameRequired');
  } else if (name.length < 2) {
    errors.name = t('taskNameTooShort');
  }

  return errors;
}

function validateTagForm(form: TagFormState, t: Translator): TagFormErrors {
  const errors: TagFormErrors = {};
  const name = form.name.trim();
  const color = form.color.trim();

  if (!name) {
    errors.name = t('tagNameRequired');
  } else if (name.length < 2) {
    errors.name = t('tagNameTooShort');
  }

  if (!/^#[0-9a-fA-F]{6}$/.test(color)) {
    errors.color = t('tagColorInvalid');
  }

  return errors;
}

function hasErrors(errors: Record<string, string | undefined>) {
  return Object.values(errors).some(Boolean);
}

function clientFormToInput(form: ClientFormState): ClientInput {
  return {
    name: form.name.trim(),
    email: form.email.trim(),
    taxId: form.taxId.trim(),
    billingAddress: form.billingAddress.trim(),
    defaultCurrency: form.defaultCurrency.trim().toUpperCase() || 'EUR',
    defaultHourlyRateMinor: rateToMinor(form.hourlyRate),
  };
}

function projectFormToInput(form: ProjectFormState): ProjectInput {
  return {
    clientId: form.clientId,
    name: form.name.trim(),
    color: form.color.trim() || '#2563eb',
    defaultHourlyRateMinor: form.hourlyRate.trim() ? rateToMinor(form.hourlyRate) : null,
  };
}

function taskFormToInput(form: TaskFormState): TaskInput {
  return {
    projectId: form.projectId,
    name: form.name.trim(),
    billable: form.billable,
  };
}

function tagFormToInput(form: TagFormState): TagInput {
  return {
    name: form.name.trim(),
    color: form.color.trim() || '#64748b',
  };
}

function rateToMinor(value: string) {
  const normalized = value.trim().replace(',', '.');
  if (!normalized) {
    return 0;
  }
  return Math.round(Number(normalized) * 100);
}

function formatRateInput(value: number) {
  if (value === 0) {
    return '';
  }
  return (value / 100).toFixed(2);
}

function formatMinor(value: number) {
  return (value / 100).toFixed(2);
}

function initials(value: string) {
  const parts = value.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) {
    return 'LT';
  }
  return parts
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase())
    .join('');
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

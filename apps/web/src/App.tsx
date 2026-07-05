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
  Save,
  Settings,
  Tag,
  Tags,
  Trash2,
  Users,
  X,
} from 'lucide-react';
import { FormEvent, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  archiveClient,
  archiveProject,
  archiveTask,
  createClient,
  createProject,
  createTag,
  createTask,
  deleteTag,
  fetchClients,
  fetchProjects,
  fetchSession,
  fetchTags,
  fetchTasks,
  fetchTimeEntries,
  fetchTimers,
  login,
  logout,
  stopTimer,
  updateClient,
  updateProject,
  updateTag,
  updateTask,
  type Client,
  type ClientInput,
  type LayoutMode,
  type Locale,
  type Project,
  type ProjectInput,
  type Tag as TagRecord,
  type TagInput,
  type Task,
  type TaskInput,
} from './lib/api';
import { translate } from './lib/i18n';
import { sortTasksByNewest } from './lib/taskSort';
import { CalendarPanel } from './lib/calendarUi';
import { InvoicePanel } from './lib/invoiceUi';
import { addMonths, endOfMonth, startOfMonth, toMonthQueryFrom, toMonthQueryTo } from './lib/calendarMonth';
import { TimeReportPanel } from './lib/reportUi';
import { ManualTimeEntryPanel, TimeEntriesList } from './lib/timeEntryUi';
import { SidebarTimer, TimerCommandRow } from './lib/timerUi';
import { addWeeks, startOfWeek, toWeekQueryFrom, toWeekQueryTo } from './lib/timesheetWeek';
import { usePersistentState } from './lib/persistentState';

export function App() {
  const [locale, setLocale] = usePersistentState<Locale>('leotime.locale', 'es');
  const [layoutMode, setLayoutMode] = usePersistentState<LayoutMode>('leotime.layout', 'solid');
  const sessionQuery = useQuery({ queryKey: ['session'], queryFn: fetchSession });

  const t = useMemo(() => (key: Parameters<typeof translate>[1]) => translate(locale, key), [locale]);

  if (sessionQuery.isLoading) {
    return (
      <main className="boot-screen">
        <Clock3 aria-hidden="true" />
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
      t={t}
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
        <div className="brand-row">
          <Clock3 aria-hidden="true" />
          <span>{t('appName')}</span>
        </div>
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
  t: Translator;
  userName: string;
};

type TimeView = 'timesheet' | 'calendar';

function Dashboard({ layoutMode, locale, setLayoutMode, setLocale, t, userName }: DashboardProps) {
  const queryClient = useQueryClient();
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

  const clientsQuery = useQuery({ queryKey: ['clients'], queryFn: fetchClients });
  const projectsQuery = useQuery({ queryKey: ['projects'], queryFn: fetchProjects });
  const tasksQuery = useQuery({ queryKey: ['tasks'], queryFn: fetchTasks });
  const tagsQuery = useQuery({ queryKey: ['tags'], queryFn: fetchTags });
  const timeEntriesQuery = useQuery({
    queryKey: ['time-entries', timeView, timeView === 'timesheet' ? weekQueryKey : monthQueryKey],
    queryFn: () =>
      timeView === 'timesheet'
        ? fetchTimeEntries({
            from: toWeekQueryFrom(weekStart),
            to: toWeekQueryTo(weekEnd),
          })
        : fetchTimeEntries({
            from: toMonthQueryFrom(monthStart),
            to: toMonthQueryTo(monthEnd),
          }),
  });
  const timersQuery = useQuery({
    queryKey: ['timers'],
    queryFn: fetchTimers,
    refetchInterval: (query) => ((query.state.data?.timers?.length ?? 0) > 0 ? 30_000 : false),
  });
  const openTimers = timersQuery.data?.timers ?? [];
  const activeTimer = openTimers[0] ?? null;
  const stopTimerMutation = useMutation({
    mutationFn: stopTimer,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['timers'] });
      queryClient.invalidateQueries({ queryKey: ['time-entries'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
    },
  });
  const logoutMutation = useMutation({
    mutationFn: logout,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['session'] }),
  });

  return (
    <div className={`app-shell layout-${layoutMode}`}>
      <aside className="sidebar" aria-label="Primary">
        <div className="org-switcher">
          <div className="org-avatar" aria-hidden="true">
            L
          </div>
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
          <a href="#dashboard">
            <LayoutDashboard aria-hidden="true" />
            {t('dashboard')}
          </a>
          <a className={timeView === 'timesheet' ? 'active' : ''} href="#timesheet" onClick={() => setTimeView('timesheet')}>
            <Clock3 aria-hidden="true" />
            {t('time')}
          </a>
          <a className={timeView === 'calendar' ? 'active' : ''} href="#calendar" onClick={() => setTimeView('calendar')}>
            <CalendarDays aria-hidden="true" />
            {t('calendar')}
          </a>
          <a className="nav-parent" href="#reports">
            <BarChart3 aria-hidden="true" />
            {t('reporting')}
            <ChevronDown aria-hidden="true" />
          </a>
          <div className="nav-children" aria-label={t('reporting')}>
            <a href="#overview">{t('overview')}</a>
            <a href="#detailed">{t('detailed')}</a>
            <a href="#shared">{t('shared')}</a>
          </div>

          <span className="nav-section-label">{t('manage')}</span>
          <a href="#projects">
            <FolderKanban aria-hidden="true" />
            {t('projects')}
          </a>
          <a href="#tasks">
            <ListTodo aria-hidden="true" />
            {t('tasks')}
          </a>
          <a href="#clients">
            <Building2 aria-hidden="true" />
            {t('clients')}
          </a>
          <a href="#members">
            <Users aria-hidden="true" />
            {t('members')}
          </a>
          <a href="#tags">
            <Tags aria-hidden="true" />
            {t('tags')}
          </a>

          <span className="nav-section-label">{t('admin')}</span>
          <a href="#import-export">
            <Import aria-hidden="true" />
            {t('importExport')}
          </a>
          <a href="#invoices">
            <FileText aria-hidden="true" />
            {t('invoices')}
          </a>
          <a href="#settings">
            <Settings aria-hidden="true" />
            {t('settings')}
          </a>
        </nav>

        <div className="sidebar-footer">
          <button type="button" title={t('language')} onClick={() => setLocale(locale === 'es' ? 'en' : 'es')}>
            <Languages aria-hidden="true" />
          </button>
          <a href="#profile">
            <Settings aria-hidden="true" />
            {t('profileSettings')}
          </a>
          <div className="profile-avatar" aria-hidden="true">
            {initials(userName)}
          </div>
        </div>
      </aside>

      <main className="workspace">
        <header className="tracker-topbar" id="dashboard">
          <div className="tracker-title">
            <Clock3 aria-hidden="true" />
            <h1>{t('timeTracker')}</h1>
          </div>
          <div className="toolbar">
            <LayoutSwitcher layoutMode={layoutMode} setLayoutMode={setLayoutMode} t={t} />
            <button type="button" title={t('logout')} onClick={() => logoutMutation.mutate()}>
              <LogOut aria-hidden="true" />
            </button>
          </div>
        </header>

        <TimerCommandRow
          onStop={(timeEntryId) => stopTimerMutation.mutate(timeEntryId)}
          projects={projectsQuery.data?.projects ?? []}
          stoppingTimerId={stopTimerMutation.isPending ? (stopTimerMutation.variables ?? null) : null}
          tasks={tasksQuery.data?.tasks ?? []}
          timers={openTimers}
          t={t}
        />

        <TimeViewSwitcher setTimeView={setTimeView} t={t} timeView={timeView} />

        {timeView === 'timesheet' ? (
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
        ) : (
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
        )}

        <section className="management-surface" aria-label={t('manage')}>
          <TimeReportPanel locale={locale} t={t} />
          <InvoicePanel clients={clientsQuery.data?.clients ?? []} locale={locale} t={t} userName={userName} />
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
          <ClientPanel clients={clientsQuery.data?.clients ?? []} isLoading={clientsQuery.isLoading} t={t} />
          <ProjectPanel
            clients={clientsQuery.data?.clients ?? []}
            isLoading={projectsQuery.isLoading}
            projects={projectsQuery.data?.projects ?? []}
            t={t}
          />
          <TaskPanel
            isLoading={tasksQuery.isLoading}
            projects={projectsQuery.data?.projects ?? []}
            tasks={tasksQuery.data?.tasks ?? []}
            t={t}
          />
          <TagPanel isLoading={tagsQuery.isLoading} tags={tagsQuery.data?.tags ?? []} t={t} />
        </section>
      </main>
    </div>
  );
}

function TimeViewSwitcher({
  timeView,
  setTimeView,
  t,
}: {
  timeView: TimeView;
  setTimeView: (view: TimeView) => void;
  t: Translator;
}) {
  return (
    <div className="time-view-switcher" role="tablist" aria-label={t('time')}>
      <div className="segmented-control">
        <button
          aria-selected={timeView === 'timesheet'}
          className={timeView === 'timesheet' ? 'selected' : undefined}
          onClick={() => setTimeView('timesheet')}
          role="tab"
          type="button"
        >
          {t('timesheet')}
        </button>
        <button
          aria-selected={timeView === 'calendar'}
          className={timeView === 'calendar' ? 'selected' : undefined}
          onClick={() => setTimeView('calendar')}
          role="tab"
          type="button"
        >
          {t('calendar')}
        </button>
      </div>
    </div>
  );
}

function EntityPill({ color, label }: { color: string; label: string }) {
  return (
    <span className="entity-pill">
      <span style={{ backgroundColor: color }} aria-hidden="true" />
      {label}
    </span>
  );
}

type ClientFormState = Omit<ClientInput, 'defaultHourlyRateMinor'> & {
  hourlyRate: string;
};

type ClientFormErrors = Partial<Record<keyof ClientFormState | 'form', string>>;

const emptyClientForm: ClientFormState = {
  name: '',
  email: '',
  taxId: '',
  billingAddress: '',
  defaultCurrency: 'EUR',
  hourlyRate: '',
};

function ClientPanel({ clients, isLoading, t }: { clients: Client[]; isLoading: boolean; t: Translator }) {
  const queryClient = useQueryClient();
  const [editingClientId, setEditingClientId] = useState<string | null>(null);
  const [form, setForm] = useState<ClientFormState>(emptyClientForm);
  const [errors, setErrors] = useState<ClientFormErrors>({});

  const createMutation = useMutation({
    mutationFn: createClient,
    onSuccess: () => {
      setForm(emptyClientForm);
      setErrors({});
      queryClient.invalidateQueries({ queryKey: ['clients'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
    },
    onError: () => setErrors((current) => ({ ...current, form: t('clientSaveFailed') })),
  });

  const updateMutation = useMutation({
    mutationFn: ({ clientId, input }: { clientId: string; input: ClientInput }) => updateClient(clientId, input),
    onSuccess: () => {
      setEditingClientId(null);
      setForm(emptyClientForm);
      setErrors({});
      queryClient.invalidateQueries({ queryKey: ['clients'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
    },
    onError: () => setErrors((current) => ({ ...current, form: t('clientSaveFailed') })),
  });

  const archiveMutation = useMutation({
    mutationFn: archiveClient,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['clients'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
    },
    onError: () => setErrors((current) => ({ ...current, form: t('clientArchiveFailed') })),
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
      updateMutation.mutate({ clientId: editingClientId, input });
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
    });
  }

  function cancelEditing() {
    setEditingClientId(null);
    setForm(emptyClientForm);
    setErrors({});
  }

  const isSaving = createMutation.isPending || updateMutation.isPending;

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
              <strong>{clients.length}</strong>
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
            {clients.map((client) => (
              <article className={editingClientId === client.id ? 'client-row selected' : 'client-row'} key={client.id}>
                <div className="client-row-main">
                  <div className="client-avatar" aria-hidden="true">
                    {client.name.slice(0, 1).toUpperCase()}
                  </div>
                  <div className="client-row-copy">
                    <div className="client-row-title">
                      <strong>{client.name}</strong>
                      <span className="status-pill">
                        <CircleCheck aria-hidden="true" />
                        {t('active')}
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
                  <button
                    className="secondary-button icon-button danger-button"
                    type="button"
                    onClick={() => archiveMutation.mutate(client.id)}
                    title={t('archive')}
                  >
                    <Trash2 aria-hidden="true" />
                  </button>
                </div>
              </article>
            ))}
          </div>
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
};

type ProjectFormErrors = Partial<Record<keyof ProjectFormState | 'form', string>>;

const emptyProjectForm: ProjectFormState = {
  clientId: '',
  name: '',
  color: '#2563eb',
  hourlyRate: '',
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
  const [editingProjectId, setEditingProjectId] = useState<string | null>(null);
  const [form, setForm] = useState<ProjectFormState>(emptyProjectForm);
  const [errors, setErrors] = useState<ProjectFormErrors>({});

  const createMutation = useMutation({
    mutationFn: createProject,
    onSuccess: () => {
      setForm(emptyProjectForm);
      setErrors({});
      queryClient.invalidateQueries({ queryKey: ['projects'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
    },
    onError: () => setErrors((current) => ({ ...current, form: t('projectSaveFailed') })),
  });

  const updateMutation = useMutation({
    mutationFn: ({ projectId, input }: { projectId: string; input: ProjectInput }) => updateProject(projectId, input),
    onSuccess: () => {
      setEditingProjectId(null);
      setForm(emptyProjectForm);
      setErrors({});
      queryClient.invalidateQueries({ queryKey: ['projects'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
    },
    onError: () => setErrors((current) => ({ ...current, form: t('projectSaveFailed') })),
  });

  const archiveMutation = useMutation({
    mutationFn: archiveProject,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
    },
    onError: () => setErrors((current) => ({ ...current, form: t('projectArchiveFailed') })),
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
      updateMutation.mutate({ projectId: editingProjectId, input });
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
    });
  }

  function cancelEditing() {
    setEditingProjectId(null);
    setForm(emptyProjectForm);
    setErrors({});
  }

  const isSaving = createMutation.isPending || updateMutation.isPending;

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
              <strong>{projects.length}</strong>
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
            {projects.map((project) => (
              <article
                className={editingProjectId === project.id ? 'client-row selected' : 'client-row'}
                key={project.id}
              >
                <div className="client-row-main">
                  <div className="project-color-dot" style={{ backgroundColor: project.color }} aria-hidden="true" />
                  <div className="client-row-copy">
                    <div className="client-row-title">
                      <strong>{project.name}</strong>
                      <span className="status-pill">
                        <CircleCheck aria-hidden="true" />
                        {t('active')}
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
                  <button
                    className="secondary-button icon-button danger-button"
                    type="button"
                    onClick={() => archiveMutation.mutate(project.id)}
                    title={t('archive')}
                  >
                    <Trash2 aria-hidden="true" />
                  </button>
                </div>
              </article>
            ))}
          </div>
        </div>

        <form className="client-editor" noValidate onSubmit={submitProject}>
          <div className="editor-header">
            <div>
              <span>{editingProjectId ? t('editingProject') : t('createProject')}</span>
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
                {clients.map((client) => (
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
};

type TaskFormErrors = Partial<Record<keyof TaskFormState | 'form', string>>;

const emptyTaskForm: TaskFormState = {
  projectId: '',
  name: '',
  billable: true,
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
  const [editingTaskId, setEditingTaskId] = useState<string | null>(null);
  const [form, setForm] = useState<TaskFormState>(emptyTaskForm);
  const [errors, setErrors] = useState<TaskFormErrors>({});
  const sortedTasks = useMemo(() => sortTasksByNewest(tasks), [tasks]);

  const createMutation = useMutation({
    mutationFn: createTask,
    onSuccess: (created) => {
      queryClient.setQueryData(['tasks'], (current: { tasks: Task[] } | undefined) => {
        if (!current) {
          return current;
        }
        return {
          tasks: sortTasksByNewest([created, ...current.tasks.filter((item) => item.id !== created.id)]),
        };
      });
      setForm(emptyTaskForm);
      setErrors({});
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
    },
    onError: () => setErrors((current) => ({ ...current, form: t('taskSaveFailed') })),
  });

  const updateMutation = useMutation({
    mutationFn: ({ taskId, input }: { taskId: string; input: TaskInput }) => updateTask(taskId, input),
    onSuccess: () => {
      setEditingTaskId(null);
      setForm(emptyTaskForm);
      setErrors({});
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
    },
    onError: () => setErrors((current) => ({ ...current, form: t('taskSaveFailed') })),
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
    onError: () => setErrors((current) => ({ ...current, form: t('taskSaveFailed') })),
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
    },
    onError: () => setErrors((current) => ({ ...current, form: t('taskArchiveFailed') })),
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
      updateMutation.mutate({ taskId: editingTaskId, input });
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
    });
  }

  function cancelEditing() {
    setEditingTaskId(null);
    setForm(emptyTaskForm);
    setErrors({});
  }

  const isSaving = createMutation.isPending || updateMutation.isPending;

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
              <strong>{sortedTasks.length}</strong>
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
            {sortedTasks.map((task) => (
              <article className={editingTaskId === task.id ? 'client-row selected' : 'client-row'} key={task.id}>
                <div className="client-row-main">
                  {task.projectColor ? (
                    <div className="project-color-dot" style={{ backgroundColor: task.projectColor }} aria-hidden="true" />
                  ) : (
                    <div className="project-color-dot" style={{ backgroundColor: '#64748b' }} aria-hidden="true" />
                  )}
                  <div className="client-row-copy">
                    <div className="client-row-title">
                      <TaskInlineNameInput onSave={saveInlineTaskName} t={t} task={task} />
                      <span className="status-pill">
                        <CircleCheck aria-hidden="true" />
                        {t('active')}
                      </span>
                    </div>
                    <span className="client-contact">
                      <FolderKanban aria-hidden="true" />
                      {task.projectName || t('taskProjectOptional')}
                    </span>
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
                  <button
                    className="secondary-button icon-button danger-button"
                    type="button"
                    onClick={() => archiveMutation.mutate(task.id)}
                    title={t('archive')}
                  >
                    <Trash2 aria-hidden="true" />
                  </button>
                </div>
              </article>
            ))}
          </div>
        </div>

        <form className="client-editor" noValidate onSubmit={submitTask}>
          <div className="editor-header">
            <div>
              <span>{editingTaskId ? t('editingTask') : t('createTask')}</span>
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
                {projects.map((project) => (
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
};

type TagFormErrors = Partial<Record<keyof TagFormState | 'form', string>>;

const emptyTagForm: TagFormState = {
  name: '',
  color: '#64748b',
};

function TagPanel({ isLoading, tags, t }: { isLoading: boolean; tags: TagRecord[]; t: Translator }) {
  const queryClient = useQueryClient();
  const [editingTagId, setEditingTagId] = useState<string | null>(null);
  const [form, setForm] = useState<TagFormState>(emptyTagForm);
  const [errors, setErrors] = useState<TagFormErrors>({});

  const createMutation = useMutation({
    mutationFn: createTag,
    onSuccess: () => {
      setForm(emptyTagForm);
      setErrors({});
      queryClient.invalidateQueries({ queryKey: ['tags'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
    },
    onError: () => setErrors((current) => ({ ...current, form: t('tagSaveFailed') })),
  });

  const updateMutation = useMutation({
    mutationFn: ({ tagId, input }: { tagId: string; input: TagInput }) => updateTag(tagId, input),
    onSuccess: () => {
      setEditingTagId(null);
      setForm(emptyTagForm);
      setErrors({});
      queryClient.invalidateQueries({ queryKey: ['tags'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
    },
    onError: () => setErrors((current) => ({ ...current, form: t('tagSaveFailed') })),
  });

  const deleteMutation = useMutation({
    mutationFn: deleteTag,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tags'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
    },
    onError: () => setErrors((current) => ({ ...current, form: t('tagDeleteFailed') })),
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
      updateMutation.mutate({ tagId: editingTagId, input });
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
    });
  }

  function cancelEditing() {
    setEditingTagId(null);
    setForm(emptyTagForm);
    setErrors({});
  }

  const isSaving = createMutation.isPending || updateMutation.isPending;

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
              <strong>{tags.length}</strong>
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
            {tags.map((tag) => (
              <article className={editingTagId === tag.id ? 'client-row selected' : 'client-row'} key={tag.id}>
                <div className="client-row-main">
                  <div className="project-color-dot" style={{ backgroundColor: tag.color }} aria-hidden="true" />
                  <div className="client-row-copy">
                    <div className="client-row-title">
                      <strong>{tag.name}</strong>
                      <span className="status-pill">
                        <CircleCheck aria-hidden="true" />
                        {t('active')}
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
                  <button
                    className="secondary-button icon-button danger-button"
                    type="button"
                    onClick={() => deleteMutation.mutate(tag.id)}
                    title={t('delete')}
                  >
                    <Trash2 aria-hidden="true" />
                  </button>
                </div>
              </article>
            ))}
          </div>
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

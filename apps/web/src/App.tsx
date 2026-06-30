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
  CirclePlay,
  CircleStop,
  Columns3,
  DollarSign,
  EllipsisVertical,
  Pencil,
  FolderKanban,
  Import,
  Languages,
  LayoutDashboard,
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
import { FormEvent, useMemo, useState } from 'react';
import {
  archiveClient,
  archiveProject,
  createClient,
  createProject,
  fetchClients,
  fetchProjects,
  fetchSession,
  login,
  logout,
  updateClient,
  updateProject,
  type Client,
  type ClientInput,
  type LayoutMode,
  type Locale,
  type Project,
  type ProjectInput,
} from './lib/api';
import { translate } from './lib/i18n';
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

function Dashboard({ layoutMode, locale, setLayoutMode, setLocale, t, userName }: DashboardProps) {
  const queryClient = useQueryClient();
  const clientsQuery = useQuery({ queryKey: ['clients'], queryFn: fetchClients });
  const projectsQuery = useQuery({ queryKey: ['projects'], queryFn: fetchProjects });
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

        <div className="sidebar-timer">
          <div>
            <span>{t('currentTimer')}</span>
            <strong>00:40:47</strong>
          </div>
          <button className="sidebar-stop-button" type="button" title={t('stop')}>
            <CircleStop aria-hidden="true" />
          </button>
        </div>

        <nav className="sidebar-nav">
          <a href="#dashboard">
            <LayoutDashboard aria-hidden="true" />
            {t('dashboard')}
          </a>
          <a className="active" href="#timesheet">
            <Clock3 aria-hidden="true" />
            {t('time')}
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

        <section className="timer-command-row" aria-label={t('currentTimer')}>
          <div className="active-timer-card">
            <div className="timer-description">Cropper de Imagenes en todo el BackOffice [Serializers]</div>
            <EntityPill color="#ff714b" label="RTVE" />
            <button className="quiet-icon-button" type="button" title={t('tags')}>
              <Tag aria-hidden="true" />
            </button>
            <button className="quiet-icon-button billable" type="button" title={t('billable')}>
              <DollarSign aria-hidden="true" />
            </button>
            <strong className="timer-clock">00:40:47</strong>
          </div>
          <button className="stop-timer-button" type="button" title={t('stop')}>
            <CircleStop aria-hidden="true" />
          </button>
          <button className="manual-entry-button" type="button">
            <Plus aria-hidden="true" />
            {t('manualTimeEntry')}
          </button>
        </section>

        <TimeEntriesPreview t={t} />

        <section className="management-surface" aria-label={t('manage')}>
          <ClientPanel clients={clientsQuery.data?.clients ?? []} isLoading={clientsQuery.isLoading} t={t} />
          <ProjectPanel
            clients={clientsQuery.data?.clients ?? []}
            isLoading={projectsQuery.isLoading}
            projects={projectsQuery.data?.projects ?? []}
            t={t}
          />
        </section>
      </main>
    </div>
  );
}

type TimeEntryPreviewItem = {
  billable?: boolean;
  color: string;
  count?: number;
  description: string;
  duration: string;
  id: string;
  project: string;
  selected?: boolean;
  timeRange: string;
};

type TimeEntryPreviewDay = {
  date: string;
  day: string;
  entries: TimeEntryPreviewItem[];
  total: string;
};

const timePreviewDays: TimeEntryPreviewDay[] = [
  {
    day: 'Monday',
    date: '2026-06-29',
    total: '9h 09min',
    entries: [
      {
        id: 'mon-1',
        description: 'Cropper de Imagenes en todo el BackOffice [Serializers]',
        project: 'RTVE',
        color: '#ff714b',
        timeRange: '16:41 - 19:33',
        duration: '2h 51min',
      },
      {
        id: 'mon-2',
        description: 'Meet [tech]',
        project: 'Meet / Mails / Catch up',
        color: '#fff15c',
        timeRange: '15:30 - 16:41',
        duration: '1h 11min',
        billable: true,
      },
      {
        id: 'mon-3',
        description: 'Refactor Quiz [Endpoint rapido con respuestas codificadas como ENACT, participacion anonima]',
        project: 'RTVE',
        color: '#ff714b',
        count: 2,
        timeRange: '08:04 - 15:30',
        duration: '5h 06min',
        selected: true,
      },
    ],
  },
  {
    day: 'Friday',
    date: '2026-06-26',
    total: '6h 00min',
    entries: [
      {
        id: 'fri-1',
        description: 'Porto General Assembly',
        project: 'ENACT',
        color: '#45aaf2',
        timeRange: '09:00 - 15:00',
        duration: '6h 00min',
      },
    ],
  },
  {
    day: 'Thursday',
    date: '2026-06-25',
    total: '8h 00min',
    entries: [
      {
        id: 'thu-1',
        description: 'Porto General Assembly',
        project: 'ENACT',
        color: '#45aaf2',
        timeRange: '09:00 - 17:00',
        duration: '8h 00min',
      },
    ],
  },
  {
    day: 'Wednesday',
    date: '2026-06-24',
    total: '1h 56min',
    entries: [
      {
        id: 'wed-1',
        description: 'Reunion Ari + Correcciones',
        project: 'Atempora.app',
        color: '#ffb02e',
        timeRange: '18:00 - 19:57',
        duration: '1h 56min',
      },
    ],
  },
  {
    day: 'Tuesday',
    date: '2026-06-23',
    total: '7h 09min',
    entries: [
      {
        id: 'tue-1',
        description: 'Alignment Meeting for the Porto General Assembly [Docs] + Tests de stress nuevos',
        project: 'ENACT',
        color: '#45aaf2',
        count: 2,
        timeRange: '09:04 - 15:10',
        duration: '4h 26min',
      },
      {
        id: 'tue-2',
        description: 'Reunion de seguimiento',
        project: 'RTVE',
        color: '#ff714b',
        timeRange: '12:26 - 14:05',
        duration: '1h 39min',
      },
      {
        id: 'tue-3',
        description: 'Visibility Results en PRE [Correcciones]',
        project: 'RTVE',
        color: '#ff714b',
        timeRange: '08:00 - 09:04',
        duration: '1h 04min',
      },
    ],
  },
  {
    day: 'Monday',
    date: '2026-06-22',
    total: '9h 09min',
    entries: [
      {
        id: 'mon-prev-1',
        description: 'Reu Nico Visibility Results',
        project: 'RTVE',
        color: '#ff714b',
        count: 2,
        timeRange: '14:09 - 17:49',
        duration: '2h 13min',
      },
      {
        id: 'mon-prev-2',
        description: 'Meet [tech]',
        project: 'Meet / Mails / Catch up',
        color: '#fff15c',
        timeRange: '15:30 - 15:59',
        duration: '0h 29min',
        billable: true,
      },
      {
        id: 'mon-prev-3',
        description: 'Visibility Results',
        project: 'RTVE',
        color: '#ff714b',
        timeRange: '15:05 - 15:30',
        duration: '0h 25min',
      },
      {
        id: 'mon-prev-4',
        description: 'Alignment Meeting for the Porto General Assembly [Docs] + Tests de stress nuevos',
        project: 'ENACT',
        color: '#45aaf2',
        timeRange: '08:07 - 14:09',
        duration: '6h 01min',
      },
    ],
  },
];

function TimeEntriesPreview({ t }: { t: Translator }) {
  return (
    <section className="time-list-panel" id="timesheet" aria-labelledby="timesheet-title">
      <div className="time-list-toolbar">
        <label className="select-all-control">
          <span className="entry-checkbox" aria-hidden="true" />
          {t('selectAll')}
        </label>
        <strong id="timesheet-title">{t('timesheet')}</strong>
      </div>
      <div className="time-entry-list" role="table" aria-label={t('timesheet')}>
        {timePreviewDays.map((day) => (
          <div className="time-day-group" role="rowgroup" key={`${day.day}-${day.date}`}>
            <div className="day-group-header" role="row">
              <div>
                <CalendarDays aria-hidden="true" />
                <strong>{day.day}</strong>
                <span>{day.date}</span>
              </div>
              <strong>{day.total}</strong>
            </div>
            {day.entries.map((entry) => (
              <div className={entry.selected ? 'time-entry-row selected' : 'time-entry-row'} role="row" key={entry.id}>
                <span className="entry-checkbox" aria-hidden="true" />
                <div className="entry-task">
                  {entry.count ? <span className="entry-count">{entry.count}</span> : null}
                  <strong>{entry.description}</strong>
                </div>
                <EntityPill color={entry.color} label={entry.project} />
                <div className="entry-flags">
                  <Tag aria-hidden="true" />
                  <DollarSign aria-hidden="true" className={entry.billable ? 'billable-on' : undefined} />
                </div>
                <span className="entry-time">{entry.timeRange}</span>
                <strong className="entry-duration">{entry.duration}</strong>
                <button className={entry.selected ? 'play-entry-button active' : 'play-entry-button'} type="button">
                  <CirclePlay aria-hidden="true" />
                </button>
                <button className="more-entry-button" type="button" title={t('moreActions')}>
                  <EllipsisVertical aria-hidden="true" />
                </button>
              </div>
            ))}
          </div>
        ))}
      </div>
    </section>
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

function fieldClass(error?: string) {
  return error ? 'form-field has-error' : 'form-field';
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

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  BadgeDollarSign,
  CalendarDays,
  Building2,
  Clock3,
  CircleAlert,
  CircleCheck,
  Columns3,
  Download,
  Pencil,
  FileText,
  FolderKanban,
  Languages,
  LayoutDashboard,
  LogOut,
  Mail,
  Minimize2,
  PanelLeft,
  Play,
  Plus,
  Save,
  Square,
  Tags,
  Trash2,
  X,
} from 'lucide-react';
import { FormEvent, useMemo, useState } from 'react';
import {
  archiveClient,
  archiveProject,
  createClient,
  createProject,
  fetchClients,
  fetchOverview,
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
  type Overview,
  type Project,
  type ProjectInput,
} from './lib/api';
import { translate } from './lib/i18n';
import { usePersistentState } from './lib/persistentState';

const emptyOverview: Overview = {
  clientsTotal: 0,
  projectsTotal: 0,
  tasksTotal: 0,
  tagsTotal: 0,
  timeEntriesTotal: 0,
  invoicesTotal: 0,
  openTimers: 0,
};

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
  const overviewQuery = useQuery({ queryKey: ['overview'], queryFn: fetchOverview });
  const clientsQuery = useQuery({ queryKey: ['clients'], queryFn: fetchClients });
  const projectsQuery = useQuery({ queryKey: ['projects'], queryFn: fetchProjects });
  const overview = overviewQuery.data ?? emptyOverview;
  const logoutMutation = useMutation({
    mutationFn: logout,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['session'] }),
  });

  return (
    <div className={`app-shell layout-${layoutMode}`}>
      <aside className="sidebar" aria-label="Primary">
        <div className="brand-row">
          <Clock3 aria-hidden="true" />
          <span>{t('appName')}</span>
        </div>
        <nav>
          <a className="active" href="#dashboard">
            <LayoutDashboard aria-hidden="true" />
            {t('dashboard')}
          </a>
          <a href="#timesheet">
            <Clock3 aria-hidden="true" />
            {t('timesheet')}
          </a>
          <a href="#calendar">
            <CalendarDays aria-hidden="true" />
            {t('calendar')}
          </a>
          <a href="#reports">
            <FileText aria-hidden="true" />
            {t('reports')}
          </a>
          <a href="#clients">
            <Building2 aria-hidden="true" />
            {t('clients')}
          </a>
          <a href="#projects">
            <FolderKanban aria-hidden="true" />
            {t('projects')}
          </a>
        </nav>
      </aside>

      <main className="workspace">
        <header className="topbar">
          <div>
            <p>{t('today')}</p>
            <h1>{userName}</h1>
          </div>
          <div className="toolbar">
            <button type="button" title={t('language')} onClick={() => setLocale(locale === 'es' ? 'en' : 'es')}>
              <Languages aria-hidden="true" />
            </button>
            <LayoutSwitcher layoutMode={layoutMode} setLayoutMode={setLayoutMode} t={t} />
            <button type="button" title={t('logout')} onClick={() => logoutMutation.mutate()}>
              <LogOut aria-hidden="true" />
            </button>
          </div>
        </header>

        <section className="timer-band" aria-labelledby="timer-title">
          <div>
            <p>{t('offlineReady')}</p>
            <h2 id="timer-title">{t('trackWork')}</h2>
          </div>
          <div className="timer-display">00:00:00</div>
          <div className="timer-actions">
            <button type="button">
              <Play aria-hidden="true" />
              {t('start')}
            </button>
            <button className="secondary-button" type="button">
              <Square aria-hidden="true" />
              {t('stop')}
            </button>
          </div>
        </section>

        <section className="metrics-grid" aria-label="Overview">
          <Metric label={t('clients')} value={overview.clientsTotal} />
          <Metric label={t('projects')} value={overview.projectsTotal} />
          <Metric label={t('tasks')} value={overview.tasksTotal} />
          <Metric label={t('invoices')} value={overview.invoicesTotal} />
        </section>

        <ClientPanel clients={clientsQuery.data?.clients ?? []} isLoading={clientsQuery.isLoading} t={t} />
        <ProjectPanel
          clients={clientsQuery.data?.clients ?? []}
          isLoading={projectsQuery.isLoading}
          projects={projectsQuery.data?.projects ?? []}
          t={t}
        />

        <section className="work-grid">
          <div className="panel" id="timesheet">
            <div className="panel-heading">
              <h2>{t('thisWeek')}</h2>
              <button type="button">
                <Download aria-hidden="true" />
                {t('export')}
              </button>
            </div>
            <div className="timesheet-table" role="table" aria-label={t('timesheet')}>
              {['Cliente A', 'Proyecto API', 'Factura Junio'].map((row, index) => (
                <div className="timesheet-row" role="row" key={row}>
                  <span>{row}</span>
                  <span>{index === 0 ? '08:15' : index === 1 ? '14:40' : '03:30'}</span>
                  <span>{index === 2 ? 'EUR 280.00' : 'EUR 0.00'}</span>
                </div>
              ))}
            </div>
          </div>

          <div className="panel" id="calendar">
            <div className="panel-heading">
              <h2>{t('calendar')}</h2>
              <Tags aria-hidden="true" />
            </div>
            <div className="calendar-strip" aria-label={t('calendar')}>
              {['Lu', 'Ma', 'Mi', 'Ju', 'Vi'].map((day, index) => (
                <div className="day-column" key={day}>
                  <strong>{day}</strong>
                  <span>{index + 2}h</span>
                </div>
              ))}
            </div>
          </div>
        </section>
      </main>
    </div>
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

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <article className="metric-card">
      <span>{label}</span>
      <strong>{value}</strong>
    </article>
  );
}

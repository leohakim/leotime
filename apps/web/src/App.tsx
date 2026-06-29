import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  CalendarDays,
  Building2,
  Clock3,
  Columns3,
  Download,
  Pencil,
  FileText,
  Languages,
  LayoutDashboard,
  LogOut,
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
  createClient,
  fetchClients,
  fetchOverview,
  fetchSession,
  login,
  logout,
  updateClient,
  type Client,
  type ClientInput,
  type LayoutMode,
  type Locale,
  type Overview,
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

const emptyClientInput: ClientInput = {
  name: '',
  email: '',
  taxId: '',
  billingAddress: '',
  defaultCurrency: 'EUR',
  defaultHourlyRateMinor: 0,
};

function ClientPanel({ clients, isLoading, t }: { clients: Client[]; isLoading: boolean; t: Translator }) {
  const queryClient = useQueryClient();
  const [editingClientId, setEditingClientId] = useState<string | null>(null);
  const [form, setForm] = useState<ClientInput>(emptyClientInput);

  const createMutation = useMutation({
    mutationFn: createClient,
    onSuccess: () => {
      setForm(emptyClientInput);
      queryClient.invalidateQueries({ queryKey: ['clients'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ clientId, input }: { clientId: string; input: ClientInput }) => updateClient(clientId, input),
    onSuccess: () => {
      setEditingClientId(null);
      setForm(emptyClientInput);
      queryClient.invalidateQueries({ queryKey: ['clients'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
    },
  });

  const archiveMutation = useMutation({
    mutationFn: archiveClient,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['clients'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
    },
  });

  function submitClient(event: FormEvent) {
    event.preventDefault();
    if (editingClientId) {
      updateMutation.mutate({ clientId: editingClientId, input: form });
      return;
    }
    createMutation.mutate(form);
  }

  function startEditing(client: Client) {
    setEditingClientId(client.id);
    setForm({
      name: client.name,
      email: client.email,
      taxId: client.taxId,
      billingAddress: client.billingAddress,
      defaultCurrency: client.defaultCurrency,
      defaultHourlyRateMinor: client.defaultHourlyRateMinor,
    });
  }

  function cancelEditing() {
    setEditingClientId(null);
    setForm(emptyClientInput);
  }

  const isSaving = createMutation.isPending || updateMutation.isPending;

  return (
    <section className="panel clients-panel" id="clients" aria-labelledby="clients-title">
      <div className="panel-heading">
        <h2 id="clients-title">{t('clients')}</h2>
        <Building2 aria-hidden="true" />
      </div>

      <form className="client-form" onSubmit={submitClient}>
        <label>
          {t('name')}
          <input
            value={form.name}
            onChange={(event) => setForm({ ...form, name: event.target.value })}
            placeholder={t('newClient')}
          />
        </label>
        <label>
          {t('email')}
          <input
            value={form.email}
            onChange={(event) => setForm({ ...form, email: event.target.value })}
            type="email"
          />
        </label>
        <label>
          {t('taxId')}
          <input value={form.taxId} onChange={(event) => setForm({ ...form, taxId: event.target.value })} />
        </label>
        <label>
          {t('defaultCurrency')}
          <input
            maxLength={3}
            value={form.defaultCurrency}
            onChange={(event) => setForm({ ...form, defaultCurrency: event.target.value.toUpperCase() })}
          />
        </label>
        <label>
          {t('defaultHourlyRateMinor')}
          <input
            min={0}
            value={form.defaultHourlyRateMinor}
            onChange={(event) => setForm({ ...form, defaultHourlyRateMinor: Number(event.target.value) })}
            type="number"
          />
        </label>
        <label className="client-address-field">
          {t('billingAddress')}
          <input
            value={form.billingAddress}
            onChange={(event) => setForm({ ...form, billingAddress: event.target.value })}
          />
        </label>
        <div className="client-form-actions">
          <button type="submit" disabled={isSaving}>
            {editingClientId ? <Save aria-hidden="true" /> : <Plus aria-hidden="true" />}
            {editingClientId ? t('save') : t('newClient')}
          </button>
          {editingClientId ? (
            <button className="secondary-button" type="button" onClick={cancelEditing}>
              <X aria-hidden="true" />
              {t('cancel')}
            </button>
          ) : null}
        </div>
      </form>

      <div className="client-list" aria-busy={isLoading}>
        {clients.length === 0 ? <p className="empty-state">{t('noClients')}</p> : null}
        {clients.map((client) => (
          <article className="client-row" key={client.id}>
            <div>
              <strong>{client.name}</strong>
              <span>
                {client.defaultCurrency} {formatMinor(client.defaultHourlyRateMinor)}
              </span>
            </div>
            <div className="client-row-actions">
              <button className="secondary-button" type="button" onClick={() => startEditing(client)} title={t('edit')}>
                <Pencil aria-hidden="true" />
              </button>
              <button
                className="secondary-button danger-button"
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
    </section>
  );
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

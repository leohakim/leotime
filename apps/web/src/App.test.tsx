import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { App } from './App';

describe('App', () => {
  beforeEach(() => {
    window.localStorage.clear();
    clientsMock = [
      {
        id: 'cli_1',
        name: 'Osoigo SL',
        email: 'billing@example.com',
        taxId: 'B12345678',
        billingAddress: 'Madrid',
        defaultCurrency: 'EUR',
        defaultHourlyRateMinor: 7500,
        archivedAt: '',
        createdAt: '2026-01-01T00:00:00Z',
        updatedAt: '2026-01-01T00:00:00Z',
      },
    ];
    projectsMock = [
      {
        id: 'prj_1',
        clientId: 'cli_1',
        clientName: 'Osoigo SL',
        name: 'Portal Web',
        color: '#2563eb',
        defaultHourlyRateMinor: 8000,
        archivedAt: '',
        createdAt: '2026-01-01T00:00:00Z',
        updatedAt: '2026-01-01T00:00:00Z',
      },
    ];
    tasksMock = [
      {
        id: 'tsk_1',
        projectId: 'prj_1',
        projectName: 'Portal Web',
        projectColor: '#2563eb',
        name: 'Refactor API',
        billable: true,
        archivedAt: '',
        createdAt: '2026-01-01T00:00:00Z',
        updatedAt: '2026-01-01T00:00:00Z',
      },
    ];
    tagsMock = [
      {
        id: 'tag_1',
        name: 'Deep Work',
        color: '#2563eb',
        createdAt: '2026-01-01T00:00:00Z',
        updatedAt: '2026-01-01T00:00:00Z',
      },
    ];
    timeEntriesMock = [];
    timersMock = [];
    vi.stubGlobal('fetch', vi.fn(mockFetch));
  });

  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  test('renders the authenticated dashboard', async () => {
    renderApp();

    expect(await screen.findByRole('heading', { name: 'Time Tracker' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Entrada manual' })).toBeInTheDocument();
    expect(screen.getByRole('table', { name: 'Timesheet' })).toBeInTheDocument();
    expect(await screen.findByText('Esta semana')).toBeInTheDocument();
    expect(screen.getByText('Total semana')).toBeInTheDocument();
    expect((await screen.findAllByText('Osoigo SL')).length).toBeGreaterThan(0);
    expect((await screen.findAllByText('Portal Web')).length).toBeGreaterThan(0);
    expect((await screen.findAllByText('Refactor API')).length).toBeGreaterThan(0);
    expect((await screen.findAllByText('Deep Work')).length).toBeGreaterThan(0);
    expect(screen.getByRole('button', { name: 'Iniciar timer' })).toBeInTheDocument();
    expect(screen.getByText('Sin timer activo')).toBeInTheDocument();
  });

  test('navigates the weekly timesheet', async () => {
    renderApp();

    await screen.findByText('Esta semana');
    fireEvent.click(screen.getByTitle('Semana anterior'));

    await waitFor(() => expect(screen.queryByText('Esta semana')).not.toBeInTheDocument());
    expect(screen.getByRole('button', { name: 'Hoy' })).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Hoy' }));
    await waitFor(() => expect(screen.getByText('Esta semana')).toBeInTheDocument());
  });

  test('renders dashboard stats widgets', async () => {
    renderApp();

    expect(await screen.findByText('Entradas recientes')).toBeInTheDocument();
    expect(screen.getByText('Ultimos 7 dias')).toBeInTheDocument();
    expect(screen.getByText('Resumen semanal')).toBeInTheDocument();
    expect(screen.getByLabelText('Mes anterior')).toBeInTheDocument();
    expect(screen.getByText('Tiempo registrado')).toBeInTheDocument();
  });

  test('renders the time report panel', async () => {
    renderApp();

    expect(await screen.findByRole('heading', { name: 'Resumen' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Descargar CSV' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Descargar JSON' })).toBeInTheDocument();
  });

  test('renders the invoice panel', async () => {
    renderApp();

    expect(await screen.findByRole('heading', { name: 'Facturas' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Crear borrador' })).toBeInTheDocument();
    expect(await screen.findByText('INV-2026-001')).toBeInTheDocument();
  });

  test('opens the calendar view', async () => {
    renderApp();

    await screen.findByText('Esta semana');
    fireEvent.click(screen.getByRole('tab', { name: 'Calendario' }));

    expect(await screen.findByText('Este mes')).toBeInTheDocument();
    expect(screen.getByRole('grid', { name: 'Calendario' })).toBeInTheDocument();
  });

  test('switches language', async () => {
    renderApp();

    await screen.findByRole('heading', { name: 'Time Tracker' });
    fireEvent.click(screen.getByTitle('Idioma'));

    await waitFor(() => expect(screen.getByRole('button', { name: 'Manual time entry' })).toBeInTheDocument());
  });

  test('creates a client from the dashboard', async () => {
    renderApp();

    await screen.findAllByText('Osoigo SL');
    fireEvent.change(screen.getByPlaceholderText('Ej. Cliente ACME'), { target: { value: 'Nuevo Cliente' } });
    fireEvent.change(screen.getByPlaceholderText('facturacion@cliente.com'), {
      target: { value: 'facturacion@nuevocliente.com' },
    });
    fireEvent.change(screen.getAllByPlaceholderText('75.00')[0], { target: { value: '82.50' } });
    fireEvent.click(screen.getByRole('button', { name: 'Crear cliente' }));

    await waitFor(() => expect(screen.getAllByText('Nuevo Cliente').length).toBeGreaterThan(0));
  });

  test('validates the client form before submitting', async () => {
    renderApp();

    await screen.findAllByText('Osoigo SL');
    fireEvent.click(screen.getByRole('button', { name: 'Crear cliente' }));

    expect(await screen.findByText('El nombre es obligatorio.')).toBeInTheDocument();
    expect(clientsMock).toHaveLength(1);

    fireEvent.change(screen.getByPlaceholderText('Ej. Cliente ACME'), { target: { value: 'Cliente Valido' } });
    fireEvent.change(screen.getByPlaceholderText('facturacion@cliente.com'), { target: { value: 'correo-invalido' } });
    fireEvent.click(screen.getByRole('button', { name: 'Crear cliente' }));

    expect(await screen.findByText('Escribe un email valido o deja el campo vacio.')).toBeInTheDocument();
    expect(clientsMock).toHaveLength(1);
  });

  test('creates a project from the dashboard', async () => {
    renderApp();

    await screen.findAllByText('Portal Web');
    fireEvent.change(screen.getByPlaceholderText('Ej. Rediseño web'), { target: { value: 'Nuevo Proyecto' } });
    fireEvent.change(screen.getByLabelText('Cliente', { selector: '#project-client' }), { target: { value: 'cli_1' } });
    fireEvent.change(screen.getByPlaceholderText('#2563eb'), { target: { value: '#0f7a5b' } });
    fireEvent.change(screen.getAllByPlaceholderText('75.00')[1], { target: { value: '91.25' } });
    fireEvent.click(screen.getByRole('button', { name: 'Crear proyecto' }));

    await waitFor(() => expect(projectsMock).toHaveLength(2));
    expect(screen.getAllByText('Nuevo Proyecto').length).toBeGreaterThan(0);
  });

  test('validates the project form before submitting', async () => {
    renderApp();

    await screen.findAllByText('Portal Web');
    fireEvent.click(screen.getByRole('button', { name: 'Crear proyecto' }));

    expect(await screen.findByText('El nombre del proyecto es obligatorio.')).toBeInTheDocument();
    expect(projectsMock).toHaveLength(1);

    fireEvent.change(screen.getByPlaceholderText('Ej. Rediseño web'), { target: { value: 'Proyecto Valido' } });
    fireEvent.change(screen.getByPlaceholderText('#2563eb'), { target: { value: 'azul' } });
    fireEvent.click(screen.getByRole('button', { name: 'Crear proyecto' }));

    expect(await screen.findByText('Usa un color hex valido, por ejemplo #2563eb.')).toBeInTheDocument();
    expect(projectsMock).toHaveLength(1);
  });

  test('creates a task from the dashboard', async () => {
    renderApp();

    await screen.findAllByText('Refactor API');
    fireEvent.change(document.getElementById('task-name') as HTMLInputElement, { target: { value: 'Nueva Tarea' } });
    fireEvent.change(document.getElementById('task-project') as HTMLSelectElement, { target: { value: 'prj_1' } });
    fireEvent.click(screen.getByRole('button', { name: 'Crear tarea' }));

    await waitFor(() => expect(tasksMock).toHaveLength(2));
  });

  test('validates the task form before submitting', async () => {
    renderApp();

    await screen.findAllByText('Refactor API');
    fireEvent.click(screen.getByRole('button', { name: 'Crear tarea' }));

    expect(await screen.findByText('El nombre de la tarea es obligatorio.')).toBeInTheDocument();
    expect(tasksMock).toHaveLength(1);

    fireEvent.change(document.getElementById('task-name') as HTMLInputElement, { target: { value: 'A' } });
    fireEvent.click(screen.getByRole('button', { name: 'Crear tarea' }));

    expect(await screen.findByText('La tarea debe tener al menos 2 caracteres.')).toBeInTheDocument();
    expect(tasksMock).toHaveLength(1);
  });

  test('starts and stops a timer from the dashboard', async () => {
    renderApp();

    await screen.findByRole('button', { name: 'Iniciar timer' });
    fireEvent.change(screen.getByPlaceholderText('Que estas haciendo?'), { target: { value: 'Trabajo en vivo' } });
    fireEvent.click(screen.getByRole('button', { name: 'Iniciar timer' }));

    await waitFor(() => expect(timersMock).toHaveLength(1));
    expect(screen.getAllByText('Trabajo en vivo').length).toBeGreaterThan(0);

    fireEvent.click(screen.getAllByTitle('Parar')[0]);
    await waitFor(() => expect(timersMock).toHaveLength(0));
    expect(timeEntriesMock).toHaveLength(1);
  });

  test('opens the running timer popover to edit start time', async () => {
    renderApp();

    await screen.findByRole('button', { name: 'Iniciar timer' });
    fireEvent.click(screen.getByRole('button', { name: 'Iniciar timer' }));

    await waitFor(() => expect(timersMock).toHaveLength(1));

    fireEvent.click(screen.getByRole('button', { name: 'Editar hora de inicio' }));
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getAllByLabelText('Inicio').length).toBeGreaterThan(0);
    expect(screen.getByText('--:--')).toBeInTheDocument();

    const timeInput = screen.getAllByLabelText('Inicio').find((element) => element.getAttribute('type') === 'time');
    expect(timeInput).toBeDefined();
    const originalStartedAt = timersMock[0]?.startedAt;
    fireEvent.change(timeInput as HTMLInputElement, { target: { value: '08:30' } });

    await waitFor(() => expect(timersMock[0]?.startedAt).not.toBe(originalStartedAt));
  });

  test('creates a manual time entry from the dashboard', async () => {
    renderApp();

    await screen.findByRole('heading', { name: 'Time Tracker' });
    const manualPanel = within(document.getElementById('manual-time-entry')!);
    await waitFor(() => expect(manualPanel.getAllByText('Sin entradas todavia').length).toBeGreaterThan(0));
    fireEvent.change(document.getElementById('time-entry-description') as HTMLInputElement, { target: { value: 'Trabajo manual' } });
    fireEvent.click(manualPanel.getByRole('button', { name: 'Crear entrada' }));

    await waitFor(() => expect(timeEntriesMock).toHaveLength(1));
    expect(manualPanel.getByDisplayValue('Trabajo manual')).toBeInTheDocument();
  });

  test('creates a tag from the dashboard', async () => {
    renderApp();

    await screen.findAllByText('Deep Work');
    fireEvent.change(document.getElementById('tag-name') as HTMLInputElement, { target: { value: 'Nuevo Tag' } });
    fireEvent.change(document.getElementById('tag-color') as HTMLInputElement, { target: { value: '#0f7a5b' } });
    fireEvent.click(screen.getByRole('button', { name: 'Crear tag' }));

    await waitFor(() => expect(tagsMock).toHaveLength(2));
  });

  test('validates the tag form before submitting', async () => {
    renderApp();

    await screen.findAllByText('Deep Work');
    fireEvent.click(screen.getByRole('button', { name: 'Crear tag' }));

    expect(await screen.findByText('El nombre del tag es obligatorio.')).toBeInTheDocument();
    expect(tagsMock).toHaveLength(1);

    fireEvent.change(screen.getByPlaceholderText('Ej. Deep Work'), { target: { value: 'A' } });
    fireEvent.click(screen.getByRole('button', { name: 'Crear tag' }));

    expect(await screen.findByText('El tag debe tener al menos 2 caracteres.')).toBeInTheDocument();
    expect(tagsMock).toHaveLength(1);
  });
});

function renderApp() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>,
  );
}

let clientsMock: Array<{
  id: string;
  name: string;
  email: string;
  taxId: string;
  billingAddress: string;
  defaultCurrency: string;
  defaultHourlyRateMinor: number;
  archivedAt: string;
  createdAt: string;
  updatedAt: string;
}> = [];

let projectsMock: Array<{
  id: string;
  clientId: string;
  clientName: string;
  name: string;
  color: string;
  defaultHourlyRateMinor: number | null;
  archivedAt: string;
  createdAt: string;
  updatedAt: string;
}> = [];

let tasksMock: Array<{
  id: string;
  projectId: string;
  projectName: string;
  projectColor: string;
  name: string;
  billable: boolean;
  archivedAt: string;
  createdAt: string;
  updatedAt: string;
}> = [];

let tagsMock: Array<{
  id: string;
  name: string;
  color: string;
  createdAt: string;
  updatedAt: string;
}> = [];

let timeEntriesMock: Array<{
  id: string;
  clientId: string;
  clientName: string;
  projectId: string;
  projectName: string;
  projectColor: string;
  taskId: string;
  taskName: string;
  description: string;
  startedAt: string;
  endedAt: string;
  durationSeconds: number;
  billable: boolean;
  overlapWarning: boolean;
  source: string;
  tags: Array<{ id: string; name: string; color: string }>;
  createdAt: string;
  updatedAt: string;
}> = [];

let invoicesMock: Array<{
  id: string;
  clientId: string;
  invoiceNumber: string;
  status: 'draft' | 'issued' | 'paid' | 'cancelled';
  currency: string;
  issuedAt: string;
  dueAt: string;
  sellerName: string;
  sellerTaxId: string;
  sellerAddress: string;
  clientName: string;
  clientTaxId: string;
  clientAddress: string;
  subtotalMinor: number;
  taxMinor: number;
  withholdingMinor: number;
  totalMinor: number;
  notes: string;
  lines: Array<{
    id: string;
    timeEntryId: string;
    description: string;
    quantityMinutes: number;
    unitRateMinor: number;
    subtotalMinor: number;
    taxRateBasisPoints: number;
    createdAt: string;
  }>;
  createdAt: string;
  updatedAt: string;
}> = [
  {
    id: 'inv_1',
    clientId: 'cli_1',
    invoiceNumber: 'INV-2026-001',
    status: 'draft',
    currency: 'EUR',
    issuedAt: '',
    dueAt: '',
    sellerName: 'Administrador',
    sellerTaxId: '',
    sellerAddress: '',
    clientName: 'Osoigo SL',
    clientTaxId: '',
    clientAddress: '',
    subtotalMinor: 12000,
    taxMinor: 2520,
    withholdingMinor: 0,
    totalMinor: 14520,
    notes: '',
    lines: [
      {
        id: 'inl_1',
        timeEntryId: 'ten_1',
        description: 'Portal Web — Support',
        quantityMinutes: 60,
        unitRateMinor: 12000,
        subtotalMinor: 12000,
        taxRateBasisPoints: 2100,
        createdAt: '2026-01-01T00:00:00Z',
      },
    ],
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
  },
];

let timersMock: Array<{
  id: string;
  clientId: string;
  clientName: string;
  projectId: string;
  projectName: string;
  projectColor: string;
  taskId: string;
  taskName: string;
  description: string;
  startedAt: string;
  endedAt: string;
  durationSeconds: number;
  billable: boolean;
  overlapWarning: boolean;
  source: string;
  tags: Array<{ id: string; name: string; color: string }>;
  createdAt: string;
  updatedAt: string;
}> = [];

async function mockFetch(input: RequestInfo | URL, init?: RequestInit) {
  const url = String(input);
  if (url.endsWith('/api/v1/session')) {
    return jsonResponse({
      authenticated: true,
      user: {
        id: 'usr_test',
        email: 'admin@example.com',
        name: 'Administrador',
        locale: 'es',
        layoutMode: 'solid',
      },
    });
  }

  if (url.endsWith('/api/v1/overview')) {
    return jsonResponse({
      clientsTotal: clientsMock.length,
      projectsTotal: projectsMock.length,
      tasksTotal: tasksMock.length,
      tagsTotal: tagsMock.length,
      timeEntriesTotal: timeEntriesMock.length,
      invoicesTotal: 6,
      openTimers: timersMock.length,
    });
  }

  if (url.endsWith('/api/v1/clients') && (!init?.method || init.method === 'GET')) {
    return jsonResponse({ clients: clientsMock });
  }

  if (url.endsWith('/api/v1/clients') && init?.method === 'POST') {
    const body = JSON.parse(String(init.body));
    const client = {
      id: `cli_${clientsMock.length + 1}`,
      name: body.name,
      email: body.email,
      taxId: body.taxId,
      billingAddress: body.billingAddress,
      defaultCurrency: body.defaultCurrency,
      defaultHourlyRateMinor: body.defaultHourlyRateMinor,
      archivedAt: '',
      createdAt: '2026-01-01T00:00:00Z',
      updatedAt: '2026-01-01T00:00:00Z',
    };
    clientsMock = [...clientsMock, client];
    return jsonResponse(client, 201);
  }

  if (url.endsWith('/api/v1/projects') && (!init?.method || init.method === 'GET')) {
    return jsonResponse({ projects: projectsMock });
  }

  if (url.endsWith('/api/v1/projects') && init?.method === 'POST') {
    const body = JSON.parse(String(init.body));
    const client = clientsMock.find((item) => item.id === body.clientId);
    const project = {
      id: `prj_${projectsMock.length + 1}`,
      clientId: body.clientId,
      clientName: client?.name ?? '',
      name: body.name,
      color: body.color,
      defaultHourlyRateMinor: body.defaultHourlyRateMinor,
      archivedAt: '',
      createdAt: '2026-01-01T00:00:00Z',
      updatedAt: '2026-01-01T00:00:00Z',
    };
    projectsMock = [...projectsMock, project];
    return jsonResponse(project, 201);
  }

  if (url.endsWith('/api/v1/tasks') && (!init?.method || init.method === 'GET')) {
    return jsonResponse({ tasks: tasksMock });
  }

  if (url.endsWith('/api/v1/tasks') && init?.method === 'POST') {
    const body = JSON.parse(String(init.body));
    const project = projectsMock.find((item) => item.id === body.projectId);
    const task = {
      id: `tsk_${tasksMock.length + 1}`,
      projectId: body.projectId,
      projectName: project?.name ?? '',
      projectColor: project?.color ?? '',
      name: body.name,
      billable: body.billable,
      archivedAt: '',
      createdAt: '2026-01-01T00:00:00Z',
      updatedAt: '2026-01-01T00:00:00Z',
    };
    tasksMock = [...tasksMock, task];
    return jsonResponse(task, 201);
  }

  if (url.includes('/api/v1/tasks/') && init?.method === 'PATCH') {
    const taskId = url.split('/api/v1/tasks/')[1] ?? '';
    const body = JSON.parse(String(init.body));
    const index = tasksMock.findIndex((item) => item.id === taskId);
    if (index === -1) {
      return jsonResponse({ error: 'not found' }, 404);
    }
    const project = projectsMock.find((item) => item.id === body.projectId);
    const updated = {
      ...tasksMock[index],
      projectId: body.projectId,
      projectName: project?.name ?? '',
      projectColor: project?.color ?? '',
      name: body.name,
      billable: body.billable,
      updatedAt: '2026-01-01T00:00:00Z',
    };
    tasksMock = tasksMock.map((item) => (item.id === taskId ? updated : item));
    return jsonResponse(updated);
  }

  if (url.endsWith('/api/v1/tags') && (!init?.method || init.method === 'GET')) {
    return jsonResponse({ tags: tagsMock });
  }

  if (url.endsWith('/api/v1/tags') && init?.method === 'POST') {
    const body = JSON.parse(String(init.body));
    const tag = {
      id: `tag_${tagsMock.length + 1}`,
      name: body.name,
      color: body.color,
      createdAt: '2026-01-01T00:00:00Z',
      updatedAt: '2026-01-01T00:00:00Z',
    };
    tagsMock = [...tagsMock, tag];
    return jsonResponse(tag, 201);
  }

  if (url.includes('/api/v1/dashboard/stats') && (!init?.method || init.method === 'GET')) {
    return jsonResponse({
      activityMonth: '2026-07',
      recentEntries: timeEntriesMock.slice(0, 4).map((entry) => ({
        id: entry.id,
        clientId: entry.clientId,
        projectId: entry.projectId,
        projectName: entry.projectName,
        projectColor: entry.projectColor,
        taskId: entry.taskId,
        description: entry.description,
        startedAt: entry.startedAt,
        durationSeconds: entry.durationSeconds,
        billable: entry.billable,
      })),
      lastSevenDays: [
        { date: '2026-07-05', label: 'today', totalSeconds: 7200 },
        { date: '2026-07-04', label: 'yesterday', totalSeconds: 5400 },
        { date: '2026-07-03', label: '2d', totalSeconds: 3600 },
        { date: '2026-07-02', label: '3d', totalSeconds: 1800 },
        { date: '2026-07-01', label: '4d', totalSeconds: 0 },
        { date: '2026-06-30', label: '5d', totalSeconds: 9000 },
        { date: '2026-06-29', label: '6d', totalSeconds: 1200 },
      ],
      activityHeatmap: [
        { date: '2026-06-30', totalSeconds: 0, level: 0, inMonth: false },
        { date: '2026-07-01', totalSeconds: 5400, level: 2, inMonth: true },
        { date: '2026-07-02', totalSeconds: 7200, level: 2, inMonth: true },
      ],
      weekDays: [
        { date: '2026-06-30', weekday: 'Mon', totalSeconds: 3600 },
        { date: '2026-07-01', weekday: 'Tue', totalSeconds: 5400 },
        { date: '2026-07-02', weekday: 'Wed', totalSeconds: 1800 },
        { date: '2026-07-03', weekday: 'Thu', totalSeconds: 7200 },
        { date: '2026-07-04', weekday: 'Fri', totalSeconds: 3600 },
        { date: '2026-07-05', weekday: 'Sat', totalSeconds: 9000 },
        { date: '2026-07-06', weekday: 'Sun', totalSeconds: 0 },
      ],
      weekSpentSeconds: 30600,
      weekBillableSeconds: 12600,
      weekBillableMinor: 21000,
      weekCurrency: 'EUR',
      projectBreakdown: [
        { projectId: 'prj_1', projectName: 'Portal Web', projectColor: '#2563eb', totalSeconds: 18000 },
        { projectId: 'prj_2', projectName: 'ENACT', projectColor: '#f97316', totalSeconds: 12600 },
      ],
    });
  }

  if (url.endsWith('/api/v1/invoices') && (!init?.method || init.method === 'GET')) {
    return jsonResponse({ invoices: invoicesMock });
  }

  if (url.includes('/api/v1/invoices/') && (!init?.method || init.method === 'GET') && !url.includes('/export')) {
    const invoiceId = url.split('/api/v1/invoices/')[1]?.split('?')[0] ?? '';
    const invoice = invoicesMock.find((item) => item.id === invoiceId);
    if (!invoice) {
      return jsonResponse({ error: 'not found' }, 404);
    }
    return jsonResponse(invoice);
  }

  if (url.includes('/api/v1/reports/time') && (!init?.method || init.method === 'GET')) {
    const parsed = new URL(url, 'http://localhost');
    const includeTimestamps = parsed.searchParams.get('includeTimestamps') === 'true';
    const billableOnly = parsed.searchParams.get('billableOnly') === 'true';
    const entries = timeEntriesMock.filter((entry) => {
      if (billableOnly && !entry.billable) {
        return false;
      }
      return true;
    });
    const totalSeconds = entries.reduce((sum, entry) => sum + entry.durationSeconds, 0);
    if (includeTimestamps) {
      return jsonResponse({
        from: parsed.searchParams.get('from') ?? '',
        to: parsed.searchParams.get('to') ?? '',
        groupBy: 'project',
        includeTimestamps: true,
        billableOnly,
        totalSeconds,
        entryCount: entries.length,
        entries,
      });
    }
    return jsonResponse({
      from: parsed.searchParams.get('from') ?? '',
      to: parsed.searchParams.get('to') ?? '',
      groupBy: parsed.searchParams.get('groupBy') ?? 'project',
      includeTimestamps: false,
      billableOnly,
      totalSeconds,
      entryCount: entries.length,
      groups: [
        {
          key: 'prj_1',
          label: 'Portal Web',
          totalSeconds,
          entryCount: entries.length,
        },
      ],
    });
  }

  if (url.includes('/api/v1/time-entries') && (!init?.method || init.method === 'GET') && !url.includes('/api/v1/time-entries/')) {
    return jsonResponse({ timeEntries: timeEntriesMock });
  }

  if (url.endsWith('/api/v1/time-entries') && init?.method === 'POST') {
    const body = JSON.parse(String(init.body));
    const client = clientsMock.find((item) => item.id === body.clientId);
    const project = projectsMock.find((item) => item.id === body.projectId);
    const task = tasksMock.find((item) => item.id === body.taskId);
    const startedAt = body.startedAt;
    const endedAt = body.endedAt;
    const durationSeconds = Math.max(60, Math.floor((Date.parse(endedAt) - Date.parse(startedAt)) / 1000));
    const entry = buildTimeEntryMock(`ten_${timeEntriesMock.length + 1}`, body, {
      client,
      project,
      task,
      startedAt,
      endedAt,
      durationSeconds,
      source: 'manual',
    });
    timeEntriesMock = [...timeEntriesMock, entry];
    return jsonResponse(entry, 201);
  }

  if (url.includes('/api/v1/time-entries/') && init?.method === 'PATCH') {
    const timeEntryId = url.split('/api/v1/time-entries/')[1] ?? '';
    const body = JSON.parse(String(init.body));
    const index = timeEntriesMock.findIndex((item) => item.id === timeEntryId);
    if (index === -1) {
      return jsonResponse({ error: 'not found' }, 404);
    }
    const client = clientsMock.find((item) => item.id === body.clientId);
    const project = projectsMock.find((item) => item.id === body.projectId);
    const task = tasksMock.find((item) => item.id === body.taskId);
    const startedAt = body.startedAt;
    const endedAt = body.endedAt;
    const durationSeconds = Math.max(60, Math.floor((Date.parse(endedAt) - Date.parse(startedAt)) / 1000));
    const updated = buildTimeEntryMock(timeEntryId, body, {
      client,
      project,
      task,
      startedAt,
      endedAt,
      durationSeconds,
      source: timeEntriesMock[index].source,
    });
    timeEntriesMock = timeEntriesMock.map((item) => (item.id === timeEntryId ? updated : item));
    return jsonResponse(updated);
  }

  if (url.endsWith('/api/v1/timers') && (!init?.method || init.method === 'GET')) {
    return jsonResponse({ timers: timersMock });
  }

  if (url.endsWith('/api/v1/timers') && init?.method === 'POST') {
    const body = JSON.parse(String(init.body));
    const client = clientsMock.find((item) => item.id === body.clientId);
    const project = projectsMock.find((item) => item.id === body.projectId);
    const task = tasksMock.find((item) => item.id === body.taskId);
    const startedAt = new Date().toISOString();
    const entry = buildTimeEntryMock(`ten_timer_${timersMock.length + 1}`, body, {
      client,
      project,
      task,
      startedAt,
      endedAt: '',
      durationSeconds: 0,
      source: 'timer',
    });
    timersMock = [...timersMock, entry];
    return jsonResponse(entry, 201);
  }

  if (url.includes('/api/v1/timers/') && !url.endsWith('/stop') && init?.method === 'PATCH') {
    const timeEntryId = url.split('/api/v1/timers/')[1] ?? '';
    const timer = timersMock.find((item) => item.id === timeEntryId);
    if (!timer) {
      return jsonResponse({ error: 'not found' }, 404);
    }
    const body = JSON.parse(String(init.body));
    const updated = {
      ...timer,
      description: body.description ?? timer.description,
      startedAt: body.startedAt ?? timer.startedAt,
      billable: body.billable ?? timer.billable,
    };
    timersMock = timersMock.map((item) => (item.id === timeEntryId ? updated : item));
    return jsonResponse(updated);
  }

  if (url.includes('/api/v1/timers/') && url.endsWith('/stop') && init?.method === 'POST') {
    const timeEntryId = url.split('/api/v1/timers/')[1]?.replace('/stop', '') ?? '';
    const timer = timersMock.find((item) => item.id === timeEntryId);
    if (!timer) {
      return jsonResponse({ error: 'not found' }, 404);
    }
    const endedAt = new Date().toISOString();
    const durationSeconds = Math.max(60, Math.floor((Date.parse(endedAt) - Date.parse(timer.startedAt)) / 1000));
    const finalized = {
      ...timer,
      endedAt,
      durationSeconds,
    };
    timersMock = timersMock.filter((item) => item.id !== timeEntryId);
    timeEntriesMock = [...timeEntriesMock, finalized];
    return jsonResponse(finalized);
  }

  return jsonResponse({}, 404);
}

function buildTimeEntryMock(
  id: string,
  body: {
    clientId?: string;
    projectId?: string;
    taskId?: string;
    tagIds?: string[];
    description?: string;
    billable?: boolean;
  },
  context: {
    client?: (typeof clientsMock)[number];
    project?: (typeof projectsMock)[number];
    task?: (typeof tasksMock)[number];
    startedAt: string;
    endedAt: string;
    durationSeconds: number;
    source: string;
  },
) {
  return {
    id,
    clientId: body.clientId ?? '',
    clientName: context.client?.name ?? '',
    projectId: body.projectId ?? '',
    projectName: context.project?.name ?? '',
    projectColor: context.project?.color ?? '#64748b',
    taskId: body.taskId ?? '',
    taskName: context.task?.name ?? '',
    description: body.description ?? '',
    startedAt: context.startedAt,
    endedAt: context.endedAt,
    durationSeconds: context.durationSeconds,
    billable: body.billable ?? true,
    overlapWarning: false,
    source: context.source,
    tags: (body.tagIds ?? [])
      .map((tagId: string) => tagsMock.find((item) => item.id === tagId))
      .filter(Boolean),
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
  };
}

function jsonResponse(body: unknown, status = 200) {
  return Promise.resolve(
    new Response(JSON.stringify(body), {
      status,
      headers: {
        'Content-Type': 'application/json',
      },
    }),
  );
}

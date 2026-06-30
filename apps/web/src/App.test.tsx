import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
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
    expect(screen.getAllByText('Cropper de Imagenes en todo el BackOffice [Serializers]')).toHaveLength(2);
    expect((await screen.findAllByText('Osoigo SL')).length).toBeGreaterThan(0);
    expect(await screen.findByText('Portal Web')).toBeInTheDocument();
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

    await screen.findByText('Portal Web');
    fireEvent.change(screen.getByPlaceholderText('Ej. Rediseño web'), { target: { value: 'Nuevo Proyecto' } });
    fireEvent.change(screen.getByLabelText('Cliente'), { target: { value: 'cli_1' } });
    fireEvent.change(screen.getByPlaceholderText('#2563eb'), { target: { value: '#0f7a5b' } });
    fireEvent.change(screen.getAllByPlaceholderText('75.00')[1], { target: { value: '91.25' } });
    fireEvent.click(screen.getByRole('button', { name: 'Crear proyecto' }));

    await waitFor(() => expect(screen.getByText('Nuevo Proyecto')).toBeInTheDocument());
  });

  test('validates the project form before submitting', async () => {
    renderApp();

    await screen.findByText('Portal Web');
    fireEvent.click(screen.getByRole('button', { name: 'Crear proyecto' }));

    expect(await screen.findByText('El nombre del proyecto es obligatorio.')).toBeInTheDocument();
    expect(projectsMock).toHaveLength(1);

    fireEvent.change(screen.getByPlaceholderText('Ej. Rediseño web'), { target: { value: 'Proyecto Valido' } });
    fireEvent.change(screen.getByPlaceholderText('#2563eb'), { target: { value: 'azul' } });
    fireEvent.click(screen.getByRole('button', { name: 'Crear proyecto' }));

    expect(await screen.findByText('Usa un color hex valido, por ejemplo #2563eb.')).toBeInTheDocument();
    expect(projectsMock).toHaveLength(1);
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
      tasksTotal: 3,
      tagsTotal: 4,
      timeEntriesTotal: 5,
      invoicesTotal: 6,
      openTimers: 0,
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

  return jsonResponse({}, 404);
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

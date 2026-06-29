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
    vi.stubGlobal('fetch', vi.fn(mockFetch));
  });

  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  test('renders the authenticated dashboard', async () => {
    renderApp();

    expect(await screen.findByRole('heading', { name: 'Administrador' })).toBeInTheDocument();
    expect(screen.getByText('Registrar trabajo')).toBeInTheDocument();
    expect(screen.getByRole('table', { name: 'Timesheet' })).toBeInTheDocument();
    expect(await screen.findByText('Osoigo SL')).toBeInTheDocument();
  });

  test('switches language', async () => {
    renderApp();

    await screen.findByRole('heading', { name: 'Administrador' });
    fireEvent.click(screen.getByTitle('Idioma'));

    await waitFor(() => expect(screen.getByText('Track work')).toBeInTheDocument());
  });

  test('creates a client from the dashboard', async () => {
    renderApp();

    await screen.findByText('Osoigo SL');
    fireEvent.change(screen.getByPlaceholderText('Ej. Cliente ACME'), { target: { value: 'Nuevo Cliente' } });
    fireEvent.change(screen.getByPlaceholderText('facturacion@cliente.com'), {
      target: { value: 'facturacion@nuevocliente.com' },
    });
    fireEvent.change(screen.getByPlaceholderText('75.00'), { target: { value: '82.50' } });
    fireEvent.click(screen.getByRole('button', { name: 'Crear cliente' }));

    await waitFor(() => expect(screen.getByText('Nuevo Cliente')).toBeInTheDocument());
  });

  test('validates the client form before submitting', async () => {
    renderApp();

    await screen.findByText('Osoigo SL');
    fireEvent.click(screen.getByRole('button', { name: 'Crear cliente' }));

    expect(await screen.findByText('El nombre es obligatorio.')).toBeInTheDocument();
    expect(clientsMock).toHaveLength(1);

    fireEvent.change(screen.getByPlaceholderText('Ej. Cliente ACME'), { target: { value: 'Cliente Valido' } });
    fireEvent.change(screen.getByPlaceholderText('facturacion@cliente.com'), { target: { value: 'correo-invalido' } });
    fireEvent.click(screen.getByRole('button', { name: 'Crear cliente' }));

    expect(await screen.findByText('Escribe un email valido o deja el campo vacio.')).toBeInTheDocument();
    expect(clientsMock).toHaveLength(1);
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
      projectsTotal: 2,
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

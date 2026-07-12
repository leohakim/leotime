import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { DailySummaryPanel } from './dailySummaryUi';
import { translate } from './i18n';
import { ToastProvider } from './toast';

const t = (key: Parameters<typeof translate>[1]) => translate('es', key);

const sampleText =
  '12/7:\nResumen de hoy:\nPor la mañana avancé con RTVE — Participa: corrección de rutas API.\nHasta mañana team!';

function sampleRecord(status: 'draft' | 'approved' = 'draft') {
  return {
    date: '2026-07-12',
    clientId: '',
    projectId: '',
    status,
    draftText: sampleText,
    approvedText: status === 'approved' ? sampleText : '',
    manualNote: '',
    options: {
      date: '2026-07-12',
      clientId: '',
      projectId: '',
      includeClient: true,
      includeProject: true,
      includeClosing: true,
      billableOnly: false,
    },
    generationSource: 'template',
    generationCount: 1,
    contextJson: '',
    approvedAt: status === 'approved' ? '2026-07-12T10:00:00Z' : '',
    createdAt: '2026-07-12T09:00:00Z',
    updatedAt: '2026-07-12T09:00:00Z',
  };
}

function renderPanel() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <ToastProvider>
        <DailySummaryPanel clients={[]} projects={[]} t={t} />
      </ToastProvider>
    </QueryClientProvider>,
  );
}

describe('DailySummaryPanel', () => {
  beforeEach(() => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = String(input);
        const method = init?.method ?? 'GET';

        if (url.includes('/api/v1/daily-summaries/2026-07-12/generate') && method === 'POST') {
          return new Response(JSON.stringify(sampleRecord()), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          });
        }

        if (url.includes('/api/v1/daily-summaries/2026-07-12') && method === 'GET') {
          return new Response('not found', { status: 404 });
        }

        return new Response('not found', { status: 404 });
      }),
    );
    Object.assign(navigator, {
      clipboard: {
        writeText: vi.fn(async () => undefined),
      },
    });
  });

  it('generates a draft, allows editing, and copies slack text', async () => {
    renderPanel();

    fireEvent.click(screen.getByRole('button', { name: 'Generar resumen' }));

    const textarea = await screen.findByDisplayValue(/Resumen de hoy:/);
    expect(textarea).toBeInTheDocument();
    expect(screen.getByDisplayValue(/RTVE — Participa/)).toBeInTheDocument();

    fireEvent.change(textarea, { target: { value: `${sampleText}\nEditado a mano.` } });
    expect(screen.getByDisplayValue(/Editado a mano\./)).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Copiar para Slack' }));

    await waitFor(() => {
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith(expect.stringContaining('Editado a mano.'));
    });
  });
});

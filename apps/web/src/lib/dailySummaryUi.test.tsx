import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { DailySummaryPanel } from './dailySummaryUi';
import { translate } from './i18n';
import { ToastProvider } from './toast';

const t = (key: Parameters<typeof translate>[1]) => translate('es', key);

const sampleDate = '2026-07-12';
const sampleText =
  '12/7:\nResumen de hoy:\n- RTVE:\n    - corrección de rutas API.\nHasta mañana team!';

function sampleRecord(status: 'draft' | 'approved' = 'draft') {
  return {
    date: sampleDate,
    clientId: '',
    projectId: '',
    status,
    draftText: sampleText,
    approvedText: status === 'approved' ? sampleText : '',
    manualNote: '',
    options: {
      date: sampleDate,
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
        <DailySummaryPanel clients={[]} locale="es" projects={[]} t={t} />
      </ToastProvider>
    </QueryClientProvider>,
  );
}

describe('DailySummaryPanel', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date(`${sampleDate}T12:00:00Z`));

    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = String(input);
        const method = init?.method ?? 'GET';

        if (/(\/api\/v1\/daily-summaries)\?/.test(url) && method === 'GET') {
          return new Response(JSON.stringify({ items: [{ date: sampleDate, status: 'approved', generationSource: 'template', generationCount: 1, updatedAt: `${sampleDate}T10:00:00Z` }] }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          });
        }

        if (url.includes('/api/v1/daily-summaries/ai-usage') && method === 'GET') {
          return new Response(
            JSON.stringify({
              summary: {
                from: '2026-07-01',
                to: '2026-07-31',
                runCount: 0,
                inputTokens: 0,
                outputTokens: 0,
                cacheReadTokens: 0,
                cacheWriteTokens: 0,
                totalTokens: 0,
                estimatedCostUsd: 0,
                costPerMillionUsd: 2,
              },
              runs: [],
            }),
            {
              status: 200,
              headers: { 'Content-Type': 'application/json' },
            },
          );
        }

        if (url.includes(`/api/v1/daily-summaries/${sampleDate}/generate`) && method === 'POST') {
          return new Response(JSON.stringify(sampleRecord()), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          });
        }

        if (url.includes(`/api/v1/daily-summaries/${sampleDate}`) && method === 'GET') {
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

  afterEach(() => {
    vi.useRealTimers();
  });

  it('generates a draft, allows editing, and copies slack text', async () => {
    renderPanel();

    fireEvent.click(screen.getByRole('button', { name: 'Generar resumen' }));

    const textarea = await screen.findByDisplayValue(/Resumen de hoy:/);
    expect(textarea).toBeInTheDocument();
    expect(screen.getByDisplayValue(/- RTVE:/)).toBeInTheDocument();
    expect(screen.getByDisplayValue(/corrección de rutas API/)).toBeInTheDocument();

    fireEvent.change(textarea, { target: { value: `${sampleText}\nEditado a mano.` } });
    expect(screen.getByDisplayValue(/Editado a mano\./)).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Copiar para Slack' }));

    await waitFor(() => {
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith(expect.stringContaining('Editado a mano.'));
    });
  });
});

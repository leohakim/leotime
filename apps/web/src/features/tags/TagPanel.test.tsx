import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { TagPanel } from './TagPanel';
import { OfflineProvider } from '../../lib/offline/offlineContext';
import { ToastProvider } from '../../lib/toast';
import type { Tag } from '../../lib/api';

const tags: Tag[] = [
  {
    id: 'tag_1',
    name: 'Deep Work',
    color: '#2563eb',
    archivedAt: '',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
  },
  {
    id: 'tag_2',
    name: 'Admin',
    color: '#64748b',
    archivedAt: '2026-01-02T00:00:00Z',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-02T00:00:00Z',
  },
];

const t = (key: string) =>
  (
    ({
      tags: 'Tags',
      tagDirectory: 'Directorio de tags',
      tagPanelSubtitle: 'Taxonomia para clasificar entradas.',
      tagSummaryInventory: 'Activas: {active} / Archivadas: {archived}',
      activeTags: 'Tags activos',
      archivedTags: 'Tags archivados',
      loading: 'Cargando',
      synced: 'Sincronizado',
      newTag: 'Nuevo tag',
      noTags: 'Sin tags',
      createTag: 'Crear tag',
      tagFormCreate: 'Alta de tag',
      editingTag: 'Editando',
      tagFormEdit: 'Editar tag',
      name: 'Nombre',
      required: 'obligatorio',
      tagNamePlaceholder: 'Nombre del tag',
      tagColor: 'Color',
      tagColorPlaceholder: '#64748b',
      updateTag: 'Guardar tag',
      cleanForm: 'Limpiar',
      active: 'Activo',
      inactive: 'Inactivo',
      edit: 'Editar',
      archive: 'Archivar',
      reactivate: 'Reactivar',
      cancel: 'Cancelar',
    }) as Record<string, string>
  )[key] ?? key;

function renderTagPanel() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <ToastProvider>
        <OfflineProvider>
          <TagPanel isLoading={false} tags={tags} t={t} />
        </OfflineProvider>
      </ToastProvider>
    </QueryClientProvider>,
  );
}

describe('TagPanel', () => {
  beforeEach(() => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);
        if (url.endsWith('/api/v1/tags/summary')) {
          return new Response(JSON.stringify({ active: 3, archived: 1 }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          });
        }
        return new Response('not found', { status: 404 });
      }),
    );
  });

  it('shows tag summary inventory from the API', async () => {
    renderTagPanel();

    expect(await screen.findByText('Activas: 3 / Archivadas: 1')).toBeInTheDocument();
  });
});

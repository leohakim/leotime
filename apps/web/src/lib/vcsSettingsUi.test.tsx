import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { VCSSettingsPanel } from './vcsSettingsUi';

const t = (key: string) => key;

vi.mock('./api', () => ({
  fetchClients: vi.fn(async () => ({ clients: [{ id: 'cli_1', name: 'Osoigo' }] })),
  fetchProjects: vi.fn(async () => ({
    projects: [
      {
        id: 'prj_1',
        clientId: 'cli_1',
        clientName: 'Osoigo',
        name: 'ENACT',
        color: '#2563eb',
        defaultHourlyRateMinor: null,
        localRepoPath: '',
        gitRemoteUrl: '',
        cursorWorkspaceSlug: '',
        archivedAt: '',
        createdAt: '',
        updatedAt: '',
      },
      {
        id: 'prj_2',
        clientId: 'cli_other',
        clientName: 'Other',
        name: 'Other project',
        color: '#111',
        defaultHourlyRateMinor: null,
        localRepoPath: '',
        gitRemoteUrl: '',
        cursorWorkspaceSlug: '',
        archivedAt: '',
        createdAt: '',
        updatedAt: '',
      },
    ],
  })),
  fetchVCSConnections: vi.fn(async () => ({
    connections: [
      {
        id: 'vcs_1',
        provider: 'gitea',
        baseUrl: 'https://gitea.example.com',
        ownerIdentity: 'leo',
        defaultClientId: 'cli_1',
        tokenConfigured: true,
        enabled: true,
      },
    ],
  })),
  fetchVCSRepositories: vi.fn(async () => ({
    repositories: [
      {
        id: 'vcr_1',
        connectionId: 'vcs_1',
        owner: 'ENACT',
        name: 'backend',
        clientId: 'cli_1',
        effectiveClientId: 'cli_1',
        projectId: 'prj_1',
        enabled: true,
      },
      {
        id: 'vcr_2',
        connectionId: 'vcs_1',
        owner: 'ENACT',
        name: 'frontend',
        clientId: 'cli_1',
        effectiveClientId: 'cli_1',
        projectId: 'prj_1',
        enabled: true,
      },
    ],
  })),
  createVCSConnection: vi.fn(),
  updateVCSConnection: vi.fn(),
  deleteVCSConnection: vi.fn(async () => undefined),
  testVCSConnection: vi.fn(async () => ({ ok: true, message: 'connection ok' })),
  previewVCSConnectionContext: vi.fn(async () => ({
    ok: true,
    status: 'ready',
    date: '2026-07-27',
    repositoryCount: 2,
    counts: { commits: 1, pullRequests: 0, reviews: 0, issues: 0 },
  })),
  createVCSRepository: vi.fn(),
  updateVCSRepository: vi.fn(),
  deleteVCSRepository: vi.fn(async () => undefined),
  previewVCSRepositoryContext: vi.fn(async () => ({
    ok: true,
    status: 'ready',
    date: '2026-07-27',
    repositoryCount: 1,
    counts: { commits: 1, pullRequests: 0, reviews: 0, issues: 0 },
  })),
}));

vi.mock('./toast', () => ({
  useToast: () => ({ success: vi.fn(), error: vi.fn() }),
}));

vi.mock('./destructiveUi', () => ({
  confirmDestructiveAction: () => true,
}));

function renderPanel() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <VCSSettingsPanel t={t as never} />
    </QueryClientProvider>,
  );
}

describe('VCSSettingsPanel', () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it('lists repositories with shared project labels and edit actions', async () => {
    renderPanel();

    expect(await screen.findByText('ENACT/backend')).toBeInTheDocument();
    expect(screen.getByText('ENACT/frontend')).toBeInTheDocument();
    expect(screen.getAllByText(/· ENACT$/).length).toBeGreaterThanOrEqual(2);
    expect(screen.getAllByTitle('edit')).toHaveLength(3);
    expect(screen.getAllByTitle('vcsTestConnection')).toHaveLength(1);
    expect(screen.getAllByTitle('vcsTestContext')).toHaveLength(3);
  });

  it('filters project options to the effective client', async () => {
    renderPanel();
    await screen.findByText('ENACT/backend');

    fireEvent.change(screen.getByLabelText('vcsConnection'), { target: { value: 'vcs_1' } });

    const projectSelect = screen.getByLabelText('vcsProject');
    expect(within(projectSelect).getByRole('option', { name: 'ENACT' })).toBeInTheDocument();
    expect(within(projectSelect).queryByRole('option', { name: 'Other project' })).not.toBeInTheDocument();
  });

  it('loads a repository including its project into the form', async () => {
    renderPanel();
    const rowTitle = await screen.findByText('ENACT/backend');
    const row = rowTitle.closest('article');
    expect(row).not.toBeNull();

    fireEvent.click(within(row as HTMLElement).getByTitle('edit'));

    await waitFor(() => {
      expect(screen.getByPlaceholderText('ENACT')).toHaveValue('ENACT');
      expect(screen.getByPlaceholderText('backend')).toHaveValue('backend');
      expect(screen.getByLabelText('vcsProject')).toHaveValue('prj_1');
    });
  });
});

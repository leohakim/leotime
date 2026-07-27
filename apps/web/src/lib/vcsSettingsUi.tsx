import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { CircleAlert, GitBranch, Pencil, Plus, RadioTower, ScanSearch, Trash2, X } from 'lucide-react';
import { FormEvent, useMemo, useState } from 'react';
import {
  createVCSConnection,
  createVCSRepository,
  deleteVCSConnection,
  deleteVCSRepository,
  fetchClients,
  fetchProjects,
  fetchVCSConnections,
  fetchVCSRepositories,
  previewVCSConnectionContext,
  previewVCSRepositoryContext,
  testVCSConnection,
  updateVCSConnection,
  updateVCSRepository,
  type VCSConnection,
  type VCSContextPreview,
  type VCSRepository,
} from './api';
import { confirmDestructiveAction } from './destructiveUi';
import type { MessageKey } from './i18n';
import { useToast } from './toast';

type Translator = (key: MessageKey) => string;

type ConnectionForm = {
  baseUrl: string;
  ownerIdentity: string;
  token: string;
  defaultClientId: string;
};

type RepositoryForm = {
  connectionId: string;
  owner: string;
  name: string;
  clientId: string;
  projectId: string;
};

const emptyConnection = (): ConnectionForm => ({
  baseUrl: '',
  ownerIdentity: '',
  token: '',
  defaultClientId: '',
});

const emptyRepository = (): RepositoryForm => ({
  connectionId: '',
  owner: '',
  name: '',
  clientId: '',
  projectId: '',
});

export function VCSSettingsPanel({ t }: { t: Translator }) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const connectionsQuery = useQuery({ queryKey: ['vcs-connections'], queryFn: fetchVCSConnections, retry: false });
  const repositoriesQuery = useQuery({ queryKey: ['vcs-repositories'], queryFn: fetchVCSRepositories, retry: false });
  const clientsQuery = useQuery({ queryKey: ['clients'], queryFn: () => fetchClients() });
  const projectsQuery = useQuery({ queryKey: ['projects'], queryFn: () => fetchProjects() });

  const [connectionForm, setConnectionForm] = useState<ConnectionForm>(emptyConnection);
  const [repositoryForm, setRepositoryForm] = useState<RepositoryForm>(emptyRepository);
  const [editingConnectionId, setEditingConnectionId] = useState<string | null>(null);
  const [editingRepositoryId, setEditingRepositoryId] = useState<string | null>(null);
  const [tokenConfigured, setTokenConfigured] = useState(false);
  const [probeResult, setProbeResult] = useState('');

  const connections = connectionsQuery.data?.connections ?? [];
  const repositories = repositoriesQuery.data?.repositories ?? [];
  const clients = clientsQuery.data?.clients ?? [];
  const projects = projectsQuery.data?.projects ?? [];
  const clientNameById = new Map(clients.map((client) => [client.id, client.name]));
  const projectNameById = new Map(projects.map((project) => [project.id, project.name]));

  const effectiveClientId = useMemo(() => {
    if (repositoryForm.clientId) {
      return repositoryForm.clientId;
    }
    const connection = connections.find((item) => item.id === repositoryForm.connectionId);
    return connection?.defaultClientId ?? '';
  }, [connections, repositoryForm.clientId, repositoryForm.connectionId]);

  const projectsForClient = useMemo(
    () => projects.filter((project) => !project.archivedAt && project.clientId === effectiveClientId),
    [effectiveClientId, projects],
  );

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['vcs-connections'] });
    void queryClient.invalidateQueries({ queryKey: ['vcs-repositories'] });
  };

  const resetConnectionForm = () => {
    setEditingConnectionId(null);
    setTokenConfigured(false);
    setConnectionForm(emptyConnection());
  };

  const resetRepositoryForm = () => {
    setEditingRepositoryId(null);
    setRepositoryForm(emptyRepository());
  };

  const startEditConnection = (item: VCSConnection) => {
    setEditingConnectionId(item.id);
    setTokenConfigured(item.tokenConfigured);
    setConnectionForm({
      baseUrl: item.baseUrl,
      ownerIdentity: item.ownerIdentity,
      token: '',
      defaultClientId: item.defaultClientId || '',
    });
  };

  const startEditRepository = (item: VCSRepository) => {
    setEditingRepositoryId(item.id);
    setRepositoryForm({
      connectionId: item.connectionId,
      owner: item.owner,
      name: item.name,
      clientId: item.clientId || '',
      projectId: item.projectId || '',
    });
  };

  const updateRepositoryForm = (patch: Partial<RepositoryForm>) => {
    setRepositoryForm((current) => {
      const next = { ...current, ...patch };
      const nextEffectiveClientId = (() => {
        if (next.clientId) {
          return next.clientId;
        }
        const connection = connections.find((item) => item.id === next.connectionId);
        return connection?.defaultClientId ?? '';
      })();
      const allowed = projects.some(
        (project) =>
          !project.archivedAt && project.clientId === nextEffectiveClientId && project.id === next.projectId,
      );
      if (next.projectId && !allowed) {
        next.projectId = '';
      }
      return next;
    });
  };

  const connectionMutation = useMutation({
    mutationFn: async () => {
      const payload = {
        provider: 'gitea' as const,
        baseUrl: connectionForm.baseUrl.trim(),
        ownerIdentity: connectionForm.ownerIdentity.trim(),
        defaultClientId: connectionForm.defaultClientId || undefined,
        token: connectionForm.token.trim() || undefined,
      };
      if (editingConnectionId) {
        return updateVCSConnection(editingConnectionId, payload);
      }
      return createVCSConnection(payload);
    },
    onSuccess: (saved) => {
      resetConnectionForm();
      setRepositoryForm((current) => ({ ...current, connectionId: current.connectionId || saved.id }));
      invalidate();
      toast.success(t('vcsSaved'));
    },
    onError: () => toast.error(t('vcsSaveFailed')),
  });

  const repositoryMutation = useMutation({
    mutationFn: async () => {
      const payload = {
        connectionId: repositoryForm.connectionId,
        owner: repositoryForm.owner.trim(),
        name: repositoryForm.name.trim(),
        clientId: repositoryForm.clientId || undefined,
        projectId: repositoryForm.projectId || undefined,
      };
      if (editingRepositoryId) {
        return updateVCSRepository(editingRepositoryId, payload);
      }
      return createVCSRepository(payload);
    },
    onSuccess: () => {
      resetRepositoryForm();
      invalidate();
      toast.success(t('vcsSaved'));
    },
    onError: () => toast.error(t('vcsSaveFailed')),
  });

  const deleteConnectionMutation = useMutation({
    mutationFn: deleteVCSConnection,
    onSuccess: (_data, id) => {
      if (editingConnectionId === id) {
        resetConnectionForm();
      }
      invalidate();
      toast.success(t('vcsDeleted'));
    },
    onError: () => toast.error(t('vcsDeleteFailed')),
  });

  const deleteRepositoryMutation = useMutation({
    mutationFn: deleteVCSRepository,
    onSuccess: (_data, id) => {
      if (editingRepositoryId === id) {
        resetRepositoryForm();
      }
      invalidate();
      toast.success(t('vcsDeleted'));
    },
    onError: () => toast.error(t('vcsDeleteFailed')),
  });

  const formatPreview = (preview: VCSContextPreview) => {
    if (preview.status === 'not_configured' || preview.repositoryCount === 0) {
      return t('vcsTestContextNotConfigured');
    }
    const total =
      preview.counts.commits + preview.counts.pullRequests + preview.counts.reviews + preview.counts.issues;
    if (total === 0) {
      return t('vcsTestContextEmpty');
    }
    return t('vcsTestContextOk')
      .replace('{commits}', String(preview.counts.commits))
      .replace('{pulls}', String(preview.counts.pullRequests))
      .replace('{reviews}', String(preview.counts.reviews))
      .replace('{issues}', String(preview.counts.issues))
      .replace('{repos}', String(preview.repositoryCount));
  };

  const testConnectionMutation = useMutation({
    mutationFn: testVCSConnection,
    onSuccess: () => {
      const message = t('vcsTestConnectionOk');
      setProbeResult(message);
      toast.success(message);
    },
    onError: () => {
      setProbeResult(t('vcsTestConnectionFailed'));
      toast.error(t('vcsTestConnectionFailed'));
    },
  });

  const previewConnectionMutation = useMutation({
    mutationFn: (id: string) => previewVCSConnectionContext(id),
    onSuccess: (preview) => {
      const message = formatPreview(preview);
      setProbeResult(message);
      toast.success(message);
    },
    onError: () => {
      setProbeResult(t('vcsTestContextFailed'));
      toast.error(t('vcsTestContextFailed'));
    },
  });

  const previewRepositoryMutation = useMutation({
    mutationFn: (id: string) => previewVCSRepositoryContext(id),
    onSuccess: (preview) => {
      const message = formatPreview(preview);
      setProbeResult(message);
      toast.success(message);
    },
    onError: () => {
      setProbeResult(t('vcsTestContextFailed'));
      toast.error(t('vcsTestContextFailed'));
    },
  });

  const connectionLabel = (connectionId: string) => {
    const match = connections.find((item) => item.id === connectionId);
    return match?.baseUrl ?? connectionId;
  };

  const effectiveClientLabel = (item: VCSRepository) => {
    const id = item.clientId || item.effectiveClientId;
    if (!id) {
      return t('vcsInheritedClient');
    }
    return clientNameById.get(id) ?? id;
  };

  const projectLabel = (item: VCSRepository) => {
    if (!item.projectId) {
      return t('vcsNoProject');
    }
    return projectNameById.get(item.projectId) ?? item.projectId;
  };

  return (
    <section className="panel-section" id="vcs-settings">
      <div className="panel-section-heading">
        <span className="section-kicker">
          <GitBranch aria-hidden="true" />
          {t('vcsKicker')}
        </span>
        <h2>{t('vcsHeading')}</h2>
        <p>{t('vcsSubtitle')}</p>
      </div>

      {connectionsQuery.isError || repositoriesQuery.isError ? (
        <div className="form-alert" role="alert">
          <CircleAlert aria-hidden="true" />
          {t('vcsLoadFailed')}
        </div>
      ) : null}

      <div className="clients-heading">
        <div className="section-title-group">
          <h3>{t('vcsConnections')}</h3>
        </div>
        <button className="secondary-button" type="button" onClick={resetConnectionForm}>
          <Plus aria-hidden="true" />
          {t('vcsNewConnection')}
        </button>
      </div>

      <form
        className="profile-form"
        onSubmit={(event: FormEvent) => {
          event.preventDefault();
          connectionMutation.mutate();
        }}
      >
        <div className="client-form-grid">
          <label className="form-field">
            <span>{t('vcsBaseUrl')}</span>
            <input
              required
              type="url"
              value={connectionForm.baseUrl}
              onChange={(event) => setConnectionForm({ ...connectionForm, baseUrl: event.target.value })}
              placeholder="https://gitea.example.com"
            />
          </label>
          <label className="form-field">
            <span>{t('vcsOwnerIdentity')}</span>
            <input
              value={connectionForm.ownerIdentity}
              onChange={(event) => setConnectionForm({ ...connectionForm, ownerIdentity: event.target.value })}
            />
          </label>
          <label className="form-field">
            <span>{t('vcsToken')}</span>
            <input
              required={!editingConnectionId}
              type="password"
              autoComplete="off"
              value={connectionForm.token}
              onChange={(event) => setConnectionForm({ ...connectionForm, token: event.target.value })}
              placeholder={editingConnectionId && tokenConfigured ? t('vcsConfiguredToken') : undefined}
            />
            {editingConnectionId ? <span className="field-hint">{t('vcsTokenKeepHint')}</span> : null}
          </label>
          <ClientSelect
            label={t('vcsDefaultClient')}
            clients={clients}
            value={connectionForm.defaultClientId}
            onChange={(defaultClientId) => setConnectionForm({ ...connectionForm, defaultClientId })}
          />
        </div>
        <div className="profile-form-actions">
          {editingConnectionId ? (
            <button className="secondary-button" type="button" onClick={resetConnectionForm}>
              <X aria-hidden="true" />
              {t('cancel')}
            </button>
          ) : null}
          <button disabled={connectionMutation.isPending} type="submit">
            {editingConnectionId ? t('vcsSaveConnection') : t('vcsAddConnection')}
          </button>
        </div>
      </form>

      <div className="client-list" aria-busy={connectionsQuery.isLoading}>
        {connectionsQuery.isLoading ? <p>{t('loading')}</p> : null}
        {!connectionsQuery.isLoading && connections.length === 0 ? (
          <div className="empty-state">
            <GitBranch aria-hidden="true" />
            <p>{t('vcsNoConnections')}</p>
          </div>
        ) : null}
        {connections.map((item) => (
          <article
            className={editingConnectionId === item.id ? 'client-row selected' : 'client-row'}
            key={item.id}
          >
            <div className="client-row-main">
              <div className="client-row-copy">
                <div className="client-row-title">
                  <strong>{item.baseUrl}</strong>
                  <span className={item.tokenConfigured ? 'status-pill' : 'status-pill warning-pill'}>
                    {item.tokenConfigured ? t('vcsConfiguredToken') : t('vcsMissingToken')}
                  </span>
                </div>
                <span className="client-contact">
                  {item.ownerIdentity || '—'}
                  {item.defaultClientId
                    ? ` · ${clientNameById.get(item.defaultClientId) ?? item.defaultClientId}`
                    : ''}
                </span>
              </div>
            </div>
            <div className="client-row-actions">
              <button
                className="secondary-button icon-button"
                type="button"
                disabled={!item.tokenConfigured || testConnectionMutation.isPending}
                onClick={() => testConnectionMutation.mutate(item.id)}
                title={t('vcsTestConnection')}
              >
                <RadioTower aria-hidden="true" />
              </button>
              <button
                className="secondary-button icon-button"
                type="button"
                disabled={!item.tokenConfigured || previewConnectionMutation.isPending}
                onClick={() => previewConnectionMutation.mutate(item.id)}
                title={t('vcsTestContext')}
              >
                <ScanSearch aria-hidden="true" />
              </button>
              <button
                className="secondary-button icon-button"
                type="button"
                onClick={() => startEditConnection(item)}
                title={t('edit')}
              >
                <Pencil aria-hidden="true" />
              </button>
              <button
                aria-label={t('delete')}
                className="secondary-button icon-button danger-button"
                type="button"
                onClick={() => {
                  if (confirmDestructiveAction(t('vcsDeleteConnectionConfirm'))) {
                    deleteConnectionMutation.mutate(item.id);
                  }
                }}
                title={t('delete')}
              >
                <Trash2 aria-hidden="true" />
              </button>
            </div>
          </article>
        ))}
      </div>

      <div className="clients-heading">
        <div className="section-title-group">
          <h3>{t('vcsRepositories')}</h3>
        </div>
        <button className="secondary-button" type="button" onClick={resetRepositoryForm}>
          <Plus aria-hidden="true" />
          {t('vcsNewRepository')}
        </button>
      </div>

      <form
        className="profile-form"
        onSubmit={(event: FormEvent) => {
          event.preventDefault();
          repositoryMutation.mutate();
        }}
      >
        <div className="client-form-grid">
          <label className="form-field">
            <span>{t('vcsConnection')}</span>
            <select
              required
              value={repositoryForm.connectionId}
              onChange={(event) => updateRepositoryForm({ connectionId: event.target.value })}
            >
              <option value="" />
              {connections.map((item) => (
                <option key={item.id} value={item.id}>
                  {item.baseUrl}
                </option>
              ))}
            </select>
          </label>
          <label className="form-field">
            <span>{t('vcsRepoOwner')}</span>
            <input
              required
              value={repositoryForm.owner}
              onChange={(event) => updateRepositoryForm({ owner: event.target.value })}
              placeholder="ENACT"
            />
          </label>
          <label className="form-field">
            <span>{t('vcsRepoName')}</span>
            <input
              required
              value={repositoryForm.name}
              onChange={(event) => updateRepositoryForm({ name: event.target.value })}
              placeholder="backend"
            />
            <span className="field-hint">{t('vcsRepoPathHint')}</span>
          </label>
          <ClientSelect
            label={t('vcsDefaultClient')}
            clients={clients}
            value={repositoryForm.clientId}
            onChange={(clientId) => updateRepositoryForm({ clientId })}
          />
          <label className="form-field">
            <span>{t('vcsProject')}</span>
            <select
              value={repositoryForm.projectId}
              disabled={!effectiveClientId || projectsForClient.length === 0}
              onChange={(event) => updateRepositoryForm({ projectId: event.target.value })}
            >
              <option value="" />
              {projectsForClient.map((project) => (
                <option key={project.id} value={project.id}>
                  {project.name}
                </option>
              ))}
            </select>
            {effectiveClientId && projectsForClient.length === 0 ? (
              <span className="field-hint">{t('vcsNoProjectsForClient')}</span>
            ) : null}
          </label>
        </div>
        <div className="profile-form-actions">
          {editingRepositoryId ? (
            <button className="secondary-button" type="button" onClick={resetRepositoryForm}>
              <X aria-hidden="true" />
              {t('cancel')}
            </button>
          ) : null}
          <button disabled={repositoryMutation.isPending || connections.length === 0} type="submit">
            {editingRepositoryId ? t('vcsSaveRepository') : t('vcsAddRepository')}
          </button>
        </div>
      </form>

      <div className="client-list" aria-busy={repositoriesQuery.isLoading}>
        {repositoriesQuery.isLoading ? <p>{t('loading')}</p> : null}
        {!repositoriesQuery.isLoading && repositories.length === 0 ? (
          <div className="empty-state">
            <GitBranch aria-hidden="true" />
            <p>{t('vcsNoRepositories')}</p>
          </div>
        ) : null}
        {repositories.map((item) => (
          <article
            className={editingRepositoryId === item.id ? 'client-row selected' : 'client-row'}
            key={item.id}
          >
            <div className="client-row-main">
              <div className="client-row-copy">
                <div className="client-row-title">
                  <strong>
                    {item.owner}/{item.name}
                  </strong>
                </div>
                <span className="client-contact">
                  {connectionLabel(item.connectionId)} · {effectiveClientLabel(item)} · {projectLabel(item)}
                </span>
              </div>
            </div>
            <div className="client-row-actions">
              <button
                className="secondary-button icon-button"
                type="button"
                disabled={previewRepositoryMutation.isPending}
                onClick={() => previewRepositoryMutation.mutate(item.id)}
                title={t('vcsTestContext')}
              >
                <ScanSearch aria-hidden="true" />
              </button>
              <button
                className="secondary-button icon-button"
                type="button"
                onClick={() => startEditRepository(item)}
                title={t('edit')}
              >
                <Pencil aria-hidden="true" />
              </button>
              <button
                aria-label={t('delete')}
                className="secondary-button icon-button danger-button"
                type="button"
                onClick={() => {
                  if (confirmDestructiveAction(t('vcsDeleteRepositoryConfirm'))) {
                    deleteRepositoryMutation.mutate(item.id);
                  }
                }}
                title={t('delete')}
              >
                <Trash2 aria-hidden="true" />
              </button>
            </div>
          </article>
        ))}
      </div>

      {probeResult ? (
        <div className="form-alert" role="status">
          <ScanSearch aria-hidden="true" />
          {probeResult}
        </div>
      ) : null}
    </section>
  );
}

function ClientSelect({
  label,
  clients,
  value,
  onChange,
}: {
  label: string;
  clients: Array<{ id: string; name: string }>;
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <label className="form-field">
      <span>{label}</span>
      <select value={value} onChange={(event) => onChange(event.target.value)}>
        <option value="" />
        {clients.map((client) => (
          <option key={client.id} value={client.id}>
            {client.name}
          </option>
        ))}
      </select>
    </label>
  );
}

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { BadgeDollarSign, Building2, CircleAlert, CircleCheck, FolderKanban, Pencil, Plus, RotateCcw, Save, Trash2, X } from 'lucide-react';
import { FormEvent, useState } from 'react';
import {
  archiveProject,
  restoreProject,
  updateProject,
  type Client,
  type Project,
  type ProjectInput,
} from '../../lib/api';
import { DirectoryInactiveHeading, FieldError, fieldClass, formatMinor, formatRateInput, hasErrors, rateToMinor } from '../../lib/crudFormUi';
import { confirmDestructiveAction } from '../../lib/destructiveUi';
import { patchProjectsCache, refreshOverviewIfOnline } from '../../lib/offline/cache';
import { useOfflineStatus } from '../../lib/offline/offlineContext';
import { createProject, isLocalId } from '../../lib/offline/mutations';
import type { Translator } from '../../lib/translator';
import { toastMutationSuccess, useToast } from '../../lib/toast';

type ProjectFormState = Omit<ProjectInput, 'defaultHourlyRateMinor'> & {
  hourlyRate: string;
  active: boolean;
};

type ProjectFormErrors = Partial<Record<keyof ProjectFormState | 'form', string>>;

const emptyProjectForm: ProjectFormState = {
  clientId: '',
  name: '',
  color: '#2563eb',
  hourlyRate: '',
  active: true,
  localRepoPath: '',
  gitRemoteUrl: '',
  cursorWorkspaceSlug: '',
};

export function ProjectPanel({
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
  const toast = useToast();
  const { refreshPendingCount } = useOfflineStatus();
  const [editingProjectId, setEditingProjectId] = useState<string | null>(null);
  const [form, setForm] = useState<ProjectFormState>(emptyProjectForm);
  const [errors, setErrors] = useState<ProjectFormErrors>({});

  const createMutation = useMutation({
    mutationFn: (input: ProjectInput) => createProject(input, { clients }),
    onSuccess: (project) => {
      setForm(emptyProjectForm);
      setErrors({});
      patchProjectsCache(queryClient, project);
      void refreshPendingCount();
      void refreshOverviewIfOnline(queryClient);
      if (!isLocalId(project.id)) {
        queryClient.invalidateQueries({ queryKey: ['projects'] });
      }
      toastMutationSuccess(toast, t, 'projectCreated', project.id);
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('projectSaveFailed') }));
      toast.error(t('projectSaveFailed'));
    },
  });

  const updateMutation = useMutation({
    mutationFn: async ({
      projectId,
      input,
      active,
      wasActive,
    }: {
      projectId: string;
      input: ProjectInput;
      active: boolean;
      wasActive: boolean;
    }) => {
      const updated = await updateProject(projectId, input);
      if (active && !wasActive) {
        await restoreProject(projectId);
      } else if (!active && wasActive) {
        await archiveProject(projectId);
      }
      return updated;
    },
    onSuccess: () => {
      setEditingProjectId(null);
      setForm(emptyProjectForm);
      setErrors({});
      queryClient.invalidateQueries({ queryKey: ['projects'] });
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] });
      toast.success(t('projectUpdated'));
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('projectSaveFailed') }));
      toast.error(t('projectSaveFailed'));
    },
  });

  const archiveMutation = useMutation({
    mutationFn: archiveProject,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] });
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] });
      toast.success(t('projectArchived'));
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('projectArchiveFailed') }));
      toast.error(t('projectArchiveFailed'));
    },
  });

  const restoreMutation = useMutation({
    mutationFn: restoreProject,
    onSuccess: () => {
      setEditingProjectId(null);
      setForm(emptyProjectForm);
      setErrors({});
      queryClient.invalidateQueries({ queryKey: ['projects'] });
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] });
      toast.success(t('projectRestored'));
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('projectSaveFailed') }));
      toast.error(t('projectSaveFailed'));
    },
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
      const project = projects.find((item) => item.id === editingProjectId);
      updateMutation.mutate({
        projectId: editingProjectId,
        input,
        active: form.active,
        wasActive: !project?.archivedAt,
      });
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
      active: !project.archivedAt,
      localRepoPath: project.localRepoPath || '',
      gitRemoteUrl: project.gitRemoteUrl || '',
      cursorWorkspaceSlug: project.cursorWorkspaceSlug || '',
    });
  }

  function cancelEditing() {
    setEditingProjectId(null);
    setForm(emptyProjectForm);
    setErrors({});
  }

  const isSaving = createMutation.isPending || updateMutation.isPending;
  const activeProjectCount = projects.filter((project) => !project.archivedAt).length;
  const activeProjects = projects.filter((project) => !project.archivedAt);
  const inactiveProjects = projects.filter((project) => project.archivedAt);

  function renderProjectRow(project: Project, isActive: boolean) {
    return (
      <article
        className={editingProjectId === project.id ? 'client-row selected' : isActive ? 'client-row' : 'client-row archived'}
        key={project.id}
      >
        <div className="client-row-main">
          <div className="project-color-dot" style={{ backgroundColor: project.color }} aria-hidden="true" />
          <div className="client-row-copy">
            <div className="client-row-title">
              <strong>{project.name}</strong>
              <span className={isActive ? 'status-pill' : 'status-pill warning-pill'}>
                {isActive ? <CircleCheck aria-hidden="true" /> : null}
                {isActive ? t('active') : t('inactive')}
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
          {isActive ? (
            <button
              aria-label={t('archive')}
              className="secondary-button icon-button danger-button"
              type="button"
              onClick={() => {
                if (confirmDestructiveAction(t('archiveProjectConfirm'))) {
                  archiveMutation.mutate(project.id);
                }
              }}
              title={t('archive')}
            >
              <Trash2 aria-hidden="true" />
            </button>
          ) : (
            <button
              className="secondary-button icon-button"
              type="button"
              onClick={() => restoreMutation.mutate(project.id)}
              title={t('reactivate')}
            >
              <RotateCcw aria-hidden="true" />
            </button>
          )}
        </div>
      </article>
    );
  }

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
              <strong>{activeProjectCount}</strong>
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
            {activeProjects.map((project) => renderProjectRow(project, true))}
          </div>

          <DirectoryInactiveHeading count={inactiveProjects.length} t={t} />
          {inactiveProjects.length > 0 ? (
            <div className="client-list client-list-inactive" aria-label={t('inactiveDirectory')}>
              {inactiveProjects.map((project) => renderProjectRow(project, false))}
            </div>
          ) : null}
        </div>

        <form className="client-editor" noValidate onSubmit={submitProject}>
          <div className="editor-header">
            <div>
              <span>{editingProjectId ? t('editingProject') : t('newProject')}</span>
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
                {clients.filter((client) => !client.archivedAt).map((client) => (
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

            {editingProjectId ? (
              <label className="client-active-field" htmlFor="project-active">
                <input
                  checked={form.active}
                  id="project-active"
                  onChange={(event) => updateField('active', event.target.checked)}
                  type="checkbox"
                />
                <span>{t('projectActive')}</span>
              </label>
            ) : null}

            <label className="form-field" htmlFor="project-local-repo">
              <span>{t('projectLocalRepoPath')}</span>
              <input
                id="project-local-repo"
                onChange={(event) => updateField('localRepoPath', event.target.value)}
                placeholder="/Users/you/dev/my-project"
                value={form.localRepoPath}
              />
            </label>

            <label className="form-field" htmlFor="project-git-remote">
              <span>{t('projectGitRemoteUrl')}</span>
              <input
                id="project-git-remote"
                onChange={(event) => updateField('gitRemoteUrl', event.target.value)}
                placeholder="git@github.com:you/my-project.git"
                value={form.gitRemoteUrl}
              />
            </label>

            <label className="form-field" htmlFor="project-cursor-slug">
              <span>{t('projectCursorWorkspaceSlug')}</span>
              <input
                id="project-cursor-slug"
                onChange={(event) => updateField('cursorWorkspaceSlug', event.target.value)}
                placeholder="Users-you-dev-my-project"
                value={form.cursorWorkspaceSlug}
              />
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

function projectFormToInput(form: ProjectFormState): ProjectInput {
  return {
    clientId: form.clientId,
    name: form.name.trim(),
    color: form.color.trim() || '#2563eb',
    defaultHourlyRateMinor: form.hourlyRate.trim() ? rateToMinor(form.hourlyRate) : null,
    localRepoPath: (form.localRepoPath ?? '').trim(),
    gitRemoteUrl: (form.gitRemoteUrl ?? '').trim(),
    cursorWorkspaceSlug: (form.cursorWorkspaceSlug ?? '').trim(),
  };
}

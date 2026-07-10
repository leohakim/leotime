import { useMutation, useQueryClient } from '@tanstack/react-query';
import { CircleAlert, CircleCheck, DollarSign, ListTodo, Pencil, Plus, RotateCcw, Save, Trash2, X } from 'lucide-react';
import { FormEvent, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  archiveTask,
  isApiError,
  mapApiFieldErrors,
  restoreTask,
  updateTask,
  type Project,
  type Task,
  type TaskInput,
} from '../../lib/api';
import { DirectoryInactiveHeading, FieldError, fieldClass, hasErrors, validateProjectRequired } from '../../lib/crudFormUi';
import { patchTasksCache, refreshOverviewIfOnline } from '../../lib/offline/cache';
import { useOfflineStatus } from '../../lib/offline/offlineContext';
import { createTask, isLocalId } from '../../lib/offline/mutations';
import { ProjectBadge } from '../../lib/projectBadgeUi';
import { sortTasksByNewest } from '../../lib/taskSort';
import type { Translator } from '../../lib/translator';
import { toastMutationSuccess, useToast } from '../../lib/toast';

type TaskFormState = {
  projectId: string;
  name: string;
  billable: boolean;
  active: boolean;
};

type TaskFormErrors = Partial<Record<keyof TaskFormState | 'form', string>>;

const emptyTaskForm: TaskFormState = {
  projectId: '',
  name: '',
  billable: true,
  active: true,
};

export function TaskPanel({
  isLoading,
  projects,
  taskProjectRequired,
  tasks,
  t,
}: {
  isLoading: boolean;
  projects: Project[];
  taskProjectRequired: boolean;
  tasks: Task[];
  t: Translator;
}) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const { refreshPendingCount } = useOfflineStatus();
  const [editingTaskId, setEditingTaskId] = useState<string | null>(null);
  const [form, setForm] = useState<TaskFormState>(emptyTaskForm);
  const [errors, setErrors] = useState<TaskFormErrors>({});
  const sortedTasks = useMemo(() => sortTasksByNewest(tasks), [tasks]);

  function applyTaskSaveError(error: unknown) {
    const fieldErrors = mapApiFieldErrors<keyof TaskFormState>(error, {
      projectId: 'projectId',
      name: 'name',
    });
    setErrors((current) => ({
      ...current,
      ...fieldErrors,
      form: Object.keys(fieldErrors).length > 0 ? undefined : isApiError(error) ? error.message : t('taskSaveFailed'),
    }));
    toast.error(isApiError(error) ? error.message : t('taskSaveFailed'));
  }

  const createMutation = useMutation({
    mutationFn: (input: TaskInput) => createTask(input, { projects }),
    onSuccess: (created) => {
      patchTasksCache(queryClient, created);
      setForm(emptyTaskForm);
      setErrors({});
      void refreshPendingCount();
      void refreshOverviewIfOnline(queryClient);
      if (!isLocalId(created.id)) {
        queryClient.invalidateQueries({ queryKey: ['tasks'] });
      }
      toastMutationSuccess(toast, t, 'taskCreated', created.id);
    },
    onError: applyTaskSaveError,
  });

  const updateMutation = useMutation({
    mutationFn: async ({
      taskId,
      input,
      active,
      wasActive,
    }: {
      taskId: string;
      input: TaskInput;
      active: boolean;
      wasActive: boolean;
    }) => {
      const updated = await updateTask(taskId, input);
      if (active && !wasActive) {
        await restoreTask(taskId);
      } else if (!active && wasActive) {
        await archiveTask(taskId);
      }
      return updated;
    },
    onSuccess: () => {
      setEditingTaskId(null);
      setForm(emptyTaskForm);
      setErrors({});
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] });
      toast.success(t('taskUpdated'));
    },
    onError: applyTaskSaveError,
  });

  const inlineUpdateMutation = useMutation({
    mutationFn: ({ taskId, input }: { taskId: string; input: TaskInput }) => updateTask(taskId, input),
    onSuccess: (updated) => {
      queryClient.setQueryData(['tasks'], (current: { tasks: Task[] } | undefined) => {
        if (!current) {
          return current;
        }
        return {
          tasks: sortTasksByNewest(current.tasks.map((item) => (item.id === updated.id ? updated : item))),
        };
      });
      if (editingTaskId === updated.id) {
        setForm((current) => ({ ...current, name: updated.name }));
      }
    },
    onError: () => toast.error(t('taskSaveFailed')),
  });

  const saveInlineTaskName = useCallback(
    (taskId: string, name: string) => {
      const task = tasks.find((item) => item.id === taskId);
      if (!task) {
        return;
      }
      inlineUpdateMutation.mutate({
        taskId,
        input: {
          projectId: task.projectId,
          name,
          billable: task.billable,
        },
      });
    },
    [inlineUpdateMutation, tasks],
  );

  const archiveMutation = useMutation({
    mutationFn: archiveTask,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] });
      toast.success(t('taskArchived'));
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('taskArchiveFailed') }));
      toast.error(t('taskArchiveFailed'));
    },
  });

  const restoreMutation = useMutation({
    mutationFn: restoreTask,
    onSuccess: () => {
      setEditingTaskId(null);
      setForm(emptyTaskForm);
      setErrors({});
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] });
      toast.success(t('taskRestored'));
    },
    onError: () => {
      setErrors((current) => ({ ...current, form: t('taskSaveFailed') }));
      toast.error(t('taskSaveFailed'));
    },
  });

  function submitTask(event: FormEvent) {
    event.preventDefault();
    const validation = validateTaskForm(form, t, taskProjectRequired);
    setErrors(validation);
    if (hasErrors(validation)) {
      return;
    }

    const input = taskFormToInput(form);
    if (editingTaskId) {
      const task = tasks.find((item) => item.id === editingTaskId);
      updateMutation.mutate({
        taskId: editingTaskId,
        input,
        active: form.active,
        wasActive: !task?.archivedAt,
      });
      return;
    }
    createMutation.mutate(input);
  }

  function updateField<K extends keyof TaskFormState>(field: K, value: TaskFormState[K]) {
    const next = { ...form, [field]: value };
    setForm(next);
    if (hasErrors(errors)) {
      setErrors(validateTaskForm(next, t, taskProjectRequired));
    }
  }

  function startEditing(task: Task) {
    setEditingTaskId(task.id);
    setErrors({});
    setForm({
      projectId: task.projectId,
      name: task.name,
      billable: task.billable,
      active: !task.archivedAt,
    });
  }

  function cancelEditing() {
    setEditingTaskId(null);
    setForm(emptyTaskForm);
    setErrors({});
  }

  const isSaving = createMutation.isPending || updateMutation.isPending;
  const activeTaskCount = sortedTasks.filter((task) => !task.archivedAt).length;
  const activeTasks = sortedTasks.filter((task) => !task.archivedAt);
  const inactiveTasks = sortedTasks.filter((task) => task.archivedAt);

  function renderTaskRow(task: Task, isActive: boolean) {
    return (
      <article
        className={editingTaskId === task.id ? 'client-row selected' : isActive ? 'client-row' : 'client-row archived'}
        key={task.id}
      >
        <div className="client-row-main">
          <div className="client-row-copy">
            <div className="client-row-title">
              <TaskInlineNameInput onSave={saveInlineTaskName} t={t} task={task} />
              <span className={isActive ? 'status-pill' : 'status-pill warning-pill'}>
                {isActive ? <CircleCheck aria-hidden="true" /> : null}
                {isActive ? t('active') : t('inactive')}
              </span>
            </div>
            <ProjectBadge color={task.projectColor} emptyLabel={t('taskProjectOptional')} name={task.projectName} />
          </div>
        </div>
        <div className="client-row-meta">
          <span className={task.billable ? 'rate-pill billable-on' : 'rate-pill'}>
            <DollarSign aria-hidden="true" />
            {task.billable ? t('billable') : t('nonBillable')}
          </span>
        </div>
        <div className="client-row-actions">
          <button
            className="secondary-button icon-button"
            type="button"
            onClick={() => startEditing(task)}
            title={t('edit')}
          >
            <Pencil aria-hidden="true" />
          </button>
          {isActive ? (
            <button
              className="secondary-button icon-button danger-button"
              type="button"
              onClick={() => archiveMutation.mutate(task.id)}
              title={t('archive')}
            >
              <Trash2 aria-hidden="true" />
            </button>
          ) : (
            <button
              className="secondary-button icon-button"
              type="button"
              onClick={() => restoreMutation.mutate(task.id)}
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
    <section className="clients-section tasks-section" id="tasks" aria-labelledby="tasks-title">
      <div className="clients-heading">
        <div className="section-title-group">
          <span className="section-kicker">
            <ListTodo aria-hidden="true" />
            {t('tasks')}
          </span>
          <h2 id="tasks-title">{t('taskDirectory')}</h2>
          <p>{t('taskPanelSubtitle')}</p>
        </div>
        <button className="secondary-button" type="button" onClick={cancelEditing}>
          <Plus aria-hidden="true" />
          {t('newTask')}
        </button>
      </div>

      <div className="clients-workbench">
        <div className="client-directory">
          <div className="directory-toolbar">
            <div>
              <span>{t('activeTasks')}</span>
              <strong>{activeTaskCount}</strong>
            </div>
            {isLoading ? (
              <span className="sync-pill">{t('loading')}</span>
            ) : (
              <span className="sync-pill">{t('synced')}</span>
            )}
          </div>

          <div className="client-list" aria-busy={isLoading}>
            {sortedTasks.length === 0 ? (
              <div className="empty-state">
                <ListTodo aria-hidden="true" />
                <p>{t('noTasks')}</p>
              </div>
            ) : null}
            {activeTasks.map((task) => renderTaskRow(task, true))}
          </div>

          <DirectoryInactiveHeading count={inactiveTasks.length} t={t} />
          {inactiveTasks.length > 0 ? (
            <div className="client-list client-list-inactive" aria-label={t('inactiveDirectory')}>
              {inactiveTasks.map((task) => renderTaskRow(task, false))}
            </div>
          ) : null}
        </div>

        <form className="client-editor" noValidate onSubmit={submitTask}>
          <div className="editor-header">
            <div>
              <span>{editingTaskId ? t('editingTask') : t('newTask')}</span>
              <h3>{editingTaskId ? t('taskFormEdit') : t('taskFormCreate')}</h3>
            </div>
            {editingTaskId ? (
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
            <label className={fieldClass(errors.name)} htmlFor="task-name">
              <span>
                {t('name')} <em>{t('required')}</em>
              </span>
              <input
                aria-describedby={errors.name ? 'task-name-error' : undefined}
                aria-invalid={Boolean(errors.name)}
                id="task-name"
                onChange={(event) => updateField('name', event.target.value)}
                placeholder={t('taskNamePlaceholder')}
                value={form.name}
              />
              <FieldError id="task-name-error" message={errors.name} />
            </label>

            <label className="form-field" htmlFor="task-project">
              <span>{t('taskProject')}</span>
              <select
                id="task-project"
                onChange={(event) => updateField('projectId', event.target.value)}
                value={form.projectId}
              >
                <option value="">{t('taskProjectOptional')}</option>
                {projects.filter((project) => !project.archivedAt).map((project) => (
                  <option key={project.id} value={project.id}>
                    {project.name}
                  </option>
                ))}
              </select>
            </label>

            <label className="form-field checkbox-field" htmlFor="task-billable">
              <span>{t('billable')}</span>
              <input
                checked={form.billable}
                id="task-billable"
                onChange={(event) => updateField('billable', event.target.checked)}
                type="checkbox"
              />
            </label>

            {editingTaskId ? (
              <label className="client-active-field" htmlFor="task-active">
                <input
                  checked={form.active}
                  id="task-active"
                  onChange={(event) => updateField('active', event.target.checked)}
                  type="checkbox"
                />
                <span>{t('taskActive')}</span>
              </label>
            ) : null}
          </div>

          <div className="client-form-actions">
            <button type="submit" disabled={isSaving}>
              {editingTaskId ? <Save aria-hidden="true" /> : <Plus aria-hidden="true" />}
              {editingTaskId ? t('updateTask') : t('createTask')}
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

function TaskInlineNameInput({
  onSave,
  t,
  task,
}: {
  onSave: (taskId: string, name: string) => void;
  t: Translator;
  task: Task;
}) {
  const [liveName, setLiveName] = useState(task.name);
  const [inlineError, setInlineError] = useState('');
  const skipSaveRef = useRef(true);
  const taskRef = useRef(task);
  taskRef.current = task;

  useEffect(() => {
    skipSaveRef.current = true;
    setLiveName(task.name);
    setInlineError('');
  }, [task.id]);

  useEffect(() => {
    const trimmed = liveName.trim();
    if (skipSaveRef.current) {
      skipSaveRef.current = false;
      return;
    }
    if (trimmed === taskRef.current.name) {
      setInlineError('');
      return;
    }
    if (!trimmed) {
      setInlineError(t('taskNameRequired'));
      return;
    }
    if (trimmed.length < 2) {
      setInlineError(t('taskNameTooShort'));
      return;
    }

    setInlineError('');
    const handle = window.setTimeout(() => {
      onSave(taskRef.current.id, trimmed);
    }, 400);

    return () => window.clearTimeout(handle);
  }, [liveName, onSave, t]);

  function handleBlur() {
    const trimmed = liveName.trim();
    if (!trimmed || trimmed.length < 2) {
      setLiveName(taskRef.current.name);
      setInlineError('');
    }
  }

  return (
    <label className="client-row-inline-field">
      <span className="visually-hidden">{t('taskName')}</span>
      <input
        aria-describedby={inlineError ? `task-inline-error-${task.id}` : undefined}
        aria-invalid={Boolean(inlineError)}
        aria-label={`${t('taskName')}: ${task.name}`}
        className={inlineError ? 'client-row-inline-input invalid' : 'client-row-inline-input'}
        onBlur={handleBlur}
        onChange={(event) => setLiveName(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === 'Enter') {
            event.currentTarget.blur();
          }
        }}
        value={liveName}
      />
      {inlineError ? (
        <span className="client-row-inline-error" id={`task-inline-error-${task.id}`} role="alert">
          {inlineError}
        </span>
      ) : null}
    </label>
  );
}

function validateTaskForm(form: TaskFormState, t: Translator, taskProjectRequired: boolean): TaskFormErrors {
  const errors: TaskFormErrors = {};
  const name = form.name.trim();

  if (!name) {
    errors.name = t('taskNameRequired');
  } else if (name.length < 2) {
    errors.name = t('taskNameTooShort');
  }

  const projectError = validateProjectRequired(form.projectId, taskProjectRequired, t);
  if (projectError) {
    errors.projectId = projectError;
  }

  return errors;
}

function taskFormToInput(form: TaskFormState): TaskInput {
  return {
    projectId: form.projectId,
    name: form.name.trim(),
    billable: form.billable,
  };
}

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { CalendarDays, CircleAlert, Clock3, DollarSign, Pencil, Plus, Save, Tag, Trash2, X } from 'lucide-react';
import { FormEvent, useEffect, useMemo, useRef, useState } from 'react';
import {
  createTimeEntry,
  deleteTimeEntry,
  updateTimeEntry,
  type Client,
  type Locale,
  type Project,
  type Tag as TagRecord,
  type Task,
  type TimeEntry,
  type TimeEntryInput,
} from './api';
import type { MessageKey } from './i18n';

export type Translator = (key: MessageKey) => string;

export function scrollToManualEntryForm() {
  document.getElementById('manual-time-entry')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
}

export function TimeEntriesList({
  entries,
  isLoading,
  locale,
  projects,
  tasks,
  t,
}: {
  entries: TimeEntry[];
  isLoading: boolean;
  locale: Locale;
  projects: Project[];
  tasks: Task[];
  t: Translator;
}) {
  const groupedDays = useMemo(() => groupTimeEntriesByDay(entries, locale), [entries, locale]);

  return (
    <section className="time-list-panel" id="timesheet" aria-labelledby="timesheet-title">
      <div className="time-list-toolbar">
        <label className="select-all-control">
          <span className="entry-checkbox" aria-hidden="true" />
          {t('selectAll')}
        </label>
        <strong id="timesheet-title">{t('timesheet')}</strong>
        {isLoading ? <span className="sync-pill">{t('loading')}</span> : null}
      </div>
      <div className="time-entry-list" role="table" aria-label={t('timesheet')}>
        {groupedDays.length === 0 ? (
          <div className="empty-state">
            <Clock3 aria-hidden="true" />
            <p>{t('noTimeEntries')}</p>
          </div>
        ) : null}
        {groupedDays.map((day) => (
          <div className="time-day-group" role="rowgroup" key={day.date}>
            <div className="day-group-header" role="row">
              <div>
                <CalendarDays aria-hidden="true" />
                <strong>{day.day}</strong>
                <span>{day.date}</span>
              </div>
              <strong>{formatDuration(day.totalSeconds)}</strong>
            </div>
            {day.entries.map((entry) => (
              <TimesheetEntryRow entry={entry} key={entry.id} locale={locale} projects={projects} tasks={tasks} t={t} />
            ))}
          </div>
        ))}
      </div>
    </section>
  );
}

type ManualTimeEntryFormState = {
  clientId: string;
  projectId: string;
  taskId: string;
  tagIds: string[];
  description: string;
  startedAt: string;
  endedAt: string;
  billable: boolean;
};

type ManualTimeEntryFormErrors = Partial<Record<keyof ManualTimeEntryFormState | 'form', string>>;

function defaultManualTimeEntryForm(): ManualTimeEntryFormState {
  const now = new Date();
  const start = new Date(now);
  start.setMinutes(0, 0, 0);
  const end = new Date(start);
  end.setHours(end.getHours() + 1);
  return {
    clientId: '',
    projectId: '',
    taskId: '',
    tagIds: [],
    description: '',
    startedAt: toDateTimeLocalValue(start.toISOString()),
    endedAt: toDateTimeLocalValue(end.toISOString()),
    billable: true,
  };
}

export function ManualTimeEntryPanel({
  clients,
  isLoading,
  locale,
  projects,
  tags,
  tasks,
  t,
  timeEntries,
}: {
  clients: Client[];
  isLoading: boolean;
  locale: Locale;
  projects: Project[];
  tags: TagRecord[];
  tasks: Task[];
  t: Translator;
  timeEntries: TimeEntry[];
}) {
  const queryClient = useQueryClient();
  const [editingEntryId, setEditingEntryId] = useState<string | null>(null);
  const [form, setForm] = useState<ManualTimeEntryFormState>(defaultManualTimeEntryForm);
  const [errors, setErrors] = useState<ManualTimeEntryFormErrors>({});

  const createMutation = useMutation({
    mutationFn: createTimeEntry,
    onSuccess: () => {
      setForm(defaultManualTimeEntryForm());
      setErrors({});
      queryClient.invalidateQueries({ queryKey: ['time-entries'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
    },
    onError: () => setErrors((current) => ({ ...current, form: t('timeEntrySaveFailed') })),
  });

  const updateMutation = useMutation({
    mutationFn: ({ timeEntryId, input }: { timeEntryId: string; input: TimeEntryInput }) =>
      updateTimeEntry(timeEntryId, input),
    onSuccess: () => {
      setEditingEntryId(null);
      setForm(defaultManualTimeEntryForm());
      setErrors({});
      queryClient.invalidateQueries({ queryKey: ['time-entries'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
    },
    onError: () => setErrors((current) => ({ ...current, form: t('timeEntrySaveFailed') })),
  });

  const deleteMutation = useMutation({
    mutationFn: deleteTimeEntry,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['time-entries'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
    },
    onError: () => setErrors((current) => ({ ...current, form: t('timeEntryDeleteFailed') })),
  });

  const filteredTasks = useMemo(
    () => (form.projectId ? tasks.filter((task) => task.projectId === form.projectId || task.projectId === '') : tasks),
    [form.projectId, tasks],
  );

  function submitTimeEntry(event: FormEvent) {
    event.preventDefault();
    const validation = validateManualTimeEntryForm(form, t);
    setErrors(validation);
    if (hasErrors(validation)) {
      return;
    }

    const input = manualTimeEntryFormToInput(form);
    if (editingEntryId) {
      updateMutation.mutate({ timeEntryId: editingEntryId, input });
      return;
    }
    createMutation.mutate(input);
  }

  function updateField<K extends keyof ManualTimeEntryFormState>(field: K, value: ManualTimeEntryFormState[K]) {
    const next = { ...form, [field]: value };
    setForm(next);
    if (hasErrors(errors)) {
      setErrors(validateManualTimeEntryForm(next, t));
    }
  }

  function toggleTag(tagId: string) {
    const nextTagIds = form.tagIds.includes(tagId) ? form.tagIds.filter((id) => id !== tagId) : [...form.tagIds, tagId];
    updateField('tagIds', nextTagIds);
  }

  function startEditing(entry: TimeEntry) {
    setEditingEntryId(entry.id);
    setErrors({});
    setForm({
      clientId: entry.clientId,
      projectId: entry.projectId,
      taskId: entry.taskId,
      tagIds: entry.tags.map((tag) => tag.id),
      description: entry.description,
      startedAt: toDateTimeLocalValue(entry.startedAt),
      endedAt: toDateTimeLocalValue(entry.endedAt),
      billable: entry.billable,
    });
    scrollToManualEntryForm();
  }

  function cancelEditing() {
    setEditingEntryId(null);
    setForm(defaultManualTimeEntryForm());
    setErrors({});
  }

  const isSaving = createMutation.isPending || updateMutation.isPending;

  return (
    <section className="clients-section time-entry-section" id="manual-time-entry" aria-labelledby="manual-time-entry-title">
      <div className="clients-heading">
        <div className="section-title-group">
          <span className="section-kicker">
            <Clock3 aria-hidden="true" />
            {t('manualTimeEntry')}
          </span>
          <h2 id="manual-time-entry-title">{t('timeEntryDirectory')}</h2>
          <p>{t('timeEntryPanelSubtitle')}</p>
        </div>
        <button className="secondary-button" type="button" onClick={cancelEditing}>
          <Plus aria-hidden="true" />
          {t('newTimeEntry')}
        </button>
      </div>

      <div className="clients-workbench">
        <div className="client-directory">
          <div className="directory-toolbar">
            <div>
              <span>{t('activeTimeEntries')}</span>
              <strong>{timeEntries.length}</strong>
            </div>
            {isLoading ? (
              <span className="sync-pill">{t('loading')}</span>
            ) : (
              <span className="sync-pill">{t('synced')}</span>
            )}
          </div>

          <div className="client-list" aria-busy={isLoading}>
            {timeEntries.length === 0 ? (
              <div className="empty-state">
                <Clock3 aria-hidden="true" />
                <p>{t('noTimeEntries')}</p>
              </div>
            ) : null}
            {timeEntries.slice(0, 12).map((entry) => (
              <DirectoryEntryRow
                entry={entry}
                isSelected={editingEntryId === entry.id}
                key={entry.id}
                locale={locale}
                onDelete={() => deleteMutation.mutate(entry.id)}
                onOpenEditor={() => startEditing(entry)}
                onSynced={(updated) => {
                  if (editingEntryId === updated.id) {
                    setForm({
                      clientId: updated.clientId,
                      projectId: updated.projectId,
                      taskId: updated.taskId,
                      tagIds: updated.tags.map((tag) => tag.id),
                      description: updated.description,
                      startedAt: toDateTimeLocalValue(updated.startedAt),
                      endedAt: toDateTimeLocalValue(updated.endedAt),
                      billable: updated.billable,
                    });
                  }
                }}
                projects={projects}
                tasks={tasks}
                t={t}
              />
            ))}
          </div>
        </div>

        <form className="client-editor" noValidate onSubmit={submitTimeEntry}>
          <div className="editor-header">
            <div>
              <span>{editingEntryId ? t('editingTimeEntry') : t('createTimeEntry')}</span>
              <h3>{editingEntryId ? t('timeEntryFormEdit') : t('timeEntryFormCreate')}</h3>
            </div>
            {editingEntryId ? (
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
            <label className={fieldClass(errors.description)} htmlFor="time-entry-description">
              <span>{t('description')}</span>
              <input
                id="time-entry-description"
                onChange={(event) => updateField('description', event.target.value)}
                placeholder={t('timeEntryDescriptionPlaceholder')}
                value={form.description}
              />
            </label>

            <label className={fieldClass(errors.startedAt)} htmlFor="time-entry-start">
              <span>
                {t('startedAt')} <em>{t('required')}</em>
              </span>
              <input
                id="time-entry-start"
                onChange={(event) => updateField('startedAt', event.target.value)}
                type="datetime-local"
                value={form.startedAt}
              />
              <FieldError id="time-entry-start-error" message={errors.startedAt} />
            </label>

            <label className={fieldClass(errors.endedAt)} htmlFor="time-entry-end">
              <span>
                {t('endedAt')} <em>{t('required')}</em>
              </span>
              <input
                id="time-entry-end"
                onChange={(event) => updateField('endedAt', event.target.value)}
                type="datetime-local"
                value={form.endedAt}
              />
              <FieldError id="time-entry-end-error" message={errors.endedAt} />
            </label>

            <label className="form-field" htmlFor="time-entry-client">
              <span>{t('projectClient')}</span>
              <select id="time-entry-client" onChange={(event) => updateField('clientId', event.target.value)} value={form.clientId}>
                <option value="">{t('projectClientOptional')}</option>
                {clients.map((client) => (
                  <option key={client.id} value={client.id}>
                    {client.name}
                  </option>
                ))}
              </select>
            </label>

            <label className="form-field" htmlFor="time-entry-project">
              <span>{t('taskProject')}</span>
              <select id="time-entry-project" onChange={(event) => updateField('projectId', event.target.value)} value={form.projectId}>
                <option value="">{t('taskProjectOptional')}</option>
                {projects.map((project) => (
                  <option key={project.id} value={project.id}>
                    {project.name}
                  </option>
                ))}
              </select>
            </label>

            <label className="form-field" htmlFor="time-entry-task">
              <span>{t('tasks')}</span>
              <select id="time-entry-task" onChange={(event) => updateField('taskId', event.target.value)} value={form.taskId}>
                <option value="">{t('timeEntryTaskOptional')}</option>
                {filteredTasks.map((task) => (
                  <option key={task.id} value={task.id}>
                    {task.name}
                  </option>
                ))}
              </select>
            </label>

            <div className="form-field tag-picker-field">
              <span>{t('tags')}</span>
              <div className="tag-picker">
                {tags.map((tag) => (
                  <label key={tag.id}>
                    <input checked={form.tagIds.includes(tag.id)} onChange={() => toggleTag(tag.id)} type="checkbox" />
                    <span>{tag.name}</span>
                  </label>
                ))}
              </div>
            </div>

            <label className="form-field checkbox-field" htmlFor="time-entry-billable">
              <span>{t('billable')}</span>
              <input
                checked={form.billable}
                id="time-entry-billable"
                onChange={(event) => updateField('billable', event.target.checked)}
                type="checkbox"
              />
            </label>
          </div>

          <div className="client-form-actions">
            <button type="submit" disabled={isSaving}>
              {editingEntryId ? <Save aria-hidden="true" /> : <Plus aria-hidden="true" />}
              {editingEntryId ? t('updateTimeEntry') : t('createTimeEntry')}
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

function FieldError({ id, message }: { id: string; message?: string }) {
  if (!message) {
    return null;
  }
  return (
    <span className="field-message" id={id}>
      {message}
    </span>
  );
}

function fieldClass(error?: string) {
  return error ? 'form-field has-error' : 'form-field';
}

function hasErrors(errors: Record<string, string | undefined>) {
  return Object.values(errors).some(Boolean);
}

function validateManualTimeEntryForm(form: ManualTimeEntryFormState, t: Translator): ManualTimeEntryFormErrors {
  const errors: ManualTimeEntryFormErrors = {};

  if (!form.startedAt) {
    errors.startedAt = t('timeEntryStartRequired');
  }
  if (!form.endedAt) {
    errors.endedAt = t('timeEntryEndRequired');
  }
  if (form.startedAt && form.endedAt) {
    const start = new Date(form.startedAt);
    const end = new Date(form.endedAt);
    if (!Number.isNaN(start.getTime()) && !Number.isNaN(end.getTime())) {
      if (end <= start) {
        errors.endedAt = t('timeEntryEndAfterStart');
      } else if (end.getTime() - start.getTime() < 60_000) {
        errors.endedAt = t('timeEntryMinDuration');
      }
    }
  }

  return errors;
}

function timeEntryToInput(entry: TimeEntry, overrides: Partial<TimeEntryInlineForm>, projects: Project[], tasks: Task[]): TimeEntryInput {
  const description = overrides.description ?? entry.description;
  const projectId = overrides.projectId ?? entry.projectId;
  let taskId = overrides.taskId ?? entry.taskId;
  const startedAt =
    overrides.startedAt !== undefined ? fromDateTimeLocalValue(overrides.startedAt) : entry.startedAt;
  const endedAt = overrides.endedAt !== undefined ? fromDateTimeLocalValue(overrides.endedAt) : entry.endedAt;

  if (projectId && taskId) {
    const task = tasks.find((item) => item.id === taskId);
    if (task?.projectId && task.projectId !== projectId) {
      taskId = '';
    }
  }

  const project = projects.find((item) => item.id === projectId);

  return {
    clientId: project?.clientId ?? entry.clientId,
    projectId,
    taskId,
    tagIds: entry.tags.map((tag) => tag.id),
    description: description.trim(),
    startedAt,
    endedAt,
    billable: entry.billable,
  };
}

type TimeEntryInlineForm = {
  description: string;
  projectId: string;
  taskId: string;
  startedAt: string;
  endedAt: string;
};

function entryToInlineForm(entry: TimeEntry): TimeEntryInlineForm {
  return {
    description: entry.description,
    projectId: entry.projectId,
    taskId: entry.taskId,
    startedAt: toDateTimeLocalValue(entry.startedAt),
    endedAt: toDateTimeLocalValue(entry.endedAt),
  };
}

function validateInlineForm(form: TimeEntryInlineForm, t: Translator): string {
  if (!form.startedAt) {
    return t('timeEntryStartRequired');
  }
  if (!form.endedAt) {
    return t('timeEntryEndRequired');
  }
  const start = new Date(form.startedAt);
  const end = new Date(form.endedAt);
  if (Number.isNaN(start.getTime()) || Number.isNaN(end.getTime())) {
    return t('timeEntrySaveFailed');
  }
  if (end <= start) {
    return t('timeEntryEndAfterStart');
  }
  if (end.getTime() - start.getTime() < 60_000) {
    return t('timeEntryMinDuration');
  }
  return '';
}

function computeLiveDurationSeconds(form: TimeEntryInlineForm, fallback: number): number {
  const start = new Date(form.startedAt);
  const end = new Date(form.endedAt);
  if (Number.isNaN(start.getTime()) || Number.isNaN(end.getTime()) || end <= start) {
    return fallback;
  }
  return Math.floor((end.getTime() - start.getTime()) / 1000);
}

function useTimeEntryInlineEditor({
  entry,
  onSynced,
  projects,
  tasks,
  t,
}: {
  entry: TimeEntry;
  onSynced?: (updated: TimeEntry) => void;
  projects: Project[];
  tasks: Task[];
  t: Translator;
}) {
  const queryClient = useQueryClient();
  const [form, setForm] = useState(() => entryToInlineForm(entry));
  const [error, setError] = useState('');
  const skipSaveRef = useRef(true);
  const entryRef = useRef(entry);
  entryRef.current = entry;

  const updateMutation = useMutation({
    mutationFn: ({ timeEntryId, input }: { timeEntryId: string; input: TimeEntryInput }) =>
      updateTimeEntry(timeEntryId, input),
    onSuccess: (updated) => {
      queryClient.setQueryData(['time-entries'], (current: { timeEntries: TimeEntry[] } | undefined) => {
        if (!current) {
          return current;
        }
        return {
          timeEntries: current.timeEntries.map((item) => (item.id === updated.id ? updated : item)),
        };
      });
      onSynced?.(updated);
    },
    onError: () => setError(t('timeEntrySaveFailed')),
  });

  useEffect(() => {
    skipSaveRef.current = true;
    setForm(entryToInlineForm(entry));
    setError('');
  }, [entry.id]);

  useEffect(() => {
    if (skipSaveRef.current) {
      skipSaveRef.current = false;
      return;
    }

    const validationError = validateInlineForm(form, t);
    if (validationError) {
      setError(validationError);
      return;
    }

    setError('');
    const handle = window.setTimeout(() => {
      updateMutation.mutate({
        timeEntryId: entryRef.current.id,
        input: timeEntryToInput(entryRef.current, form, projects, tasks),
      });
    }, 400);

    return () => window.clearTimeout(handle);
  }, [form, entry.id, projects, tasks, t, updateMutation]);

  function updateField<K extends keyof TimeEntryInlineForm>(field: K, value: TimeEntryInlineForm[K]) {
    setForm((current) => {
      const next = { ...current, [field]: value };
      if (field === 'projectId') {
        const task = tasks.find((item) => item.id === current.taskId);
        if (task?.projectId && task.projectId !== value) {
          next.taskId = '';
        }
      }
      return next;
    });
  }

  const project = projects.find((item) => item.id === form.projectId);
  const liveDuration = computeLiveDurationSeconds(form, entry.durationSeconds);

  return { error, form, liveDuration, project, updateField };
}

function TimesheetEntryRow({
  entry,
  locale,
  projects,
  tasks,
  t,
}: {
  entry: TimeEntry;
  locale: Locale;
  projects: Project[];
  tasks: Task[];
  t: Translator;
}) {
  const { error, form, liveDuration, project, updateField } = useTimeEntryInlineEditor({ entry, projects, tasks, t });

  return (
    <div className="time-entry-row" role="row">
      <span className="entry-checkbox" aria-hidden="true" />
      <div className="entry-task">
        <input
          aria-label={t('description')}
          className="client-row-inline-input entry-inline-description"
          onChange={(event) => updateField('description', event.target.value)}
          placeholder={t('timeEntryDescriptionPlaceholder')}
          value={form.description}
        />
      </div>
      <select
        aria-label={t('taskProject')}
        className="entry-inline-select"
        onChange={(event) => updateField('projectId', event.target.value)}
        value={form.projectId}
      >
        <option value="">{t('taskProjectOptional')}</option>
        {projects.map((item) => (
          <option key={item.id} value={item.id}>
            {item.name}
          </option>
        ))}
      </select>
      <div className="entry-flags">
        {entry.tags.length > 0 ? <Tag aria-hidden="true" /> : null}
        <DollarSign aria-hidden="true" className={entry.billable ? 'billable-on' : undefined} />
        {entry.overlapWarning ? <CircleAlert aria-hidden="true" className="overlap-warning-icon" /> : null}
      </div>
      <input
        aria-label={t('startedAt')}
        className="entry-inline-datetime"
        onChange={(event) => updateField('startedAt', event.target.value)}
        type="datetime-local"
        value={form.startedAt}
      />
      <input
        aria-label={t('endedAt')}
        className="entry-inline-datetime"
        onChange={(event) => updateField('endedAt', event.target.value)}
        type="datetime-local"
        value={form.endedAt}
      />
      <strong className="entry-duration">{formatDuration(liveDuration)}</strong>
      {error ? (
        <span className="entry-inline-error" role="alert">
          {error}
        </span>
      ) : null}
      {project ? <span className="visually-hidden">{project.name}</span> : null}
      <span className="visually-hidden">{formatTimeRange(entry.startedAt, entry.endedAt, locale)}</span>
    </div>
  );
}

function DirectoryEntryRow({
  entry,
  isSelected,
  locale,
  onDelete,
  onOpenEditor,
  onSynced,
  projects,
  tasks,
  t,
}: {
  entry: TimeEntry;
  isSelected: boolean;
  locale: Locale;
  onDelete: () => void;
  onOpenEditor: () => void;
  onSynced?: (updated: TimeEntry) => void;
  projects: Project[];
  tasks: Task[];
  t: Translator;
}) {
  const { error, form, liveDuration, project, updateField } = useTimeEntryInlineEditor({
    entry,
    onSynced,
    projects,
    tasks,
    t,
  });

  return (
    <article className={isSelected ? 'client-row selected' : 'client-row'}>
      <div className="client-row-main">
        <div
          className="project-color-dot"
          style={{ backgroundColor: project?.color || entry.projectColor || '#64748b' }}
          aria-hidden="true"
        />
        <div className="client-row-copy entry-inline-copy">
          <div className="client-row-title">
            <input
              aria-label={t('description')}
              className="client-row-inline-input entry-inline-description"
              onChange={(event) => updateField('description', event.target.value)}
              placeholder={t('noDescription')}
              value={form.description}
            />
            {entry.overlapWarning ? (
              <span className="status-pill warning-pill">
                <CircleAlert aria-hidden="true" />
                {t('overlapWarning')}
              </span>
            ) : null}
          </div>
          <div className="entry-inline-meta">
            <select
              aria-label={t('taskProject')}
              className="entry-inline-select"
              onChange={(event) => updateField('projectId', event.target.value)}
              value={form.projectId}
            >
              <option value="">{t('taskProjectOptional')}</option>
              {projects.map((item) => (
                <option key={item.id} value={item.id}>
                  {item.name}
                </option>
              ))}
            </select>
            <input
              aria-label={t('startedAt')}
              className="entry-inline-datetime"
              onChange={(event) => updateField('startedAt', event.target.value)}
              type="datetime-local"
              value={form.startedAt}
            />
            <input
              aria-label={t('endedAt')}
              className="entry-inline-datetime"
              onChange={(event) => updateField('endedAt', event.target.value)}
              type="datetime-local"
              value={form.endedAt}
            />
            <span className="entry-inline-duration">{formatDuration(liveDuration)}</span>
          </div>
          {error ? (
            <span className="entry-inline-error" role="alert">
              {error}
            </span>
          ) : null}
        </div>
      </div>
      <div className="client-row-actions">
        <button className="secondary-button icon-button" type="button" onClick={onOpenEditor} title={t('edit')}>
          <Pencil aria-hidden="true" />
        </button>
        <button className="secondary-button icon-button danger-button" type="button" onClick={onDelete} title={t('delete')}>
          <Trash2 aria-hidden="true" />
        </button>
      </div>
      <span className="visually-hidden">{formatTimeRange(entry.startedAt, entry.endedAt, locale)}</span>
    </article>
  );
}

function manualTimeEntryFormToInput(form: ManualTimeEntryFormState): TimeEntryInput {
  return {
    clientId: form.clientId,
    projectId: form.projectId,
    taskId: form.taskId,
    tagIds: form.tagIds,
    description: form.description.trim(),
    startedAt: fromDateTimeLocalValue(form.startedAt),
    endedAt: fromDateTimeLocalValue(form.endedAt),
    billable: form.billable,
  };
}

function groupTimeEntriesByDay(entries: TimeEntry[], locale: Locale) {
  const groups = new Map<string, TimeEntry[]>();
  for (const entry of entries) {
    const dayKey = entry.startedAt.slice(0, 10);
    const current = groups.get(dayKey) ?? [];
    current.push(entry);
    groups.set(dayKey, current);
  }

  return Array.from(groups.entries()).map(([date, dayEntries]) => ({
    date,
    day: new Date(`${date}T12:00:00`).toLocaleDateString(locale === 'es' ? 'es-ES' : 'en-US', { weekday: 'long' }),
    entries: dayEntries,
    totalSeconds: dayEntries.reduce((sum, entry) => sum + entry.durationSeconds, 0),
  }));
}

export function formatDuration(totalSeconds: number) {
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  return `${hours}h ${String(minutes).padStart(2, '0')}min`;
}

function formatTimeRange(startedAt: string, endedAt: string, locale: Locale) {
  const formatter = new Intl.DateTimeFormat(locale === 'es' ? 'es-ES' : 'en-US', {
    hour: '2-digit',
    minute: '2-digit',
  });
  return `${formatter.format(new Date(startedAt))} - ${formatter.format(new Date(endedAt))}`;
}

function toDateTimeLocalValue(iso: string) {
  const date = new Date(iso);
  const pad = (value: number) => String(value).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function fromDateTimeLocalValue(value: string) {
  return new Date(value).toISOString();
}

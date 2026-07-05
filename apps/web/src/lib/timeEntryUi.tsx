import { useMutation, useQueryClient } from '@tanstack/react-query';
import { CalendarDays, ChevronLeft, ChevronRight, CircleAlert, Clock3, DollarSign, Pencil, Plus, Save, Tag, Trash2, X } from 'lucide-react';
import { FormEvent, useEffect, useMemo, useRef, useState, type CSSProperties } from 'react';
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
import { ProjectBadge, ProjectBadgeSelect } from './projectBadgeUi';
import {
  formatWeekRange,
  groupTimeEntriesByWeek,
  isSameWeek,
  startOfWeek,
  sumWeekSeconds,
  type TimesheetDayGroup,
} from './timesheetWeek';

export type Translator = (key: MessageKey) => string;

export function scrollToManualEntryForm() {
  document.getElementById('manual-time-entry')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
}

export function TimeEntriesList({
  entries,
  isLoading,
  locale,
  onNextWeek,
  onPreviousWeek,
  onTodayWeek,
  projects,
  tasks,
  t,
  weekAnchor,
}: {
  entries: TimeEntry[];
  isLoading: boolean;
  locale: Locale;
  onNextWeek: () => void;
  onPreviousWeek: () => void;
  onTodayWeek: () => void;
  projects: Project[];
  tasks: Task[];
  t: Translator;
  weekAnchor: Date;
}) {
  const weekStart = useMemo(() => startOfWeek(weekAnchor), [weekAnchor]);
  const weekEnd = useMemo(() => {
    const end = new Date(weekStart);
    end.setDate(end.getDate() + 6);
    return end;
  }, [weekStart]);
  const groupedDays = useMemo(
    () => groupTimeEntriesByWeek(entries, weekStart, locale),
    [entries, locale, weekStart],
  );
  const weekTotalSeconds = useMemo(() => sumWeekSeconds(groupedDays), [groupedDays]);
  const viewingCurrentWeek = isSameWeek(weekAnchor, new Date());

  return (
    <section className="time-list-panel" id="timesheet" aria-labelledby="timesheet-title">
      <div className="time-list-toolbar">
        <label className="select-all-control">
          <span className="entry-checkbox" aria-hidden="true" />
          {t('selectAll')}
        </label>

        <div className="week-nav" aria-label={t('timesheet')}>
          <button className="ghost-button icon-button week-nav-button" onClick={onPreviousWeek} type="button" title={t('previousWeek')}>
            <ChevronLeft aria-hidden="true" />
            <span className="visually-hidden">{t('previousWeek')}</span>
          </button>
          <div className="week-nav-label">
            <strong id="timesheet-title">{viewingCurrentWeek ? t('thisWeek') : t('timesheet')}</strong>
            <span>{formatWeekRange(weekStart, weekEnd, locale)}</span>
          </div>
          <button className="ghost-button icon-button week-nav-button" onClick={onNextWeek} type="button" title={t('nextWeek')}>
            <ChevronRight aria-hidden="true" />
            <span className="visually-hidden">{t('nextWeek')}</span>
          </button>
          {!viewingCurrentWeek ? (
            <button className="ghost-button week-today-button" onClick={onTodayWeek} type="button">
              {t('today')}
            </button>
          ) : null}
        </div>

        <div className="week-summary">
          <span>{t('weekTotal')}</span>
          <strong>{formatDuration(weekTotalSeconds)}</strong>
          {isLoading ? <span className="sync-pill">{t('loading')}</span> : null}
        </div>
      </div>
      <div className="time-entry-list" role="table" aria-label={t('timesheet')}>
        {groupedDays.map((day) => (
          <TimesheetDaySection day={day} key={day.date} locale={locale} projects={projects} tasks={tasks} t={t} />
        ))}
      </div>
    </section>
  );
}

function TimesheetDaySection({
  day,
  locale,
  projects,
  tasks,
  t,
}: {
  day: TimesheetDayGroup;
  locale: Locale;
  projects: Project[];
  tasks: Task[];
  t: Translator;
}) {
  if (day.entries.length === 0) {
    return (
      <div className="time-day-group time-day-group-empty" role="rowgroup">
        <div className="day-group-header" role="row">
          <div>
            <CalendarDays aria-hidden="true" />
            <strong>{day.day}</strong>
            <span>{formatDayDate(day.date, locale)}</span>
          </div>
          <strong>{formatDuration(0)}</strong>
        </div>
      </div>
    );
  }

  return (
    <div className="time-day-group" role="rowgroup">
      <div className="day-group-header" role="row">
        <div>
          <CalendarDays aria-hidden="true" />
          <strong>{day.day}</strong>
          <span>{formatDayDate(day.date, locale)}</span>
        </div>
        <strong>{formatDuration(day.totalSeconds)}</strong>
      </div>
      {day.entries.map((entry) => (
        <TimesheetEntryRow entry={entry} key={entry.id} locale={locale} projects={projects} tasks={tasks} t={t} />
      ))}
    </div>
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

export function tasksForManualEntryForm(projectId: string, taskId: string, tasks: Task[]): Task[] {
  const activeTasks = tasks.filter((task) => !task.archivedAt);
  const base = projectId
    ? activeTasks.filter((task) => task.projectId === projectId || task.projectId === '')
    : activeTasks;
  if (!taskId || base.some((task) => task.id === taskId)) {
    return base;
  }
  const selected = tasks.find((task) => task.id === taskId);
  return selected ? [...base, selected] : base;
}

export function applyManualEntryFieldUpdate(
  form: ManualTimeEntryFormState,
  field: keyof ManualTimeEntryFormState,
  value: ManualTimeEntryFormState[keyof ManualTimeEntryFormState],
  projects: Project[],
  tasks: Task[],
): ManualTimeEntryFormState {
  const next = { ...form, [field]: value } as ManualTimeEntryFormState;

  if (field === 'taskId' && typeof value === 'string' && value) {
    const task = tasks.find((item) => item.id === value);
    if (task?.projectId) {
      next.projectId = task.projectId;
      const project = projects.find((item) => item.id === task.projectId);
      if (project?.clientId) {
        next.clientId = project.clientId;
      }
    }
  }

  if (field === 'projectId' && typeof value === 'string') {
    const task = tasks.find((item) => item.id === next.taskId);
    if (task?.projectId && task.projectId !== value) {
      next.taskId = '';
    }
  }

  return next;
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
    () => tasksForManualEntryForm(form.projectId, form.taskId, tasks),
    [form.projectId, form.taskId, tasks],
  );

  const visibleTags = useMemo(() => {
    const activeTags = tags.filter((tag) => !tag.archivedAt);
    const selectedArchived = tags.filter((tag) => tag.archivedAt && form.tagIds.includes(tag.id));
    return [...activeTags, ...selectedArchived];
  }, [form.tagIds, tags]);

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
    const next = applyManualEntryFieldUpdate(form, field, value, projects, tasks);
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
                pauseInlineSave={editingEntryId === entry.id}
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
                {clients.filter((client) => !client.archivedAt).map((client) => (
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
                {projects.filter((project) => !project.archivedAt).map((project) => (
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
                {visibleTags.map((tag) => (
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
  autoSave = true,
  entry,
  projects,
  tasks,
  t,
}: {
  autoSave?: boolean;
  entry: TimeEntry;
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
    },
    onError: () => setError(t('timeEntrySaveFailed')),
  });

  useEffect(() => {
    skipSaveRef.current = true;
    setForm(entryToInlineForm(entry));
    setError('');
  }, [entry.id, entry.updatedAt]);

  useEffect(() => {
    if (!autoSave) {
      skipSaveRef.current = true;
      return;
    }

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
  }, [autoSave, form, entry.id, projects, tasks, t, updateMutation]);

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
  const projectColor = project?.color || entry.projectColor;
  const liveDuration = computeLiveDurationSeconds(form, entry.durationSeconds);

  return { error, form, liveDuration, project, projectColor, updateField };
}

export function TimesheetEntryRow({
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
      <ProjectBadgeSelect
        ariaLabel={t('taskProject')}
        emptyLabel={t('taskProjectOptional')}
        onChange={(projectId) => updateField('projectId', projectId)}
        projects={projects}
        value={form.projectId}
      />
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
  pauseInlineSave,
  projects,
  tasks,
  t,
}: {
  entry: TimeEntry;
  isSelected: boolean;
  locale: Locale;
  onDelete: () => void;
  onOpenEditor: () => void;
  pauseInlineSave: boolean;
  projects: Project[];
  tasks: Task[];
  t: Translator;
}) {
  const { error, form, liveDuration, projectColor, updateField } = useTimeEntryInlineEditor({
    autoSave: !pauseInlineSave,
    entry,
    projects,
    tasks,
    t,
  });

  const entryAccentStyle = projectColor ? ({ '--entry-accent': projectColor } as CSSProperties) : undefined;

  return (
    <article className={isSelected ? 'client-row time-entry-directory-row selected' : 'client-row time-entry-directory-row'}>
      <div className="time-entry-directory-top">
        <div className="time-entry-directory-title">
          <input
            aria-label={t('description')}
            className="client-row-inline-input entry-inline-description"
            onChange={(event) => updateField('description', event.target.value)}
            placeholder={t('noDescription')}
            style={entryAccentStyle}
            value={form.description}
          />
          {entry.overlapWarning ? (
            <span className="status-pill warning-pill">
              <CircleAlert aria-hidden="true" />
              {t('overlapWarning')}
            </span>
          ) : null}
        </div>
        <div className="client-row-actions">
          <button className="secondary-button icon-button" type="button" onClick={onOpenEditor} title={t('edit')}>
            <Pencil aria-hidden="true" />
          </button>
          <button className="secondary-button icon-button danger-button" type="button" onClick={onDelete} title={t('delete')}>
            <Trash2 aria-hidden="true" />
          </button>
        </div>
      </div>
      <div className="entry-inline-meta">
        <ProjectBadgeSelect
          ariaLabel={t('taskProject')}
          color={projectColor}
          emptyLabel={t('taskProjectOptional')}
          onChange={(projectId) => updateField('projectId', projectId)}
          projects={projects}
          value={form.projectId}
        />
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
        <span className="entry-inline-error time-entry-directory-error" role="alert">
          {error}
        </span>
      ) : null}
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

export function formatDuration(totalSeconds: number) {
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  return `${hours}h ${String(minutes).padStart(2, '0')}min`;
}

function formatDayDate(date: string, locale: Locale) {
  return new Date(`${date}T12:00:00`).toLocaleDateString(locale === 'es' ? 'es-ES' : 'en-US', {
    day: 'numeric',
    month: 'short',
  });
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

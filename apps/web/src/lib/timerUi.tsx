import { useMutation, useQueryClient } from '@tanstack/react-query';
import { CircleAlert, DollarSign, Plus, Tag } from 'lucide-react';
import { FormEvent, useEffect, useMemo, useRef, useState } from 'react';
import { startTimer, updateTimer, type Project, type Task, type TimeEntry, type TimerStartInput } from './api';
import type { MessageKey } from './i18n';
import { scrollToManualEntryForm } from './timeEntryUi';
import { ProjectBadge } from './projectBadgeUi';
import { TimerPlayIcon, TimerStopIcon } from './timerIcons';

export type Translator = (key: MessageKey) => string;

type TimerStartFormState = {
  projectId: string;
  taskId: string;
  description: string;
  billable: boolean;
};

const emptyTimerForm: TimerStartFormState = {
  projectId: '',
  taskId: '',
  description: '',
  billable: true,
};

export function SidebarTimer({
  activeTimer,
  onStop,
  stoppingTimerId,
  t,
}: {
  activeTimer: TimeEntry | null;
  onStop: (timeEntryId: string) => void;
  stoppingTimerId: string | null;
  t: Translator;
}) {
  const elapsed = useElapsedSeconds(activeTimer?.startedAt ?? null);

  return (
    <div className="sidebar-timer">
      <div>
        <span>{t('currentTimer')}</span>
        <strong>{activeTimer ? formatElapsedClock(elapsed) : '--:--:--'}</strong>
        {activeTimer ? <small>{activeTimer.description || t('noDescription')}</small> : <small>{t('noActiveTimer')}</small>}
      </div>
      {activeTimer ? (
        <button
          className="sidebar-stop-button"
          disabled={stoppingTimerId === activeTimer.id}
          onClick={() => onStop(activeTimer.id)}
          type="button"
          title={t('stop')}
        >
          <TimerStopIcon className="timer-stop-icon" />
          <span className="visually-hidden">{t('stop')}</span>
        </button>
      ) : (
        <div aria-hidden="true" className="sidebar-idle-indicator" title={t('startTimer')}>
          <TimerPlayIcon className="timer-play-icon timer-play-icon-idle" />
        </div>
      )}
    </div>
  );
}

export function TimerCommandRow({
  onStop,
  projects,
  stoppingTimerId,
  tasks,
  timers,
  t,
}: {
  onStop: (timeEntryId: string) => void;
  projects: Project[];
  stoppingTimerId: string | null;
  tasks: Task[];
  timers: TimeEntry[];
  t: Translator;
}) {
  const queryClient = useQueryClient();
  const [form, setForm] = useState<TimerStartFormState>(emptyTimerForm);
  const [error, setError] = useState('');
  const activeTimer = timers[0] ?? null;
  const elapsed = useElapsedSeconds(activeTimer?.startedAt ?? null);
  const [liveDescription, setLiveDescription] = useState('');
  const [clockPopoverOpen, setClockPopoverOpen] = useState(false);
  const [liveStartedDate, setLiveStartedDate] = useState('');
  const [liveStartedTime, setLiveStartedTime] = useState('');
  const skipDescriptionSaveRef = useRef(true);
  const skipStartedAtSaveRef = useRef(true);
  const clockPopoverRef = useRef<HTMLDivElement>(null);
  const activeTimerRef = useRef(activeTimer);
  activeTimerRef.current = activeTimer;

  useEffect(() => {
    skipDescriptionSaveRef.current = true;
    skipStartedAtSaveRef.current = true;
    setLiveDescription(activeTimer?.description ?? '');
    if (activeTimer?.startedAt) {
      const parts = timerStartParts(activeTimer.startedAt);
      setLiveStartedDate(parts.date);
      setLiveStartedTime(parts.time);
    } else {
      setLiveStartedDate('');
      setLiveStartedTime('');
    }
    setClockPopoverOpen(false);
  }, [activeTimer?.id, activeTimer?.startedAt, activeTimer?.description]);

  const filteredTasks = useMemo(() => {
    const activeTasks = tasks.filter((task) => !task.archivedAt);
    return form.projectId
      ? activeTasks.filter((task) => task.projectId === form.projectId || task.projectId === '')
      : activeTasks;
  }, [form.projectId, tasks]);

  const startMutation = useMutation({
    mutationFn: startTimer,
    onSuccess: () => {
      setForm(emptyTimerForm);
      setError('');
      queryClient.invalidateQueries({ queryKey: ['timers'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
    },
    onError: () => setError(t('timerStartFailed')),
  });

  const updateMutation = useMutation({
    mutationFn: ({ timeEntryId, input }: { timeEntryId: string; input: TimerStartInput }) =>
      updateTimer(timeEntryId, input),
    onSuccess: (updated) => {
      queryClient.setQueryData(['timers'], (current: { timers: TimeEntry[] } | undefined) => {
        if (!current) {
          return current;
        }
        return {
          timers: current.timers.map((timer) => (timer.id === updated.id ? updated : timer)),
        };
      });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
    },
    onError: () => setError(t('timerUpdateFailed')),
  });

  useEffect(() => {
    const timer = activeTimerRef.current;
    if (!timer) {
      return;
    }
    if (skipDescriptionSaveRef.current) {
      skipDescriptionSaveRef.current = false;
      return;
    }
    if (liveDescription === timer.description) {
      return;
    }

    const handle = window.setTimeout(() => {
      const currentTimer = activeTimerRef.current;
      if (!currentTimer) {
        return;
      }
      updateMutation.mutate({
        timeEntryId: currentTimer.id,
        input: timerEntryToInput(currentTimer, { description: liveDescription }),
      });
    }, 400);

    return () => window.clearTimeout(handle);
  }, [activeTimer?.id, liveDescription, updateMutation]);

  useEffect(() => {
    const timer = activeTimerRef.current;
    if (!timer || !liveStartedDate || !liveStartedTime) {
      return;
    }
    if (skipStartedAtSaveRef.current) {
      skipStartedAtSaveRef.current = false;
      return;
    }

    const nextStartedAt = timerStartISO(liveStartedDate, liveStartedTime);
    if (timerStartParts(timer.startedAt).date === liveStartedDate && timerStartParts(timer.startedAt).time === liveStartedTime) {
      return;
    }
    if (Date.parse(nextStartedAt) > Date.now()) {
      setError(t('timerUpdateFailed'));
      return;
    }

    const handle = window.setTimeout(() => {
      const currentTimer = activeTimerRef.current;
      if (!currentTimer) {
        return;
      }
      updateMutation.mutate({
        timeEntryId: currentTimer.id,
        input: timerEntryToInput(currentTimer, {
          description: liveDescription,
          startedAt: nextStartedAt,
        }),
      });
    }, 400);

    return () => window.clearTimeout(handle);
  }, [activeTimer?.id, liveDescription, liveStartedDate, liveStartedTime, t, updateMutation]);

  useEffect(() => {
    if (!clockPopoverOpen) {
      return;
    }

    let removeListener: (() => void) | undefined;
    const timeoutId = window.setTimeout(() => {
      function handlePointerDown(event: MouseEvent) {
        if (!clockPopoverRef.current?.contains(event.target as Node)) {
          setClockPopoverOpen(false);
        }
      }

      document.addEventListener('mousedown', handlePointerDown);
      removeListener = () => document.removeEventListener('mousedown', handlePointerDown);
    }, 0);

    return () => {
      window.clearTimeout(timeoutId);
      removeListener?.();
    };
  }, [clockPopoverOpen]);

  function submitStart(event: FormEvent) {
    event.preventDefault();
    const input: TimerStartInput = {
      clientId: '',
      projectId: form.projectId,
      taskId: form.taskId,
      tagIds: [],
      description: form.description.trim(),
      billable: form.billable,
    };
    startMutation.mutate(input);
  }

  return (
    <section className="timer-command-row" aria-label={t('currentTimer')}>
      {activeTimer ? (
        <>
          <div className="active-timer-card">
            <div className="timer-card-main">
              <input
                aria-label={t('description')}
                className="timer-description-input timer-description-live"
                onChange={(event) => setLiveDescription(event.target.value)}
                placeholder={t('timerDescriptionPlaceholder')}
                value={liveDescription}
              />
              <div className="timer-card-badges">
                <ProjectBadge
                  color={activeTimer.projectColor}
                  emptyLabel={t('taskProjectOptional')}
                  name={activeTimer.projectName}
                />
                {activeTimer.overlapWarning ? (
                  <span className="status-pill warning-pill">
                    <CircleAlert aria-hidden="true" />
                    {t('overlapWarning')}
                  </span>
                ) : null}
              </div>
            </div>
            <div className="timer-card-controls">
              {activeTimer.tags.length > 0 ? (
                <button className="quiet-icon-button" disabled type="button" title={t('tags')}>
                  <Tag aria-hidden="true" />
                </button>
              ) : null}
              <button
                className={`quiet-icon-button${activeTimer.billable ? ' billable' : ''}`}
                disabled
                type="button"
                title={t('billable')}
              >
                <DollarSign aria-hidden="true" />
              </button>
              <div
                className={`timer-clock-wrap${clockPopoverOpen ? ' is-open' : ''}`}
                onMouseDown={(event) => event.stopPropagation()}
                ref={clockPopoverRef}
              >
                <button
                  aria-expanded={clockPopoverOpen}
                  aria-haspopup="dialog"
                  aria-label={t('timerEditStart')}
                  className="timer-clock-button"
                  onClick={() => setClockPopoverOpen((open) => !open)}
                  type="button"
                >
                  {formatElapsedClock(elapsed)}
                </button>
                {clockPopoverOpen ? (
                  <div className="timer-clock-popover" role="dialog">
                    <div className="timer-clock-popover-head">
                      <span>{t('startedAt')}</span>
                      <span>{t('endedAt')}</span>
                    </div>
                    <div className="timer-clock-popover-body">
                      <input
                        aria-label={t('startedAt')}
                        className="timer-clock-input timer-clock-input-time"
                        onChange={(event) => setLiveStartedTime(event.target.value)}
                        type="time"
                        value={liveStartedTime}
                      />
                      <span aria-hidden="true" className="timer-clock-end-value">
                        {t('timerRunningEnd')}
                      </span>
                      <input
                        aria-label={t('startedAt')}
                        className="timer-clock-input timer-clock-input-date"
                        onChange={(event) => setLiveStartedDate(event.target.value)}
                        type="date"
                        value={liveStartedDate}
                      />
                    </div>
                  </div>
                ) : null}
              </div>
            </div>
          </div>
          <button
            className="stop-timer-button"
            disabled={stoppingTimerId === activeTimer.id}
            onClick={() => onStop(activeTimer.id)}
            type="button"
            title={t('stop')}
          >
            <TimerStopIcon className="timer-stop-icon" />
            <span className="visually-hidden">{t('stop')}</span>
          </button>
        </>
      ) : (
        <form className="active-timer-card timer-start-form" onSubmit={submitStart}>
          <input
            aria-label={t('description')}
            className="timer-description-input"
            onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))}
            placeholder={t('timerDescriptionPlaceholder')}
            value={form.description}
          />
          <select
            aria-label={t('taskProject')}
            onChange={(event) => setForm((current) => ({ ...current, projectId: event.target.value, taskId: '' }))}
            value={form.projectId}
          >
            <option value="">{t('taskProjectOptional')}</option>
            {projects.filter((project) => !project.archivedAt).map((project) => (
              <option key={project.id} value={project.id}>
                {project.name}
              </option>
            ))}
          </select>
          <select
            aria-label={t('taskName')}
            onChange={(event) => setForm((current) => ({ ...current, taskId: event.target.value }))}
            value={form.taskId}
          >
            <option value="">{t('taskProjectOptional')}</option>
            {filteredTasks.map((task) => (
              <option key={task.id} value={task.id}>
                {task.name}
              </option>
            ))}
          </select>
          <button className="start-timer-button" disabled={startMutation.isPending} type="submit" title={t('startTimer')}>
            <TimerPlayIcon className="timer-play-icon" />
            {t('startTimer')}
          </button>
        </form>
      )}

      <button className="manual-entry-button" type="button" onClick={() => scrollToManualEntryForm()}>
        <Plus aria-hidden="true" />
        {t('manualTimeEntry')}
      </button>

      {error ? (
        <div className="timer-inline-error" role="alert">
          {error}
        </div>
      ) : null}

      {timers.length > 1 ? (
        <div className="open-timers-note">
          {t('openTimersCount').replace('{count}', String(timers.length))}
        </div>
      ) : null}
    </section>
  );
}

export function useElapsedSeconds(startedAt: string | null) {
  const [elapsed, setElapsed] = useState(0);

  useEffect(() => {
    if (!startedAt) {
      setElapsed(0);
      return;
    }

    function tick() {
      const started = Date.parse(startedAt!);
      setElapsed(Math.max(0, Math.floor((Date.now() - started) / 1000)));
    }

    tick();
    const interval = window.setInterval(tick, 1000);
    return () => window.clearInterval(interval);
  }, [startedAt]);

  return elapsed;
}

export function formatElapsedClock(totalSeconds: number) {
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  return `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
}

function timerEntryToInput(
  timer: TimeEntry,
  overrides: { description?: string; startedAt?: string } = {},
): TimerStartInput {
  return {
    clientId: timer.clientId,
    projectId: timer.projectId,
    taskId: timer.taskId,
    tagIds: timer.tags.map((tag) => tag.id),
    description: overrides.description ?? timer.description,
    billable: timer.billable,
    ...(overrides.startedAt ? { startedAt: overrides.startedAt } : {}),
  };
}

function timerStartParts(iso: string): { date: string; time: string } {
  const value = new Date(iso);
  const pad = (part: number) => String(part).padStart(2, '0');
  return {
    date: `${value.getFullYear()}-${pad(value.getMonth() + 1)}-${pad(value.getDate())}`,
    time: `${pad(value.getHours())}:${pad(value.getMinutes())}`,
  };
}

function timerStartISO(date: string, time: string): string {
  return new Date(`${date}T${time}`).toISOString();
}

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { CircleAlert, DollarSign, Plus, Tag } from 'lucide-react';
import { FormEvent, useEffect, useMemo, useRef, useState } from 'react';
import { startTimer, updateTimer, type Project, type Task, type TimeEntry, type TimerStartInput } from './api';
import type { MessageKey } from './i18n';
import { scrollToManualEntryForm } from './timeEntryUi';
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
  const skipDescriptionSaveRef = useRef(true);
  const activeTimerRef = useRef(activeTimer);
  activeTimerRef.current = activeTimer;

  useEffect(() => {
    skipDescriptionSaveRef.current = true;
    setLiveDescription(activeTimer?.description ?? '');
  }, [activeTimer?.id]);

  const filteredTasks = useMemo(
    () => (form.projectId ? tasks.filter((task) => task.projectId === form.projectId || task.projectId === '') : tasks),
    [form.projectId, tasks],
  );

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
        input: timerEntryToInput(currentTimer, liveDescription),
      });
    }, 400);

    return () => window.clearTimeout(handle);
  }, [activeTimer?.id, liveDescription, updateMutation]);

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
                {activeTimer.projectName ? (
                  <span className="entity-pill">
                    <span style={{ backgroundColor: activeTimer.projectColor || '#64748b' }} aria-hidden="true" />
                    {activeTimer.projectName}
                  </span>
                ) : null}
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
              <strong className="timer-clock">{formatElapsedClock(elapsed)}</strong>
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
            {projects.map((project) => (
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

function timerEntryToInput(timer: TimeEntry, description: string): TimerStartInput {
  return {
    clientId: timer.clientId,
    projectId: timer.projectId,
    taskId: timer.taskId,
    tagIds: timer.tags.map((tag) => tag.id),
    description,
    billable: timer.billable,
  };
}

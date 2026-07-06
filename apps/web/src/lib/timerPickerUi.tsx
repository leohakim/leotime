import { useMutation, useQueryClient } from '@tanstack/react-query';
import { ChevronDown, ChevronRight, DollarSign, Plus, Tag } from 'lucide-react';
import { useEffect, useMemo, useRef, useState } from 'react';
import { createTag, type Project, type Tag as TagRecord, type Task } from './api';
import type { MessageKey } from './i18n';
import { ProjectBadge } from './projectBadgeUi';

export type Translator = (key: MessageKey) => string;

export type TimerMetaSelection = {
  projectId: string;
  taskId: string;
  tagIds: string[];
  billable: boolean;
};

type ProjectGroup = {
  clientKey: string;
  clientLabel: string;
  projects: Project[];
};

export function groupProjectsForPicker(projects: Project[], query: string): ProjectGroup[] {
  const normalized = query.trim().toLowerCase();
  const activeProjects = projects.filter((project) => !project.archivedAt);
  const filtered = normalized
    ? activeProjects.filter(
        (project) =>
          project.name.toLowerCase().includes(normalized) ||
          project.clientName.toLowerCase().includes(normalized),
      )
    : activeProjects;

  const groups = new Map<string, ProjectGroup>();
  for (const project of filtered) {
    const clientKey = project.clientId || '__none__';
    const clientLabel = project.clientName.trim() || '';
    const existing = groups.get(clientKey);
    if (existing) {
      existing.projects.push(project);
      continue;
    }
    groups.set(clientKey, {
      clientKey,
      clientLabel,
      projects: [project],
    });
  }

  return [...groups.values()].sort((left, right) => {
    const leftLabel = left.clientLabel || '\uffff';
    const rightLabel = right.clientLabel || '\uffff';
    return leftLabel.localeCompare(rightLabel, undefined, { sensitivity: 'base' });
  });
}

function tasksForProject(tasks: Task[], projectId: string, query: string) {
  const normalized = query.trim().toLowerCase();
  return tasks.filter((task) => {
    if (task.archivedAt || task.projectId !== projectId) {
      return false;
    }
    if (!normalized) {
      return true;
    }
    return task.name.toLowerCase().includes(normalized);
  });
}

function usePopoverDismiss(open: boolean, setOpen: (open: boolean) => void) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) {
      return;
    }

    let removeListener: (() => void) | undefined;
    const timeoutId = window.setTimeout(() => {
      function handlePointerDown(event: MouseEvent) {
        if (!ref.current?.contains(event.target as Node)) {
          setOpen(false);
        }
      }

      document.addEventListener('mousedown', handlePointerDown);
      removeListener = () => document.removeEventListener('mousedown', handlePointerDown);
    }, 0);

    return () => {
      window.clearTimeout(timeoutId);
      removeListener?.();
    };
  }, [open, setOpen]);

  return ref;
}

export function TimerProjectPicker({
  onChange,
  onCreateProject,
  projects,
  selection,
  tasks,
  t,
}: {
  onChange: (next: Pick<TimerMetaSelection, 'projectId' | 'taskId'>) => void;
  onCreateProject: () => void;
  projects: Project[];
  selection: Pick<TimerMetaSelection, 'projectId' | 'taskId'>;
  tasks: Task[];
  t: Translator;
}) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState('');
  const [expandedProjects, setExpandedProjects] = useState<Record<string, boolean>>({});
  const popoverRef = usePopoverDismiss(open, setOpen);

  const groups = useMemo(() => groupProjectsForPicker(projects, query), [projects, query]);
  const selectedProject = projects.find((project) => project.id === selection.projectId);
  const selectedTask = tasks.find((task) => task.id === selection.taskId);
  const triggerLabel = selectedTask?.name ?? selectedProject?.name ?? t('timerNoProject');

  function selectProject(projectId: string, taskId = '') {
    onChange({ projectId, taskId });
    setOpen(false);
    setQuery('');
  }

  function toggleProjectTasks(projectId: string) {
    setExpandedProjects((current) => ({ ...current, [projectId]: !current[projectId] }));
  }

  return (
    <div className={`timer-picker-wrap${open ? ' is-open' : ''}`} onMouseDown={(event) => event.stopPropagation()} ref={popoverRef}>
      <button
        aria-expanded={open}
        aria-haspopup="dialog"
        className="timer-project-trigger"
        onClick={() => setOpen((value) => !value)}
        type="button"
      >
        <ProjectBadge
          color={selectedProject?.color}
          compact
          emptyLabel={t('timerNoProject')}
          name={selectedProject ? triggerLabel : undefined}
        />
      </button>
      {open ? (
        <div className="timer-picker-popover timer-project-popover" role="dialog">
          <div className="timer-picker-search">
            <input
              aria-label={t('timerSearchProjectTask')}
              autoFocus
              onChange={(event) => setQuery(event.target.value)}
              placeholder={t('timerSearchProjectTask')}
              type="search"
              value={query}
            />
          </div>
          <div className="timer-picker-list">
            <button
              className={selection.projectId ? 'timer-picker-option' : 'timer-picker-option is-selected'}
              onClick={() => selectProject('')}
              type="button"
            >
              <ProjectBadge compact emptyLabel={t('timerNoProject')} />
            </button>
            {groups.map((group) => (
              <div className="timer-picker-group" key={group.clientKey}>
                {group.clientLabel ? <div className="timer-picker-group-label">{group.clientLabel}</div> : null}
                {group.projects.map((project) => {
                  const projectTasks = tasksForProject(tasks, project.id, query);
                  const expanded = expandedProjects[project.id] ?? Boolean(query.trim());
                  const taskCount = tasks.filter((task) => !task.archivedAt && task.projectId === project.id).length;
                  const isSelected = selection.projectId === project.id && !selection.taskId;

                  return (
                    <div className="timer-picker-project-block" key={project.id}>
                      <div className="timer-picker-option-row">
                        <button
                          className={isSelected ? 'timer-picker-option is-selected' : 'timer-picker-option'}
                          onClick={() => selectProject(project.id)}
                          type="button"
                        >
                          <span aria-hidden="true" className="timer-picker-dot" style={{ backgroundColor: project.color }} />
                          <span className="timer-picker-option-label">{project.name}</span>
                        </button>
                        {taskCount > 0 ? (
                          <button
                            aria-expanded={expanded}
                            aria-label={t('timerTasksCount').replace('{count}', String(taskCount))}
                            className="timer-picker-expand"
                            onClick={() => toggleProjectTasks(project.id)}
                            type="button"
                          >
                            <span>{t('timerTasksCount').replace('{count}', String(taskCount))}</span>
                            {expanded ? <ChevronDown aria-hidden="true" /> : <ChevronRight aria-hidden="true" />}
                          </button>
                        ) : null}
                      </div>
                      {expanded && projectTasks.length > 0 ? (
                        <div className="timer-picker-task-list">
                          {projectTasks.map((task) => (
                            <button
                              className={
                                selection.taskId === task.id ? 'timer-picker-option is-selected timer-picker-task' : 'timer-picker-option timer-picker-task'
                              }
                              key={task.id}
                              onClick={() => selectProject(project.id, task.id)}
                              type="button"
                            >
                              <span className="timer-picker-task-marker" />
                              <span className="timer-picker-option-label">{task.name}</span>
                            </button>
                          ))}
                        </div>
                      ) : null}
                    </div>
                  );
                })}
              </div>
            ))}
          </div>
          <div className="timer-picker-footer">
            <button className="timer-picker-create" onClick={onCreateProject} type="button">
              <Plus aria-hidden="true" />
              {t('timerCreateProject')}
            </button>
          </div>
        </div>
      ) : null}
    </div>
  );
}

export function TimerTagPicker({
  onChange,
  t,
  tagIds,
  tags,
}: {
  onChange: (tagIds: string[]) => void;
  t: Translator;
  tagIds: string[];
  tags: TagRecord[];
}) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState('');
  const popoverRef = usePopoverDismiss(open, setOpen);
  const queryClient = useQueryClient();

  const createTagMutation = useMutation({
    mutationFn: createTag,
    onSuccess: (created) => {
      onChange([...new Set([...tagIds, created.id])]);
      setQuery('');
      queryClient.invalidateQueries({ queryKey: ['tags'] });
    },
  });

  const activeTags = useMemo(() => {
    const normalized = query.trim().toLowerCase();
    return tags.filter((tag) => {
      if (tag.archivedAt) {
        return tagIds.includes(tag.id);
      }
      if (!normalized) {
        return true;
      }
      return tag.name.toLowerCase().includes(normalized);
    });
  }, [query, tagIds, tags]);

  const trimmedQuery = query.trim();
  const canCreate =
    trimmedQuery.length >= 2 &&
    !tags.some((tag) => !tag.archivedAt && tag.name.toLowerCase() === trimmedQuery.toLowerCase());

  function toggleTag(tagId: string) {
    if (tagIds.includes(tagId)) {
      onChange(tagIds.filter((id) => id !== tagId));
      return;
    }
    onChange([...tagIds, tagId]);
  }

  function handleCreateTag() {
    if (!canCreate || createTagMutation.isPending) {
      return;
    }
    createTagMutation.mutate({ name: trimmedQuery, color: '#64748b' });
  }

  return (
    <div className={`timer-picker-wrap${open ? ' is-open' : ''}`} onMouseDown={(event) => event.stopPropagation()} ref={popoverRef}>
      <button
        aria-expanded={open}
        aria-haspopup="dialog"
        className={`quiet-icon-button${open || tagIds.length > 0 ? ' is-active' : ''}`}
        onClick={() => setOpen((value) => !value)}
        type="button"
        title={t('tags')}
      >
        <Tag aria-hidden="true" />
      </button>
      {open ? (
        <div className="timer-picker-popover timer-tag-popover" role="dialog">
          <div className="timer-picker-search">
            <input
              aria-label={t('timerSearchTag')}
              autoFocus
              onChange={(event) => setQuery(event.target.value)}
              placeholder={t('timerSearchTag')}
              type="search"
              value={query}
            />
          </div>
          <div className="timer-picker-list">
            {activeTags.map((tag) => (
              <label className="timer-picker-tag-option" key={tag.id}>
                <input checked={tagIds.includes(tag.id)} onChange={() => toggleTag(tag.id)} type="checkbox" />
                <span aria-hidden="true" className="timer-picker-dot" style={{ backgroundColor: tag.color }} />
                <span>{tag.name}</span>
              </label>
            ))}
            {activeTags.length === 0 && !canCreate ? <p className="timer-picker-empty">{t('noTags')}</p> : null}
          </div>
          <div className="timer-picker-footer">
            <button
              className="timer-picker-create"
              disabled={!canCreate || createTagMutation.isPending}
              onClick={handleCreateTag}
              type="button"
            >
              <Plus aria-hidden="true" />
              {t('timerCreateTag')}
            </button>
          </div>
        </div>
      ) : null}
    </div>
  );
}

export function TimerBillableToggle({
  billable,
  onChange,
  t,
}: {
  billable: boolean;
  onChange: (billable: boolean) => void;
  t: Translator;
}) {
  return (
    <button
      aria-pressed={billable}
      className={`quiet-icon-button${billable ? ' billable' : ''}`}
      onClick={() => onChange(!billable)}
      type="button"
      title={billable ? t('billable') : t('nonBillable')}
    >
      <DollarSign aria-hidden="true" />
    </button>
  );
}

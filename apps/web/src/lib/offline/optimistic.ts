import type {
  Client,
  ClientInput,
  Project,
  ProjectInput,
  Tag,
  TagInput,
  Task,
  TaskInput,
  TimeEntry,
  TimeEntryInput,
  TimerStartInput,
} from '../api';

export type EntityLookup = {
  clients?: Client[];
  projects?: Project[];
  tasks?: Task[];
  tags?: Tag[];
};

function nowIso(): string {
  return new Date().toISOString();
}

function lookupProject(projectId: string, lookup: EntityLookup) {
  return lookup.projects?.find((project) => project.id === projectId);
}

function lookupTask(taskId: string, lookup: EntityLookup) {
  return lookup.tasks?.find((task) => task.id === taskId);
}

function lookupClient(clientId: string, lookup: EntityLookup) {
  return lookup.clients?.find((client) => client.id === clientId);
}

function lookupTags(tagIds: string[], lookup: EntityLookup) {
  return (lookup.tags ?? []).filter((tag) => tagIds.includes(tag.id)).map((tag) => ({
    id: tag.id,
    name: tag.name,
    color: tag.color,
  }));
}

function durationSeconds(startedAt: string, endedAt: string): number {
  const started = Date.parse(startedAt);
  const ended = Date.parse(endedAt);
  if (Number.isNaN(started) || Number.isNaN(ended) || ended <= started) {
    return 0;
  }
  return Math.floor((ended - started) / 1000);
}

export function buildOptimisticClient(localId: string, input: ClientInput): Client {
  const now = nowIso();
  return {
    id: localId,
    name: input.name,
    email: input.email,
    taxId: input.taxId,
    billingAddress: input.billingAddress,
    defaultCurrency: input.defaultCurrency,
    defaultHourlyRateMinor: input.defaultHourlyRateMinor,
    archivedAt: '',
    createdAt: now,
    updatedAt: now,
  };
}

export function buildOptimisticProject(localId: string, input: ProjectInput, lookup: EntityLookup): Project {
  const now = nowIso();
  const client = lookupClient(input.clientId, lookup);
  return {
    id: localId,
    clientId: input.clientId,
    clientName: client?.name ?? '',
    name: input.name,
    color: input.color,
    defaultHourlyRateMinor: input.defaultHourlyRateMinor,
    archivedAt: '',
    createdAt: now,
    updatedAt: now,
  };
}

export function buildOptimisticTask(localId: string, input: TaskInput, lookup: EntityLookup): Task {
  const now = nowIso();
  const project = lookupProject(input.projectId, lookup);
  return {
    id: localId,
    projectId: input.projectId,
    projectName: project?.name ?? '',
    projectColor: project?.color ?? '#64748b',
    name: input.name,
    billable: input.billable,
    archivedAt: '',
    createdAt: now,
    updatedAt: now,
  };
}

export function buildOptimisticTag(localId: string, input: TagInput): Tag {
  const now = nowIso();
  return {
    id: localId,
    name: input.name,
    color: input.color,
    archivedAt: '',
    createdAt: now,
    updatedAt: now,
  };
}

export function buildOptimisticTimeEntry(
  localId: string,
  input: TimeEntryInput,
  lookup: EntityLookup,
  overrides: Partial<TimeEntry> = {},
): TimeEntry {
  const now = nowIso();
  const project = lookupProject(input.projectId, lookup);
  const task = lookupTask(input.taskId, lookup);
  const client = lookupClient(input.clientId, lookup);
  const endedAt = input.endedAt || '';
  return {
    id: localId,
    clientId: input.clientId,
    clientName: client?.name ?? '',
    projectId: input.projectId,
    projectName: project?.name ?? '',
    projectColor: project?.color ?? '#64748b',
    taskId: input.taskId,
    taskName: task?.name ?? '',
    description: input.description,
    startedAt: input.startedAt,
    endedAt,
    durationSeconds: endedAt ? durationSeconds(input.startedAt, endedAt) : 0,
    billable: input.billable,
    overlapWarning: false,
    source: 'offline',
    tags: lookupTags(input.tagIds, lookup),
    createdAt: now,
    updatedAt: now,
    ...overrides,
  };
}

export function buildOptimisticTimer(localId: string, input: TimerStartInput, lookup: EntityLookup): TimeEntry {
  const startedAt = input.startedAt ?? nowIso();
  return buildOptimisticTimeEntry(
    localId,
    {
      clientId: input.clientId,
      projectId: input.projectId,
      taskId: input.taskId,
      tagIds: input.tagIds,
      description: input.description,
      startedAt,
      endedAt: '',
      billable: input.billable,
    },
    lookup,
    { source: 'timer' },
  );
}

export function buildOptimisticStoppedTimer(timer: TimeEntry, endedAt = nowIso()): TimeEntry {
  return {
    ...timer,
    endedAt,
    durationSeconds: durationSeconds(timer.startedAt, endedAt),
    updatedAt: endedAt,
  };
}

export function mergeTimeEntryUpdate(existing: TimeEntry, input: TimeEntryInput, lookup: EntityLookup): TimeEntry {
  return buildOptimisticTimeEntry(existing.id, input, lookup, {
    createdAt: existing.createdAt,
    overlapWarning: existing.overlapWarning,
    source: existing.source,
  });
}

export function mergeTimerUpdate(existing: TimeEntry, input: TimerStartInput, lookup: EntityLookup): TimeEntry {
  return buildOptimisticTimer(existing.id, { ...input, startedAt: input.startedAt ?? existing.startedAt }, lookup);
}

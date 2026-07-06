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
import {
  createClient as apiCreateClient,
  createProject as apiCreateProject,
  createTag as apiCreateTag,
  createTask as apiCreateTask,
  createTimeEntry as apiCreateTimeEntry,
  startTimer as apiStartTimer,
  stopTimer as apiStopTimer,
  updateTimeEntry as apiUpdateTimeEntry,
  updateTimer as apiUpdateTimer,
} from '../api';
import {
  buildOptimisticClient,
  buildOptimisticProject,
  buildOptimisticStoppedTimer,
  buildOptimisticTag,
  buildOptimisticTask,
  buildOptimisticTimeEntry,
  buildOptimisticTimer,
  mergeTimeEntryUpdate,
  mergeTimerUpdate,
  type EntityLookup,
} from './optimistic';
import { isNetworkFailure, isOnline } from './network';
import {
  createLocalId,
  enqueueMutation,
  isLocalId,
  remapTimeEntryInput,
  remapTimerInput,
  resolveEntityId,
} from './sync';

export { createLocalId, isLocalId, pendingMutationCount, flushOfflineQueue } from './sync';
export type { EntityLookup } from './optimistic';

async function runOrQueue<T>(onlineAction: () => Promise<T>, queueAction: () => Promise<T>): Promise<T> {
  if (!isOnline()) {
    return queueAction();
  }
  try {
    return await onlineAction();
  } catch (error) {
    if (isNetworkFailure(error)) {
      return queueAction();
    }
    throw error;
  }
}

function hasUnresolvedLocalIds(input: TimeEntryInput | TimerStartInput): boolean {
  if (input.clientId && isLocalId(input.clientId)) {
    return true;
  }
  if (input.projectId && isLocalId(input.projectId)) {
    return true;
  }
  if (input.taskId && isLocalId(input.taskId)) {
    return true;
  }
  return input.tagIds.some((tagId) => isLocalId(tagId));
}

export async function createClient(input: ClientInput): Promise<Client> {
  return runOrQueue(
    () => apiCreateClient(input),
    async () => {
      const localId = createLocalId('cli');
      await enqueueMutation({ operation: 'createClient', localId, payload: input });
      return buildOptimisticClient(localId, input);
    },
  );
}

export async function createProject(input: ProjectInput, lookup: EntityLookup = {}): Promise<Project> {
  return runOrQueue(
    () => apiCreateProject(input),
    async () => {
      const localId = createLocalId('prj');
      await enqueueMutation({ operation: 'createProject', localId, payload: input });
      return buildOptimisticProject(localId, input, lookup);
    },
  );
}

export async function createTask(input: TaskInput, lookup: EntityLookup = {}): Promise<Task> {
  return runOrQueue(
    () => apiCreateTask(input),
    async () => {
      const localId = createLocalId('tsk');
      await enqueueMutation({ operation: 'createTask', localId, payload: input });
      return buildOptimisticTask(localId, input, lookup);
    },
  );
}

export async function createTag(input: TagInput): Promise<Tag> {
  return runOrQueue(
    () => apiCreateTag(input),
    async () => {
      const localId = createLocalId('tag');
      await enqueueMutation({ operation: 'createTag', localId, payload: input });
      return buildOptimisticTag(localId, input);
    },
  );
}

export async function createTimeEntry(input: TimeEntryInput, lookup: EntityLookup = {}): Promise<TimeEntry> {
  return runOrQueue(
    () => apiCreateTimeEntry(input),
    async () => {
      const localId = createLocalId('te');
      await enqueueMutation({ operation: 'createTimeEntry', localId, payload: input });
      return buildOptimisticTimeEntry(localId, input, lookup);
    },
  );
}

export async function updateTimeEntry(
  timeEntryId: string,
  input: TimeEntryInput,
  lookup: EntityLookup = {},
  existing?: TimeEntry,
): Promise<TimeEntry> {
  const queueUpdate = async () => {
    await enqueueMutation({
      operation: 'updateTimeEntry',
      entityId: timeEntryId,
      payload: input,
    });
    if (existing) {
      return mergeTimeEntryUpdate(existing, input, lookup);
    }
    return buildOptimisticTimeEntry(timeEntryId, input, lookup);
  };

  const resolvedTimeEntryId = await resolveEntityId(timeEntryId);
  const remappedInput = await remapTimeEntryInput(input);
  if (isLocalId(resolvedTimeEntryId) || hasUnresolvedLocalIds(remappedInput)) {
    return queueUpdate();
  }

  return runOrQueue(
    () => apiUpdateTimeEntry(resolvedTimeEntryId, remappedInput),
    queueUpdate,
  );
}

export async function startTimer(input: TimerStartInput, lookup: EntityLookup = {}): Promise<TimeEntry> {
  return runOrQueue(
    () => apiStartTimer(input),
    async () => {
      const localId = createLocalId('te');
      await enqueueMutation({ operation: 'startTimer', localId, payload: input });
      return buildOptimisticTimer(localId, input, lookup);
    },
  );
}

export async function updateTimer(
  timeEntryId: string,
  input: TimerStartInput,
  lookup: EntityLookup = {},
  existing?: TimeEntry,
): Promise<TimeEntry> {
  const queueUpdate = async () => {
    await enqueueMutation({
      operation: 'updateTimer',
      entityId: timeEntryId,
      payload: input,
    });
    if (existing) {
      return mergeTimerUpdate(existing, input, lookup);
    }
    return buildOptimisticTimer(timeEntryId, input, lookup);
  };

  const resolvedTimeEntryId = await resolveEntityId(timeEntryId);
  const remappedInput = await remapTimerInput(input);
  if (isLocalId(resolvedTimeEntryId) || hasUnresolvedLocalIds(remappedInput)) {
    return queueUpdate();
  }

  return runOrQueue(
    () => apiUpdateTimer(resolvedTimeEntryId, remappedInput),
    queueUpdate,
  );
}

export async function stopTimer(timeEntryId: string, existing?: TimeEntry): Promise<TimeEntry> {
  const queueStop = async () => {
    await enqueueMutation({
      operation: 'stopTimer',
      entityId: timeEntryId,
      payload: {},
    });
    if (!existing) {
      throw new Error('timer context required for offline stop');
    }
    return buildOptimisticStoppedTimer(existing);
  };

  const resolvedTimeEntryId = await resolveEntityId(timeEntryId);
  if (isLocalId(resolvedTimeEntryId)) {
    return queueStop();
  }

  return runOrQueue(() => apiStopTimer(resolvedTimeEntryId), queueStop);
}

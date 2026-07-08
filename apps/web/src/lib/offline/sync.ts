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
  type ClientInput,
  type ProjectInput,
  type TagInput,
  type TaskInput,
  type TimeEntryInput,
  type TimerStartInput,
} from '../api';
import { deleteQueuedMutation, getServerId, listQueuedMutations, putQueuedMutation, setServerId } from './db';
import type { OfflineOperationType, QueuedMutation } from './types';

export function createLocalId(prefix: string): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return `local_${prefix}_${crypto.randomUUID()}`;
  }
  return `local_${prefix}_${Date.now()}_${Math.random().toString(16).slice(2)}`;
}

export function isLocalId(id: string): boolean {
  return id.startsWith('local_');
}

export async function resolveEntityId(entityId: string): Promise<string> {
  if (!isLocalId(entityId)) {
    return entityId;
  }
  const mapped = await getServerId(entityId);
  return mapped ?? entityId;
}

export async function enqueueMutation(input: {
  operation: OfflineOperationType;
  localId?: string;
  entityId?: string;
  payload: Record<string, unknown>;
}): Promise<QueuedMutation> {
  const mutation: QueuedMutation = {
    id: createLocalId('mq'),
    operation: input.operation,
    localId: input.localId,
    entityId: input.entityId,
    payload: input.payload,
    createdAt: new Date().toISOString(),
    retryCount: 0,
    lastError: '',
  };
  await putQueuedMutation(mutation);
  return mutation;
}

export async function pendingMutationCount(): Promise<number> {
  const mutations = await listQueuedMutations();
  return mutations.length;
}

async function processMutation(mutation: QueuedMutation): Promise<void> {
  switch (mutation.operation) {
    case 'createClient': {
      const created = await apiCreateClient(mutation.payload as ClientInput);
      if (mutation.localId) {
        await setServerId(mutation.localId, created.id);
      }
      return;
    }
    case 'createProject': {
      const created = await apiCreateProject(await remapProjectInput(mutation.payload as ProjectInput));
      if (mutation.localId) {
        await setServerId(mutation.localId, created.id);
      }
      return;
    }
    case 'createTask': {
      const created = await apiCreateTask(await remapTaskInput(mutation.payload as TaskInput));
      if (mutation.localId) {
        await setServerId(mutation.localId, created.id);
      }
      return;
    }
    case 'createTag': {
      const created = await apiCreateTag(mutation.payload as TagInput);
      if (mutation.localId) {
        await setServerId(mutation.localId, created.id);
      }
      return;
    }
    case 'createTimeEntry': {
      const payload = await remapTimeEntryInput(mutation.payload as TimeEntryInput);
      const created = await apiCreateTimeEntry(payload);
      if (mutation.localId) {
        await setServerId(mutation.localId, created.id);
      }
      return;
    }
    case 'updateTimeEntry': {
      const entityId = await resolveEntityId(mutation.entityId ?? '');
      const payload = await remapTimeEntryInput(mutation.payload as TimeEntryInput);
      await apiUpdateTimeEntry(entityId, payload);
      return;
    }
    case 'startTimer': {
      const payload = await remapTimerInput(mutation.payload as TimerStartInput);
      const created = await apiStartTimer(payload);
      if (mutation.localId) {
        await setServerId(mutation.localId, created.id);
      }
      return;
    }
    case 'updateTimer': {
      const entityId = await resolveEntityId(mutation.entityId ?? '');
      const payload = await remapTimerInput(mutation.payload as TimerStartInput);
      await apiUpdateTimer(entityId, payload);
      return;
    }
    case 'stopTimer': {
      const entityId = await resolveEntityId(mutation.entityId ?? '');
      await apiStopTimer(entityId);
      return;
    }
    default:
      throw new Error(`unsupported offline operation: ${mutation.operation as string}`);
  }
}

export async function remapProjectInput(input: ProjectInput): Promise<ProjectInput> {
  return {
    ...input,
    clientId: input.clientId ? await resolveEntityId(input.clientId) : input.clientId,
  };
}

export async function remapTaskInput(input: TaskInput): Promise<TaskInput> {
  return {
    ...input,
    projectId: input.projectId ? await resolveEntityId(input.projectId) : input.projectId,
  };
}

export async function remapTimeEntryInput(input: TimeEntryInput): Promise<TimeEntryInput> {
  return {
    ...input,
    clientId: input.clientId ? await resolveEntityId(input.clientId) : input.clientId,
    projectId: input.projectId ? await resolveEntityId(input.projectId) : input.projectId,
    taskId: input.taskId ? await resolveEntityId(input.taskId) : input.taskId,
    tagIds: await Promise.all(input.tagIds.map((tagId) => resolveEntityId(tagId))),
  };
}

export async function remapTimerInput(input: TimerStartInput): Promise<TimerStartInput> {
  return {
    ...input,
    clientId: input.clientId ? await resolveEntityId(input.clientId) : input.clientId,
    projectId: input.projectId ? await resolveEntityId(input.projectId) : input.projectId,
    taskId: input.taskId ? await resolveEntityId(input.taskId) : input.taskId,
    tagIds: await Promise.all(input.tagIds.map((tagId) => resolveEntityId(tagId))),
  };
}

export type FlushOfflineQueueResult = {
  synced: number;
  failed: number;
  lastError: string;
};

export async function flushOfflineQueue(): Promise<FlushOfflineQueueResult> {
  const mutations = await listQueuedMutations();
  let synced = 0;
  let failed = 0;
  let lastError = '';

  for (const mutation of mutations) {
    try {
      await processMutation(mutation);
      await deleteQueuedMutation(mutation.id);
      synced += 1;
    } catch (error) {
      failed += 1;
      lastError = error instanceof Error ? error.message : 'sync failed';
      await putQueuedMutation({
        ...mutation,
        retryCount: mutation.retryCount + 1,
        lastError,
      });
      break;
    }
  }

  return { synced, failed, lastError };
}

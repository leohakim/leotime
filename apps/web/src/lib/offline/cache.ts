import type { QueryClient } from '@tanstack/react-query';
import type { Client, Project, Tag, Task, TimeEntry, TimersResponse } from '../api';
import { isOnline } from './network';

export function prependEntity<T extends { id: string }>(items: T[] | undefined, entity: T): T[] {
  const filtered = (items ?? []).filter((item) => item.id !== entity.id);
  return [entity, ...filtered];
}

export function upsertEntity<T extends { id: string }>(items: T[] | undefined, entity: T): T[] {
  return prependEntity(items, entity);
}

export function patchClientsCache(queryClient: QueryClient, client: Client) {
  queryClient.setQueryData<{ clients: Client[] }>(['clients'], (current) => ({
    clients: upsertEntity(current?.clients, client),
  }));
}

export function patchProjectsCache(queryClient: QueryClient, project: Project) {
  queryClient.setQueryData<{ projects: Project[] }>(['projects'], (current) => ({
    projects: upsertEntity(current?.projects, project),
  }));
}

export function patchTasksCache(queryClient: QueryClient, task: Task) {
  queryClient.setQueryData<{ tasks: Task[] }>(['tasks'], (current) => ({
    tasks: upsertEntity(current?.tasks, task),
  }));
}

export function patchTagsCache(queryClient: QueryClient, tag: Tag) {
  queryClient.setQueryData<{ tags: Tag[] }>(['tags'], (current) => ({
    tags: upsertEntity(current?.tags, tag),
  }));
}

export function patchTimeEntriesCache(queryClient: QueryClient, entry: TimeEntry) {
  queryClient.setQueriesData<{ timeEntries: TimeEntry[] }>({ queryKey: ['time-entries'] }, (current) => {
    if (!current) {
      return current;
    }
    return {
      timeEntries: upsertEntity(current.timeEntries, entry),
    };
  });
}

export function patchTimersCache(queryClient: QueryClient, timer: TimeEntry) {
  queryClient.setQueryData<TimersResponse>(['timers'], (current) => ({
    timers: upsertEntity(current?.timers, timer),
  }));
}

export function removeTimerFromCache(queryClient: QueryClient, timerId: string) {
  queryClient.setQueryData<TimersResponse>(['timers'], (current) => ({
    timers: (current?.timers ?? []).filter((timer) => timer.id !== timerId),
  }));
}

export async function invalidateOrKeepLocal(queryClient: QueryClient, queryKey: string[]) {
  if (isOnline()) {
    await queryClient.invalidateQueries({ queryKey });
    return;
  }
}

export async function refreshOverviewIfOnline(queryClient: QueryClient) {
  if (isOnline()) {
    await queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] });
  }
}

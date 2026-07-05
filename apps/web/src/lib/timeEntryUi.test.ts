import { describe, expect, test } from 'vitest';
import type { Project, Task } from './api';
import { applyManualEntryFieldUpdate, tasksForManualEntryForm } from './timeEntryUi';

const projects: Project[] = [
  {
    id: 'prj_1',
    clientId: 'cli_1',
    clientName: 'ACME',
    name: 'Portal Web',
    color: '#2563eb',
    defaultHourlyRateMinor: null,
    archivedAt: '',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
  },
  {
    id: 'prj_2',
    clientId: 'cli_1',
    clientName: 'ACME',
    name: 'ENACT',
    color: '#f97316',
    defaultHourlyRateMinor: null,
    archivedAt: '',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
  },
];

const tasks: Task[] = [
  {
    id: 'tsk_1',
    projectId: 'prj_1',
    projectName: 'Portal Web',
    projectColor: '#2563eb',
    name: 'Refactor API',
    billable: true,
    archivedAt: '',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
  },
  {
    id: 'tsk_2',
    projectId: 'prj_2',
    projectName: 'ENACT',
    projectColor: '#f97316',
    name: 'Deep Work',
    billable: true,
    archivedAt: '',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
  },
];

const baseForm = {
  clientId: '',
  projectId: 'prj_2',
  taskId: '',
  tagIds: [] as string[],
  description: 'Old entry',
  startedAt: '2026-07-01T08:00',
  endedAt: '2026-07-01T09:00',
  billable: true,
};

describe('tasksForManualEntryForm', () => {
  test('keeps the selected task visible even when it belongs to another project filter', () => {
    const options = tasksForManualEntryForm('prj_2', 'tsk_1', tasks);
    expect(options.map((task) => task.id)).toEqual(['tsk_2', 'tsk_1']);
  });
});

describe('applyManualEntryFieldUpdate', () => {
  test('assigns project and client when a task is selected', () => {
    const next = applyManualEntryFieldUpdate(baseForm, 'taskId', 'tsk_1', projects, tasks);
    expect(next.taskId).toBe('tsk_1');
    expect(next.projectId).toBe('prj_1');
    expect(next.clientId).toBe('cli_1');
  });

  test('clears task when project changes to an incompatible project', () => {
    const withTask = applyManualEntryFieldUpdate(baseForm, 'taskId', 'tsk_1', projects, tasks);
    const next = applyManualEntryFieldUpdate(withTask, 'projectId', 'prj_2', projects, tasks);
    expect(next.projectId).toBe('prj_2');
    expect(next.taskId).toBe('');
  });
});

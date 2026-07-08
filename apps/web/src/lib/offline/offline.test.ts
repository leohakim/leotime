import { describe, expect, test } from 'vitest';
import { buildOptimisticTimeEntry, buildOptimisticTimer } from './optimistic';
import { createLocalId, isLocalId, remapProjectInput, remapTaskInput } from './sync';
import { resetOfflineStorageForTests, setServerId } from './db';

describe('offline helpers', () => {
  test('creates stable local id prefix', () => {
    const id = createLocalId('te');
    expect(isLocalId(id)).toBe(true);
    expect(id.startsWith('local_te_')).toBe(true);
  });

  test('builds optimistic timer without end time', () => {
    const timer = buildOptimisticTimer(
      'local_te_1',
      {
        clientId: '',
        projectId: 'prj_1',
        taskId: '',
        tagIds: [],
        description: 'Design',
        billable: true,
      },
      {
        projects: [
          {
            id: 'prj_1',
            clientId: 'cli_1',
            clientName: 'Acme',
            name: 'Portal',
            color: '#2563eb',
            defaultHourlyRateMinor: null,
            archivedAt: '',
            createdAt: '2026-01-01T00:00:00.000Z',
            updatedAt: '2026-01-01T00:00:00.000Z',
          },
        ],
      },
    );

    expect(timer.projectName).toBe('Portal');
    expect(timer.endedAt).toBe('');
    expect(timer.source).toBe('timer');
  });

  test('builds optimistic manual entry duration', () => {
    const entry = buildOptimisticTimeEntry(
      'local_te_2',
      {
        clientId: '',
        projectId: '',
        taskId: '',
        tagIds: [],
        description: 'Review',
        startedAt: '2026-07-06T08:00:00.000Z',
        endedAt: '2026-07-06T09:00:00.000Z',
        billable: true,
      },
      {},
    );

    expect(entry.durationSeconds).toBe(3600);
    expect(entry.source).toBe('offline');
  });

  test('remaps local client id when building project input for sync', async () => {
    resetOfflineStorageForTests();
    await setServerId('local_cli_1', 'cli_server_1');

    const remapped = await remapProjectInput({
      clientId: 'local_cli_1',
      name: 'Portal',
      color: '#2563eb',
      defaultHourlyRateMinor: null,
    });

    expect(remapped.clientId).toBe('cli_server_1');
  });

  test('remaps local project id when building task input for sync', async () => {
    resetOfflineStorageForTests();
    await setServerId('local_prj_1', 'prj_server_1');

    const remapped = await remapTaskInput({
      projectId: 'local_prj_1',
      name: 'Support',
      billable: true,
    });

    expect(remapped.projectId).toBe('prj_server_1');
  });
});

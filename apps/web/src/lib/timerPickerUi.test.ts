import { describe, expect, test } from 'vitest';
import { groupProjectsForPicker } from './timerPickerUi';
import type { Project } from './api';

function project(overrides: Partial<Project> = {}): Project {
  return {
    id: 'p1',
    clientId: 'c1',
    clientName: 'Acme',
    name: 'Portal',
    color: '#2563eb',
    defaultHourlyRateMinor: null,
    archivedAt: '',
    createdAt: '2026-01-01T00:00:00.000Z',
    updatedAt: '2026-01-01T00:00:00.000Z',
    ...overrides,
  };
}

describe('groupProjectsForPicker', () => {
  test('groups active projects by client name', () => {
    const groups = groupProjectsForPicker(
      [
        project({ id: 'p1', clientId: 'c1', clientName: 'Beta', name: 'Beta Project' }),
        project({ id: 'p2', clientId: 'c2', clientName: 'Alpha', name: 'Alpha Project' }),
      ],
      '',
    );

    expect(groups.map((group) => group.clientLabel)).toEqual(['Alpha', 'Beta']);
    expect(groups[0]?.projects.map((item) => item.name)).toEqual(['Alpha Project']);
  });

  test('filters by project or client name', () => {
    const groups = groupProjectsForPicker(
      [
        project({ id: 'p1', clientName: 'Acme', name: 'Portal' }),
        project({ id: 'p2', clientId: 'c2', clientName: 'Other', name: 'Ops' }),
      ],
      'acme',
    );

    expect(groups).toHaveLength(1);
    expect(groups[0]?.projects).toHaveLength(1);
    expect(groups[0]?.projects[0]?.name).toBe('Portal');
  });

  test('skips archived projects', () => {
    const groups = groupProjectsForPicker([project({ archivedAt: '2026-01-02T00:00:00.000Z' })], '');

    expect(groups).toHaveLength(0);
  });
});

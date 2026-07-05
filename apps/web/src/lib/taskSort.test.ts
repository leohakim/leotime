import { describe, expect, test } from 'vitest';
import type { Task } from './api';
import { sortTasksByNewest } from './taskSort';

function task(id: string, createdAt: string): Task {
  return {
    id,
    projectId: '',
    projectName: '',
    projectColor: '',
    name: id,
    billable: true,
    archivedAt: '',
    createdAt,
    updatedAt: createdAt,
  };
}

describe('sortTasksByNewest', () => {
  test('sorts by createdAt descending', () => {
    const sorted = sortTasksByNewest([
      task('tsk_old', '2026-01-01T00:00:00Z'),
      task('tsk_new', '2026-06-01T00:00:00Z'),
      task('tsk_mid', '2026-03-01T00:00:00Z'),
    ]);

    expect(sorted.map((item) => item.id)).toEqual(['tsk_new', 'tsk_mid', 'tsk_old']);
  });
});

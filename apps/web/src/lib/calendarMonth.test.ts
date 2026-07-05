import { describe, expect, test } from 'vitest';
import type { TimeEntry } from './api';
import { buildMonthGrid, formatMonthLabel, startOfMonth, toMonthQueryFrom, toMonthQueryTo } from './calendarMonth';

function entry(startedAt: string, durationSeconds: number): TimeEntry {
  return {
    id: startedAt,
    clientId: '',
    clientName: '',
    projectId: '',
    projectName: '',
    projectColor: '',
    taskId: '',
    taskName: '',
    description: 'Work',
    startedAt,
    endedAt: startedAt,
    durationSeconds,
    billable: true,
    overlapWarning: false,
    source: 'manual',
    tags: [],
    createdAt: startedAt,
    updatedAt: startedAt,
  };
}

describe('calendarMonth', () => {
  test('builds a month grid with padding weeks', () => {
    const monthStart = startOfMonth(new Date(2026, 6, 1));
    const cells = buildMonthGrid(monthStart, []);
    expect(cells.length % 7).toBe(0);
    expect(cells.some((cell) => cell.date === '2026-07-01' && cell.inMonth)).toBe(true);
    expect(cells.some((cell) => cell.date === '2026-06-29' && !cell.inMonth)).toBe(true);
  });

  test('groups entries on local days newest first', () => {
    const monthStart = startOfMonth(new Date(2026, 6, 1));
    const cells = buildMonthGrid(
      monthStart,
      [
        entry('2026-07-05T08:00:00.000Z', 1800),
        entry('2026-07-05T18:00:00.000Z', 3600),
      ],
    );
    const day = cells.find((cell) => cell.date === '2026-07-05');
    expect(day?.totalSeconds).toBe(5400);
    expect(day?.entries.map((item) => item.startedAt)).toEqual([
      '2026-07-05T18:00:00.000Z',
      '2026-07-05T08:00:00.000Z',
    ]);
  });

  test('formats month label and query bounds', () => {
    const monthStart = startOfMonth(new Date(2026, 6, 1));
    const monthEnd = new Date(2026, 6, 31);
    expect(formatMonthLabel(monthStart, 'es')).toContain('2026');
    expect(toMonthQueryFrom(monthStart)).toBe(new Date(2026, 6, 1).toISOString());
    expect(toMonthQueryTo(monthEnd)).toBe(new Date(2026, 6, 31, 23, 59, 59, 999).toISOString());
  });
});

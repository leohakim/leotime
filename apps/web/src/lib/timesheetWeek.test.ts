import { describe, expect, test } from 'vitest';
import type { TimeEntry } from './api';
import {
  addWeeks,
  endOfWeek,
  formatWeekRange,
  groupTimeEntriesByWeek,
  isSameWeek,
  MANUAL_ENTRY_DIRECTORY_DAYS,
  MANUAL_ENTRY_DIRECTORY_PAGE_SIZE,
  manualEntryDirectoryRange,
  startOfWeek,
  toWeekQueryFrom,
  toWeekQueryTo,
} from './timesheetWeek';

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
    description: '',
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

describe('timesheetWeek', () => {
  test('starts weeks on Monday', () => {
    const weekStart = startOfWeek(new Date(2026, 6, 5));
    expect(weekStart.getDay()).toBe(1);
    expect(weekStart.getDate()).toBe(29);
    expect(weekStart.getMonth()).toBe(5);
  });

  test('builds seven day groups newest day first', () => {
    const weekStart = startOfWeek(new Date(2026, 6, 5));
    const days = groupTimeEntriesByWeek([], weekStart, 'es');
    expect(days).toHaveLength(7);
    expect(days.every((day) => day.entries.length === 0)).toBe(true);
    expect(days[0]?.date).toBe('2026-07-05');
    expect(days[6]?.date).toBe('2026-06-29');
  });

  test('groups entries by day and sorts newest entry first', () => {
    const weekStart = startOfWeek(new Date(2026, 6, 5));
    const days = groupTimeEntriesByWeek(
      [
        entry('2026-07-01T14:00:00.000Z', 3600),
        entry('2026-07-01T08:00:00.000Z', 1800),
        entry('2026-06-30T09:00:00.000Z', 900),
      ],
      weekStart,
      'en',
    );

    const wednesday = days.find((day) => day.date === '2026-07-01');
    expect(wednesday?.entries.map((item) => item.startedAt)).toEqual([
      '2026-07-01T14:00:00.000Z',
      '2026-07-01T08:00:00.000Z',
    ]);
    expect(wednesday?.totalSeconds).toBe(5400);
  });

  test('formats week range and query bounds', () => {
    const weekStart = startOfWeek(new Date(2026, 5, 29));
    const weekEnd = endOfWeek(weekStart);

    expect(formatWeekRange(weekStart, weekEnd, 'es')).toContain('29');
    expect(toWeekQueryFrom(weekStart)).toBe(
      new Date(weekStart.getFullYear(), weekStart.getMonth(), weekStart.getDate()).toISOString(),
    );
    expect(toWeekQueryTo(weekEnd)).toBe(
      new Date(weekEnd.getFullYear(), weekEnd.getMonth(), weekEnd.getDate(), 23, 59, 59, 999).toISOString(),
    );
  });

  test('detects same week', () => {
    expect(isSameWeek(new Date(2026, 6, 1), new Date(2026, 6, 5))).toBe(true);
    expect(isSameWeek(new Date(2026, 6, 1), new Date(2026, 6, 8))).toBe(false);
  });

  test('builds manual entry directory range for the last N days', () => {
    const now = new Date(2026, 6, 9, 15, 30, 0);
    const range = manualEntryDirectoryRange(now, MANUAL_ENTRY_DIRECTORY_DAYS);

    expect(range.from).toBe(new Date(2026, 3, 11).toISOString());
    expect(range.to).toBe(new Date(2026, 6, 9, 23, 59, 59, 999).toISOString());
    expect(MANUAL_ENTRY_DIRECTORY_PAGE_SIZE).toBe(25);
  });
});

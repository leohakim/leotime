import { describe, expect, test } from 'vitest';
import { currentMonthKey, groupHeatmapByWeek, isFutureMonth, shiftMonthKey, weekBarHeight, weekChartAxisTicks, weekChartPeak } from './dashboardHeatmap';

describe('groupHeatmapByWeek', () => {
  test('groups month grid days into week rows', () => {
    const days = [
      { date: '2026-06-30', totalSeconds: 0, level: 0, inMonth: false },
      { date: '2026-07-01', totalSeconds: 3600, level: 1, inMonth: true },
      { date: '2026-07-02', totalSeconds: 7200, level: 2, inMonth: true },
    ];

    const weeks = groupHeatmapByWeek(days);
    expect(weeks.length).toBeGreaterThan(0);
    expect(weeks[0].days.some((day) => day?.date === '2026-07-01')).toBe(true);
  });
});

describe('month navigation helpers', () => {
  test('shifts month keys', () => {
    expect(shiftMonthKey('2026-07', -1)).toBe('2026-06');
    expect(shiftMonthKey('2026-07', 1)).toBe('2026-08');
  });

  test('detects future months', () => {
    expect(isFutureMonth('2099-01', new Date('2026-07-05T12:00:00'))).toBe(true);
    expect(isFutureMonth(currentMonthKey(new Date('2026-07-05T12:00:00')), new Date('2026-07-05T12:00:00'))).toBe(false);
  });
});

describe('weekBarHeight', () => {
  test('keeps a visible minimum for non-zero totals', () => {
    expect(weekBarHeight(3600, 7200)).toBeGreaterThanOrEqual(12);
    expect(weekBarHeight(0, 7200)).toBe(0);
  });
});

describe('weekChartAxisTicks', () => {
  test('returns descending ticks from peak to zero', () => {
    expect(weekChartAxisTicks(7200)).toEqual([7200, 5400, 3600, 1800, 0]);
  });

  test('uses a default scale when there is no activity', () => {
    expect(weekChartAxisTicks(0)).toEqual([14400, 10800, 7200, 3600, 0]);
    expect(weekChartPeak(0)).toBe(14400);
  });
});

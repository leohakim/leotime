import type { DashboardHeatmapDay } from './api';

export type HeatmapCell = {
  date: string;
  level: number;
  totalSeconds: number;
  inMonth: boolean;
};

export type HeatmapWeekRow = {
  days: Array<HeatmapCell | null>;
};

export function groupHeatmapByWeek(days: DashboardHeatmapDay[]): HeatmapWeekRow[] {
  if (days.length === 0) {
    return [];
  }

  const byDate = new Map(days.map((day) => [day.date, day]));
  const firstDate = parseDay(days[0].date);
  const lastDate = parseDay(days[days.length - 1].date);
  let cursor = startOfWeekMonday(firstDate);
  const rows: HeatmapWeekRow[] = [];

  while (cursor.getTime() <= lastDate.getTime()) {
    const weekDays: Array<HeatmapCell | null> = [];
    for (let index = 0; index < 7; index += 1) {
      const current = addDays(cursor, index);
      const key = toDayKey(current);
      const day = byDate.get(key);
      if (!day) {
        weekDays.push(null);
        continue;
      }
      weekDays.push({
        date: key,
        level: day.level,
        totalSeconds: day.totalSeconds,
        inMonth: day.inMonth,
      });
    }
    rows.push({ days: weekDays });
    cursor = addDays(cursor, 7);
  }

  return rows;
}

export function currentMonthKey(date = new Date()): string {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`;
}

export function shiftMonthKey(monthKey: string, delta: number): string {
  const [year, month] = monthKey.split('-').map(Number);
  const shifted = new Date(year, month - 1 + delta, 1);
  return currentMonthKey(shifted);
}

export function isFutureMonth(monthKey: string, reference = new Date()): boolean {
  const [year, month] = monthKey.split('-').map(Number);
  const target = year * 12 + (month - 1);
  const current = reference.getFullYear() * 12 + reference.getMonth();
  return target > current;
}

function parseDay(value: string): Date {
  const [year, month, day] = value.split('-').map(Number);
  return new Date(year, month - 1, day);
}

function toDayKey(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

function addDays(date: Date, days: number): Date {
  const next = new Date(date.getFullYear(), date.getMonth(), date.getDate());
  next.setDate(next.getDate() + days);
  return next;
}

function startOfWeekMonday(date: Date): Date {
  const normalized = new Date(date.getFullYear(), date.getMonth(), date.getDate());
  const offset = (normalized.getDay() + 6) % 7;
  normalized.setDate(normalized.getDate() - offset);
  return normalized;
}

export function donutGradient(shares: Array<{ color: string; totalSeconds: number }>): string {
  const total = shares.reduce((sum, share) => sum + share.totalSeconds, 0);
  if (total <= 0) {
    return '#24262e';
  }

  let cursor = 0;
  const stops = shares.map((share) => {
    const start = (cursor / total) * 100;
    cursor += share.totalSeconds;
    const end = (cursor / total) * 100;
    return `${share.color} ${start}% ${end}%`;
  });
  return `conic-gradient(${stops.join(', ')})`;
}

export function weekBarHeight(totalSeconds: number, peak: number): number {
  if (totalSeconds <= 0 || peak <= 0) {
    return 0;
  }
  const ratio = totalSeconds / peak;
  return Math.max(12, Math.round(ratio * 100));
}

export function weekChartAxisTicks(peakSeconds: number, steps = 4): number[] {
  const max = peakSeconds > 0 ? peakSeconds : 4 * 3600;
  return Array.from({ length: steps + 1 }, (_, index) => Math.round((max * (steps - index)) / steps));
}

export function weekChartPeak(peakSeconds: number): number {
  return peakSeconds > 0 ? peakSeconds : 4 * 3600;
}

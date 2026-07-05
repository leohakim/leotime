import type { Locale, TimeEntry } from './api';
import { startOfWeek } from './timesheetWeek';

export type CalendarDayCell = {
  date: string;
  dayNumber: number;
  entries: TimeEntry[];
  inMonth: boolean;
  totalSeconds: number;
};

const DAY_MS = 24 * 60 * 60 * 1000;

export function startOfMonth(date: Date): Date {
  return new Date(date.getFullYear(), date.getMonth(), 1);
}

export function endOfMonth(monthStart: Date): Date {
  return new Date(monthStart.getFullYear(), monthStart.getMonth() + 1, 0);
}

export function addMonths(date: Date, months: number): Date {
  return new Date(date.getFullYear(), date.getMonth() + months, 1);
}

export function isSameMonth(a: Date, b: Date): boolean {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth();
}

export function isSameLocalDay(a: string, b: string): boolean {
  return a === b;
}

export function toMonthQueryFrom(monthStart: Date): string {
  return new Date(monthStart.getFullYear(), monthStart.getMonth(), 1).toISOString();
}

export function toMonthQueryTo(monthEnd: Date): string {
  return new Date(monthEnd.getFullYear(), monthEnd.getMonth(), monthEnd.getDate(), 23, 59, 59, 999).toISOString();
}

export function formatMonthLabel(monthStart: Date, locale: Locale): string {
  return monthStart.toLocaleDateString(locale === 'es' ? 'es-ES' : 'en-US', {
    month: 'long',
    year: 'numeric',
  });
}

export function weekdayLabels(locale: Locale): string[] {
  const formatter = new Intl.DateTimeFormat(locale === 'es' ? 'es-ES' : 'en-US', { weekday: 'short' });
  const monday = new Date(2026, 0, 5);
  return Array.from({ length: 7 }, (_, index) => formatter.format(new Date(monday.getTime() + index * DAY_MS)));
}

export function buildMonthGrid(monthStart: Date, entries: TimeEntry[]): CalendarDayCell[] {
  const monthEnd = endOfMonth(monthStart);
  const gridStart = startOfWeek(monthStart, 1);
  const gridEnd = endOfMonthWeek(monthEnd);
  const byDay = groupEntriesByLocalDay(entries);
  const cells: CalendarDayCell[] = [];

  for (let cursor = gridStart.getTime(); cursor <= gridEnd.getTime(); cursor += DAY_MS) {
    const dayDate = new Date(cursor);
    const date = toDayKey(dayDate);
    const dayEntries = (byDay.get(date) ?? []).sort(
      (left, right) => Date.parse(right.startedAt) - Date.parse(left.startedAt),
    );
    cells.push({
      date,
      dayNumber: dayDate.getDate(),
      inMonth: dayDate.getMonth() === monthStart.getMonth(),
      entries: dayEntries,
      totalSeconds: dayEntries.reduce((sum, entry) => sum + entry.durationSeconds, 0),
    });
  }

  return cells;
}

export function sumMonthSeconds(cells: CalendarDayCell[]): number {
  return cells.filter((cell) => cell.inMonth).reduce((sum, cell) => sum + cell.totalSeconds, 0);
}

export function defaultSelectedDay(monthStart: Date, today = new Date()): string {
  if (isSameMonth(monthStart, today)) {
    return toDayKey(today);
  }
  return '';
}

function endOfMonthWeek(monthEnd: Date): Date {
  const weekStart = startOfWeek(monthEnd, 1);
  const weekEnd = new Date(weekStart.getFullYear(), weekStart.getMonth(), weekStart.getDate() + 6);
  return weekEnd;
}

function groupEntriesByLocalDay(entries: TimeEntry[]): Map<string, TimeEntry[]> {
  const byDay = new Map<string, TimeEntry[]>();
  for (const entry of entries) {
    const dayKey = localDayKey(entry.startedAt);
    const current = byDay.get(dayKey) ?? [];
    current.push(entry);
    byDay.set(dayKey, current);
  }
  return byDay;
}

function toDayKey(date: Date): string {
  const pad = (value: number) => String(value).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
}

function localDayKey(iso: string): string {
  return toDayKey(new Date(iso));
}

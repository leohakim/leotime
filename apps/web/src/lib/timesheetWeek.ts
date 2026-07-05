import type { Locale, TimeEntry } from './api';

export type TimesheetDayGroup = {
  date: string;
  day: string;
  entries: TimeEntry[];
  totalSeconds: number;
};

const DAY_MS = 24 * 60 * 60 * 1000;

export function startOfWeek(date: Date, weekStartsOn: 0 | 1 = 1): Date {
  const normalized = new Date(date.getFullYear(), date.getMonth(), date.getDate());
  const weekday = normalized.getDay();
  const offset = (weekday - weekStartsOn + 7) % 7;
  normalized.setDate(normalized.getDate() - offset);
  return normalized;
}

export function endOfWeek(weekStart: Date): Date {
  const weekEnd = new Date(weekStart.getFullYear(), weekStart.getMonth(), weekStart.getDate());
  weekEnd.setDate(weekEnd.getDate() + 6);
  return weekEnd;
}

export function addWeeks(date: Date, weeks: number): Date {
  const next = new Date(date.getFullYear(), date.getMonth(), date.getDate());
  next.setDate(next.getDate() + weeks * 7);
  return next;
}

export function isSameWeek(a: Date, b: Date, weekStartsOn: 0 | 1 = 1): boolean {
  return startOfWeek(a, weekStartsOn).getTime() === startOfWeek(b, weekStartsOn).getTime();
}

export function toWeekQueryFrom(weekStart: Date): string {
  return new Date(weekStart.getFullYear(), weekStart.getMonth(), weekStart.getDate()).toISOString();
}

export function toWeekQueryTo(weekEnd: Date): string {
  return new Date(weekEnd.getFullYear(), weekEnd.getMonth(), weekEnd.getDate(), 23, 59, 59, 999).toISOString();
}

export function formatWeekRange(weekStart: Date, weekEnd: Date, locale: Locale): string {
  const formatter = new Intl.DateTimeFormat(locale === 'es' ? 'es-ES' : 'en-US', {
    day: 'numeric',
    month: 'short',
    year: weekStart.getFullYear() === weekEnd.getFullYear() ? undefined : 'numeric',
  });
  const startLabel = formatter.format(weekStart);
  const endLabel = formatter.format(weekEnd);
  return `${startLabel} – ${endLabel}`;
}

export function groupTimeEntriesByWeek(
  entries: TimeEntry[],
  weekStart: Date,
  locale: Locale,
): TimesheetDayGroup[] {
  const byDay = new Map<string, TimeEntry[]>();
  for (const entry of entries) {
    const dayKey = localDayKey(entry.startedAt);
    const current = byDay.get(dayKey) ?? [];
    current.push(entry);
    byDay.set(dayKey, current);
  }

  const days: TimesheetDayGroup[] = [];
  for (let index = 6; index >= 0; index -= 1) {
    const dayDate = new Date(weekStart.getTime() + index * DAY_MS);
    const date = toDayKey(dayDate);
    const dayEntries = (byDay.get(date) ?? []).sort(
      (left, right) => Date.parse(right.startedAt) - Date.parse(left.startedAt),
    );
    days.push({
      date,
      day: dayDate.toLocaleDateString(locale === 'es' ? 'es-ES' : 'en-US', { weekday: 'long' }),
      entries: dayEntries,
      totalSeconds: dayEntries.reduce((sum, entry) => sum + entry.durationSeconds, 0),
    });
  }

  return days;
}

export function sumWeekSeconds(days: TimesheetDayGroup[]): number {
  return days.reduce((sum, day) => sum + day.totalSeconds, 0);
}

function toDayKey(date: Date): string {
  const pad = (value: number) => String(value).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
}

function localDayKey(iso: string): string {
  return toDayKey(new Date(iso));
}

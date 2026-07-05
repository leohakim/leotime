import { ChevronLeft, ChevronRight } from 'lucide-react';
import { useEffect, useMemo } from 'react';
import type { Locale, Project, Task, TimeEntry } from './api';
import {
  buildMonthGrid,
  defaultSelectedDay,
  formatMonthLabel,
  isSameLocalDay,
  isSameMonth,
  sumMonthSeconds,
  weekdayLabels,
} from './calendarMonth';
import type { Translator } from './timeEntryUi';
import { formatDuration, TimesheetEntryRow } from './timeEntryUi';

export function CalendarPanel({
  entries,
  isLoading,
  locale,
  monthAnchor,
  onNextMonth,
  onPreviousMonth,
  onSelectDay,
  onTodayMonth,
  projects,
  selectedDay,
  tasks,
  t,
}: {
  entries: TimeEntry[];
  isLoading: boolean;
  locale: Locale;
  monthAnchor: Date;
  onNextMonth: () => void;
  onPreviousMonth: () => void;
  onSelectDay: (date: string) => void;
  onTodayMonth: () => void;
  projects: Project[];
  selectedDay: string;
  tasks: Task[];
  t: Translator;
}) {
  const monthStart = useMemo(() => new Date(monthAnchor.getFullYear(), monthAnchor.getMonth(), 1), [monthAnchor]);
  const cells = useMemo(() => buildMonthGrid(monthStart, entries), [entries, monthStart]);
  const monthTotalSeconds = useMemo(() => sumMonthSeconds(cells), [cells]);
  const viewingCurrentMonth = isSameMonth(monthStart, new Date());
  const todayKey = useMemo(() => defaultSelectedDay(monthStart), [monthStart]);
  const selectedCell = cells.find((cell) => isSameLocalDay(cell.date, selectedDay));
  const weekdays = useMemo(() => weekdayLabels(locale), [locale]);

  useEffect(() => {
    if (selectedDay && cells.some((cell) => cell.date === selectedDay)) {
      return;
    }
    onSelectDay(todayKey);
  }, [cells, onSelectDay, selectedDay, todayKey]);

  return (
    <section className="time-list-panel calendar-panel" id="calendar" aria-labelledby="calendar-title">
      <div className="time-list-toolbar calendar-toolbar">
        <div className="calendar-toolbar-spacer" aria-hidden="true" />

        <div className="week-nav" aria-label={t('calendar')}>
          <button className="ghost-button icon-button week-nav-button" onClick={onPreviousMonth} type="button" title={t('previousMonth')}>
            <ChevronLeft aria-hidden="true" />
            <span className="visually-hidden">{t('previousMonth')}</span>
          </button>
          <div className="week-nav-label">
            <strong id="calendar-title">{viewingCurrentMonth ? t('thisMonth') : t('calendar')}</strong>
            <span>{formatMonthLabel(monthStart, locale)}</span>
          </div>
          <button className="ghost-button icon-button week-nav-button" onClick={onNextMonth} type="button" title={t('nextMonth')}>
            <ChevronRight aria-hidden="true" />
            <span className="visually-hidden">{t('nextMonth')}</span>
          </button>
          {!viewingCurrentMonth ? (
            <button className="ghost-button week-today-button" onClick={onTodayMonth} type="button">
              {t('today')}
            </button>
          ) : null}
        </div>

        <div className="week-summary">
          <span>{t('monthTotal')}</span>
          <strong>{formatDuration(monthTotalSeconds)}</strong>
          {isLoading ? <span className="sync-pill">{t('loading')}</span> : null}
        </div>
      </div>

      <div className="calendar-grid" role="grid" aria-label={t('calendar')}>
        <div className="calendar-weekdays" role="row">
          {weekdays.map((label) => (
            <span className="calendar-weekday" key={label} role="columnheader">
              {label}
            </span>
          ))}
        </div>
        <div className="calendar-days">
          {cells.map((cell) => {
            const isSelected = selectedDay === cell.date;
            const isToday = cell.date === todayKey;
            return (
              <button
                aria-label={`${cell.dayNumber}, ${formatDuration(cell.totalSeconds)}`}
                aria-pressed={isSelected}
                className={`calendar-day${cell.inMonth ? '' : ' outside-month'}${isSelected ? ' selected' : ''}${isToday ? ' today' : ''}`}
                key={cell.date}
                onClick={() => onSelectDay(cell.date)}
                type="button"
              >
                <span className="calendar-day-number">{cell.dayNumber}</span>
                {cell.totalSeconds > 0 ? (
                  <span className="calendar-day-total">{formatDuration(cell.totalSeconds)}</span>
                ) : null}
                {cell.entries.length > 0 ? (
                  <span className="calendar-day-count">
                    {cell.entries.length} {cell.entries.length === 1 ? t('calendarEntry') : t('calendarEntries')}
                  </span>
                ) : null}
              </button>
            );
          })}
        </div>
      </div>

      <div className="calendar-day-detail" aria-live="polite">
        {selectedCell ? (
          <>
            <div className="day-group-header calendar-day-detail-header">
              <div>
                <strong>{selectedCell.dayNumber}</strong>
                <span>
                  {new Date(`${selectedCell.date}T12:00:00`).toLocaleDateString(locale === 'es' ? 'es-ES' : 'en-US', {
                    weekday: 'long',
                    day: 'numeric',
                    month: 'short',
                  })}
                </span>
              </div>
              <strong>{formatDuration(selectedCell.totalSeconds)}</strong>
            </div>
            {selectedCell.entries.length === 0 ? (
              <div className="empty-state calendar-empty-day">
                <p>{t('noEntriesThisDay')}</p>
              </div>
            ) : (
              <div className="time-entry-list" role="table" aria-label={t('calendarDayEntries')}>
                {selectedCell.entries.map((entry) => (
                  <TimesheetEntryRow entry={entry} key={entry.id} locale={locale} projects={projects} tasks={tasks} t={t} />
                ))}
              </div>
            )}
          </>
        ) : (
          <div className="empty-state calendar-empty-day">
            <p>{t('selectCalendarDay')}</p>
          </div>
        )}
      </div>
    </section>
  );
}

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { useMemo, useState } from 'react';
import {
  fetchDashboardStats,
  startTimer,
  type DashboardDaySummary,
  type DashboardRecentEntry,
  type DashboardStats,
  type Locale,
} from './api';
import { formatMonthLabel, startOfMonth, weekdayLabels } from './calendarMonth';
import {
  currentMonthKey,
  donutGradient,
  groupHeatmapByWeek,
  isFutureMonth,
  shiftMonthKey,
  weekBarHeight,
  weekChartAxisTicks,
  weekChartPeak,
} from './dashboardHeatmap';
import { formatMoneyMinor } from './invoiceUi';
import { TimerPlayIcon } from './timerIcons';
import type { Translator } from './timeEntryUi';
import { formatDuration } from './timeEntryUi';

function daySummaryLabel(label: string, t: Translator): string {
  switch (label) {
    case 'today':
      return t('today');
    case 'yesterday':
      return t('dashboardYesterday');
    default:
      return t('dashboardDaysAgo').replace('{count}', label.replace('d', ''));
  }
}

function maxSeconds(values: number[]): number {
  return values.reduce((max, value) => Math.max(max, value), 0);
}

function monthAnchorFromKey(monthKey: string): Date {
  const [year, month] = monthKey.split('-').map(Number);
  return startOfMonth(new Date(year, month - 1, 1));
}

function dayNumberFromKey(dateKey: string): number {
  return Number(dateKey.split('-')[2]);
}

function MiniBar({ ratio }: { ratio: number }) {
  return (
    <span aria-hidden="true" className="dashboard-mini-bar">
      <span style={{ width: `${Math.max(4, ratio * 100)}%` }} />
    </span>
  );
}

function RecentEntryRow({
  entry,
  onRestart,
  pending,
  t,
}: {
  entry: DashboardRecentEntry;
  onRestart: (entry: DashboardRecentEntry) => void;
  pending: boolean;
  t: Translator;
}) {
  return (
    <div className="dashboard-recent-row">
      <div>
        <strong>{entry.description || t('noDescription')}</strong>
        <span className="dashboard-recent-project">
          {entry.projectName ? (
            <>
              <span style={{ backgroundColor: entry.projectColor || '#64748b' }} aria-hidden="true" />
              {entry.projectName}
            </>
          ) : (
            t('taskProjectOptional')
          )}
        </span>
      </div>
      <button
        className="dashboard-restart-button"
        disabled={pending}
        onClick={() => onRestart(entry)}
        title={t('startTimer')}
        type="button"
      >
        <TimerPlayIcon className="timer-play-icon" />
        <span className="visually-hidden">{t('startTimer')}</span>
      </button>
    </div>
  );
}

function LastSevenDaysCard({ days, t }: { days: DashboardDaySummary[]; t: Translator }) {
  const peak = maxSeconds(days.map((day) => day.totalSeconds));

  return (
    <div className="dashboard-card">
      <h3>{t('dashboardLastSevenDays')}</h3>
      <div className="dashboard-seven-list">
        {days.map((day) => (
          <div className="dashboard-seven-row" key={day.date}>
            <span>{daySummaryLabel(day.label, t)}</span>
            <MiniBar ratio={peak > 0 ? day.totalSeconds / peak : 0} />
            <strong>{formatDuration(day.totalSeconds)}</strong>
          </div>
        ))}
      </div>
    </div>
  );
}

function ActivityGraphCard({
  activityMonth,
  heatmap,
  locale,
  onNextMonth,
  onPreviousMonth,
  t,
}: {
  activityMonth: string;
  heatmap: DashboardStats['activityHeatmap'];
  locale: Locale;
  onNextMonth: () => void;
  onPreviousMonth: () => void;
  t: Translator;
}) {
  const weeks = useMemo(() => groupHeatmapByWeek(heatmap), [heatmap]);
  const dayNames = useMemo(() => weekdayLabels(locale), [locale]);
  const monthLabel = formatMonthLabel(monthAnchorFromKey(activityMonth), locale);
  const canGoNext = !isFutureMonth(shiftMonthKey(activityMonth, 1));

  return (
    <div className="dashboard-card dashboard-activity-card">
      <div className="dashboard-card-header">
        <h3>{t('dashboardActivityGraph')}</h3>
        <div className="dashboard-month-nav">
          <button aria-label={t('previousMonth')} onClick={onPreviousMonth} type="button">
            <ChevronLeft aria-hidden="true" />
          </button>
          <span>{monthLabel}</span>
          <button aria-label={t('nextMonth')} disabled={!canGoNext} onClick={onNextMonth} type="button">
            <ChevronRight aria-hidden="true" />
          </button>
        </div>
      </div>

      <div className="dashboard-heatmap-calendar">
        <div aria-hidden="true" className="dashboard-heatmap-head">
          {dayNames.map((label) => (
            <span className="dashboard-heatmap-weekday" key={label}>
              {label.replace('.', '').slice(0, 2)}
            </span>
          ))}
        </div>
        {weeks.map((week, weekIndex) => (
          <div className="dashboard-heatmap-row" key={`week-${weekIndex}`}>
            {week.days.map((day, dayIndex) =>
              day ? (
                <span
                  className={`dashboard-heatmap-cell level-${day.inMonth ? day.level : 'out'}`}
                  key={day.date}
                  title={day.inMonth ? `${day.date}: ${formatDuration(day.totalSeconds)}` : undefined}
                >
                  <span className="dashboard-heatmap-day">{dayNumberFromKey(day.date)}</span>
                </span>
              ) : (
                <span className="dashboard-heatmap-cell empty" key={`empty-${weekIndex}-${dayIndex}`} />
              ),
            )}
          </div>
        ))}
        <div aria-hidden="true" className="dashboard-heatmap-legend">
          <span>{t('dashboardActivityLess')}</span>
          <span className="dashboard-heatmap-cell level-0" />
          <span className="dashboard-heatmap-cell level-1" />
          <span className="dashboard-heatmap-cell level-2" />
          <span className="dashboard-heatmap-cell level-3" />
          <span className="dashboard-heatmap-cell level-4" />
          <span>{t('dashboardActivityMore')}</span>
        </div>
      </div>
    </div>
  );
}

function WeekOverview({ locale, stats, t }: { locale: Locale; stats: DashboardStats; t: Translator }) {
  const peak = maxSeconds(stats.weekDays.map((day) => day.totalSeconds));
  const chartPeak = weekChartPeak(peak);
  const axisTicks = useMemo(() => weekChartAxisTicks(peak), [peak]);
  const donut = useMemo(
    () =>
      donutGradient(
        stats.projectBreakdown.map((share) => ({
          color: share.projectColor || '#64748b',
          totalSeconds: share.totalSeconds,
        })),
      ),
    [stats.projectBreakdown],
  );

  return (
    <div className="dashboard-week-layout">
      <div className="dashboard-card dashboard-week-chart-card">
        <h3>{t('dashboardThisWeek')}</h3>
        <div className="dashboard-week-chart-body">
          <div aria-hidden="true" className="dashboard-week-chart-axis">
            {axisTicks.map((tick) => (
              <span key={tick}>{formatDuration(tick)}</span>
            ))}
          </div>
          <div className="dashboard-week-bars">
            {stats.weekDays.map((day) => (
              <div className="dashboard-week-bar-column" key={day.date}>
                <div className="dashboard-week-bar-track">
                  <span
                    className="dashboard-week-bar-fill"
                    style={{ height: `${weekBarHeight(day.totalSeconds, chartPeak)}%` }}
                  />
                </div>
                <small>{day.weekday}</small>
              </div>
            ))}
          </div>
        </div>
      </div>

      <div className="dashboard-week-side">
        <div className="dashboard-stat-card">
          <span>{t('dashboardSpentTime')}</span>
          <strong>{formatDuration(stats.weekSpentSeconds)}</strong>
        </div>
        <div className="dashboard-stat-card">
          <span>{t('dashboardBillableTime')}</span>
          <strong>{formatDuration(stats.weekBillableSeconds)}</strong>
        </div>
        <div className="dashboard-stat-card">
          <span>{t('dashboardBillableAmount')}</span>
          <strong>{formatMoneyMinor(stats.weekBillableMinor, stats.weekCurrency, locale)}</strong>
        </div>
        <div className="dashboard-donut-wrap">
          <div aria-hidden="true" className="dashboard-donut" style={{ background: donut }} />
          <ul className="dashboard-donut-legend">
            {stats.projectBreakdown.map((share) => (
              <li key={share.projectId || share.projectName || 'none'}>
                <span style={{ backgroundColor: share.projectColor || '#64748b' }} />
                {share.projectName || t('taskProjectOptional')}
              </li>
            ))}
          </ul>
        </div>
      </div>
    </div>
  );
}

export function DashboardPanel({ locale, t }: { locale: Locale; t: Translator }) {
  const queryClient = useQueryClient();
  const [activityMonth, setActivityMonth] = useState(currentMonthKey);

  const statsQuery = useQuery({
    queryKey: ['dashboard-stats', activityMonth],
    queryFn: () => fetchDashboardStats(activityMonth),
    retry: false,
  });

  const restartMutation = useMutation({
    mutationFn: (entry: DashboardRecentEntry) =>
      startTimer({
        clientId: entry.clientId,
        projectId: entry.projectId,
        taskId: entry.taskId,
        description: entry.description,
        billable: entry.billable,
        tagIds: [],
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['timers'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] });
    },
  });

  const stats = statsQuery.data;

  return (
    <section className="dashboard-panel" id="dashboard" aria-labelledby="dashboard-title">
      <h2 className="visually-hidden" id="dashboard-title">
        {t('dashboard')}
      </h2>

      {statsQuery.isLoading ? <p className="dashboard-loading">{t('loading')}</p> : null}
      {statsQuery.isError ? <p className="form-error">{t('dashboardLoadFailed')}</p> : null}

      {stats ? (
        <>
          <div className="dashboard-top-grid">
            <div className="dashboard-card">
              <h3>{t('dashboardRecentEntries')}</h3>
              {stats.recentEntries.length === 0 ? (
                <p className="empty-state">{t('noTimeEntries')}</p>
              ) : (
                <div className="dashboard-recent-list">
                  {stats.recentEntries.map((entry) => (
                    <RecentEntryRow
                      entry={entry}
                      key={entry.id}
                      onRestart={(value) => restartMutation.mutate(value)}
                      pending={restartMutation.isPending}
                      t={t}
                    />
                  ))}
                </div>
              )}
            </div>

            <LastSevenDaysCard days={stats.lastSevenDays} t={t} />

            <ActivityGraphCard
              activityMonth={activityMonth}
              heatmap={stats.activityHeatmap}
              locale={locale}
              onNextMonth={() => setActivityMonth((current) => shiftMonthKey(current, 1))}
              onPreviousMonth={() => setActivityMonth((current) => shiftMonthKey(current, -1))}
              t={t}
            />
          </div>

          <WeekOverview locale={locale} stats={stats} t={t} />
        </>
      ) : null}
    </section>
  );
}

import { useQuery } from '@tanstack/react-query';
import { Download, FileJson } from 'lucide-react';
import { FormEvent, useMemo, useState } from 'react';
import {
  downloadTimeReportExport,
  fetchTimeReport,
  type Locale,
  type TimeReportGroupBy,
  type TimeReportParams,
} from './api';
import { endOfMonth, startOfMonth, toMonthQueryFrom, toMonthQueryTo } from './calendarMonth';
import type { Translator } from './timeEntryUi';
import { formatDuration } from './timeEntryUi';
import { ProjectBadge } from './projectBadgeUi';

type ReportFormState = {
  billableOnly: boolean;
  from: string;
  groupBy: TimeReportGroupBy;
  includeTimestamps: boolean;
  to: string;
};

function defaultReportForm(detailed = false): ReportFormState {
  const monthStart = startOfMonth(new Date());
  const monthEnd = endOfMonth(monthStart);
  return {
    from: toMonthQueryFrom(monthStart).slice(0, 10),
    to: toMonthQueryTo(monthEnd).slice(0, 10),
    groupBy: 'project',
    includeTimestamps: detailed,
    billableOnly: false,
  };
}

function formToReportParams(form: ReportFormState): TimeReportParams {
  return {
    from: toReportQueryFrom(form.from),
    to: toReportQueryTo(form.to),
    groupBy: form.groupBy,
    includeTimestamps: form.includeTimestamps,
    billableOnly: form.billableOnly,
  };
}

function toReportQueryFrom(dateValue: string): string {
  return new Date(`${dateValue}T00:00:00`).toISOString();
}

function toReportQueryTo(dateValue: string): string {
  return new Date(`${dateValue}T23:59:59`).toISOString();
}

function triggerDownload(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}

export function TimeReportPanel({
  locale,
  t,
  detailed = false,
}: {
  locale: Locale;
  t: Translator;
  detailed?: boolean;
}) {
  const [form, setForm] = useState<ReportFormState>(() => defaultReportForm(detailed));
  const [applied, setApplied] = useState<TimeReportParams>(() => formToReportParams(defaultReportForm(detailed)));
  const [exportError, setExportError] = useState('');

  const reportQuery = useQuery({
    queryKey: ['time-report', applied],
    queryFn: () => fetchTimeReport(applied),
    retry: false,
  });

  const groupByLabel = useMemo(
    () => ({
      day: t('reportGroupDay'),
      client: t('reportGroupClient'),
      project: t('reportGroupProject'),
      task: t('reportGroupTask'),
    }),
    [t],
  );

  function submitPreview(event: FormEvent) {
    event.preventDefault();
    setApplied(formToReportParams(form));
  }

  async function handleExport(format: 'csv' | 'json') {
    if (!applied || !reportQuery.isSuccess) {
      return;
    }
    setExportError('');
    try {
      const blob = await downloadTimeReportExport(applied, format);
      triggerDownload(blob, format === 'csv' ? 'leotime-report.csv' : 'leotime-report.json');
    } catch {
      setExportError(t('reportExportFailed'));
    }
  }

  const report = reportQuery.data;
  const canExport = reportQuery.isSuccess;

  return (
    <section className="clients-section report-section" id="overview" aria-labelledby="overview-title">
      <div className="clients-heading">
        <div className="section-title-group">
          <span className="section-kicker">
            <Download aria-hidden="true" />
            {t('reporting')}
          </span>
          <h2 id="overview-title">{t('reporting')}</h2>
          <p>{t('reportPanelSubtitle')}</p>
        </div>
      </div>

      <div className="report-workbench">
        <aside className="report-filters-panel">
          <h3>{t('reportFilters')}</h3>
          <form className="report-form" noValidate onSubmit={submitPreview}>
            <label className="form-field">
              {t('reportFrom')}
              <input onChange={(event) => setForm((current) => ({ ...current, from: event.target.value }))} type="date" value={form.from} />
            </label>
            <label className="form-field">
              {t('reportTo')}
              <input onChange={(event) => setForm((current) => ({ ...current, to: event.target.value }))} type="date" value={form.to} />
            </label>
            <label className="form-field">
              {t('reportGroupBy')}
              <select
                disabled={form.includeTimestamps}
                onChange={(event) => setForm((current) => ({ ...current, groupBy: event.target.value as TimeReportGroupBy }))}
                value={form.groupBy}
              >
                <option value="day">{groupByLabel.day}</option>
                <option value="client">{groupByLabel.client}</option>
                <option value="project">{groupByLabel.project}</option>
                <option value="task">{groupByLabel.task}</option>
              </select>
            </label>
            <label className="checkbox-field">
              <input
                checked={form.includeTimestamps}
                onChange={(event) => setForm((current) => ({ ...current, includeTimestamps: event.target.checked }))}
                type="checkbox"
              />
              <span>{t('reportIncludeTimestamps')}</span>
            </label>
            <label className="checkbox-field">
              <input
                checked={form.billableOnly}
                onChange={(event) => setForm((current) => ({ ...current, billableOnly: event.target.checked }))}
                type="checkbox"
              />
              <span>{t('reportBillableOnly')}</span>
            </label>
            <div className="report-form-actions">
              <button className="secondary-button" type="submit">
                {t('reportPreview')}
              </button>
              <button className="secondary-button" disabled={!canExport} onClick={() => void handleExport('csv')} type="button">
                <Download aria-hidden="true" />
                {t('reportDownloadCsv')}
              </button>
              <button className="secondary-button" disabled={!canExport} onClick={() => void handleExport('json')} type="button">
                <FileJson aria-hidden="true" />
                {t('reportDownloadJson')}
              </button>
            </div>
          </form>
        </aside>

        <div className="report-results-panel" aria-live="polite">
          <div className="report-results-header">
            <h3>{t('reportResults')}</h3>
            {report && report.entryCount > 0 ? (
              <div className="report-results-summary">
                <span>{formatDuration(report.totalSeconds)}</span>
                <span>
                  {report.entryCount} {t('reportEntries')}
                </span>
              </div>
            ) : null}
          </div>

          {exportError ? (
            <div className="timer-inline-error" role="alert">
              {exportError}
            </div>
          ) : null}

          <div className="report-preview">
            {reportQuery.isLoading ? <span className="sync-pill">{t('loading')}</span> : null}
            {reportQuery.isError ? (
              <div className="timer-inline-error" role="alert">
                {t('reportLoadFailed')}
              </div>
            ) : null}
            {report && report.entryCount === 0 ? (
              <div className="panel-empty-state">
                <p>{t('reportNoData')}</p>
              </div>
            ) : null}
            {report && report.entryCount > 0 && !report.includeTimestamps ? (
              <div className="report-table-wrap">
                <table className="report-table">
                  <thead>
                    <tr>
                      <th>{groupByLabel[report.groupBy]}</th>
                      <th>{t('reportEntries')}</th>
                      <th>{t('reportDuration')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {report.groups?.map((group) => (
                      <tr key={`${group.key}-${group.label}`}>
                        <td>
                          {report.groupBy === 'project' ? (
                            <ProjectBadge
                              color={group.projectColor}
                              compact
                              emptyLabel={group.label}
                              name={group.key ? group.label : undefined}
                            />
                          ) : (
                            group.label
                          )}
                        </td>
                        <td>{group.entryCount}</td>
                        <td>{formatDuration(group.totalSeconds)}</td>
                      </tr>
                    ))}
                  </tbody>
                  <tfoot>
                    <tr>
                      <td>{t('reportGrandTotal')}</td>
                      <td>{report.entryCount}</td>
                      <td>{formatDuration(report.totalSeconds)}</td>
                    </tr>
                  </tfoot>
                </table>
              </div>
            ) : null}
            {report && report.includeTimestamps && report.entries && report.entries.length > 0 ? (
              <div className="report-table-wrap">
                <table className="report-table">
                  <thead>
                    <tr>
                      <th>{t('description')}</th>
                      <th>{t('taskProject')}</th>
                      <th>{t('startedAt')}</th>
                      <th>{t('endedAt')}</th>
                      <th>{t('reportDuration')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {report.entries.map((entry) => (
                      <tr key={entry.id}>
                        <td>{entry.description || t('noDescription')}</td>
                        <td>
                          <ProjectBadge
                            color={entry.projectColor}
                            compact
                            emptyLabel={t('taskProjectOptional')}
                            name={entry.projectName}
                          />
                        </td>
                        <td>{formatReportDateTime(entry.startedAt, locale)}</td>
                        <td>{formatReportDateTime(entry.endedAt, locale)}</td>
                        <td>{formatDuration(entry.durationSeconds)}</td>
                      </tr>
                    ))}
                  </tbody>
                  <tfoot>
                    <tr>
                      <td colSpan={4}>{t('reportGrandTotal')}</td>
                      <td>{formatDuration(report.totalSeconds)}</td>
                    </tr>
                  </tfoot>
                </table>
              </div>
            ) : null}
          </div>
        </div>
      </div>

      <div className="report-detailed-anchor" id="detailed" aria-hidden="true" />
    </section>
  );
}

function formatReportDateTime(value: string, locale: Locale) {
  return new Intl.DateTimeFormat(locale === 'es' ? 'es-ES' : 'en-US', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value));
}

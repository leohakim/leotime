import { useMutation, useQueryClient } from '@tanstack/react-query';
import { CircleAlert, CircleCheck, Download, FileJson, Import, Upload } from 'lucide-react';
import { FormEvent, useMemo, useRef, useState } from 'react';
import {
  downloadTimeReportExport,
  importSolidtimeExport,
  type ImportEntityStats,
  type SolidtimeImportSummary,
} from './api';
import { endOfMonth, startOfMonth, toMonthQueryFrom, toMonthQueryTo } from './calendarMonth';
import type { Translator } from './timeEntryUi';
import { useToast } from './toast';

type ExportFormState = {
  billableOnly: boolean;
  from: string;
  to: string;
};

function defaultExportForm(): ExportFormState {
  const monthStart = startOfMonth(new Date());
  const monthEnd = endOfMonth(monthStart);
  return {
    from: toMonthQueryFrom(monthStart).slice(0, 10),
    to: toMonthQueryTo(monthEnd).slice(0, 10),
    billableOnly: false,
  };
}

function toExportQueryFrom(dateValue: string): string {
  return new Date(`${dateValue}T00:00:00`).toISOString();
}

function toExportQueryTo(dateValue: string): string {
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

function formatEntityStats(stats: ImportEntityStats | undefined, t: Translator): string {
  const safe = stats ?? { seen: 0, created: 0, updated: 0, skipped: 0 };
  return t('importEntityStats')
    .replace('{created}', String(safe.created))
    .replace('{updated}', String(safe.updated))
    .replace('{skipped}', String(safe.skipped));
}

function ImportSummary({ summary, t }: { summary: SolidtimeImportSummary; t: Translator }) {
  const rows = useMemo(
    () => [
      { label: t('importEntityClients'), stats: summary.clients },
      { label: t('importEntityProjects'), stats: summary.projects },
      { label: t('importEntityTasks'), stats: summary.tasks },
      { label: t('importEntityTags'), stats: summary.tags },
      { label: t('importEntityTimeEntries'), stats: summary.timeEntries },
    ],
    [summary, t],
  );

  return (
    <div className="import-summary">
      <h3>{t('importSummaryTitle')}</h3>
      <dl className="import-summary-grid">
        <div>
          <dt>Export ID</dt>
          <dd>{summary.exportId || '—'}</dd>
        </div>
        <div>
          <dt>Version</dt>
          <dd>{summary.version || '—'}</dd>
        </div>
        <div>
          <dt>Provider</dt>
          <dd>{summary.provider}</dd>
        </div>
        <div>
          <dt>Mode</dt>
          <dd>{summary.dryRun ? t('importDryRun') : t('importRun')}</dd>
        </div>
      </dl>
      <ul className="import-summary-stats">
        {rows.map((row) => (
          <li key={row.label}>
            <strong>{row.label}</strong>
            <span>{formatEntityStats(row.stats, t)}</span>
            <span className="import-summary-seen">{row.stats?.seen ?? 0} seen</span>
          </li>
        ))}
      </ul>
      {summary.warnings.length > 0 ? (
        <div className="import-summary-warnings">
          <strong>{t('importWarnings')}</strong>
          <ul>
            {summary.warnings.map((warning) => (
              <li key={warning}>{warning}</li>
            ))}
          </ul>
        </div>
      ) : null}
      {summary.errors.length > 0 ? (
        <ul className="form-error-list">
          {summary.errors.map((error) => (
            <li key={error}>{error}</li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}

export function ImportExportPanel({ t }: { t: Translator }) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [dryRun, setDryRun] = useState(true);
  const [importError, setImportError] = useState('');
  const [importMessage, setImportMessage] = useState('');
  const [summary, setSummary] = useState<SolidtimeImportSummary | null>(null);
  const [exportForm, setExportForm] = useState<ExportFormState>(defaultExportForm);
  const [exportError, setExportError] = useState('');

  const importMutation = useMutation({
    mutationFn: () => {
      if (!selectedFile) {
        throw new Error(t('importFileRequired'));
      }
      return importSolidtimeExport(selectedFile, dryRun);
    },
    onSuccess: (result) => {
      setSummary(result);
      if (result.errors.length > 0) {
        setImportMessage('');
        const message = result.errors[0] ?? t('importFailed');
        setImportError(message);
        toast.error(message);
        return;
      }
      setImportError('');
      const message = result.dryRun ? t('importValidateSuccess') : t('importSuccess');
      setImportMessage(message);
      toast.success(message);
      if (!result.dryRun) {
        void queryClient.invalidateQueries({ queryKey: ['clients'] });
        void queryClient.invalidateQueries({ queryKey: ['projects'] });
        void queryClient.invalidateQueries({ queryKey: ['tasks'] });
        void queryClient.invalidateQueries({ queryKey: ['tags'] });
        void queryClient.invalidateQueries({ queryKey: ['time-entries'] });
        void queryClient.invalidateQueries({ queryKey: ['timers'] });
        void queryClient.invalidateQueries({ queryKey: ['overview'] });
        void queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] });
      }
    },
    onError: (error) => {
      setImportMessage('');
      const message = error instanceof Error ? error.message : t('importFailed');
      setImportError(message);
      toast.error(message);
    },
  });

  function submitImport(event: FormEvent) {
    event.preventDefault();
    setImportError('');
    setImportMessage('');
    if (!selectedFile) {
      setImportError(t('importFileRequired'));
      return;
    }
    if (!selectedFile.name.toLowerCase().endsWith('.zip')) {
      setImportError(t('importInvalidFile'));
      return;
    }
    importMutation.mutate();
  }

  async function handleExport(format: 'csv' | 'json') {
    setExportError('');
    try {
      const blob = await downloadTimeReportExport(
        {
          from: toExportQueryFrom(exportForm.from),
          to: toExportQueryTo(exportForm.to),
          groupBy: 'project',
          includeTimestamps: true,
          billableOnly: exportForm.billableOnly,
        },
        format,
      );
      triggerDownload(blob, format === 'csv' ? 'leotime-export.csv' : 'leotime-export.json');
      toast.success(t('reportExportSuccess'));
    } catch {
      setExportError(t('reportExportFailed'));
      toast.error(t('reportExportFailed'));
    }
  }

  return (
    <section className="clients-section import-export-section" id="import-export" aria-labelledby="import-export-title">
      <div className="clients-heading">
        <div className="section-title-group">
          <span className="section-kicker">
            <Import aria-hidden="true" />
            {t('importExport')}
          </span>
          <h2 id="import-export-title">{t('importExport')}</h2>
          <p>{t('importExportPanelSubtitle')}</p>
        </div>
      </div>

      <div className="import-export-grid">
        <article className="import-export-card">
          <h3>{t('importSectionImport')}</h3>
          <p className="import-export-help">{t('importSolidtimeHelp')}</p>
          <form className="import-form" noValidate onSubmit={submitImport}>
            <label className="form-field">
              {t('importFileLabel')}
              <div className="import-file-picker">
                <button
                  className="secondary-button"
                  onClick={() => fileInputRef.current?.click()}
                  type="button"
                >
                  {t('importChooseFile')}
                </button>
                <span className={selectedFile ? 'import-file-name is-selected' : 'import-file-name'}>
                  {selectedFile?.name ?? t('importNoFileChosen')}
                </span>
                <input
                  ref={fileInputRef}
                  accept=".zip,application/zip"
                  className="visually-hidden"
                  onChange={(event) => {
                    setSelectedFile(event.target.files?.[0] ?? null);
                    setSummary(null);
                    setImportMessage('');
                    setImportError('');
                  }}
                  type="file"
                />
              </div>
            </label>
            <label className="checkbox-field">
              <input checked={dryRun} onChange={(event) => setDryRun(event.target.checked)} type="checkbox" />
              <span>{t('importDryRun')}</span>
            </label>
            <div className="import-form-actions">
              <button className="secondary-button" disabled={importMutation.isPending} type="submit">
                <Upload aria-hidden="true" />
                {dryRun ? t('importRunDryRun') : t('importRun')}
              </button>
            </div>
            {importError ? (
              <p className="form-error" role="alert">
                <CircleAlert aria-hidden="true" />
                {importError}
              </p>
            ) : null}
            {importMessage ? (
              <p className="form-success" role="status">
                <CircleCheck aria-hidden="true" />
                {importMessage}
              </p>
            ) : null}
          </form>
          {summary ? <ImportSummary summary={summary} t={t} /> : null}
        </article>

        <article className="import-export-card">
          <h3>{t('importSectionExport')}</h3>
          <p className="import-export-help">{t('reportPanelSubtitle')}</p>
          <form
            className="import-export-form"
            onSubmit={(event) => {
              event.preventDefault();
              void handleExport('csv');
            }}
          >
            <label className="form-field">
              {t('reportFrom')}
              <input
                onChange={(event) => setExportForm((current) => ({ ...current, from: event.target.value }))}
                type="date"
                value={exportForm.from}
              />
            </label>
            <label className="form-field">
              {t('reportTo')}
              <input
                onChange={(event) => setExportForm((current) => ({ ...current, to: event.target.value }))}
                type="date"
                value={exportForm.to}
              />
            </label>
            <label className="checkbox-field">
              <input
                checked={exportForm.billableOnly}
                onChange={(event) => setExportForm((current) => ({ ...current, billableOnly: event.target.checked }))}
                type="checkbox"
              />
              <span>{t('reportBillableOnly')}</span>
            </label>
            <div className="import-form-actions">
              <button className="secondary-button" type="submit">
                <Download aria-hidden="true" />
                {t('exportDownloadCsv')}
              </button>
              <button className="secondary-button" type="button" onClick={() => void handleExport('json')}>
                <FileJson aria-hidden="true" />
                {t('exportDownloadJson')}
              </button>
            </div>
            {exportError ? <p className="form-error">{exportError}</p> : null}
          </form>
        </article>
      </div>
    </section>
  );
}

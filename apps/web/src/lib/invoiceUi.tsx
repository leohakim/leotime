import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { CircleAlert, Download, FileJson, FileText, Pencil, Trash2 } from 'lucide-react';
import { FormEvent, useEffect, useMemo, useState } from 'react';
import {
  cancelInvoice,
  createInvoiceDraftFromTime,
  deleteInvoice,
  downloadInvoiceDocument,
  downloadInvoiceExport,
  fetchInvoice,
  fetchInvoiceSeries,
  fetchInvoices,
  isApiError,
  issueInvoice,
  previewInvoice,
  updateInvoice,
  updateInvoiceStatus,
  type Client,
  type Invoice,
  type InvoiceStatus,
  type Locale,
  type WorkProtocolDetail,
  type InvoiceLineDetail,
} from './api';
import { endOfMonth, startOfMonth, toMonthQueryFrom, toMonthQueryTo } from './calendarMonth';
import { FieldError, fieldClass, hasErrors } from './crudFormUi';
import { confirmDestructiveAction } from './destructiveUi';
import { SurfaceEmpty, SurfaceError, SurfaceLoading } from './feedbackUi';
import { isLocalId } from './offline/mutations';
import type { Translator } from './timeEntryUi';
import { useToast } from './toast';

type DraftFormState = {
  clientId: string;
  from: string;
  taxRatePercent: string;
  to: string;
  withholding: string;
  withholdingLabel: string;
  notes: string;
  seriesId: string;
  workProtocolDetail: WorkProtocolDetail;
  invoiceLineDetail: InvoiceLineDetail;
};

type DraftEditFormState = {
  notes: string;
  taxRatePercent: string;
  withholding: string;
  withholdingLabel: string;
  seriesId: string;
  workProtocolDetail: WorkProtocolDetail;
  invoiceLineDetail: InvoiceLineDetail;
};

type DraftFormErrors = Partial<Record<keyof DraftFormState | 'form', string>>;
type DraftEditFormErrors = Partial<Record<keyof DraftEditFormState | 'form', string>>;

function resolveWithholdingLabel(label: string | undefined, t: Translator): string {
  const trimmed = label?.trim();
  return trimmed ? trimmed : t('invoiceWithholding');
}

function invoiceToEditForm(invoice: Invoice): DraftEditFormState {
  const taxRateBasisPoints = invoice.lines[0]?.taxRateBasisPoints ?? 2100;
  return {
    notes: invoice.notes,
    taxRatePercent: String(taxRateBasisPoints / 100),
    withholding: invoice.withholdingMinor > 0 ? (invoice.withholdingMinor / 100).toFixed(2) : '',
    withholdingLabel: invoice.withholdingLabel ?? '',
    seriesId: invoice.seriesId ?? '',
    workProtocolDetail: invoice.workProtocolDetail ?? 'standard',
    invoiceLineDetail: invoice.invoiceLineDetail ?? 'by_project',
  };
}

function defaultDraftForm(defaultWithholdingLabel = ''): DraftFormState {
  const monthStart = startOfMonth(new Date());
  const monthEnd = endOfMonth(monthStart);
  return {
    clientId: '',
    from: toMonthQueryFrom(monthStart).slice(0, 10),
    to: toMonthQueryTo(monthEnd).slice(0, 10),
    taxRatePercent: '21',
    withholding: '',
    withholdingLabel: defaultWithholdingLabel,
    notes: '',
    seriesId: '',
    workProtocolDetail: 'standard',
    invoiceLineDetail: 'by_project',
  };
}

function toInvoiceQueryFrom(dateValue: string): string {
  return new Date(`${dateValue}T00:00:00`).toISOString();
}

function toInvoiceQueryTo(dateValue: string): string {
  return new Date(`${dateValue}T23:59:59`).toISOString();
}

export function formatMoneyMinor(amountMinor: number, currency: string, locale: Locale): string {
  const formatter = new Intl.NumberFormat(locale === 'es' ? 'es-ES' : 'en-US', {
    style: 'currency',
    currency: currency || 'EUR',
    minimumFractionDigits: 2,
  });
  return formatter.format(amountMinor / 100);
}

function triggerDownload(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}

const statusClassName: Record<InvoiceStatus, string> = {
  draft: 'invoice-status-draft',
  issued: 'invoice-status-issued',
  paid: 'invoice-status-paid',
  cancelled: 'invoice-status-cancelled',
};

const statusLabelKey: Record<InvoiceStatus, 'invoiceStatusDraft' | 'invoiceStatusIssued' | 'invoiceStatusPaid' | 'invoiceStatusCancelled'> = {
  draft: 'invoiceStatusDraft',
  issued: 'invoiceStatusIssued',
  paid: 'invoiceStatusPaid',
  cancelled: 'invoiceStatusCancelled',
};

export function InvoicePanel({
  clients,
  defaultWithholdingLabel = '',
  locale,
  t,
  userName,
}: {
  clients: Client[];
  defaultWithholdingLabel?: string;
  locale: Locale;
  t: Translator;
  userName: string;
}) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const [form, setForm] = useState<DraftFormState>(() => defaultDraftForm(defaultWithholdingLabel));
  const [selectedInvoiceId, setSelectedInvoiceId] = useState<string | null>(null);
  const [isEditingDraft, setIsEditingDraft] = useState(false);
  const [editForm, setEditForm] = useState<DraftEditFormState | null>(null);
  const [editErrors, setEditErrors] = useState<DraftEditFormErrors>({});
  const [formErrors, setFormErrors] = useState<DraftFormErrors>({});
  const [exportError, setExportError] = useState('');

  const invoicesQuery = useQuery({
    queryKey: ['invoices'],
    queryFn: fetchInvoices,
    retry: false,
  });

  const seriesQuery = useQuery({
    queryKey: ['invoice-series'],
    queryFn: fetchInvoiceSeries,
    retry: false,
  });

  const invoiceDetailQuery = useQuery({
    queryKey: ['invoice', selectedInvoiceId],
    queryFn: () => fetchInvoice(selectedInvoiceId as string),
    enabled: selectedInvoiceId != null,
  });

  const activeClients = useMemo(
    () => clients.filter((client) => !client.archivedAt && !isLocalId(client.id)),
    [clients],
  );
  const invoiceSeries = seriesQuery.data?.series ?? [];
  const defaultSeriesId = invoiceSeries.find((series) => series.default)?.id ?? invoiceSeries[0]?.id ?? '';
  const invoices = invoicesQuery.data?.invoices ?? [];
  const selectedInvoice = invoiceDetailQuery.data ?? invoices.find((invoice) => invoice.id === selectedInvoiceId) ?? null;

  useEffect(() => {
    setForm((current) => ({
      ...current,
      withholdingLabel: current.withholdingLabel || defaultWithholdingLabel,
    }));
  }, [defaultWithholdingLabel]);

  useEffect(() => {
    setIsEditingDraft(false);
    setEditForm(null);
    setEditErrors({});
  }, [selectedInvoiceId]);

  function applyDraftCreateError(error: unknown) {
    const mapped = mapInvoiceDraftApiError(error, t);
    setFormErrors(mapped);
    const toastMessage =
      mapped.form || mapped.clientId || mapped.from || mapped.to || mapped.taxRatePercent || mapped.withholding || t('invoiceDraftFailed');
    toast.error(toastMessage);
  }

  function applyDraftEditError(error: unknown) {
    const mapped = mapInvoiceDraftEditApiError(error, t);
    setEditErrors(mapped);
    const toastMessage =
      mapped.form || mapped.taxRatePercent || mapped.withholding || t('invoiceDraftSaveFailed');
    toast.error(toastMessage);
  }

  function updateDraftField<K extends keyof DraftFormState>(key: K, value: DraftFormState[K]) {
    setForm((current) => ({ ...current, [key]: value }));
    setFormErrors((current) => ({ ...current, [key]: undefined, form: undefined }));
  }

  function updateEditField<K extends keyof DraftEditFormState>(key: K, value: DraftEditFormState[K]) {
    setEditForm((current) => (current ? { ...current, [key]: value } : current));
    setEditErrors((current) => ({ ...current, [key]: undefined, form: undefined }));
  }

  const createMutation = useMutation({
    mutationFn: () => {
      const taxRatePercent = Number.parseFloat(form.taxRatePercent.replace(',', '.'));
      const withholding =
        form.withholding.trim() === '' ? 0 : Math.round(Number.parseFloat(form.withholding.replace(',', '.')) * 100);
      return createInvoiceDraftFromTime({
        clientId: form.clientId,
        from: toInvoiceQueryFrom(form.from),
        to: toInvoiceQueryTo(form.to),
        sellerName: userName,
        taxRateBasisPoints: Math.round(taxRatePercent * 100),
        withholdingMinor: withholding,
        withholdingLabel: form.withholdingLabel.trim(),
        notes: form.notes.trim(),
        seriesId: form.seriesId || defaultSeriesId || undefined,
        workProtocolDetail: form.workProtocolDetail,
        invoiceLineDetail: form.invoiceLineDetail,
      });
    },
    onSuccess: (invoice) => {
      setFormErrors({});
      setSelectedInvoiceId(invoice.id);
      queryClient.invalidateQueries({ queryKey: ['invoices'] });
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] });
      toast.success(t('invoiceDraftCreated'));
    },
    onError: applyDraftCreateError,
  });

  const updateDraftMutation = useMutation({
    mutationFn: (input: { invoiceId: string; form: DraftEditFormState }) => {
      const taxRatePercent = Number.parseFloat(input.form.taxRatePercent.replace(',', '.'));
      const withholding =
        input.form.withholding.trim() === '' ? 0 : Math.round(Number.parseFloat(input.form.withholding.replace(',', '.')) * 100);
      return updateInvoice(input.invoiceId, {
        notes: input.form.notes.trim(),
        taxRateBasisPoints: Math.round(taxRatePercent * 100),
        withholdingMinor: withholding,
        withholdingLabel: input.form.withholdingLabel.trim(),
        seriesId: input.form.seriesId || undefined,
        workProtocolDetail: input.form.workProtocolDetail,
        invoiceLineDetail: input.form.invoiceLineDetail,
      });
    },
    onSuccess: (invoice) => {
      setEditErrors({});
      setIsEditingDraft(false);
      setEditForm(null);
      queryClient.invalidateQueries({ queryKey: ['invoices'] });
      queryClient.invalidateQueries({ queryKey: ['invoice', invoice.id] });
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] });
      toast.success(t('invoiceDraftSaved'));
    },
    onError: applyDraftEditError,
  });

  const statusMutation = useMutation({
    mutationFn: ({ invoiceId, status }: { invoiceId: string; status: InvoiceStatus }) =>
      updateInvoiceStatus(invoiceId, status),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['invoices'] });
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] });
      toast.success(t('invoiceStatusUpdated'));
    },
    onError: () => toast.error(t('invoiceStatusUpdateFailed')),
  });

  const deleteMutation = useMutation({
    mutationFn: deleteInvoice,
    onSuccess: () => {
      setSelectedInvoiceId(null);
      queryClient.invalidateQueries({ queryKey: ['invoices'] });
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] });
      toast.success(t('invoiceDeleted'));
    },
    onError: () => toast.error(t('invoiceDeleteFailed')),
  });

  const issueMutation = useMutation({
    mutationFn: issueInvoice,
    onSuccess: (invoice) => {
      setSelectedInvoiceId(invoice.id);
      queryClient.invalidateQueries({ queryKey: ['invoices'] });
      queryClient.invalidateQueries({ queryKey: ['invoice', invoice.id] });
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] });
      toast.success(t('invoiceIssueSuccess'));
    },
    onError: () => toast.error(t('invoiceIssueFailed')),
  });

  const cancelMutation = useMutation({
    mutationFn: ({ invoiceId, reason }: { invoiceId: string; reason: string }) => cancelInvoice(invoiceId, reason),
    onSuccess: (invoice) => {
      queryClient.invalidateQueries({ queryKey: ['invoices'] });
      queryClient.invalidateQueries({ queryKey: ['invoice', invoice.id] });
      toast.success(t('invoiceStatusUpdated'));
    },
    onError: () => toast.error(t('invoiceStatusUpdateFailed')),
  });

  function submitDraft(event: FormEvent) {
    event.preventDefault();
    const validation = validateDraftForm(form, t);
    setFormErrors(validation);
    if (hasErrors(validation)) {
      const firstMessage =
        validation.clientId || validation.from || validation.to || validation.taxRatePercent || validation.withholding;
      if (firstMessage) {
        toast.error(firstMessage);
      }
      return;
    }
    createMutation.mutate();
  }

  function submitDraftEdit(event: FormEvent) {
    event.preventDefault();
    if (!selectedInvoice || !editForm) {
      return;
    }
    const validation = validateDraftEditForm(editForm, t);
    setEditErrors(validation);
    if (hasErrors(validation)) {
      const firstMessage = validation.taxRatePercent || validation.withholding;
      if (firstMessage) {
        toast.error(firstMessage);
      }
      return;
    }
    updateDraftMutation.mutate({ invoiceId: selectedInvoice.id, form: editForm });
  }

  function startDraftEdit(invoice: Invoice) {
    setIsEditingDraft(true);
    setEditForm(invoiceToEditForm(invoice));
    setEditErrors({});
  }

  async function handleExport(invoice: Invoice, format: 'html' | 'csv' | 'json') {
    setExportError('');
    try {
      const blob = await downloadInvoiceExport(invoice.id, format);
      const extension = format === 'html' ? 'html' : format;
      triggerDownload(blob, `${invoice.invoiceNumber}.${extension}`);
      toast.success(t('invoiceExportSuccess'));
    } catch {
      setExportError(t('invoiceExportFailed'));
      toast.error(t('invoiceExportFailed'));
    }
  }

  async function handlePreview(invoice: Invoice) {
    setExportError('');
    try {
      const blob = await previewInvoice(invoice.id);
      const url = URL.createObjectURL(blob);
      window.open(url, '_blank', 'noopener,noreferrer');
      window.setTimeout(() => URL.revokeObjectURL(url), 60_000);
    } catch {
      setExportError(t('invoicePreviewFailed'));
      toast.error(t('invoicePreviewFailed'));
    }
  }

  async function handleDocumentDownload(invoice: Invoice, documentId: string, filename: string) {
    setExportError('');
    try {
      const blob = await downloadInvoiceDocument(invoice.id, documentId);
      triggerDownload(blob, filename);
      toast.success(t('invoiceExportSuccess'));
    } catch {
      setExportError(t('invoiceExportFailed'));
      toast.error(t('invoiceExportFailed'));
    }
  }

  function handleCancel(invoice: Invoice) {
    if (!confirmDestructiveAction(t('invoiceCancelConfirm'))) {
      return;
    }
    const reason = window.prompt(t('invoiceCancelReason'));
    if (!reason?.trim()) {
      return;
    }
    cancelMutation.mutate({ invoiceId: invoice.id, reason: reason.trim() });
  }

  return (
    <section className="clients-section invoice-section" id="invoices" aria-labelledby="invoices-title">
      <div className="clients-heading">
        <div className="section-title-group">
          <span className="section-kicker">
            <FileText aria-hidden="true" />
            {t('invoices')}
          </span>
          <h2 id="invoices-title">{t('invoices')}</h2>
          <p>{t('invoicePanelSubtitle')}</p>
        </div>
      </div>

      <div className="invoice-workbench">
        <aside className="invoice-draft-panel">
          <h3>{t('invoiceNewDraft')}</h3>
          <form className="invoice-form" noValidate onSubmit={submitDraft}>
            {formErrors.form ? (
              <div className="form-alert" role="alert">
                <CircleAlert aria-hidden="true" />
                {formErrors.form}
              </div>
            ) : null}
            <label className={fieldClass(formErrors.clientId)} htmlFor="invoice-draft-client">
              {t('invoiceClient')}
              <select
                aria-describedby={formErrors.clientId ? 'invoice-draft-client-error' : undefined}
                aria-invalid={Boolean(formErrors.clientId)}
                id="invoice-draft-client"
                onChange={(event) => updateDraftField('clientId', event.target.value)}
                value={form.clientId}
              >
                <option value="">{t('invoiceClientPlaceholder')}</option>
                {activeClients.map((client) => (
                  <option key={client.id} value={client.id}>
                    {client.name}
                  </option>
                ))}
              </select>
              <FieldError id="invoice-draft-client-error" message={formErrors.clientId} />
            </label>
            <label className="form-field" htmlFor="invoice-draft-series">
              {t('invoiceSeries')}
              <select
                id="invoice-draft-series"
                onChange={(event) => updateDraftField('seriesId', event.target.value)}
                value={form.seriesId || defaultSeriesId}
              >
                {invoiceSeries.map((series) => (
                  <option key={series.id} value={series.id}>
                    {series.code} — {series.name}
                  </option>
                ))}
              </select>
            </label>
            <label className="form-field" htmlFor="invoice-draft-line-detail">
              {t('invoiceLineDetail')}
              <select
                id="invoice-draft-line-detail"
                onChange={(event) => updateDraftField('invoiceLineDetail', event.target.value as InvoiceLineDetail)}
                value={form.invoiceLineDetail}
              >
                <option value="summary">{t('invoiceLineSummary')}</option>
                <option value="by_project">{t('invoiceLineByProject')}</option>
                <option value="granular">{t('invoiceLineGranular')}</option>
              </select>
            </label>
            <label className="form-field" htmlFor="invoice-draft-work-protocol">
              {t('invoiceWorkProtocolDetail')}
              <select
                id="invoice-draft-work-protocol"
                onChange={(event) => updateDraftField('workProtocolDetail', event.target.value as WorkProtocolDetail)}
                value={form.workProtocolDetail}
              >
                <option value="summary">{t('invoiceWorkProtocolSummary')}</option>
                <option value="standard">{t('invoiceWorkProtocolStandard')}</option>
                <option value="detailed">{t('invoiceWorkProtocolDetailed')}</option>
              </select>
            </label>
            <label className={fieldClass(formErrors.from)} htmlFor="invoice-draft-from">
              {t('reportFrom')}
              <input
                aria-describedby={formErrors.from ? 'invoice-draft-from-error' : undefined}
                aria-invalid={Boolean(formErrors.from)}
                id="invoice-draft-from"
                onChange={(event) => updateDraftField('from', event.target.value)}
                type="date"
                value={form.from}
              />
              <FieldError id="invoice-draft-from-error" message={formErrors.from} />
            </label>
            <label className={fieldClass(formErrors.to)} htmlFor="invoice-draft-to">
              {t('reportTo')}
              <input
                aria-describedby={formErrors.to ? 'invoice-draft-to-error' : undefined}
                aria-invalid={Boolean(formErrors.to)}
                id="invoice-draft-to"
                onChange={(event) => updateDraftField('to', event.target.value)}
                type="date"
                value={form.to}
              />
              <FieldError id="invoice-draft-to-error" message={formErrors.to} />
            </label>
            <label className={fieldClass(formErrors.taxRatePercent)} htmlFor="invoice-draft-tax">
              {t('invoiceTaxRate')}
              <input
                aria-describedby={formErrors.taxRatePercent ? 'invoice-draft-tax-error' : undefined}
                aria-invalid={Boolean(formErrors.taxRatePercent)}
                id="invoice-draft-tax"
                inputMode="decimal"
                onChange={(event) => updateDraftField('taxRatePercent', event.target.value)}
                placeholder="21"
                value={form.taxRatePercent}
              />
              <FieldError id="invoice-draft-tax-error" message={formErrors.taxRatePercent} />
            </label>
            <label className={fieldClass(formErrors.withholding)} htmlFor="invoice-draft-withholding-label">
              {t('invoiceWithholdingLabel')}
              <input
                id="invoice-draft-withholding-label"
                onChange={(event) => updateDraftField('withholdingLabel', event.target.value)}
                placeholder={t('invoiceWithholding')}
                value={form.withholdingLabel}
              />
            </label>
            <label className={fieldClass(formErrors.withholding)} htmlFor="invoice-draft-withholding">
              {resolveWithholdingLabel(form.withholdingLabel, t)} — {t('invoiceWithholdingAmount')}
              <input
                aria-describedby={formErrors.withholding ? 'invoice-draft-withholding-error' : undefined}
                aria-invalid={Boolean(formErrors.withholding)}
                id="invoice-draft-withholding"
                inputMode="decimal"
                onChange={(event) => updateDraftField('withholding', event.target.value)}
                placeholder="0.00"
                value={form.withholding}
              />
              <FieldError id="invoice-draft-withholding-error" message={formErrors.withholding} />
            </label>
            <label className="form-field invoice-notes-field" htmlFor="invoice-draft-notes">
              {t('invoiceNotes')}
              <textarea
                id="invoice-draft-notes"
                onChange={(event) => updateDraftField('notes', event.target.value)}
                rows={2}
                value={form.notes}
              />
            </label>
            <div className="invoice-form-actions">
              <button disabled={createMutation.isPending} type="submit">
                {t('invoiceCreateDraft')}
              </button>
            </div>
          </form>
        </aside>

        <div className="invoice-directory-panel">
          <h3>{t('invoiceDirectory')}</h3>
          {invoicesQuery.isError ? (
            <SurfaceError
              message={t('invoiceLoadFailed')}
              onRetry={() => void invoicesQuery.refetch()}
              retryLabel={t('retry')}
            />
          ) : null}
          {invoicesQuery.isLoading ? <SurfaceLoading label={t('loading')} /> : null}
          {!invoicesQuery.isLoading && invoices.length === 0 ? (
            <SurfaceEmpty>
              <p>{t('invoiceNoInvoices')}</p>
            </SurfaceEmpty>
          ) : null}

          {invoices.length > 0 ? (
            <div className="invoice-layout">
              <div className="invoice-list" role="list">
                {invoices.map((invoice) => (
                  <button
                    key={invoice.id}
                    className={`invoice-list-item${selectedInvoiceId === invoice.id ? ' selected' : ''}`}
                    onClick={() => setSelectedInvoiceId(invoice.id)}
                    type="button"
                  >
                    <div>
                      <strong>{invoice.invoiceNumber}</strong>
                      <span>{invoice.clientName}</span>
                    </div>
                    <div className="invoice-list-meta">
                      <span className={`invoice-status ${statusClassName[invoice.status]}`}>{t(statusLabelKey[invoice.status])}</span>
                      <span>{formatMoneyMinor(invoice.totalMinor, invoice.currency, locale)}</span>
                    </div>
                  </button>
                ))}
              </div>

              {selectedInvoice ? (
                <article className="invoice-detail">
              <header className="invoice-detail-header">
                <div>
                  <h3>{selectedInvoice.invoiceNumber}</h3>
                  <p>{selectedInvoice.clientName}</p>
                </div>
                <span className={`invoice-status ${statusClassName[selectedInvoice.status]}`}>
                  {t(statusLabelKey[selectedInvoice.status])}
                </span>
              </header>

              <dl className="invoice-totals">
                <div>
                  <dt>{t('invoiceTotalHours')}</dt>
                  <dd>
                    {Math.round(
                      (selectedInvoice.totalQuantityMinutes ??
                        selectedInvoice.lines.reduce((sum, line) => sum + line.quantityMinutes, 0)) / 60,
                    )}
                  </dd>
                </div>
                <div>
                  <dt>{t('invoiceSubtotal')}</dt>
                  <dd>{formatMoneyMinor(selectedInvoice.subtotalMinor, selectedInvoice.currency, locale)}</dd>
                </div>
                {selectedInvoice.taxMinor > 0 ? (
                  <div>
                    <dt>{t('invoiceTax')}</dt>
                    <dd>{formatMoneyMinor(selectedInvoice.taxMinor, selectedInvoice.currency, locale)}</dd>
                  </div>
                ) : null}
                {selectedInvoice.withholdingMinor > 0 ? (
                  <div>
                    <dt>{resolveWithholdingLabel(selectedInvoice.withholdingLabel, t)}</dt>
                    <dd>-{formatMoneyMinor(selectedInvoice.withholdingMinor, selectedInvoice.currency, locale)}</dd>
                  </div>
                ) : null}
                <div>
                  <dt>{t('invoiceTotal')}</dt>
                  <dd>{formatMoneyMinor(selectedInvoice.totalMinor, selectedInvoice.currency, locale)}</dd>
                </div>
              </dl>

              <table className="invoice-lines-table">
                <thead>
                  <tr>
                    <th>{t('description')}</th>
                    <th>{t('invoiceHours')}</th>
                    <th>{t('hourlyRate')}</th>
                    <th>{t('invoiceSubtotal')}</th>
                  </tr>
                </thead>
                <tbody>
                  {(selectedInvoice.lines ?? []).map((line) => (
                    <tr key={line.id}>
                      <td>{line.description}</td>
                      <td>{Math.round(line.quantityMinutes / 60)}</td>
                      <td>{formatMoneyMinor(line.unitRateMinor, selectedInvoice.currency, locale)}</td>
                      <td>{formatMoneyMinor(line.subtotalMinor, selectedInvoice.currency, locale)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>

              {selectedInvoice.notes && !isEditingDraft ? <p className="invoice-notes">{selectedInvoice.notes}</p> : null}

              {selectedInvoice.status === 'draft' && isEditingDraft && editForm ? (
                <form className="invoice-edit-form" noValidate onSubmit={submitDraftEdit}>
                  {editErrors.form ? (
                    <div className="form-alert" role="alert">
                      <CircleAlert aria-hidden="true" />
                      {editErrors.form}
                    </div>
                  ) : null}
                  <label className="form-field" htmlFor="invoice-edit-series">
                    {t('invoiceSeries')}
                    <select
                      id="invoice-edit-series"
                      onChange={(event) => updateEditField('seriesId', event.target.value)}
                      value={editForm.seriesId || defaultSeriesId}
                    >
                      {invoiceSeries.map((series) => (
                        <option key={series.id} value={series.id}>
                          {series.code} — {series.name}
                        </option>
                      ))}
                    </select>
                  </label>
                  <label className="form-field" htmlFor="invoice-edit-line-detail">
                    {t('invoiceLineDetail')}
                    <select
                      id="invoice-edit-line-detail"
                      onChange={(event) => updateEditField('invoiceLineDetail', event.target.value as InvoiceLineDetail)}
                      value={editForm.invoiceLineDetail}
                    >
                      <option value="summary">{t('invoiceLineSummary')}</option>
                      <option value="by_project">{t('invoiceLineByProject')}</option>
                      <option value="granular">{t('invoiceLineGranular')}</option>
                    </select>
                  </label>
                  <label className="form-field" htmlFor="invoice-edit-work-protocol">
                    {t('invoiceWorkProtocolDetail')}
                    <select
                      id="invoice-edit-work-protocol"
                      onChange={(event) => updateEditField('workProtocolDetail', event.target.value as WorkProtocolDetail)}
                      value={editForm.workProtocolDetail}
                    >
                      <option value="summary">{t('invoiceWorkProtocolSummary')}</option>
                      <option value="standard">{t('invoiceWorkProtocolStandard')}</option>
                      <option value="detailed">{t('invoiceWorkProtocolDetailed')}</option>
                    </select>
                  </label>
                  <label className={fieldClass(editErrors.taxRatePercent)} htmlFor="invoice-edit-tax">
                    {t('invoiceTaxRate')}
                    <input
                      aria-describedby={editErrors.taxRatePercent ? 'invoice-edit-tax-error' : undefined}
                      aria-invalid={Boolean(editErrors.taxRatePercent)}
                      id="invoice-edit-tax"
                      inputMode="decimal"
                      onChange={(event) => updateEditField('taxRatePercent', event.target.value)}
                      value={editForm.taxRatePercent}
                    />
                    <FieldError id="invoice-edit-tax-error" message={editErrors.taxRatePercent} />
                  </label>
                  <label className={fieldClass(editErrors.withholding)} htmlFor="invoice-edit-withholding-label">
                    {t('invoiceWithholdingLabel')}
                    <input
                      id="invoice-edit-withholding-label"
                      onChange={(event) => updateEditField('withholdingLabel', event.target.value)}
                      placeholder={t('invoiceWithholding')}
                      value={editForm.withholdingLabel}
                    />
                  </label>
                  <label className={fieldClass(editErrors.withholding)} htmlFor="invoice-edit-withholding">
                    {resolveWithholdingLabel(editForm.withholdingLabel, t)} — {t('invoiceWithholdingAmount')}
                    <input
                      aria-describedby={editErrors.withholding ? 'invoice-edit-withholding-error' : undefined}
                      aria-invalid={Boolean(editErrors.withholding)}
                      id="invoice-edit-withholding"
                      inputMode="decimal"
                      onChange={(event) => updateEditField('withholding', event.target.value)}
                      placeholder="0.00"
                      value={editForm.withholding}
                    />
                    <FieldError id="invoice-edit-withholding-error" message={editErrors.withholding} />
                  </label>
                  <label className="form-field invoice-notes-field" htmlFor="invoice-edit-notes">
                    {t('invoiceNotes')}
                    <textarea
                      id="invoice-edit-notes"
                      onChange={(event) => updateEditField('notes', event.target.value)}
                      rows={2}
                      value={editForm.notes}
                    />
                  </label>
                  <div className="invoice-form-actions">
                    <button disabled={updateDraftMutation.isPending} type="submit">
                      {t('invoiceSaveDraft')}
                    </button>
                    <button
                      className="ghost-button"
                      onClick={() => {
                        setIsEditingDraft(false);
                        setEditForm(null);
                        setEditErrors({});
                      }}
                      type="button"
                    >
                      {t('cancel')}
                    </button>
                  </div>
                </form>
              ) : null}

              <div className="invoice-actions">
                {selectedInvoice.status === 'draft' ? (
                  <>
                    {!isEditingDraft ? (
                      <button className="ghost-button" onClick={() => startDraftEdit(selectedInvoice)} type="button">
                        <Pencil aria-hidden="true" />
                        {t('invoiceEditDraft')}
                      </button>
                    ) : null}
                    <button onClick={() => void handlePreview(selectedInvoice)} type="button">
                      {t('invoicePreview')}
                    </button>
                    <button disabled={issueMutation.isPending} onClick={() => issueMutation.mutate(selectedInvoice.id)} type="button">
                      {t('invoiceIssueOfficial')}
                    </button>
                    <button
                      className="danger-button"
                      disabled={deleteMutation.isPending}
                      onClick={() => {
                        if (confirmDestructiveAction(t('deleteDraftInvoiceConfirm'))) {
                          deleteMutation.mutate(selectedInvoice.id);
                        }
                      }}
                      type="button"
                    >
                      <Trash2 aria-hidden="true" />
                      {t('deletePermanently')}
                    </button>
                  </>
                ) : null}
                {selectedInvoice.status === 'issued' ? (
                  <button disabled={statusMutation.isPending} onClick={() => statusMutation.mutate({ invoiceId: selectedInvoice.id, status: 'paid' })} type="button">
                    {t('invoiceMarkPaid')}
                  </button>
                ) : null}
                {selectedInvoice.status === 'issued' ? (
                  <button className="ghost-button" disabled={cancelMutation.isPending} onClick={() => handleCancel(selectedInvoice)} type="button">
                    {t('invoiceCancel')}
                  </button>
                ) : null}
                {(selectedInvoice.documents ?? []).map((document) => (
                  <button
                    className="ghost-button"
                    key={document.id}
                    onClick={() =>
                      void handleDocumentDownload(
                        selectedInvoice,
                        document.id,
                        `${selectedInvoice.invoiceNumber}-${document.kind === 'work_protocol_pdf' ? 'work-protocol' : 'invoice'}.pdf`,
                      )
                    }
                    type="button"
                  >
                    <Download aria-hidden="true" />
                    {document.kind === 'work_protocol_pdf' ? t('invoiceDownloadWorkProtocol') : t('invoiceDownloadPdf')}
                  </button>
                ))}
                <button className="ghost-button" onClick={() => handleExport(selectedInvoice, 'html')} type="button">
                  <Download aria-hidden="true" />
                  {t('invoiceDownloadHtml')}
                </button>
                <button className="ghost-button" onClick={() => handleExport(selectedInvoice, 'csv')} type="button">
                  <Download aria-hidden="true" />
                  {t('reportDownloadCsv')}
                </button>
                <button className="ghost-button" onClick={() => handleExport(selectedInvoice, 'json')} type="button">
                  <FileJson aria-hidden="true" />
                  {t('reportDownloadJson')}
                </button>
              </div>
              {exportError ? <SurfaceError message={exportError} /> : null}
                </article>
              ) : (
                <SurfaceEmpty>
                  <p>{t('invoiceSelectOne')}</p>
                </SurfaceEmpty>
              )}
            </div>
          ) : null}
        </div>
      </div>
    </section>
  );
}

function validateDraftForm(form: DraftFormState, t: Translator): DraftFormErrors {
  const errors: DraftFormErrors = {};

  if (!form.clientId.trim()) {
    errors.clientId = t('invoiceClientRequired');
  }

  if (!form.from.trim() || !form.to.trim()) {
    if (!form.from.trim()) {
      errors.from = t('invoiceDateRangeRequired');
    }
    if (!form.to.trim()) {
      errors.to = t('invoiceDateRangeRequired');
    }
  } else {
    const fromDate = new Date(`${form.from}T00:00:00`);
    const toDate = new Date(`${form.to}T23:59:59`);
    if (Number.isNaN(fromDate.getTime())) {
      errors.from = t('invoiceDateInvalid');
    }
    if (Number.isNaN(toDate.getTime())) {
      errors.to = t('invoiceDateInvalid');
    }
    if (!errors.from && !errors.to && fromDate > toDate) {
      errors.to = t('invoiceDateRangeInvalid');
    }
  }

  const taxRatePercent = Number.parseFloat(form.taxRatePercent.replace(',', '.'));
  if (!Number.isFinite(taxRatePercent) || taxRatePercent < 0) {
    errors.taxRatePercent = t('invoiceTaxRateInvalid');
  }

  const withholdingRaw = form.withholding.trim();
  if (withholdingRaw !== '') {
    const withholding = Number.parseFloat(withholdingRaw.replace(',', '.'));
    if (!Number.isFinite(withholding) || withholding < 0) {
      errors.withholding = t('invoiceWithholdingInvalid');
    }
  }

  return errors;
}

function validateDraftEditForm(form: DraftEditFormState, t: Translator): DraftEditFormErrors {
  const errors: DraftEditFormErrors = {};
  const taxRatePercent = Number.parseFloat(form.taxRatePercent.replace(',', '.'));
  if (!Number.isFinite(taxRatePercent) || taxRatePercent < 0) {
    errors.taxRatePercent = t('invoiceTaxRateInvalid');
  }

  const withholdingRaw = form.withholding.trim();
  if (withholdingRaw !== '') {
    const withholding = Number.parseFloat(withholdingRaw.replace(',', '.'));
    if (!Number.isFinite(withholding) || withholding < 0) {
      errors.withholding = t('invoiceWithholdingInvalid');
    }
  }

  return errors;
}

function mapInvoiceDraftApiError(error: unknown, t: Translator): DraftFormErrors {
  if (!isApiError(error)) {
    return { form: t('invoiceDraftFailed') };
  }

  const errors: DraftFormErrors = {};
  for (const field of error.fields) {
    switch (field.field) {
      case 'clientId':
        errors.clientId = t('invoiceClientRequired');
        break;
      case 'from':
        if (field.message.includes('no billable uninvoiced')) {
          errors.from = t('invoiceNoBillableTimeInRange');
        } else if (field.message.includes('no billable time with positive duration')) {
          errors.from = t('invoiceNoBillableDuration');
        } else if (field.message.includes('required')) {
          errors.from = t('invoiceDateRangeRequired');
        } else {
          errors.from = t('invoiceDateInvalid');
        }
        break;
      case 'to':
        if (field.message.includes('on or after')) {
          errors.to = t('invoiceDateRangeInvalid');
        } else {
          errors.to = t('invoiceDateInvalid');
        }
        break;
      case 'taxRateBasisPoints':
        errors.taxRatePercent = t('invoiceTaxRateInvalid');
        break;
      case 'withholdingBasisPoints':
      case 'withholdingMinor':
        errors.withholding = t('invoiceWithholdingInvalid');
        break;
      default:
        break;
    }
  }

  if (!hasErrors(errors)) {
    errors.form = error.message || t('invoiceDraftFailed');
  }

  return errors;
}

function mapInvoiceDraftEditApiError(error: unknown, t: Translator): DraftEditFormErrors {
  if (!isApiError(error)) {
    return { form: t('invoiceDraftSaveFailed') };
  }

  const errors: DraftEditFormErrors = {};
  for (const field of error.fields) {
    switch (field.field) {
      case 'taxRateBasisPoints':
        errors.taxRatePercent = t('invoiceTaxRateInvalid');
        break;
      case 'withholdingBasisPoints':
      case 'withholdingMinor':
        errors.withholding = t('invoiceWithholdingInvalid');
        break;
      default:
        break;
    }
  }

  if (!hasErrors(errors)) {
    errors.form = error.message || t('invoiceDraftSaveFailed');
  }

  return errors;
}

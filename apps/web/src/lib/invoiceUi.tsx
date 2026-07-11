import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Download, FileJson, FileText, Pencil, Trash2 } from 'lucide-react';
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
  issueInvoice,
  previewInvoice,
  updateInvoice,
  updateInvoiceStatus,
  type Client,
  type Invoice,
  type InvoiceStatus,
  type Locale,
  type WorkProtocolDetail,
} from './api';
import { endOfMonth, startOfMonth, toMonthQueryFrom, toMonthQueryTo } from './calendarMonth';
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
  notes: string;
  seriesId: string;
  workProtocolDetail: WorkProtocolDetail;
};

type DraftEditFormState = {
  notes: string;
  taxRatePercent: string;
  withholding: string;
  seriesId: string;
  workProtocolDetail: WorkProtocolDetail;
};

function invoiceToEditForm(invoice: Invoice): DraftEditFormState {
  const taxRateBasisPoints = invoice.lines[0]?.taxRateBasisPoints ?? 2100;
  return {
    notes: invoice.notes,
    taxRatePercent: String(taxRateBasisPoints / 100),
    withholding: invoice.withholdingMinor > 0 ? (invoice.withholdingMinor / 100).toFixed(2) : '',
    seriesId: invoice.seriesId ?? '',
    workProtocolDetail: invoice.workProtocolDetail ?? 'standard',
  };
}

function defaultDraftForm(): DraftFormState {
  const monthStart = startOfMonth(new Date());
  const monthEnd = endOfMonth(monthStart);
  return {
    clientId: '',
    from: toMonthQueryFrom(monthStart).slice(0, 10),
    to: toMonthQueryTo(monthEnd).slice(0, 10),
    taxRatePercent: '21',
    withholding: '',
    notes: '',
    seriesId: '',
    workProtocolDetail: 'standard',
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
  locale,
  t,
  userName,
}: {
  clients: Client[];
  locale: Locale;
  t: Translator;
  userName: string;
}) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const [form, setForm] = useState<DraftFormState>(defaultDraftForm);
  const [selectedInvoiceId, setSelectedInvoiceId] = useState<string | null>(null);
  const [isEditingDraft, setIsEditingDraft] = useState(false);
  const [editForm, setEditForm] = useState<DraftEditFormState | null>(null);
  const [editError, setEditError] = useState('');
  const [formError, setFormError] = useState('');
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
    setIsEditingDraft(false);
    setEditForm(null);
    setEditError('');
  }, [selectedInvoiceId]);

  const createMutation = useMutation({
    mutationFn: () => {
      const taxRatePercent = Number.parseFloat(form.taxRatePercent.replace(',', '.'));
      const withholding = form.withholding.trim() === '' ? 0 : Math.round(Number.parseFloat(form.withholding.replace(',', '.')) * 100);
      if (!form.clientId) {
        throw new Error('client_required');
      }
      if (!Number.isFinite(taxRatePercent) || taxRatePercent < 0) {
        throw new Error('tax_invalid');
      }
      if (!Number.isFinite(withholding) || withholding < 0) {
        throw new Error('withholding_invalid');
      }
      return createInvoiceDraftFromTime({
        clientId: form.clientId,
        from: toInvoiceQueryFrom(form.from),
        to: toInvoiceQueryTo(form.to),
        sellerName: userName,
        taxRateBasisPoints: Math.round(taxRatePercent * 100),
        withholdingMinor: withholding,
        notes: form.notes.trim(),
        seriesId: form.seriesId || defaultSeriesId || undefined,
        workProtocolDetail: form.workProtocolDetail,
      });
    },
    onSuccess: (invoice) => {
      setFormError('');
      setSelectedInvoiceId(invoice.id);
      queryClient.invalidateQueries({ queryKey: ['invoices'] });
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] });
      toast.success(t('invoiceDraftCreated'));
    },
    onError: () => {
      setFormError(t('invoiceDraftFailed'));
      toast.error(t('invoiceDraftFailed'));
    },
  });

  const updateDraftMutation = useMutation({
    mutationFn: (input: { invoiceId: string; form: DraftEditFormState }) => {
      const taxRatePercent = Number.parseFloat(input.form.taxRatePercent.replace(',', '.'));
      const withholding =
        input.form.withholding.trim() === '' ? 0 : Math.round(Number.parseFloat(input.form.withholding.replace(',', '.')) * 100);
      if (!Number.isFinite(taxRatePercent) || taxRatePercent < 0) {
        throw new Error('tax_invalid');
      }
      if (!Number.isFinite(withholding) || withholding < 0) {
        throw new Error('withholding_invalid');
      }
      return updateInvoice(input.invoiceId, {
        notes: input.form.notes.trim(),
        taxRateBasisPoints: Math.round(taxRatePercent * 100),
        withholdingMinor: withholding,
        seriesId: input.form.seriesId || undefined,
        workProtocolDetail: input.form.workProtocolDetail,
      });
    },
    onSuccess: (invoice) => {
      setEditError('');
      setIsEditingDraft(false);
      setEditForm(null);
      queryClient.invalidateQueries({ queryKey: ['invoices'] });
      queryClient.invalidateQueries({ queryKey: ['invoice', invoice.id] });
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] });
      toast.success(t('invoiceDraftSaved'));
    },
    onError: () => {
      setEditError(t('invoiceDraftSaveFailed'));
      toast.error(t('invoiceDraftSaveFailed'));
    },
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
    setFormError('');
    createMutation.mutate();
  }

  function submitDraftEdit(event: FormEvent) {
    event.preventDefault();
    if (!selectedInvoice || !editForm) {
      return;
    }
    setEditError('');
    updateDraftMutation.mutate({ invoiceId: selectedInvoice.id, form: editForm });
  }

  function startDraftEdit(invoice: Invoice) {
    setIsEditingDraft(true);
    setEditForm(invoiceToEditForm(invoice));
    setEditError('');
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
            <label className="form-field">
              {t('invoiceClient')}
              <select
                onChange={(event) => setForm((current) => ({ ...current, clientId: event.target.value }))}
                required
                value={form.clientId}
              >
                <option value="">{t('invoiceClientPlaceholder')}</option>
                {activeClients.map((client) => (
                  <option key={client.id} value={client.id}>
                    {client.name}
                  </option>
                ))}
              </select>
            </label>
            <label className="form-field">
              {t('invoiceSeries')}
              <select
                onChange={(event) => setForm((current) => ({ ...current, seriesId: event.target.value }))}
                value={form.seriesId || defaultSeriesId}
              >
                {invoiceSeries.map((series) => (
                  <option key={series.id} value={series.id}>
                    {series.code} — {series.name}
                  </option>
                ))}
              </select>
            </label>
            <label className="form-field">
              {t('invoiceWorkProtocolDetail')}
              <select
                onChange={(event) =>
                  setForm((current) => ({ ...current, workProtocolDetail: event.target.value as WorkProtocolDetail }))
                }
                value={form.workProtocolDetail}
              >
                <option value="summary">{t('invoiceWorkProtocolSummary')}</option>
                <option value="standard">{t('invoiceWorkProtocolStandard')}</option>
                <option value="detailed">{t('invoiceWorkProtocolDetailed')}</option>
              </select>
            </label>
            <label className="form-field">
              {t('reportFrom')}
              <input onChange={(event) => setForm((current) => ({ ...current, from: event.target.value }))} type="date" value={form.from} />
            </label>
            <label className="form-field">
              {t('reportTo')}
              <input onChange={(event) => setForm((current) => ({ ...current, to: event.target.value }))} type="date" value={form.to} />
            </label>
            <label className="form-field">
              {t('invoiceTaxRate')}
              <input
                inputMode="decimal"
                onChange={(event) => setForm((current) => ({ ...current, taxRatePercent: event.target.value }))}
                placeholder="21"
                value={form.taxRatePercent}
              />
            </label>
            <label className="form-field">
              {t('invoiceWithholding')}
              <input
                inputMode="decimal"
                onChange={(event) => setForm((current) => ({ ...current, withholding: event.target.value }))}
                placeholder="0.00"
                value={form.withholding}
              />
            </label>
            <label className="form-field invoice-notes-field">
              {t('invoiceNotes')}
              <textarea onChange={(event) => setForm((current) => ({ ...current, notes: event.target.value }))} rows={2} value={form.notes} />
            </label>
            <div className="invoice-form-actions">
              <button disabled={createMutation.isPending} type="submit">
                {t('invoiceCreateDraft')}
              </button>
            </div>
          </form>
          {formError ? <SurfaceError message={formError} /> : null}
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
                  <dt>{t('invoiceSubtotal')}</dt>
                  <dd>{formatMoneyMinor(selectedInvoice.subtotalMinor, selectedInvoice.currency, locale)}</dd>
                </div>
                <div>
                  <dt>{t('invoiceTax')}</dt>
                  <dd>{formatMoneyMinor(selectedInvoice.taxMinor, selectedInvoice.currency, locale)}</dd>
                </div>
                {selectedInvoice.withholdingMinor > 0 ? (
                  <div>
                    <dt>{t('invoiceWithholding')}</dt>
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
                    <th>{t('invoiceMinutes')}</th>
                    <th>{t('hourlyRate')}</th>
                    <th>{t('invoiceSubtotal')}</th>
                  </tr>
                </thead>
                <tbody>
                  {(selectedInvoice.lines ?? []).map((line) => (
                    <tr key={line.id}>
                      <td>{line.description}</td>
                      <td>{line.quantityMinutes}</td>
                      <td>{formatMoneyMinor(line.unitRateMinor, selectedInvoice.currency, locale)}</td>
                      <td>{formatMoneyMinor(line.subtotalMinor, selectedInvoice.currency, locale)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>

              {selectedInvoice.notes && !isEditingDraft ? <p className="invoice-notes">{selectedInvoice.notes}</p> : null}

              {selectedInvoice.status === 'draft' && isEditingDraft && editForm ? (
                <form className="invoice-edit-form" noValidate onSubmit={submitDraftEdit}>
                  <label className="form-field">
                    {t('invoiceSeries')}
                    <select
                      onChange={(event) => setEditForm((current) => (current ? { ...current, seriesId: event.target.value } : current))}
                      value={editForm.seriesId || defaultSeriesId}
                    >
                      {invoiceSeries.map((series) => (
                        <option key={series.id} value={series.id}>
                          {series.code} — {series.name}
                        </option>
                      ))}
                    </select>
                  </label>
                  <label className="form-field">
                    {t('invoiceWorkProtocolDetail')}
                    <select
                      onChange={(event) =>
                        setEditForm((current) =>
                          current ? { ...current, workProtocolDetail: event.target.value as WorkProtocolDetail } : current,
                        )
                      }
                      value={editForm.workProtocolDetail}
                    >
                      <option value="summary">{t('invoiceWorkProtocolSummary')}</option>
                      <option value="standard">{t('invoiceWorkProtocolStandard')}</option>
                      <option value="detailed">{t('invoiceWorkProtocolDetailed')}</option>
                    </select>
                  </label>
                  <label className="form-field">
                    {t('invoiceTaxRate')}
                    <input
                      inputMode="decimal"
                      onChange={(event) => setEditForm((current) => (current ? { ...current, taxRatePercent: event.target.value } : current))}
                      value={editForm.taxRatePercent}
                    />
                  </label>
                  <label className="form-field">
                    {t('invoiceWithholding')}
                    <input
                      inputMode="decimal"
                      onChange={(event) => setEditForm((current) => (current ? { ...current, withholding: event.target.value } : current))}
                      placeholder="0.00"
                      value={editForm.withholding}
                    />
                  </label>
                  <label className="form-field invoice-notes-field">
                    {t('invoiceNotes')}
                    <textarea
                      onChange={(event) => setEditForm((current) => (current ? { ...current, notes: event.target.value } : current))}
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
                        setEditError('');
                      }}
                      type="button"
                    >
                      {t('cancel')}
                    </button>
                  </div>
                </form>
              ) : null}
              {editError ? <SurfaceError message={editError} /> : null}

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

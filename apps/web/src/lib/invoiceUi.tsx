import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Download, FileJson, FileText, Trash2 } from 'lucide-react';
import { FormEvent, useMemo, useState } from 'react';
import {
  createInvoiceDraftFromTime,
  deleteInvoice,
  downloadInvoiceExport,
  fetchInvoice,
  fetchInvoices,
  updateInvoiceStatus,
  type Client,
  type Invoice,
  type InvoiceStatus,
  type Locale,
} from './api';
import { endOfMonth, startOfMonth, toMonthQueryFrom, toMonthQueryTo } from './calendarMonth';
import type { Translator } from './timeEntryUi';
import { useToast } from './toast';

type DraftFormState = {
  clientId: string;
  from: string;
  taxRatePercent: string;
  to: string;
  withholding: string;
  notes: string;
};

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
  const [formError, setFormError] = useState('');
  const [exportError, setExportError] = useState('');

  const invoicesQuery = useQuery({
    queryKey: ['invoices'],
    queryFn: fetchInvoices,
    retry: false,
  });

  const invoiceDetailQuery = useQuery({
    queryKey: ['invoice', selectedInvoiceId],
    queryFn: () => fetchInvoice(selectedInvoiceId as string),
    enabled: selectedInvoiceId != null,
  });

  const activeClients = useMemo(() => clients.filter((client) => !client.archivedAt), [clients]);
  const invoices = invoicesQuery.data?.invoices ?? [];
  const selectedInvoice = invoiceDetailQuery.data ?? invoices.find((invoice) => invoice.id === selectedInvoiceId) ?? null;

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
      });
    },
    onSuccess: (invoice) => {
      setFormError('');
      setSelectedInvoiceId(invoice.id);
      queryClient.invalidateQueries({ queryKey: ['invoices'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
      toast.success(t('invoiceDraftCreated'));
    },
    onError: () => {
      setFormError(t('invoiceDraftFailed'));
      toast.error(t('invoiceDraftFailed'));
    },
  });

  const statusMutation = useMutation({
    mutationFn: ({ invoiceId, status }: { invoiceId: string; status: InvoiceStatus }) =>
      updateInvoiceStatus(invoiceId, status),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['invoices'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
      toast.success(t('invoiceStatusUpdated'));
    },
    onError: () => toast.error(t('invoiceStatusUpdateFailed')),
  });

  const deleteMutation = useMutation({
    mutationFn: deleteInvoice,
    onSuccess: () => {
      setSelectedInvoiceId(null);
      queryClient.invalidateQueries({ queryKey: ['invoices'] });
      queryClient.invalidateQueries({ queryKey: ['overview'] });
      toast.success(t('invoiceDeleted'));
    },
    onError: () => toast.error(t('invoiceDeleteFailed')),
  });

  function submitDraft(event: FormEvent) {
    event.preventDefault();
    setFormError('');
    createMutation.mutate();
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
      {formError ? <p className="form-error">{formError}</p> : null}

      {invoicesQuery.isError ? <p className="form-error">{t('invoiceLoadFailed')}</p> : null}
      {invoicesQuery.isLoading ? <p>{t('loading')}</p> : null}
      {!invoicesQuery.isLoading && invoices.length === 0 ? <p className="empty-state">{t('invoiceNoInvoices')}</p> : null}

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

              {selectedInvoice.notes ? <p className="invoice-notes">{selectedInvoice.notes}</p> : null}

              <div className="invoice-actions">
                {selectedInvoice.status === 'draft' ? (
                  <>
                    <button disabled={statusMutation.isPending} onClick={() => statusMutation.mutate({ invoiceId: selectedInvoice.id, status: 'issued' })} type="button">
                      {t('invoiceIssue')}
                    </button>
                    <button
                      className="danger-button"
                      disabled={deleteMutation.isPending}
                      onClick={() => deleteMutation.mutate(selectedInvoice.id)}
                      type="button"
                    >
                      <Trash2 aria-hidden="true" />
                      {t('delete')}
                    </button>
                  </>
                ) : null}
                {selectedInvoice.status === 'issued' ? (
                  <button disabled={statusMutation.isPending} onClick={() => statusMutation.mutate({ invoiceId: selectedInvoice.id, status: 'paid' })} type="button">
                    {t('invoiceMarkPaid')}
                  </button>
                ) : null}
                {selectedInvoice.status !== 'cancelled' && selectedInvoice.status !== 'paid' ? (
                  <button
                    className="ghost-button"
                    disabled={statusMutation.isPending}
                    onClick={() => statusMutation.mutate({ invoiceId: selectedInvoice.id, status: 'cancelled' })}
                    type="button"
                  >
                    {t('invoiceCancel')}
                  </button>
                ) : null}
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
              {exportError ? <p className="form-error">{exportError}</p> : null}
            </article>
          ) : (
            <p className="empty-state">{t('invoiceSelectOne')}</p>
          )}
        </div>
      ) : null}
    </section>
  );
}

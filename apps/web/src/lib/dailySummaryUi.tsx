import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Check, ClipboardCopy, MessageSquareText, RefreshCw, Sparkles } from 'lucide-react';
import { FormEvent, useEffect, useMemo, useState } from 'react';
import {
  applyDailySummaryEnrichment,
  approveDailySummaryRecord,
  enrichDailySummaryLocally,
  fetchDailySummaryEnrichContext,
  fetchDailySummaryRecord,
  generateDailySummaryRecord,
  reopenDailySummaryRecord,
  saveDailySummaryDraft,
  type DailySummaryParams,
  type DailySummaryRecord,
} from './api';
import { SurfaceEmpty, SurfaceError, SurfaceLoading } from './feedbackUi';
import type { Translator } from './timeEntryUi';
import { useToast } from './toast';

type DailySummaryFormState = {
  billableOnly: boolean;
  date: string;
  feedback: string;
  includeClient: boolean;
  includeClosing: boolean;
  includeProject: boolean;
  note: string;
};

function todayInputValue(): string {
  const today = new Date();
  const month = String(today.getMonth() + 1).padStart(2, '0');
  const day = String(today.getDate()).padStart(2, '0');
  return `${today.getFullYear()}-${month}-${day}`;
}

function defaultDailySummaryForm(): DailySummaryFormState {
  return {
    date: todayInputValue(),
    includeClient: true,
    includeProject: true,
    includeClosing: true,
    billableOnly: false,
    note: '',
    feedback: '',
  };
}

function formToParams(form: DailySummaryFormState): DailySummaryParams {
  return {
    date: form.date,
    includeClient: form.includeClient,
    includeProject: form.includeProject,
    includeClosing: form.includeClosing,
    billableOnly: form.billableOnly,
  };
}

export function DailySummaryPanel({ t }: { t: Translator }) {
  const toast = useToast();
  const queryClient = useQueryClient();
  const [form, setForm] = useState<DailySummaryFormState>(() => defaultDailySummaryForm());
  const [draftText, setDraftText] = useState('');
  const [copyState, setCopyState] = useState<'idle' | 'copied' | 'failed'>('idle');

  const params = useMemo(() => formToParams(form), [form]);

  const recordQuery = useQuery({
    queryKey: ['daily-summary-record', form.date],
    queryFn: () => fetchDailySummaryRecord(form.date),
    retry: false,
  });

  const record = recordQuery.data ?? null;
  const isApproved = record?.status === 'approved';

  useEffect(() => {
    if (!record) {
      return;
    }
    setDraftText(record.status === 'approved' ? record.approvedText : record.draftText);
    setForm((current) => ({
      ...current,
      note: record.manualNote || current.note,
    }));
  }, [record]);

  const generateMutation = useMutation({
    mutationFn: () => generateDailySummaryRecord(form.date, params, form.note),
    onSuccess: (saved) => {
      setDraftText(saved.draftText);
      void queryClient.invalidateQueries({ queryKey: ['daily-summary-record', form.date] });
      toast.success(t('dailySummaryGenerated'));
    },
    onError: () => toast.error(t('dailySummaryLoadFailed')),
  });

  const saveMutation = useMutation({
    mutationFn: () =>
      saveDailySummaryDraft(form.date, {
        draftText,
        manualNote: form.note,
        options: params,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['daily-summary-record', form.date] });
      toast.success(t('dailySummarySaved'));
    },
    onError: () => toast.error(t('dailySummarySaveFailed')),
  });

  const enrichMutation = useMutation({
    mutationFn: async () => {
      const context = await fetchDailySummaryEnrichContext(form.date, params, form.note);
      const enriched = await enrichDailySummaryLocally({
        date: form.date,
        templateText: context.templateText,
        manualNote: form.note,
        feedback: form.feedback,
        currentDraft: draftText,
        locale: context.locale,
        authorEmail: context.authorEmail,
        projects: context.projects,
      });
      return applyDailySummaryEnrichment(form.date, {
        text: enriched.text,
        manualNote: form.note,
        generationSource: enriched.source,
      });
    },
    onSuccess: (saved) => {
      setDraftText(saved.draftText);
      setForm((current) => ({ ...current, feedback: '' }));
      void queryClient.invalidateQueries({ queryKey: ['daily-summary-record', form.date] });
      toast.success(t('dailySummaryEnriched'));
    },
    onError: () => toast.error(t('dailySummaryEnrichFailed')),
  });

  const approveMutation = useMutation({
    mutationFn: () => approveDailySummaryRecord(form.date, draftText),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['daily-summary-record', form.date] });
      toast.success(t('dailySummaryApprovedToast'));
    },
    onError: () => toast.error(t('dailySummaryApproveFailed')),
  });

  const reopenMutation = useMutation({
    mutationFn: () => reopenDailySummaryRecord(form.date),
    onSuccess: (saved) => {
      setDraftText(saved.draftText);
      void queryClient.invalidateQueries({ queryKey: ['daily-summary-record', form.date] });
      toast.success(t('dailySummaryReopened'));
    },
    onError: () => toast.error(t('dailySummaryReopenFailed')),
  });

  function submitPreview(event: FormEvent) {
    event.preventDefault();
    generateMutation.mutate();
  }

  async function handleCopy() {
    if (!draftText.trim()) {
      return;
    }
    try {
      await navigator.clipboard.writeText(draftText);
      setCopyState('copied');
      toast.success(t('dailySummaryCopySuccess'));
    } catch {
      setCopyState('failed');
      toast.error(t('dailySummaryCopyFailed'));
    }
  }

  const busy =
    generateMutation.isPending ||
    saveMutation.isPending ||
    enrichMutation.isPending ||
    approveMutation.isPending ||
    reopenMutation.isPending;

  return (
    <section className="clients-section report-section daily-summary-section" id="daily-summary" aria-labelledby="daily-summary-title">
      <div className="clients-heading">
        <div className="section-title-group">
          <span className="section-kicker">
            <MessageSquareText aria-hidden="true" />
            {t('dailySummary')}
          </span>
          <h2 id="daily-summary-title">{t('dailySummaryTitle')}</h2>
          <p>{t('dailySummarySubtitle')}</p>
          {record ? (
            <p className="daily-summary-status">
              {isApproved ? t('dailySummaryApprovedStatus') : t('dailySummaryDraftStatus')}
              {record.generationCount > 0 ? ` · ${t('dailySummaryGenerationCount').replace('{count}', String(record.generationCount))}` : ''}
            </p>
          ) : null}
        </div>
      </div>

      <div className="report-workbench daily-summary-workbench">
        <aside className="report-filters-panel">
          <h3>{t('dailySummaryOptions')}</h3>
          <form className="report-form" noValidate onSubmit={submitPreview}>
            <label className="form-field">
              {t('dailySummaryDate')}
              <input
                disabled={busy}
                onChange={(event) => setForm((current) => ({ ...current, date: event.target.value }))}
                type="date"
                value={form.date}
              />
            </label>
            <label className="checkbox-field">
              <input
                checked={form.includeClient}
                disabled={isApproved}
                onChange={(event) => setForm((current) => ({ ...current, includeClient: event.target.checked }))}
                type="checkbox"
              />
              <span>{t('dailySummaryIncludeClient')}</span>
            </label>
            <label className="checkbox-field">
              <input
                checked={form.includeProject}
                disabled={isApproved}
                onChange={(event) => setForm((current) => ({ ...current, includeProject: event.target.checked }))}
                type="checkbox"
              />
              <span>{t('dailySummaryIncludeProject')}</span>
            </label>
            <label className="checkbox-field">
              <input
                checked={form.includeClosing}
                disabled={isApproved}
                onChange={(event) => setForm((current) => ({ ...current, includeClosing: event.target.checked }))}
                type="checkbox"
              />
              <span>{t('dailySummaryIncludeClosing')}</span>
            </label>
            <label className="checkbox-field">
              <input
                checked={form.billableOnly}
                disabled={isApproved}
                onChange={(event) => setForm((current) => ({ ...current, billableOnly: event.target.checked }))}
                type="checkbox"
              />
              <span>{t('reportBillableOnly')}</span>
            </label>
            <label className="form-field">
              {t('dailySummaryNote')}
              <textarea
                disabled={isApproved}
                onChange={(event) => setForm((current) => ({ ...current, note: event.target.value }))}
                placeholder={t('dailySummaryNotePlaceholder')}
                rows={3}
                value={form.note}
              />
            </label>
            <label className="form-field">
              {t('dailySummaryFeedback')}
              <textarea
                disabled={isApproved}
                onChange={(event) => setForm((current) => ({ ...current, feedback: event.target.value }))}
                placeholder={t('dailySummaryFeedbackPlaceholder')}
                rows={3}
                value={form.feedback}
              />
            </label>
            <div className="report-form-actions">
              <button className="secondary-button" disabled={busy || isApproved} type="submit">
                <RefreshCw aria-hidden="true" />
                {t('dailySummaryGenerate')}
              </button>
              <button
                className="secondary-button"
                disabled={busy || isApproved}
                onClick={() => enrichMutation.mutate()}
                type="button"
              >
                <Sparkles aria-hidden="true" />
                {t('dailySummaryEnrich')}
              </button>
            </div>
          </form>
        </aside>

        <div className="report-results-panel daily-summary-results" aria-live="polite">
          <div className="report-results-header">
            <h3>{t('dailySummaryPreview')}</h3>
          </div>

          <div className="report-preview">
            {recordQuery.isLoading ? <SurfaceLoading label={t('loading')} /> : null}
            {recordQuery.isError ? (
              <SurfaceError message={t('dailySummaryLoadFailed')} onRetry={() => void recordQuery.refetch()} retryLabel={t('retry')} />
            ) : null}
            {!recordQuery.isLoading && !draftText ? (
              <SurfaceEmpty>
                <p>{t('dailySummaryNoDraft')}</p>
              </SurfaceEmpty>
            ) : null}
            {draftText ? (
              <>
                <label className="form-field daily-summary-editor" htmlFor="daily-summary-text">
                  <span>{t('dailySummarySlackText')}</span>
                  <textarea
                    id="daily-summary-text"
                    onChange={(event) => {
                      setDraftText(event.target.value);
                      setCopyState('idle');
                    }}
                    readOnly={isApproved}
                    rows={14}
                    value={draftText}
                  />
                </label>
                <div className="report-form-actions">
                  <button disabled={!draftText.trim() || busy} onClick={() => void handleCopy()} type="button">
                    <ClipboardCopy aria-hidden="true" />
                    {copyState === 'copied' ? t('dailySummaryCopied') : t('dailySummaryCopySlack')}
                  </button>
                  {!isApproved ? (
                    <>
                      <button
                        className="secondary-button"
                        disabled={!draftText.trim() || busy}
                        onClick={() => saveMutation.mutate()}
                        type="button"
                      >
                        {t('dailySummarySaveDraft')}
                      </button>
                      <button disabled={!draftText.trim() || busy} onClick={() => approveMutation.mutate()} type="button">
                        <Check aria-hidden="true" />
                        {t('dailySummaryApprove')}
                      </button>
                    </>
                  ) : (
                    <button className="secondary-button" disabled={busy} onClick={() => reopenMutation.mutate()} type="button">
                      {t('dailySummaryReopen')}
                    </button>
                  )}
                </div>
              </>
            ) : null}
          </div>
        </div>
      </div>
    </section>
  );
}

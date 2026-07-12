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
  type Client,
  type DailySummaryParams,
  type Project,
} from './api';
import { SurfaceEmpty, SurfaceError, SurfaceLoading } from './feedbackUi';
import type { Translator } from './timeEntryUi';
import { useToast } from './toast';

type DailySummaryFormState = {
  billableOnly: boolean;
  clientId: string;
  date: string;
  feedback: string;
  includeClient: boolean;
  includeClosing: boolean;
  includeProject: boolean;
  note: string;
  projectId: string;
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
    clientId: '',
    projectId: '',
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
    clientId: form.clientId,
    projectId: form.projectId,
    includeClient: form.includeClient,
    includeProject: form.includeProject,
    includeClosing: form.includeClosing,
    billableOnly: form.billableOnly,
  };
}

function scopeLabel(
  form: DailySummaryFormState,
  clients: Client[],
  projects: Project[],
  t: Translator,
): string {
  if (form.projectId) {
    const project = projects.find((item) => item.id === form.projectId);
    return project?.name ?? t('dailySummaryScopeProject');
  }
  if (form.clientId) {
    const client = clients.find((item) => item.id === form.clientId);
    return client?.name ?? t('dailySummaryScopeClient');
  }
  return t('dailySummaryScopeAll');
}

export function DailySummaryPanel({
  clients,
  projects,
  t,
}: {
  clients: Client[];
  projects: Project[];
  t: Translator;
}) {
  const toast = useToast();
  const queryClient = useQueryClient();
  const [form, setForm] = useState<DailySummaryFormState>(() => defaultDailySummaryForm());
  const [draftText, setDraftText] = useState('');
  const [copyState, setCopyState] = useState<'idle' | 'copied' | 'failed'>('idle');

  const params = useMemo(() => formToParams(form), [form]);
  const activeClients = useMemo(() => clients.filter((client) => !client.archivedAt), [clients]);
  const scopedProjects = useMemo(() => {
    let list = projects.filter((project) => !project.archivedAt);
    if (form.clientId) {
      list = list.filter((project) => project.clientId === form.clientId);
    }
    return list;
  }, [form.clientId, projects]);

  const recordQuery = useQuery({
    queryKey: ['daily-summary-record', form.date, form.clientId, form.projectId],
    queryFn: () => fetchDailySummaryRecord(form.date, { clientId: form.clientId, projectId: form.projectId }),
    retry: false,
  });

  const record = recordQuery.data ?? null;
  const isApproved = record?.status === 'approved';
  const currentScope = scopeLabel(form, clients, projects, t);

  useEffect(() => {
    setDraftText('');
    setCopyState('idle');
  }, [form.date, form.clientId, form.projectId]);

  useEffect(() => {
    if (!record || recordQuery.isFetching) {
      return;
    }
    setDraftText(record.status === 'approved' ? record.approvedText : record.draftText);
    setForm((current) => ({
      ...current,
      note: record.manualNote || current.note,
      clientId: record.clientId || current.clientId,
      projectId: record.projectId || current.projectId,
      includeClient: record.options.includeClient ?? current.includeClient,
      includeProject: record.options.includeProject ?? current.includeProject,
      includeClosing: record.options.includeClosing ?? current.includeClosing,
      billableOnly: record.options.billableOnly ?? current.billableOnly,
    }));
  }, [record, recordQuery.isFetching]);

  const generateMutation = useMutation({
    mutationFn: () => generateDailySummaryRecord(form.date, params, form.note),
    onSuccess: (saved) => {
      setDraftText(saved.draftText);
      void queryClient.invalidateQueries({ queryKey: ['daily-summary-record', form.date, form.clientId, form.projectId] });
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
      void queryClient.invalidateQueries({ queryKey: ['daily-summary-record', form.date, form.clientId, form.projectId] });
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
        options: params,
      });
    },
    onSuccess: (saved) => {
      setDraftText(saved.draftText);
      setForm((current) => ({ ...current, feedback: '' }));
      void queryClient.invalidateQueries({ queryKey: ['daily-summary-record', form.date, form.clientId, form.projectId] });
      toast.success(t('dailySummaryEnriched'));
    },
    onError: () => toast.error(t('dailySummaryEnrichFailed')),
  });

  const approveMutation = useMutation({
    mutationFn: () => approveDailySummaryRecord(form.date, draftText, { clientId: form.clientId, projectId: form.projectId }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['daily-summary-record', form.date, form.clientId, form.projectId] });
      toast.success(t('dailySummaryApprovedToast'));
    },
    onError: () => toast.error(t('dailySummaryApproveFailed')),
  });

  const reopenMutation = useMutation({
    mutationFn: () => reopenDailySummaryRecord(form.date, { clientId: form.clientId, projectId: form.projectId }),
    onSuccess: (saved) => {
      setDraftText(saved.draftText);
      void queryClient.invalidateQueries({ queryKey: ['daily-summary-record', form.date, form.clientId, form.projectId] });
      toast.success(t('dailySummaryReopened'));
    },
    onError: () => toast.error(t('dailySummaryReopenFailed')),
  });

  function updateClientId(clientId: string) {
    setForm((current) => {
      const nextProjectId =
        clientId && current.projectId
          ? projects.some((project) => project.id === current.projectId && project.clientId === clientId)
            ? current.projectId
            : ''
          : current.projectId;
      return { ...current, clientId, projectId: nextProjectId };
    });
  }

  function updateProjectId(projectId: string) {
    setForm((current) => {
      const project = projects.find((item) => item.id === projectId);
      return {
        ...current,
        projectId,
        clientId: project?.clientId ?? current.clientId,
      };
    });
  }

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
          <p className="daily-summary-scope-label">
            {t('dailySummaryScopeLabel').replace('{scope}', currentScope)}
          </p>
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
            <label className="form-field" htmlFor="daily-summary-client">
              {t('dailySummaryFilterClient')}
              <select
                disabled={isApproved}
                id="daily-summary-client"
                onChange={(event) => updateClientId(event.target.value)}
                value={form.clientId}
              >
                <option value="">{t('dailySummaryAllClients')}</option>
                {activeClients.map((client) => (
                  <option key={client.id} value={client.id}>
                    {client.name}
                  </option>
                ))}
              </select>
            </label>
            <label className="form-field" htmlFor="daily-summary-project">
              {t('dailySummaryFilterProject')}
              <select
                disabled={isApproved}
                id="daily-summary-project"
                onChange={(event) => updateProjectId(event.target.value)}
                value={form.projectId}
              >
                <option value="">{form.clientId ? t('dailySummaryAllClientProjects') : t('dailySummaryAllProjects')}</option>
                {scopedProjects.map((project) => (
                  <option key={project.id} value={project.id}>
                    {project.clientName ? `${project.clientName} — ${project.name}` : project.name}
                  </option>
                ))}
              </select>
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

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Check, ChevronLeft, ChevronRight, ClipboardCopy, MessageSquareText, RefreshCw, Sparkles } from 'lucide-react';
import { FormEvent, useEffect, useMemo, useState } from 'react';
import {
  applyDailySummaryEnrichment,
  approveDailySummaryRecord,
  enrichDailySummaryLocally,
  fetchDailySummaryAIUsage,
  fetchDailySummaryEnrichContext,
  fetchDailySummaryIndex,
  fetchDailySummaryRecord,
  generateDailySummaryRecord,
  reopenDailySummaryRecord,
  saveDailySummaryDraft,
  type Client,
  type DailySummaryIndexItem,
  type DailySummaryParams,
  type Locale,
  type Project,
} from './api';
import {
  addMonths,
  buildMonthGrid,
  endOfMonth,
  formatMonthLabel,
  isSameLocalDay,
  isSameMonth,
  weekdayLabels,
} from './calendarMonth';
import {
  buildDailySummaryEnrichConfirmMessage,
  DailySummaryAIUsageChip,
  DailySummaryLastRunBadge,
} from './dailySummaryCostUi';
import { SurfaceEmpty, SurfaceError, SurfaceLoading } from './feedbackUi';
import {
  DailySummaryProgressOverlay,
  DailySummaryStatusBadge,
  type DailySummaryWorkflowStep,
} from './dailySummaryProgress';
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

type WorkflowProgress =
  | { kind: 'generate'; step: string }
  | { kind: 'enrich'; step: string }
  | null;

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

function monthAnchorFromDate(date: string): Date {
  const [year, month] = date.split('-').map(Number);
  return new Date(year, month - 1, 1);
}

function toDateKey(date: Date): string {
  const pad = (value: number) => String(value).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
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

function formatStatusTimestamp(value: string, locale: Locale): string {
  if (!value) {
    return '';
  }
  const parsed = Date.parse(value);
  if (Number.isNaN(parsed)) {
    return '';
  }
  return new Date(parsed).toLocaleString(locale === 'es' ? 'es-ES' : 'en-US', {
    day: '2-digit',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function generationSourceLabel(source: string, t: Translator): string {
  if (source === 'cursor') {
    return t('dailySummarySourceCursor');
  }
  if (source === 'context') {
    return t('dailySummarySourceContext');
  }
  return t('dailySummarySourceTemplate');
}

function aggregateSummaryIndex(items: DailySummaryIndexItem[]): Map<string, DailySummaryIndexItem> {
  const map = new Map<string, DailySummaryIndexItem>();
  for (const item of items) {
    const existing = map.get(item.date);
    if (!existing) {
      map.set(item.date, item);
      continue;
    }
    if (existing.status === 'approved' || item.status === 'approved') {
      map.set(item.date, {
        ...existing,
        status: 'approved',
        updatedAt: item.updatedAt > existing.updatedAt ? item.updatedAt : existing.updatedAt,
      });
      continue;
    }
    map.set(item.date, item.updatedAt > existing.updatedAt ? item : existing);
  }
  return map;
}

function DailySummaryMonthCalendar({
  indexByDate,
  locale,
  monthAnchor,
  onNextMonth,
  onPreviousMonth,
  onSelectDay,
  onTodayMonth,
  selectedDate,
  t,
}: {
  indexByDate: Map<string, DailySummaryIndexItem>;
  locale: Locale;
  monthAnchor: Date;
  onNextMonth: () => void;
  onPreviousMonth: () => void;
  onSelectDay: (date: string) => void;
  onTodayMonth: () => void;
  selectedDate: string;
  t: Translator;
}) {
  const monthStart = useMemo(() => new Date(monthAnchor.getFullYear(), monthAnchor.getMonth(), 1), [monthAnchor]);
  const cells = useMemo(() => buildMonthGrid(monthStart, []), [monthStart]);
  const weekdays = useMemo(() => weekdayLabels(locale), [locale]);
  const viewingCurrentMonth = isSameMonth(monthStart, new Date());
  const todayKey = todayInputValue();

  return (
    <div className="daily-summary-calendar">
      <div className="daily-summary-calendar-toolbar">
        <button aria-label={t('previousMonth')} className="icon-button" onClick={onPreviousMonth} type="button">
          <ChevronLeft aria-hidden="true" />
        </button>
        <div className="daily-summary-calendar-heading">
          <strong>{formatMonthLabel(monthStart, locale)}</strong>
          {!viewingCurrentMonth ? (
            <button className="link-button" onClick={onTodayMonth} type="button">
              {t('thisMonth')}
            </button>
          ) : null}
        </div>
        <button aria-label={t('nextMonth')} className="icon-button" onClick={onNextMonth} type="button">
          <ChevronRight aria-hidden="true" />
        </button>
      </div>

      <div className="calendar-grid daily-summary-calendar-grid" role="grid" aria-label={t('dailySummaryCalendarLabel')}>
        <div className="calendar-weekdays" role="row">
          {weekdays.map((label) => (
            <span className="calendar-weekday" key={label} role="columnheader">
              {label}
            </span>
          ))}
        </div>
        <div className="calendar-days">
          {cells.map((cell) => {
            const index = indexByDate.get(cell.date);
            const statusClass = index?.status === 'approved' ? 'has-approved' : index ? 'has-draft' : 'is-empty';
            const isSelected = isSameLocalDay(cell.date, selectedDate);
            const isToday = isSameLocalDay(cell.date, todayKey);
            return (
              <button
                aria-label={`${cell.dayNumber} ${index?.status === 'approved' ? t('dailySummaryLegendApproved') : index ? t('dailySummaryLegendDraft') : t('dailySummaryLegendEmpty')}`}
                aria-pressed={isSelected}
                className={`calendar-day daily-summary-calendar-day ${statusClass}${cell.inMonth ? '' : ' outside-month'}${isSelected ? ' selected' : ''}${isToday ? ' today' : ''}`}
                key={cell.date}
                onClick={() => onSelectDay(cell.date)}
                type="button"
              >
                <span className="calendar-day-number">{cell.dayNumber}</span>
                <span aria-hidden="true" className="daily-summary-calendar-marker" />
              </button>
            );
          })}
        </div>
      </div>

      <ul className="daily-summary-calendar-legend">
        <li>
          <span className="daily-summary-calendar-marker has-approved" />
          {t('dailySummaryLegendApproved')}
        </li>
        <li>
          <span className="daily-summary-calendar-marker has-draft" />
          {t('dailySummaryLegendDraft')}
        </li>
        <li>
          <span className="daily-summary-calendar-marker is-empty" />
          {t('dailySummaryLegendEmpty')}
        </li>
      </ul>
    </div>
  );
}

export function DailySummaryPanel({
  clients,
  locale,
  projects,
  t,
}: {
  clients: Client[];
  locale: Locale;
  projects: Project[];
  t: Translator;
}) {
  const toast = useToast();
  const queryClient = useQueryClient();
  const [form, setForm] = useState<DailySummaryFormState>(() => defaultDailySummaryForm());
  const [monthAnchor, setMonthAnchor] = useState<Date>(() => monthAnchorFromDate(defaultDailySummaryForm().date));
  const [draftText, setDraftText] = useState('');
  const [copyState, setCopyState] = useState<'idle' | 'copied' | 'failed'>('idle');
  const [workflowProgress, setWorkflowProgress] = useState<WorkflowProgress>(null);

  const params = useMemo(() => formToParams(form), [form]);
  const activeClients = useMemo(() => clients.filter((client) => !client.archivedAt), [clients]);
  const scopedProjects = useMemo(() => {
    let list = projects.filter((project) => !project.archivedAt);
    if (form.clientId) {
      list = list.filter((project) => project.clientId === form.clientId);
    }
    return list;
  }, [form.clientId, projects]);

  const monthStart = useMemo(() => new Date(monthAnchor.getFullYear(), monthAnchor.getMonth(), 1), [monthAnchor]);
  const billingPeriodFrom = toDateKey(monthStart);
  const billingPeriodTo = toDateKey(endOfMonth(monthStart));
  const billingPeriodLabel = formatMonthLabel(monthStart, locale);
  const calendarCells = useMemo(() => buildMonthGrid(monthStart, []), [monthStart]);
  const calendarFrom = calendarCells[0]?.date ?? '';
  const calendarTo = calendarCells[calendarCells.length - 1]?.date ?? '';

  const indexQuery = useQuery({
    queryKey: ['daily-summary-index', calendarFrom, calendarTo, 'all-scopes'],
    queryFn: () => fetchDailySummaryIndex(calendarFrom, calendarTo, { allScopes: true }),
    enabled: Boolean(calendarFrom && calendarTo),
  });

  const indexByDate = useMemo(() => aggregateSummaryIndex(indexQuery.data ?? []), [indexQuery.data]);

  const usageQuery = useQuery({
    queryKey: ['daily-summary-ai-usage', billingPeriodFrom, billingPeriodTo],
    queryFn: () => fetchDailySummaryAIUsage(billingPeriodFrom, billingPeriodTo),
  });

  const scopedRuns = useMemo(
    () =>
      (usageQuery.data?.runs ?? []).filter(
        (run) => run.clientId === form.clientId && run.projectId === form.projectId,
      ),
    [form.clientId, form.projectId, usageQuery.data?.runs],
  );
  const lastScopedRun = scopedRuns[0] ?? null;

  const recordQuery = useQuery({
    queryKey: ['daily-summary-record', form.date, form.clientId, form.projectId],
    queryFn: () => fetchDailySummaryRecord(form.date, { clientId: form.clientId, projectId: form.projectId }),
    retry: false,
  });

  const record = recordQuery.data ?? null;
  const isApproved = record?.status === 'approved';
  const currentScope = scopeLabel(form, clients, projects, t);
  const savedText = record ? (record.status === 'approved' ? record.approvedText : record.draftText) : '';
  const hasUnsavedChanges = Boolean(record && !isApproved && draftText !== savedText);

  const generateSteps = useMemo<DailySummaryWorkflowStep[]>(
    () => [
      { id: 'entries', label: t('dailySummaryStepEntries') },
      { id: 'template', label: t('dailySummaryStepTemplate') },
    ],
    [t],
  );

  const enrichSteps = useMemo<DailySummaryWorkflowStep[]>(
    () => [
      { id: 'collect', label: t('dailySummaryStepCollect') },
      { id: 'context', label: t('dailySummaryStepContext') },
      { id: 'ai', label: t('dailySummaryStepAI') },
      { id: 'save', label: t('dailySummaryStepSave') },
    ],
    [t],
  );

  useEffect(() => {
    setMonthAnchor(monthAnchorFromDate(form.date));
  }, [form.date]);

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

  function invalidateSummaryQueries() {
    void queryClient.invalidateQueries({ queryKey: ['daily-summary-record', form.date, form.clientId, form.projectId] });
    void queryClient.invalidateQueries({ queryKey: ['daily-summary-index'] });
    void queryClient.invalidateQueries({ queryKey: ['daily-summary-ai-usage'] });
  }

  function requestEnrichConfirmation(): boolean {
    return window.confirm(buildDailySummaryEnrichConfirmMessage(usageQuery.data?.summary, t, locale));
  }

  function triggerEnrich() {
    if (!requestEnrichConfirmation()) {
      return;
    }
    enrichMutation.mutate();
  }

  const generateMutation = useMutation({
    mutationFn: async () => {
      setWorkflowProgress({ kind: 'generate', step: 'entries' });
      await new Promise((resolve) => window.setTimeout(resolve, 250));
      setWorkflowProgress({ kind: 'generate', step: 'template' });
      return generateDailySummaryRecord(form.date, params, form.note);
    },
    onSuccess: (saved) => {
      setDraftText(saved.draftText);
      invalidateSummaryQueries();
      toast.success(t('dailySummaryGenerated'));
    },
    onError: () => toast.error(t('dailySummaryLoadFailed')),
    onSettled: () => setWorkflowProgress(null),
  });

  const saveMutation = useMutation({
    mutationFn: () =>
      saveDailySummaryDraft(form.date, {
        draftText,
        manualNote: form.note,
        options: params,
      }),
    onSuccess: () => {
      invalidateSummaryQueries();
      toast.success(t('dailySummarySaved'));
    },
    onError: () => toast.error(t('dailySummarySaveFailed')),
  });

  const enrichMutation = useMutation({
    mutationFn: async () => {
      setWorkflowProgress({ kind: 'enrich', step: 'collect' });
      const context = await fetchDailySummaryEnrichContext(form.date, params, form.note);
      setWorkflowProgress({ kind: 'enrich', step: 'context' });
      await new Promise((resolve) => window.setTimeout(resolve, 200));
      setWorkflowProgress({ kind: 'enrich', step: 'ai' });
      const enriched = await enrichDailySummaryLocally({
        date: form.date,
        templateText: context.templateText,
        manualNote: form.note,
        feedback: form.feedback,
        currentDraft: draftText,
        locale: context.locale,
        authorEmail: context.authorEmail,
        entryFacts: context.entryFacts ?? [],
        projects: context.projects,
      });
      setWorkflowProgress({ kind: 'enrich', step: 'save' });
      return applyDailySummaryEnrichment(form.date, {
        text: enriched.text,
        manualNote: form.note,
        generationSource: enriched.source,
        modelId: enriched.modelId,
        aiUsage: enriched.usage,
        options: params,
      });
    },
    onSuccess: (saved) => {
      setDraftText(saved.draftText);
      setForm((current) => ({ ...current, feedback: '' }));
      invalidateSummaryQueries();
      toast.success(t('dailySummaryEnriched'));
    },
    onError: () => toast.error(t('dailySummaryEnrichFailed')),
    onSettled: () => setWorkflowProgress(null),
  });

  const approveMutation = useMutation({
    mutationFn: () => approveDailySummaryRecord(form.date, draftText, { clientId: form.clientId, projectId: form.projectId }),
    onSuccess: () => {
      invalidateSummaryQueries();
      toast.success(t('dailySummaryApprovedToast'));
    },
    onError: () => toast.error(t('dailySummaryApproveFailed')),
  });

  const reopenMutation = useMutation({
    mutationFn: () => reopenDailySummaryRecord(form.date, { clientId: form.clientId, projectId: form.projectId }),
    onSuccess: (saved) => {
      setDraftText(saved.draftText);
      invalidateSummaryQueries();
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

  const statusBadge = (() => {
    if (isApproved) {
      return {
        tone: 'approved' as const,
        label: t('dailySummaryStatusApproved'),
        detail: formatStatusTimestamp(record?.approvedAt ?? record?.updatedAt ?? '', locale),
      };
    }
    if (hasUnsavedChanges) {
      return {
        tone: 'unsaved' as const,
        label: t('dailySummaryStatusUnsaved'),
        detail: t('dailySummaryStatusUnsavedHint'),
      };
    }
    if (record) {
      return {
        tone: 'draft' as const,
        label: t('dailySummaryStatusDraftSaved'),
        detail: `${generationSourceLabel(record.generationSource, t)} · ${t('dailySummaryGenerationCount').replace('{count}', String(record.generationCount))}`,
      };
    }
    if (draftText.trim()) {
      return {
        tone: 'draft' as const,
        label: t('dailySummaryStatusGenerated'),
        detail: t('dailySummaryStatusGeneratedHint'),
      };
    }
    return {
      tone: 'empty' as const,
      label: t('dailySummaryStatusEmpty'),
      detail: t('dailySummaryStatusEmptyHint'),
    };
  })();

  const progressOverlay =
    workflowProgress?.kind === 'generate' ? (
      <DailySummaryProgressOverlay
        activeStepId={workflowProgress.step}
        steps={generateSteps}
        title={t('dailySummaryGeneratingTitle')}
      />
    ) : workflowProgress?.kind === 'enrich' ? (
      <DailySummaryProgressOverlay activeStepId={workflowProgress.step} steps={enrichSteps} title={t('dailySummaryEnrichingTitle')} />
    ) : null;

  return (
    <section className="clients-section report-section daily-summary-section" id="daily-summary" aria-labelledby="daily-summary-title">
      <header className="daily-summary-topbar">
        <div className="daily-summary-intro">
          <span className="section-kicker">
            <MessageSquareText aria-hidden="true" />
            {t('dailySummary')}
          </span>
          <h2 id="daily-summary-title">{t('dailySummaryTitle')}</h2>
          <p>{t('dailySummarySubtitle')}</p>
          <p className="daily-summary-scope-label">
            {t('dailySummaryScopeLabel').replace('{scope}', currentScope)}
          </p>
        </div>
        <DailySummaryAIUsageChip
          isLoading={usageQuery.isLoading}
          locale={locale}
          periodLabel={billingPeriodLabel}
          runs={usageQuery.data?.runs ?? []}
          summary={usageQuery.data?.summary}
          t={t}
        />
      </header>

      <div className="daily-summary-workbench">
        <aside className="daily-summary-sidebar">
          <div className="daily-summary-sidebar-card daily-summary-calendar-card">
            <DailySummaryMonthCalendar
              indexByDate={indexByDate}
              locale={locale}
              monthAnchor={monthAnchor}
              onNextMonth={() => setMonthAnchor((current) => addMonths(current, 1))}
              onPreviousMonth={() => setMonthAnchor((current) => addMonths(current, -1))}
              onSelectDay={(date) => setForm((current) => ({ ...current, date }))}
              onTodayMonth={() => {
                const today = todayInputValue();
                setMonthAnchor(monthAnchorFromDate(today));
                setForm((current) => ({ ...current, date: today }));
              }}
              selectedDate={form.date}
              t={t}
            />
          </div>

          <div className="daily-summary-sidebar-card daily-summary-options-card">
            <h3>{t('dailySummaryOptions')}</h3>
            <form className="report-form daily-summary-options-form" noValidate onSubmit={submitPreview}>
              <label className="form-field">
                {t('dailySummaryDate')}
                <input
                  disabled={busy}
                  onChange={(event) => setForm((current) => ({ ...current, date: event.target.value }))}
                  type="date"
                  value={form.date}
                />
              </label>
              <div className="daily-summary-options-grid">
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
              </div>
              <div className="daily-summary-toggle-grid">
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
              </div>
              <label className="form-field">
                {t('dailySummaryNote')}
                <textarea
                  disabled={isApproved}
                  onChange={(event) => setForm((current) => ({ ...current, note: event.target.value }))}
                  placeholder={t('dailySummaryNotePlaceholder')}
                  rows={2}
                  value={form.note}
                />
              </label>
              <label className="form-field">
                {t('dailySummaryFeedback')}
                <textarea
                  disabled={isApproved}
                  onChange={(event) => setForm((current) => ({ ...current, feedback: event.target.value }))}
                  placeholder={t('dailySummaryFeedbackPlaceholder')}
                  rows={2}
                  value={form.feedback}
                />
              </label>
              <div className="report-form-actions daily-summary-action-row">
                <button className="secondary-button" disabled={busy || isApproved} type="submit">
                  <RefreshCw aria-hidden="true" />
                  {t('dailySummaryGenerate')}
                </button>
                <button
                  className="secondary-button daily-summary-enrich-button"
                  disabled={busy || isApproved}
                  onClick={triggerEnrich}
                  type="button"
                >
                  <Sparkles aria-hidden="true" />
                  {t('dailySummaryEnrich')}
                </button>
              </div>
            </form>
          </div>
        </aside>

        <main className="daily-summary-editor-panel" aria-live="polite">
          <div className="daily-summary-editor-header">
            <div className="daily-summary-editor-heading">
              <h3>{t('dailySummaryPreview')}</h3>
              <DailySummaryStatusBadge detail={statusBadge.detail} label={statusBadge.label} tone={statusBadge.tone} />
              <DailySummaryLastRunBadge locale={locale} run={lastScopedRun} t={t} />
            </div>
          </div>

          <div className="daily-summary-preview-shell">
            {progressOverlay}
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
                  <textarea
                    aria-label={t('dailySummarySlackText')}
                    id="daily-summary-text"
                    onChange={(event) => {
                      setDraftText(event.target.value);
                      setCopyState('idle');
                    }}
                    readOnly={isApproved}
                    rows={18}
                    value={draftText}
                  />
                </label>
                <div className="report-form-actions daily-summary-editor-actions">
                  <button disabled={!draftText.trim() || busy} onClick={() => void handleCopy()} type="button">
                    <ClipboardCopy aria-hidden="true" />
                    {copyState === 'copied' ? t('dailySummaryCopied') : t('dailySummaryCopySlack')}
                  </button>
                  {!isApproved ? (
                    <>
                      <button
                        className="secondary-button"
                        disabled={!draftText.trim() || busy || !hasUnsavedChanges}
                        onClick={() => saveMutation.mutate()}
                        type="button"
                      >
                        {t('dailySummarySaveDraft')}
                      </button>
                      <button disabled={!draftText.trim() || busy || hasUnsavedChanges} onClick={() => approveMutation.mutate()} type="button">
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
        </main>
      </div>
    </section>
  );
}

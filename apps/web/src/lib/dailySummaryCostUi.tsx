import { ChevronDown, Coins, Sparkles } from 'lucide-react';
import type { DailySummaryAIRun, DailySummaryAIUsageSummary, Locale } from './api';
import type { Translator } from './timeEntryUi';

function formatTokens(value: number, locale: Locale): string {
  return new Intl.NumberFormat(locale === 'es' ? 'es-ES' : 'en-US', {
    notation: value >= 10_000 ? 'compact' : 'standard',
    maximumFractionDigits: 1,
  }).format(value);
}

function formatUsd(value: number, locale: Locale): string {
  return new Intl.NumberFormat(locale === 'es' ? 'es-ES' : 'en-US', {
    currency: 'USD',
    maximumFractionDigits: 4,
    minimumFractionDigits: 2,
    style: 'currency',
  }).format(value);
}

function formatRunDate(value: string, locale: Locale): string {
  const parsed = Date.parse(value);
  if (Number.isNaN(parsed)) {
    return value;
  }
  return new Date(parsed).toLocaleString(locale === 'es' ? 'es-ES' : 'en-US', {
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    month: 'short',
  });
}

export function buildDailySummaryEnrichConfirmMessage(
  summary: DailySummaryAIUsageSummary | undefined,
  t: Translator,
  locale: Locale,
): string {
  const periodTokens = formatTokens(summary?.totalTokens ?? 0, locale);
  const periodCost = formatUsd(summary?.estimatedCostUsd ?? 0, locale);
  const runCount = String(summary?.runCount ?? 0);
  return t('dailySummaryEnrichConfirm')
    .replace('{periodTokens}', periodTokens)
    .replace('{periodCost}', periodCost)
    .replace('{runCount}', runCount);
}

export function DailySummaryAIUsageChip({
  isLoading,
  locale,
  periodLabel,
  runs,
  summary,
  t,
}: {
  isLoading: boolean;
  locale: Locale;
  periodLabel: string;
  runs: DailySummaryAIRun[];
  summary: DailySummaryAIUsageSummary | undefined;
  t: Translator;
}) {
  const recentRuns = runs.slice(0, 5);

  return (
    <details className="daily-summary-ai-chip">
      <summary aria-label={t('dailySummaryCostTitle')}>
        <span className="daily-summary-ai-chip-icon" aria-hidden="true">
          <Coins />
        </span>
        <span className="daily-summary-ai-chip-copy">
          <strong>{t('dailySummaryCostTitle')}</strong>
          <span>
            {isLoading
              ? t('loading')
              : `${periodLabel} · ${formatTokens(summary?.totalTokens ?? 0, locale)} tokens · ${formatUsd(summary?.estimatedCostUsd ?? 0, locale)}`}
          </span>
        </span>
        <ChevronDown aria-hidden="true" className="daily-summary-ai-chip-chevron" />
      </summary>

      <div className="daily-summary-ai-chip-body">
        {summary ? (
          <div className="daily-summary-ai-chip-metrics">
            <div>
              <span>{t('dailySummaryCostPeriodTokens')}</span>
              <strong>{formatTokens(summary.totalTokens, locale)}</strong>
            </div>
            <div>
              <span>{t('dailySummaryCostPeriodEstimate')}</span>
              <strong>{formatUsd(summary.estimatedCostUsd, locale)}</strong>
            </div>
            <div>
              <span>{t('dailySummaryCostPeriodRuns')}</span>
              <strong>{summary.runCount}</strong>
            </div>
          </div>
        ) : null}

        {summary ? (
          <p className="daily-summary-ai-chip-rate">
            {t('dailySummaryCostRateHint').replace('{rate}', formatUsd(summary.costPerMillionUsd, locale))}
          </p>
        ) : null}

        {recentRuns.length > 0 ? (
          <div className="daily-summary-ai-chip-runs">
            <h4>
              <Sparkles aria-hidden="true" />
              {t('dailySummaryCostRecentRuns')}
            </h4>
            <ul>
              {recentRuns.map((run) => (
                <li key={run.id}>
                  <span>{run.summaryDate}</span>
                  <span>{formatTokens(run.totalTokens, locale)}</span>
                  <span>{formatUsd(run.estimatedCostUsd, locale)}</span>
                </li>
              ))}
            </ul>
          </div>
        ) : (
          <p className="daily-summary-ai-chip-empty">{t('dailySummaryCostEmpty')}</p>
        )}
      </div>
    </details>
  );
}

export function DailySummaryLastRunBadge({
  locale,
  run,
  t,
}: {
  locale: Locale;
  run: DailySummaryAIRun | null;
  t: Translator;
}) {
  if (!run) {
    return null;
  }
  return (
    <div className="daily-summary-last-run">
      <span className="daily-summary-last-run-label">{t('dailySummaryLastRunLabel')}</span>
      <span className="daily-summary-last-run-value">
        {formatTokens(run.totalTokens, locale)} tokens · {formatUsd(run.estimatedCostUsd, locale)}
      </span>
    </div>
  );
}

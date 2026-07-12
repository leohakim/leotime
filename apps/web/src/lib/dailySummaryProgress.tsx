import { Check, Circle, LoaderCircle } from 'lucide-react';

export type DailySummaryWorkflowStep = {
  id: string;
  label: string;
};

export function DailySummaryProgressOverlay({
  activeStepId,
  steps,
  title,
}: {
  activeStepId: string;
  steps: DailySummaryWorkflowStep[];
  title: string;
}) {
  const activeIndex = steps.findIndex((step) => step.id === activeStepId);

  return (
    <div aria-busy="true" aria-live="polite" className="daily-summary-progress-overlay" role="status">
      <div className="daily-summary-progress-card">
        <p className="daily-summary-progress-title">{title}</p>
        <ol className="daily-summary-progress-steps">
          {steps.map((step, index) => {
            const isDone = activeIndex > index;
            const isActive = step.id === activeStepId;
            return (
              <li
                className={`daily-summary-progress-step${isDone ? ' is-done' : ''}${isActive ? ' is-active' : ''}`}
                key={step.id}
              >
                <span aria-hidden="true" className="daily-summary-progress-icon">
                  {isDone ? <Check /> : isActive ? <LoaderCircle className="spin-icon" /> : <Circle />}
                </span>
                <span>{step.label}</span>
              </li>
            );
          })}
        </ol>
      </div>
    </div>
  );
}

export function DailySummaryStatusBadge({
  detail,
  label,
  tone,
}: {
  detail?: string;
  label: string;
  tone: 'approved' | 'draft' | 'empty' | 'unsaved';
}) {
  return (
    <div className={`daily-summary-status-badge tone-${tone}`}>
      <span className="daily-summary-status-badge-label">{label}</span>
      {detail ? <span className="daily-summary-status-badge-detail">{detail}</span> : null}
    </div>
  );
}

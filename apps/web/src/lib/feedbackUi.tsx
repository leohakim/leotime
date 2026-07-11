import { CircleAlert } from 'lucide-react';
import type { ReactNode } from 'react';

export function SurfaceLoading({ label }: { label: string }) {
  return (
    <div aria-live="polite" className="surface-feedback surface-feedback-loading">
      <span className="sync-pill">{label}</span>
    </div>
  );
}

export function SurfaceError({
  message,
  onRetry,
  retryLabel,
}: {
  message: string;
  onRetry?: () => void;
  retryLabel?: string;
}) {
  return (
    <div className="surface-feedback surface-feedback-error" role="alert">
      <CircleAlert aria-hidden="true" />
      <p>{message}</p>
      {onRetry && retryLabel ? (
        <button type="button" onClick={onRetry}>
          {retryLabel}
        </button>
      ) : null}
    </div>
  );
}

export function SurfaceEmpty({ children }: { children: ReactNode }) {
  return <div className="panel-empty-state">{children}</div>;
}

import type { ReactNode } from 'react';

export function SettingsPanel({
  id,
  kicker,
  title,
  subtitle,
  meta,
  children,
}: {
  id: string;
  kicker?: ReactNode;
  title: string;
  subtitle?: string;
  meta?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className="settings-panel" id={id} aria-labelledby={`${id}-title`}>
      <header className="settings-panel-header">
        <div className="settings-panel-heading">
          {kicker ? <span className="section-kicker">{kicker}</span> : null}
          <h2 id={`${id}-title`}>{title}</h2>
          {subtitle ? <p>{subtitle}</p> : null}
        </div>
        {meta ? <div className="settings-panel-meta">{meta}</div> : null}
      </header>
      <div className="settings-panel-body">{children}</div>
    </section>
  );
}

export function SettingsCard({ title, children, actions }: { title?: string; children: ReactNode; actions?: ReactNode }) {
  return (
    <div className="settings-card">
      {title ? <h3 className="settings-card-title">{title}</h3> : null}
      <div className="settings-card-body">{children}</div>
      {actions ? <div className="settings-card-actions">{actions}</div> : null}
    </div>
  );
}

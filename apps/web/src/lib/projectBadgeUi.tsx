import type { CSSProperties } from 'react';
import type { Project } from './api';

export const NO_PROJECT_COLOR = '#64748b';

function resolveProjectColor(color?: string): string {
  return color?.trim() || NO_PROJECT_COLOR;
}

function projectColorStyle(color?: string): CSSProperties {
  return { '--project-color': resolveProjectColor(color) } as CSSProperties;
}

export function ProjectBadge({
  color,
  compact = false,
  emptyLabel,
  name,
}: {
  color?: string;
  compact?: boolean;
  emptyLabel: string;
  name?: string;
}) {
  const hasProject = Boolean(name?.trim());

  return (
    <span
      className={`project-badge${hasProject ? '' : ' project-badge-empty'}${compact ? ' project-badge-compact' : ''}`}
      style={hasProject ? projectColorStyle(color) : undefined}
    >
      <span aria-hidden="true" className="project-badge-dot" />
      <span className="project-badge-label">{hasProject ? name : emptyLabel}</span>
    </span>
  );
}

export function ProjectBadgeSelect({
  ariaLabel,
  color,
  emptyLabel,
  onChange,
  projects,
  value,
}: {
  ariaLabel: string;
  color?: string;
  emptyLabel: string;
  onChange: (projectId: string) => void;
  projects: Project[];
  value: string;
}) {
  const selected = projects.find((project) => project.id === value);
  const hasProject = Boolean(selected);
  const resolvedColor = hasProject ? resolveProjectColor(selected?.color || color) : undefined;

  return (
    <label
      className={`project-badge project-badge-select${hasProject ? '' : ' project-badge-empty'}`}
      style={hasProject ? projectColorStyle(resolvedColor) : undefined}
    >
      <span aria-hidden="true" className="project-badge-dot" />
      <span className="project-badge-label">{selected?.name ?? emptyLabel}</span>
      <select
        aria-label={ariaLabel}
        className="project-badge-select-input"
        onChange={(event) => onChange(event.target.value)}
        value={value}
      >
        <option value="">{emptyLabel}</option>
        {projects.map((project) => (
          <option key={project.id} value={project.id}>
            {project.name}
          </option>
        ))}
      </select>
    </label>
  );
}

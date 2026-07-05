import { render, screen } from '@testing-library/react';
import { describe, expect, test } from 'vitest';
import { NO_PROJECT_COLOR, ProjectBadge } from './projectBadgeUi';

describe('ProjectBadge', () => {
  test('renders project name with color dot', () => {
    const { container } = render(<ProjectBadge color="#2563eb" emptyLabel="Sin proyecto" name="Portal Web" />);
    expect(screen.getByText('Portal Web')).toBeInTheDocument();
    const badge = container.querySelector('.project-badge');
    expect(badge).toHaveStyle({ '--project-color': '#2563eb' });
  });

  test('renders empty badge when project is missing', () => {
    render(<ProjectBadge color="#2563eb" emptyLabel="Sin proyecto" name="" />);
    expect(screen.getByText('Sin proyecto')).toBeInTheDocument();
  });

  test('falls back to default project color', () => {
    const { container } = render(<ProjectBadge emptyLabel="Sin proyecto" name="Portal Web" />);
    expect(container.querySelector('.project-badge')).toHaveStyle({ '--project-color': NO_PROJECT_COLOR });
  });
});

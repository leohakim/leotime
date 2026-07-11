import '@testing-library/jest-dom/vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { createElement } from 'react';
import { describe, expect, test, vi, afterEach, beforeEach } from 'vitest';
import type { Client, Project, Task, TimeEntry } from './api';
import {
  applyManualEntryFieldUpdate,
  formatTimesheetEntryLabel,
  formatTimeRange,
  MANUAL_ENTRY_DESCRIPTION_ID,
  MANUAL_ENTRY_EDITOR_ID,
  scrollToManualEntryForm,
  tasksForManualEntryForm,
  TIMESHEET_COMPACT_MEDIA,
  TimesheetEntryRow,
} from './timeEntryUi';
import { translate } from './i18n';

const t = (key: Parameters<typeof translate>[1]) => translate('es', key);

function stubCompactTimesheet() {
  vi.stubGlobal(
    'matchMedia',
    vi.fn().mockImplementation((query: string) => ({
      matches: query === TIMESHEET_COMPACT_MEDIA,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  );
}

function renderTimesheetRow(entry: TimeEntry) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return render(
    createElement(
      QueryClientProvider,
      { client: queryClient },
      createElement(TimesheetEntryRow, { entry, locale: 'es', projects, tasks, t }),
    ),
  );
}

const clients: Client[] = [
  {
    id: 'cli_1',
    name: 'ACME',
    email: '',
    taxId: '',
    billingAddress: '',
    defaultCurrency: 'EUR',
    defaultHourlyRateMinor: 3500,
    archivedAt: '',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
  },
];

const projects: Project[] = [
  {
    id: 'prj_1',
    clientId: 'cli_1',
    clientName: 'ACME',
    name: 'Portal Web',
    color: '#2563eb',
    defaultHourlyRateMinor: null,
    archivedAt: '',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
  },
  {
    id: 'prj_2',
    clientId: 'cli_1',
    clientName: 'ACME',
    name: 'ENACT',
    color: '#f97316',
    defaultHourlyRateMinor: null,
    archivedAt: '',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
  },
];

const tasks: Task[] = [
  {
    id: 'tsk_1',
    projectId: 'prj_1',
    projectName: 'Portal Web',
    projectColor: '#2563eb',
    name: 'Refactor API',
    billable: true,
    archivedAt: '',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
  },
  {
    id: 'tsk_2',
    projectId: 'prj_2',
    projectName: 'ENACT',
    projectColor: '#f97316',
    name: 'Deep Work',
    billable: true,
    archivedAt: '',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
  },
];

const baseForm = {
  clientId: '',
  projectId: 'prj_2',
  taskId: '',
  tagIds: [] as string[],
  description: 'Old entry',
  startedAt: '2026-07-01T08:00',
  endedAt: '2026-07-01T09:00',
  billable: true,
};

afterEach(() => {
  window.location.hash = '';
  document.body.innerHTML = '';
  vi.unstubAllGlobals();
});

describe('TimesheetEntryRow compact mode', () => {
  beforeEach(() => {
    stubCompactTimesheet();
  });

  test('expands summary rows for inline editing', async () => {
    const entry: TimeEntry = {
      id: 'ten_1',
      clientId: 'cli_1',
      clientName: 'ACME',
      projectId: 'prj_1',
      projectName: 'Portal Web',
      projectColor: '#2563eb',
      taskId: '',
      taskName: '',
      description: 'Deep work',
      startedAt: '2026-07-11T17:40:00.000Z',
      endedAt: '2026-07-11T18:40:00.000Z',
      durationSeconds: 3600,
      billable: true,
      overlapWarning: false,
      source: 'manual',
      tags: [],
      createdAt: '2026-01-01T00:00:00Z',
      updatedAt: '2026-01-01T00:00:00Z',
    };

    renderTimesheetRow(entry);

    expect(screen.getByText('Deep work')).toBeInTheDocument();
    expect(screen.queryByDisplayValue('Deep work')).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Editar' }));

    await waitFor(() => {
      expect(screen.getByDisplayValue('Deep work')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Listo' }));

    await waitFor(() => {
      expect(screen.queryByDisplayValue('Deep work')).not.toBeInTheDocument();
      expect(screen.getByText('Deep work')).toBeInTheDocument();
    });
  });
});

describe('formatTimesheetEntryLabel', () => {
  test('falls back when description is empty', () => {
    expect(formatTimesheetEntryLabel('  ', 'Sin descripcion')).toBe('Sin descripcion');
    expect(formatTimesheetEntryLabel(' Deep work ', 'Sin descripcion')).toBe('Deep work');
  });
});

describe('formatTimeRange', () => {
  test('formats a readable start-end range', () => {
    const range = formatTimeRange('2026-07-11T09:15:00.000Z', '2026-07-11T10:30:00.000Z', 'es');
    expect(range).toMatch(/\d{1,2}:\d{2}/);
    expect(range).toContain('-');
  });
});

describe('scrollToManualEntryForm', () => {
  test('navigates to manual entry and focuses the editor', async () => {
    const editor = document.createElement('form');
    editor.id = MANUAL_ENTRY_EDITOR_ID;
    const description = document.createElement('input');
    description.id = MANUAL_ENTRY_DESCRIPTION_ID;
    editor.append(description);
    document.body.append(editor);

    const scrollIntoView = vi.fn();
    editor.scrollIntoView = scrollIntoView;
    const focus = vi.fn();
    description.focus = focus;

    window.location.hash = '#manual-time-entry';
    scrollToManualEntryForm();

    await new Promise<void>((resolve) => {
      window.requestAnimationFrame(() => resolve());
    });

    expect(scrollIntoView).toHaveBeenCalledWith({ behavior: 'smooth', block: 'start' });
    expect(focus).toHaveBeenCalledWith({ preventScroll: true });
  });
});

describe('tasksForManualEntryForm', () => {
  test('keeps the selected task visible even when it belongs to another project filter', () => {
    const options = tasksForManualEntryForm('prj_2', 'tsk_1', tasks);
    expect(options.map((task) => task.id)).toEqual(['tsk_2', 'tsk_1']);
  });
});

describe('applyManualEntryFieldUpdate', () => {
  test('assigns project and client when a task is selected', () => {
    const next = applyManualEntryFieldUpdate(baseForm, 'taskId', 'tsk_1', projects, tasks, clients);
    expect(next.taskId).toBe('tsk_1');
    expect(next.projectId).toBe('prj_1');
    expect(next.clientId).toBe('cli_1');
  });

  test('assigns client when project changes', () => {
    const next = applyManualEntryFieldUpdate(baseForm, 'projectId', 'prj_1', projects, tasks, clients);
    expect(next.projectId).toBe('prj_1');
    expect(next.clientId).toBe('cli_1');
  });

  test('clears task when project changes to an incompatible project', () => {
    const withTask = applyManualEntryFieldUpdate(baseForm, 'taskId', 'tsk_1', projects, tasks, clients);
    const next = applyManualEntryFieldUpdate(withTask, 'projectId', 'prj_2', projects, tasks, clients);
    expect(next.projectId).toBe('prj_2');
    expect(next.taskId).toBe('');
  });
});

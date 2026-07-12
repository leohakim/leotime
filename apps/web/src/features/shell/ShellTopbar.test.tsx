import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { ExperiencePreset } from '../../lib/experience';
import { OfflineProvider } from '../../lib/offline/offlineContext';
import { ToastProvider } from '../../lib/toast';
import { ShellTopbar } from './ShellTopbar';

const preset: ExperiencePreset = 'workbench-pro';
const t = (key: string) =>
  (
    ({
      experience: 'Experiencia',
      logout: 'Salir',
    }) as Record<string, string>
  )[key] ?? key;

describe('ShellTopbar', () => {
  it('labels toolbar controls for assistive tech', () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <ToastProvider>
          <OfflineProvider>
            <ShellTopbar
              layoutMode="solid"
              navigationMode="sidebar"
              onApplyExperiencePreset={() => undefined}
              onLogout={() => undefined}
              pageTitle="Timesheet"
              preset={preset}
              setLayoutMode={() => undefined}
              setNavigationMode={() => undefined}
              setThemeMode={() => undefined}
              themeMode="solid"
              t={t}
            />
          </OfflineProvider>
        </ToastProvider>
      </QueryClientProvider>,
    );

    expect(screen.getByRole('button', { name: 'Experiencia' })).toHaveAttribute('aria-expanded', 'false');
    expect(screen.getByRole('button', { name: 'Salir' })).toBeInTheDocument();
  });
});

import { cleanup, fireEvent, render, screen, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { SettingsSectionNav, scrollToSettingsSection } from './settingsSectionNavUi';

const t = (key: string) =>
  (
    ({
      profileAccountSection: 'Cuenta',
      settings: 'Ajustes',
      profileEmailNotificationsSection: 'Notificaciones',
      profilePasswordSection: 'Seguridad',
      backupHeading: 'Copias',
      settingsSectionNavLabel: 'Secciones',
    }) as Record<string, string>
  )[key] ?? key;

describe('settingsSectionNavUi', () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });
  it('renders section jump controls', () => {
    render(<SettingsSectionNav t={t} />);

    expect(screen.getByRole('navigation', { name: 'Secciones' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Cuenta' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Copias' })).toBeInTheDocument();
  });

  it('scrolls to the requested section', () => {
    const scrollIntoView = vi.fn();
    const target = document.createElement('div');
    target.scrollIntoView = scrollIntoView;
    vi.spyOn(document, 'getElementById').mockReturnValue(target);

    scrollToSettingsSection('backups');
    expect(scrollIntoView).toHaveBeenCalledWith({ behavior: 'smooth', block: 'start' });
  });

  it('jumps from nav buttons without changing the hash route', () => {
    const scrollIntoView = vi.fn();
    const target = document.createElement('div');
    target.scrollIntoView = scrollIntoView;
    vi.spyOn(document, 'getElementById').mockReturnValue(target);
    window.location.hash = '#profile';

    render(<SettingsSectionNav t={t} />);
    const nav = screen.getByRole('navigation', { name: 'Secciones' });
    fireEvent.click(within(nav).getByRole('button', { name: 'Seguridad' }));

    expect(window.location.hash).toBe('#profile');
    expect(scrollIntoView).toHaveBeenCalled();
  });
});

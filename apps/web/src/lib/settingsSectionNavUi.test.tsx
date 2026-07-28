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
      aiSettingsHeading: 'IA',
      backupHeading: 'Copias',
      vcsHeading: 'Integraciones VCS',
      settingsSectionNavLabel: 'Secciones',
    }) as Record<string, string>
  )[key] ?? key;

describe('settingsSectionNavUi', () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it('renders section controls with the active section marked', () => {
    render(<SettingsSectionNav activeSection="settings" onSelect={() => undefined} t={t} />);

    expect(screen.getByRole('navigation', { name: 'Secciones' })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: 'Cuenta' })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: 'IA' })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: 'Copias' })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: 'Integraciones VCS' })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: 'Ajustes' })).toHaveAttribute('aria-selected', 'true');
  });

  it('notifies when a section is selected without changing the hash route', () => {
    const onSelect = vi.fn();
    window.location.hash = '#profile';

    render(<SettingsSectionNav activeSection="profile-section-account" onSelect={onSelect} t={t} />);
    const nav = screen.getByRole('navigation', { name: 'Secciones' });
    fireEvent.click(within(nav).getByRole('tab', { name: 'Seguridad' }));

    expect(window.location.hash).toBe('#profile');
    expect(onSelect).toHaveBeenCalledWith('profile-section-password');
  });

  it('keeps scroll helper for legacy deep links', () => {
    const scrollIntoView = vi.fn();
    const target = document.createElement('div');
    target.scrollIntoView = scrollIntoView;
    vi.spyOn(document, 'getElementById').mockReturnValue(target);

    scrollToSettingsSection('backups');
    expect(scrollIntoView).toHaveBeenCalledWith({ behavior: 'smooth', block: 'start' });
  });
});

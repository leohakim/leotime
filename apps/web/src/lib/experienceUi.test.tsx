import { cleanup, fireEvent, render, screen, within } from '@testing-library/react';
import { afterEach, describe, expect, test, vi } from 'vitest';
import { translate } from './i18n';
import { ExperienceSwitcher } from './experienceUi';

const t = (key: Parameters<typeof translate>[1]) => translate('es', key);

afterEach(cleanup);

describe('ExperienceSwitcher', () => {
  test('applies a named preset when the select changes', () => {
    const onApplyPreset = vi.fn();

    render(
      <ExperienceSwitcher
        layoutMode="solid"
        navigationMode="sidebar"
        onApplyPreset={onApplyPreset}
        preset="workbench-pro"
        setLayoutMode={vi.fn()}
        setNavigationMode={vi.fn()}
        setThemeMode={vi.fn()}
        themeMode="solid"
        t={t}
      />,
    );

    fireEvent.change(screen.getByLabelText('Experiencia sugerida'), { target: { value: 'focus-dark' } });

    expect(onApplyPreset).toHaveBeenCalledWith('focus-dark');
  });

  test('shows the custom preset label when active', () => {
    render(
      <ExperienceSwitcher
        layoutMode="light"
        navigationMode="bottom-tabs"
        onApplyPreset={vi.fn()}
        preset="custom"
        setLayoutMode={vi.fn()}
        setNavigationMode={vi.fn()}
        setThemeMode={vi.fn()}
        themeMode="light"
        t={t}
      />,
    );

    expect(screen.getByRole('option', { name: 'Personalizado' })).toBeInTheDocument();
  });

  test('changes navigation mode from the nav control', () => {
    const setNavigationMode = vi.fn();

    render(
      <ExperienceSwitcher
        layoutMode="solid"
        navigationMode="sidebar"
        onApplyPreset={vi.fn()}
        preset="workbench-pro"
        setLayoutMode={vi.fn()}
        setNavigationMode={setNavigationMode}
        setThemeMode={vi.fn()}
        themeMode="solid"
        t={t}
      />,
    );

    const navGroup = screen.getByRole('group', { name: 'Navegacion' });
    fireEvent.click(within(navGroup).getByRole('button', { name: 'Tabs inferiores' }));

    expect(setNavigationMode).toHaveBeenCalledWith('bottom-tabs');
  });
});

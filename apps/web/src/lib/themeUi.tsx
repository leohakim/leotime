import { Contrast, Moon, Palette, Sun } from 'lucide-react';
import { useEffect } from 'react';
import type { ThemeMode } from './api';
import { applyExperienceAttributes, type ExperienceState } from './experience';
import type { MessageKey } from './i18n';

export type Translator = (key: MessageKey) => string;

export function useExperienceEffect(state: ExperienceState) {
  useEffect(() => {
    applyExperienceAttributes(state);
  }, [state]);
}

export function ThemeSwitcher({
  setThemeMode,
  t,
  themeMode,
}: {
  setThemeMode: (themeMode: ThemeMode) => void;
  t: Translator;
  themeMode: ThemeMode;
}) {
  const options: Array<{ value: ThemeMode; label: string; icon: typeof Palette }> = [
    { value: 'solid', label: t('themeSolid'), icon: Palette },
    { value: 'light', label: t('themeLight'), icon: Sun },
    { value: 'dark', label: t('themeDark'), icon: Moon },
    { value: 'minimal', label: t('themeMinimal'), icon: Contrast },
  ];

  return (
    <div className="segmented-control theme-switcher" aria-label={t('theme')}>
      {options.map((option) => {
        const Icon = option.icon;
        return (
          <button
            aria-label={option.label}
            className={themeMode === option.value ? 'selected' : ''}
            key={option.value}
            onClick={() => setThemeMode(option.value)}
            title={option.label}
            type="button"
          >
            <Icon aria-hidden="true" />
          </button>
        );
      })}
    </div>
  );
}

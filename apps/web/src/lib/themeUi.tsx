import { Contrast, Moon, Palette, Sun } from 'lucide-react';
import { useEffect } from 'react';
import type { ThemeMode } from './api';
import type { MessageKey } from './i18n';

export type Translator = (key: MessageKey) => string;

const THEME_META_COLORS: Record<ThemeMode, string> = {
  solid: '#0c0d10',
  light: '#eef0f4',
  dark: '#050608',
  minimal: '#101114',
};

export function applyTheme(themeMode: ThemeMode) {
  document.documentElement.dataset.theme = themeMode;
  document.querySelector('meta[name="theme-color"]')?.setAttribute('content', THEME_META_COLORS[themeMode]);
}

if (typeof window !== 'undefined') {
  const storedTheme = window.localStorage.getItem('leotime.theme');
  const initialTheme =
    storedTheme === 'solid' || storedTheme === 'light' || storedTheme === 'dark' || storedTheme === 'minimal'
      ? storedTheme
      : 'solid';
  applyTheme(initialTheme);
}

export function useThemeEffect(themeMode: ThemeMode) {
  useEffect(() => {
    applyTheme(themeMode);
  }, [themeMode]);
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

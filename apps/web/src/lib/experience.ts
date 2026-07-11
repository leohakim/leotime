import type { LayoutMode, ThemeMode } from './api';

export type NavigationMode = 'sidebar';
export type ExperiencePreset = 'workbench-pro' | 'custom';

export type ExperienceState = {
  themeMode: ThemeMode;
  layoutMode: LayoutMode;
  navigationMode: NavigationMode;
  preset: ExperiencePreset;
};

export const DEFAULT_NAVIGATION_MODE: NavigationMode = 'sidebar';
export const DEFAULT_EXPERIENCE_PRESET: ExperiencePreset = 'workbench-pro';

const THEME_META_COLORS: Record<ThemeMode, string> = {
  solid: '#0c0d10',
  light: '#eef0f4',
  dark: '#050608',
  minimal: '#101114',
};

export function inferExperiencePreset({ themeMode, layoutMode, navigationMode }: Omit<ExperienceState, 'preset'>): ExperiencePreset {
  return themeMode === 'solid' && layoutMode === 'solid' && navigationMode === 'sidebar'
    ? DEFAULT_EXPERIENCE_PRESET
    : 'custom';
}

export function readNavigationMode(): NavigationMode {
  return window.localStorage.getItem('leotime.nav') === 'sidebar' ? 'sidebar' : DEFAULT_NAVIGATION_MODE;
}

export function readExperiencePreset(): ExperiencePreset {
  const value = window.localStorage.getItem('leotime.preset');
  return value === 'workbench-pro' || value === 'custom' ? value : 'custom';
}

export function applyExperienceMetaColor(themeMode: ThemeMode) {
  document.querySelector('meta[name="theme-color"]')?.setAttribute('content', THEME_META_COLORS[themeMode]);
}

export function applyExperienceAttributes(state: ExperienceState) {
  const root = document.documentElement;
  root.dataset.theme = state.themeMode;
  root.dataset.layout = state.layoutMode;
  root.dataset.nav = state.navigationMode;
  root.dataset.preset = state.preset;
  applyExperienceMetaColor(state.themeMode);
}

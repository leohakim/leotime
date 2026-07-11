import type { LayoutMode, ThemeMode } from './api';

export type NavigationMode = 'sidebar' | 'sidebar-compact' | 'bottom-tabs';
export type NamedExperiencePreset =
  | 'workbench-pro'
  | 'calm-light'
  | 'focus-dark'
  | 'compact-power'
  | 'mobile-flow'
  | 'solidtime-exact';
export type ExperiencePreset = NamedExperiencePreset | 'custom';

export type ExperienceDimensions = {
  themeMode: ThemeMode;
  layoutMode: LayoutMode;
  navigationMode: NavigationMode;
};

export type ExperienceState = ExperienceDimensions & {
  preset: ExperiencePreset;
};

export const DEFAULT_NAVIGATION_MODE: NavigationMode = 'sidebar';
export const DEFAULT_EXPERIENCE_PRESET: ExperiencePreset = 'workbench-pro';

export const NAVIGATION_MODES: NavigationMode[] = ['sidebar', 'sidebar-compact', 'bottom-tabs'];

export const NAMED_EXPERIENCE_PRESETS: NamedExperiencePreset[] = [
  'workbench-pro',
  'calm-light',
  'focus-dark',
  'compact-power',
  'mobile-flow',
  'solidtime-exact',
];

export const EXPERIENCE_PRESET_DEFINITIONS: Record<NamedExperiencePreset, ExperienceDimensions> = {
  'workbench-pro': { themeMode: 'solid', layoutMode: 'solid', navigationMode: 'sidebar' },
  'calm-light': { themeMode: 'light', layoutMode: 'minimal', navigationMode: 'sidebar' },
  'focus-dark': { themeMode: 'dark', layoutMode: 'solid', navigationMode: 'sidebar' },
  'compact-power': { themeMode: 'dark', layoutMode: 'compact', navigationMode: 'sidebar-compact' },
  'mobile-flow': { themeMode: 'light', layoutMode: 'compact', navigationMode: 'bottom-tabs' },
  'solidtime-exact': { themeMode: 'solid', layoutMode: 'solid', navigationMode: 'sidebar' },
};

const PRESET_INFERENCE_ORDER: NamedExperiencePreset[] = [
  'workbench-pro',
  'calm-light',
  'focus-dark',
  'compact-power',
  'mobile-flow',
  'solidtime-exact',
];

const THEME_META_COLORS: Record<ThemeMode, string> = {
  solid: '#0c0d10',
  light: '#eef0f4',
  dark: '#050608',
  minimal: '#101114',
};

function isNavigationMode(value: string | null): value is NavigationMode {
  return value !== null && NAVIGATION_MODES.includes(value as NavigationMode);
}

function isNamedExperiencePreset(value: string | null): value is NamedExperiencePreset {
  return value !== null && NAMED_EXPERIENCE_PRESETS.includes(value as NamedExperiencePreset);
}

export function getExperiencePresetDimensions(preset: NamedExperiencePreset): ExperienceDimensions {
  return EXPERIENCE_PRESET_DEFINITIONS[preset];
}

export function inferExperiencePreset(dimensions: ExperienceDimensions): ExperiencePreset {
  for (const preset of PRESET_INFERENCE_ORDER) {
    const definition = EXPERIENCE_PRESET_DEFINITIONS[preset];
    if (
      definition.themeMode === dimensions.themeMode &&
      definition.layoutMode === dimensions.layoutMode &&
      definition.navigationMode === dimensions.navigationMode
    ) {
      return preset;
    }
  }
  return 'custom';
}

export function readNavigationMode(): NavigationMode {
  const value = window.localStorage.getItem('leotime.nav');
  return isNavigationMode(value) ? value : DEFAULT_NAVIGATION_MODE;
}

export function readExperiencePreset(): ExperiencePreset {
  const value = window.localStorage.getItem('leotime.preset');
  if (value === 'custom') {
    return 'custom';
  }
  return isNamedExperiencePreset(value) ? value : 'custom';
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

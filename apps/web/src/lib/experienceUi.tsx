import { Columns3, Minimize2, PanelLeft, PanelLeftClose, Smartphone } from 'lucide-react';
import type { LayoutMode, ThemeMode } from './api';
import {
  NAMED_EXPERIENCE_PRESETS,
  type ExperiencePreset,
  type NamedExperiencePreset,
  type NavigationMode,
} from './experience';
import type { MessageKey } from './i18n';
import { ThemeSwitcher, type Translator } from './themeUi';

const PRESET_MESSAGE_KEYS: Record<NamedExperiencePreset, MessageKey> = {
  'workbench-pro': 'presetWorkbenchPro',
  'calm-light': 'presetCalmLight',
  'focus-dark': 'presetFocusDark',
  'compact-power': 'presetCompactPower',
  'mobile-flow': 'presetMobileFlow',
  'solidtime-exact': 'presetSolidtimeExact',
};

const NAV_MESSAGE_KEYS: Record<NavigationMode, MessageKey> = {
  sidebar: 'navSidebar',
  'sidebar-compact': 'navSidebarCompact',
  'bottom-tabs': 'navBottomTabs',
};

const LAYOUT_OPTIONS: Array<{ value: LayoutMode; labelKey: MessageKey; icon: typeof PanelLeft }> = [
  { value: 'solid', labelKey: 'solid', icon: PanelLeft },
  { value: 'minimal', labelKey: 'minimal', icon: Minimize2 },
  { value: 'compact', labelKey: 'compact', icon: Columns3 },
];

const NAV_OPTIONS: Array<{ value: NavigationMode; labelKey: MessageKey; icon: typeof PanelLeft }> = [
  { value: 'sidebar', labelKey: 'navSidebar', icon: PanelLeft },
  { value: 'sidebar-compact', labelKey: 'navSidebarCompact', icon: PanelLeftClose },
  { value: 'bottom-tabs', labelKey: 'navBottomTabs', icon: Smartphone },
];

export function ExperienceSwitcher({
  layoutMode,
  navigationMode,
  onApplyPreset,
  preset,
  setLayoutMode,
  setNavigationMode,
  setThemeMode,
  themeMode,
  t,
  variant = 'toolbar',
}: {
  layoutMode: LayoutMode;
  navigationMode: NavigationMode;
  onApplyPreset: (preset: NamedExperiencePreset) => void;
  preset: ExperiencePreset;
  setLayoutMode: (layoutMode: LayoutMode) => void;
  setNavigationMode: (navigationMode: NavigationMode) => void;
  setThemeMode: (themeMode: ThemeMode) => void;
  themeMode: ThemeMode;
  t: Translator;
  variant?: 'toolbar' | 'settings';
}) {
  return (
    <div className={`experience-switcher experience-switcher-${variant}`} aria-label={t('experience')}>
      <label className="experience-field">
        <span>{t('experiencePreset')}</span>
        <select
          aria-label={t('experiencePreset')}
          onChange={(event) => {
            const value = event.target.value;
            if (value !== 'custom' && NAMED_EXPERIENCE_PRESETS.includes(value as NamedExperiencePreset)) {
              onApplyPreset(value as NamedExperiencePreset);
            }
          }}
          value={preset}
        >
          {preset === 'custom' ? <option value="custom">{t('experienceCustom')}</option> : null}
          {NAMED_EXPERIENCE_PRESETS.map((option) => (
            <option key={option} value={option}>
              {t(PRESET_MESSAGE_KEYS[option])}
            </option>
          ))}
        </select>
      </label>

      <ThemeSwitcher setThemeMode={setThemeMode} themeMode={themeMode} t={t} />

      <LayoutSwitcher layoutMode={layoutMode} setLayoutMode={setLayoutMode} t={t} />

      <NavigationSwitcher navigationMode={navigationMode} setNavigationMode={setNavigationMode} t={t} />
    </div>
  );
}

function LayoutSwitcher({
  layoutMode,
  setLayoutMode,
  t,
}: {
  layoutMode: LayoutMode;
  setLayoutMode: (layoutMode: LayoutMode) => void;
  t: Translator;
}) {
  return (
    <div className="segmented-control" aria-label={t('layout')}>
      {LAYOUT_OPTIONS.map((option) => {
        const Icon = option.icon;
        const label = t(option.labelKey);
        return (
          <button
            aria-label={label}
            className={layoutMode === option.value ? 'selected' : ''}
            key={option.value}
            onClick={() => setLayoutMode(option.value)}
            title={label}
            type="button"
          >
            <Icon aria-hidden="true" />
          </button>
        );
      })}
    </div>
  );
}

function NavigationSwitcher({
  navigationMode,
  setNavigationMode,
  t,
}: {
  navigationMode: NavigationMode;
  setNavigationMode: (navigationMode: NavigationMode) => void;
  t: Translator;
}) {
  return (
    <div className="segmented-control experience-nav-switcher" aria-label={t('nav')} role="group">
      {NAV_OPTIONS.map((option) => {
        const Icon = option.icon;
        const label = t(option.labelKey);
        return (
          <button
            aria-label={label}
            className={navigationMode === option.value ? 'selected' : ''}
            key={option.value}
            onClick={() => setNavigationMode(option.value)}
            title={label}
            type="button"
          >
            <Icon aria-hidden="true" />
          </button>
        );
      })}
    </div>
  );
}

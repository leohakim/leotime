import { afterEach, describe, expect, test } from 'vitest';
import {
  applyExperienceAttributes,
  applyExperienceMetaColor,
  EXPERIENCE_PRESET_DEFINITIONS,
  getExperiencePresetDimensions,
  inferExperiencePreset,
  NAMED_EXPERIENCE_PRESETS,
  readExperiencePreset,
  readNavigationMode,
  SOLIDTIME_EXACT_REFERENCE,
  THEME_META_COLORS,
} from './experience';
import type { ThemeMode } from './api';

afterEach(() => {
  window.localStorage.clear();
  document.documentElement.removeAttribute('data-theme');
  document.documentElement.removeAttribute('data-layout');
  document.documentElement.removeAttribute('data-nav');
  document.documentElement.removeAttribute('data-preset');
});

describe('experience state', () => {
  test('recognizes the legacy default as workbench-pro', () => {
    expect(inferExperiencePreset({ themeMode: 'solid', layoutMode: 'solid', navigationMode: 'sidebar' })).toBe('workbench-pro');
  });

  test('maps every named preset to its dimensions', () => {
    for (const preset of NAMED_EXPERIENCE_PRESETS) {
      expect(getExperiencePresetDimensions(preset)).toEqual(EXPERIENCE_PRESET_DEFINITIONS[preset]);
      expect(inferExperiencePreset(EXPERIENCE_PRESET_DEFINITIONS[preset])).toBe(
        preset === 'solidtime-exact' ? 'workbench-pro' : preset,
      );
    }
  });

  test('marks non-catalog combinations as custom', () => {
    expect(inferExperiencePreset({ themeMode: 'dark', layoutMode: 'compact', navigationMode: 'sidebar' })).toBe('custom');
  });

  test('reads valid local navigation and preset values', () => {
    window.localStorage.setItem('leotime.nav', 'bottom-tabs');
    window.localStorage.setItem('leotime.preset', 'focus-dark');

    expect(readNavigationMode()).toBe('bottom-tabs');
    expect(readExperiencePreset()).toBe('focus-dark');
  });

  test('falls back safely for invalid local navigation and preset values', () => {
    window.localStorage.setItem('leotime.nav', 'drawer');
    window.localStorage.setItem('leotime.preset', 'not-a-preset');

    expect(readNavigationMode()).toBe('sidebar');
    expect(readExperiencePreset()).toBe('custom');
  });

  test('applies all four root attributes', () => {
    applyExperienceAttributes({
      themeMode: 'light',
      layoutMode: 'compact',
      navigationMode: 'bottom-tabs',
      preset: 'mobile-flow',
    });

    expect(document.documentElement.dataset).toMatchObject({
      theme: 'light',
      layout: 'compact',
      nav: 'bottom-tabs',
      preset: 'mobile-flow',
    });
  });

  test('maps each theme to a distinct meta theme-color', () => {
    const meta = document.createElement('meta');
    meta.setAttribute('name', 'theme-color');
    document.head.appendChild(meta);

    const expected: Record<ThemeMode, string> = {
      solid: '#0c0d10',
      light: '#eef2f7',
      dark: '#07080a',
      minimal: '#f3f3f0',
    };

    expect(THEME_META_COLORS).toEqual(expected);

    for (const [themeMode, color] of Object.entries(expected) as Array<[ThemeMode, string]>) {
      applyExperienceMetaColor(themeMode);
      expect(meta.getAttribute('content')).toBe(color);
    }

    meta.remove();
  });

  test('documents the SolidTime Exact reference pin', () => {
    expect(SOLIDTIME_EXACT_REFERENCE.release).toBe('v0.15.1');
    expect(getExperiencePresetDimensions('solidtime-exact')).toEqual(EXPERIENCE_PRESET_DEFINITIONS['workbench-pro']);
  });
});

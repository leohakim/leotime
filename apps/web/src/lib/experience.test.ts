import { afterEach, describe, expect, test } from 'vitest';
import {
  applyExperienceAttributes,
  inferExperiencePreset,
  readExperiencePreset,
  readNavigationMode,
} from './experience';

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

  test('marks non-baseline legacy combinations as custom', () => {
    expect(inferExperiencePreset({ themeMode: 'dark', layoutMode: 'compact', navigationMode: 'sidebar' })).toBe('custom');
  });

  test('falls back safely for invalid local navigation and preset values', () => {
    window.localStorage.setItem('leotime.nav', 'bottom-tabs');
    window.localStorage.setItem('leotime.preset', 'not-a-preset');

    expect(readNavigationMode()).toBe('sidebar');
    expect(readExperiencePreset()).toBe('custom');
  });

  test('applies all four root attributes', () => {
    applyExperienceAttributes({ themeMode: 'light', layoutMode: 'compact', navigationMode: 'sidebar', preset: 'custom' });

    expect(document.documentElement.dataset).toMatchObject({
      theme: 'light',
      layout: 'compact',
      nav: 'sidebar',
      preset: 'custom',
    });
  });
});

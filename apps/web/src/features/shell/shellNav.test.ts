import { describe, expect, test } from 'vitest';
import {
  isReportingRoute,
  isShellMoreRoute,
  shellNavItemIsActive,
  SHELL_BOTTOM_NAV_ROUTES,
  SHELL_MORE_NAV,
} from './shellNav';

describe('shellNav', () => {
  test('treats overview and detailed as reporting routes', () => {
    expect(isReportingRoute('overview')).toBe(true);
    expect(isReportingRoute('detailed')).toBe(true);
    expect(isReportingRoute('daily-summary')).toBe(true);
    expect(isReportingRoute('timesheet')).toBe(false);
  });

  test('marks overview active from detailed reporting child', () => {
    expect(shellNavItemIsActive('detailed', 'overview')).toBe(true);
    expect(shellNavItemIsActive('timesheet', 'overview')).toBe(false);
  });

  test('routes secondary destinations through the more menu', () => {
    expect(SHELL_BOTTOM_NAV_ROUTES).toEqual(['dashboard', 'timesheet', 'calendar', 'overview']);
    expect(isShellMoreRoute('projects')).toBe(true);
    expect(isShellMoreRoute('timesheet')).toBe(false);
    expect(SHELL_MORE_NAV.some((item) => item.route === 'settings')).toBe(true);
  });
});

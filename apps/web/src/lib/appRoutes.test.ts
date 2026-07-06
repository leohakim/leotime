import { describe, expect, test } from 'vitest';
import { DEFAULT_ROUTE, parseRoute, routeHref } from './appRoutes';

describe('appRoutes', () => {
  test('defaults to timesheet when hash is empty', () => {
    expect(parseRoute('')).toBe(DEFAULT_ROUTE);
    expect(parseRoute('#')).toBe(DEFAULT_ROUTE);
  });

  test('parses known routes from hash', () => {
    expect(parseRoute('#dashboard')).toBe('dashboard');
    expect(parseRoute('#clients')).toBe('clients');
    expect(parseRoute('#import-export')).toBe('import-export');
    expect(parseRoute('#reports')).toBe('overview');
  });

  test('builds route hrefs', () => {
    expect(routeHref('invoices')).toBe('#invoices');
  });
});

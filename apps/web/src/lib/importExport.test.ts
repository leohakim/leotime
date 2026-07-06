import { describe, expect, test } from 'vitest';
import { normalizeSolidtimeImportSummary } from './api';

describe('normalizeSolidtimeImportSummary', () => {
  test('fills null warnings and errors from API responses', () => {
    const summary = normalizeSolidtimeImportSummary({
      provider: 'solidtime',
      exportId: 'export-1',
      version: '1.0',
      dryRun: true,
      warnings: null as unknown as string[],
      errors: null as unknown as string[],
    });

    expect(summary.warnings).toEqual([]);
    expect(summary.errors).toEqual([]);
  });
});

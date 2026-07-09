import { describe, expect, test } from 'vitest';
import type { Client, Project } from './api';
import { hasBillableRate } from './billable';

const clients: Client[] = [
  {
    id: 'cli_1',
    name: 'Osoigo',
    email: '',
    taxId: '',
    billingAddress: '',
    defaultCurrency: 'EUR',
    defaultHourlyRateMinor: 3500,
    archivedAt: '',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
  },
];

const projectWithRate: Project = {
  id: 'prj_1',
  clientId: 'cli_1',
  clientName: 'Osoigo',
  name: 'RTVE',
  color: '#2563eb',
  defaultHourlyRateMinor: 3500,
  archivedAt: '',
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
};

const projectWithoutRate: Project = {
  ...projectWithRate,
  id: 'prj_2',
  defaultHourlyRateMinor: null,
};

describe('hasBillableRate', () => {
  test('returns true when project has a rate', () => {
    expect(hasBillableRate(projectWithRate, clients)).toBe(true);
  });

  test('returns true when only the client has a rate', () => {
    expect(hasBillableRate(projectWithoutRate, clients)).toBe(true);
  });

  test('returns false when neither project nor client has a rate', () => {
    expect(
      hasBillableRate(
        { ...projectWithoutRate, clientId: 'cli_2' },
        [{ ...clients[0], id: 'cli_2', defaultHourlyRateMinor: 0 }],
      ),
    ).toBe(false);
  });
});

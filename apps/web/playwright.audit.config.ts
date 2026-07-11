import { defineConfig } from '@playwright/test';
import { mkdtempSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { auditAuthFile } from './e2e/audit-auth';

const auditDataDir = mkdtempSync(join(tmpdir(), 'leotime-ui-audit-'));
const apiEnv = {
  LEOTIME_DB_PATH: join(auditDataDir, 'leotime.db'),
  LEOTIME_DOCUMENT_ROOT: join(auditDataDir, 'documents'),
  LEOTIME_HTTP_ADDR: ':18080',
  LEOTIME_SCHEDULER_ENABLED: 'false',
  LEOTIME_BACKUP_SCHEDULER_ENABLED: 'false',
};

export default defineConfig({
  testDir: './e2e',
  testMatch: /(visual|accessibility)-audit\.spec\.ts/,
  timeout: 60_000,
  fullyParallel: false,
  workers: 1,
  outputDir: 'test-results/ui-audit',
  use: {
    baseURL: 'http://127.0.0.1:5174',
    colorScheme: 'light',
    locale: 'es-ES',
    trace: 'retain-on-failure',
  },
  webServer: [
    {
      command: 'go run ./cmd/leotime seed --user-email admin@example.com && go run ./cmd/leotime',
      cwd: '../api',
      url: 'http://127.0.0.1:18080/api/health',
      reuseExistingServer: false,
      env: apiEnv,
      timeout: 120_000,
    },
    {
      command: 'npm run dev -- --host 127.0.0.1 --port 5174',
      url: 'http://127.0.0.1:5174',
      reuseExistingServer: false,
      env: { LEOTIME_API_PROXY_TARGET: 'http://127.0.0.1:18080' },
      timeout: 120_000,
    },
  ],
  projects: [
    {
      name: 'setup',
      testMatch: /audit-auth\.setup\.ts/,
    },
    {
      name: 'desktop-1440',
      testMatch: /(visual|accessibility)-audit\.spec\.ts/,
      use: { viewport: { width: 1440, height: 1100 }, storageState: auditAuthFile },
      dependencies: ['setup'],
    },
    {
      name: 'tablet-834',
      testMatch: /(visual|accessibility)-audit\.spec\.ts/,
      use: { viewport: { width: 834, height: 1112 }, storageState: auditAuthFile },
      dependencies: ['setup'],
    },
    {
      name: 'mobile-390',
      testMatch: /(visual|accessibility)-audit\.spec\.ts/,
      use: { viewport: { width: 390, height: 844 }, storageState: auditAuthFile },
      dependencies: ['setup'],
    },
  ],
});

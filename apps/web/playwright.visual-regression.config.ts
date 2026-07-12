import { defineConfig } from '@playwright/test';
import { mkdtempSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { auditAuthFile, visualRegressionNow } from './e2e/audit-auth';

const regressionDataDir = mkdtempSync(join(tmpdir(), 'leotime-ui-regression-'));
const apiEnv = {
  LEOTIME_DB_PATH: join(regressionDataDir, 'leotime.db'),
  LEOTIME_DOCUMENT_ROOT: join(regressionDataDir, 'documents'),
  LEOTIME_HTTP_ADDR: ':18081',
  LEOTIME_SCHEDULER_ENABLED: 'false',
  LEOTIME_BACKUP_SCHEDULER_ENABLED: 'false',
  LEOTIME_SEED_NOW: visualRegressionNow,
};

export default defineConfig({
  testDir: './e2e',
  testMatch: 'visual-regression.spec.ts',
  snapshotPathTemplate: '{testDir}/{testFilePath}-snapshots/{arg}-{projectName}{ext}',
  timeout: 60_000,
  fullyParallel: false,
  workers: 1,
  outputDir: 'test-results/ui-regression',
  expect: {
    toHaveScreenshot: {
      animations: 'disabled',
      maxDiffPixelRatio: 0.02,
    },
  },
  use: {
    baseURL: 'http://127.0.0.1:5175',
    colorScheme: 'light',
    locale: 'es-ES',
    trace: 'retain-on-failure',
  },
  webServer: [
    {
      command: 'go run ./cmd/leotime seed --user-email admin@example.com && go run ./cmd/leotime',
      cwd: '../api',
      url: 'http://127.0.0.1:18081/api/health',
      reuseExistingServer: false,
      env: apiEnv,
      timeout: 120_000,
    },
    {
      command: 'npm run dev -- --host 127.0.0.1 --port 5175',
      url: 'http://127.0.0.1:5175',
      reuseExistingServer: false,
      env: { LEOTIME_API_PROXY_TARGET: 'http://127.0.0.1:18081' },
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
      testMatch: 'visual-regression.spec.ts',
      use: { viewport: { width: 1440, height: 1100 }, storageState: auditAuthFile },
      dependencies: ['setup'],
    },
    {
      name: 'tablet-834',
      testMatch: 'visual-regression.spec.ts',
      use: { viewport: { width: 834, height: 1112 }, storageState: auditAuthFile },
      dependencies: ['setup'],
    },
    {
      name: 'mobile-390',
      testMatch: 'visual-regression.spec.ts',
      use: { viewport: { width: 390, height: 844 }, storageState: auditAuthFile },
      dependencies: ['setup'],
    },
  ],
});

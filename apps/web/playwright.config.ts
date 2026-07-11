import { defineConfig, devices } from '@playwright/test';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

const e2eDataDir = join(tmpdir(), 'leotime-e2e');
const apiEnv = {
  LEOTIME_DB_PATH: join(e2eDataDir, 'leotime.db'),
  LEOTIME_DOCUMENT_ROOT: join(e2eDataDir, 'documents'),
  LEOTIME_SCHEDULER_ENABLED: 'false',
  LEOTIME_BACKUP_SCHEDULER_ENABLED: 'false',
};

export default defineConfig({
  testDir: './e2e',
  testIgnore: /(visual|accessibility)-audit\.spec\.ts|audit-auth\.setup\.ts/,
  timeout: 30_000,
  use: {
    baseURL: 'http://127.0.0.1:5173',
    trace: 'on-first-retry',
  },
  webServer: [
    {
      command: 'go run ./cmd/leotime',
      cwd: '../api',
      url: 'http://127.0.0.1:8080/api/health',
      reuseExistingServer: !process.env.CI,
      env: apiEnv,
      timeout: 120_000,
    },
    {
      command: 'npm run dev -- --host 127.0.0.1',
      url: 'http://127.0.0.1:5173',
      reuseExistingServer: !process.env.CI,
    },
  ],
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});

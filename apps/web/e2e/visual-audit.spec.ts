import { expect, test, type Page, type TestInfo } from '@playwright/test';
import { resolve } from 'node:path';
import { emptyStorageState, openAuthenticatedRoute } from './audit-auth';

const evidenceRoot = resolve(process.cwd(), '../../docs/assets/ui-audit/2026-07-11');
const authenticatedSurfaces = [
  ['timesheet', 'timesheet'],
  ['manual-time-entry', 'manual-entry'],
  ['calendar', 'calendar'],
  ['dashboard', 'dashboard'],
  ['overview', 'reports'],
  ['invoices', 'invoices'],
  ['profile', 'settings-profile'],
] as const;

function evidencePath(testInfo: TestInfo, surface: string): string {
  return resolve(evidenceRoot, testInfo.project.name, `${surface}.jpg`);
}

async function capture(page: Page, testInfo: TestInfo, surface: string) {
  await page.screenshot({
    path: evidencePath(testInfo, surface),
    fullPage: true,
    type: 'jpeg',
    quality: 82,
  });
}

test.describe('login screen', () => {
  test.use({ storageState: emptyStorageState });

  test('login', async ({ page }, testInfo) => {
    await page.goto('/');
    await expect(page.locator('.login-hero')).toBeVisible();
    await expect(page.locator('.login-panel')).toBeVisible();
    await capture(page, testInfo, 'login');
  });
});

test.describe('seeded owner surfaces', () => {
  test('captures all authenticated routes', async ({ page }, testInfo) => {
    for (const [route, surface] of authenticatedSurfaces) {
      await openAuthenticatedRoute(page, route);
      await expect(page.locator('.page-content')).toBeVisible();
      await page.waitForLoadState('networkidle');
      await capture(page, testInfo, surface);
    }
  });
});

import { expect, test, type Page, type TestInfo } from '@playwright/test';
import { resolve } from 'node:path';

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

async function signIn(page: Page) {
  await page.goto('/');
  await page.getByLabel(/email/i).fill('admin@example.com');
  await page.getByLabel(/contrase|password/i).fill('change-me-now');
  await page.getByRole('button', { name: /entrar|sign in/i }).click();
  await expect(page.locator('.app-shell')).toBeVisible();
}

test('login', async ({ page }, testInfo) => {
  await page.goto('/');
  await expect(page.locator('.login-hero')).toBeVisible();
  await expect(page.locator('.login-panel')).toBeVisible();
  await capture(page, testInfo, 'login');
});

test.describe('seeded owner surfaces', () => {
  test('captures all authenticated routes', async ({ page }, testInfo) => {
    await signIn(page);

    for (const [route, surface] of authenticatedSurfaces) {
      await page.goto(`/#${route}`);
      await expect(page.locator('.app-shell')).toBeVisible();
      await expect(page.locator('.page-content')).toBeVisible();
      await page.waitForLoadState('networkidle');
      await capture(page, testInfo, surface);
    }
  });
});

import { expect, type Page } from '@playwright/test';
import { mkdirSync } from 'node:fs';
import { dirname, join } from 'node:path';

export const auditAuthFile = join(process.cwd(), 'e2e/.auth/audit-owner.json');

export const visualRegressionNow = '2026-07-11T12:00:00Z';

export const visualRegressionWeekAnchor = '2026-07-07';

export const emptyStorageState = { cookies: [] as [], origins: [] as [] };

export async function prepareVisualRegressionPage(page: Page) {
  await page.addInitScript(
    ({ week, month, preset }) => {
      window.localStorage.setItem('leotime.timesheetWeek', week);
      window.localStorage.setItem('leotime.calendarMonth', month);
      window.localStorage.setItem('leotime.calendarDay', '');
      window.localStorage.setItem('leotime.timeView', 'timesheet');
      window.localStorage.setItem('leotime.theme', 'solid');
      window.localStorage.setItem('leotime.layout', 'solid');
      window.localStorage.setItem('leotime.nav', 'sidebar');
      window.localStorage.setItem('leotime.preset', preset);
      window.localStorage.setItem('leotime.locale', 'es');

      if (!document.getElementById('visual-regression-fonts')) {
        const fontLink = document.createElement('link');
        fontLink.id = 'visual-regression-fonts';
        fontLink.rel = 'stylesheet';
        fontLink.href =
          'https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap';
        document.head.appendChild(fontLink);

        const style = document.createElement('style');
        style.id = 'visual-regression-font-lock';
        style.textContent = `
          html, body, *, *::before, *::after {
            font-family: Inter, Helvetica, Arial, sans-serif !important;
            font-synthesis: none !important;
          }
        `;
        document.head.appendChild(style);
      }
    },
    {
      week: visualRegressionWeekAnchor,
      month: '2026-07-01',
      preset: 'workbench-pro',
    },
  );
}

export async function stabilizeVisualRegressionRendering(page: Page) {
  await page.evaluate(async () => {
    try {
      await document.fonts.load('400 16px Inter');
      await document.fonts.load('500 16px Inter');
      await document.fonts.load('600 16px Inter');
    } catch {
      // Fall back to whatever fonts are available.
    }
    await document.fonts.ready;
  });
}

export async function waitForTimesheetSurface(page: Page) {
  await expect(page.locator('#timesheet')).toBeVisible();
  await page
    .waitForResponse((response) => response.url().includes('/api/v1/time-entries') && response.ok(), {
      timeout: 15_000,
    })
    .catch(() => undefined);
  await expect(page.locator('#timesheet .time-entry-row').first()).toBeVisible({ timeout: 15_000 });
  await expect(page.locator('#timesheet .sync-pill')).toHaveCount(0, { timeout: 15_000 });
  await stabilizeVisualRegressionRendering(page);
}

export async function waitForDashboardSurface(page: Page) {
  await expect(page.locator('#dashboard')).toBeVisible();
  await page
    .waitForResponse((response) => response.url().includes('/api/v1/dashboard/stats') && response.ok(), {
      timeout: 15_000,
    })
    .catch(() => undefined);
  await expect(page.locator('#dashboard .dashboard-stat-card').first()).toBeVisible({ timeout: 15_000 });
  await expect(page.locator('#dashboard .surface-feedback-loading')).toHaveCount(0, { timeout: 15_000 });
  await stabilizeVisualRegressionRendering(page);
}

export async function signIn(page: Page) {
  await page.goto('/');
  await page.getByLabel(/email/i).fill('admin@example.com');
  await page.getByLabel(/contrase|password/i).fill('change-me-now');
  await page.getByRole('button', { name: /entrar|sign in/i }).click();
  await expect(page.locator('.app-shell')).toBeVisible();
}

export async function saveAuditAuthState(page: Page) {
  mkdirSync(dirname(auditAuthFile), { recursive: true });
  await page.context().storageState({ path: auditAuthFile });
}

export async function openAuthenticatedRoute(page: Page, route: string) {
  await page.goto(`/#${route}`);
  await expect(page.locator('.app-shell')).toBeVisible();
}

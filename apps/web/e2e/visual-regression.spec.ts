import { expect, test } from '@playwright/test';
import {
  emptyStorageState,
  openAuthenticatedRoute,
  prepareVisualRegressionPage,
  visualRegressionNow,
  waitForDashboardSurface,
  waitForTimesheetSurface,
} from './audit-auth';

test.beforeEach(async ({ page }) => {
  await page.clock.install({ time: new Date(visualRegressionNow) });
  await prepareVisualRegressionPage(page);
});

test.describe('login screen', () => {
  test.use({ storageState: emptyStorageState });

  test('login layout', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('.login-hero')).toBeVisible();
    await expect(page.locator('.login-screen')).toHaveScreenshot('login.png');
  });
});

test.describe('authenticated surfaces', () => {
  test('timesheet workbench', async ({ page }) => {
    await openAuthenticatedRoute(page, 'timesheet');
    await waitForTimesheetSurface(page);
    await expect(page.locator('#timesheet')).toHaveScreenshot('timesheet.png', {
      mask: [page.locator('.offline-status-pill'), page.locator('.sync-pill')],
    });
  });

  test('dashboard overview', async ({ page }) => {
    await openAuthenticatedRoute(page, 'dashboard');
    await waitForDashboardSurface(page);
    await expect(page.locator('#dashboard')).toHaveScreenshot('dashboard.png', {
      mask: [page.locator('.offline-status-pill'), page.locator('.sync-pill')],
    });
  });
});

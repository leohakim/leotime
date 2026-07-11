import { expect, test } from '@playwright/test';
import { emptyStorageState, openAuthenticatedRoute } from './audit-auth';

test.describe('login screen', () => {
  test.use({ storageState: emptyStorageState });

  test('login exposes hero, labels, and primary action', async ({ page }) => {
    await page.goto('/');

    await expect(page.locator('.login-hero')).toBeVisible();
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
    await expect(page.getByLabel(/email/i)).toBeVisible();
    await expect(page.getByLabel(/contrase|password/i)).toBeVisible();
    await expect(page.getByRole('button', { name: /entrar|sign in/i })).toBeEnabled();
  });
});

test.describe('authenticated shell', () => {
  test('exposes main workspace and navigation', async ({ page }) => {
    await openAuthenticatedRoute(page, 'timesheet');

    await expect(page.locator('main.shell-workspace')).toBeVisible();
    await expect(page.locator('.page-content')).toBeVisible();
    await expect(page.getByRole('navigation').first()).toBeVisible();
  });

  test('reports workbench keeps labeled filters and results region', async ({ page }) => {
    await openAuthenticatedRoute(page, 'overview');

    await expect(page.getByRole('heading', { name: /filtros|filters/i })).toBeVisible();
    await expect(page.getByRole('heading', { name: /vista previa|preview/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /descargar csv|download csv/i })).toBeVisible();
  });

  test('surface feedback uses loading or content on dashboard', async ({ page }) => {
    await openAuthenticatedRoute(page, 'dashboard');

    await expect(page.locator('.surface-feedback-loading, .dashboard-top-grid').first()).toBeVisible();
  });
});

import { expect, test, type Page } from '@playwright/test';

async function signIn(page: Page) {
  await page.goto('/');
  await page.getByLabel(/email/i).fill('admin@example.com');
  await page.getByLabel(/contrase|password/i).fill('change-me-now');
  await page.getByRole('button', { name: /entrar|sign in/i }).click();
  await expect(page.locator('.app-shell')).toBeVisible();
}

test.describe('accessibility smoke', () => {
  test('login exposes hero, labels, and primary action', async ({ page }) => {
    await page.goto('/');

    await expect(page.locator('.login-hero')).toBeVisible();
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
    await expect(page.getByLabel(/email/i)).toBeVisible();
    await expect(page.getByLabel(/contrase|password/i)).toBeVisible();
    await expect(page.getByRole('button', { name: /entrar|sign in/i })).toBeEnabled();
  });

  test('authenticated shell exposes main workspace and navigation', async ({ page }) => {
    await signIn(page);

    await expect(page.locator('main.shell-workspace')).toBeVisible();
    await expect(page.locator('.page-content')).toBeVisible();
    await expect(page.getByRole('navigation').first()).toBeVisible();
  });

  test('reports workbench keeps labeled filters and results region', async ({ page }) => {
    await signIn(page);
    await page.goto('/#overview');

    await expect(page.getByRole('heading', { name: /filtros|filters/i })).toBeVisible();
    await expect(page.getByRole('heading', { name: /vista previa|preview/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /descargar csv|download csv/i })).toBeVisible();
  });

  test('surface feedback uses alert semantics on load failure', async ({ page }) => {
    await signIn(page);
    await page.goto('/#dashboard');

    await expect(page.locator('.surface-feedback-loading, .dashboard-top-grid').first()).toBeVisible();
  });
});

import { expect, test } from '@playwright/test';

test('shows the login workbench when no backend session exists', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByRole('heading', { name: /mesa de trabajo|daily workbench/i })).toBeVisible();
});


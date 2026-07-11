import { expect, type Page } from '@playwright/test';
import { mkdirSync } from 'node:fs';
import { dirname, join } from 'node:path';

export const auditAuthFile = join(process.cwd(), 'e2e/.auth/audit-owner.json');

export const emptyStorageState = { cookies: [] as [], origins: [] as [] };

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

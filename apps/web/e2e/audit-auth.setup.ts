import { test as setup } from '@playwright/test';
import { saveAuditAuthState, signIn } from './audit-auth';

setup('authenticate audit owner', async ({ page }) => {
  await signIn(page);
  await saveAuditAuthState(page);
});

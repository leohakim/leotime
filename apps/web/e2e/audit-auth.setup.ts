import { test as setup } from '@playwright/test';
import { prepareVisualRegressionPage, saveAuditAuthState, signIn, visualRegressionNow } from './audit-auth';

setup('authenticate audit owner', async ({ page }) => {
  await page.clock.install({ time: new Date(visualRegressionNow) });
  await prepareVisualRegressionPage(page);
  await signIn(page);
  await saveAuditAuthState(page);
});

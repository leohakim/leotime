# UI/UX Sprint 1 Visual Audit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Produce a reproducible desktop, tablet, and mobile audit of leotime's main workflows, with versioned visual evidence and a prioritized friction map that directly scopes UI/UX Sprint 2.

**Architecture:** Keep the existing smoke suite unchanged and add an isolated Playwright audit configuration with its own temporary seeded SQLite database and ports. A focused capture spec signs in with the synthetic bootstrap owner, visits the approved core surfaces at three fixed viewports, and writes JPEG evidence under `docs/assets/ui-audit/2026-07-11/`; a current-behavior document turns that evidence into prioritized, testable findings without changing product UI.

**Tech Stack:** React 19, Vite 7, Playwright 1.57, Go seed command, SQLite, Markdown.

## Global Constraints

- This slice is audit and evidence only; do not redesign components, alter business behavior, or introduce the Sprint 2 token architecture.
- Preserve the current `solid`, `minimal`, and `compact` layout modes and the existing hash routes.
- Use only the synthetic `admin@example.com` / `change-me-now` audit owner and the repository seed command; never use production data, exports, credentials, invoices, or PDFs.
- Capture exactly three baselines: desktop `1440x1100`, tablet `834x1112`, and mobile `390x844`.
- Audit login, shell/navigation, timer and quick capture, manual entry, timesheet, calendar, dashboard, reports, invoices, and settings/profile.
- Classify findings as `P0` (workflow blocked or content inaccessible), `P1` (high-frequency friction or serious responsive/accessibility problem), or `P2` (polish and consistency).
- Keep Spanish as the baseline locale and mention English only when copy expansion or localization exposes a distinct issue.
- Do not run `git commit` unless the user explicitly asks; provide the proposed Conventional Commit message at handoff.

---

### Task 1: Add an isolated, seeded visual-audit runner

**Files:**
- Modify: `apps/web/vite.config.ts`
- Modify: `apps/web/playwright.config.ts`
- Create: `apps/web/playwright.audit.config.ts`
- Modify: `apps/web/package.json`

**Interfaces:**
- Consumes: `go run ./cmd/leotime seed`, the default bootstrap credentials, and Vite's existing `/api` proxy.
- Produces: `npm --workspace @leotime/web run test:e2e:audit`, with API on `127.0.0.1:18080`, Vite on `127.0.0.1:5174`, and Playwright projects named `desktop-1440`, `tablet-834`, and `mobile-390`.

- [ ] **Step 1: Make the Vite API proxy configurable while preserving its current default**

Replace the literal proxy target in `apps/web/vite.config.ts` with:

```ts
const apiProxyTarget = process.env.LEOTIME_API_PROXY_TARGET ?? 'http://127.0.0.1:8080';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': apiProxyTarget,
    },
  },
});
```

- [ ] **Step 2: Verify the existing frontend build still accepts the default proxy configuration**

Run: `npm --workspace @leotime/web run build`

Expected: TypeScript and Vite finish successfully and write `apps/web/dist/`.

- [ ] **Step 3: Add the dedicated audit Playwright configuration**

Create `apps/web/playwright.audit.config.ts` with a fresh temporary data directory, disabled schedulers, explicit audit ports, no server reuse, and the three fixed viewports:

```ts
import { defineConfig } from '@playwright/test';
import { mkdtempSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

const auditDataDir = mkdtempSync(join(tmpdir(), 'leotime-ui-audit-'));
const apiEnv = {
  LEOTIME_DB_PATH: join(auditDataDir, 'leotime.db'),
  LEOTIME_DOCUMENT_ROOT: join(auditDataDir, 'documents'),
  LEOTIME_HTTP_ADDR: ':18080',
  LEOTIME_SCHEDULER_ENABLED: 'false',
  LEOTIME_BACKUP_SCHEDULER_ENABLED: 'false',
};

export default defineConfig({
  testDir: './e2e',
  testMatch: 'visual-audit.spec.ts',
  timeout: 60_000,
  fullyParallel: false,
  workers: 1,
  outputDir: 'test-results/ui-audit',
  use: {
    baseURL: 'http://127.0.0.1:5174',
    colorScheme: 'light',
    locale: 'es-ES',
    trace: 'retain-on-failure',
  },
  webServer: [
    {
      command: 'go run ./cmd/leotime seed --user-email admin@example.com && go run ./cmd/leotime',
      cwd: '../api',
      url: 'http://127.0.0.1:18080/api/health',
      reuseExistingServer: false,
      env: apiEnv,
      timeout: 120_000,
    },
    {
      command: 'npm run dev -- --host 127.0.0.1 --port 5174',
      url: 'http://127.0.0.1:5174',
      reuseExistingServer: false,
      env: { LEOTIME_API_PROXY_TARGET: 'http://127.0.0.1:18080' },
      timeout: 120_000,
    },
  ],
  projects: [
    { name: 'desktop-1440', use: { viewport: { width: 1440, height: 1100 } } },
    { name: 'tablet-834', use: { viewport: { width: 834, height: 1112 } } },
    { name: 'mobile-390', use: { viewport: { width: 390, height: 844 } } },
  ],
});
```

- [ ] **Step 4: Expose the audit command**

Add this script to `apps/web/package.json` without changing `test:e2e`:

```json
"test:e2e:audit": "playwright test --config playwright.audit.config.ts"
```

Add `testIgnore: 'visual-audit.spec.ts'` to `apps/web/playwright.config.ts` so
the ordinary E2E command does not collect the dedicated audit suite.

- [ ] **Step 5: Verify the isolated configuration is discoverable before adding the capture suite**

Run: `npm --workspace @leotime/web run test:e2e:audit -- --list`

Expected: Playwright loads the configuration successfully and reports no matching tests; it must not bind either audit server during `--list`.

- [ ] **Step 6: Review the task diff and prepare its commit message**

Run: `git diff --check -- apps/web/vite.config.ts apps/web/playwright.config.ts apps/web/playwright.audit.config.ts apps/web/package.json`

Expected: no whitespace errors.

Proposed commit subject: `test: add isolated UI audit runner`

### Task 2: Capture the approved core surfaces at all three baselines

**Files:**
- Create: `apps/web/e2e/visual-audit.spec.ts`
- Create through the audit command: `docs/assets/ui-audit/2026-07-11/desktop-1440/*.jpg`
- Create through the audit command: `docs/assets/ui-audit/2026-07-11/tablet-834/*.jpg`
- Create through the audit command: `docs/assets/ui-audit/2026-07-11/mobile-390/*.jpg`

**Interfaces:**
- Consumes: the three Playwright project names from Task 1, hash routes from `src/lib/appRoutes.ts`, and the synthetic seeded owner.
- Produces: 24 deterministic screenshots: eight surfaces for each of the three viewport baselines.

- [ ] **Step 1: Write the capture suite with stable navigation and evidence paths**

Create `apps/web/e2e/visual-audit.spec.ts`:

```ts
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
```

- [ ] **Step 2: List the suite and verify the evidence matrix before starting servers**

Run: `npm --workspace @leotime/web run test:e2e:audit -- --list`

Expected: six tests total: one login test and one authenticated capture test under each of `desktop-1440`, `tablet-834`, and `mobile-390`. The authenticated test reuses one login per viewport so the audit does not trip the production login rate limit.

- [ ] **Step 3: Run the audit capture**

Run: `npm --workspace @leotime/web run test:e2e:audit`

Expected: six tests pass and 24 JPEG files are written below `docs/assets/ui-audit/2026-07-11/`.

- [ ] **Step 4: Verify the exact evidence inventory and image dimensions**

Run:

```bash
find docs/assets/ui-audit/2026-07-11 -type f -name '*.jpg' | sort
sips -g pixelWidth -g pixelHeight docs/assets/ui-audit/2026-07-11/*/*.jpg
```

Expected: eight files per viewport directory; every image has its project's exact width and a height of at least the configured viewport because `fullPage` is enabled.

- [ ] **Step 5: Inspect all 24 images**

Use the in-app browser and local image viewer to inspect every image. For each surface and viewport, record navigation reachability, horizontal overflow, clipped or overlapping controls, density, hierarchy, form/table readability, touch-target concerns, empty/loading/error-state clarity visible in the current component, and whether the primary daily action is obvious.

- [ ] **Step 6: Review the task diff and prepare its commit message**

Run: `git diff --check -- apps/web/e2e/visual-audit.spec.ts docs/assets/ui-audit/2026-07-11`

Expected: no whitespace errors.

Proposed commit subject: `test: capture responsive UI audit baselines`

### Task 3: Publish the prioritized friction map

**Files:**
- Create: `docs/36-ui-ux-visual-audit.md`
- Modify: `docs/00-documentation-index.md`

**Interfaces:**
- Consumes: the 24 screenshots and direct inspection notes from Task 2.
- Produces: stable finding IDs `UXA-001`, `UXA-002`, and onward; each finding has a priority, affected surfaces/viewports, evidence links, impact, and a concrete target sprint.

- [ ] **Step 1: Write the audit document from observed evidence**

Create `docs/36-ui-ux-visual-audit.md` with these sections and no speculative findings:

```markdown
# UI/UX Visual Audit

Audit date: **2026-07-11**

## Scope and method

Document the seeded owner, Spanish locale, current default visual settings,
three exact viewport sizes, eight captured surfaces, and the distinction between
observed screenshot evidence and code-inspected loading/error/empty behavior.

## Evidence matrix

Provide one row per surface and relative Markdown links to its desktop, tablet,
and mobile JPEG files under `assets/ui-audit/2026-07-11/`.

## Priority model

- P0: workflow blocked or content inaccessible.
- P1: high-frequency friction or serious responsive/accessibility problem.
- P2: polish, density, hierarchy, or consistency issue.

## Findings

Use a table with columns: ID, Priority, Surface/viewports, Observation,
User impact, Evidence, Target sprint.

## Cross-cutting themes

Summarize navigation, responsive layout, information hierarchy, density,
forms/tables, system states, and accessibility patterns supported by multiple
findings.

## Sprint 2 input

List only the foundation decisions required for experience attributes, tokens,
backward-compatible preference hydration, and the `custom` state. Route
component redesign work to Sprints 4–8 rather than pulling it into Sprint 2.

## Verification

Record the exact capture and quality-gate commands and their results.
```

Every finding must cite at least one relative image link or name the exact current component inspected for a non-visual state. Do not assign P0 merely for aesthetic weakness.

- [ ] **Step 2: Embed representative evidence without making the document unwieldy**

Below the evidence matrix, embed one representative desktop image and one representative mobile image with standard Markdown image syntax. Keep the remaining 22 files as links in the matrix so the document remains scannable.

- [ ] **Step 3: Add the audit to the documentation index**

Add `UI/UX visual audit` to the current-status/product documentation area in `docs/00-documentation-index.md`, linking to `36-ui-ux-visual-audit.md` and describing it as the responsive baseline and prioritized friction map for the experience-theme roadmap.

- [ ] **Step 4: Validate links, finding IDs, and priority coverage**

Run:

```bash
rg -n 'UXA-[0-9]{3}|P0|P1|P2|assets/ui-audit/2026-07-11' docs/36-ui-ux-visual-audit.md
test "$(find docs/assets/ui-audit/2026-07-11 -type f -name '*.jpg' | wc -l | tr -d ' ')" = 24
```

Expected: every finding has a stable ID and priority; the document links the dated evidence root; the image count check succeeds.

- [ ] **Step 5: Review the task diff and prepare its commit message**

Run: `git diff --check -- docs/36-ui-ux-visual-audit.md docs/00-documentation-index.md`

Expected: no whitespace errors.

Proposed commit subject: `docs: publish responsive UI friction audit`

### Task 4: Close Sprint 1 and hand Sprint 2 the verified scope

**Files:**
- Modify: `docs/superpowers/specs/2026-07-08-ui-ux-experience-themes-design.md`
- Modify: `docs/12-implementation-plan.md`
- Modify: `docs/13-backlog.md`

**Interfaces:**
- Consumes: completed evidence and `UXA-*` findings from Task 3.
- Produces: roadmap status showing Phase 5 as `Doing`, Sprint 1 as complete, and Sprint 2 (experience attributes and token foundation) as the sole next implementation slice.

- [ ] **Step 1: Record Sprint 1 completion in the approved design spec**

Update the implementation-status section to state that Sprint 1 completed on 2026-07-11 and link `../../36-ui-ux-visual-audit.md` from the spec location. Keep Sprints 2–10 explicitly unimplemented.

- [ ] **Step 2: Correct the stale current-next-task handoff**

Replace the completed H-BACKUP-04 handoff in `docs/12-implementation-plan.md` with Sprint 2 of the UI/UX experience roadmap. Link both the design spec and `docs/36-ui-ux-visual-audit.md`, and state that Sprint 2 is limited to experience state/attributes, semantic token foundations, preference compatibility, and tests—not shell or feature redesign.

- [ ] **Step 3: Update the phase backlog without marking the whole phase done**

In `docs/13-backlog.md`:

- change Phase 5 from `Backlog` to `Doing`;
- add a compact Sprint 1–10 status table under the accepted UI/UX design entry;
- mark Sprint 1 `Done`, Sprint 2 `Next`, and Sprints 3–10 `Backlog`;
- keep visual regression checks in Phase 6 `Backlog`, because this sprint creates audit evidence rather than pixel-diff regression gates.

- [ ] **Step 4: Run focused frontend and documentation checks**

Run:

```bash
npm --workspace @leotime/web test -- --run
npm --workspace @leotime/web run build
npm --workspace @leotime/web run test:e2e
npm --workspace @leotime/web run test:e2e:audit
git diff --check
```

Expected: unit tests pass, production build succeeds, existing smoke E2E passes unchanged, all 24 audit captures pass, and Git reports no whitespace errors.

- [ ] **Step 5: Run the repository-required completion gates**

Run:

```bash
make pre-commit
make smoke
```

Expected: the pre-commit gate passes. Smoke passes against a running production-style app; if no app is running, start the repository's documented local stack and report the exact target URL used.

- [ ] **Step 6: Review scope and prepare the final commit proposal**

Run:

```bash
git status --short
git diff --stat
git diff --check
```

Expected: only the audit runner, capture suite, 24 dated evidence images, audit/index documents, roadmap handoff, and this plan are changed; no production data or UI behavior is present.

Proposed commit:

```text
docs: complete responsive UI and UX audit

Add an isolated seeded Playwright capture matrix for desktop, tablet, and
mobile, publish prioritized UX findings with versioned evidence, and hand the
experience roadmap to the Sprint 2 token foundation.
```

## Plan Self-Review

- **Spec coverage:** Sprint 1's desktop/tablet/mobile capture, prioritized friction map, responsive and system-state review, and visible verification are covered by Tasks 2–4.
- **Scope boundary:** No theme, layout, navigation, shell, or feature behavior changes are included; those remain in Sprints 2–9.
- **Determinism:** The audit uses a fresh temporary database, synthetic seed data, explicit ports, fixed locale/color scheme, fixed viewports, and serial execution.
- **Compatibility:** The ordinary Vite proxy and existing `test:e2e` command retain their current defaults.
- **Data safety:** Only repository-generated synthetic records and development bootstrap credentials are used.

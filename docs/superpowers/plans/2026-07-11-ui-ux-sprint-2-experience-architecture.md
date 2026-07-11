# UI/UX Sprint 2 Experience Architecture Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Establish root experience attributes, semantic design tokens, local navigation/preset state, and backward-compatible preference hydration without redesigning any product screen.

**Architecture:** Keep the server profile contract unchanged: it remains authoritative for locale, `layoutMode`, and `themeMode`. A focused frontend experience module owns `NavigationMode`, `ExperiencePreset`, legacy-safe local storage reads, root DOM attributes, and preset derivation; `App` composes it with the existing profile hydration guard. CSS adds semantic aliases and interaction foundations while existing component selectors continue to use their current classes.

**Tech Stack:** React 19, TypeScript 5.9, Vite 7, Vitest 4, CSS custom properties, localStorage.

## Global Constraints

- Implement only Sprint 2 from the approved [experience design](../specs/2026-07-08-ui-ux-experience-themes-design.md) and the [Sprint 1 audit](../36-ui-ux-visual-audit.md).
- Do not redesign the shell, navigation, timer, timesheet, calendar, dashboard, reports, invoices, or settings; route those changes to Sprints 4–8.
- Preserve API types, profile routes, existing `leotime.theme` and `leotime.layout` keys, existing hash routes, and the `solid`, `minimal`, and `compact` layout classes.
- Keep `nav` and `preset` local-only in Sprint 2. Do not add database columns, migrations, profile API fields, or profile form controls.
- Root attributes must be `data-theme`, `data-layout`, `data-nav`, and `data-preset` on `document.documentElement`.
- The sole initial preset is `workbench-pro` (`solid` theme, `solid` layout, `sidebar` nav). Any legacy combination other than that baseline is `custom`; changing a dimension after a preset is active changes the preset to `custom`.
- Support Spanish and English through existing translations; this slice adds no new visible UI labels.
- Do not run `git commit` unless the user explicitly asks; provide a Conventional Commit proposal at handoff.

---

### Task 1: Define and test the experience-state contract

**Files:**
- Create: `apps/web/src/lib/experience.ts`
- Create: `apps/web/src/lib/experience.test.ts`

**Interfaces:**
- Consumes: `ThemeMode` and `LayoutMode` from `apps/web/src/lib/api.ts`.
- Produces: `NavigationMode`, `ExperiencePreset`, `ExperienceState`, `DEFAULT_NAVIGATION_MODE`, `DEFAULT_EXPERIENCE_PRESET`, `inferExperiencePreset`, `readNavigationMode`, `readExperiencePreset`, `applyExperienceAttributes`, and `applyExperienceMetaColor`.

- [ ] **Step 1: Write the failing contract tests**

Create `apps/web/src/lib/experience.test.ts` with tests that prove the initial preset mapping, invalid local storage fallbacks, and root attributes:

```ts
import { afterEach, describe, expect, test } from 'vitest';
import {
  applyExperienceAttributes,
  inferExperiencePreset,
  readExperiencePreset,
  readNavigationMode,
} from './experience';

afterEach(() => {
  window.localStorage.clear();
  document.documentElement.removeAttribute('data-theme');
  document.documentElement.removeAttribute('data-layout');
  document.documentElement.removeAttribute('data-nav');
  document.documentElement.removeAttribute('data-preset');
});

describe('experience state', () => {
  test('recognizes the legacy default as workbench-pro', () => {
    expect(inferExperiencePreset({ themeMode: 'solid', layoutMode: 'solid', navigationMode: 'sidebar' })).toBe('workbench-pro');
  });

  test('marks non-baseline legacy combinations as custom', () => {
    expect(inferExperiencePreset({ themeMode: 'dark', layoutMode: 'compact', navigationMode: 'sidebar' })).toBe('custom');
  });

  test('falls back safely for invalid local navigation and preset values', () => {
    window.localStorage.setItem('leotime.nav', 'bottom-tabs');
    window.localStorage.setItem('leotime.preset', 'not-a-preset');

    expect(readNavigationMode()).toBe('sidebar');
    expect(readExperiencePreset()).toBe('custom');
  });

  test('applies all four root attributes', () => {
    applyExperienceAttributes({ themeMode: 'light', layoutMode: 'compact', navigationMode: 'sidebar', preset: 'custom' });

    expect(document.documentElement.dataset).toMatchObject({
      theme: 'light',
      layout: 'compact',
      nav: 'sidebar',
      preset: 'custom',
    });
  });
});
```

- [ ] **Step 2: Run the test to verify it fails because the module is absent**

Run: `npm --workspace @leotime/web test -- experience.test.ts --run`

Expected: FAIL with module resolution for `./experience`.

- [ ] **Step 3: Implement the smallest explicit contract**

Create `apps/web/src/lib/experience.ts` using this public contract:

```ts
import type { LayoutMode, ThemeMode } from './api';

export type NavigationMode = 'sidebar';
export type ExperiencePreset = 'workbench-pro' | 'custom';

export type ExperienceState = {
  themeMode: ThemeMode;
  layoutMode: LayoutMode;
  navigationMode: NavigationMode;
  preset: ExperiencePreset;
};

export const DEFAULT_NAVIGATION_MODE: NavigationMode = 'sidebar';
export const DEFAULT_EXPERIENCE_PRESET: ExperiencePreset = 'workbench-pro';

const THEME_META_COLORS: Record<ThemeMode, string> = {
  solid: '#0c0d10', light: '#eef0f4', dark: '#050608', minimal: '#101114',
};

export function inferExperiencePreset({ themeMode, layoutMode, navigationMode }: Omit<ExperienceState, 'preset'>): ExperiencePreset {
  return themeMode === 'solid' && layoutMode === 'solid' && navigationMode === 'sidebar'
    ? DEFAULT_EXPERIENCE_PRESET
    : 'custom';
}

export function readNavigationMode(): NavigationMode {
  return window.localStorage.getItem('leotime.nav') === 'sidebar' ? 'sidebar' : DEFAULT_NAVIGATION_MODE;
}

export function readExperiencePreset(): ExperiencePreset {
  const value = window.localStorage.getItem('leotime.preset');
  return value === 'workbench-pro' || value === 'custom' ? value : 'custom';
}

export function applyExperienceMetaColor(themeMode: ThemeMode) {
  document.querySelector('meta[name="theme-color"]')?.setAttribute('content', THEME_META_COLORS[themeMode]);
}

export function applyExperienceAttributes(state: ExperienceState) {
  const root = document.documentElement;
  root.dataset.theme = state.themeMode;
  root.dataset.layout = state.layoutMode;
  root.dataset.nav = state.navigationMode;
  root.dataset.preset = state.preset;
  applyExperienceMetaColor(state.themeMode);
}
```

- [ ] **Step 4: Run the contract test and the existing theme test**

Run:

```bash
npm --workspace @leotime/web test -- experience.test.ts --run
npm --workspace @leotime/web test -- App.test.tsx --run
```

Expected: new contract tests pass; the existing toolbar behavior remains green before wiring the new state.

- [ ] **Step 5: Review the focused diff and prepare the commit proposal**

Run: `git diff --check -- apps/web/src/lib/experience.ts apps/web/src/lib/experience.test.ts`

Expected: no whitespace errors.

Proposed commit subject: `feat: define UI experience state contract`

### Task 2: Wire experience state through hydration and existing controls

**Files:**
- Modify: `apps/web/src/App.tsx`
- Modify: `apps/web/src/lib/themeUi.tsx`
- Modify: `apps/web/src/App.test.tsx`

**Interfaces:**
- Consumes: Task 1's `ExperienceState`, attribute application, preset inference, and local storage readers.
- Produces: root attributes that always reflect rendered state; existing toolbar/profile theme or layout changes mark the active experience `custom`; existing theme and layout keys keep their current persisted values.

- [ ] **Step 1: Add failing App tests for hydration and the custom transition**

First, replace the inline profile object in `mockFetch` with a mutable
`profileMock` declared beside the existing client/project/task mocks. Reset it
to the current solid/solid profile fixture in the suite's `beforeEach`, and
return it from the `GET /api/v1/profile` branch. Then add these two tests near
the existing toolbar-theme test:

```tsx
test('hydrates every root experience attribute from legacy profile preferences', async () => {
  profileMock.settings.themeMode = 'dark';
  profileMock.layoutMode = 'compact';

  renderApp();

  await screen.findByRole('heading', { name: 'Time Tracker' });
  await waitFor(() => expect(document.documentElement.dataset).toMatchObject({
    theme: 'dark', layout: 'compact', nav: 'sidebar', preset: 'custom',
  }));
});

test('marks the experience custom after changing a preset dimension', async () => {
  renderApp();

  await screen.findByRole('heading', { name: 'Time Tracker' });
  expect(document.documentElement.dataset.preset).toBe('workbench-pro');

  fireEvent.click(screen.getByRole('button', { name: 'Claro' }));

  await waitFor(() => expect(document.documentElement.dataset.preset).toBe('custom'));
  expect(window.localStorage.getItem('leotime.theme')).toBe('light');
  expect(window.localStorage.getItem('leotime.preset')).toBe('custom');
});
```

The `beforeEach` reset prevents a profile override from leaking to another test.

- [ ] **Step 2: Run the two tests to verify they fail on missing attributes/preset persistence**

Run: `npm --workspace @leotime/web test -- App.test.tsx --run`

Expected: the added tests fail because `data-layout`, `data-nav`, `data-preset`, and `leotime.preset` are not yet maintained.

- [ ] **Step 3: Replace the theme-only root effect with the composed experience effect**

In `apps/web/src/lib/themeUi.tsx`, remove the module-level local-storage boot code and `applyTheme`. Export a `useExperienceEffect(state: ExperienceState)` hook that invokes `applyExperienceAttributes(state)` in an effect.

In `apps/web/src/App.tsx`:

1. retain `leotime.theme` and `leotime.layout` state exactly as today;
2. add `navigationMode` from `usePersistentState<NavigationMode>('leotime.nav', readNavigationMode())`;
3. add `preset` from `usePersistentState<ExperiencePreset>('leotime.preset', readExperiencePreset())`;
4. call `useExperienceEffect({ themeMode, layoutMode, navigationMode, preset })`;
5. in `applyThemeMode` and `applyLayoutMode`, set `preset` to `custom` only when the requested value differs from the current value;
6. in the existing profile hydration effect, set theme/layout first, then set the preset with `inferExperiencePreset({ themeMode: profile.settings.themeMode, layoutMode: profile.layoutMode, navigationMode })`;
7. leave `navigationMode` local-only and do not pass it to profile updates.

Use `useCallback` dependencies that include `themeMode`, `layoutMode`, and `setPreset` so a toolbar or saved profile update cannot retain an old preset.

- [ ] **Step 4: Run the focused test cycle**

Run: `npm --workspace @leotime/web test -- App.test.tsx experience.test.ts --run`

Expected: the root attributes hydrate to the server profile values; changing theme creates `custom`; all existing App tests remain green.

- [ ] **Step 5: Build the production client to verify module boundaries**

Run: `npm --workspace @leotime/web run build`

Expected: TypeScript passes; no Node globals are introduced into `vite.config.ts` or browser code.

- [ ] **Step 6: Review the focused diff and prepare the commit proposal**

Run: `git diff --check -- apps/web/src/App.tsx apps/web/src/lib/themeUi.tsx apps/web/src/App.test.tsx`

Expected: no whitespace errors.

Proposed commit subject: `feat: hydrate UI experience attributes`

### Task 3: Add semantic foundations without component redesign

**Files:**
- Modify: `apps/web/src/styles.css`
- Test: `apps/web/src/App.test.tsx`

**Interfaces:**
- Consumes: the root attributes produced by Task 2.
- Produces: semantic aliases for surfaces, content, borders, statuses, focus, spacing, radius, and minimum interactive target size. Existing component CSS remains visually compatible.

- [ ] **Step 1: Add a failing CSS-contract assertion to the App test**

Add a focused assertion that the root keeps layout and navigation attributes when changing theme, proving token selectors can depend on all dimensions:

```tsx
test('keeps layout and navigation attributes while changing theme', async () => {
  renderApp();

  await screen.findByRole('heading', { name: 'Time Tracker' });
  fireEvent.click(screen.getByRole('button', { name: 'Oscuro' }));

  await waitFor(() => expect(document.documentElement.dataset).toMatchObject({
    theme: 'dark', layout: 'solid', nav: 'sidebar', preset: 'custom',
  }));
});
```

- [ ] **Step 2: Run the focused test to verify it fails before Task 2 wiring is complete**

Run: `npm --workspace @leotime/web test -- App.test.tsx --run`

Expected: FAIL while any required root attribute is absent; after Task 2 it must pass without a separate behavioral CSS test.

- [ ] **Step 3: Add foundation and semantic token aliases at the top of `styles.css`**

Inside the existing `:root, [data-theme='solid']` block, add aliases rather than replacing every component value:

```css
  --surface-canvas: var(--bg);
  --surface-panel: var(--surface);
  --surface-raised: var(--surface-2);
  --surface-interactive: var(--surface-3);
  --content-primary: var(--text);
  --content-secondary: var(--muted);
  --content-subtle: var(--faint);
  --border-default: var(--border);
  --border-interactive: var(--border-strong);
  --status-info: var(--blue);
  --status-success: var(--green);
  --status-warning: var(--amber);
  --status-danger: var(--rose);
  --focus-ring: rgb(95 179 217 / 24%);
  --radius-control: 8px;
  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --target-min: 44px;
```

Add equivalent aliases to each existing theme block after its primitive colors. Then migrate only `body`, default `button`, `input/select/button/a:focus-visible`, and `.login-panel` to these aliases. Keep `button` at its current `38px` height: `--target-min` is a later shell/component migration target, not a behavior change in this sprint.

Add inert root selectors that document the future contract without changing layout:

```css
html[data-layout='solid'],
html[data-layout='minimal'],
html[data-layout='compact'],
html[data-nav='sidebar'] {
  color: var(--content-primary);
}
```

- [ ] **Step 4: Run the focused test and build**

Run:

```bash
npm --workspace @leotime/web test -- App.test.tsx experience.test.ts --run
npm --workspace @leotime/web run build
```

Expected: both commands pass and the visual output remains compatible with the Sprint 1 baseline.

- [ ] **Step 5: Review the CSS diff and prepare the commit proposal**

Run: `git diff --check -- apps/web/src/styles.css`

Expected: no whitespace errors.

Proposed commit subject: `style: add semantic experience tokens`

### Task 4: Document the implemented boundary and verify the full slice

**Files:**
- Modify: `docs/25-theme-selector.md`
- Modify: `docs/26-profile-settings-api.md`
- Modify: `docs/12-implementation-plan.md`
- Modify: `docs/13-backlog.md`
- Modify: `docs/superpowers/specs/2026-07-08-ui-ux-experience-themes-design.md`

**Interfaces:**
- Consumes: local-only Sprint 2 experience behavior.
- Produces: accurate current-behavior documentation and roadmap handoff to Sprint 3.

- [ ] **Step 1: Document the root attributes and storage ownership**

Update `docs/25-theme-selector.md` to show the four root attributes and state:

```html
<html data-theme="solid" data-layout="solid" data-nav="sidebar" data-preset="workbench-pro">
```

Document that `leotime.theme` and `leotime.layout` remain profile-compatible; `leotime.nav` and `leotime.preset` are local-only in Sprint 2; `workbench-pro` is the sole initial mapping; and manual dimension changes yield `custom`.

- [ ] **Step 2: State that the API has intentionally not expanded**

In `docs/26-profile-settings-api.md`, retain the exact current profile JSON and add a short Sprint 2 note: navigation and preset are local frontend state, so they are absent from GET/PATCH profile contracts until a later scoped decision.

- [ ] **Step 3: Mark Sprint 2 done and hand off only Sprint 3**

Update the design spec implementation status, the `docs/13-backlog.md` Sprint table, and the `docs/12-implementation-plan.md` current-next-task section. Sprint 2 becomes **Done**, Sprint 3 becomes **Next**, and the handoff specifies selector controls, preset choices, local persistence, and profile synchronization only where API fields already exist.

- [ ] **Step 4: Run full frontend verification and the repository gate**

Run:

```bash
npm --workspace @leotime/web test -- --run
npm --workspace @leotime/web run build
npm --workspace @leotime/web run test:e2e
make pre-commit
```

Expected: unit tests, production build, ordinary E2E smoke, and the required Go/frontend gate pass. Run `make smoke` against a documented temporary production-style instance because this changes root client behavior.

- [ ] **Step 5: Run smoke and inspect the root contract in a browser**

Start the built app with a temporary database and document root under `/tmp`, then run:

```bash
make smoke BASE_URL=http://127.0.0.1:18080
```

Use the in-app browser to sign into that local instance and inspect `document.documentElement.dataset`; it must contain `theme`, `layout`, `nav`, and `preset` after hydration.

- [ ] **Step 6: Final review and commit proposal**

Run:

```bash
git status --short
git diff --check
```

Expected: changes are limited to experience state, attribute/token foundations, tests, and their documentation; no backend schema/API changes or captured user data.

Proposed commit:

```text
feat: add UI experience architecture foundation

Apply theme, layout, navigation, and preset attributes at the document root,
preserve profile-compatible preferences, and add semantic tokens for later
responsive shell and feature work.
```

## Plan Self-Review

- **Spec coverage:** root attributes, tokens, `custom`, legacy theme/layout compatibility, hydration, and tests map to Tasks 1–3.
- **Audit scope:** UXA findings remain deferred to their assigned component sprints; only shared foundation is introduced.
- **API scope:** no profile schema or migration appears in the plan; current profile hydration remains intact.
- **State consistency:** `NavigationMode`, `ExperiencePreset`, storage keys, root attributes, and preset derivation use the same names throughout.
- **No placeholders:** each code or verification action has an exact path, expected result, and concrete interface.

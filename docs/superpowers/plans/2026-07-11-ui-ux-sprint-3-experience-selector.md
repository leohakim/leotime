# UI/UX Sprint 3 Experience Selector Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add visible theme, layout, navigation, and preset controls with local persistence and profile-compatible synchronization for existing fields, without shell or feature redesign.

**Architecture:** Extend the Sprint 2 `experience` module with a preset catalog and additional `NavigationMode` values. Introduce `ExperienceSwitcher` that composes preset, theme, layout, and nav pickers. `App` owns apply handlers that set `custom` when a dimension diverges from the active preset. Shell layout behavior for `sidebar-compact` and `bottom-tabs` remains deferred to Sprint 4; Sprint 3 only sets root attributes and storage.

**Tech Stack:** React 19, TypeScript 5.9, Vitest 4, CSS custom properties, localStorage.

## Global Constraints

- Implement only Sprint 3 from the approved [experience design](../specs/2026-07-08-ui-ux-experience-themes-design.md).
- Do not redesign shell navigation, timer, timesheet, calendar, dashboard, reports, invoices, or settings screens beyond adding selector controls.
- Do not add profile API fields, migrations, or backend changes. `leotime.theme` and `leotime.layout` remain profile-compatible; `leotime.nav` and `leotime.preset` stay local-only.
- Root attributes remain `data-theme`, `data-layout`, `data-nav`, and `data-preset` on `document.documentElement`.
- Manual changes to any dimension after a named preset is active set `preset` to `custom`.
- Do not run `git commit` unless the user explicitly asks; provide a Conventional Commit proposal at handoff.

---

### Task 1: Expand the experience contract

**Files:**
- Modify: `apps/web/src/lib/experience.ts`
- Modify: `apps/web/src/lib/experience.test.ts`

- [ ] **Step 1: Add failing tests for preset catalog and nav modes**

Extend `experience.test.ts` with tests for every named preset mapping, valid `bottom-tabs` navigation reads, and `getExperiencePresetDimensions`.

- [ ] **Step 2: Run tests to verify failure**

Run: `npm --workspace @leotime/web test -- experience.test.ts --run`

- [ ] **Step 3: Implement catalog, inference, and safe readers**

Add `NavigationMode` values `sidebar-compact` and `bottom-tabs`, six named presets plus `custom`, `EXPERIENCE_PRESET_DEFINITIONS`, `getExperiencePresetDimensions`, and updated `inferExperiencePreset`, `readNavigationMode`, and `readExperiencePreset`.

- [ ] **Step 4: Run contract tests**

Run: `npm --workspace @leotime/web test -- experience.test.ts --run`

### Task 2: Build ExperienceSwitcher

**Files:**
- Create: `apps/web/src/lib/experienceUi.tsx`
- Create: `apps/web/src/lib/experienceUi.test.tsx`
- Modify: `apps/web/src/lib/i18n.ts`

- [ ] **Step 1: Add i18n keys for presets, navigation, and experience section labels**

- [ ] **Step 2: Add failing component tests for preset and nav selection**

- [ ] **Step 3: Implement `ExperienceSwitcher` composing preset select, `ThemeSwitcher`, layout buttons, and nav buttons**

- [ ] **Step 4: Run component tests**

Run: `npm --workspace @leotime/web test -- experienceUi.test.tsx --run`

### Task 3: Wire App state and custom transitions

**Files:**
- Modify: `apps/web/src/App.tsx`
- Modify: `apps/web/src/App.test.tsx`

- [ ] **Step 1: Add failing App tests for preset application and nav persistence**

- [ ] **Step 2: Implement `applyExperiencePreset`, `applyNavigationMode`, and pass handlers through `DashboardShell`**

- [ ] **Step 3: Run App and experience tests**

Run: `npm --workspace @leotime/web test -- App.test.tsx experience.test.ts experienceUi.test.tsx --run`

### Task 4: Integrate selectors in toolbar and profile settings

**Files:**
- Modify: `apps/web/src/features/shell/DashboardShell.tsx`
- Modify: `apps/web/src/lib/profileSettingsUi.tsx`
- Modify: `apps/web/src/styles.css`

- [ ] **Step 1: Replace separate toolbar theme/layout controls with `ExperienceSwitcher`**

- [ ] **Step 2: Add experience section to profile settings using the same switcher**

- [ ] **Step 3: Add inert CSS selectors for `data-nav='sidebar-compact'` and `data-nav='bottom-tabs'`**

### Task 5: Document and verify the slice

**Files:**
- Modify: `docs/25-theme-selector.md`
- Modify: `docs/12-implementation-plan.md`
- Modify: `docs/13-backlog.md`
- Modify: `docs/superpowers/specs/2026-07-08-ui-ux-experience-themes-design.md`

- [ ] **Step 1: Document preset catalog, nav modes, and selector locations**

- [ ] **Step 2: Mark Sprint 3 done and hand off Sprint 4**

- [ ] **Step 3: Run full gate**

Run: `make pre-commit`

Proposed final commit:

```text
feat: add experience selector with presets and nav controls

Expose theme, layout, navigation, and preset pickers in the toolbar and
profile settings, expand the experience contract, and keep profile sync
limited to existing theme and layout fields.
```

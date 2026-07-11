# UI/UX Sprint 9 Initial Preset Pack Implementation Plan

**Goal:** Polish and verify the six named experience presets plus login context (UXA-009).

**Approach:** Preset-specific CSS refinements, login hero layout, SolidTime Exact reference pin, and integration tests.

**Status:** Done (2026-07-11).

## Delivered

- Login hero with product features beside the sign-in panel; stacks below `1180px`.
- `data-preset` refinements for all six presets (density, radius, touch targets, shadows).
- `SOLIDTIME_EXACT_REFERENCE` constant in `experience.ts`.
- `docs/37-experience-presets.md` catalog and refresh guidance.
- Auth, experience, and App preset tests; `make pre-commit` passes.

## Files

| Area | Location |
| --- | --- |
| Preset definitions | `apps/web/src/lib/experience.ts` |
| Login screen | `apps/web/src/lib/authUi.tsx` |
| Preset CSS | `apps/web/src/styles.css` |
| Docs | `docs/37-experience-presets.md` |
| Tests | `authUi.test.tsx`, `experience.test.ts`, `App.test.tsx` |

# UI/UX Sprint 6 Weekly Timesheet Implementation Plan

**Goal:** Improve timesheet scanning and mobile row density (UXA-004) without backend changes.

**Approach:** Summary-first rows below `760px` with expand-to-edit inline controls; desktop keeps inline editing.

**Status:** Done (2026-07-11).

## Delivered

- `useCompactTimesheetRows()` listens to `(max-width: 760px)` via `matchMedia`.
- `TimesheetEntryRow` renders a summary row (description, project badge, time range, flags, duration) on compact viewports.
- Edit expands inline controls; **Listo** collapses back to summary mode.
- Desktop keeps the existing always-inline editing layout.
- CSS for `.time-entry-row.is-summary`, `.time-entry-edit-button`, and `.time-entry-done-button`.
- Unit and integration tests; `make pre-commit` passes.

## Files

| Area | Location |
| --- | --- |
| Row logic | `apps/web/src/lib/timeEntryUi.tsx` |
| Styles | `apps/web/src/styles.css` |
| i18n | `apps/web/src/lib/i18n.ts` (`done`) |
| Tests | `apps/web/src/lib/timeEntryUi.test.ts`, `apps/web/src/App.test.tsx` |
| Test setup | `apps/web/src/test/setup.ts` (`matchMedia` polyfill) |

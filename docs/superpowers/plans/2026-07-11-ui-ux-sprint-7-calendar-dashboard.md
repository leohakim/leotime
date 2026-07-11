# UI/UX Sprint 7 Calendar Dashboard Implementation Plan

**Goal:** Improve dashboard and calendar readability on tablet/mobile (UXA-003) without backend changes.

**Approach:** Collapse dashboard grids at `980px`, stack calendar/toolbar navigation earlier, and tighten heatmap/month-nav overflow.

**Status:** Done (2026-07-11).

## Delivered

- `.dashboard-top-grid` and `.dashboard-week-layout` stack at `max-width: 980px` (was `760px`).
- Activity heatmap month navigation wraps without clipping; month label no longer forces `120px` min-width.
- `.time-list-toolbar` stacks on calendar/timesheet at `980px` for centered week/month controls.
- Calendar day cells hide entry-count chips on tablet; day-detail padding tightens on narrow widths.
- Donut breakdown stacks below the chart on tablet.
- Integration test for calendar day detail entries.

## Files

| Area | Location |
| --- | --- |
| Dashboard layout | `apps/web/src/lib/dashboardUi.tsx` |
| Calendar panel | `apps/web/src/lib/calendarUi.tsx` |
| Responsive CSS | `apps/web/src/styles.css` |
| Tests | `apps/web/src/App.test.tsx` |

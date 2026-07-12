# UI/UX QA Checklist

Repeatable gates for experience presets, responsive layouts, and shared feedback
states after the ten-sprint UI/UX roadmap.

## Automated gates

| Gate | Command | What it verifies |
| --- | --- | --- |
| Unit + build | `make pre-commit` | Go vet/tests, Vitest, TypeScript, Vite production build |
| E2E product flows | `make test-e2e` | Authenticated CRUD, timer, navigation smoke |
| Visual + a11y audit | `make audit-ui` | Screenshots at `1440`, `834`, `390` plus accessibility smoke |
| CI accessibility smoke | `make audit-ui-smoke` | Shared-session Playwright checks without regenerating JPEG evidence |

Audit screenshots land in `docs/assets/ui-audit/<date>/` (JPEG, full page). Re-run
after layout changes and commit refreshed assets when the visual baseline should
move forward.

## Responsive checklist

Test each preset at **1440px**, **834px**, and **390px** on:

| Surface | Focus |
| --- | --- |
| Login | Hero + form split; stacked hero below `1180px` |
| Timesheet | Compact summary rows below `760px`; expand-to-edit |
| Manual entry | Sticky editor; directory scroll on narrow viewports |
| Calendar | Toolbar wrap; day cell readability |
| Dashboard | Top grid stacks at `980px`; no horizontal clip |
| Reports | Filter/results workbench; auto-loaded preview |
| Invoices | Draft/directory workbench; empty directory state |
| Settings | Long document acceptable; section headings scannable |

Presets to spot-check manually: `workbench-pro`, `mobile-flow`, `solidtime-exact`.

## Feedback state contract (UXA-010)

| State | Component | Visual |
| --- | --- | --- |
| Loading | `SurfaceLoading` | `.surface-feedback-loading` + `sync-pill` |
| Error (retry) | `SurfaceError`, `QueryErrorBanner` | `.surface-feedback-error`, `role="alert"` |
| Empty | `SurfaceEmpty` | `.panel-empty-state`, muted centered copy |
| Boot / maintenance | `App` boot screen, maintenance banner | Full-screen or warning tint on banner |

Directory lists should keep using `QueryErrorBanner` at the panel top; surface
panels (dashboard stats, report preview, invoice directory) use `Surface*`
helpers for inline fetch states.

## Accessibility smoke

The audit spec checks:

- Login: `h1` hero title, labeled email/password, enabled sign-in button
- Shell: `main.shell-workspace`, landmark navigation (`nav`)
- Reports: filter and preview headings, export control names
- Dashboard: loading feedback or content visible after navigation

For deeper audits, run browser DevTools Lighthouse accessibility on login and
timesheet, or add `@axe-core/playwright` in a future hardening slice.

## Adding a new experience preset

1. Add a definition to `EXPERIENCE_PRESET_DEFINITIONS` in `experience.ts`
   (theme, layout, navigation, label keys).
2. Add i18n strings for the preset name and description.
3. Add optional `[data-preset="your-id"]` refinements in `styles.css` (density,
   radius, touch targets) — keep changes token-driven.
4. Extend `experience.test.ts` and an `App.test.tsx` smoke route if the preset
   changes navigation mode.
5. Run `make pre-commit` and manual responsive spot-check at three widths.
6. Document the preset in [37-experience-presets.md](37-experience-presets.md).

## SolidTime Exact refresh

When SolidTime ships a new UI baseline:

1. Record release tag and commit in `SOLIDTIME_EXACT_REFERENCE` (`experience.ts`).
2. Diff SolidTime CSS/components against leotime `solid` theme tokens.
3. Limit CSS changes to `[data-preset="solidtime-exact"]` where possible.
4. Re-capture audit screenshots and update the verified date in preset docs.
5. Ship as a dedicated sprint; do not mix with unrelated product work.

## Related docs

- [Visual audit findings](36-ui-ux-visual-audit.md)
- [Experience presets](37-experience-presets.md)
- [Theme selector](25-theme-selector.md)
- [Experience design spec](superpowers/specs/2026-07-08-ui-ux-experience-themes-design.md)

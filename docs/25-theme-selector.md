# Theme Selector

The authenticated app supports four color themes, persisted in `localStorage` under `leotime.theme`.

## Experience selector

Sprint 3 exposes the full experience contract in the toolbar and profile settings through `ExperienceSwitcher`:

```html
<html data-theme="solid" data-layout="solid" data-nav="sidebar" data-preset="workbench-pro">
```

### Storage ownership

| Key | Scope | Notes |
| --- | --- | --- |
| `leotime.theme` | Profile-compatible | Hydrated from profile; toolbar/profile changes sync on save |
| `leotime.layout` | Profile-compatible | Hydrated from profile; toolbar/profile changes sync on save |
| `leotime.nav` | Local-only | `sidebar`, `sidebar-compact`, or `bottom-tabs` |
| `leotime.preset` | Local-only | Named preset or `custom` |

### Named presets

| Preset | Theme | Layout | Nav |
| --- | --- | --- | --- |
| `workbench-pro` | `solid` | `solid` | `sidebar` |
| `calm-light` | `light` | `minimal` | `sidebar` |
| `focus-dark` | `dark` | `solid` | `sidebar` |
| `compact-power` | `dark` | `compact` | `sidebar-compact` |
| `mobile-flow` | `light` | `compact` | `bottom-tabs` |
| `solidtime-exact` | `solid` | `solid` | `sidebar` |

Selecting a preset applies all three dimensions. Changing theme, layout, or navigation independently sets `custom`. Shell layout behavior for `sidebar-compact` and `bottom-tabs` is implemented in Sprint 4: compact sidebar on desktop and bottom navigation on tablet/mobile.

## Themes

| Theme | Purpose |
| --- | --- |
| `solid` | Default Solidtime-like dark palette. |
| `light` | Light workspace with higher contrast text, white surfaces, and readable accents. |
| `dark` | Deeper neutral dark palette. |
| `minimal` | Muted, low-saturation dark palette. |

Layout density remains separate under `leotime.layout` (`solid`, `minimal`, `compact`).

## Timer and manual entry capture

Sprint 5 improves the daily capture flow:

- `scrollToManualEntryForm()` navigates to `#manual-time-entry`, scrolls to
  `#manual-time-entry-editor`, and focuses the description field.
- **Nueva entrada** resets the form and scrolls to the editor (UXA-002).
- `.time-entry-workbench` keeps the editor sticky on desktop (UXA-007) and shows
  the form before the directory below `1180px`.
- `.timer-capture-bar` stacks timer actions on narrow viewports with larger touch
  targets for start/stop and manual entry.

## Weekly timesheet rows

Sprint 6 improves mobile scanning for the weekly timesheet (UXA-004):

- Below `760px`, `TimesheetEntryRow` shows a summary row (description, project,
  time range, flags, duration) instead of inline controls.
- **Editar** expands the existing inline editor; **Listo** collapses back to summary.
- Desktop keeps always-inline editing.

## Dashboard and calendar

Sprint 7 improves tablet layout for the overview and calendar (UXA-003):

- Below `980px`, `.dashboard-top-grid` and `.dashboard-week-layout` stack to one column.
- Activity heatmap month navigation wraps without clipping; donut legend stacks under the chart.
- Calendar/timesheet `.time-list-toolbar` stacks week/month controls instead of squeezing three columns.

## Reports and invoices

Sprint 8 clarifies preview hierarchy on reporting screens (UXA-008):

- Reports use a filters + preview workbench; the current month loads automatically on open.
- Invoices split **Nuevo borrador** from the invoice directory/detail panel.
- Compact `.panel-empty-state` replaces tall placeholder regions.

## Experience presets

Sprint 9 polishes the six named presets and the login entry screen (UXA-009):

- Login shows a product hero with feature bullets beside the sign-in panel.
- `data-preset` applies light density, radius, and touch-target refinements per preset.
- `solidtime-exact` is pinned to SolidTime `v0.15.1` — see [37-experience-presets.md](37-experience-presets.md).

## UI

- `ExperienceSwitcher` in the Time Tracker toolbar and profile preferences section.
- `ShellSidebar`, `SidebarNav`, `ShellTopbar`, and `MobileBottomNav` in `apps/web/src/features/shell/`.
- At `max-width: 980px`, the sidebar link grid is hidden and a fixed bottom tab bar plus a More overflow menu handle navigation (UXA-001).
- `data-nav='sidebar-compact'` uses an 84px icon rail on desktop.
- Sets `data-theme`, `data-layout`, `data-nav`, and `data-preset` on `document.documentElement`.
- Updates the mobile `theme-color` meta tag.

## Where To Read The Behavior

| Layer | Location |
| --- | --- |
| Experience switcher UI | `apps/web/src/lib/experienceUi.tsx` |
| Timesheet rows | `apps/web/src/lib/timeEntryUi.tsx` |
| Reports and invoices | `apps/web/src/lib/reportUi.tsx`, `apps/web/src/lib/invoiceUi.tsx` |
| Shell navigation | `apps/web/src/features/shell/` |
| Theme buttons | `apps/web/src/lib/themeUi.tsx` |
| Experience state and root attributes | `apps/web/src/lib/experience.ts` |
| Preset catalog and SolidTime reference | [37-experience-presets.md](37-experience-presets.md) |
| App wiring | `apps/web/src/App.tsx` |
| CSS tokens | `apps/web/src/styles.css` (`[data-theme=...]`) |
| Visual reference | `docs/15-solidtime-theme.md` |

## Checks

```bash
make pre-commit
```

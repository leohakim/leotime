# Experience Presets

leotime ships six named experience presets plus a `custom` state when theme,
layout, or navigation diverge from a catalog combination.

## Catalog

| Preset | Theme | Layout | Nav | Best for |
| --- | --- | --- | --- | --- |
| `workbench-pro` | `solid` | `solid` | `sidebar` | Default daily work on desktop |
| `calm-light` | `light` | `minimal` | `sidebar` | Bright, low-chrome reading |
| `focus-dark` | `dark` | `solid` | `sidebar` | Deep contrast without compact density |
| `compact-power` | `dark` | `compact` | `sidebar-compact` | Dense desktop scanning |
| `mobile-flow` | `light` | `compact` | `bottom-tabs` | Tablet/mobile capture flows |
| `solidtime-exact` | `solid` | `solid` | `sidebar` | SolidTime reference baseline |

Definitions live in `apps/web/src/lib/experience.ts` as `EXPERIENCE_PRESET_DEFINITIONS`.

## SolidTime Exact reference

`solidtime-exact` mirrors the current SolidTime UI baseline and is versioned
separately from the other presets:

| Field | Value |
| --- | --- |
| Repo | https://github.com/solidtime-io/solidtime |
| Release | `v0.15.1` |
| Commit | `ab9f6e6` |
| Verified | 2026-07-08 |

When SolidTime ships a new UI baseline, refresh only this preset in a dedicated
sprint and update `SOLIDTIME_EXACT_REFERENCE` in `experience.ts`.

## Sprint 9 polish

- Login uses a hero + form split layout on desktop and stacks on narrow screens (UXA-009).
- `data-preset` adds light visual refinements per preset (radius, density, touch targets).
- `mobile-flow` enforces `44px` minimum targets on shell controls and timer capture.

## Verification

```bash
make pre-commit
```

Manual spot-check each preset at `1440px`, `834px`, and `390px` on dashboard,
timesheet, and settings. See [38-ui-ux-qa-checklist.md](38-ui-ux-qa-checklist.md)
for the full responsive, visual, and accessibility gates.

## Adding a new preset

1. Extend `EXPERIENCE_PRESET_DEFINITIONS` in `apps/web/src/lib/experience.ts`.
2. Add preset label/description keys to `apps/web/src/lib/i18n.ts`.
3. Optional: add `[data-preset="<id>"]` token overrides in `apps/web/src/styles.css`.
4. Add unit coverage in `experience.test.ts` and a navigation smoke in `App.test.tsx`
   when the preset changes layout or nav mode.
5. Document the row in the catalog table above and run `make pre-commit` plus the
   three-width manual spot-check from the QA checklist.

# Theme Selector

The authenticated app supports four color themes, persisted in `localStorage` under `leotime.theme`.

## Experience foundation

Sprint 2 applies the full experience contract to the root element while
preserving the current UI:

```html
<html data-theme="solid" data-layout="solid" data-nav="sidebar" data-preset="workbench-pro">
```

- `leotime.theme` and `leotime.layout` remain compatible with the current
  profile hydration flow.
- `leotime.nav` is local-only and currently has the single value `sidebar`.
- `leotime.preset` is local-only. `workbench-pro` maps to solid theme, solid
  layout, and sidebar navigation; every other combination is `custom`.
- Changing theme or layout independently changes the active preset to `custom`.

Sprint 3 will add visible selector controls and additional preset choices. No
navigation or preset field is sent to the profile API in this foundation slice.

## Themes

| Theme | Purpose |
| --- | --- |
| `solid` | Default Solidtime-like dark palette. |
| `light` | Light workspace with higher contrast text, white surfaces, and readable accents. |
| `dark` | Deeper neutral dark palette. |
| `minimal` | Muted, low-saturation dark palette. |

Layout density remains separate under `leotime.layout` (`solid`, `minimal`, `compact`).

## UI

- Toolbar control next to the layout switcher in the Time Tracker header.
- Sets `data-theme`, `data-layout`, `data-nav`, and `data-preset` on
  `document.documentElement`.
- Updates the mobile `theme-color` meta tag.

## Where To Read The Behavior

| Layer | Location |
| --- | --- |
| Theme switcher UI | `apps/web/src/lib/themeUi.tsx` |
| Experience state and root attributes | `apps/web/src/lib/experience.ts` |
| App wiring | `apps/web/src/App.tsx` |
| CSS tokens | `apps/web/src/styles.css` (`[data-theme=...]`) |
| Visual reference | `docs/15-solidtime-theme.md` |

## Checks

```bash
make pre-commit
```

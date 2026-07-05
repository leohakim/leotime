# Theme Selector

The authenticated app supports four color themes, persisted in `localStorage` under `leotime.theme`.

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
- Sets `data-theme` on `document.documentElement`.
- Updates the mobile `theme-color` meta tag.

## Where To Read The Behavior

| Layer | Location |
| --- | --- |
| Theme switcher UI | `apps/web/src/lib/themeUi.tsx` |
| App wiring | `apps/web/src/App.tsx` |
| CSS tokens | `apps/web/src/styles.css` (`[data-theme=...]`) |
| Visual reference | `docs/15-solidtime-theme.md` |

## Checks

```bash
make pre-commit
```

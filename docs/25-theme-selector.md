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
| Shell navigation | `apps/web/src/features/shell/` |
| Theme buttons | `apps/web/src/lib/themeUi.tsx` |
| Experience state and root attributes | `apps/web/src/lib/experience.ts` |
| App wiring | `apps/web/src/App.tsx` |
| CSS tokens | `apps/web/src/styles.css` (`[data-theme=...]`) |
| Visual reference | `docs/15-solidtime-theme.md` |

## Checks

```bash
make pre-commit
```

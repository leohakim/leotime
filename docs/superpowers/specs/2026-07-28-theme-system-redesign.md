# Theme system redesign

**Status:** draft — awaiting user review of this file  
**Date:** 2026-07-28  
**Branch:** work on the current feature branch (do not land on `main` without an explicit merge)

## Goal

Replace the near-identical Solidtime / Dark / Minimal skins and the light theme that leaks dark hardcodes with a real four-theme design system. Differences must be obvious in color, typography, radius, density, and accent.

## Decisions (from brainstorming)

| Decision | Choice |
|---|---|
| Approach | **A — token system**: full CSS variable contract per theme; components use tokens only |
| Solidtime | Clone-style: Inter, cyan accent `#5fb3d9`, charcoal UI, 8px controls (match Solidtime v0.15.x feel) |
| Light | Clean diurnal: Source Sans 3, white surfaces, blue accent `#2563eb`, no stray blacks |
| Dark | Neutral OLED: IBM Plex Sans, near-black, light-gray accent (no cyan), tighter radii |
| Minimal | Editorial hybrid: warm paper base, Source Serif 4 for titles + Source Sans 3 for controls, high contrast, more air, less chrome, square edges |
| Fonts | Distinct families per theme (OFL): Inter, Source Sans 3, IBM Plex Sans, Source Serif 4 |
| API | Keep `ThemeMode = 'solid' \| 'light' \| 'dark' \| 'minimal'` unchanged |

## Architecture

### Token contract

Every `[data-theme='…']` (and `:root` = `solid`) defines:

**Color:** `--bg`, `--sidebar`, `--surface`, `--surface-2`, `--surface-3`, `--input`, `--border`, `--border-strong`, `--text`, `--muted`, `--faint`, `--ink`, `--accent`, `--accent-soft`, `--blue` (alias or keep for legacy), `--green`, `--green-soft`, `--amber`, `--orange`, `--rose`, `--rose-soft`, `--shadow`

**Typography:** `--font-ui`, `--font-display` (Minimal uses display for headings; others may set both to the same family), `--text-xs`…`--text-display`, `--fw-*`

**Shape / density:** `--radius-control`, `--radius-panel`, `--control-height`, `--space-1`…`--space-4`

Semantic aliases already in use stay mapped: `--surface-canvas`, `--content-primary`, `--border-default`, `--status-*`, `--focus-ring`.

### Rules

1. Components and base rules use `var(--…)` only — no raw dark hex in shared selectors.
2. Define `--accent` (currently referenced but undefined).
3. Prefer deleting large `[data-theme='light']` component override blocks once tokens are correct.
4. `layoutMode` / `navigationMode` / experience presets stay orthogonal; themes must not require a specific layout.
5. `applyExperienceMetaColor` maps each theme to its canvas color.

### Theme tokens (target personality)

#### `solid` (Solidtime clone)

- Font: Inter
- Canvas `#0c0d10`, sidebar `#08090b`, surfaces charcoal ladder
- Accent cyan `#5fb3d9`
- Radius control `8px`
- Density: current Solidtime-like (baseline)

#### `light`

- Font: Source Sans 3
- Canvas cool gray-blue (`#eef2f7`), sidebar/surfaces white
- Accent blue `#2563eb`
- Radius slightly softer (`10–12px` panels)
- Density: comfortable, clear borders, soft shadow

#### `dark`

- Font: IBM Plex Sans
- Canvas near-black (`#07080a`), sidebar black
- Accent light gray / off-white for primary actions (not cyan)
- Radius tighter (`4–6px`)
- Density: compact, technical

#### `minimal`

- Display font: Source Serif 4; UI font: Source Sans 3
- Warm paper canvas (`#f7f4ef`), stone text (`#1c1917`)
- Accent = ink (monochrome primary buttons)
- Radius `0` (or near-zero)
- Density: more padding, less chrome (hairline borders, flatter panels)

## Implementation scope

### In scope

1. Full token blocks for all four themes in `apps/web/src/styles.css` (keep one file unless the theme block alone exceeds ~400 lines — then split to `themes.css` imported from the main stylesheet).
2. Font loading: self-host OFL files under `apps/web/public/fonts` and `@font-face` in CSS (Docker-friendly, no CDN dependency at runtime).
3. Sweep hardcoded dark colors in `styles.css` → tokens.
4. Shrink/remove redundant light-theme override lists.
5. Update `THEME_META_COLORS` in `experience.ts`.
6. Tests: theme attribute application + token presence smoke; keep ThemeSwitcher coverage; `make pre-commit`.

### Out of scope

- Renaming theme IDs or backend profile schema
- Redesigning layout modes / experience presets beyond theme tokens
- Adding a fifth theme or system/auto theme
- Pixel-perfect DOM clone of upstream Solidtime (feel + Inter + cyan + radii is the bar)

## Verification

- Switching themes in Settings / experience switcher changes font, accent, and surfaces immediately.
- Light theme: no black/charcoal panels or inputs left over.
- Dark ≠ Solidtime (no cyan primary; different font/radius).
- Minimal clearly editorial (paper + serif headings).
- `make pre-commit` passes.

## Reference mockups

Brainstorm companion screen: `.superpowers/brainstorm/theme-session/content/theme-identities.html` (local only; not shipped).

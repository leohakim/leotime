# Theme System Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans (or subagent-driven-development) to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the four `ThemeMode` values (`solid`, `light`, `dark`, `minimal`) visually distinct via a real CSS token system, self-hosted OFL fonts, and removal of dark hardcodes that break Light.

**Architecture:** Keep `data-theme` on `<html>`. Expand each theme block to a full token contract (`--accent`, fonts, radii, density). Point shared selectors at tokens. Bundle fonts with Vite (no CDN at runtime). Shrink redundant `[data-theme='light']` overrides after tokens work.

**Tech Stack:** React 19, Vite, Vitest, CSS custom properties, OFL fonts (Inter, Source Sans 3, IBM Plex Sans, Source Serif 4).

## Global Constraints

- Do not rename `ThemeMode` or change the profile API.
- Solidtime (`solid`) stays Inter + cyan `#5fb3d9` + charcoal + 8px controls.
- Light: Source Sans 3, blue `#2563eb`, white surfaces, no charcoal leftovers.
- Dark: IBM Plex Sans, OLED/near-black, gray/off-white accent (no cyan), tighter radii.
- Minimal: warm paper, Source Serif 4 headings + Source Sans 3 controls, monochrome accent, square edges, more air.
- No CDN font loads at runtime (Docker-friendly). Prefer `@fontsource*` packages imported so Vite bundles them.
- Do not commit unless the user explicitly asks.
- Finish with `make pre-commit`.

## File map

| File | Responsibility |
|---|---|
| `apps/web/package.json` | Add `@fontsource-variable/inter`, `@fontsource/source-sans-3`, `@fontsource/ibm-plex-sans`, `@fontsource/source-serif-4` |
| `apps/web/src/main.tsx` (or `styles.css`) | Import fontsource CSS once |
| `apps/web/src/styles.css` | Theme token blocks + replace hardcodes + trim light overrides |
| `apps/web/src/lib/experience.ts` | `THEME_META_COLORS` per theme canvas |
| `apps/web/src/lib/experience.test.ts` | Assert meta colors + theme attributes |
| `apps/web/src/lib/themeTokens.test.ts` | Optional smoke: document CSS vars after applying each theme (jsdom limited — prefer attribute + meta tests) |

---

### Task 1: Bundle OFL fonts

**Files:**
- Modify: `apps/web/package.json` / lockfile via npm
- Modify: `apps/web/src/main.tsx`

**Interfaces:**
- Produces font-family names usable in CSS: `InterVariable` or `Inter`, `Source Sans 3`, `IBM Plex Sans`, `Source Serif 4` (exact family names from the installed packages — verify in node_modules README after install).

- [ ] **Step 1: Install font packages**

```bash
npm --workspace @leotime/web install @fontsource-variable/inter @fontsource/source-sans-3 @fontsource/ibm-plex-sans @fontsource/source-serif-4
```

- [ ] **Step 2: Import fonts in `main.tsx` before app CSS**

```tsx
import '@fontsource-variable/inter/wght.css';
import '@fontsource/source-sans-3/400.css';
import '@fontsource/source-sans-3/500.css';
import '@fontsource/source-sans-3/600.css';
import '@fontsource/source-sans-3/700.css';
import '@fontsource/ibm-plex-sans/400.css';
import '@fontsource/ibm-plex-sans/500.css';
import '@fontsource/ibm-plex-sans/600.css';
import '@fontsource/source-serif-4/500.css';
import '@fontsource/source-serif-4/600.css';
import '@fontsource/source-serif-4/700.css';
```

Confirm the Inter variable family name (often `'Inter Variable'`). Use that exact string in theme tokens.

- [ ] **Step 3: Smoke the web build**

Run: `npm --workspace @leotime/web run build`  
Expected: PASS (fonts resolve).

---

### Task 2: Full theme token blocks + meta colors

**Files:**
- Modify: `apps/web/src/styles.css` (`:root` / `[data-theme='solid']`, `[data-theme='light']`, `[data-theme='dark']`, `[data-theme='minimal']`)
- Modify: `apps/web/src/lib/experience.ts` (`THEME_META_COLORS`)
- Modify: `apps/web/src/lib/experience.test.ts`

**Interfaces:**
- Every theme sets at least: `--font-ui`, `--font-display`, `--accent`, `--accent-soft`, `--radius-control`, `--radius-panel`, plus existing color tokens.
- `body` / `html` use `font-family: var(--font-ui)`.
- `h1,h2,h3,h4` use `font-family: var(--font-display)`.

- [ ] **Step 1: Extend experience tests for meta colors**

```ts
expect(THEME_META_COLORS or applyExperienceMetaColor effects):
  solid → '#0c0d10'
  light → '#eef2f7'
  dark → '#07080a'
  minimal → '#f7f4ef'
```

Export `THEME_META_COLORS` for testing or assert via `applyExperienceMetaColor` + a stubbed meta tag.

- [ ] **Step 2: Run test RED/GREEN as needed after updating `experience.ts`**

Run: `npm --workspace @leotime/web test -- --run src/lib/experience.test.ts`

- [ ] **Step 3: Rewrite theme token blocks in CSS**

`:root, [data-theme='solid']` — Inter, cyan accent `#5fb3d9`, charcoal ladder, `--radius-control: 8px`, `--radius-panel: 10px`, `--accent` = cyan, `--blue` = cyan (compat).

`[data-theme='light']` — Source Sans 3 for both fonts, canvas `#eef2f7`, white surfaces, accent `#2563eb`, `--radius-control: 10px`, `--radius-panel: 12px`, soft shadow.

`[data-theme='dark']` — IBM Plex Sans, canvas `#07080a`, sidebar `#000`, accent `#e5e7eb`, `--radius-control: 4px`, `--radius-panel: 6px`, no cyan (`--blue` can stay as a muted info blue distinct from primary accent, or map `--blue` to a cool gray-blue that is NOT the Solidtime cyan).

`[data-theme='minimal']` — `--font-display: 'Source Serif 4'`, `--font-ui: 'Source Sans 3'`, paper `#f7f4ef`, text `#1c1917`, accent = ink, radii `0`, larger `--space-*` and `--text-*` for headings.

Wire base rules:

```css
body {
  font-family: var(--font-ui);
}
h1, h2, h3, h4 {
  font-family: var(--font-display);
}
```

Primary buttons / focus where appropriate: prefer `var(--accent)` over hardcoded blues.

---

### Task 3: Replace shared hardcodes with tokens

**Files:**
- Modify: `apps/web/src/styles.css` (shared selectors only — not inside a single-theme override unless converting)

**Approach:**
1. Replace repeated dark surfaces `#111217`, `#0f1014`, `#121318`, `#17181e`, `#15161c`, `#0f1116` → `var(--surface)`, `var(--surface-2)`, `var(--input)`, `var(--bg)` as appropriate.
2. Replace border hexes `#1c1d22`, `#1b1c20`, `#24262e` → `var(--border)` / `var(--border-strong)`.
3. Leave intentional status colors that already have tokens (`--green`, `--rose`) alone; map near-duplicates to those tokens.
4. Do not leave charcoal backgrounds in selectors that also render under `light` / `minimal`.

- [ ] **Step 1: Script-assisted inventory**

```bash
rg -n "background: #1|background: #0|border: .*#1|border-.*: #1" apps/web/src/styles.css | head -80
```

- [ ] **Step 2: Replace shared hardcodes with tokens in batches**

Each batch: replace → rebuild/test web unit tests if touching layout critically.

- [ ] **Step 3: Delete obsolete light-only color force lists** that only restate `color: var(--text)` / `background: var(--surface)` after tokens work. Keep light-specific tweaks that still differ (e.g. calendar picker invert) only if still required.

---

### Task 4: Theme-specific chrome polish

**Files:**
- Modify: `apps/web/src/styles.css`

- [ ] **Step 1: Solid** — ensure start/primary actions read as Solidtime (accent cyan fill or cyan border), Inter metrics unchanged.

- [ ] **Step 2: Dark** — primary button uses light accent on dark; nav active without cyan glow.

- [ ] **Step 3: Minimal** — optional small rules: flatter panels (`box-shadow: none`), hairline borders, extra panel padding via `--space-*`, hide heavy chrome only where it does not break usability (do not remove sidebar structure — layout modes own that).

- [ ] **Step 4: Light** — verify sidebar/nav/timer/inputs are white/gray, not charcoal, by visual check under `make dev` or screenshot.

---

### Task 5: Verification gate

- [ ] **Step 1: Unit tests**

```bash
npm --workspace @leotime/web test -- --run src/lib/experience.test.ts src/lib/experienceUi.test.tsx
```

Expected: PASS.

- [ ] **Step 2: Full pre-commit**

```bash
make pre-commit
```

Expected: PASS.

- [ ] **Step 3: Manual checklist (dev)**

With `make dev`, switch themes in Settings:
1. Solidtime ≠ Dark (cyan vs gray accent; Inter vs Plex).
2. Light has no black panels.
3. Minimal shows serif headings on paper background.

---

## Spec coverage check

| Spec requirement | Task |
|---|---|
| Full token contract + `--accent` | Task 2 |
| Distinct fonts per theme | Tasks 1–2 |
| Self-hosted / no CDN runtime | Task 1 |
| Hardcode sweep | Task 3 |
| Shrink light overrides | Task 3 |
| Meta theme-color | Task 2 |
| Tests + pre-commit | Tasks 2, 5 |
| Solidtime clone feel | Tasks 2, 4 |
| Dark / Light / Minimal personalities | Tasks 2, 4 |

## Execution note

User said **avanza** after approving the spec — prefer inline execution of this plan in the same session unless they ask for subagents.

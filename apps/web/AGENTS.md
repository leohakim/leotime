# Web Agent Instructions

## Scope

This directory owns the React/Vite frontend, UI tests, E2E smoke tests, and client-side offline behavior.

## UI Rules

- The first screen should be the usable app or login, not a marketing page.
- Keep layout modes working: `solid`, `minimal`, and `compact`.
- Keep Spanish and English text routed through the i18n helper.
- Use icons for compact controls when appropriate.
- Avoid nested cards and unstable layout shifts.

## Checks

```bash
npm test -- --run
npm run build
npm run test:e2e
```


# Visual Regression

Playwright snapshot comparisons guard the login screen, timesheet, and dashboard
layouts across the three responsive baselines used in the UI audit.

## Commands

| Command | Purpose |
| --- | --- |
| `make audit-ui-regression` | Compare current UI against committed PNG baselines |
| `make audit-ui-regression-update` | Refresh baselines after an intentional visual change |
| `make audit-ui` | Capture human-readable JPEG evidence (no pixel diff) |

Snapshots live next to the spec:

```text
apps/web/e2e/visual-regression.spec.ts-snapshots/
```

## Deterministic seed time

Regression runs seed the API with a fixed clock so week labels and demo entries
stay stable:

| Variable | Value |
| --- | --- |
| `LEOTIME_SEED_NOW` | `2026-07-11T12:00:00Z` |
| Browser clock | same instant via Playwright `clock.install` |

Set `LEOTIME_SEED_NOW` manually when you need reproducible demo data outside the
Playwright harness:

```bash
LEOTIME_SEED_NOW=2026-07-11T12:00:00Z make seed
```

## When to update baselines

1. Make the visual change and verify it in the product UI.
2. Run `make audit-ui-regression-update`.
3. Review the PNG diff in git and commit the updated snapshots with the code change.
4. Optionally refresh JPEG evidence with `make audit-ui` for docs.

## Thresholds

`playwright.visual-regression.config.ts` allows up to **2%** differing pixels per
snapshot (`maxDiffPixelRatio: 0.02`) and disables CSS animations during capture.
Snapshot filenames omit the OS suffix so the same baselines are used in CI and
locally; if Linux rendering drifts from your dev machine, refresh baselines from
an environment that matches CI (Ubuntu) or tune the diff threshold.

## Related docs

- [UI/UX QA checklist](38-ui-ux-qa-checklist.md)
- [UI/UX visual audit](36-ui-ux-visual-audit.md)
- [Operations](10-operations.md)

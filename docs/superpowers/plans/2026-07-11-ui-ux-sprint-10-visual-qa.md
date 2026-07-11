# UI/UX Sprint 10 Visual QA & Documentation Implementation Plan

**Goal:** Close the experience-themes roadmap with shared feedback states, repeatable
visual/accessibility audits, and operator documentation (UXA-010).

**Approach:** `feedbackUi` primitives, audit Playwright specs, QA checklist, and
preset maintenance guide.

**Status:** Done (2026-07-11).

## Delivered

- `SurfaceLoading`, `SurfaceError`, and `SurfaceEmpty` in `feedbackUi.tsx` with shared
  `.surface-feedback-*` CSS; `QueryErrorBanner` aligned to the same error contract.
- Dashboard, reports, and invoices migrated to the shared feedback components.
- `e2e/accessibility-audit.spec.ts` added to the audit Playwright config.
- Visual audit login check updated for the Sprint 9 hero layout.
- `make audit-ui` runs visual + accessibility smoke via `npm run test:e2e:audit`.
- `docs/38-ui-ux-qa-checklist.md` documents responsive, visual, and a11y gates.
- `docs/37-experience-presets.md` extended with “Adding a preset” guidance.

## Files

| Area | Location |
| --- | --- |
| Feedback primitives | `apps/web/src/lib/feedbackUi.tsx` |
| Shared styles | `apps/web/src/styles.css` |
| Surface migrations | `dashboardUi.tsx`, `reportUi.tsx`, `invoiceUi.tsx`, `crudFormUi.tsx` |
| Visual audit | `apps/web/e2e/visual-audit.spec.ts` |
| Accessibility audit | `apps/web/e2e/accessibility-audit.spec.ts` |
| Audit config | `apps/web/playwright.audit.config.ts` |
| Makefile target | `audit-ui` |
| QA checklist | `docs/38-ui-ux-qa-checklist.md` |
| Tests | `feedbackUi.test.tsx`, audit specs |

## Verification

```bash
make pre-commit
make audit-ui   # optional; captures screenshots under docs/assets/ui-audit/
```

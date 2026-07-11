# UI/UX Sprint 8 Reports Invoices Implementation Plan

**Goal:** Clarify report preview and invoice draft/directory hierarchy (UXA-008) without backend changes.

**Approach:** Split both screens into filter/draft + results/directory workbenches; auto-load report preview on mount.

**Status:** Done (2026-07-11).

## Delivered

- `TimeReportPanel` uses `.report-workbench` with filters and preview panels; default month range loads on open.
- `InvoicePanel` uses `.invoice-workbench` with draft form and invoice directory side by side.
- Compact `.panel-empty-state` replaces tall empty regions in preview/directory panels.
- Responsive stacking below `1180px`.
- Integration tests updated; `make pre-commit` passes.

## Files

| Area | Location |
| --- | --- |
| Reports UI | `apps/web/src/lib/reportUi.tsx` |
| Invoices UI | `apps/web/src/lib/invoiceUi.tsx` |
| Styles | `apps/web/src/styles.css` |
| i18n | `apps/web/src/lib/i18n.ts` |
| Tests | `apps/web/src/App.test.tsx` |

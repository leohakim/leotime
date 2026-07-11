# Documentation Alignment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox syntax for tracking.

**Goal:** Make documentation describe delivered billing and import behavior
accurately, then publish an actionable hardening handoff.

**Architecture:** Separate current behavior, historical design, and future
hardening. The documentation index points to a capability matrix and one
curated backlog; each later behavior slice owns a focused implementation plan.

**Tech Stack:** Markdown, existing Go/SQLite/React implementation, Make gates.

## Global Constraints

- Do not commit real exports, PDFs, credentials, tokens, or client data.
- Describe current behavior before future design intent.
- Preserve the single-owner Docker-first product scope.
- Mark historical designs with their actual implementation state.
- Run make pre-commit before handoff.

---

## File Structure

| Path | Responsibility |
| --- | --- |
| docs/35-curated-hardening-backlog.md | Priorities and handoff for later behavior slices |
| docs/00-documentation-index.md | Discovery and current source-of-truth links |
| docs/01-product-vision.md | Product-level delivered capability statement |
| docs/02-architecture-go.md | Accurate CLI command naming |
| docs/09-solidtime-import.md | Current import contract and limitation |
| docs/12-implementation-plan.md | Current next-work link |
| docs/13-backlog.md | Phase ordering and backlog link |
| docs/23-invoices-api.md | Official route and legacy-status limitation |
| docs/31-s3-daily-backups.md | Current restore limitation |
| docs/32-billing-documents.md | Delivered versus planned billing behavior |
| docs/33-mvp-delivery-status.md | Accurate commands and migrations |
| docs/34-known-gaps-and-audit.md | Historical audit and successor link |
| docs/adr/README.md | ADR 0004 implementation state |
| README.md | Accurate navigation labels |

## Task 1: Publish the curated hardening backlog

**Files:**

- Create: docs/35-curated-hardening-backlog.md
- Modify: docs/00-documentation-index.md
- Modify: docs/13-backlog.md

- [ ] Record only verified P0, P1, and P2 work from current code.
- [ ] Give every slice dependencies, affected subsystem, acceptance tests,
      documentation responsibilities, and Make gates.
- [ ] Require a focused plan under docs/superpowers/plans before behavior
      changes.
- [ ] Verify all four P0 IDs appear with acceptance and gate sections:

      rg -n 'H-INV-01|H-DATA-02|H-IMP-03|H-BACKUP-04' docs/35-curated-hardening-backlog.md

## Task 2: Align shipped behavior and design status

**Files:**

- Modify: README.md, docs/00-documentation-index.md, docs/01-product-vision.md
- Modify: docs/02-architecture-go.md, docs/09-solidtime-import.md
- Modify: docs/12-implementation-plan.md, docs/13-backlog.md
- Modify: docs/23-invoices-api.md, docs/31-s3-daily-backups.md
- Modify: docs/32-billing-documents.md, docs/33-mvp-delivery-status.md
- Modify: docs/34-known-gaps-and-audit.md, docs/adr/README.md

- [ ] Mark ADR 0004 as partially implemented: series, snapshots, preview,
      issue, PDFs, document metadata/downloads, and archive-aware backup exist.
- [ ] Correct the CLI to leotime import solidtime and list migrations 000009,
      000010, and 000011.
- [ ] Record the legacy invoice issuance bypass, ZIP boundary, and document
      restore limitation as explicit backlog items.
- [ ] Preserve billing design history but stop labeling shipped code as planned.

## Task 3: Consistency audit and handoff verification

**Files:**

- Verify: all files above

- [ ] Search for superseded claims:

      rg -n 'Status: planned, not implemented|No migration beyond|Billing documents \(planned\)|ADR 0004.*\\*\\*No\\*\\*' README.md docs --glob '!docs/superpowers/plans/**'

- [ ] Inspect the diff:

      git diff --check
      git diff -- docs README.md

- [ ] Run the repository gate:

      make pre-commit

- [ ] Propose this commit message:

      docs: align shipped capabilities and curate hardening backlog

      Record the delivered billing and migration state, correct CLI references,
      document current hardening limits, and add an agent-ready P0/P1/P2 queue.

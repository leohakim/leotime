# UI/UX Sprint 4 Shell Navigation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refactor shell navigation into focused components and fix tablet/mobile navigation hierarchy (UXA-001) without breaking hash routes or access to timer, settings, language, offline status, or logout.

**Architecture:** Extract `SidebarNav`, `ShellTopbar`, `ShellSidebar`, and `MobileBottomNav` under `features/shell/`. Drive layout from `data-nav` (`sidebar`, `sidebar-compact`, `bottom-tabs`). At `max-width: 980px`, hide the sidebar link grid and show a fixed bottom tab bar with a More overflow menu.

**Tech Stack:** React 19, TypeScript, CSS custom properties, Vitest.

## Global Constraints

- Do not change hash routes or `appRoutes.ts` behavior.
- Keep timer command row, offline pill, experience switcher, language toggle, profile link, and logout reachable.
- Do not redesign feature screens (timesheet, calendar, etc.).
- Shell visual changes only; no backend/API changes.

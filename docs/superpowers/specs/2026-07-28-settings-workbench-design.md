# Settings workbench redesign

**Status:** approved  
**Date:** 2026-07-28  
**Branch:** `codex/vcs-gitea-client-context`

## Goal

Replace the long scrolling Settings page with a professional workbench: persistent section nav and one active panel at a time, with aligned forms across Account, Preferences, Notifications, Security, AI, Backups, and VCS.

## Layout

- Desktop: `settings-workbench` CSS grid `240px 1fr`; sticky vertical nav always visible.
- Mobile: horizontal section chips (existing pattern, extended).
- Only the active section panel is shown; `activeSection` state with optional deep-link via existing `focusSection` / section ids.
- Sections: account, preferences (`settings`), notifications, password, AI, backups, VCS.

## Forms and lists

- Shared `settings-panel` / `settings-card` chrome (heading + short subtitle + body).
- Consistent field grid, hints under inputs, actions row (secondary left / primary right).
- VCS: keep manage/probe actions; present create/edit in the panel card rather than stacking raw profile-forms.

## Out of scope

- New VCS providers, datepicker, shell-wide redesign, new color themes.

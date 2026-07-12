# AI daily summary enrichment (Cursor + git + time entries)

> **Status:** proposed — Phase 9  
> **Author:** product design 2026-07-12  
> **Depends on:** Phase 8 daily summary (`GET /api/v1/reports/daily-summary`, `#daily-summary` UI)

## Problem

The template-based daily summary is useful but reads like structured bullets turned into sentences. The owner's real Slack bitácora is narrative: it mentions meetings, blockers, pivots, and *what changed in the codebase*, not only tracked hours.

Goal: enrich the daily standup text using:

1. **Time entries** already in leotime (project, client, task, description, duration).
2. **Development activity** (git commits, files touched, PR titles).
3. **Cursor work context** (agent chats, searches, tasks run that day) when available.

Billing should optionally use the owner's **Cursor API key** and consume Cursor credits/token pricing — not leotime-hosted LLM infra.

## Constraints discovered

### What Cursor exposes today

| Source | Available? | Notes |
| --- | --- | --- |
| [Cursor SDK](https://cursor.com/docs/sdk/typescript) / Cloud Agents API | Yes | Run agents with `Agent.prompt` / `Agent.create` + `send`. Billed on Cursor account. |
| IDE chat / Composer history export API | **No** | No public endpoint to list today's chats or `@codebase` searches. |
| Local agent transcripts | Yes, **local files only** | `~/.cursor/projects/<workspace>/agent-transcripts/*.jsonl` on the developer machine. |
| Cloud agent runs launched via API | Partial | `GET /v1/agents`, conversation endpoints — only for agents *you* started through the API, not all IDE sessions. |

### leotime deployment model

- Docker-first VPS: **no access** to Mac paths, Cursor app data, or local git clones unless explicitly mounted or bridged.
- Single owner: OK to store encrypted third-party API keys (same pattern as S3 backup credentials).
- Privacy: transcripts may contain paths, snippets, and personal notes — must stay opt-in and never log raw content.

**Conclusion:** a **hybrid architecture** is required. The VPS can own time entries + optional GitHub commit fetch; **Cursor-rich context must be collected on the developer machine** (or via explicit repo mount + transcript path config).

## Recommended architecture (hybrid)

```text
┌─────────────────────────────────────────────────────────────────┐
│  leotime (VPS or local Docker)                                   │
│  • time entries for day                                          │
│  • template summary (Phase 8)                                    │
│  • optional GitHub commits (project → repo mapping + PAT)        │
│  • stores encrypted Cursor API key (optional, for cloud path)    │
└───────────────────────────┬─────────────────────────────────────┘
                            │ context bundle JSON
┌───────────────────────────▼─────────────────────────────────────┐
│  enricher (runs on Mac — CLI or localhost sidecar)               │
│  • git log --since for mapped local repo paths                   │
│  • parse Cursor agent-transcripts for matching workspace/day     │
│  • build prompt + call Cursor SDK Agent.prompt (user API key)    │
│  • return prose → UI textarea or POST /daily-summary/enriched      │
└─────────────────────────────────────────────────────────────────┘
```

### Data model additions (Phase 9a)

**Project workspace link** (new optional fields on `projects`):

| Field | Purpose |
| --- | --- |
| `localRepoPath` | Absolute path on dev machine (enricher only; not used server-side on VPS) |
| `gitRemoteUrl` | `https://github.com/org/repo` for server-side commit fetch |
| `cursorWorkspaceSlug` | Match `~/.cursor/projects/<slug>/` transcript folder |

**Profile / settings** (encrypted at rest, like S3 secrets):

| Field | Purpose |
| --- | --- |
| `cursorApiKey` | User API key from Cursor Dashboard → API Keys |
| `githubPat` | Optional; fetch commits when VPS has no local git |
| `gitAuthorEmail` | Filter commits to owner |
| `aiSummaryEnabled` | Master toggle |
| `aiSummaryModel` | e.g. `composer-2.5` (fast/cheap default) |

### Enrichment pipeline

1. **Collect facts** (deterministic, no LLM):
   - Time entries grouped by project/client.
   - Git: `git log --since=… --until=… --author=… --pretty=format:…` per linked repo.
   - Cursor transcripts: scan JSONL for `role=user` messages and `tool_use` names (`Grep`, `Read`, `Shell`) on the selected calendar day; extract user query first lines and top file paths (redacted/capped).

2. **Build prompt** (Spanish by default, matches profile locale):

```text
Eres mi asistente de bitácora diaria para Slack. Escribe UN párrafo fluido por bloque
(mañana/tarde/noche) en estilo equipo remoto, sin inventar reuniones ni personas.

Datos del día:
- Entradas de tiempo: …
- Commits: …
- Actividad Cursor (consultas y archivos): …

Formato exacto:
DD/M:
Resumen de hoy:
…
Hasta mañana team!
```

3. **Call Cursor** via SDK one-shot:

```typescript
await Agent.prompt(prompt, {
  apiKey: userKey,
  model: { id: "composer-2.5" },
  // local enricher only:
  local: { cwd: primaryRepoPath },
});
```

4. **Post-process**: user edits in UI, regenerates with feedback, saves draft, and **approves** when satisfied. Approved text is persisted per user/day and becomes the canonical record for that date.

### Approval workflow (Phase 9a — implemented)

| Step | API | UI |
| --- | --- | --- |
| Load saved state | `GET /api/v1/daily-summaries/{date}` | Opens draft or approved text for the date |
| Generate template | `POST /api/v1/daily-summaries/{date}/generate` | "Generar resumen" |
| Enrich locally | `GET …/enrich-context` + Mac `POST http://127.0.0.1:9333/enrich` + `POST …/enrich` | "Enriquecer con contexto" |
| Edit + feedback | `PUT /api/v1/daily-summaries/{date}` | Textarea + optional feedback for next regeneration |
| Approve | `POST /api/v1/daily-summaries/{date}/approve` | Locks record; stores `approved_text` |
| Reopen | `POST /api/v1/daily-summaries/{date}/reopen` | Returns to draft for further edits |

**Future — Work Protocols (Phase 10+):** approved daily summaries aggregate into `billing.WorkProtocolSnapshot` day rows (`date`, `hours`, narrative `items[]`) when generating invoice PDFs. Template-only drafts never feed billing; only approved records count.

### Delivery slices

| Slice | Scope | Runs where |
| --- | --- | --- |
| **9a** | Project → repo/workspace mapping UI + settings placeholders | VPS |
| **9b** | `leotime enrich-summary` CLI: git + transcript scrape + stdout prose | Mac |
| **9c** | Encrypted Cursor API key in settings; CLI reads from env or `leotime credentials` | Mac + VPS storage |
| **9d** | `POST /api/v1/reports/daily-summary/enrich` accepts context bundle; server calls Cursor Cloud API | VPS (no local git/transcripts) |
| **9e** | Localhost sidecar started by `make dev-enricher`; web UI button "Enriquecer con IA" | Mac dev loop |
| **9f** | GitHub-only server path (no Cursor transcripts) for VPS-only users | VPS |

**Recommended order:** 9a → 9b → 9c → 9e → 9d → 9f.

### Security rules

- Never commit API keys or transcript exports.
- Encrypt `cursorApiKey` and `githubPat` in SQLite (reuse backup crypto patterns).
- Cap transcript extraction: max N sessions, max M KB per day, strip `[REDACTED]` blocks.
- Log only: `enrichment_ok`, `enrichment_failed`, token usage if API returns it — not prompt body.
- User confirms before first enrichment: credits billed to Cursor account.

### Testing

- Unit: transcript parser fixtures (synthetic JSONL snippets).
- Unit: git log parser with fake repo in `t.TempDir()`.
- Integration: mock Cursor SDK HTTP; assert prompt contains time entry labels.
- Manual: compare output to `bitacora-diaria-osoigo.txt` tone check.

### Alternatives considered

| Approach | Pros | Cons |
| --- | --- | --- |
| **A. Hybrid (recommended)** | Full context on Mac; VPS still works for template summary | Two processes to run locally |
| **B. Server-only + GitHub API** | Simple VPS story | No Cursor IDE context |
| **C. Browser calls Cursor directly** | No sidecar | Exposes API key in frontend — rejected |
| **D. OpenAI/Anthropic BYOK in leotime** | Generic | User asked specifically for Cursor credits |

## Open product questions

1. **VPS-only workflow:** Is GitHub commits + time entries enough without transcripts, or is Mac enricher mandatory for you?
2. **Transcript scope:** All workspaces that day, or only workspaces linked to projects with time logged?
3. **Cost guard:** Daily cap (e.g. max 1 enrichment/day) or confirm dialog each time?

## Success criteria

- With 2+ time entries and a linked repo, enriched summary mentions at least one real commit or file from git/transcripts.
- Prose matches bitácora tone closer than template-only (subjective owner review).
- API key never appears in logs or API responses after save.
- `make pre-commit` green; enrichment disabled by default until configured.

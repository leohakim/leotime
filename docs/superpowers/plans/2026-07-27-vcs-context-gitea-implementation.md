1# Client-scoped Gitea VCS context Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (- [ ]) syntax for tracking.

**Goal:** Add encrypted, read-only Gitea connections and client-scoped repository context to the AI daily-summary workflow without mixing customer activity.

**Architecture:** Persist provider connections separately from repository links so a Gitea instance can be dedicated to one client or shared safely across clients. A Gitea adapter normalizes remote activity into a provider-neutral context, which the existing daily-summary context endpoint passes to the local enricher and prompt builder. The web app manages links in Settings and previews the exact capped facts before enrichment.

**Tech Stack:** Go 1.25, SQLite migrations, Chi HTTP API, React 19, TanStack Query, Vitest, Playwright, existing AES-GCM secret encryption.

## Global Constraints

- First active provider: Gitea, including configurable self-hosted Osoigo URLs.
- GitHub and GitLab remain future adapters behind the same Go interface; do not add their HTTP clients in this slice.
- Every VCS repository must resolve to one client, explicitly or through its connection default; project links are optional and must belong to that client.
- Tokens use the existing LEOTIME_SECRETS_KEY encryption path, are never returned or logged, and need only read permissions.
- Production connections use HTTPS and a validated LEOTIME_VCS_ALLOWED_HOSTS allowlist.
- Cap and normalize VCS facts. Never persist or send diffs, full source, raw provider payloads, or unbounded review text to Cursor.
- Keep Spanish and English UI text in apps/web/src/lib/i18n.ts.
- Preserve local gitRemoteUrl, localRepoPath, and Cursor transcript enrichment behavior.
- Use synthetic fixtures only. Do not contact Osoigo, GitHub, Gitea, or GitLab during automated tests.
- Do not commit unless the user explicitly asks.

---

### Task 1: Persist and validate VCS connections and repository links

**Files:**
- Create: apps/api/internal/db/migrations/000017_vcs_context_gitea.sql
- Create: apps/api/internal/store/vcs.go
- Create: apps/api/internal/store/vcs_test.go
- Modify: apps/api/internal/config/config.go
- Modify: apps/api/internal/config/config_test.go

**Interfaces:**
- Produces store.VCSConnection, store.VCSRepository, store.VCSConnectionInput, and store.VCSRepositoryInput.
- Produces ListVCSConnections, CreateVCSConnection, UpdateVCSConnection, DeleteVCSConnection, ListVCSRepositories, UpsertVCSRepository, and DeleteVCSRepository.
- Produces config.Config.VCSAllowedHosts parsed from LEOTIME_VCS_ALLOWED_HOSTS.
- Consumed by Tasks 2–4.

- [ ] **Step 1: Write the failing store tests**

Add tests that create Client A and Client B, a Gitea connection whose default client is A, and an inheriting repository:

~~~go
connection, err := st.CreateVCSConnection(ctx, user.ID, store.VCSConnectionInput{
    Provider: "gitea", BaseURL: "https://gitea.example.test",
    DefaultClientID: clientA.ID, OwnerIdentity: "leo",
}, "encrypted-token")
if err != nil { t.Fatal(err) }
repo, err := st.UpsertVCSRepository(ctx, user.ID, store.VCSRepositoryInput{
    ConnectionID: connection.ID, Owner: "osoigo", Name: "backend",
})
if err != nil || repo.EffectiveClientID != clientA.ID { t.Fatal("expected inherited client") }
~~~

Assert that a repository with neither explicit nor inherited client, and one linked to Client A plus Client B's project, return store.ErrInvalidVCSInput. Add config tests for invalid hosts and production allowlist rejection.

- [ ] **Step 2: Run the store tests to verify RED**

Run: cd apps/api && go test ./internal/store ./internal/config -run 'TestVCS|TestConfig'

Expected: FAIL because VCS types, tables, and config fields do not exist.

- [ ] **Step 3: Add schema, config, and minimal store implementation**

Create forward-only connection and repository tables. Connections contain provider, base URL, owner identity, encrypted token, default client, enabled state, and timestamps. Repositories contain connection, owner/name, optional explicit client/project, include-category toggles, enabled state, and a unique (connection, owner, name) key.

Normalize provider, owner/name, and base URL. Validate user ownership of connection, client, and project; require one effective client; require a linked project's client to equal the effective client. Response structs expose tokenConfigured, never tokenEnc.

Parse LEOTIME_VCS_ALLOWED_HOSTS as normalized comma-separated hosts. Require a normalized HTTP(S) URL; reject URL credentials, paths, queries, fragments, loopback/link-local addresses, and malformed hosts. Production requires HTTPS and an allowed host.

- [ ] **Step 4: Run store and config tests to verify GREEN**

Run: cd apps/api && go test ./internal/store ./internal/config -run 'TestVCS|TestConfig'

Expected: PASS; inherited/explicit links work, cross-client links fail, and bad hosts fail.

### Task 2: Implement the provider-neutral Gitea collector

**Files:**
- Create: apps/api/internal/vcs/types.go
- Create: apps/api/internal/vcs/gitea.go
- Create: apps/api/internal/vcs/gitea_test.go
- Create: apps/api/internal/vcs/service.go
- Modify: apps/api/internal/httpapi/router.go

**Interfaces:**
- Produces vcs.Provider with TestConnection(context.Context, Connection) error and CollectDayContext(context.Context, CollectionRequest) (Context, error).
- Produces vcs.Context containing capped Commit, PullRequest, Review, and Issue facts.
- httpapi.Server receives a vcs.Service with an injected HTTP client.
- Consumed by Tasks 3 and 4.

- [ ] **Step 1: Write failing adapter tests against httptest.Server**

Serve synthetic Gitea responses for a commit, pull request, review, and linked issue:

~~~go
context, err := provider.CollectDayContext(ctx, vcs.CollectionRequest{
    Date: "2026-07-27", OwnerIdentity: "leo",
    Repositories: []vcs.Repository{{Owner: "osoigo", Name: "backend"}},
})
if err != nil || context.Commits[0].Subject != "fix auth redirect" {
    t.Fatal("expected normalized commit")
}
if strings.Contains(context.PromptFacts(), "raw diff") {
    t.Fatal("must not expose raw provider payload")
}
~~~

Cover 401, 429, timeout, duplicate hashes, and response bodies larger than the read cap.

- [ ] **Step 2: Run adapter tests to verify RED**

Run: cd apps/api && go test ./internal/vcs -run TestGitea

Expected: FAIL because the package and adapter do not exist.

- [ ] **Step 3: Implement minimal normalized Gitea collection**

Implement a provider interface with token handling contained in the adapter, bounded response reads, fixed timeout, owner/date filtering, stable sort, identifier deduplication, and per-kind limits. Return typed errors for unauthorized, rate-limited, unavailable, and invalid responses.

Create vcs.Service to select only gitea today. It decrypts the token immediately before collection and receives an injectable client. Add it to httpapi.Server construction without changing the other router dependencies.

- [ ] **Step 4: Run focused VCS tests to verify GREEN**

Run: cd apps/api && go test ./internal/vcs

Expected: PASS; remote JSON becomes capped facts and no raw payload/token leaves the package.

### Task 3: Expose authenticated APIs and enrich daily-summary context

**Files:**
- Create: apps/api/internal/httpapi/vcs.go
- Create: apps/api/internal/httpapi/vcs_test.go
- Modify: apps/api/internal/httpapi/router.go
- Modify: apps/api/internal/httpapi/daily_summary_records.go
- Modify: apps/api/internal/httpapi/daily_summary_records_test.go
- Modify: apps/api/internal/enrich/context.go
- Modify: apps/api/internal/enrich/prompt.go
- Modify: apps/api/internal/enrich/cursor_test.go

**Interfaces:**
- Adds authenticated connection and repository CRUD plus POST /api/v1/vcs/connections/{id}/test.
- Extends dailySummaryEnrichContextResponse and enrich.ContextBundle with VCSContext and partial-context status.
- Consumed by Task 4.

- [ ] **Step 1: Write failing HTTP and prompt tests**

Create a connection with a token and assert API JSON contains provider, tokenConfigured, and defaultClientId, but never the token. With Client A and B repositories, request clientId=ClientA and assert only A facts enter the enrichment context.

~~~go
prompt := BuildCursorPrompt(ContextBundle{
    VCS: VCSContext{Commits: []VCSCommit{{Subject: "fix auth redirect"}}},
})
if !strings.Contains(prompt, "Actividad VCS verificada") {
    t.Fatal("missing VCS prompt facts")
}
~~~

- [ ] **Step 2: Run focused tests to verify RED**

Run: cd apps/api && go test ./internal/httpapi ./internal/enrich -run 'TestVCS|TestBuildCursorPrompt.*VCS'

Expected: FAIL because routes, response fields, and prompt formatting are missing.

- [ ] **Step 3: Implement handlers and enrichment integration**

Follow encryptSecret, decryptSecret, decodeJSONBody, writeValidationStoreError, and requireUser patterns. Add connection/repository handlers and connection test. Return a 400 secrets_key_missing when a token cannot be stored.

In getDailySummaryEnrichContext, select enabled links by requested client then optional project. Attach only normalized facts. Provider errors set structured partial state instead of breaking template generation, editing, or approval. Extend BuildCursorPrompt with translated, labelled VCS facts while preserving existing bullet structure.

- [ ] **Step 4: Run focused HTTP and enrich tests to verify GREEN**

Run: cd apps/api && go test ./internal/httpapi ./internal/enrich

Expected: PASS; credentials are masked, scope isolation holds, and collection degrades gracefully.

### Task 4: Add settings UI and daily-summary VCS preview

**Files:**
- Create: apps/web/src/lib/vcsSettingsUi.tsx
- Create: apps/web/src/lib/vcsSettingsUi.test.tsx
- Modify: apps/web/src/lib/api.ts
- Modify: apps/web/src/lib/dailySummaryUi.tsx
- Modify: apps/web/src/lib/dailySummaryUi.test.tsx
- Modify: apps/web/src/lib/i18n.ts
- Modify: apps/web/src/lib/settingsSectionNavUi.tsx
- Modify: apps/web/src/features/shell/DashboardShell.tsx
- Modify: apps/web/src/styles.css

**Interfaces:**
- Adds typed VCS API clients and DailySummaryEnrichContext.vcs.
- Adds VCSSettingsPanel({ t }) under Settings and an expandable VCS preview in the daily-summary workbench.
- Consumes Task 3 APIs.

- [ ] **Step 1: Write failing UI tests**

Mock connection responses and assert:

~~~tsx
render(<VCSSettingsPanel t={t} />)
expect(await screen.findByText("Conexiones VCS")).toBeInTheDocument()
await user.type(screen.getByLabelText("URL base de Gitea"), "https://gitea.osoigo.test")
await user.selectOptions(screen.getByLabelText("Cliente predeterminado"), "cli_osoigo")
await user.click(screen.getByRole("button", { name: "Guardar conexión" }))
~~~

Assert the daily-summary preview labels VCS facts as verified, shows partial/unavailable honestly, and renders neither token fields nor raw provider payloads.

- [ ] **Step 2: Run UI tests to verify RED**

Run: npm --workspace @leotime/web test -- vcsSettingsUi.test.tsx dailySummaryUi.test.tsx --run

Expected: FAIL because panel, API types, and preview do not exist.

- [ ] **Step 3: Implement focused settings and preview components**

Add API types for VCSConnection, VCSRepository, input forms, and summary facts. Build VCSSettingsPanel with Gitea URL, owner identity, masked token replacement, enabled switch, default-client selector, connection test feedback, repository owner/name, explicit client/project selectors, include-category switches, and delete confirmation.

Add Spanish/English messages, mount panel in Settings navigation/workbench, and render a compact mobile-stable expandable VCS preview in dailySummaryUi. Do not add a raw-data inspector.

- [ ] **Step 4: Run frontend tests and production build to verify GREEN**

Run: npm --workspace @leotime/web test -- --run && npm --workspace @leotime/web run build

Expected: PASS; both locales compile and the VCS flows remain stable.

### Task 5: Document operation and prove the complete flow

**Files:**
- Create: docs/41-vcs-context-providers.md
- Modify: .env.example
- Modify: docs/00-documentation-index.md
- Modify: docs/13-backlog.md
- Modify: docs/superpowers/specs/2026-07-27-vcs-context-gitea-design.md
- Create: apps/web/e2e/vcs-context.spec.ts

**Interfaces:**
- Documents LEOTIME_VCS_ALLOWED_HOSTS, minimum read-only Gitea scope, client/repository linking, privacy limits, rotation, and partial-context behavior.

- [ ] **Step 1: Write failing E2E scenario**

Use only synthetic data. Configure a synthetic Gitea endpoint, connect a Client A repository, request its summary, and assert the VCS preview omits Client B facts.

- [ ] **Step 2: Run E2E test to verify RED**

Run: npm --workspace @leotime/web run test:e2e -- vcs-context.spec.ts

Expected: FAIL until the API/UI flow exists.

- [ ] **Step 3: Complete docs and fixture setup**

Document Docker/VPS prerequisites, explicit host allowlist, read-only Gitea token creation, dedicated/shared service linking, rotation/disable, prompt limits, and partial context. Mark Phase 10 Doing only after the implementation begins; retain GitHub and GitLab as future adapters.

- [ ] **Step 4: Run delivery gates**

Run: make pre-commit

Run: make smoke

Run: make deploy-check

Run the focused E2E and visual regression suites after UI changes. Every command must exit 0; external-provider checks stay synthetic.

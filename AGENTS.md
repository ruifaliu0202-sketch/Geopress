# AGENTS.md

## Project Overview

Geopress is a multi-tenant content auto-publishing platform for individuals and companies.

Core product loop:

```text
login -> choose workspace -> maintain knowledge base and media accounts -> generate draft from keywords -> edit/review -> schedule publish -> execute publish job -> collect result
```

The project is currently a working product skeleton. Core business read/write paths are backed by PostgreSQL through handler-level persistence helpers; the next cleanup step is to split those paths into repository/service packages.

## Tech Stack

- Backend: Go + Gin.
- Frontend workspace console: React + MUI + Vite.
- Frontend platform admin: `react-admin` on React + MUI.
- Database target: PostgreSQL.
- AI/RAG target: PostgreSQL + `pgvector` for embeddings and semantic retrieval.
- Future queue/cache recommendation: Redis for worker queue buffering, locks, cache, and rate limits.

## Repository Layout

```text
backend/
  cmd/api/main.go                  API process entrypoint
  internal/app/server.go           Gin router and application wiring
  internal/config/config.go        Environment config
  internal/database/database.go    PostgreSQL connection and seed helpers
  internal/database/snapshot.go    PostgreSQL snapshot/read-write persistence helpers
  internal/database/system_config.go
                                      PostgreSQL-backed system settings and secrets helpers
  internal/database/media_account_login_session.go
                                      PostgreSQL-backed media account login session helpers
  internal/http/handler/           HTTP handlers
  internal/http/middleware/        Auth, tenant, CORS middleware
  internal/model/models.go         Domain models
  internal/systemconfig/           System configuration loaders/persistence adapters
  internal/web/                    Embedded frontend static asset server and build output
  migrations/                      PostgreSQL schema migrations

frontend/
  src/App.tsx                      Tenant workspace console
  src/admin/AdminConsole.tsx       Platform admin UI using react-admin
  src/admin/dataProvider.ts        react-admin dataProvider for admin APIs
  src/api.ts                       Workspace API client
  src/appTypes.ts                  Workspace app navigation/dialog types
  src/components/                  Shared workspace UI, data tables, AI Thinking, and tour components
  src/features/workspace/          Workspace views and workflow dialogs
  src/types.ts                     Shared frontend types
  src/theme.ts                     MUI theme
  src/utils/formatters.ts          Shared frontend formatting helpers
  vite.config.ts                   Vite config and API proxy

docs/
  architecture.md                  Architecture notes
  product-plan.md                  Product nodes and implementation plan

scripts/
  build-native.sh                  Builds frontend assets into backend/internal/web/dist and emits dist/geopress-api
  migrate.sh                       Migration runner
  xiaohongshu-browser-login.mjs    Playwright worker for Xiaohongshu QR login
  xiaohongshu-browser-publish.mjs  Playwright worker for Xiaohongshu browser publishing

deploy/
  backend.Dockerfile               Optional Docker backend image with Node, Chromium, Playwright, migrations
  frontend.Dockerfile              Optional Docker frontend static image
  nginx.conf                       Optional Docker Nginx frontend/API proxy
  .env.example                     Optional Docker Compose environment template
```

## Implemented Capabilities

- Demo login, password login, database-backed sessions, and open registration.
- First-run onboarding after registration: industry selection, multi-select writing tone, subscription selection, and optional subscription skip.
- Subscription plan model with `free` and `vip`; VIP currently has a 100 USD monthly AI token budget.
- AI token usage accounting for generation requests with per-event token/cost records and monthly user usage totals.
- Personal/company workspace selection.
- Workspace-scoped business data.
- Knowledge base list/create.
- Knowledge item list/create with many-to-many assignment to knowledge base packages.
- Platform-maintained knowledge base/item marketplace model for future knowledge product sales.
- Media platform list.
- Media platform definitions are platform-admin managed; the tenant workspace only binds tenant accounts.
- Tenant media account list/create.
- Xiaohongshu media account QR binding via server-managed Playwright persistent browser context.
- Xiaohongshu media account browser login sessions are persisted in PostgreSQL, not only process memory.
- Xiaohongshu QR login watcher state file for debugging scan confirmation and cookie/page state.
- Manual content create.
- Keyword-based draft generation through an `AIProvider` interface with mock and OpenAI-compatible providers.
- Workspace knowledge chunk retrieval, writing skill selection, structured draft validation, and generation request logging.
- VIP-only AI formatting for knowledge entries and generation keywords; formatting uses the configured AI provider with a mock fallback and replaces the edited field instead of appending.
- Shared AI Thinking drawer/overlay for generation and formatting traces. Formatting makes one backend request and uses a deterministic client trace timeline that defaults to 5 seconds across the configured nodes.
- Publish schedule create.
- Publish job list.
- Xiaohongshu publish preparation, manual publish confirmation, and run-now publish job execution through the publisher interface.
- Xiaohongshu browser publish success detection treats leaving the editor/settings screen after clicking publish as a submitted/published outcome.
- Platform admin authorization.
- Platform admin resource lists for users, workspaces, members, media platforms, tenant media accounts, platform knowledge bases, and platform knowledge items.
- Platform admin create/update for media platforms and platform knowledge marketplace resources.
- Platform admin AI provider configuration persisted through PostgreSQL system settings/secrets. Environment variables seed defaults and remain the fallback source.
- PostgreSQL health check via `DATABASE_URL`.
- PostgreSQL seed/save/read paths for demo workspace metadata and core business resources.
- PostgreSQL migrations for users, sessions, subscription plans, AI token usage, system settings/secrets, workspaces, knowledge bases/items, platform knowledge resources, media platforms/accounts, media account login sessions, contents/versions, generation requests, publish schedules/jobs/results, and audit logs.
- Tenant workspace frontend is split into app shell, workspace views, workflow dialogs, common components, data tables, and utility formatters instead of keeping all workflow code in `App.tsx`.
- Workspace console has an in-product onboarding tour with overlay, target highlighting, automatic page switching, Back/Next/Enter/ESC controls, and manual restart from the top bar. It teaches the full workflow: choose workspace, create knowledge base package, create guide item, connect Xiaohongshu, generate from keywords, create publish task, and confirm publish result.
- Native deployment builds the Vite frontend into `backend/internal/web/dist` and embeds those assets in the Go API binary. The same backend process serves `/api/*` and the SPA frontend, with unknown non-API routes falling back to `index.html`.
- Optional Docker Compose deployment remains available, but native single-binary deployment is the preferred deployment path for this skeleton.

## Demo Auth

Demo users:

- `demo@geopress.local`: platform admin, token `demo-token`.
- `growth@geopress.local`: normal user, token `growth-token`.

Demo passwords are `demo`. Registered users receive database-backed session tokens. In in-memory test mode, registered sessions are stored in the handler's local session map through a custom auth token resolver.

Protected workspace APIs require:

```text
Authorization: Bearer <token>
X-Workspace-ID: <workspace-id>
```

Admin APIs require a platform admin token.

## Backend API Shape

Main workspace APIs:

- `POST /api/auth/login`
- `POST /api/auth/register`
- `GET /api/me`
- `GET /api/workspaces`
- `GET /api/subscription-plans`
- `POST /api/onboarding/complete`
- `GET /api/overview`
- `GET|POST /api/knowledge-bases`
- `GET|POST /api/knowledge-items`
- `POST /api/knowledge-items/format`
- `POST /api/knowledge-items/assign-bases`
- `GET /api/media-platforms`
- `GET|POST /api/media-accounts`
- `POST /api/media-accounts/:accountId/browser-login/start`
- `POST /api/media-accounts/:accountId/browser-login/complete`
- `GET|POST /api/contents`
- `POST /api/contents/generate`
- `GET|POST /api/publish-schedules`
- `GET /api/publish-jobs`
- `POST /api/publish/prepare`
- `POST /api/publish-jobs/:jobId/run`
- `POST /api/publish-jobs/:jobId/confirm`

Platform admin APIs:

- `GET /api/admin/overview`
- `GET /api/admin/users`
- `PUT /api/admin/users/:userId/subscription`
- `GET /api/admin/workspaces`
- `GET /api/admin/workspace-members`
- `GET|POST /api/admin/platform-knowledge-bases`
- `PUT /api/admin/platform-knowledge-bases/:knowledgeBaseId`
- `GET|POST /api/admin/platform-knowledge-items`
- `PUT /api/admin/platform-knowledge-items/:knowledgeItemId`
- `GET|POST /api/admin/media-platforms`
- `PUT /api/admin/media-platforms/:platformId`
- `GET /api/admin/media-accounts`
- `GET|PUT /api/admin/ai-config`

## Local Development Commands

Backend:

```bash
cd backend
go mod tidy
go run ./cmd/api
```

Frontend:

```bash
cd frontend
npm install
npm run dev
```

Frontend tooling uses the system-installed Node.js 26 runtime. Run `npm run dev`, `npm run build`, and `npm run lint` with Node 26. In nvm shells, make sure Node 26 is active before running npm scripts; otherwise the npm wrapper may still invoke an older Node binary.

Validation:

```bash
cd backend
go test ./...

cd ../frontend
npm run build
npm run lint
```

Native build:

```bash
./scripts/build-native.sh
```

This runs the frontend build, copies `frontend/dist` into `backend/internal/web/dist`, and emits `dist/geopress-api`. The generated `dist/geopress-api` and `frontend/dist` are ignored build outputs; `backend/internal/web/dist` is the embedded asset source used by Go builds.

The backend defaults to `http://localhost:18080`. The frontend dev server defaults to `http://localhost:5173` and proxies `/api` to `http://localhost:18080`.

Relevant environment variables:

- `DATABASE_URL`: PostgreSQL connection string required by the API process.
- `AI_PROVIDER`: `mock` or `openai`.
- `OPENAI_API_KEY`: OpenAI-compatible provider key.
- `OPENAI_BASE_URL`: OpenAI-compatible API base URL.
- `OPENAI_MODEL`: generation model name.
- `AI_REQUEST_TIMEOUT_SECONDS`: generation request timeout.
- `GEOPRESS_NODE_BIN`: Node.js binary used by the Xiaohongshu Playwright worker.
- `GEOPRESS_CHROME_PATH`: Chromium/Chrome executable used by the Playwright worker.
- `GEOPRESS_XHS_BROWSER_LOGIN_SCRIPT`: override path for the Xiaohongshu browser login script.
- `GEOPRESS_BROWSER_HEADLESS`: set to `false` for visible local Playwright debugging.

Service startup rule for agents:

- If the user already has backend or frontend dev servers running, prefer those for validation and do not stop them.
- If an agent must start temporary validation services, use alternate free ports, report the ports while testing, and stop the processes before finishing the turn unless the user explicitly asks to keep them running.
- Do not leave background dev servers running as a side effect of validation.
- Native production runtime can run only `dist/geopress-api`; it does not need a separate frontend server. If Nginx or Caddy is used in front, it should reverse proxy all paths to the backend service for HTTPS/host handling rather than serving a separate static directory.

## Deployment Guidance

Preferred deployment is native single-binary:

```text
browser -> Nginx/Caddy optional HTTPS proxy -> geopress-api
                                      /api/* -> backend handlers
                                      /*     -> embedded frontend SPA
```

Native build and run:

```bash
./scripts/build-native.sh
DATABASE_URL='postgres://geo_app:<password>@localhost:5432/geo?sslmode=disable' ./scripts/migrate.sh
APP_ENV=production \
HTTP_ADDR=127.0.0.1:18080 \
FRONTEND_ORIGIN=https://your-domain.example \
DATABASE_URL='postgres://geo_app:<password>@localhost:5432/geo?sslmode=disable' \
AI_PROVIDER=mock \
./dist/geopress-api
```

For systemd deployments, keep the binary under `/opt/geopress/geopress-api`, runtime state under `/var/lib/geopress/runtime`, and environment in `/etc/geopress/geopress.env`. Set `GEOPRESS_PROJECT_ROOT=/var/lib/geopress` so Xiaohongshu browser profiles persist under `/var/lib/geopress/runtime/browser-profiles`.

If Xiaohongshu browser login or publishing is used outside Docker, the host must provide Node.js, frontend `node_modules` containing Playwright, Chromium/Chrome, and CJK fonts. Configure `GEOPRESS_NODE_BIN`, `GEOPRESS_CHROME_PATH`, and `GEOPRESS_BROWSER_HEADLESS` explicitly in production.

Optional Docker Compose deployment:

- `docker-compose.yml` now defaults to connecting containers to host PostgreSQL through `host.docker.internal:5432`.
- `api` and `migrate` include `extra_hosts: host.docker.internal:host-gateway` for Linux Docker.
- The local bundled PostgreSQL service is behind the `local-db` profile and should be treated as a development fallback.
- Docker runtime state is persisted in the `geopress-runtime` volume.

## Xiaohongshu Browser Login Notes

Xiaohongshu account binding should use a server-managed browser session with QR login.

Recommended flow:

```text
workspace user clicks bind/login -> backend starts browser login session -> backend opens Xiaohongshu QR login page -> frontend displays QR image -> user scans in Xiaohongshu app -> backend confirms session -> browser profile is saved -> media account becomes connected
```

Implementation direction:

- Keep the backend boundary as `start browser QR login` and `complete browser QR login`.
- Store the browser profile under `runtime/browser-profiles/{workspaceId}/{accountId}`.
- Use a Playwright persistent browser context, not a mock QR session.
- The backend starts a persistent Chromium context for the workspace/account profile, opens the Xiaohongshu login page, screenshots the visible login QR code, and returns that screenshot to the frontend.
- The complete step must re-open/read the same persistent browser profile and confirm the platform login state before marking the media account connected.
- Browser login session metadata is stored in `media_account_login_sessions` so the start/complete flow survives handler memory loss. Account credential metadata remains a compatibility fallback.
- Keep the QR login watcher browser alive while the user scans; do not close the browser immediately after taking the QR screenshot.
- The watcher writes `geopress-login-state.json` inside the account browser profile. Use it to debug scan confirmation state, current URL, visible page text, and cookie names.
- For local visual debugging, set `GEOPRESS_BROWSER_HEADLESS=false` before starting the backend so the Playwright browser window is visible.
- Do not model or persist dynamic Xiaohongshu web headers such as `x-s`, `x-s-common`, or `x-t`; those should be generated naturally by the platform page running inside the managed browser.

## Xiaohongshu Publish Notes

- Current Xiaohongshu publish execution is represented by `internal/integration/xiaohongshu.MockHumanPublisher`.
- The workspace console can prepare a Xiaohongshu post from a content item and a connected Xiaohongshu media account.
- Prepared posts contain title, body, hashtags, copy blocks, checklist, warnings, character count, and prepared time.
- `POST /api/publish/prepare` creates a manual publish job and optionally runs it immediately.
- `POST /api/publish-jobs/:jobId/run` executes the current publisher path and records the returned external URL when available.
- `POST /api/publish-jobs/:jobId/confirm` lets an operator paste a manually published URL and mark the job/content as published.
- The Playwright publish worker records `publishOutcome.leftEditor=true` when the page leaves the Xiaohongshu editor/settings surface after clicking publish. The backend treats that signal as successful submission/publish instead of leaving the job as manual pending.
- Later browser-based publish automation should reuse the saved Xiaohongshu browser profile instead of calling private web APIs directly.

## AI Generation Implementation Notes

- `internal/ai` defines the provider interface, runtime configuration, prompt contract, writing skill selection, mock provider, and OpenAI-compatible provider.
- Workspace generation retrieves scoped knowledge chunks, selects a system-controlled writing skill and publish format, builds a structured prompt, validates structured output, saves a draft, and records generation metadata.
- Generation is a configurable multi-stage pipeline: input analysis, knowledge retrieval, content plan, draft generation, quality check, optional rewrite rounds, and draft persistence.
- Pipeline settings can differ between free and VIP subscription tiers through the platform admin AI configuration.
- The tenant user supplies keywords, selected knowledge packages, and content type; system templates own the output contract and prompt boundaries.
- Generation responses include a trace drawer payload for the tenant UI.
- Formatting knowledge entries and generation keywords uses a dedicated formatter prompt boundary. The model is asked to return structured markdown that improves prompt clarity without fabricating facts or overriding system-selected content type/output rules.
- Formatting is gated by VIP subscription status. The frontend shows the VIP-marked formatting action and routes the formatter result back into the active content/keyword field as replacement text.
- AI token usage is persisted to `generation_requests` and `ai_token_usage_events`; user monthly token usage totals are updated after successful generation.
- Admin AI configuration is exposed in the platform admin and persisted to `system_settings`; provider API keys are stored in `system_secrets` and are not echoed back. On startup, environment values seed missing database configuration and persisted configuration becomes the runtime source.
- PostgreSQL persistence is used for generated contents, generation requests, knowledge resources, media accounts, contents, schedules, and publish jobs.

## System Configuration Notes

- Runtime AI configuration is stored as a typed JSON setting under `system_settings`.
- Provider secrets such as OpenAI-compatible API keys are stored separately in `system_secrets`.
- Environment variables remain useful for bootstrap and local fallback, but administrator changes should go through the platform admin UI and persist to the database.
- Use dedicated system configuration helpers under `internal/systemconfig` instead of scattering direct env reads through handlers.

## Registration And Subscription Notes

- Registration creates a global user, a personal workspace, and an owner membership.
- A newly registered user has `onboardingCompleted=false` and is routed to the workspace onboarding page before the normal console.
- Onboarding saves the selected industry and selected tones to the initial workspace.
- Onboarding subscription selection writes `users.subscription_plan_id`, `subscription_tier`, status, current period, and monthly AI token budget.
- The VIP plan is seeded as `price_cents=10000`, `currency=USD`, and `monthly_token_budget_cents=10000`.
- Skipping subscription selection keeps the user on the free plan with zero monthly AI token budget.

## Database Guidance

Use a dedicated application database user instead of the PostgreSQL superuser.

Recommended names:

- Database: `geo`
- Application role: `geo_app`

Recommended connection format:

```bash
DATABASE_URL='postgres://geo_app:<password>@localhost:5432/geo?sslmode=disable'
```

Run migrations:

```bash
DATABASE_URL='postgres://geo_app:<password>@localhost:5432/geo?sslmode=disable' ./scripts/migrate.sh
```

If `CREATE EXTENSION vector` requires elevated privileges, create the extension once with a PostgreSQL admin account, then run application migrations with the lower-privilege application role.

For Docker containers connecting to host PostgreSQL, the host PostgreSQL service must listen on the Docker gateway address and `pg_hba.conf` must allow the active Docker bridge subnet for `geo_app`. Common local subnets include `172.17.0.0/16` and Compose-created networks such as `172.18.0.0/16`.

## AI Implementation Plan

Use RAG + writing skill templates + structured output first. Do not start with fine-tuning.

Recommended flow:

```text
keywords -> retrieve relevant knowledge chunks -> select WritingSkill -> build prompt -> call AIProvider -> validate structured output -> save draft -> review/edit -> schedule publish
```

Implementation notes:

- Tenant knowledge should stay in the database and be retrieved at generation time.
- Do not send the entire knowledge base to the model.
- Add knowledge chunking and embeddings for `knowledge_items`.
- Store embeddings in PostgreSQL via `pgvector`.
- Retrieve TopK chunks by workspace, knowledge base, keywords, and content type.
- Add a backend `AIProvider` interface. Start with `mock`, then one real provider.
- Add `WritingSkill` records for article styles and output contracts.
- Use structured JSON output for title, summary, body, keywords, sections, used knowledge IDs, and warnings.
- Save every generation request with prompt version, skill version, provider, model, token/cost metadata, retrieved knowledge IDs, raw output, parsed output, and errors.
- AI output must become a `draft`; do not auto-publish model output.
- Fine-tuning is a later optimization for stable tone/format after enough approved examples exist. Do not use fine-tuning to store tenant-specific knowledge.

## Frontend Boundaries

- `frontend/src/App.tsx` is the tenant workspace console shell: authentication state, workspace fetch, active view routing, dialogs, AI Thinking overlay, and workspace tour wiring.
- Production frontend assets are built by Vite and embedded in the Go backend under `backend/internal/web/dist`; do not add a separate production frontend server unless explicitly changing deployment strategy.
- Tenant workspace feature code lives under `frontend/src/features/workspace/`.
- Shared workspace UI lives under `frontend/src/components/`, including `AIThinkingOverlay`, `aiThinkingModel`, `OnboardingTour`, `common`, and `dataTables`.
- `frontend/src/admin/AdminConsole.tsx` is the platform management backend.
- The registration page is a simple login/register card; successful registration routes to the onboarding workflow when `user.onboardingCompleted` is false.
- The onboarding workflow is three steps: industry, writing tone, subscription plan. The subscription step can be skipped.
- The workspace onboarding tour is separate from account onboarding. It is a reusable overlay/highlight teaching component that targets `data-tour-id` anchors and stores completion in localStorage.
- Keep tenant workflows out of the platform admin unless they are system/operator views.
- Keep global platform configuration, users, workspace/member inspection, channel definitions, and audit resources in the platform admin.
- Use MUI components and existing theme conventions.
- Use `react-admin` resources and `frontend/src/admin/dataProvider.ts` for admin CRUD/list behavior.

## Backend Boundaries

- Current handlers maintain in-memory snapshots loaded from PostgreSQL and write changes through database helper methods. This is a temporary structure for skeleton speed.
- `internal/web` owns embedded frontend static serving. Register API routes before `web.Register(router)` so `/api/*` remains backend-only and all other unknown routes can fall back to the SPA.
- Next persistence step should introduce `internal/repository` and `internal/service`.
- Repository methods must always receive workspace/tenant context for tenant-scoped resources.
- Admin-only operations must check platform-admin authorization.
- Do not let ordinary workspace users mutate global media platform definitions.

Recommended future backend layout:

```text
internal/repository
internal/service
internal/ai
internal/queue
internal/integration
```

## Security Notes

- Do not use the PostgreSQL `postgres` superuser as the application connection user.
- Do not commit real passwords, API keys, provider tokens, or media account credentials.
- Store media account credentials encrypted before persistence work starts.
- Keep AI provider keys in environment variables.
- Treat tenant knowledge base content as private data.

## Current Known Limitations

- Handler-level persistence should be refactored into repository/service layers.
- Login uses bcrypt password checks and session tokens, but full password reset, email verification, logout, and session revocation are not implemented yet.
- Payment collection is not implemented; subscription selection updates the plan and AI token budget directly.
- AI generation can use mock or OpenAI-compatible providers, but semantic embeddings/pgvector retrieval are still planned.
- Publish execution has a browser-based Xiaohongshu path, but robust cross-platform queueing, retries, and external result reconciliation are still planned.
- Admin delete operations are not implemented yet.
- Bundle size is larger after adding `react-admin`; code splitting can be added later.

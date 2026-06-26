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
  internal/integration/browserplatform/
                                      Shared browser login/publish adapters for article platforms
  internal/knowledge/              Knowledge asset extraction, chunking, and OCR boundary
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
  src/components/workflowModel.ts  Generic step workflow state helpers
  src/components/WorkflowDrawer.tsx
                                      Shared right-side workflow drawer
  src/components/assistant/        Pluggable floating AI assistant components and action registry
  src/components/layout/           Workspace shell layout components
  src/features/workspace/          Workspace views and workflow dialogs
  src/features/workspace/productPages.tsx
                                      Productized workspace pages for media matrix, campaigns, creators, skill packages, and compliance
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
  lib/geopress-browser-login.mjs   Shared QR browser login worker helper
  lib/geopress-browser-publish.mjs Shared browser article publishing worker helper
  xiaohongshu-browser-login.mjs    Playwright worker for Xiaohongshu QR login
  xiaohongshu-browser-publish.mjs  Playwright worker for Xiaohongshu browser publishing
  netease-browser-login.mjs        Playwright worker for NetEase/网易号 QR login
  netease-browser-publish.mjs      Playwright worker for NetEase/网易号 publishing
  toutiao-browser-login.mjs        Playwright worker for Toutiao/头条号 QR login
  toutiao-browser-publish.mjs      Playwright worker for Toutiao/头条号 publishing
  sohu-browser-phone-login.mjs     Playwright worker for Sohu/搜狐号 phone/SMS login
  sohu-browser-publish.mjs         Playwright worker for Sohu/搜狐号 publishing

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
- Knowledge base list/create, soft-delete to trash, restore, permanent delete, and 30-day expired trash purge.
- Workspace knowledge assets are the tenant content-asset entity. Assets can be created from text or uploaded files, assigned to one or more knowledge bases, removed from bases, moved to trash, restored, permanently deleted, and retried for extraction/chunking.
- Workspace knowledge assets persist original upload bytes in PostgreSQL `knowledge_assets.source_data` so failed or stale parsing can be retried without asking the user to upload again.
- Knowledge asset ingestion supports plain text, markdown, DOCX, legacy DOC detection, PDF, and images. Text/markdown/DOCX are extracted locally; legacy DOC is accepted but extraction currently fails without an external converter; PDF and image extraction require AI vision OCR.
- Knowledge asset parsing creates `knowledge_processing_tasks` and `knowledge_chunks`. Default chunking is deterministic code-based chunking; optional AI enhancement is a VIP-only asynchronous job that can refine chunk text/metadata after baseline extraction succeeds.
- Knowledge asset AI enhancement can be applied later to assets that were originally created without AI enhancement through `POST /api/knowledge-assets/:assetId/ai-enhancement`; it reuses the asynchronous enhancement task path instead of retrying failed parsing.
- Workspace knowledge asset list/detail UI uses Chinese labels and styled status chips for task states. The asset table operation column is intentionally narrow and only keeps the primary "查看知识片段" action plus conditional actions: "重试" only for failed assets, delete/trash actions, and `VIPFeatureButton`-backed "AI增强" when enhancement is available.
- `internal/ai` exposes an `OCRProvider` boundary. The current OCR strategy is the OpenAI-compatible vision/document path; the mock provider intentionally does not fake OCR. Image/PDF assets created by non-VIP users or without a configured OpenAI-compatible provider are persisted but marked failed during parsing.
- Platform-maintained knowledge base/item marketplace model for future knowledge product sales.
- Media platform list includes Xiaohongshu/小红书, NetEase/网易号, Toutiao/头条号, and Sohu/搜狐号 seed definitions.
- Media platform definitions are platform-admin managed; the tenant workspace only binds tenant accounts.
- Tenant media account list/create.
- Media account authorization is strategy-based. QR browser strategies use `qr_login`; interactive phone/SMS browser strategies use `phone_sms`; future platforms should add or select a strategy instead of forcing all login flows through the shared QR script.
- Xiaohongshu, NetEase, and Toutiao media account QR binding use server-managed Playwright persistent browser contexts.
- Sohu media account binding uses an interactive phone/SMS Playwright flow: switch to phone login, collect phone number in the right drawer, screenshot the graphical captcha, send the SMS code request, collect SMS code, check agreements, and confirm login state.
- Media account browser login sessions are persisted in PostgreSQL, not only process memory. Watchers write `geopress-login-state.json` and, for interactive flows, read `geopress-login-command.json` inside the account profile directory.
- Browser login watchers report structured state for QR waiting, phone/captcha/SMS steps, manual intervention, profile-in-use, action failure, expiration, and connected states.
- Manual content create.
- Keyword-based draft generation through an `AIProvider` interface with mock and OpenAI-compatible providers.
- Workspace knowledge chunk retrieval from ready, enabled, non-trashed assets, writing skill selection, structured draft validation, and generation request logging.
- VIP-only AI formatting remains available for generation keywords. Knowledge entry formatting has moved to the knowledge asset AI enhancement flow instead of the removed workspace `knowledge-items/format` endpoint.
- Shared workflow drawer/overlay for long-running server steps. `AIThinkingOverlay` is now an adapter over `WorkflowDrawer`; AI generation/formatting traces, QR login, and Sohu phone/SMS login all use the same step model and right-side drawer pattern.
- Publish schedule create.
- Publish job list.
- Xiaohongshu publish preparation, manual publish confirmation, and run-now publish job execution through the publisher interface.
- NetEase, Toutiao, and Sohu have browser article publish adapters and platform-specific Playwright publish scripts wired through `internal/integration/browserplatform`.
- Xiaohongshu browser publish success detection treats leaving the editor/settings screen after clicking publish as a submitted/published outcome.
- Platform admin authorization.
- Platform admin resource lists for users, workspaces, members, media platforms, tenant media accounts, platform knowledge bases, and platform knowledge items.
- Platform admin create/update for media platforms and platform knowledge marketplace resources.
- Platform admin AI provider configuration persisted through PostgreSQL system settings/secrets. Environment variables seed defaults and remain the fallback source.
- PostgreSQL health check via `DATABASE_URL`.
- PostgreSQL seed/save/read paths for demo workspace metadata and core business resources.
- PostgreSQL migrations for users, sessions, subscription plans, AI token usage, system settings/secrets, workspaces, knowledge bases, legacy knowledge items, knowledge assets/chunks/tasks, platform knowledge resources, media platforms/accounts, media account login sessions, contents/versions, generation requests, publish schedules/jobs/results, and audit logs.
- Tenant workspace frontend is split into app shell, workspace views, workflow dialogs, common components, data tables, and utility formatters instead of keeping all workflow code in `App.tsx`.
- Tenant workspace shell now uses a productized left/center/right layout: a collapsible desktop side menu, mobile drawer menu, center workspace content, right-side context rail, and a top shortcut area for workspace selection, refresh, guide restart, platform-admin entry, user identity, and logout.
- The workspace console has a preference-driven MUI product theme with shared surface tokens, subtle dimensional backgrounds, shadows, branded card/paper treatments, and selectable `sage`, `ocean`, and `plum` palettes persisted in localStorage.
- Shared frontend components include reusable highlighted and VIP feature buttons. `VIPFeatureButton` owns the VIP background raster asset, gold border, looping sweep highlight, reduced-motion handling, and selected pressed-shadow state.
- The previous inline workspace workbench has been replaced by a pluggable floating AI assistant surface. The default persona is a Corgi-themed assistant implemented as a replaceable component asset, with typed action callbacks for generating content, creating knowledge bases/assets, binding media accounts, creating schedules, replaying the onboarding guide, and refreshing workspace data.
- The floating assistant is intentionally contract-based: persona assets, actions, disabled-state fallbacks, and action handlers live behind typed descriptors so future IP avatars or assistant panels can be swapped without rewriting business workflows.
- Tenant workspace product pages are visible in the left navigation for media matrix, campaigns, creator collaboration, skill package marketplace, and brand compliance/reporting. These pages use the existing workspace API client and include list, create, action, loading, empty, and error states where supported by backend endpoints.
- The media matrix workspace page currently uses a mock-first tabbed layout for `总览`, `搜狐号`, `小红书`, `头条号`, and `网易号`. The overview tab shows all-platform aggregate metrics, platform status cards, pending actions, and recent publish metric backflow; each platform tab shows account assets, workflow health, snapshot entry points, content metrics, visible-field checklists, and staged roadmap notes. Backend sync and metric persistence are still planned.
- Installed creative skill packages can be selected from the content generation dialog through `skillPackageVersionId`, connecting marketplace purchase/install flows to the AI generation request path.
- Workspace tables and product pages include responsive hardening: horizontal table containers, stable metric cards, wrapping text, empty rows, mobile-friendly section actions, and overflow controls for long titles, IDs, URLs, and generated content.
- Workspace console has an in-product onboarding tour with overlay, target highlighting, automatic page switching, Back/Next/Enter/ESC controls, and manual restart from the top bar. It teaches the full workflow: choose workspace, create knowledge base package, create or upload guide assets, connect a media account, generate from keywords, create publish task, and confirm publish result.
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
- `POST /api/knowledge-bases/:baseId/trash`
- `POST /api/knowledge-bases/:baseId/restore`
- `DELETE /api/knowledge-bases/:baseId`
- `GET|POST /api/knowledge-assets`
- `GET /api/knowledge-assets/:assetId`
- `PUT /api/knowledge-assets/:assetId/bases`
- `POST /api/knowledge-assets/:assetId/trash`
- `POST /api/knowledge-assets/:assetId/restore`
- `POST /api/knowledge-assets/:assetId/retry`
- `POST /api/knowledge-assets/:assetId/ai-enhancement`
- `DELETE /api/knowledge-assets/:assetId`
- `GET /api/knowledge-assets/:assetId/chunks`
- `GET /api/knowledge-assets/:assetId/tasks`
- `GET /api/knowledge-trash`
- `POST /api/knowledge-trash/purge-expired`
- `GET /api/media-platforms`
- `GET|POST /api/media-accounts`
- `POST /api/media-accounts/:accountId/browser-login/start`
- `POST /api/media-accounts/:accountId/browser-login/complete`
- `POST /api/media-accounts/:accountId/auth/start`
- `GET /api/media-accounts/:accountId/auth/status`
- `POST /api/media-accounts/:accountId/auth/actions`
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
- `GEOPRESS_NODE_BIN`: Node.js binary used by browser Playwright workers.
- `GEOPRESS_CHROME_PATH`: Chromium/Chrome executable used by the Playwright worker.
- `GEOPRESS_XHS_BROWSER_LOGIN_SCRIPT`: override path for the Xiaohongshu browser login script.
- `GEOPRESS_XHS_BROWSER_PUBLISH_SCRIPT`: override path for the Xiaohongshu browser publish script.
- `GEOPRESS_NETEASE_LOGIN_URL`, `GEOPRESS_NETEASE_PUBLISH_URL`: override NetEase/网易号 browser login and publish URLs.
- `GEOPRESS_NETEASE_BROWSER_LOGIN_SCRIPT`, `GEOPRESS_NETEASE_BROWSER_PUBLISH_SCRIPT`: override NetEase/网易号 Playwright worker scripts.
- `GEOPRESS_TOUTIAO_LOGIN_URL`, `GEOPRESS_TOUTIAO_PUBLISH_URL`: override Toutiao/头条号 browser login and publish URLs. The default login URL is `https://mp.toutiao.com/auth/page/login/`.
- `GEOPRESS_TOUTIAO_BROWSER_LOGIN_SCRIPT`, `GEOPRESS_TOUTIAO_BROWSER_PUBLISH_SCRIPT`: override Toutiao/头条号 Playwright worker scripts.
- `GEOPRESS_SOHU_LOGIN_URL`, `GEOPRESS_SOHU_PUBLISH_URL`: override Sohu/搜狐号 browser login and publish URLs. The default phone login URL is `https://mp.sohu.com/mpfe/v4/login`.
- `GEOPRESS_SOHU_BROWSER_LOGIN_SCRIPT`, `GEOPRESS_SOHU_BROWSER_PUBLISH_SCRIPT`: override Sohu/搜狐号 Playwright worker scripts.
- `GEOPRESS_BROWSER_HEADLESS`: set to `false` for visible local Playwright debugging.
- `GEOPRESS_CHROMIUM_NO_SANDBOX`: set to `true` only when the runtime requires no-sandbox Chromium launch flags.

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

For systemd deployments, keep the binary under `/opt/geopress/geopress-api`, runtime state under `/var/lib/geopress/runtime`, and environment in `/etc/geopress/geopress.env`. Set `GEOPRESS_PROJECT_ROOT=/var/lib/geopress` so browser profiles persist under `/var/lib/geopress/runtime/browser-profiles`.

If browser login or publishing is used outside Docker, the host must provide Node.js, frontend `node_modules` containing Playwright, Chromium/Chrome, and CJK fonts. Configure `GEOPRESS_NODE_BIN`, `GEOPRESS_CHROME_PATH`, and `GEOPRESS_BROWSER_HEADLESS` explicitly in production.

Optional Docker Compose deployment:

- `docker-compose.yml` now defaults to connecting containers to host PostgreSQL through `host.docker.internal:5432`.
- `api` and `migrate` include `extra_hosts: host.docker.internal:host-gateway` for Linux Docker.
- The local bundled PostgreSQL service is behind the `local-db` profile and should be treated as a development fallback.
- Docker runtime state is persisted in the `geopress-runtime` volume.

## Browser Media Login Notes

Media account binding should use a server-managed browser session and a platform-specific authorization strategy.

QR login flow:

```text
workspace user clicks bind/login -> backend resolves QR browser strategy -> backend opens the platform login page -> frontend displays QR image in WorkflowDrawer -> user scans in the platform app -> backend confirms session -> browser profile is saved -> media account becomes connected
```

Implementation direction:

- Keep legacy QR endpoints as `start browser QR login` and `complete browser QR login` for QR-capable platforms.
- Use `/api/media-accounts/:accountId/auth/start`, `/auth/status`, and `/auth/actions` for strategy-based interactive flows such as Sohu phone/SMS login.
- Strategy resolution lives behind `mediaAuthStrategy`: `qr_browser` supports `qr_login`, and `phone_sms_browser` supports `phone_sms`.
- Store the browser profile under `runtime/browser-profiles/{workspaceId}/{accountId}`.
- Use a Playwright persistent browser context, not a mock login session.
- The backend starts a persistent Chromium context for the workspace/account profile, opens the platform login page, screenshots the visible QR/captcha element when needed, and returns that image data to the frontend.
- The complete step must re-open/read the same persistent browser profile and confirm the platform login state before marking the media account connected.
- Browser login session metadata is stored in `media_account_login_sessions` so the start/complete flow survives handler memory loss. Account credential metadata remains a compatibility fallback.
- Keep the QR login watcher browser alive while the user scans; do not close the browser immediately after taking the QR screenshot.
- Watchers write `geopress-login-state.json` inside the account browser profile. Use it to debug confirmation state, current URL, visible page text, cookie names, captcha screenshots, and action status.
- Interactive watchers read `geopress-login-command.json`; backend actions include a generated `commandId`, and returned state should echo `lastCommandId` so the API does not hand the frontend a stale pre-action state.
- Do not start a second Chromium process against the same persistent profile. If the script reports `profile_in_use`, reuse the active session state or close the owning browser process; do not blindly delete `SingletonLock`.
- For local visual debugging, set `GEOPRESS_BROWSER_HEADLESS=false` before starting the backend so the Playwright browser window is visible.
- If a platform triggers a slider or risk challenge, Playwright should report `manual_intervention_required`; do not try to bypass the challenge. The operator must complete it in the visible browser, then trigger `continue_check`. Production remote operation will need a browser visibility path such as VNC/noVNC before these challenges are ergonomic.

Platform-specific login notes:

- Xiaohongshu uses QR browser login and still must not model or persist dynamic web headers such as `x-s`, `x-s-common`, or `x-t`; those should be generated naturally by the platform page running inside the managed browser.
- NetEase/网易号 uses the shared QR browser login helper through `scripts/netease-browser-login.mjs`.
- Toutiao/头条号 uses the shared QR browser login helper through `scripts/toutiao-browser-login.mjs`; the default login URL is `https://mp.toutiao.com/auth/page/login/`.
- Sohu/搜狐号 uses `scripts/sohu-browser-phone-login.mjs`, not the QR helper. It switches to phone login, accepts `submit_phone`, returns `captchaScreenshotData`, accepts `submit_captcha`, requests SMS, accepts `submit_sms_code`, checks agreements, and then confirms login.

## Browser Media Publish Notes

- Xiaohongshu browser publish execution is represented by `internal/integration/xiaohongshu.BrowserLongArticlePublisher` and `scripts/xiaohongshu-browser-publish.mjs`.
- The workspace console can prepare a platform-specific publish package from a content item and a connected media account.
- Prepared posts contain title, body, hashtags or platform keywords where applicable, copy blocks, checklist, warnings, character count, and prepared time.
- `POST /api/publish/prepare` creates a manual publish job and optionally runs it immediately.
- `POST /api/publish-jobs/:jobId/run` executes the current publisher path and records the returned external URL when available.
- `POST /api/publish-jobs/:jobId/confirm` lets an operator paste a manually published URL and mark the job/content as published.
- The Playwright publish worker records `publishOutcome.leftEditor=true` when the page leaves the Xiaohongshu editor/settings surface after clicking publish. The backend treats that signal as successful submission/publish instead of leaving the job as manual pending.
- NetEase, Toutiao, and Sohu browser article publishing use `internal/integration/browserplatform.Publisher` plus `scripts/{platform}-browser-publish.mjs`; they should reuse the saved browser profile instead of calling private web APIs directly.
- Robust cross-platform queueing, retries, and result reconciliation are still planned; current browser publishing keeps the manual confirmation fallback.

## Media Matrix Product Notes

The media matrix is the workspace's media account asset and operations control surface, not only an account list. It should answer:

- Which platform accounts the workspace owns or operates.
- Whether each account can currently log in, publish, and refresh metrics.
- Which accounts need operator action.
- How the recent and scheduled publishing workload is distributed.
- How account-level and content-level metrics are changing over time.

Layout direction:

- Use platform tabs as the primary navigation: `总览`, `搜狐号`, `小红书`, `头条号`, `网易号`. Do not duplicate this with a separate top platform filter. Reserve secondary filters for status, account group, owner, date range, and metric freshness.
- `总览` should show all-platform aggregate metrics, platform cards, pending actions, and recent publish result backflow. It should be scan-first and should not expose every account field.
- Each platform tab should show platform-scoped aggregate metrics, account asset table, workflow status, account detail and snapshot entry points, content metric rows, visible-field checklist, and the platform's next implementation steps.
- Sohu/搜狐号 is the first chain to make concrete because it already has a phone/SMS browser login flow and browser publish adapter. Other platforms should reuse the same product shape and only customize the platform adapter details.

Media account metadata should be modeled in layers:

- Identity: platform, display name, external account ID or handle, homepage URL, avatar URL, introduction, verification status, and account type such as main account, sub-account, test account, creator account, or brand account.
- Ownership and access: workspace, owner, operations group, account group, ownership type such as owned, client-authorized, creator-operated, or agency-operated, login method, authorization status, and capability summary such as article publishing, image/text publishing, scheduling, and metric reading.
- Positioning: account persona, content positioning, target audience, categories, suitable content types, compliance notes, prohibited topics, and account objective.
- Operational state: health status, last login check time, last profile sync time, last metric sync time, recent publish time, next scheduled action, warning reason, and operator notes.
- Metric provenance: source type, captured time, freshness status, and whether the value came from manual entry, browser-assisted page reading, browser-context request, official API, or publish-result backflow.
- Platform-specific fields should live in typed `platformMetadata` or JSON metadata until the field becomes cross-platform. Examples include Sohu masked phone and backend IDs, Xiaohongshu profile/handle fields, Toutiao account benefits, and NetEase account identifiers.

Metrics should be snapshot-based:

- Store account metric snapshots for follower count, following count when visible, content count, total reads/views, likes, comments, favorites/shares when visible, engagement rate, captured time, and data source.
- Store content metric snapshots for each successfully published task: external content ID, external URL, publish job ID, read/view count, like count, comment count, share/favorite/click counts when visible, captured time, and data source.
- Do not require real-time metric collection. Scheduled refresh is enough for the first product version.
- Keep historical snapshots and let the service compute totals, trends, deltas, stale-data flags, and platform summaries. Do not only store the latest computed total because that loses trend and auditability.
- Use staged refresh windows for published content, such as 1 hour, 6 hours, 24 hours, 3 days, 7 days, 14 days, and 30 days after publish. After the stable window, stop frequent refresh unless an operator requests manual sync.
- Avoid crawling every historical work on every run. Refresh recently published tracked content, stale accounts, and operator-selected accounts first.

Collection strategy:

- Prefer official APIs only when the platform clearly provides the needed account and content metrics for the workspace's authorization scope.
- The baseline implementation should be browser-assisted and based on the same server-managed persistent Playwright profile used for login and publishing.
- Direct browser-context requests can be used inside a platform adapter when they are observed to be stable and share the logged-in browser state, but they should have a fallback to visible page reading or manual snapshot entry.
- Do not build the product around private reverse-engineered web APIs as the primary contract. Logged-in cookies may make some requests work, but private endpoints often depend on dynamic headers, signatures, CSRF tokens, risk controls, and page version changes.
- If a platform shows slider verification, risk review, or manual confirmation, the adapter should report `manual_intervention_required`; operators complete the challenge in the visible browser and then continue the sync.

Planned backend model alignment:

- `media_accounts`: account master data, authorization state, platform metadata, operations metadata, and capability summary.
- `media_account_metric_snapshots`: account-level metric snapshots and provenance.
- `content_metrics`: content-level metric snapshots tied to `publish_jobs` and external content IDs.
- `media_account_sync_jobs`: scheduled and manual sync attempts, status, error reason, capture source, and next retry time.
- `media_account_login_sessions`: browser login session state and persisted browser profile metadata.

## AI Generation Implementation Notes

- `internal/ai` defines the generation provider interface, OCR provider interface, runtime configuration, prompt contract, writing skill selection, mock provider, and OpenAI-compatible provider.
- `internal/knowledge` owns workspace knowledge asset extraction, OCR handoff, text cleanup, deterministic chunking, and chunk search text construction.
- Workspace generation retrieves scoped knowledge chunks, selects a system-controlled writing skill and publish format, builds a structured prompt, validates structured output, saves a draft, and records generation metadata.
- Generation is a configurable multi-stage pipeline: input analysis, knowledge retrieval, content plan, draft generation, quality check, optional rewrite rounds, and draft persistence.
- Pipeline settings can differ between free and VIP subscription tiers through the platform admin AI configuration.
- The tenant user supplies keywords, selected knowledge bases, and content type; system templates own the output contract and prompt boundaries.
- Generation responses include a trace drawer payload for the tenant UI.
- Formatting generation keywords uses a dedicated formatter prompt boundary. The model is asked to return structured markdown that improves prompt clarity without fabricating facts or overriding system-selected content type/output rules.
- Knowledge asset AI enhancement reuses the formatter capability asynchronously after baseline extraction/chunking. The default path always performs code-based chunking first; AI enhancement is an optional VIP feature and exposes progress/status through the asset and processing task records.
- AI vision OCR is represented by `ai.OCRProvider`. `NewOCRProvider` currently returns the OpenAI-compatible provider only when `AI_PROVIDER=openai`; otherwise OCR is unavailable. PDF/image OCR is gated by active VIP status and fails parsing for non-VIP users or unconfigured providers.
- AI token usage is persisted to `generation_requests` and `ai_token_usage_events`; user monthly token usage totals are updated after successful generation.
- Admin AI configuration is exposed in the platform admin and persisted to `system_settings`; provider API keys are stored in `system_secrets` and are not echoed back. On startup, environment values seed missing database configuration and persisted configuration becomes the runtime source.
- PostgreSQL persistence is used for generated contents, generation requests, knowledge bases, knowledge assets, knowledge chunks, knowledge processing tasks, media accounts, contents, schedules, and publish jobs.

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

Tenant knowledge assets use `knowledge_assets`, `knowledge_asset_bases`, `knowledge_chunks`, and `knowledge_processing_tasks`. The legacy workspace `knowledge_items` tables remain in early migrations and compatibility/migration helpers, while new workspace product behavior should use the asset/chunk model. Platform-admin marketplace resources continue to use `platform_knowledge_bases` and `platform_knowledge_items`.

For Docker containers connecting to host PostgreSQL, the host PostgreSQL service must listen on the Docker gateway address and `pg_hba.conf` must allow the active Docker bridge subnet for `geo_app`. Common local subnets include `172.17.0.0/16` and Compose-created networks such as `172.18.0.0/16`.

## AI Implementation Plan

Use RAG + writing skill templates + structured output first. Do not start with fine-tuning.

Recommended flow:

```text
keywords -> selected knowledge bases -> retrieve relevant knowledge asset chunks -> select WritingSkill -> build prompt -> call AIProvider -> validate structured output -> save draft -> review/edit -> schedule publish
```

Implementation notes:

- Tenant knowledge should stay in the database and be retrieved at generation time.
- Do not send the entire knowledge base or full knowledge asset to the model.
- Workspace tenant content should use `knowledge_assets` as the durable asset entity and `knowledge_chunks` as the retrieval entity. `knowledge_items` is legacy compatibility/migration data for the workspace path and should not be extended for new tenant workflows.
- Knowledge bases classify assets and constrain retrieval scope. Assets may belong to multiple bases; generation should retrieve TopK chunks by workspace, selected knowledge bases, keywords, content type, enabled state, ready status, and non-trashed asset state.
- Store chunk embeddings in PostgreSQL via `pgvector` as semantic retrieval is expanded. The schema includes vector support, but retrieval behavior should remain correct with lexical scoring until embedding generation/ranking is fully wired.
- Keep the backend `AIProvider` interface for generation and the separate `OCRProvider` interface for document/image text extraction.
- Add `WritingSkill` records for article styles and output contracts.
- Use structured JSON output for title, summary, body, keywords, sections, used knowledge IDs, and warnings.
- Save every generation request with prompt version, skill version, provider, model, token/cost metadata, retrieved knowledge IDs, raw output, parsed output, and errors.
- AI output must become a `draft`; do not auto-publish model output.
- Fine-tuning is a later optimization for stable tone/format after enough approved examples exist. Do not use fine-tuning to store tenant-specific knowledge.

## Frontend Boundaries

- `frontend/src/App.tsx` is the tenant workspace console orchestration layer: authentication state, workspace fetch, active view routing, dialogs, AI Thinking overlay, floating assistant action wiring, and workspace tour wiring.
- Production frontend assets are built by Vite and embedded in the Go backend under `backend/internal/web/dist`; do not add a separate production frontend server unless explicitly changing deployment strategy.
- Tenant workspace feature code lives under `frontend/src/features/workspace/`.
- The workspace layout shell lives under `frontend/src/components/layout/`. Keep layout concerns such as side navigation, top shortcut placement, center content, and right context rail there instead of spreading shell structure across feature pages.
- Shared workspace UI lives under `frontend/src/components/`, including `AIThinkingOverlay`, `aiThinkingModel`, `workflowModel`, `WorkflowDrawer`, `OnboardingTour`, `common`, `dataTables`, `assistant`, and `layout`.
- `WorkflowDrawer` is the generic right-side step-progress surface for long-running server work. `AIThinkingOverlay` should stay a thin adapter over it, and media login flows should render QR, captcha, phone, and SMS controls inside this drawer instead of blocking the page with a large modal.
- Floating AI assistant code lives under `frontend/src/components/assistant/`. Keep it pluggable: use typed action descriptors, action callback maps, and replaceable persona assets; do not let assistant presentation components call workspace APIs directly.
- `frontend/src/components/common.tsx` owns shared product primitives such as `ProductSurface`, `HighlightedActionButton`, `VIPFeatureButton`, `MetricCard`, `Section`, `InfoRow`, dialog wrappers, and select helpers. Prefer extending these primitives before adding page-local visual duplicates.
- `frontend/src/components/surfaceStyles.ts` owns reusable selected-surface treatments such as `selectedSurfaceSx`. Use it for selectable knowledge base/package cards instead of stacking page-local selected CSS.
- `frontend/src/theme.ts` owns product-level MUI theme tokens, dimensional shadows, component overrides, shared surface colors, the `ThemePreference` type, and `createAppTheme`. Theme preference is changed from the personal settings entry in the top shortcut area and persisted in localStorage.
- Workspace knowledge UI must use knowledge bases as classification/retrieval scopes and knowledge assets as content items. Knowledge base cards support selection styling, remove/trash controls, and drag-to-trash with a bottom overlay and confirmation. The selected base scopes the asset list.
- `KnowledgeAssetsTable` owns the workspace asset list behaviors: selection, open detail, remove from selected base, batch remove, move to trash, retry parsing/chunking, view chunks, apply AI enhancement, and progress/status chips for extraction and AI enhancement.
- Knowledge asset operation columns should remain compact: remove redundant "view asset detail" and "view prompt" actions from the table, keep "查看知识片段" as the primary inspect action, show "重试" only for failed assets, and use `VIPFeatureButton` for AI enhancement instead of reimplementing the VIP style.
- Trash and personal settings belong in the top shortcut area, not the left navigation. Trash supports knowledge bases and assets, restore, permanent delete, and expired item purge messaging.
- `VIPFeatureButton` is the standard component for VIP-only feature actions such as knowledge asset AI enhancement. It should keep the `frontend/src/assets/vip-gold.png` background, gold border, looping sweep highlight, hover tooltip support, disabled state, and selected pressed-shadow state. Do not reimplement this style page-locally.
- `frontend/src/features/workspace/productPages.tsx` owns productized tenant workspace pages for media account matrix, campaigns, creator collaboration, skill package marketplace, and brand compliance/reporting. Keep tenant operator workflows here unless a feature clearly belongs to the platform admin.
- Media account binding UI lives in workspace dialogs. QR and phone/SMS login should share the same workflow drawer model, while account creation, platform selection, and start buttons can remain in the modal.
- `frontend/src/admin/AdminConsole.tsx` is the platform management backend.
- The registration page is a simple login/register card; successful registration routes to the onboarding workflow when `user.onboardingCompleted` is false.
- The onboarding workflow is three steps: industry, writing tone, subscription plan. The subscription step can be skipped.
- The workspace onboarding tour is separate from account onboarding. It is a reusable overlay/highlight teaching component that targets `data-tour-id` anchors and stores completion in localStorage.
- Keep tenant workflows out of the platform admin unless they are system/operator views.
- Keep global platform configuration, users, workspace/member inspection, channel definitions, and audit resources in the platform admin.
- Use MUI components and existing theme conventions.
- Build new product UI as reusable, typed, replaceable components. Prefer slots, typed registries, small adapters, and explicit callbacks for extension points; avoid one-off duplicated card/button/surface styling.
- Keep data loading and mutations in feature containers or existing API clients. Presentational components should receive view models, state, and callbacks rather than importing workspace API functions directly.
- Use generated raster assets only when the task truly needs bitmap visuals. For future assistant/IP assets, place final project-bound images under `frontend/src/assets/` and reference them from the persona contract; do not leave referenced assets in temporary image-generation directories.
- Use `react-admin` resources and `frontend/src/admin/dataProvider.ts` for admin CRUD/list behavior.

## Backend Boundaries

- Current handlers maintain in-memory snapshots loaded from PostgreSQL and write changes through database helper methods. This is a temporary structure for skeleton speed.
- Workspace tenant knowledge writes should target knowledge bases, knowledge assets, knowledge chunks, and knowledge processing tasks. Do not add new workspace behavior to the legacy `knowledge_items` route/model path.
- `internal/knowledge` owns file type detection, extraction, OCR handoff, text cleanup, deterministic chunking, and chunk search text construction. HTTP handlers should orchestrate persistence, authorization, subscription gating, and task status updates around that package instead of duplicating parsing logic.
- Knowledge asset upload must persist the original source bytes before processing so retry can rebuild chunks from `source_data`. Retry should clear old chunks, create a processing task, rerun extraction/chunking, and only require restore first if the asset is trashed.
- Knowledge base and knowledge asset trash is soft-delete first with a 30-day recovery window. Permanent delete APIs should be explicit; expired purge should remove only expired trashed resources.
- AI enhancement for knowledge assets should remain asynchronous behind the handler-managed worker/queue until a dedicated queue package is introduced. Baseline extraction and deterministic chunking must be available without AI enhancement.
- Knowledge asset retry and AI enhancement are separate operations. Retry is a failed parsing/chunking recovery path and should only be offered when the asset status is failed; AI enhancement is a later opt-in refinement path for assets that have usable baseline chunks but were not enhanced when created.
- OCR for image/PDF assets should go through `ai.OCRProvider` via the `internal/knowledge` OCR adapter. The current concrete OCR path is OpenAI-compatible AI vision/document extraction and is gated by active VIP subscription.
- Media account authentication should go through `mediaAuthStrategy` instead of platform-specific conditionals in handlers. Add new authorization styles by implementing a strategy and declaring platform `authorizationMethods`/capability modes.
- QR-capable platforms may use the shared `scripts/lib/geopress-browser-login.mjs` helper. Non-QR platforms should own a platform-specific interactive script that reports structured state and supported actions through `browserplatform.InteractiveLoginState`.
- Browser profile locking must be handled as an account/session concern. Reuse an active profile state when possible, and only clean profile locks after confirming no Chromium process owns the profile.
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
- Knowledge chunks and pgvector schema exist, but full embedding generation, vector ranking, and hybrid semantic retrieval are still incomplete.
- PDF/image OCR only has the OpenAI-compatible AI vision/document strategy. The mock provider does not fake OCR, and non-VIP or unconfigured OCR uploads will persist the asset but fail parsing.
- OCR token/cost metadata is stored on knowledge asset metadata when available, but OCR usage is not yet folded into `ai_token_usage_events` and monthly token accounting.
- Knowledge asset AI enhancement uses an in-process async worker/queue; Redis or another durable queue is still recommended for production-grade retries, worker recovery, and rate limits.
- Publish execution has browser-based paths for Xiaohongshu plus article platform adapters for NetEase, Toutiao, and Sohu, but robust cross-platform queueing, retries, and external result reconciliation are still planned.
- Sohu phone/SMS login can detect slider or risk-control interruptions, but fully remote manual resolution still needs a visible browser delivery mechanism such as VNC/noVNC in production.
- Admin delete operations are not implemented yet.
- Bundle size is larger after adding `react-admin`; code splitting can be added later.

# AGENTS.md

## Project Overview

Geopress is a multi-tenant content auto-publishing platform for individuals and companies.

Core product loop:

```text
login -> choose workspace -> maintain knowledge base and media accounts -> generate draft from keywords -> edit/review -> schedule publish -> execute publish job -> collect result
```

The project is currently a working skeleton. Business data is still in memory, while PostgreSQL migrations are already present for the planned persistent schema.

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
  internal/database/database.go    Optional PostgreSQL connection
  internal/http/handler/           HTTP handlers
  internal/http/middleware/        Auth, tenant, CORS middleware
  internal/model/models.go         Domain models
  migrations/                      PostgreSQL schema migrations

frontend/
  src/App.tsx                      Tenant workspace console
  src/admin/AdminConsole.tsx       Platform admin UI using react-admin
  src/admin/dataProvider.ts        react-admin dataProvider for admin APIs
  src/api.ts                       Workspace API client
  src/types.ts                     Shared frontend types
  src/theme.ts                     MUI theme
  vite.config.ts                   Vite config and API proxy

docs/
  architecture.md                  Architecture notes
  product-plan.md                  Product nodes and implementation plan

scripts/
  migrate.sh                       Migration runner
```

## Implemented Capabilities

- Demo login.
- Personal/company workspace selection.
- Workspace-scoped business data.
- Knowledge base list/create.
- Knowledge item list/create.
- Media platform list.
- Tenant media account list/create.
- Manual content create.
- Mock keyword-based content generation.
- Publish schedule create.
- Publish job list.
- Platform admin authorization.
- Platform admin resource lists for users, workspaces, members, media platforms, and tenant media accounts.
- Platform admin create media platform.
- Optional PostgreSQL health check via `DATABASE_URL`.
- PostgreSQL migrations for users, workspaces, knowledge bases/items, media platforms/accounts, contents/versions, generation requests, publish schedules/jobs/results, and audit logs.

## Demo Auth

Demo users:

- `demo@geopress.local`: platform admin, token `demo-token`.
- `growth@geopress.local`: normal user, token `growth-token`.

Passwords are ignored in the current in-memory demo login.

Protected workspace APIs require:

```text
Authorization: Bearer <token>
X-Workspace-ID: <workspace-id>
```

Admin APIs require a platform admin token.

## Backend API Shape

Main workspace APIs:

- `POST /api/auth/login`
- `GET /api/me`
- `GET /api/workspaces`
- `GET /api/overview`
- `GET|POST /api/knowledge-bases`
- `GET|POST /api/knowledge-items`
- `GET /api/media-platforms`
- `GET|POST /api/media-accounts`
- `GET|POST /api/contents`
- `POST /api/contents/generate`
- `GET|POST /api/publish-schedules`
- `GET /api/publish-jobs`

Platform admin APIs:

- `GET /api/admin/overview`
- `GET /api/admin/users`
- `GET /api/admin/workspaces`
- `GET /api/admin/workspace-members`
- `GET|POST /api/admin/media-platforms`
- `GET /api/admin/media-accounts`

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

Validation:

```bash
cd backend
go test ./...

cd ../frontend
npm run build
npm run lint
```

The frontend dev server defaults to `http://localhost:5173` and proxies `/api` to `http://localhost:8080`.

## Database Guidance

Use a dedicated application database user instead of the PostgreSQL superuser.

Recommended names:

- Database: `geopress`
- Application role: `geopress_app`

Recommended connection format:

```bash
DATABASE_URL='postgres://geopress_app:<password>@localhost:5432/geopress?sslmode=disable'
```

Run migrations:

```bash
DATABASE_URL='postgres://geopress_app:<password>@localhost:5432/geopress?sslmode=disable' ./scripts/migrate.sh
```

If `CREATE EXTENSION vector` requires elevated privileges, create the extension once with a PostgreSQL admin account, then run application migrations with the lower-privilege application role.

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

- `frontend/src/App.tsx` is the tenant workspace console.
- `frontend/src/admin/AdminConsole.tsx` is the platform management backend.
- Keep tenant workflows out of the platform admin unless they are system/operator views.
- Keep global platform configuration, users, workspace/member inspection, channel definitions, and audit resources in the platform admin.
- Use MUI components and existing theme conventions.
- Use `react-admin` resources and `frontend/src/admin/dataProvider.ts` for admin CRUD/list behavior.

## Backend Boundaries

- Current handlers keep in-memory slices for skeleton speed.
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

- Business read/write paths still use in-memory data.
- Login uses demo tokens and no password hashing.
- AI generation is mock-only.
- Publish execution is mock-only.
- Admin create exists for media platforms, but update/delete are not implemented yet.
- Bundle size is larger after adding `react-admin`; code splitting can be added later.

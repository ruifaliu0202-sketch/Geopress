# Geopress

多租户内容自动发布平台骨架，后端使用 Go + Gin，前端使用 React + MUI。

## 目录

```text
backend/   Gin API 服务
frontend/  React + MUI 控制台
docs/      架构和业务说明
```

## 本地启动

当前骨架需要本机安装 Go、Node.js 和 npm。

### 数据库

后端业务链路已接 PostgreSQL。启动 API 前需要先运行迁移，迁移会写入 Demo 用户、工作区、知识库、内容、排程和小红书媒体平台示例数据。

使用 Docker Compose 启动项目数据库：

```bash
docker compose up -d postgres
./scripts/migrate.sh
```

也可以使用已有 PostgreSQL：

```bash
DATABASE_URL='postgres://user:password@localhost:5432/geopress?sslmode=disable' ./scripts/migrate.sh
```

后端需要可用的 `DATABASE_URL`。数据库连接失败、迁移缺失或初始化数据写入失败时，API 会启动失败。

### 后端

```bash
cd backend
go mod tidy
go run ./cmd/api
```

默认监听 `http://localhost:18080`。

### 前端

```bash
cd frontend
npm install
npm run dev
```

默认监听 `http://localhost:5173`，并通过 Vite proxy 转发 `/api` 到 `http://localhost:18080`。如需改代理目标，可设置 `VITE_API_PROXY_TARGET`。

## 当前能力

- 登录/注册：支持开放注册；Demo 账号 `demo@geopress.local` 或 `growth@geopress.local`，密码 `demo`。
- 工作区：工作区是租户空间，支持个人/公司工作区切换；用户通过成员关系加入工作区，业务数据按工作区隔离。
- 平台后台：已接入开源 `react-admin`，平台管理员可查看用户、工作区、成员、小红书媒体平台和租户媒体账号资源。
- 知识库：支持知识库和知识条目维护，作为后续 AI 生成上下文。
- 媒体平台和账号：当前仅保留小红书媒体平台，支持租户小红书账号绑定。
- 内容：支持手动创建内容，以及关键词 mock AI 生成草稿。
- 发布计划：支持创建一次性/周期性计划，并生成发布任务。
- 数据库：已提供 PostgreSQL + pgvector 迁移结构和示例数据，核心业务读写从 PostgreSQL 加载并持久化。

## 管理后台选型

平台管理后台使用 `react-admin`，它和当前 React + MUI 技术栈匹配，适合快速搭建面向平台运营的资源管理界面。租户工作台继续保留为自研业务界面，承载知识库、内容生成、媒体账号绑定和发布计划等高频业务流程。

## 下一步建议

- 将当前 handler 内的数据库访问继续拆分为 service/repository。
- 完善邀请员工、角色权限、退出登录和会话管理。
- 完善平台后台的媒体渠道编辑、字段 schema、渠道启停、渠道适配器配置和账号资源审计。
- 增加内容编辑、版本、审核流和状态流转。
- 加入后台 worker、任务队列、重试和幂等发布。
- 接入真实 AI Provider 和第一个真实媒体平台。

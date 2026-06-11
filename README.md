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

阶段 2 已加入 PostgreSQL schema 和迁移脚本。当前业务接口仍以内存数据运行，后续会逐步切到 repository。

使用 Docker Compose 启动项目数据库：

```bash
docker compose up -d postgres
./scripts/migrate.sh
```

也可以使用已有 PostgreSQL：

```bash
DATABASE_URL='postgres://user:password@localhost:5432/geopress?sslmode=disable' ./scripts/migrate.sh
```

未配置 `DATABASE_URL` 时，后端会继续以内存模式启动，`/api/healthz` 会返回 `database: "not_configured"`。

### 后端

```bash
cd backend
go mod tidy
go run ./cmd/api
```

默认监听 `http://localhost:8080`。

### 前端

```bash
cd frontend
npm install
npm run dev
```

默认监听 `http://localhost:5173`，并通过 Vite proxy 转发 `/api` 到后端。

## 当前能力

- 登录：内存版 Demo 登录，账号 `demo@geopress.local` 或 `growth@geopress.local`，任意密码。
- 工作区：支持个人/公司工作区切换，业务数据按工作区隔离。
- 平台后台：已接入开源 `react-admin`，平台管理员可查看用户、工作区、成员、全局媒体渠道和租户媒体账号资源，并可新增全局媒体渠道。
- 知识库：支持知识库和知识条目维护，作为后续 AI 生成上下文。
- 媒体平台和账号：支持平台能力展示和租户媒体账号绑定。
- 内容：支持手动创建内容，以及关键词 mock AI 生成草稿。
- 发布计划：支持创建一次性/周期性计划，并生成发布任务。
- 数据库：已提供 PostgreSQL + pgvector 迁移结构，后端健康检查会报告数据库连接状态。

## 管理后台选型

平台管理后台使用 `react-admin`，它和当前 React + MUI 技术栈匹配，适合快速搭建面向平台运营的资源管理界面。租户工作台继续保留为自研业务界面，承载知识库、内容生成、媒体账号绑定和发布计划等高频业务流程。

## 下一步建议

- 将内存 handler 拆分为 service/repository，并逐步切到 PostgreSQL。
- 增加真实用户注册、密码哈希、角色和权限校验。
- 完善平台后台的媒体渠道编辑、字段 schema、渠道启停、渠道适配器配置和账号资源审计。
- 增加内容编辑、版本、审核流和状态流转。
- 加入后台 worker、任务队列、重试和幂等发布。
- 接入真实 AI Provider 和第一个真实媒体平台。

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

使用本机 PostgreSQL：

```bash
DATABASE_URL='postgres://geo_app:<password>@localhost:5432/geo?sslmode=disable' ./scripts/migrate.sh
```

也可以使用 Docker Compose 内置 PostgreSQL：

```bash
docker compose --profile local-db up -d postgres
DATABASE_URL='postgres://geo_app:geo_app_dev_password@localhost:5432/geo?sslmode=disable' ./scripts/migrate.sh
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

## 原生环境部署

推荐的非 Docker 部署方式是构建一个单体 Go 服务：前端 `frontend/dist` 会被复制到 `backend/internal/web/dist` 并内嵌进后端 binary。运行时同一个进程同时提供前端页面和 `/api`。

```text
browser
  -> geopress-api
    -> /api/*        后端接口
    -> /*            内嵌 React/Vite 静态资源，SPA fallback 到 index.html
```

### 1. 构建单体服务

需要本机已有 Node.js 26、npm、Go 和 PostgreSQL 客户端。

```bash
./scripts/build-native.sh
```

产物：

```text
dist/geopress-api
```

脚本会执行：

```text
frontend npm run build
复制 frontend/dist -> backend/internal/web/dist
go build -> dist/geopress-api
```

### 2. 准备数据库

使用本机 PostgreSQL 时，建议数据库名为 `geo`，应用用户为 `geo_app`：

```bash
sudo -u postgres psql
```

```sql
CREATE DATABASE geo;
CREATE USER geo_app WITH PASSWORD 'geo_app_dev_password';

\c geo

CREATE EXTENSION IF NOT EXISTS vector;
GRANT CONNECT ON DATABASE geo TO geo_app;
GRANT USAGE, CREATE ON SCHEMA public TO geo_app;

ALTER DEFAULT PRIVILEGES IN SCHEMA public
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO geo_app;

ALTER DEFAULT PRIVILEGES IN SCHEMA public
GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO geo_app;

\q
```

运行迁移：

```bash
DATABASE_URL='postgres://geo_app:geo_app_dev_password@localhost:5432/geo?sslmode=disable' ./scripts/migrate.sh
```

### 3. 直接运行

```bash
APP_ENV=production \
HTTP_ADDR=127.0.0.1:18080 \
FRONTEND_ORIGIN=http://localhost:18080 \
DATABASE_URL='postgres://geo_app:geo_app_dev_password@localhost:5432/geo?sslmode=disable' \
AI_PROVIDER=mock \
./dist/geopress-api
```

访问：

```text
http://localhost:18080
```

健康检查：

```bash
curl http://localhost:18080/api/healthz
```

### 4. systemd 托管

如果要使用小红书扫码登录和浏览器发布，宿主机还需要安装 Node.js、Playwright 依赖和 Chromium：

```bash
cd frontend
npm ci
sudo apt-get install -y chromium fonts-noto-cjk fonts-noto-color-emoji
```

创建运行用户和目录：

```bash
sudo useradd --system --create-home --shell /usr/sbin/nologin geopress
sudo mkdir -p /opt/geopress /var/lib/geopress/runtime /etc/geopress
sudo cp dist/geopress-api /opt/geopress/geopress-api
sudo mkdir -p /opt/geopress/backend
sudo cp -R scripts /opt/geopress/
sudo cp -R backend/migrations /opt/geopress/backend/
sudo chown -R geopress:geopress /opt/geopress /var/lib/geopress
```

环境文件 `/etc/geopress/geopress.env`：

```text
APP_ENV=production
HTTP_ADDR=127.0.0.1:18080
FRONTEND_ORIGIN=https://your-domain.example
DATABASE_URL=postgres://geo_app:geo_app_dev_password@localhost:5432/geo?sslmode=disable
AI_PROVIDER=mock
OPENAI_API_KEY=
OPENAI_BASE_URL=https://api.openai.com/v1
OPENAI_MODEL=gpt-5.5
AI_REQUEST_TIMEOUT_SECONDS=45
GEOPRESS_PROJECT_ROOT=/var/lib/geopress
GEOPRESS_NODE_BIN=/usr/bin/node
GEOPRESS_CHROME_PATH=/usr/bin/chromium
GEOPRESS_BROWSER_HEADLESS=true
```

systemd unit `/etc/systemd/system/geopress.service`：

```ini
[Unit]
Description=Geopress API
After=network.target postgresql.service

[Service]
User=geopress
Group=geopress
WorkingDirectory=/var/lib/geopress
EnvironmentFile=/etc/geopress/geopress.env
ExecStart=/opt/geopress/geopress-api
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
```

启动：

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now geopress
sudo systemctl status geopress
```

### 5. 可选 Nginx HTTPS 反代

因为后端已经直接服务前端资源，Nginx 只需要做 HTTPS 终止和反向代理：

```nginx
server {
    listen 80;
    server_name your-domain.example;

    location / {
        proxy_pass http://127.0.0.1:18080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

生产环境建议用 Caddy 或 certbot 给域名加 HTTPS。

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

## Docker Compose 部署

仓库提供了一套面向 MVP/小规模生产的 Compose 部署骨架：

```text
Internet
  -> web: Nginx 静态前端 + /api 反向代理
    -> api: Go API + Playwright/Chromium
      -> PostgreSQL + pgvector
```

部署文件位于：

```text
docker-compose.yml
deploy/
  backend.Dockerfile
  frontend.Dockerfile
  nginx.conf
  .env.example
```

### 1. 准备环境变量

```bash
cp deploy/.env.example .env
```

默认部署配置连接宿主机 PostgreSQL 的 `5432` 端口，数据库名为 `geo`，应用用户为 `geo_app`：

```bash
DATABASE_URL='postgres://geo_app:<password>@host.docker.internal:5432/geo?sslmode=disable'
FRONTEND_ORIGIN='https://your-domain.example'
AI_PROVIDER='openai'
OPENAI_API_KEY='<provider-key>'
OPENAI_BASE_URL='https://api.openai.com/v1'
OPENAI_MODEL='<model>'
```

如果改用托管 PostgreSQL，把 `DATABASE_URL` 的 host、密码和 `sslmode` 改成云数据库要求的值。

### 2. 创建本机 PostgreSQL 数据库

在宿主机上创建 `geo` 数据库和专用应用用户：

```bash
sudo -u postgres psql
```

进入 `psql` 后执行：

```sql
CREATE DATABASE geo;

CREATE USER geo_app WITH PASSWORD 'geo_app_dev_password';

\c geo

CREATE EXTENSION IF NOT EXISTS vector;

GRANT CONNECT ON DATABASE geo TO geo_app;
GRANT USAGE, CREATE ON SCHEMA public TO geo_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO geo_app;
GRANT USAGE, SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA public TO geo_app;

ALTER DEFAULT PRIVILEGES IN SCHEMA public
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO geo_app;

ALTER DEFAULT PRIVILEGES IN SCHEMA public
GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO geo_app;
```

退出：

```sql
\q
```

如果 `CREATE EXTENSION vector` 报错，需要先安装 PostgreSQL 的 pgvector 扩展包，或在云数据库控制台启用 `pgvector`。

Docker 容器访问宿主机 PostgreSQL 使用 `host.docker.internal`。Compose 已经给 `api` 和 `migrate` 配置了：

```yaml
extra_hosts:
  - "host.docker.internal:host-gateway"
```

确认宿主机 PostgreSQL 监听 Docker 网关地址，并在 `pg_hba.conf` 允许 Docker 网段用密码连接。常见本机开发配置是允许 `172.17.0.0/16`：

```text
host    geo     geo_app     172.17.0.0/16     scram-sha-256
```

然后重启 PostgreSQL。

### 3. 启动应用

使用本机 PostgreSQL：

```bash
docker compose run --rm migrate
docker compose up -d api web
```

如果临时改用 Compose 内置 PostgreSQL：

```bash
docker compose --profile local-db up -d postgres
DATABASE_URL='postgres://geo_app:geo_app_dev_password@postgres:5432/geo?sslmode=disable' docker compose --profile local-db run --rm migrate
DATABASE_URL='postgres://geo_app:geo_app_dev_password@postgres:5432/geo?sslmode=disable' docker compose --profile local-db up -d api web
```

### 4. 发版更新

每次拉取新代码后执行：

```bash
docker compose build api web
docker compose run --rm migrate
docker compose up -d api web
```

如果使用内置 PostgreSQL，在上述命令中加上 `--profile local-db`。

### 5. 持久化数据

Compose 会创建两个 volume：

- `geopress-postgres-data`：仅内置 PostgreSQL 使用。
- `geopress-runtime`：后端运行时目录，包含 `runtime/browser-profiles`，用于保存小红书浏览器登录态和发布截图。

不要删除 `geopress-runtime`，否则已绑定的小红书浏览器会话会丢失。

### 6. HTTPS 和域名

当前 `web` 容器监听 HTTP 80。正式上线建议在云主机前面放 Caddy、Nginx、云负载均衡或 CDN 做 HTTPS 终止，再转发到 `WEB_PORT`。

如果直接让 Compose 暴露公网 80 端口：

```bash
WEB_PORT=80
FRONTEND_ORIGIN=https://your-domain.example
```

### 7. 健康检查

API 健康检查：

```bash
curl http://localhost:18080/api/healthz
```

前端入口：

```bash
curl http://localhost/
```

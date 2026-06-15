# 架构草案

## 目标

Geopress 面向多租户内容自动发布场景，核心流程是：内容生产、审批、渠道映射、排程发布、结果回收。

## 模块边界

- Tenant：租户、套餐、隔离策略。
- Identity：用户、角色、权限、团队。
- Content：文章、素材、标签、版本、审核状态。
- Channel：发布目标、账号授权、渠道配置。
- Publishing：发布任务、排程、重试、发布结果。
- Automation：AI 改写、模板、规则、批量生成。
- Observability：审计日志、任务日志、指标和告警。

## 后端分层

```text
cmd/api                进程入口
internal/app           应用装配和路由
internal/config        配置读取
internal/http          handler 和 middleware
internal/model         领域模型
```

骨架阶段使用内存数据，后续建议增加：

```text
internal/repository    数据访问
internal/service       业务用例
internal/queue         队列和 worker
internal/integration   第三方发布渠道适配器
migrations             数据库迁移
```

## 多租户策略

推荐从共享数据库、共享表、强制 `tenant_id` 过滤起步。原因是实现和运维成本低，适合产品早期快速迭代。

需要补足的约束：

- 所有业务表必须包含 `tenant_id`。
- repository 层禁止绕过租户上下文查询。
- 审计日志记录租户、操作者、资源和动作。
- 文件存储路径带租户前缀。
- 后台任务必须携带租户上下文。

当单个大客户需要隔离或合规要求提高时，再扩展到独立 schema 或独立数据库。

## 数据库建议

推荐 PostgreSQL 作为主数据库。它本身就是成熟的关系型数据库，适合保存租户、用户、内容、渠道、排程、任务、审计日志等强结构化业务数据。

同时，PostgreSQL 对内容自动发布和 AI 场景也比较友好：

- `jsonb` 可保存渠道配置、AI 生成参数、第三方回调原始响应等半结构化数据。
- 全文检索能力可支持内容检索、标题/正文搜索和运营后台筛选。
- `pgvector` 扩展可保存 embedding，用于语义搜索、相似内容查重、知识库召回和推荐。
- 事务、唯一约束、外键、行级锁适合处理排程发布、任务重试、幂等发布等一致性要求。
- Row Level Security 可以作为后续租户隔离的额外防线，但业务层仍应显式携带 `tenant_id`。

早期可以使用 PostgreSQL + Redis：

- PostgreSQL：主业务数据、内容正文、渠道配置、发布任务状态、审计日志。
- Redis：队列缓冲、分布式锁、短期缓存、限流计数。

如果后续 AI 内容检索规模明显增大，再评估独立向量数据库或搜索引擎；在产品早期，PostgreSQL + `pgvector` 通常足够。

当前项目已加入 `backend/migrations`，初始 schema 包含用户、工作区、知识库、媒体平台、媒体账号、内容、生成请求、发布计划、发布任务、发布结果和审计日志。核心业务接口已从 PostgreSQL 加载并持久化，后续需要继续把 handler 内的数据访问拆到 repository/service 层。

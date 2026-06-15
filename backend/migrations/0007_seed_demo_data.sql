BEGIN;

INSERT INTO users (
    id, name, email, password_hash, status, is_platform_admin,
    subscription_tier, subscription_status, subscription_expires_at,
    created_at, updated_at
)
VALUES
    ('usr_demo', 'Ava Chen', 'demo@geopress.local', 'demo-password-disabled', 'active', TRUE, 'vip', 'active', now() + interval '1 year', now() - interval '3 months', now()),
    ('usr_growth', 'Noah Wang', 'growth@geopress.local', 'demo-password-disabled', 'active', FALSE, 'free', 'active', NULL, now() - interval '2 months', now())
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    email = EXCLUDED.email,
    is_platform_admin = EXCLUDED.is_platform_admin,
    subscription_tier = EXCLUDED.subscription_tier,
    subscription_status = EXCLUDED.subscription_status,
    subscription_expires_at = EXCLUDED.subscription_expires_at,
    updated_at = EXCLUDED.updated_at;

INSERT INTO workspaces (id, name, type, plan, status, industry, language, tone, created_at, updated_at)
VALUES
    ('wks_personal', 'Ava 的个人工作区', 'personal', 'Personal', 'active', '独立创作者', 'zh-CN', '专业、清晰、克制', now() - interval '3 months', now()),
    ('wks_acme', 'Acme Growth Team', 'company', 'Team', 'active', 'B2B SaaS', 'zh-CN', '可信、实用、面向增长负责人', now() - interval '2 months', now())
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    type = EXCLUDED.type,
    plan = EXCLUDED.plan,
    status = EXCLUDED.status,
    industry = EXCLUDED.industry,
    language = EXCLUDED.language,
    tone = EXCLUDED.tone,
    updated_at = EXCLUDED.updated_at;

INSERT INTO workspace_members (workspace_id, user_id, role)
VALUES
    ('wks_personal', 'usr_demo', 'owner'),
    ('wks_acme', 'usr_demo', 'admin'),
    ('wks_acme', 'usr_growth', 'editor')
ON CONFLICT (workspace_id, user_id) DO UPDATE SET role = EXCLUDED.role;

INSERT INTO knowledge_bases (id, workspace_id, name, description, created_at, updated_at)
VALUES
    ('kb_brand', 'wks_acme', '品牌与产品资料', '公司定位、产品价值、目标客户和常用表达。', now() - interval '7 days', now() - interval '5 hours'),
    ('kb_personal', 'wks_personal', '个人写作素材', '个人介绍、服务范围、案例和写作风格。', now() - interval '7 days', now() - interval '24 hours')
ON CONFLICT (id) DO UPDATE SET
    workspace_id = EXCLUDED.workspace_id,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    updated_at = EXCLUDED.updated_at;

INSERT INTO knowledge_items (id, workspace_id, type, title, content, enabled, created_at, updated_at)
VALUES
    ('kbi_1001', 'wks_acme', 'brand', '品牌定位', 'Acme 面向 B2B SaaS 团队，帮助市场和增长负责人规划内容生产、分发和复盘。', TRUE, now() - interval '7 days', now() - interval '5 hours'),
    ('kbi_1002', 'wks_acme', 'audience', '目标受众', '主要读者是市场负责人、内容运营、创始人和增长团队。', TRUE, now() - interval '7 days', now() - interval '6 hours'),
    ('kbi_2001', 'wks_personal', 'style', '写作风格', '文章应直接、具体，避免夸张营销话术，强调可执行建议。', TRUE, now() - interval '7 days', now() - interval '24 hours')
ON CONFLICT (id) DO UPDATE SET
    workspace_id = EXCLUDED.workspace_id,
    type = EXCLUDED.type,
    title = EXCLUDED.title,
    content = EXCLUDED.content,
    enabled = EXCLUDED.enabled,
    updated_at = EXCLUDED.updated_at;

INSERT INTO knowledge_item_bases (knowledge_item_id, knowledge_base_id, workspace_id)
VALUES
    ('kbi_1001', 'kb_brand', 'wks_acme'),
    ('kbi_1002', 'kb_brand', 'wks_acme'),
    ('kbi_2001', 'kb_personal', 'wks_personal')
ON CONFLICT DO NOTHING;

INSERT INTO platform_knowledge_bases (
    id, name, description, category, price_cents, currency, marketplace_listed, created_at, updated_at
)
VALUES
    ('pkb_xhs_local_life', '小红书本地生活种草包', '适合餐饮、门店、本地服务账号的选题、结构和表达规则。', '小红书', 9900, 'CNY', TRUE, now() - interval '7 days', now() - interval '12 hours'),
    ('pkb_b2b_saas_seo', 'B2B SaaS SEO 文章包', '面向 SaaS 官网博客的文章结构、受众痛点和 CTA 写法。', 'SEO', 12900, 'CNY', FALSE, now() - interval '7 days', now() - interval '18 hours')
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    price_cents = EXCLUDED.price_cents,
    currency = EXCLUDED.currency,
    marketplace_listed = EXCLUDED.marketplace_listed,
    updated_at = EXCLUDED.updated_at;

INSERT INTO platform_knowledge_items (id, type, title, content, enabled, created_at, updated_at)
VALUES
    ('pki_xhs_1001', 'structure', '本地生活笔记结构', '开头直接给场景和人群，正文按到店理由、体验细节、价格/预约信息、避坑提醒组织，结尾用低压行动引导。', TRUE, now() - interval '7 days', now() - interval '12 hours'),
    ('pki_xhs_1002', 'compliance', '本地生活表达边界', '避免绝对化承诺和医疗化效果描述，优惠信息需明确适用条件，体验描述不要伪装成未披露广告。', TRUE, now() - interval '7 days', now() - interval '12 hours'),
    ('pki_seo_1001', 'template', 'SaaS SEO 文章骨架', '标题围绕具体问题，导语定义读者处境，正文使用问题-原因-操作步骤-指标复盘结构，结尾连接产品能力但避免硬广。', TRUE, now() - interval '7 days', now() - interval '18 hours')
ON CONFLICT (id) DO UPDATE SET
    type = EXCLUDED.type,
    title = EXCLUDED.title,
    content = EXCLUDED.content,
    enabled = EXCLUDED.enabled,
    updated_at = EXCLUDED.updated_at;

INSERT INTO platform_knowledge_item_bases (platform_knowledge_item_id, platform_knowledge_base_id)
VALUES
    ('pki_xhs_1001', 'pkb_xhs_local_life'),
    ('pki_xhs_1002', 'pkb_xhs_local_life'),
    ('pki_seo_1001', 'pkb_b2b_saas_seo')
ON CONFLICT DO NOTHING;

DELETE FROM publish_results
WHERE media_account_id IN (
    SELECT id FROM media_accounts WHERE platform_id <> 'plt_xiaohongshu'
);

DELETE FROM publish_results
WHERE job_id IN (
    SELECT id
    FROM publish_jobs
    WHERE media_account_id IN (
        SELECT id FROM media_accounts WHERE platform_id <> 'plt_xiaohongshu'
    )
);

DELETE FROM publish_jobs
WHERE media_account_id IN (
    SELECT id FROM media_accounts WHERE platform_id <> 'plt_xiaohongshu'
);

DELETE FROM publish_schedules
WHERE media_account_id IN (
    SELECT id FROM media_accounts WHERE platform_id <> 'plt_xiaohongshu'
);

DELETE FROM media_accounts
WHERE platform_id <> 'plt_xiaohongshu';

DELETE FROM media_platforms
WHERE id <> 'plt_xiaohongshu';

INSERT INTO media_platforms (
    id, name, type, enabled, supports_article, supports_image, supports_scheduling, credential_fields, created_at, updated_at
)
VALUES (
    'plt_xiaohongshu', '小红书', 'xiaohongshu', TRUE, TRUE, TRUE, FALSE, '["qrLogin"]'::jsonb, now() - interval '7 days', now()
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    type = EXCLUDED.type,
    enabled = EXCLUDED.enabled,
    supports_article = EXCLUDED.supports_article,
    supports_image = EXCLUDED.supports_image,
    supports_scheduling = EXCLUDED.supports_scheduling,
    credential_fields = EXCLUDED.credential_fields,
    updated_at = EXCLUDED.updated_at;

INSERT INTO media_accounts (
    id, workspace_id, platform_id, name, external_id, status, credentials, expires_at, last_checked_at, created_at, updated_at
)
VALUES
    ('acc_xhs_acme', 'wks_acme', 'plt_xiaohongshu', 'Acme 小红书', 'AcmeGrowth', 'pending_login', '{"loginMethod":"qr"}'::jsonb, NULL, now() - interval '90 minutes', now() - interval '7 days', now()),
    ('acc_xhs_personal', 'wks_personal', 'plt_xiaohongshu', 'Ava 小红书', 'AvaCreator', 'pending_login', '{"loginMethod":"qr"}'::jsonb, NULL, now() - interval '3 hours', now() - interval '7 days', now())
ON CONFLICT (id) DO UPDATE SET
    workspace_id = EXCLUDED.workspace_id,
    platform_id = EXCLUDED.platform_id,
    name = EXCLUDED.name,
    external_id = EXCLUDED.external_id,
    status = EXCLUDED.status,
    credentials = EXCLUDED.credentials,
    expires_at = EXCLUDED.expires_at,
    last_checked_at = EXCLUDED.last_checked_at,
    updated_at = EXCLUDED.updated_at;

INSERT INTO contents (
    id, workspace_id, knowledge_base_id, title, summary, body, keywords, status, author_name, source, created_at, updated_at
)
VALUES
    ('cnt_1001', 'wks_acme', 'kb_brand', 'Q3 SaaS 增长内容规划', '围绕获客、转化和留存的内容发布计划。', '这是一篇示例草稿，用于展示内容生命周期和排程发布。', ARRAY['SaaS', '增长', '内容营销'], 'scheduled', 'Ava Chen', 'mock_ai', now() - interval '7 days', now() - interval '2 hours'),
    ('cnt_2001', 'wks_personal', 'kb_personal', '独立顾问如何搭建内容飞轮', '用稳定输出和案例沉淀提升获客效率。', '这是一篇个人工作区示例内容。', ARRAY['独立顾问', '内容飞轮'], 'draft', 'Ava Chen', 'manual', now() - interval '7 days', now() - interval '20 hours')
ON CONFLICT (id) DO UPDATE SET
    workspace_id = EXCLUDED.workspace_id,
    knowledge_base_id = EXCLUDED.knowledge_base_id,
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    body = EXCLUDED.body,
    keywords = EXCLUDED.keywords,
    status = EXCLUDED.status,
    author_name = EXCLUDED.author_name,
    source = EXCLUDED.source,
    updated_at = EXCLUDED.updated_at;

INSERT INTO publish_schedules (
    id, workspace_id, name, content_id, media_account_id, frequency, rule, next_run_at, enabled, created_at, updated_at
)
VALUES (
    'sch_1001', 'wks_acme', '每周三小红书长文', 'cnt_1001', 'acc_xhs_acme', 'weekly', '{}'::jsonb, now() + interval '48 hours', TRUE, now() - interval '24 hours', now()
)
ON CONFLICT (id) DO UPDATE SET
    workspace_id = EXCLUDED.workspace_id,
    name = EXCLUDED.name,
    content_id = EXCLUDED.content_id,
    media_account_id = EXCLUDED.media_account_id,
    frequency = EXCLUDED.frequency,
    rule = EXCLUDED.rule,
    next_run_at = EXCLUDED.next_run_at,
    enabled = EXCLUDED.enabled,
    updated_at = EXCLUDED.updated_at;

INSERT INTO publish_jobs (
    id, workspace_id, schedule_id, content_id, media_account_id, status, scheduled_at,
    external_url, idempotency_key, last_message, created_at, updated_at
)
VALUES (
    'job_9001', 'wks_acme', 'sch_1001', 'cnt_1001', 'acc_xhs_acme', 'manual_pending', now() + interval '48 hours',
    '', 'job_9001', '小红书发布需要登录浏览器确认。', now() - interval '24 hours', now()
)
ON CONFLICT (id) DO UPDATE SET
    workspace_id = EXCLUDED.workspace_id,
    schedule_id = EXCLUDED.schedule_id,
    content_id = EXCLUDED.content_id,
    media_account_id = EXCLUDED.media_account_id,
    status = EXCLUDED.status,
    scheduled_at = EXCLUDED.scheduled_at,
    external_url = EXCLUDED.external_url,
    last_message = EXCLUDED.last_message,
    updated_at = EXCLUDED.updated_at;

COMMIT;

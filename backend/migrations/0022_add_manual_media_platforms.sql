BEGIN;

-- 新增平台通过服务端托管浏览器完成二维码登录和发布；失败时仍保留人工确认兜底。
INSERT INTO media_platforms (
    id, name, type, enabled, supports_article, supports_image, supports_scheduling, credential_fields, capabilities, created_at, updated_at
)
VALUES
    (
        'plt_netease',
        '网易号',
        'netease',
        TRUE,
        TRUE,
        TRUE,
        FALSE,
        '["qrLogin"]'::jsonb,
        '{
            "authorizationMethods": ["qr_login"],
            "publishModes": ["manual", "browser"],
            "contentFormats": ["article", "image"],
            "capabilities": [
                {
                    "name": "authorization",
                    "mode": "browser",
                    "enabled": true,
                    "manualFallback": true,
                    "notes": "通过服务端托管浏览器完成二维码登录；不保存动态平台请求头。"
                },
                {
                    "name": "content_publish",
                    "mode": "browser",
                    "enabled": true,
                    "manualFallback": true,
                    "notes": "通过已登录浏览器会话发布网易号文章，失败时保留人工确认路径。"
                }
            ],
            "rateLimits": {}
        }'::jsonb,
        now(),
        now()
    ),
    (
        'plt_toutiao',
        '头条号',
        'toutiao',
        TRUE,
        TRUE,
        TRUE,
        FALSE,
        '["qrLogin"]'::jsonb,
        '{
            "authorizationMethods": ["qr_login"],
            "publishModes": ["manual", "browser"],
            "contentFormats": ["article", "image"],
            "capabilities": [
                {
                    "name": "authorization",
                    "mode": "browser",
                    "enabled": true,
                    "manualFallback": true,
                    "notes": "通过服务端托管浏览器完成二维码登录；不保存动态平台请求头。"
                },
                {
                    "name": "content_publish",
                    "mode": "browser",
                    "enabled": true,
                    "manualFallback": true,
                    "notes": "通过已登录浏览器会话发布头条号文章，失败时保留人工确认路径。"
                }
            ],
            "rateLimits": {}
        }'::jsonb,
        now(),
        now()
    ),
    (
        'plt_sohu',
        '搜狐号',
        'sohu',
        TRUE,
        TRUE,
        TRUE,
        FALSE,
        '["phoneNumber"]'::jsonb,
        '{
            "authorizationMethods": ["phone_sms"],
            "publishModes": ["manual", "browser"],
            "contentFormats": ["article", "image"],
            "capabilities": [
                {
                    "name": "authorization",
                    "mode": "browser",
                    "enabled": true,
                    "manualFallback": true,
                    "notes": "通过服务端托管浏览器完成手机号短信验证码登录；验证码由用户在前端输入，不保存短信验证码。"
                },
                {
                    "name": "content_publish",
                    "mode": "browser",
                    "enabled": true,
                    "manualFallback": true,
                    "notes": "通过已登录浏览器会话发布搜狐号文章，失败时保留人工确认路径。"
                }
            ],
            "rateLimits": {}
        }'::jsonb,
        now(),
        now()
    )
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    type = EXCLUDED.type,
    enabled = EXCLUDED.enabled,
    supports_article = EXCLUDED.supports_article,
    supports_image = EXCLUDED.supports_image,
    supports_scheduling = EXCLUDED.supports_scheduling,
    credential_fields = EXCLUDED.credential_fields,
    capabilities = EXCLUDED.capabilities,
    updated_at = EXCLUDED.updated_at;

COMMIT;

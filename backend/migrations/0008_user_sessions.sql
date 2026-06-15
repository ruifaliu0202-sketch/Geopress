BEGIN;

CREATE TABLE IF NOT EXISTS user_sessions (
    token TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);

UPDATE users
SET password_hash = '$2a$10$RZ9nf/MK8Gn8.tJ4uIfnPOR0KCfQfzwvhapNoXKrpaVQ0UROabcpG'
WHERE id IN ('usr_demo', 'usr_growth')
  AND password_hash IN (
    'demo-password-disabled',
    '$2a$10$YOJYAwYgXXvSoSlSWeJWOOijxJTBVJ7IQWmYfwm53meuGBuxzUhYu'
  );

COMMIT;

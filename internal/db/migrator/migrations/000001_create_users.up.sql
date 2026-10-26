CREATE TABLE IF NOT EXISTS users
(
    id                BIGSERIAL PRIMARY KEY,
    username          VARCHAR(64)   NOT NULL,
    password_hash     VARCHAR(255)  NOT NULL DEFAULT '', -- 空串表示未设置密码(仅第三方登录)
    email             VARCHAR(255)  NOT NULL DEFAULT '',
    email_verified_at TIMESTAMPTZ,
    nickname          VARCHAR(64)   NOT NULL DEFAULT '',
    avatar_url        VARCHAR(1024) NOT NULL DEFAULT '',
    is_disabled       BOOLEAN       NOT NULL DEFAULT FALSE,
    is_admin          BOOLEAN       NOT NULL DEFAULT FALSE,
    banned_until      TIMESTAMPTZ,
    ban_reason        VARCHAR(255)  NOT NULL DEFAULT '',
    last_login_at     TIMESTAMPTZ,
    created_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- 用户名/邮箱大小写不敏感唯一, 查询需使用 lower(...) 才能命中; 未填写的邮箱不参与唯一约束
CREATE UNIQUE INDEX IF NOT EXISTS uniq_users_username_lower ON users (LOWER(username));
CREATE UNIQUE INDEX IF NOT EXISTS uniq_users_email_lower ON users (LOWER(email)) WHERE email <> '';
CREATE INDEX IF NOT EXISTS idx_users_banned_until ON users (banned_until);
CREATE INDEX IF NOT EXISTS idx_users_last_login_at ON users (last_login_at);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users (created_at);

-- 第三方登录身份, 一个用户可绑定多个
CREATE TABLE IF NOT EXISTS user_identities
(
    id               BIGSERIAL PRIMARY KEY,
    user_id          BIGINT       NOT NULL,
    provider         VARCHAR(32)  NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_user_identities_provider_uid ON user_identities (provider, provider_user_id);
CREATE INDEX IF NOT EXISTS idx_user_identities_user_id ON user_identities (user_id);

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

-- 操作日志, 用户和管理员的操作统一记录在这里, 只有管理员可以查看
CREATE TABLE IF NOT EXISTS audit_logs
(
    id          BIGSERIAL PRIMARY KEY,
    actor_id    BIGINT       NOT NULL DEFAULT 0,  -- 操作人, 0 表示未登录(如登录失败)
    scope       VARCHAR(16)  NOT NULL DEFAULT '', -- user: 用户自己的操作, admin: 管理员操作
    action      VARCHAR(64)  NOT NULL DEFAULT '',
    target_type VARCHAR(32)  NOT NULL DEFAULT '', -- 操作对象类型, 如 user
    target_id   BIGINT       NOT NULL DEFAULT 0,
    detail      JSONB        NOT NULL DEFAULT '{}',
    ip          VARCHAR(64)  NOT NULL DEFAULT '',
    user_agent  VARCHAR(512) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_id_created_at ON audit_logs (actor_id, created_at);
CREATE INDEX IF NOT EXISTS idx_audit_logs_target ON audit_logs (target_type, target_id, created_at);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action_created_at ON audit_logs (action, created_at);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs (created_at);

-- 货币定义, 新增货币只需插入一行
CREATE TABLE IF NOT EXISTS currencies
(
    code       VARCHAR(32) PRIMARY KEY,               -- 货币代码, 如 chip / gold
    name       VARCHAR(32) NOT NULL,
    decimals   SMALLINT    NOT NULL DEFAULT 0,        -- 展示用小数位, 金额统一按最小单位的整数存储
    sort_order INTEGER     NOT NULL DEFAULT 0,
    is_enabled BOOLEAN     NOT NULL DEFAULT TRUE,     -- 停用后不能再变动余额, 已有余额保留
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO currencies (code, name, decimals, sort_order)
VALUES ('chip', '筹码', 0, 10), -- 基础货币
       ('gold', '金条', 0, 20)  -- 珍贵货币
ON CONFLICT (code) DO NOTHING;

-- 用户余额, 每个用户每种货币一行, 首次入账时创建
CREATE TABLE IF NOT EXISTS wallet_balances
(
    user_id    BIGINT      NOT NULL,
    currency   VARCHAR(32) NOT NULL,
    balance    BIGINT      NOT NULL DEFAULT 0, -- 可用余额
    frozen     BIGINT      NOT NULL DEFAULT 0, -- 冻结余额(如竞拍押金), 不计入可用
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, currency),
    CONSTRAINT chk_wallet_balances_balance CHECK (balance >= 0),
    CONSTRAINT chk_wallet_balances_frozen CHECK (frozen >= 0)
);

-- 余额流水, 只追加不修改; 每次余额变动对应一行
CREATE TABLE IF NOT EXISTS wallet_transactions
(
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT       NOT NULL,
    currency        VARCHAR(32)  NOT NULL,
    amount          BIGINT       NOT NULL DEFAULT 0,  -- 可用余额变动, 正数入账, 负数出账
    frozen_amount   BIGINT       NOT NULL DEFAULT 0,  -- 冻结余额变动
    balance_after   BIGINT       NOT NULL,            -- 变动后可用余额
    frozen_after    BIGINT       NOT NULL,            -- 变动后冻结余额
    biz_type        VARCHAR(32)  NOT NULL,            -- 业务类型, 如 admin_adjust
    biz_id          VARCHAR(64)  NOT NULL DEFAULT '', -- 业务单号, 如保险柜 ID
    idempotency_key VARCHAR(128) NOT NULL DEFAULT '', -- 幂等键, 非空时全局唯一
    operator_id     BIGINT       NOT NULL DEFAULT 0,  -- 操作人, 0 表示系统
    remark          VARCHAR(255) NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_wallet_transactions_nonzero CHECK (amount <> 0 OR frozen_amount <> 0)
);

CREATE INDEX IF NOT EXISTS idx_wallet_transactions_user_currency ON wallet_transactions (user_id, currency, id);
CREATE INDEX IF NOT EXISTS idx_wallet_transactions_biz ON wallet_transactions (biz_type, biz_id);
CREATE INDEX IF NOT EXISTS idx_wallet_transactions_operator_id ON wallet_transactions (operator_id) WHERE operator_id <> 0;
CREATE INDEX IF NOT EXISTS idx_wallet_transactions_created_at ON wallet_transactions (created_at);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_wallet_transactions_idempotency_key ON wallet_transactions (idempotency_key) WHERE idempotency_key <> '';

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

-- 系统设置, 一个设置组一行, 未保存过的设置组没有记录, 使用代码中的默认值
CREATE TABLE IF NOT EXISTS system_settings
(
    key        VARCHAR(64) PRIMARY KEY,
    value      JSONB       NOT NULL DEFAULT '{}', -- 密钥字段为密文
    updated_by BIGINT      NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 物品定义, 删除为软删除, ID 不会被新物品复用; 已删除的物品在玩家背包中按占位物品处理
CREATE TABLE IF NOT EXISTS items
(
    id            BIGSERIAL PRIMARY KEY,
    name          VARCHAR(64)   NOT NULL,
    description   VARCHAR(255)  NOT NULL DEFAULT '',
    quality       SMALLINT      NOT NULL DEFAULT 1,      -- 品质 1-7: 灰/绿/蓝/紫/金/红/炫彩
    image_url     VARCHAR(1024) NOT NULL DEFAULT '',
    sell_currency VARCHAR(32)   NOT NULL DEFAULT 'chip', -- 卖出得到的货币
    sell_price    BIGINT        NOT NULL DEFAULT 0,      -- 卖出价格, 0 表示不可卖出
    weight        INTEGER       NOT NULL DEFAULT 0,      -- 掉落权重, 0 表示不会掉落
    width         SMALLINT      NOT NULL DEFAULT 1,      -- 在保险柜中占的格数
    height        SMALLINT      NOT NULL DEFAULT 1,
    is_enabled    BOOLEAN       NOT NULL DEFAULT FALSE,  -- 下架后不会掉落, 也不能再发放, 已有的保留
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ,                           -- 软删除时间
    CONSTRAINT chk_items_quality CHECK (quality BETWEEN 1 AND 7),
    CONSTRAINT chk_items_sell_price CHECK (sell_price >= 0),
    CONSTRAINT chk_items_weight CHECK (weight >= 0),
    CONSTRAINT chk_items_width CHECK (width >= 1),
    CONSTRAINT chk_items_height CHECK (height >= 1)
);

CREATE INDEX IF NOT EXISTS idx_items_deleted_at ON items (deleted_at);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_items_name ON items (name) WHERE deleted_at IS NULL; -- 已删除的物品不占用名称

-- 用户背包, 每个用户每种物品一行, 首次获得时创建, 数量扣到 0 也保留
CREATE TABLE IF NOT EXISTS inventory_items
(
    user_id    BIGINT      NOT NULL,
    item_id    BIGINT      NOT NULL,
    quantity   BIGINT      NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), -- 首次获得时间
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, item_id),
    CONSTRAINT chk_inventory_items_quantity CHECK (quantity >= 0)
);

CREATE INDEX IF NOT EXISTS idx_inventory_items_item_id ON inventory_items (item_id) WHERE quantity > 0; -- 统计物品的持有人数

-- 背包流水, 只追加不修改; 每次背包变动对应一行
CREATE TABLE IF NOT EXISTS inventory_transactions
(
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT       NOT NULL,
    item_id         BIGINT       NOT NULL,
    item_name       VARCHAR(64)  NOT NULL,            -- 变动时的物品名称
    quantity        BIGINT       NOT NULL,            -- 正数获得, 负数消耗
    quantity_after  BIGINT       NOT NULL,            -- 变动后数量
    biz_type        VARCHAR(32)  NOT NULL,            -- 业务类型, 如 admin_adjust
    biz_id          VARCHAR(64)  NOT NULL DEFAULT '', -- 业务单号
    idempotency_key VARCHAR(128) NOT NULL DEFAULT '', -- 幂等键, 非空时全局唯一
    operator_id     BIGINT       NOT NULL DEFAULT 0,  -- 操作人, 0 表示系统
    remark          VARCHAR(255) NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_inventory_transactions_nonzero CHECK (quantity <> 0)
);

CREATE INDEX IF NOT EXISTS idx_inventory_transactions_user_item ON inventory_transactions (user_id, item_id, id);
CREATE INDEX IF NOT EXISTS idx_inventory_transactions_biz ON inventory_transactions (biz_type, biz_id);
CREATE INDEX IF NOT EXISTS idx_inventory_transactions_operator_id ON inventory_transactions (operator_id) WHERE operator_id <> 0;
CREATE INDEX IF NOT EXISTS idx_inventory_transactions_created_at ON inventory_transactions (created_at);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_inventory_transactions_idempotency_key ON inventory_transactions (idempotency_key) WHERE idempotency_key <> '';

-- 出售单, 一次出售一行; 出售产生的背包流水和余额流水, biz_id 都是出售单 ID
CREATE TABLE IF NOT EXISTS inventory_sell_orders
(
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT      NOT NULL,
    idempotency_key VARCHAR(64) NOT NULL, -- 幂等键, 同一用户内唯一
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_inventory_sell_orders_user_id ON inventory_sell_orders (user_id, id);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_inventory_sell_orders_idempotency_key ON inventory_sell_orders (user_id, idempotency_key);

-- 出售单明细, 每种物品一行, 记录出售时的物品名称和价格
CREATE TABLE IF NOT EXISTS inventory_sell_order_items
(
    id            BIGSERIAL PRIMARY KEY,
    order_id      BIGINT      NOT NULL,
    item_id       BIGINT      NOT NULL,
    item_name     VARCHAR(64) NOT NULL, -- 出售时的物品名称
    quantity      BIGINT      NOT NULL,
    sell_currency VARCHAR(32) NOT NULL,
    sell_price    BIGINT      NOT NULL, -- 出售时的单价
    amount        BIGINT      NOT NULL, -- 数量 × 单价
    CONSTRAINT chk_inventory_sell_order_items_quantity CHECK (quantity > 0),
    CONSTRAINT chk_inventory_sell_order_items_sell_price CHECK (sell_price > 0)
);

CREATE INDEX IF NOT EXISTS idx_inventory_sell_order_items_order_id ON inventory_sell_order_items (order_id);
CREATE INDEX IF NOT EXISTS idx_inventory_sell_order_items_item_id ON inventory_sell_order_items (item_id);

-- 掉落池, 玩法按代码引用, 如开箱、转盘; 删除时连同条目一起删除
CREATE TABLE IF NOT EXISTS loot_pools
(
    id          BIGSERIAL PRIMARY KEY,
    code        VARCHAR(64)  NOT NULL,               -- 玩法中引用的代码, 如 crate_basic
    name        VARCHAR(64)  NOT NULL,
    description VARCHAR(255) NOT NULL DEFAULT '',
    is_enabled  BOOLEAN      NOT NULL DEFAULT FALSE, -- 停用后不能再抽取
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_loot_pools_code ON loot_pools (code);

-- 掉落池条目, 每行掉落一种物品或货币; 修改时整池替换
CREATE TABLE IF NOT EXISTS loot_pool_entries
(
    id           BIGSERIAL PRIMARY KEY,
    pool_id      BIGINT      NOT NULL,
    item_id      BIGINT      NOT NULL DEFAULT 0,  -- 掉落的物品, 与 currency 二选一
    currency     VARCHAR(32) NOT NULL DEFAULT '', -- 掉落的货币
    weight       INTEGER     NOT NULL,            -- 抽中的概率为权重 / 池中可抽取条目的权重之和
    min_quantity BIGINT      NOT NULL DEFAULT 1,  -- 每次掉落的数量在该范围内均匀随机, 货币为金额
    max_quantity BIGINT      NOT NULL DEFAULT 1,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_loot_pool_entries_target CHECK ((item_id <> 0) <> (currency <> '')),
    CONSTRAINT chk_loot_pool_entries_weight CHECK (weight > 0),
    CONSTRAINT chk_loot_pool_entries_quantity CHECK (min_quantity >= 1 AND max_quantity >= min_quantity)
);

CREATE INDEX IF NOT EXISTS idx_loot_pool_entries_pool_id ON loot_pool_entries (pool_id);

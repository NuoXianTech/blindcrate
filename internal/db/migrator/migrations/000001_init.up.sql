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
    width         SMALLINT      NOT NULL DEFAULT 1,      -- 在保险柜中占的格数
    height        SMALLINT      NOT NULL DEFAULT 1,
    is_enabled    BOOLEAN       NOT NULL DEFAULT FALSE,  -- 下架后不会掉落, 也不能再发放, 已有的保留
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ,                           -- 软删除时间
    CONSTRAINT chk_items_quality CHECK (quality BETWEEN 1 AND 7),
    CONSTRAINT chk_items_sell_price CHECK (sell_price >= 0),
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
    idempotency_key VARCHAR(64) NOT NULL,           -- 幂等键, 同一用户内唯一
    fee_rate        INTEGER     NOT NULL DEFAULT 0, -- 出售时的手续费率, 单位为基点
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

-- 对局, 由匹配创建, 创建时即开始; 阶段由玩法定义
CREATE TABLE IF NOT EXISTS matches
(
    id            BIGSERIAL PRIMARY KEY,
    mode          VARCHAR(32) NOT NULL,            -- 玩法代码
    status        VARCHAR(16) NOT NULL,            -- playing、finished 或 cancelled
    phase         VARCHAR(32) NOT NULL DEFAULT '', -- 当前阶段, 结束后保留最后一个阶段
    round         INTEGER     NOT NULL DEFAULT 0,
    phase_ends_at TIMESTAMPTZ,                     -- 当前阶段的截止时间, 不限时或已结束时为空
    version       BIGINT      NOT NULL DEFAULT 0,  -- 每次切换阶段加 1
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at   TIMESTAMPTZ
);

-- 巡检到期仍未推进的对局
CREATE INDEX IF NOT EXISTS idx_matches_status_phase_ends_at ON matches (status, phase_ends_at);

-- 对局中的座位, 坐的是真人或人机
CREATE TABLE IF NOT EXISTS match_players
(
    id         BIGSERIAL PRIMARY KEY,
    match_id   BIGINT      NOT NULL,
    seat       SMALLINT    NOT NULL,             -- 座位号, 从 1 开始
    user_id    BIGINT      NOT NULL DEFAULT 0,   -- 人机为 0
    bot_name   VARCHAR(32) NOT NULL DEFAULT '',
    bot_level  SMALLINT    NOT NULL DEFAULT 0,   -- 人机等级, 真人为 0
    rank       SMALLINT    NOT NULL DEFAULT 0,   -- 结束时的名次, 0 表示没有名次
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_match_players_bot CHECK ((user_id = 0) = (bot_level > 0))
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_match_players_match_id_seat ON match_players (match_id, seat);
CREATE INDEX IF NOT EXISTS idx_match_players_user_id ON match_players (user_id);

-- 玩家的对局统计, 每个用户每个统计范围每个周期一行, 结算时累加
CREATE TABLE IF NOT EXISTS user_stats
(
    user_id    BIGINT      NOT NULL,
    scope      VARCHAR(64) NOT NULL,           -- 统计范围, 如 all、mode:ranked、venue:dock
    period     VARCHAR(32) NOT NULL,           -- 统计周期, 如 total、day:2026-10-28
    matches    BIGINT      NOT NULL DEFAULT 0, -- 完成的对局数
    wins       BIGINT      NOT NULL DEFAULT 0, -- 胜利次数
    profit     BIGINT      NOT NULL DEFAULT 0, -- 净收益, 可以为负
    bid_won    BIGINT      NOT NULL DEFAULT 0, -- 成交次数
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, scope, period)
);

-- 地图(拍卖场), 每张地图有人机和排位两种玩法; 软删除, 代码在已删除的地图中也唯一
CREATE TABLE IF NOT EXISTS venues
(
    id             BIGSERIAL PRIMARY KEY,
    code           VARCHAR(32)   NOT NULL,                -- 拼进玩法代码和统计范围, 创建后不能修改
    name           VARCHAR(64)   NOT NULL,
    description    VARCHAR(255)  NOT NULL DEFAULT '',
    image_url      VARCHAR(1024) NOT NULL DEFAULT '',
    sort_order     INTEGER       NOT NULL DEFAULT 0,      -- 大厅中从小到大排列
    is_enabled     BOOLEAN       NOT NULL DEFAULT FALSE,  -- 停用后不能加入队列, 进行中的对局照常打完
    pve_enabled    BOOLEAN       NOT NULL DEFAULT FALSE,
    ranked_enabled BOOLEAN       NOT NULL DEFAULT FALSE,
    bid_currency   VARCHAR(32)   NOT NULL DEFAULT 'chip', -- 出价和计算仓库价值的货币
    entry_cost     JSONB         NOT NULL DEFAULT '{}',   -- 入场代价 reward.Bundle, 数量为正数
    requirements   JSONB         NOT NULL DEFAULT '[]',   -- 入场条件 []condition.Condition
    warehouse      JSONB         NOT NULL DEFAULT '{}',   -- 仓库生成规则
    auction        JSONB         NOT NULL DEFAULT '{}',   -- 拍卖参数
    bots           JSONB         NOT NULL DEFAULT '{}',   -- 人机参数
    pve            JSONB         NOT NULL DEFAULT '{}',   -- 人机模式防刷参数
    simulation     JSONB         NOT NULL DEFAULT '{}',   -- 保存时的模拟结果
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_venues_code ON venues (code);
CREATE INDEX IF NOT EXISTS idx_venues_deleted_at ON venues (deleted_at);

-- 暗拍的对局级数据, 与 matches 一一对应, 开局时创建
CREATE TABLE IF NOT EXISTS auction_matches
(
    match_id     BIGINT PRIMARY KEY,
    venue_id     BIGINT      NOT NULL,
    mode         VARCHAR(16) NOT NULL,              -- pve 或 ranked
    venue        JSONB       NOT NULL,              -- 开局时的地图配置快照
    value        BIGINT      NOT NULL,              -- 仓库价值, 出价货币的回收价之和
    piece_count  INTEGER     NOT NULL,              -- 藏品件数
    contenders   JSONB       NOT NULL DEFAULT '[]', -- 加轮时可以出价的座位, 为空表示所有人
    extra_rounds INTEGER     NOT NULL DEFAULT 0,    -- 已加的轮数
    outcome      VARCHAR(16) NOT NULL DEFAULT '',   -- 结算前为空, 之后为 sold、unsold 或 cancelled
    winner_seat  SMALLINT    NOT NULL DEFAULT 0,    -- 成交者的座位, 没有成交时为 0
    price        BIGINT      NOT NULL DEFAULT 0,    -- 成交价
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    settled_at   TIMESTAMPTZ                        -- 结算或取消的时间
);

-- 暗拍中的座位, 真人在准入时创建(此时座位号为 0), 人机在开局时创建
CREATE TABLE IF NOT EXISTS auction_players
(
    id          BIGSERIAL PRIMARY KEY,
    match_id    BIGINT      NOT NULL,
    seat        SMALLINT    NOT NULL DEFAULT 0,
    user_id     BIGINT      NOT NULL DEFAULT 0,     -- 人机为 0
    bot_level   SMALLINT    NOT NULL DEFAULT 0,
    is_practice BOOLEAN     NOT NULL DEFAULT FALSE, -- 练习局, 不扣入场代价、不冻结资金、不发藏品、不写统计
    entry_cost  JSONB       NOT NULL DEFAULT '{}',  -- 实际扣除的入场代价
    carry       BIGINT      NOT NULL DEFAULT 0,     -- 带入资金, 出价的上限
    frozen      BIGINT      NOT NULL DEFAULT 0,     -- 冻结的带入资金
    estimate    BIGINT      NOT NULL DEFAULT 0,     -- 人机对仓库价值的估值
    bid         BIGINT      NOT NULL DEFAULT 0,     -- 当前出价
    bid_at      TIMESTAMPTZ,                        -- 当前出价的提交时间
    feedback    JSONB       NOT NULL DEFAULT '[]',  -- 每轮结束后的反馈
    profit      BIGINT      NOT NULL DEFAULT 0,     -- 结算后的盈亏
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auction_players_match_id ON auction_players (match_id);

-- 暗拍每轮的出价, 每人每轮一条, 0 表示不加价
CREATE TABLE IF NOT EXISTS auction_bids
(
    id         BIGSERIAL PRIMARY KEY,
    match_id   BIGINT      NOT NULL,
    round      INTEGER     NOT NULL,
    seat       SMALLINT    NOT NULL,
    amount     BIGINT      NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_auction_bids_match_id_round_seat ON auction_bids (match_id, round, seat);

-- 暗拍仓库里的藏品, 开局时快照名称和回收价
CREATE TABLE IF NOT EXISTS auction_warehouse_items
(
    id            BIGSERIAL PRIMARY KEY,
    match_id      BIGINT        NOT NULL,
    reveal_order  INTEGER       NOT NULL, -- 公开品质的顺序, 从 1 开始, 第 N 轮公开前 N 件
    item_id       BIGINT        NOT NULL,
    name          VARCHAR(64)   NOT NULL,
    image_url     VARCHAR(1024) NOT NULL DEFAULT '',
    quality       SMALLINT      NOT NULL,
    width         SMALLINT      NOT NULL,
    height        SMALLINT      NOT NULL,
    x             SMALLINT      NOT NULL,
    y             SMALLINT      NOT NULL,
    sell_currency VARCHAR(32)   NOT NULL,
    sell_price    BIGINT        NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_auction_warehouse_items_match_id_reveal_order ON auction_warehouse_items (match_id, reveal_order);

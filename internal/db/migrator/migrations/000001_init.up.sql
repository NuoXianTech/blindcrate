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
    biz_id          VARCHAR(64)  NOT NULL DEFAULT '', -- 业务单号, 如对局 ID
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

-- 设置, 一个设置组一行, 未保存过的设置组没有记录, 使用代码中的默认值
CREATE TABLE IF NOT EXISTS settings
(
    key        VARCHAR(64) PRIMARY KEY,
    value      JSONB       NOT NULL DEFAULT '{}', -- 密钥字段为密文
    version    BIGINT      NOT NULL DEFAULT 0,    -- 每次保存加 1, 保存时校验, 防止覆盖别人的修改
    updated_by BIGINT      NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 设置组的版本, 每次保存写入一条, 用于查看历史和回滚
CREATE TABLE IF NOT EXISTS setting_revisions
(
    id           BIGSERIAL PRIMARY KEY,
    key          VARCHAR(64) NOT NULL,
    version      BIGINT      NOT NULL,
    value        JSONB       NOT NULL DEFAULT '{}', -- 保存后的完整内容, 密钥字段为密文
    changes      JSONB       NOT NULL DEFAULT '{}', -- 相对上一个版本改动的字段, 格式同操作日志
    action       VARCHAR(16) NOT NULL,              -- update / rollback / reset / baseline
    from_version BIGINT      NOT NULL DEFAULT 0,    -- 回滚时为回滚到的版本
    updated_by   BIGINT      NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_setting_revisions_key_version ON setting_revisions (key, version);

-- 物品品质, 新增品质只需插入一行; 等级越大越稀有, 物品和仓库藏品按等级引用
CREATE TABLE IF NOT EXISTS qualities
(
    level      SMALLINT PRIMARY KEY,
    code       VARCHAR(32) NOT NULL,             -- 自然键, 导入导出时用来对应
    name       VARCHAR(32) NOT NULL,
    color      VARCHAR(7)  NOT NULL,             -- #rrggbb
    effect     VARCHAR(16) NOT NULL DEFAULT '',  -- 特效, 空表示纯色, rainbow 为流动渐变
    scan_ms    INTEGER     NOT NULL DEFAULT 0,   -- 开箱时扫描这一品质藏品的时长, 毫秒
    sort_order INTEGER     NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_qualities_code ON qualities (code);

INSERT INTO qualities (level, code, name, color, effect, scan_ms, sort_order)
VALUES (1, 'gray', '灰色', '#a1a5ab', '', 400, 10),
       (2, 'green', '绿色', '#4bcb71', '', 500, 20),
       (3, 'blue', '蓝色', '#3ba6f5', '', 700, 30),
       (4, 'purple', '紫色', '#ad74ff', '', 1000, 40),
       (5, 'gold', '金色', '#fab72a', '', 1500, 50),
       (6, 'red', '红色', '#f94144', '', 2000, 60),
       (7, 'rainbow', '炫彩', '#ff6ad5', 'rainbow', 3000, 70)
ON CONFLICT (level) DO NOTHING;

-- 物品定义, 删除为软删除, ID 不会被新物品复用; 已删除的物品在玩家背包中按占位物品处理
CREATE TABLE IF NOT EXISTS items
(
    id            BIGSERIAL PRIMARY KEY,
    code          VARCHAR(64)   NOT NULL,                       -- 英文别名, 可以重复, 物品以 ID 区分
    name          VARCHAR(64)   NOT NULL,
    description   VARCHAR(255)  NOT NULL DEFAULT '',
    quality       SMALLINT      NOT NULL DEFAULT 1,             -- 品质等级, 见 qualities
    category      VARCHAR(32)   NOT NULL DEFAULT 'collectible', -- 类别代码, 由代码注册; 仓库只抽藏品
    image_url     VARCHAR(1024) NOT NULL DEFAULT '',
    sell_currency VARCHAR(32)   NOT NULL,                       -- 卖出得到的货币
    sell_price    BIGINT        NOT NULL DEFAULT 0,             -- 卖出价格, 0 表示不可卖出
    width         SMALLINT      NOT NULL DEFAULT 1,             -- 在仓库中占的格数
    height        SMALLINT      NOT NULL DEFAULT 1,
    effect        VARCHAR(32)   NOT NULL DEFAULT '',            -- 使用效果代码, 空表示不能使用, 藏品没有效果
    effect_params JSONB         NOT NULL DEFAULT '{}',          -- 效果参数, 结构由效果决定
    is_enabled    BOOLEAN       NOT NULL DEFAULT FALSE,         -- 下架后不会掉落, 也不能再发放, 已有的保留
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ,                                  -- 软删除时间
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
    released_at   TIMESTAMPTZ,                     -- 玩家可以离开的时间, 之后不再算作进行中的对局
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
CREATE INDEX IF NOT EXISTS idx_match_players_user_id_match_id ON match_players (user_id, match_id DESC);

-- 玩家的统计数据, 每个用户每个统计范围每个周期每个指标一行, 结算时累加; 新增指标不需要改表
CREATE TABLE IF NOT EXISTS stats
(
    user_id    BIGINT      NOT NULL,
    scope      VARCHAR(64) NOT NULL,           -- 统计范围, 如 all、mode:ranked、venue:dock
    period     VARCHAR(32) NOT NULL,           -- 统计周期, 如 total、day:2026-10-28
    metric     VARCHAR(64) NOT NULL,           -- 指标, 如 matches、wins、profit:chip
    value      BIGINT      NOT NULL DEFAULT 0, -- 累计值, 金额按最小单位计, 可以为负
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, scope, period, metric)
);

-- 排行榜按范围、周期和指标取数值最大的若干行
CREATE INDEX IF NOT EXISTS idx_stats_board ON stats (scope, period, metric, value DESC);

-- 排行榜, 由管理员配置, 按 stats 中的某个指标或赛季的段位分排名
CREATE TABLE IF NOT EXISTS leaderboards
(
    id          BIGSERIAL PRIMARY KEY,
    code        VARCHAR(64)  NOT NULL,               -- 自然键, 如 ranked_wins
    name        VARCHAR(64)  NOT NULL,
    description VARCHAR(255) NOT NULL DEFAULT '',
    metric      VARCHAR(64)  NOT NULL,               -- 排名的指标, 如 wins、profit:chip; rating 为赛季的段位分
    scope       VARCHAR(64)  NOT NULL,               -- 统计范围, 如 all、mode:ranked
    period      VARCHAR(16)  NOT NULL,               -- 统计周期, total、day 或 season
    size        INTEGER      NOT NULL DEFAULT 100,   -- 展示前多少名
    min_matches BIGINT       NOT NULL DEFAULT 0,     -- 同一范围和周期内至少完成多少局才上榜
    sort_order  INTEGER      NOT NULL DEFAULT 0,
    is_enabled  BOOLEAN      NOT NULL DEFAULT FALSE, -- 停用后玩家看不到
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_leaderboards_code ON leaderboards (code);

INSERT INTO leaderboards (code, name, description, metric, scope, period, size, min_matches, sort_order, is_enabled)
VALUES ('ranked_wins', '排位胜场', '排位中拿到第一名的次数', 'wins', 'mode:ranked', 'total', 100, 0, 10, TRUE),
       ('daily_profit_chip', '今日收益', '今天所有对局的筹码净收益, 至少完成 3 局才上榜', 'profit:chip', 'all', 'day', 100, 3, 20, TRUE),
       ('bid_won', '成交次数', '累计拍下仓库的次数', 'bid_won', 'all', 'total', 100, 0, 30, TRUE),
       ('season_rating', '赛季段位分', '本赛季的段位分', 'rating', 'all', 'season', 100, 0, 40, TRUE)
ON CONFLICT (code) DO NOTHING;

-- 赛季, 时间不能互相重叠; 状态由时间推算, 赛季奖励发完后写入 settled_at
CREATE TABLE IF NOT EXISTS seasons
(
    id           BIGSERIAL PRIMARY KEY,
    code         VARCHAR(24) NOT NULL,              -- 自然键, 拼进统计周期 season:<code>, 创建后不能修改
    name         VARCHAR(64) NOT NULL,
    starts_at    TIMESTAMPTZ NOT NULL,              -- 包含
    ends_at      TIMESTAMPTZ NOT NULL,              -- 不包含
    settled_at   TIMESTAMPTZ,                       -- 赛季奖励发完的时间
    rating_carry SMALLINT    NOT NULL DEFAULT 50,   -- 段位分继承上一个赛季的百分比
    rewards      JSONB       NOT NULL DEFAULT '[]', -- 赛季奖励, [{min_rating, bundle}], 按分数线从低到高
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_seasons_time CHECK (ends_at > starts_at),
    CONSTRAINT chk_seasons_rating_carry CHECK (rating_carry BETWEEN 0 AND 100)
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_seasons_code ON seasons (code);
CREATE INDEX IF NOT EXISTS idx_seasons_starts_at ON seasons (starts_at);

-- 段位分, 每人每个赛季一行, 赛季之外的排位记在赛季 0; 第一次结算排位时按初始分创建
CREATE TABLE IF NOT EXISTS ranked_ratings
(
    user_id    BIGINT      NOT NULL,
    season_id  BIGINT      NOT NULL,           -- 赛季之外为 0
    rating     INTEGER     NOT NULL,
    peak       INTEGER     NOT NULL,           -- 本赛季的最高分
    matches    INTEGER     NOT NULL DEFAULT 0, -- 本赛季计分的排位局数
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, season_id)
);

-- 段位分排行榜按赛季取分数最高的若干行
CREATE INDEX IF NOT EXISTS idx_ranked_ratings_board ON ranked_ratings (season_id, rating DESC);

-- 段位, 段位分不低于分数线时属于分数线最高的那个段位; 低于最低的分数线时没有段位
CREATE TABLE IF NOT EXISTS ranked_tiers
(
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(32) NOT NULL,
    min_rating INTEGER     NOT NULL, -- 分数线, 包含
    color      VARCHAR(7)  NOT NULL, -- #rrggbb
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_ranked_tiers_min_rating CHECK (min_rating >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_ranked_tiers_min_rating ON ranked_tiers (min_rating);

INSERT INTO ranked_tiers (name, min_rating, color)
VALUES ('青铜', 0, '#b08d57'),
       ('白银', 1100, '#a8b4c0'),
       ('黄金', 1250, '#fab72a'),
       ('铂金', 1400, '#3fc1b0'),
       ('钻石', 1600, '#3ba6f5'),
       ('大师', 1800, '#ad74ff')
ON CONFLICT (min_rating) DO NOTHING;

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
    bid_currency   VARCHAR(32)   NOT NULL,                -- 出价和计算仓库价值的货币
    entry_cost     JSONB         NOT NULL DEFAULT '{}',   -- 入场代价 reward.Bundle, 数量为正数
    requirements   JSONB         NOT NULL DEFAULT '[]',   -- 入场条件 []condition.Condition
    warehouse      JSONB         NOT NULL DEFAULT '{}',   -- 仓库生成规则
    auction        JSONB         NOT NULL DEFAULT '{}',   -- 拍卖参数
    bots           JSONB         NOT NULL DEFAULT '{}',   -- 人机参数
    pve            JSONB         NOT NULL DEFAULT '{}',   -- 人机模式防刷参数
    consumables    JSONB         NOT NULL DEFAULT '{}',   -- 对局中使用道具的规则
    simulation     JSONB         NOT NULL DEFAULT '{}',   -- 保存时的模拟结果
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_venues_code ON venues (code);
CREATE INDEX IF NOT EXISTS idx_venues_deleted_at ON venues (deleted_at);

-- 暗拍的对局级数据, 与 matches 一一对应, 开局时创建; 模式取 matches.mode 中冒号之前的部分
CREATE TABLE IF NOT EXISTS auction_matches
(
    match_id     BIGINT PRIMARY KEY,
    venue_id     BIGINT      NOT NULL,
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
    id           BIGSERIAL PRIMARY KEY,
    match_id     BIGINT      NOT NULL,
    seat         SMALLINT    NOT NULL DEFAULT 0,
    user_id      BIGINT      NOT NULL DEFAULT 0,     -- 人机为 0
    is_practice  BOOLEAN     NOT NULL DEFAULT FALSE, -- 练习局, 不扣入场代价、不冻结资金、不发藏品、不写统计
    entry_cost   JSONB       NOT NULL DEFAULT '{}',  -- 实际扣除的入场代价
    carry        BIGINT      NOT NULL DEFAULT 0,     -- 带入资金, 出价的上限
    frozen       BIGINT      NOT NULL DEFAULT 0,     -- 冻结的带入资金
    estimate     BIGINT      NOT NULL DEFAULT 0,     -- 人机对仓库价值的估值
    bid          BIGINT      NOT NULL DEFAULT 0,     -- 当前出价
    bid_at       TIMESTAMPTZ,                        -- 当前出价的提交时间
    feedback     JSONB       NOT NULL DEFAULT '[]',  -- 每轮结束后的反馈
    profit       BIGINT      NOT NULL DEFAULT 0,     -- 结算后的盈亏
    rating_delta INTEGER     NOT NULL DEFAULT 0,     -- 本局段位分的变化, 只有排位成交时有
    rating       INTEGER     NOT NULL DEFAULT 0,     -- 结算后的段位分, 只有排位成交时有
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
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

-- 暗拍仓库里的藏品, 开局时快照名称、回收价和扫描时长
CREATE TABLE IF NOT EXISTS auction_warehouse_items
(
    id            BIGSERIAL PRIMARY KEY,
    match_id      BIGINT        NOT NULL,
    reveal_order  INTEGER       NOT NULL,           -- 公开品质的顺序, 从 1 开始, 第 N 轮公开前 N 件
    scan_order    INTEGER       NOT NULL DEFAULT 0, -- 开箱时扫描的顺序, 从 1 开始, 按位置从上到下、从左到右
    scan_ms       INTEGER       NOT NULL DEFAULT 0, -- 开箱时扫描这件的时长, 毫秒, 开局时按品质快照
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

-- 暗拍中用掉的道具, 结果在使用时算出并保存; 每人每轮同一种道具只用一次
CREATE TABLE IF NOT EXISTS auction_uses
(
    id         BIGSERIAL PRIMARY KEY,
    match_id   BIGINT      NOT NULL,
    seat       SMALLINT    NOT NULL,
    item_id    BIGINT      NOT NULL,
    round      INTEGER     NOT NULL,
    name       VARCHAR(64) NOT NULL,              -- 使用时的物品名称
    effect     VARCHAR(32) NOT NULL,              -- 使用时的效果代码
    params     JSONB       NOT NULL DEFAULT '{}', -- 使用时的效果参数
    result     JSONB       NOT NULL DEFAULT '{}', -- 得到的情报, 只给使用者看
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_auction_uses_match_id_seat_item_id_round ON auction_uses (match_id, seat, item_id, round);

CREATE TABLE IF NOT EXISTS auction_suspect_reviews
(
    user_a        BIGINT       NOT NULL,            -- 较小的用户 ID
    user_b        BIGINT       NOT NULL,            -- 较大的用户 ID
    last_match_id BIGINT       NOT NULL,            -- 处理时两人最近一起打的对局, 之后又一起打了排位时重新进入待处理
    note          VARCHAR(255) NOT NULL DEFAULT '',
    reviewed_by   BIGINT       NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_a, user_b)
);

-- 商品, 一份商品为一组货币和物品; 删除为软删除, 代码在已删除的商品中也唯一
CREATE TABLE IF NOT EXISTS shop_items
(
    id             BIGSERIAL PRIMARY KEY,
    code           VARCHAR(32)   NOT NULL,              -- 拼进限购计数的指标 shop_buy:<code>, 创建后不能修改
    name           VARCHAR(64)   NOT NULL,
    description    VARCHAR(255)  NOT NULL DEFAULT '',
    image_url      VARCHAR(1024) NOT NULL DEFAULT '',
    tab            VARCHAR(16)   NOT NULL DEFAULT '',   -- 页签, 为空时归入默认页签
    sort_order     INTEGER       NOT NULL DEFAULT 0,    -- 商城中从小到大排列
    goods          JSONB         NOT NULL DEFAULT '{}', -- 一份商品 reward.Bundle, 数量为正数
    price          JSONB         NOT NULL DEFAULT '{}', -- 一份的价格 reward.Bundle, 数量为正数, 为空时免费
    original_price JSONB         NOT NULL DEFAULT '{}', -- 原价, 只用于展示折扣
    requirements   JSONB         NOT NULL DEFAULT '[]', -- 购买条件 []condition.Condition
    limits         JSONB         NOT NULL DEFAULT '{}', -- 每人限购的份数: total、daily、season, 0 表示不限
    stock          BIGINT,                              -- 全服剩余库存, 为空时不限
    sold           BIGINT        NOT NULL DEFAULT 0,    -- 已售出的份数, 退款时扣回
    starts_at      TIMESTAMPTZ,                         -- 上架时间, 包含, 为空时不限
    ends_at        TIMESTAMPTZ,                         -- 下架时间, 不包含, 为空时不限
    is_enabled     BOOLEAN       NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ,
    CONSTRAINT chk_shop_items_stock CHECK (stock >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_shop_items_code ON shop_items (code);
CREATE INDEX IF NOT EXISTS idx_shop_items_deleted_at ON shop_items (deleted_at);

-- 订单, 记录下单时的商品名称、总价和全部商品
CREATE TABLE IF NOT EXISTS shop_orders
(
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT      NOT NULL,
    idempotency_key VARCHAR(64) NOT NULL,                -- 幂等键, 同一用户内唯一
    shop_item_id    BIGINT      NOT NULL,
    item_code       VARCHAR(32) NOT NULL,
    name            VARCHAR(64) NOT NULL,                -- 下单时的商品名称
    quantity        BIGINT      NOT NULL,                -- 购买的份数
    price           JSONB       NOT NULL DEFAULT '{}',   -- 实际支付的总价 reward.Bundle
    goods           JSONB       NOT NULL DEFAULT '{}',   -- 实际得到的全部商品 reward.Bundle
    status          VARCHAR(16) NOT NULL DEFAULT 'paid', -- paid 或 refunded
    refunded_at     TIMESTAMPTZ,
    refunded_by     BIGINT      NOT NULL DEFAULT 0,      -- 退款的管理员, 没有退款时为 0
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_shop_orders_quantity CHECK (quantity > 0)
);

CREATE INDEX IF NOT EXISTS idx_shop_orders_user_id ON shop_orders (user_id, id);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_shop_orders_idempotency_key ON shop_orders (user_id, idempotency_key);
CREATE INDEX IF NOT EXISTS idx_shop_orders_shop_item_id ON shop_orders (shop_item_id);

-- 上传的图片, 同一用途的相同图片只存一份, 没有被引用的定时清理
CREATE TABLE IF NOT EXISTS uploads
(
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT       NOT NULL,               -- 第一次上传的用户
    purpose      VARCHAR(16)  NOT NULL,               -- avatar / item / venue / shop
    key          VARCHAR(255) NOT NULL,               -- 存储中的 key, 如 item/ab/abcd….png
    content_type VARCHAR(32)  NOT NULL,               -- 保存后的格式
    size         BIGINT       NOT NULL,               -- 保存后的字节数
    width        INTEGER      NOT NULL,
    height       INTEGER      NOT NULL,
    sha256       VARCHAR(64)  NOT NULL,               -- 保存后内容的 sha256
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(), -- 上传时间, 重复上传时刷新, 清理按它计算保留天数
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_uploads_user_id ON uploads (user_id);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_uploads_key ON uploads (key) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_uploads_created_at ON uploads (created_at);
CREATE INDEX IF NOT EXISTS idx_uploads_deleted_at ON uploads (deleted_at);

-- 导入配置包的记录, 只增不改
CREATE TABLE IF NOT EXISTS imports
(
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT      NOT NULL,              -- 导入的管理员
    name       VARCHAR(64) NOT NULL,              -- 配置包的名称
    author     VARCHAR(64) NOT NULL DEFAULT '',
    version    INTEGER     NOT NULL,              -- 配置包的格式版本
    digest     VARCHAR(64) NOT NULL,              -- 文件的 sha256
    size       BIGINT      NOT NULL,              -- 文件大小, 字节
    created    INTEGER     NOT NULL DEFAULT 0,
    updated    INTEGER     NOT NULL DEFAULT 0,
    skipped    INTEGER     NOT NULL DEFAULT 0,    -- 跳过的项, 包括与已有配置相同的项
    entries    JSONB       NOT NULL DEFAULT '[]', -- 每一项的处理结果
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_imports_user_id ON imports (user_id);
CREATE INDEX IF NOT EXISTS idx_imports_created_at ON imports (created_at);

-- 游戏邮件, 每个收件人一行; 玩家删除和超出保留封数时为软删除, 来源单号仍然占用
CREATE TABLE IF NOT EXISTS mails
(
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT        NOT NULL,
    source      VARCHAR(16)   NOT NULL,             -- 来源, 如 admin / broadcast
    source_id   VARCHAR(64)   NOT NULL,             -- 来源单号, 同一来源和用户内唯一
    sender      VARCHAR(32)   NOT NULL,
    title       VARCHAR(64)   NOT NULL,
    content     VARCHAR(2000) NOT NULL DEFAULT '',
    attachments JSONB         NOT NULL DEFAULT '{}', -- 附件 reward.Bundle, 为空时没有附件
    read_at     TIMESTAMPTZ,
    claimed_at  TIMESTAMPTZ,                        -- 领取附件的时间
    expires_at  TIMESTAMPTZ   NOT NULL,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_mails_user_id ON mails (user_id, id);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_mails_source ON mails (source, source_id, user_id);
CREATE INDEX IF NOT EXISTS idx_mails_expires_at ON mails (expires_at);
CREATE INDEX IF NOT EXISTS idx_mails_deleted_at ON mails (deleted_at);

-- 全服邮件, 玩家打开信箱时按收件条件展开成 mails 中的一行
CREATE TABLE IF NOT EXISTS mail_broadcasts
(
    id                BIGSERIAL PRIMARY KEY,
    sender            VARCHAR(32)   NOT NULL,
    title             VARCHAR(64)   NOT NULL,
    content           VARCHAR(2000) NOT NULL DEFAULT '',
    attachments       JSONB         NOT NULL DEFAULT '{}', -- 附件 reward.Bundle
    requirements      JSONB         NOT NULL DEFAULT '[]', -- 收件条件, 需要全部满足
    registered_before TIMESTAMPTZ,                        -- 只发给这个时间之前注册的用户, 为空时不限
    expires_at        TIMESTAMPTZ   NOT NULL,             -- 展开的邮件同时过期
    idempotency_key   VARCHAR(64)   NOT NULL,             -- 幂等键, 重复提交只发送一次
    created_by        BIGINT        NOT NULL DEFAULT 0,
    revoked_at        TIMESTAMPTZ,
    revoked_by        BIGINT        NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mail_broadcasts_expires_at ON mail_broadcasts (expires_at);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_mail_broadcasts_idempotency_key ON mail_broadcasts (idempotency_key);

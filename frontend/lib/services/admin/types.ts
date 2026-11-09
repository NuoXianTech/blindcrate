import type {Quality} from '../../quality';
import type {Bundle} from '../auction';
import type {PageParams} from '../core';
import type {InventoryTransaction} from '../inventory';
import type {LeaderboardPeriod} from '../leaderboard';
import type {Tier} from '../ranked';
import type {Season, SeasonStatus} from '../season';
import type {ShopOrderStatus} from '../shop';
import type {AuctionRule, Grid, HostRule} from '../venue';
import type {WalletTransaction} from '../wallet';

/** 管理端看到的完整用户, 对应后端 auth.User */
export interface AdminUser {
  id: number;
  username: string;
  email: string;
  email_verified_at: string | null;
  nickname: string;
  avatar_url: string;
  is_disabled: boolean;
  is_admin: boolean;
  banned_until: string | null;
  ban_reason: string;
  last_login_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface ListUsersParams extends PageParams {
  /** 按用户名或邮箱前缀搜索 */
  keyword?: string;
  is_admin?: boolean;
  is_disabled?: boolean;
  is_banned?: boolean;
}

export interface ListUsersData {
  total: number;
  users: AdminUser[];
}

export interface UpdateUserRequest {
  nickname?: string;
  avatar_url?: string;
}

export type AuditScope = 'user' | 'admin';

export interface AuditLog {
  id: number;
  actor_id: number;
  /** 未登录或用户已不存在时为空 */
  actor_username: string;
  scope: AuditScope;
  action: string;
  target_type: string;
  target_id: number;
  detail: Record<string, unknown> | null;
  ip: string;
  user_agent: string;
  created_at: string;
}

export interface ListAuditLogsParams extends PageParams {
  actor_id?: number;
  scope?: AuditScope;
  action?: string;
  target_type?: string;
  target_id?: number;
  /** RFC3339, 包含 */
  start_time?: string;
  /** RFC3339, 不包含 */
  end_time?: string;
}

export interface ListAuditLogsData {
  total: number;
  logs: AuditLog[];
}

/** 货币定义, 对应后端 wallet.Currency */
export interface Currency {
  code: string;
  name: string;
  /** 展示用小数位, 金额统一按最小单位的整数存储 */
  decimals: number;
  sort_order: number;
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
}

/** 修改货币, 代码创建后不能修改; 已被使用的货币不能修改小数位, 基础货币不能停用 */
export type CurrencyRequest = Pick<Currency, 'name' | 'decimals' | 'sort_order' | 'is_enabled'>;

/** 新建货币 */
export type CreateCurrencyRequest = CurrencyRequest & Pick<Currency, 'code'>;

/** 物品定义, 对应后端 item.Item */
export interface Item {
  id: number;
  /** 英文别名, 可以重复, 物品以 ID 区分 */
  code: string;
  name: string;
  description: string;
  /** 品质等级, 见 useQualities */
  quality: number;
  /** 类别代码, 见 ItemCategory */
  category: string;
  image_url: string;
  sell_currency: string;
  /** 官方回收价, 最小单位; 0 表示不可卖出 */
  sell_price: number;
  /** 在仓库中占的格数 */
  width: number;
  height: number;
  /** 使用效果代码, 见 ItemEffect; 空表示不能使用, 藏品没有效果 */
  effect: string;
  /** 效果参数, 结构由效果决定 */
  effect_params: Record<string, string | number | boolean>;
  /** 下架后不会掉落, 也不能再发放, 已有的保留 */
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
}

/** 物品类别, 由后端代码注册, 对应后端 item.Category */
export interface ItemCategory {
  code: string;
  label: string;
}

/** 管理端品质列表项, 附带使用该品质的物品数 */
export interface AdminQuality extends Quality {
  /** 使用该品质的未删除物品数, 不为 0 时不能删除 */
  item_count: number;
  created_at: string;
  updated_at: string;
}

/** 修改品质, 等级和代码创建后不能修改 */
export type QualityRequest = Pick<Quality, 'name' | 'color' | 'effect' | 'scan_ms' | 'sort_order'>;

/** 新建品质 */
export type CreateQualityRequest = QualityRequest & Pick<Quality, 'level' | 'code'>;

/** 管理端物品列表项, 附带持有人数 */
export interface AdminItem extends Item {
  holder_count: number;
}

export interface ListItemsParams extends PageParams {
  /** 按代码或名称搜索, 不区分大小写 */
  keyword?: string;
  quality?: number;
  category?: string;
  is_enabled?: boolean;
}

export interface ListItemsData {
  total: number;
  items: AdminItem[];
}

/** 新建或修改物品, 修改时也需提交全部字段 */
export type ItemRequest = Pick<
  Item,
  | 'code'
  | 'name'
  | 'description'
  | 'quality'
  | 'category'
  | 'image_url'
  | 'sell_currency'
  | 'sell_price'
  | 'width'
  | 'height'
  | 'effect'
  | 'effect_params'
  | 'is_enabled'
>;

/** 物品的使用效果, 由后端代码注册, 对应后端 effect.TypeView */
export interface ItemEffect {
  code: string;
  /** 使用的场景, 如 match 对局中 */
  scene: string;
  /** 场景的展示名称, 如 "对局中" */
  scene_name: string;
  label: string;
  description: string;
  /** 参数的表单描述, 与条件类型相同 */
  fields: ConditionField[];
}

/** 设置项的值: 整数列表为数组 */
export type SettingValue = string | number | boolean | number[];

/** 下拉框的可选值 */
export interface SettingOption {
  value: string;
  label: string;
}

/** 设置项的描述, 后台据此生成表单, 对应后端 settings.Field */
export interface SettingField {
  key: string;
  label: string;
  help: string;
  type: 'string' | 'bool' | 'int' | 'int_list';
  input: 'text' | 'textarea' | 'password' | 'select' | 'switch' | 'number' | 'list';
  /** 下拉框的可选值, 引用字段为读取时的可选值 */
  options?: SettingOption[];
  /** 引用其他模块的数据, 如 currency 为启用的货币 */
  ref?: string;
  /** 整数的单位, bp 为基点, 按百分比输入 */
  unit?: 'bp' | 'second' | 'millisecond';
  /** 密钥, 读取时不返回明文 */
  secret: boolean;
  /** 会通过公开接口返回给前端 */
  public: boolean;
}

/** 设置组的分类 */
export interface SettingCategory {
  key: string;
  label: string;
}

/** 一个设置组, 对应后端 settings.GroupView */
export interface SettingGroup {
  key: string;
  label: string;
  description: string;
  category: string;
  fields: SettingField[];
  /** 密钥字段固定为空字符串 */
  value: Record<string, SettingValue>;
  /** 密钥字段是否已设置 */
  secrets_set: Record<string, boolean>;
  /** 无法解密的密钥字段, 需要重新填写或清空才能保存 */
  secrets_broken: string[] | null;
  /** 保存时原样提交, 未保存过为 0 */
  version: number;
  updated_by: number;
  /** 未保存过为空 */
  updated_at: string | null;
}

export interface ListSettingsData {
  categories: SettingCategory[];
  groups: SettingGroup[];
}

export interface UpdateSettingRequest {
  /** 读取时的版本, 已被别人修改时返回 409 */
  version: number;
  /** 只需提交改动的字段, 密钥字段留空表示不修改 */
  value: Record<string, SettingValue>;
  /** 需要清空的密钥字段 */
  clear_secrets?: string[];
}

/** 版本的来源: 保存、回滚、恢复默认值、启用版本记录前的内容、导入配置包 */
export type SettingRevisionAction = 'update' | 'rollback' | 'reset' | 'baseline' | 'import';

/** 一个字段的修改, 密钥字段只记录 <updated> 或 <cleared> */
export type SettingChange = { old: SettingValue; new: SettingValue } | '<updated>' | '<cleared>';

/** 设置组的一个版本, 对应后端 admin.SettingRevisionItem */
export interface SettingRevision {
  version: number;
  action: SettingRevisionAction;
  /** 回滚时为回滚到的版本 */
  from_version: number;
  changes: Record<string, SettingChange>;
  updated_by: number;
  /** 系统操作或用户已不存在时为空 */
  updated_by_username: string;
  created_at: string;
}

export interface ListSettingRevisionsParams {
  current: number;
  size: number;
}

export interface ListSettingRevisionsData {
  total: number;
  revisions: SettingRevision[];
}

/** 一个版本的快照, 对应后端 settings.RevisionView */
export interface SettingRevisionDetail extends Omit<SettingRevision, 'updated_by_username'> {
  key: string;
  value: Record<string, SettingValue>;
  secrets_set: Record<string, boolean>;
  /** 无法解密的密钥字段, 有这类字段时不能回滚到此版本 */
  secrets_broken: string[] | null;
}

export interface AdjustBalanceRequest {
  currency: string;
  /** 最小单位, 正数加款, 负数扣款, 扣款后余额不能为负 */
  amount: number;
  /** 调账原因, 可以为空, 用户可在流水中看到 */
  remark: string;
  /** 每次打开表单生成一个, 重复提交只生效一次 */
  idempotency_key: string;
}

export interface AdjustInventoryRequest {
  item_id: number;
  /** 正数发放, 负数扣除, 扣除后数量不能为负 */
  quantity: number;
  remark: string;
  idempotency_key: string;
}

/** 流水查询共用的筛选条件 */
interface TransactionFilters extends PageParams {
  user_id?: number;
  operator_id?: number;
  biz_type?: string;
  biz_id?: string;
  /** RFC3339, 包含 */
  start_time?: string;
  /** RFC3339, 不包含 */
  end_time?: string;
}

/** 流水的用户和操作人, 用户已不存在或系统操作时用户名为空 */
interface TransactionActors {
  user_id: number;
  username: string;
  operator_id: number;
  operator_username: string;
  idempotency_key: string;
}

/** 全部用户的余额流水, 对应后端 admin.WalletTransactionItem */
export type AdminWalletTransaction = WalletTransaction & TransactionActors;

export interface ListAdminWalletTransactionsParams extends TransactionFilters {
  currency?: string;
}

export interface ListAdminWalletTransactionsData {
  total: number;
  transactions: AdminWalletTransaction[];
}

/** 全部用户的背包流水, 对应后端 admin.InventoryTransactionItem */
export type AdminInventoryTransaction = InventoryTransaction & TransactionActors;

export interface ListAdminInventoryTransactionsParams extends TransactionFilters {
  item_id?: number;
}

export interface ListAdminInventoryTransactionsData {
  total: number;
  transactions: AdminInventoryTransaction[];
}

/** 掉落池, 对应后端 loot.Pool */
export interface LootPool {
  id: number;
  /** 玩法中引用的代码, 小写字母、数字和下划线 */
  code: string;
  name: string;
  description: string;
  /** 停用后不能再抽取 */
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
}

/** 掉落池列表项, 附带条目数 */
export interface LootPoolListItem extends LootPool {
  entry_count: number;
}

export interface ListLootPoolsParams extends PageParams {
  /** 按代码或名称搜索, 不区分大小写 */
  keyword?: string;
  is_enabled?: boolean;
}

export interface ListLootPoolsData {
  total: number;
  pools: LootPoolListItem[];
}

export type LootPoolRequest = Pick<LootPool, 'code' | 'name' | 'description' | 'is_enabled'>;

/** 掉落池条目, 对应后端 loot.EntryDetail */
export interface LootEntry {
  id: number;
  pool_id: number;
  /** 掉落的物品, 与 currency 二选一 */
  item_id: number;
  currency: string;
  /** 抽中的概率为权重 / 池中可抽取条目的权重之和 */
  weight: number;
  /** 每次掉落的数量范围, 货币为金额 */
  min_quantity: number;
  max_quantity: number;
  created_at: string;
  /** 物品或货币的名称, 已删除的物品为占位名称 */
  name: string;
  /** 物品品质, 货币为 0 */
  quality: number;
  image_url: string;
  /** 物品已上架或货币已启用, 否则不参与抽取 */
  is_available: boolean;
  /** 当前每次抽中的概率, 不参与抽取时为 0 */
  probability: number;
}

export type LootEntryRequest = Pick<LootEntry, 'item_id' | 'currency' | 'weight' | 'min_quantity' | 'max_quantity'>;

export interface LootPoolEntriesData {
  pool: LootPool;
  /** 按条目 ID 排序, 包含不参与抽取的条目 */
  entries: LootEntry[];
}

/** 一个条件, 对应后端 condition.Condition */
export interface Condition {
  type: string;
  /** 参数, 结构由条件类型决定 */
  params: Record<string, string | number | boolean>;
}

/** 条件参数的描述, 后台据此生成表单, 对应后端 condition.Field */
export interface ConditionField {
  key: string;
  label: string;
  help: string;
  type: 'string' | 'bool' | 'int';
  /** text / select / switch / number, 或选择器 currency / item / quality / scope / metric */
  input: 'text' | 'select' | 'switch' | 'number' | 'currency' | 'item' | 'quality' | 'scope' | 'metric';
  /** 下拉框 select 和选择器 metric 的可选值 */
  options?: ConditionOption[];
  /** 整数的单位, bp 为基点, 按百分比输入 */
  unit?: 'bp';
}

/** 条件参数选择器的一个可选值 */
export interface ConditionOption {
  value: string;
  label: string;
}

/** 条件类型, 对应后端 condition.TypeView */
export interface ConditionType {
  type: string;
  label: string;
  description: string;
  fields: ConditionField[];
}

/** 仓库的一个档位, 从掉落池抽取若干次 */
export interface WarehouseSlot {
  /** 掉落池代码, 只能有物品条目 */
  pool: string;
  /** 抽取次数在 [min, max] 内均匀随机 */
  min: number;
  max: number;
  /** 这一档出现的概率, 基点 */
  chance: number;
}

/** 仓库的一档数量波动, 对应后端 venue.Swing */
export interface WarehouseSwing {
  /** 抽中这一档时按它缩放各档位的抽取次数, 基点 */
  factor: number;
  weight: number;
}

/** 仓库生成规则, 对应后端 venue.WarehouseRule */
export interface WarehouseRule {
  grid: Grid;
  slots: WarehouseSlot[];
  /** 每局按权重抽一档, 为空表示不波动; 旧数据可能为 null */
  swings?: WarehouseSwing[] | null;
  /** 摆不下时重抽, 超过后开局失败 */
  max_rerolls: number;
}

/** 档次的期望价值和推导出的金额, 保存时推导, 对应后端 venue.TierAuction */
export interface TierAuction {
  /** 这个档次的期望价值, 排位模式 */
  mean: number;
  start_price: number;
  speed_floor: number;
  intel_price: number;
}

/** 拍品的一种档次, 对应后端 venue.Tier */
export interface LotTier {
  /** 档次中唯一 */
  code: string;
  /** 只在管理端和拍场结束后显示 */
  name: string;
  /** 1-10000 */
  weight: number;
  warehouse: WarehouseRule;
  /** 保存时推导, 提交的值会被覆盖 */
  auction: TierAuction;
}

/**
 * 拍场, 对应后端 venue.LotRule: 一局依次开拍 [min, max] 个拍品, 共用一笔带入资金, 每个拍品按权重抽一种档次
 * 没有配置档次时只有 warehouse 一档, 一局一个仓库
 */
export interface LotRule {
  /** 配置了档次时为 1-8, 不超过 max; 没有档次时忽略 */
  min: number;
  max: number;
  /** 最多 6 种; 旧数据可能为 null */
  tiers: LotTier[] | null;
  /** 一个拍品开箱后停留多久进入下一个, 0-30 秒 */
  gap_seconds: number;
  /** 带入覆盖率, 基点; 0 表示用玩法平衡设置的默认值 */
  coverage: number;
}

/** 提交的拍场, 档次的金额由后端推导, 不需要提交 */
export type LotRequest = Omit<LotRule, 'tiers'> & {
  tiers: Omit<LotTier, 'auction'>[];
};

/** 地图上一种出场的人机风格和它的权重, 对应后端 bot.Weight */
export interface BotWeight {
  /** 风格代码 */
  persona: string;
  /** 1-100 */
  weight: number;
}

/** 每个模式出场的人机风格, 开局时每个人机按权重随机选一种, 对应后端 venue.BotRule */
export interface BotRule {
  /** 开放的模式至少配置一种, 最多 10 种, 不能重复 */
  pve: BotWeight[];
  ranked: BotWeight[];
}

/** 一种模式中使用道具的次数限制, 对应后端 venue.ConsumableLimit */
export interface ConsumableLimit {
  /** 每局最多使用几个道具, 0 表示这个模式不能用 */
  max_per_match: number;
}

/** 对局中使用道具的规则, 只有效果属于对局场景的物品可以用, 对应后端 venue.ConsumableRule */
export interface ConsumableRule {
  pve: ConsumableLimit;
  ranked: ConsumableLimit;
  /** 可以用的物品 ID, 为空表示所有对局中可用的道具 */
  allowed: number[];
}

/** 人机模式的防刷参数, 对应后端 venue.PveRule */
export interface PveRule {
  practice_only: boolean;
  /** 当天在这张地图人机模式的净收益上限, 0 表示不单独限制 */
  daily_profit_cap: number;
  /** 按这个系数缩放各档位的抽取次数, 基点 */
  value_factor: number;
}

/** 模拟仓库的价值分布, 价值为出价货币的回收价之和 */
export interface ValueStats {
  /** 摆放失败率, 基点 */
  failure_rate: number;
  mean: number;
  p10: number;
  p50: number;
  p90: number;
  max: number;
  /** 变异系数, 标准差 ÷ 平均值, 基点 */
  cv: number;
  mean_pieces: number;
}

/** 验算的阵容, 对应后端 venue 的 Squad */
export type Squad = 'humans' | 'ranked' | 'pve';

/** 一种阵容的验算结果, 金额为出价货币的最小单位, 比例为基点, 对应后端 venue.SquadStats */
export interface SquadStats {
  squad: Squad;
  /** 真人数, 其余为人机 */
  humans: number;
  /** 成交率 */
  sold: number;
  /** 成交价之和 ÷ 仓库价值之和 */
  price_value: number;
  /** 速胜率, 最后一轮之前成交的比例 */
  speed_win: number;
  /** 出价离散度 */
  bid_cv: number;
  /** 最贵一件占仓库价值的比例 */
  top_share: number;
  /** 人机估值之和 ÷ 仓库价值之和, 没有人机时为 0 */
  appraise_bias: number;
  /** 平均在第几轮结束, 百分之一轮 */
  avg_rounds: number;
  /** 每个真人席位每个拍品的成交率, 一局一个拍品时就是每局的成交率 */
  win_rate: number;
  /** 每个真人席位每局(拍场)的平均盈亏 */
  seat_ev: number;
  /** 每局平均的拍品数, 百分之一个; 旧数据没有 */
  lots?: number;
  /** 每个真人席位拿到拍场第 1 名(盈亏最高且赚钱)的比例; 多拍品时胜率闸门看它 */
  rank_win?: number;
  /** 拍场结束时剩余资金不够最低起拍价的真人席位比例 */
  budget_out?: number;
  /** 会规划预算的模拟真人比不规划的每局多赚多少, 只有两种都有时有 */
  plan_gain?: number;
  /** 成交的拍品在最后一轮前的最高价 ÷ 成交价的平均值 */
  early_share?: number;
  /** 真人成交时的平均盈亏, 没成交时只亏入场费, 流拍时不亏 */
  winner_ev: number;
  /** 每轮结束后热度火热(不低于 70)的比例, 基点; 加轮和成交的那一轮不算 */
  hype_hot: number;
  /** 同上, 冷清(不高于 30)的比例 */
  hype_cold: number;
  /** 结束时热度的平均值, 0-100 */
  hype_end: number;
}

/** 验算闸门, 对应后端 venue 的 Check */
export type CheckCode =
  | 'rounds'
  | 'grid_width'
  | 'grid_height'
  | 'top_share'
  | 'price_value'
  | 'bid_cv'
  | 'speed_win'
  | 'appraise_bias'
  | 'seat_ev'
  | 'win_rate';

/** 验算的一项闸门, 对应后端 venue.Check */
export interface BalanceCheck {
  code: CheckCode;
  /** 按阵容判定时为阵容 */
  squad?: Squad;
  value: number;
  /** 下限, 为空表示不限 */
  min: number | null;
  /** 上限, 为空表示不限 */
  max: number | null;
  /** count 个数, bp 基点, amount 出价货币的最小单位 */
  unit: 'count' | 'bp' | 'amount';
  passed: boolean;
}

/** 对局验算, 对应后端 venue.Balance */
export interface Balance {
  /** 每种阵容模拟的局数 */
  runs: number;
  /** 成交价 ÷ 仓库价值的目标, 基点 */
  target: number;
  /** 参考阵容在前 */
  squads: SquadStats[];
  checks: BalanceCheck[];
  /** 全部闸门通过, 未通过的地图不能启用 */
  passed: boolean;
  /** 推导和验算时玩法平衡设置的版本, 与当前版本不同时需要重新推导 */
  version: number;
}

/** 一种档次的价值分布, 对应后端 venue.TierStats */
export interface TierStats {
  code: string;
  ranked: ValueStats;
  pve: ValueStats;
}

/** 仓库模拟结果, 对应后端 venue.Simulation */
export interface Simulation {
  runs: number;
  /** 排位模式, 不缩放抽取次数; 配置了档次时为单个拍品(按权重混合各档次)的分布 */
  ranked: ValueStats;
  /** 人机模式, 按仓库价值系数缩放抽取次数 */
  pve: ValueStats;
  /** 各档次的价值分布, 没有配置档次时为空; 旧数据没有 */
  tiers?: TierStats[] | null;
  /** 对局验算, 旧数据为空 */
  balance: Balance | null;
  simulated_at: string;
}

/** 试算结果, 对应后端 venueadmin.SimulateResponseData */
export interface SimulateResult {
  simulation: Simulation;
  /** 按模拟结果推导出的拍卖参数, 保存时相同 */
  auction: AuctionRule;
}

/** 管理端看到的完整地图, 对应后端 venue.Venue */
export interface AdminVenue {
  id: number;
  /** 拼进玩法代码和统计范围, 创建后不能修改 */
  code: string;
  name: string;
  description: string;
  image_url: string;
  /** 大厅中从小到大排列 */
  sort_order: number;
  /** 停用后不能加入队列, 进行中的对局照常打完 */
  is_enabled: boolean;
  pve_enabled: boolean;
  ranked_enabled: boolean;
  bid_currency: string;
  /** 入场代价, 数量为正数, 开局时扣除; 不能有出价货币, 出价货币的入场费由对局验算求出 */
  entry_cost: Bundle;
  /** 入场条件, 需要全部满足 */
  requirements: Condition[];
  /** 仓库生成规则, 配置了档次时不用 */
  warehouse: WarehouseRule;
  /** 拍场: 一局开拍几个仓库, 从哪些档次中抽; 旧数据可能没有 */
  lots?: LotRule;
  auction: AuctionRule;
  bots: BotRule;
  pve: PveRule;
  /** 对局中使用道具的规则 */
  consumables: ConsumableRule;
  /** 主持人的提示和情报 */
  host: HostRule;
  /** 保存时的模拟结果 */
  simulation: Simulation;
  created_at: string;
  updated_at: string;
}

export interface ListVenuesParams extends PageParams {
  /** 按代码或名称搜索, 不区分大小写 */
  keyword?: string;
  is_enabled?: boolean;
}

export interface ListVenuesData {
  total: number;
  venues: AdminVenue[];
  /** 当前玩法平衡设置的版本 */
  balance_version: number;
}

/** 一个地图的重新推导结果 */
export interface RecalibrateResult {
  id: number;
  code: string;
  name: string;
  /** 是否已按当前设置保存 */
  updated: boolean;
  /** 未保存的原因, 一般是启用的地图验算未通过 */
  reason: string;
}

export interface RecalibrateData {
  /** 用到的玩法平衡设置版本 */
  version: number;
  results: RecalibrateResult[];
}

/** 管理员配置的拍卖参数, 金额和比例由后端按模拟结果推导 */
export type AuctionSettings = Pick<AuctionRule, 'rounds' | 'round_seconds' | 'reveal_seconds' | 'quick_win' | 'max_extra_rounds'>;

/** 新建、修改或模拟地图, 修改时也需提交全部字段; 拍卖的金额和比例不需要提交, 提交了也会被覆盖 */
export type VenueRequest = Omit<AdminVenue, 'id' | 'lots' | 'auction' | 'simulation' | 'created_at' | 'updated_at'> & {
  lots: LotRequest;
  auction: AuctionSettings;
};

/** 榜单, 对应后端 leaderboard.Board */
export interface AdminLeaderboard {
  id: number;
  /** 自然键, 小写字母、数字和下划线 */
  code: string;
  name: string;
  description: string;
  /** 排名的指标, 如 wins、profit:chip */
  metric: string;
  /** 统计范围, 如 all、mode:ranked */
  scope: string;
  period: LeaderboardPeriod;
  /** 展示前多少名, 10-500 */
  size: number;
  /** 同一范围和周期内至少完成多少局才上榜 */
  min_matches: number;
  sort_order: number;
  /** 停用后玩家看不到 */
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface ListLeaderboardsParams extends PageParams {
  /** 按代码或名称搜索, 不区分大小写 */
  keyword?: string;
  is_enabled?: boolean;
}

export interface ListLeaderboardsData {
  total: number;
  boards: AdminLeaderboard[];
}

/** 新建或修改榜单, 修改时也需提交全部字段 */
export type LeaderboardRequest = Omit<AdminLeaderboard, 'id' | 'created_at' | 'updated_at'>;

export interface ListSeasonsParams extends PageParams {
  /** 按代码或名称搜索, 不区分大小写 */
  keyword?: string;
  status?: SeasonStatus;
}

export interface ListSeasonsData {
  total: number;
  seasons: Season[];
}

/** 修改赛季, 需提交全部字段; 时间精确到分钟, 开始后不能修改开始时间和段位分继承比例, 结束后只能修改名称 */
export type SeasonRequest = Pick<Season, 'name' | 'starts_at' | 'ends_at' | 'rating_carry' | 'rewards'>;

/** 新建赛季 */
export type CreateSeasonRequest = SeasonRequest & Pick<Season, 'code'>;

/** 新建或修改段位, 修改时也需提交全部字段 */
export type TierRequest = Pick<Tier, 'name' | 'min_rating' | 'color'>;

/** 可以排名的指标, 对应后端 stats.Metric */
export interface StatMetric {
  key: string;
  label: string;
}

/** 管理员看到的座位, 对应后端 match.PlayerDetail */
export interface AdminMatchPlayer {
  id: number;
  match_id: number;
  seat: number;
  /** 人机为 0 */
  user_id: number;
  /** 人机的昵称, 真人为空 */
  bot_name: string;
  /** 结束时的名次, 0 表示没有名次 */
  rank: number;
  created_at: string;
  is_bot: boolean;
  /** 真人的昵称, 未设置时为用户名; 人机为人机的昵称 */
  nickname: string;
  avatar_url: string;
}

/** 管理员看到的对局, 对应后端 match.MatchDetail */
export interface AdminMatch {
  id: number;
  mode: string;
  status: string;
  phase: string;
  round: number;
  /** 当前阶段的截止时间, 不限时或已结束时为空 */
  phase_ends_at: string | null;
  version: number;
  /** 玩家可以离开的时间, 为空表示还没释放 */
  released_at: string | null;
  created_at: string;
  updated_at: string;
  finished_at: string | null;
  /** 按座位号排序 */
  players: AdminMatchPlayer[];
}

export interface ListAdminMatchesParams extends PageParams {
  /** 玩法代码冒号前的部分, 如 pve */
  prefix?: string;
  /** 玩法代码冒号后的部分, 如地图代码 */
  suffix?: string;
  status?: string;
  /** 只看该用户参与的对局 */
  user_id?: number;
  /** 只看该用户也参与的对局, 与 user_id 一起用于查看两个人一起打的对局 */
  other_id?: number;
}

export interface ListAdminMatchesData {
  total: number;
  matches: AdminMatch[];
}

/** 可疑名单中的一个账号, 对应后端 auction.SuspectUser */
export interface SuspectUser {
  id: number;
  username: string;
  nickname: string;
  is_disabled: boolean;
  banned_until: string | null;
}

/** 一对可疑账号的处理记录, 对应后端 auction.SuspectReview */
export interface SuspectReview {
  user_a: number;
  user_b: number;
  /** 处理时两人最近一起打的对局 */
  last_match_id: number;
  note: string;
  reviewed_by: number;
  created_at: string;
  updated_at: string;
}

/** 可疑名单中的一对账号, 对应后端 auction.Suspect */
export interface Suspect {
  /** 用户 ID 较小的一个 */
  user_a: SuspectUser;
  user_b: SuspectUser;
  /** 一起打的排位局数 */
  matches: number;
  /** 其中一人成交的拍品数 */
  wins: number;
  /** 其中一人低价成交的拍品数 */
  low_wins: number;
  /** 其中一人成交时成交价占仓库价值的平均比例, 基点; 没有成交时为 0 */
  price_rate: number;
  last_match_id: number;
  last_match_at: string;
  /** 最近一次处理记录, 没处理过时为空 */
  review: SuspectReview | null;
}

export interface ListSuspectsParams extends PageParams {
  /** 不传返回全部; false 只返回待处理的, 即没处理过或处理后又一起打过排位; true 只返回其余的 */
  reviewed?: boolean;
}

export interface ListSuspectsData {
  total: number;
  suspects: Suspect[];
}

/** 把一对可疑账号标记为已处理 */
export interface ReviewSuspectRequest {
  user_a: number;
  /** 大于 user_a */
  user_b: number;
  /** 处理到的对局, 即名单中的 last_match_id */
  last_match_id: number;
  note: string;
}

/** 每人限购的份数, 0 为不限 */
export interface ShopLimits {
  /** 累计 */
  total: number;
  /** 每天, 按服务器时区 */
  daily: number;
  /** 每赛季, 没有进行中的赛季时不能购买 */
  season: number;
}

/** 商品, 对应后端 shop.Item */
export interface AdminShopItem {
  id: number;
  /** 拼进限购计数的指标, 创建后不能修改 */
  code: string;
  name: string;
  description: string;
  image_url: string;
  /** 商城中的页签, 为空时归入默认页签 */
  tab: string;
  /** 商城中从小到大排列 */
  sort_order: number;
  /** 一份商品, 不能为空 */
  goods: Bundle;
  /** 一份的价格, 为空时免费 */
  price: Bundle;
  /** 原价, 只用于展示折扣 */
  original_price: Bundle;
  /** 购买条件, 需要全部满足 */
  requirements: Condition[];
  limits: ShopLimits;
  /** 全服剩余库存, 为空时不限 */
  stock: number | null;
  /** 已售出的份数, 退款时减去 */
  sold: number;
  /** 上架时间, 包含, 为空时不限 */
  starts_at: string | null;
  /** 下架时间, 不包含, 为空时不限 */
  ends_at: string | null;
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface ListShopItemsParams extends PageParams {
  /** 按代码或名称搜索, 不区分大小写 */
  keyword?: string;
  tab?: string;
  is_enabled?: boolean;
}

export interface ListShopItemsData {
  total: number;
  items: AdminShopItem[];
  /** 未删除的商品用到的页签, 按名称排列, 不含空页签 */
  tabs: string[];
}

/** 新建或修改商品, 修改时也需提交全部字段, 代码不能修改; 时间精确到分钟 */
export type ShopItemRequest = Omit<AdminShopItem, 'id' | 'sold' | 'created_at' | 'updated_at'>;

/** 订单, 对应后端 admin.ShopOrderItem */
export interface AdminShopOrder {
  id: number;
  user_id: number;
  /** 用户已不存在时为空 */
  username: string;
  shop_item_id: number;
  /** 下单时的商品代码和名称 */
  item_code: string;
  name: string;
  quantity: number;
  /** 实际支付的总价 */
  price: Bundle;
  /** 实际得到的全部商品 */
  goods: Bundle;
  status: ShopOrderStatus;
  refunded_at: string | null;
  /** 退款的管理员, 没有退款时为 0 */
  refunded_by: number;
  /** 没有退款时为空 */
  refunded_username: string;
  created_at: string;
}

export interface ListShopOrdersParams extends PageParams {
  user_id?: number;
  shop_item_id?: number;
  status?: ShopOrderStatus;
  /** RFC3339, 包含 */
  start_time?: string;
  /** RFC3339, 不包含 */
  end_time?: string;
}

export interface ListShopOrdersData {
  total: number;
  orders: AdminShopOrder[];
}

/** 配置包中的配置种类代码, 也是包中的文件名; 由后端各模块注册, 列表见 PackSectionInfo */
export type PackSection = string;

/** 一种配置, 对应后端 pack.SectionView; 列表按依赖排列, 导入时按这个顺序写入 */
export interface PackSectionInfo {
  code: PackSection;
  label: string;
  /** 导出时的说明 */
  hint: string;
  /** 是否可以改名导入 */
  renamable: boolean;
}

/** 一项配置与已有配置的比较结果: 新增、覆盖、相同、冲突 */
export type PackStatus = 'new' | 'changed' | 'same' | 'conflict';

/** 一项配置的处理方式: 新增或覆盖、跳过、以新代码导入 */
export type PackActionType = 'overwrite' | 'skip' | 'rename';

export interface ExportPackRequest {
  name: string;
  author: string;
  sections: PackSection[];
}

/** 配置包的说明, 对应后端 admin.PackManifest */
export interface PackManifest {
  format: string;
  /** 格式版本 */
  version: number;
  name: string;
  author: string;
  exported_at: string;
  sections: PackSection[];
}

/** 一项配置的处理结果, 对应后端 admin.PackEntryResult */
export interface PackEntryResult {
  section: PackSection;
  /** 包中的代码, 品质为等级, 物品为 ID, 设置为设置组的 key */
  key: string;
  name: string;
  /** 跳过的项为与已有配置比较的结果 */
  status: PackStatus;
  action: PackActionType;
  /** 改名导入时的新代码 */
  code?: string;
  /** 无法导入的原因 */
  reason?: string;
}

/** 一项配置的处理方式, 没有提交的项按覆盖处理 */
export interface PackAction {
  section: PackSection;
  key: string;
  action: PackActionType;
  code: string;
}

export interface PackPreview {
  manifest: PackManifest;
  /** 文件的 sha256 */
  digest: string;
  size: number;
  entries: PackEntryResult[];
}

/** 一次导入的记录, 对应后端 admin.Import */
export interface PackImport {
  id: number;
  user_id: number;
  name: string;
  author: string;
  version: number;
  digest: string;
  size: number;
  created: number;
  updated: number;
  /** 包括与已有配置相同的项 */
  skipped: number;
  /** 每一项的处理结果, 列表中没有 */
  entries?: PackEntryResult[];
  created_at: string;
}

export type ListPackImportsParams = PageParams;

export interface ListPackImportsData {
  total: number;
  imports: PackImport[];
}

import type {Bundle} from '../auction';
import type {PageParams} from '../core';
import type {InventoryTransaction} from '../inventory';
import type {LeaderboardPeriod} from '../leaderboard';
import type {AuctionRule, Grid} from '../venue';
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
  username?: string;
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

/** 物品定义, 对应后端 item.Item */
export interface Item {
  id: number;
  name: string;
  description: string;
  /** 1-7, 见 lib/quality.ts */
  quality: number;
  image_url: string;
  sell_currency: string;
  /** 官方回收价, 最小单位; 0 表示不可卖出 */
  sell_price: number;
  /** 在仓库中占的格数 */
  width: number;
  height: number;
  /** 下架后不会掉落, 也不能再发放, 已有的保留 */
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
}

/** 管理端物品列表项, 附带持有人数 */
export interface AdminItem extends Item {
  holder_count: number;
}

export interface ListItemsParams extends PageParams {
  /** 按名称搜索, 不区分大小写 */
  keyword?: string;
  quality?: number;
  is_enabled?: boolean;
}

export interface ListItemsData {
  total: number;
  items: AdminItem[];
}

/** 新建或修改物品, 修改时也需提交全部字段 */
export type ItemRequest = Pick<
  Item,
  'name' | 'description' | 'quality' | 'image_url' | 'sell_currency' | 'sell_price' | 'width' | 'height' | 'is_enabled'
>;

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
  /** 下拉框的可选值 */
  options?: SettingOption[];
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

/** 版本的来源: 保存、回滚、恢复默认值、启用版本记录前的内容 */
export type SettingRevisionAction = 'update' | 'rollback' | 'reset' | 'baseline';

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
  /** 调账原因, 用户可在流水中看到 */
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
  /** text / switch / number, 或选择器 currency / item / scope / metric */
  input: 'text' | 'switch' | 'number' | 'currency' | 'item' | 'scope' | 'metric';
  /** 选择器 metric 的可选值 */
  options?: ConditionOption[];
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

/** 仓库生成规则, 对应后端 venue.WarehouseRule */
export interface WarehouseRule {
  grid: Grid;
  slots: WarehouseSlot[];
  /** 摆不下时重抽, 超过后开局失败 */
  max_rerolls: number;
}

/** 人机参数, 系数都是基点, 对应后端 venue.BotRule */
export interface BotRule {
  /** 低、中、高三个等级的权重 */
  level_weights: number[];
  /** 低、中、高三个等级的估值误差 */
  estimate_errors: number[];
  /** 每轮出价 = 估值 × 系数, 轮数更多时沿用最后一个 */
  round_factors: number[];
  jitter: number;
  max_factor: number;
  bluff_chance: number;
  /** 只在前几轮诈唬 */
  bluff_rounds: number;
  bluff_factor: number;
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
  mean_pieces: number;
}

/** 仓库模拟结果, 对应后端 venue.Simulation */
export interface Simulation {
  runs: number;
  /** 排位模式, 不缩放抽取次数 */
  ranked: ValueStats;
  /** 人机模式, 按仓库价值系数缩放抽取次数 */
  pve: ValueStats;
  simulated_at: string;
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
  /** 入场代价, 数量为正数, 开局时扣除 */
  entry_cost: Bundle;
  /** 入场条件, 需要全部满足 */
  requirements: Condition[];
  warehouse: WarehouseRule;
  auction: AuctionRule;
  bots: BotRule;
  pve: PveRule;
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
}

/** 新建、修改或模拟地图, 修改时也需提交全部字段 */
export type VenueRequest = Omit<AdminVenue, 'id' | 'simulation' | 'created_at' | 'updated_at'>;

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

/** 可以排名的指标, 对应后端 stats.Metric */
export interface StatMetric {
  key: string;
  label: string;
}

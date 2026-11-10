import type {Condition} from '../condition';
import type {PageParams} from '../core';
import type {Bundle} from '../reward';

/** 入场代价的一行, 对应后端 venue.CostLine */
export interface CostLine {
  /** 货币代码, 与 item_id 二选一 */
  currency?: string;
  item_id?: number;
  name: string;
  image_url: string;
  /** 货币金额或物品数量 */
  amount: number;
}

/** 入场条件的达成情况, 对应后端 condition.Progress */
export interface RequirementProgress {
  type: string;
  met: boolean;
  /** 当前值, 没有进度的条件为 0 */
  current: number;
  /** 目标值, 没有进度的条件为 0 */
  target: number;
  /** 展示给玩家的描述, 如 "完成 10 局排位(3/10)" */
  text: string;
}

/** 仓库价值区间, 保存地图时模拟得到 */
export interface ValueRange {
  p10: number;
  p50: number;
  p90: number;
}

/** 仓库的格子 */
export interface Grid {
  width: number;
  height: number;
}

/** 拍场中一种格子大小的拍品, 对应后端 venue.LotGrid */
export interface LotGrid {
  grid: Grid;
  /** 拍品是这种格子大小的概率, 基点 */
  share: number;
  start_price: number;
  speed_floor: number;
}

/** 玩家看到的拍场: 拍品数和每种格子大小, 档次不公开, 对应后端 venue.LotsView */
export interface LotsView {
  min: number;
  max: number;
  /** 拍品开箱后停留多久进入下一个, 秒 */
  gap_seconds: number;
  grids: LotGrid[];
}

/**
 * 拍卖参数, 金额按出价货币的最小单位, 对应后端 venue.AuctionRule
 *
 * 管理员只配置轮次、时长和速胜倍率; 金额和比例由保存时模拟的仓库价值分布推导, 旧数据中为 0
 */
export interface AuctionRule {
  rounds: number;
  round_seconds: number;
  reveal_seconds: number;
  /** 前 rounds-1 轮的速胜倍率, 基点; 第一名不低于第二名的这么多倍且不低于速胜下限时直接成交 */
  quick_win: number[];
  max_extra_rounds: number;
  /** 出价货币的入场费, 开局时与入场代价一起扣除, 流拍时退还; 由对局验算求出 */
  entry_fee: number;
  /** 起拍价; 配置了档次时为各档次中最低的, 也是最低带入, 每个拍品按档次的起拍价 */
  start_price: number;
  /** 带入资金上限, 玩家在 [start_price, max_carry] 内自选带入金额, 整个拍场共用 */
  max_carry: number;
  /** 人机的带入资金, 不超过带入资金上限; 旧地图为 0, 按带入资金上限 */
  bot_carry: number;
  /** 速胜时第一名的出价不能低于它 */
  speed_floor: number;
  /** 领先折让, 基点: 成交价 = 第二名的出价 + 领先部分 × (1 - 折让) */
  lead_discount: number;
  /** 首轮扫描比例, 基点: 第 1 轮每件藏品按这个概率扫出轮廓, 至少一件; 其余直到开箱都不公开 */
  preview_rate: number;
}

/**
 * 主持人, 对应后端 venue.HostRule: 每轮开始时公开一条关于仓库的提示, 有一定概率说错
 */
export interface HostRule {
  /** 关闭时没有提示和热度 */
  enabled: boolean;
  /** 开局时从中随机选一种性格, 为空表示全部; 见 lib/services/auction 的 HOST_PERSONA */
  personas: string[];
  /** 提示说错的概率, 基点, 不超过 2500; 按性格和热度调整 */
  lie_rate: number;
  /** 提示涉及的品质不低于它时撒谎率 × 1.5, 0 表示不区分; 也是"高品质件数"提示统计的最低品质 */
  high_quality: number;
}

/** 玩家看到的人机模式参数 */
export interface PveView {
  /** 人机模式只能打练习局 */
  practice_only: boolean;
  /** 仓库价值系数, 基点 */
  value_factor: number;
  value_range: ValueRange;
  /** 今天受到的每日收益上限, 不限制时为空; 任一上限达到后人机对局转为练习局 */
  profit_caps: ProfitCap[];
}

/** 人机模式的一个每日收益上限和当天的净收益, 金额为出价货币的最小单位, 对应后端 venue.ProfitCap */
export interface ProfitCap {
  /** mode:pve 为全局的上限, venue:<地图代码>:pve 为这张地图的上限 */
  scope: string;
  limit: number;
  profit: number;
}

/** 玩家看到的地图, 对应后端 venue.VenueView */
export interface Venue {
  code: string;
  name: string;
  description: string;
  image_url: string;
  pve_enabled: boolean;
  ranked_enabled: boolean;
  bid_currency: string;
  entry_cost: CostLine[];
  requirements: RequirementProgress[];
  /** 满足全部入场条件 */
  eligible: boolean;
  grid: Grid;
  auction: AuctionRule;
  /** 主持人的提示 */
  host: HostRule;
  /** 排位模式的仓库价值区间 */
  value_range: ValueRange;
  pve: PveView;
  /** 拍场的拍品数和格子大小, 一局一个仓库时为 null */
  lots: LotsView | null;
}

/** 仓库的一个档位, 对应后端 venue.Slot */
export interface Slot {
  /** 掉落池代码 */
  pool: string;
  /** 抽取次数在 [min, max] 内均匀随机 */
  min: number;
  max: number;
  /** 这一档出现的概率, 基点, 10000 表示一定出现 */
  chance: number;
}

/** 掉落池中可抽取的一个条目, 对应后端 loot.EntryDetail */
export interface OddsEntry {
  id: number;
  /** 掉落的物品, 与 currency 二选一 */
  item_id: number;
  currency: string;
  /** 每次掉落的数量范围, 货币为金额 */
  min_quantity: number;
  max_quantity: number;
  name: string;
  /** 物品品质, 货币为 0 */
  quality: number;
  image_url: string;
  /** 每次抽中的概率 */
  probability: number;
}

/** 掉落池的公示概率, 对应后端 loot.PoolOdds */
export interface PoolOdds {
  code: string;
  name: string;
  description: string;
  entries: OddsEntry[];
}

/** 档位和它的掉落概率, 对应后端 venue.SlotOdds */
export interface SlotOdds extends Slot {
  /** 掉落池不可用时为 null */
  odds: PoolOdds | null;
}

/** 玩家看到的地图详情, 对应后端 venue.VenueDetail */
export interface VenueDetail extends Venue {
  /** 一局一个仓库时的档位, 拍场为空, 见 lot_slots */
  slots: SlotOdds[];
  /** 拍场每种格子大小的仓库, 一局一个仓库时为空 */
  lot_slots: GridOdds[];
}

/** 拍场中一种格子大小的仓库, 同一格子大小的各档次不公开名称 */
export interface GridOdds {
  grid: Grid;
  variants: {
    /** 这个格子大小的拍品是这一档的概率, 基点 */
    share: number;
    slots: SlotOdds[];
  }[];
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
  /** 主持人的提示 */
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

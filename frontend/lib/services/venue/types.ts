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
  start_price: number;
  /** 开局时冻结 min(可用余额, 上限), 出价不能超过它 */
  max_carry: number;
  /** 人机的带入资金, 不超过带入资金上限; 旧地图为 0, 按带入资金上限 */
  bot_carry: number;
  /** 速胜时第一名的出价不能低于它 */
  speed_floor: number;
  /** 领先折让, 基点: 成交价 = 第二名的出价 + 领先部分 × (1 - 折让) */
  lead_discount: number;
  /** 首轮扫描比例, 基点: 第 1 轮每件藏品按这个概率扫出轮廓, 至少一件; 其余直到开箱都不公开 */
  preview_rate: number;
  /** 向主持人买一次情报的价格, 从可用余额扣除; 0 表示不卖情报 */
  intel_price: number;
}

/**
 * 主持人, 对应后端 venue.HostRule: 每轮开始时公开一条关于仓库的提示, 有一定概率说错;
 * 出价阶段还可以向他买只有自己看得到的情报, 价格按期望价值推导
 */
export interface HostRule {
  /** 关闭时没有提示和热度, 情报商店仍按 intel_limit 开放 */
  enabled: boolean;
  /** 开局时从中随机选一种性格, 为空表示全部; 见 lib/services/auction 的 HOST_PERSONA */
  personas: string[];
  /** 提示说错的概率, 基点, 不超过 2500; 按性格和热度调整 */
  lie_rate: number;
  /** 提示涉及的品质不低于它时撒谎率 × 1.5, 0 表示不区分; 也是"高品质件数"提示统计的最低品质 */
  high_quality: number;
  /** 每人每局最多买几次情报, 每轮最多一次, 0 表示不卖情报 */
  intel_limit: number;
  /** 估价情报的误差, 基点, 区间宽度为价值 × 误差 × 2; 卖情报时不低于 500 */
  intel_precision: number;
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
  /** 主持人的提示和情报 */
  host: HostRule;
  /** 排位模式的仓库价值区间 */
  value_range: ValueRange;
  pve: PveView;
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
  slots: SlotOdds[];
}

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

/** 拍卖参数, 金额按出价货币的最小单位, 对应后端 venue.AuctionRule */
export interface AuctionRule {
  start_price: number;
  /** 开局时冻结 min(可用余额, 上限), 出价不能超过它 */
  max_carry: number;
  rounds: number;
  round_seconds: number;
  reveal_seconds: number;
  /** 前 rounds-1 轮的速胜倍率, 基点 */
  quick_win: number[];
  max_extra_rounds: number;
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

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

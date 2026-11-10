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

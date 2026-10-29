import type {Grid} from '../venue';

/** 暗拍阶段, 与后端 internal/apps/auction/constants.go 对应 */
export const AUCTION_PHASE = {
  /** 出价, round 为第几轮, 加轮时继续递增 */
  bidding: 'bidding',
  /** 开箱, 已结算 */
  reveal: 'reveal',
} as const;

/** 结算结果 */
export const AUCTION_OUTCOME = {
  sold: 'sold',
  unsold: 'unsold',
  cancelled: 'cancelled',
} as const;

/** 暗拍相关的 ws 消息 */
export const AUCTION_MESSAGES = {
  /** 出价请求, data 为 BidRequest, 回复的 data 为 AuctionView */
  bid: 'auction:bid',
  /** 开局、每轮结束和开箱时推送, data 为 AuctionView */
  updated: 'auction:updated',
  /** 使用道具, data 为 UseRequest, 回复的 data 为 AuctionView */
  use: 'auction:use',
} as const;

/** 对局中的道具效果, 与后端 internal/apps/auction/effects.go 的 Effect 常量对应 */
export const AUCTION_EFFECT = {
  /** 看几件还不知道品质的藏品的品质 */
  revealQuality: 'reveal_quality',
  /** 某个品质及以上的藏品各有几件 */
  countQuality: 'count_quality',
  /** 看几件某个品质及以上的藏品的名称 */
  revealName: 'reveal_name',
  /** 仓库价值的区间 */
  valueRange: 'value_range',
} as const;

/** 对局中展示的地图参数, 来自开局时的快照 */
export interface AuctionVenue {
  code: string;
  name: string;
  bid_currency: string;
  grid: Grid;
  start_price: number;
  rounds: number;
  /** 前 rounds-1 轮的速胜倍率, 基点 */
  quick_win: number[];
  max_extra_rounds: number;
  /** 这个模式每局最多使用几个道具, 0 表示不能用 */
  max_uses: number;
  /** 可以用的道具, 为空表示所有对局中可用的道具 */
  allowed_items: number[];
}

/** 已公开品质的藏品 */
export interface RevealedPiece {
  /** 公开顺序, 从 1 开始 */
  order: number;
  quality: number;
}

/** 一轮结束后的反馈, 不透露其他人的出价 */
export interface Feedback {
  round: number;
  /** 自己的出价在出过价的人中排第几, 没出过价为 0 */
  rank: number;
  /** 见 lib/auction.ts 的 HINTS */
  hint: string;
}

/** 一组货币和物品, 对应后端 reward.Bundle */
export interface Bundle {
  currencies?: { currency: string; amount: number }[];
  items?: { item_id: number; quantity: number }[];
}

/** 自己在这局中的状态 */
export interface AuctionSelf {
  seat: number;
  is_practice: boolean;
  /** 实际扣除的入场代价, 练习局为空 */
  entry_cost: Bundle;
  /** 带入资金, 出价的上限 */
  carry: number;
  /** 当前出价, 0 表示还没出过价 */
  bid: number;
  /** 本轮是否已出价 */
  submitted: boolean;
  /** 现在是否可以出价 */
  can_bid: boolean;
  feedback: Feedback[];
  /** 自己用过的道具和得到的情报, 按使用顺序 */
  uses: UseView[];
  /** 本局还能用几个道具, 练习局为 0; 不考虑阶段和轮次 */
  uses_left: number;
}

/** 道具看到的一件藏品, 以扫描顺序对应 Footprint */
export interface PieceIntel {
  scan_order: number;
  quality: number;
  /** 只有看名称的道具有 */
  name?: string;
  image_url?: string;
}

/** 一种品质的藏品件数 */
export interface QualityCount {
  quality: number;
  count: number;
}

/** 使用一个道具得到的情报, 字段按效果使用其中的一部分, 对应后端 auction.Intel */
export interface Intel {
  /** 看到品质或名称的藏品, 按扫描顺序; 没有符合条件的藏品时没有 */
  pieces?: PieceIntel[];
  /** 统计或查看的最低品质 */
  min_quality?: number;
  /** 最低品质及以上各品质的件数, 品质从高到低, 没有的品质不列出 */
  counts?: QualityCount[];
  /** 仓库价值的区间, 出价货币 */
  value_min?: number;
  value_max?: number;
}

/** 自己用过的一个道具 */
export interface UseView {
  round: number;
  item_id: number;
  /** 使用时的物品名称 */
  name: string;
  /** 效果代码, 见 AUCTION_EFFECT, 按它展示 result */
  effect: string;
  result: Intel;
  created_at: string;
}

/** 开箱后公开的一个座位用某种道具的个数 */
export interface UseCount {
  item_id: number;
  name: string;
  count: number;
}

/** 开箱后公开的一件藏品 */
export interface AuctionPiece {
  item_id: number;
  name: string;
  image_url: string;
  quality: number;
  width: number;
  height: number;
  /** 从 0 开始, 左上角为原点 */
  x: number;
  y: number;
  sell_currency: string;
  /** 开局时的回收价 */
  sell_price: number;
  /** 开箱时扫描的顺序, 从 1 开始 */
  scan_order: number;
  /** 扫描这件的时长, 毫秒 */
  scan_ms: number;
}

/** 一件藏品在仓库中占的格子, 出价时就公开, 不包含名称和回收价 */
export interface Footprint {
  /** 开箱时扫描的顺序, 从 1 开始 */
  scan_order: number;
  x: number;
  y: number;
  width: number;
  height: number;
  /** 已公开品质时为品质, 否则为 0 */
  quality: number;
}

/** 开箱后公开的一个座位的出价 */
export interface SeatResult {
  seat: number;
  /** 最后的出价, 0 表示没出过价 */
  bid: number;
  /** 用过的道具, 按第一次使用的顺序, 不公开得到的情报 */
  uses: UseCount[];
}

/** 结算结果 */
export interface AuctionResult {
  outcome: string;
  /** 没有成交时为 0 */
  winner_seat: number;
  price: number;
  /** 仓库价值 */
  value: number;
  /** 自己的盈亏, 已扣除入场代价; 练习局按正式局计算; 管理员查看时为 0 */
  profit: number;
  /** 自己段位分的变化, 只有排位成交时有; 管理员查看时为 0 */
  rating_delta: number;
  /** 自己结算后的段位分, 只有排位成交时有; 管理员查看时为 0 */
  rating: number;
  /** 按扫描顺序 */
  pieces: AuctionPiece[];
  /** 按座位号排序 */
  seats: SeatResult[];
  /** 开始扫描的时间, 即结算时间 */
  scan_from: string | null;
}

/** 一局暗拍中所有人都看得到的部分, 对应后端 auction.PublicView */
export interface AuctionPublicView {
  match_id: number;
  mode: string;
  status: string;
  phase: string;
  round: number;
  phase_ends_at: string | null;
  /** 与对局的 version 一致, 只采用 version 更大的推送 */
  version: number;
  venue: AuctionVenue;
  piece_count: number;
  /** 全部藏品占的格子, 按扫描顺序 */
  footprints: Footprint[];
  /** 第 N 轮公开前 N 件藏品的品质 */
  revealed: RevealedPiece[];
  /** 加轮时可以出价的座位, 为空表示所有人 */
  contenders: number[];
  /** 结算后才有 */
  result: AuctionResult | null;
}

/** 玩家看到的一局暗拍, 对应后端 auction.AuctionView */
export interface AuctionView extends AuctionPublicView {
  self: AuctionSelf;
}

/** 管理员看到的一个座位的暗拍数据, 对应后端 auction.Player */
export interface AuctionPlayer {
  seat: number;
  /** 人机为 0 */
  user_id: number;
  is_practice: boolean;
  /** 实际扣除的入场代价, 练习局和人机为空 */
  entry_cost: Bundle;
  /** 带入资金, 出价的上限 */
  carry: number;
  /** 冻结的带入资金, 练习局和人机为 0 */
  frozen: number;
  /** 人机对仓库价值的估值, 真人为 0 */
  estimate: number;
  /** 当前出价, 0 表示还没出过价 */
  bid: number;
  /** 结算后的盈亏; 练习局按正式局计算 */
  profit: number;
  /** 本局段位分的变化, 只有排位成交时有 */
  rating_delta: number;
  /** 结算后的段位分, 只有排位成交时有 */
  rating: number;
  feedback: Feedback[];
}

/** 管理员看到的一局暗拍, 对应后端 auction.AdminView */
export interface AuctionAdminView extends AuctionPublicView {
  /** 按座位号排序 */
  players: AuctionPlayer[];
  /** 全部座位用过的道具和得到的情报, 按使用顺序 */
  uses: AuctionUse[];
}

/** 对局中用掉的一个道具, 对应后端 auction.Use */
export interface AuctionUse extends UseView {
  id: number;
  match_id: number;
  seat: number;
  /** 使用时的效果参数 */
  params: Record<string, unknown>;
}

/** 出价请求 */
export interface BidRequest {
  match_id: number;
  /** 出价的轮次, 与当前轮次不一致时被拒绝 */
  round: number;
  /** 0 表示不加价 */
  amount: number;
}

/** 使用道具 */
export interface UseRequest {
  match_id: number;
  /** 使用时的轮次, 与当前轮次不一致时被拒绝 */
  round: number;
  item_id: number;
}

/** 背包中这局可以用的一种道具, 对应后端 auction.UsableItem */
export interface UsableItem {
  item_id: number;
  name: string;
  description: string;
  quality: number;
  image_url: string;
  effect: string;
  quantity: number;
}

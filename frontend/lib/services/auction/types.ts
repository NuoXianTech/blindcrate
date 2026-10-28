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
}

/** 开箱后公开的一个座位的出价 */
export interface SeatResult {
  seat: number;
  /** 最后的出价, 0 表示没出过价 */
  bid: number;
}

/** 结算结果 */
export interface AuctionResult {
  outcome: string;
  /** 没有成交时为 0 */
  winner_seat: number;
  price: number;
  /** 仓库价值 */
  value: number;
  /** 自己的盈亏, 已扣除入场代价; 练习局按正式局计算 */
  profit: number;
  pieces: AuctionPiece[];
  /** 按座位号排序 */
  seats: SeatResult[];
}

/** 玩家看到的一局暗拍, 对应后端 auction.AuctionView */
export interface AuctionView {
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
  /** 第 N 轮公开前 N 件藏品的品质 */
  revealed: RevealedPiece[];
  /** 加轮时可以出价的座位, 为空表示所有人 */
  contenders: number[];
  self: AuctionSelf;
  /** 结算后才有 */
  result: AuctionResult | null;
}

/** 出价请求 */
export interface BidRequest {
  match_id: number;
  /** 出价的轮次, 与当前轮次不一致时被拒绝 */
  round: number;
  /** 0 表示不加价 */
  amount: number;
}

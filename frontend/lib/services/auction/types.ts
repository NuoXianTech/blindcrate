import type {BotSnapshot} from '../bot';
import type {Grid} from '../venue';

/** 暗拍阶段, 与后端 internal/apps/auction/constants.go 对应 */
export const AUCTION_PHASE = {
  /** 出价, round 为拍场内的第几轮, 整场连续编号, 加轮时继续递增 */
  bidding: 'bidding',
  /** 当前拍品开箱, 已结算; 不是最后一个拍品时之后开拍下一个 */
  reveal: 'reveal',
} as const;

/** 拍品和拍场的结算结果; 拍场至少成交一个拍品时为 sold */
export const AUCTION_OUTCOME = {
  sold: 'sold',
  unsold: 'unsold',
  cancelled: 'cancelled',
} as const;

/** 暗拍相关的 ws 消息 */
export const AUCTION_MESSAGES = {
  /** 出价请求, data 为 BidRequest, 回复的 data 为 AuctionView */
  bid: 'auction:bid',
  /** 开局、每轮结束、开箱和下一个拍品开拍时推送, data 为 AuctionView */
  updated: 'auction:updated',
  /** 使用道具, data 为 UseRequest, 回复的 data 为 AuctionView */
  use: 'auction:use',
  /** 向主持人买情报, data 为 IntelRequest, 回复的 data 为 AuctionView */
  intel: 'auction:intel',
} as const;

/** 主持人的性格, 与后端 internal/apps/venue/constants.go 的 Host 常量对应 */
export const HOST_PERSONA = {
  /** 稳重: 撒谎率减半, 说错时随机偏高或偏低 */
  steady: 'steady',
  /** 煽动: 说错时只往高了说, 热度高时更敢说错 */
  showman: 'showman',
  /** 狡黠: 说错时只往低了说, 热度低时更敢说错 */
  sly: 'sly',
} as const;

/** 主持人提示的类型, 与后端 internal/apps/auction/constants.go 的 HostHint 常量对应 */
export const HOST_HINT = {
  /** 品质不低于 quality 的藏品共有 count 件 */
  count: 'count',
  /** 仓库里最好的藏品是 quality */
  best: 'best',
  /** 位置为 (x, y) 的藏品是 quality, 只指第 1 轮扫出轮廓的 */
  piece: 'piece',
  /** 仓库价值高于(above)或不高于期望价值 */
  value: 'value',
} as const;

/** 对局中的道具效果, 与后端 internal/apps/auction/effects.go 的 Effect 常量对应 */
export const AUCTION_EFFECT = {
  /** 看几件还不知道轮廓的藏品的轮廓 */
  revealOutline: 'reveal_outline',
  /** 看几件还不知道品质的藏品的品质和轮廓 */
  revealQuality: 'reveal_quality',
  /** 某个品质及以上的藏品各有几件 */
  countQuality: 'count_quality',
  /** 看几件某个品质及以上的藏品的名称、外观和轮廓 */
  revealName: 'reveal_name',
  /** 仓库价值的区间 */
  valueRange: 'value_range',
} as const;

/** 对局中展示的地图参数, 来自开局时的快照; 每个拍品的起拍价等见 LotView */
export interface AuctionVenue {
  code: string;
  name: string;
  bid_currency: string;
  /** 每个拍品的轮数 */
  rounds: number;
  /** 前 rounds-1 轮的速胜倍率, 基点 */
  quick_win: number[];
  max_extra_rounds: number;
  /** 入场费, 以下字段见 lib/services/venue 的 AuctionRule */
  entry_fee: number;
  /** 带入资金上限 */
  max_carry: number;
  /** 领先折让, 基点: 成交价 = 第二名的出价 + 领先部分 × (1 - 折让) */
  lead_discount: number;
  /** 首轮扫描比例, 基点: 每个拍品的第 1 轮每件藏品按这个概率扫出轮廓, 至少一件 */
  preview_rate: number;
  /** 这个模式每局(整个拍场)最多使用几个道具, 0 表示不能用 */
  max_uses: number;
  /** 可以用的道具, 为空表示所有对局中可用的道具 */
  allowed_items: number[];
  /** 每人每局(整个拍场)最多买几次情报, 每轮最多一次 */
  intel_limit: number;
  /** 拍品开箱后停留多久进入下一个, 秒 */
  gap_seconds: number;
}

/** 主持人的一条提示, 对应后端 auction.HintView */
export interface HostHint {
  /** 提示针对的拍品 */
  lot_no: number;
  /** 拍场内的轮次 */
  round: number;
  /** 见 HOST_HINT */
  kind: string;
  quality: number;
  count: number;
  /** 指认的藏品的位置, 是第 1 轮扫出轮廓的一件 */
  x: number;
  y: number;
  above: boolean;
  /** 是否属实, 拍场结束后才公开 */
  truthful: boolean | null;
}

/** 热度的一次变动 */
export interface HypeEvent {
  lot_no: number;
  /** 开局和拍品开场(热度回落)时为 0, 之后为结束的轮次 */
  round: number;
  /** 这一轮加价的人数 */
  raises: number;
  delta: number;
  /** 变动后的热度, 0-100 */
  hype: number;
}

/** 公开的主持人状态, 对应后端 auction.HostView */
export interface HostView {
  /** 见 HOST_PERSONA, 拍场结束后才公开, 之前为空 */
  persona: string;
  hype: number;
  /** 按轮次 */
  hints: HostHint[];
  /** 按时间 */
  hypes: HypeEvent[];
}

/** 一个座位买情报的次数, 只公开次数, 不公开内容 */
export interface SeatIntel {
  seat: number;
  count: number;
  /** 最近一次买的轮次 */
  round: number;
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
  /** 实际扣除的入场费和入场代价, 练习局为空 */
  entry_cost: Bundle;
  /** 带入资金, 整个拍场共用 */
  carry: number;
  /** 已拿下的拍品的成交价之和 */
  spent: number;
  /** 剩余资金, 当前拍品出价的上限 */
  remaining: number;
  /** 当前拍品的出价, 0 表示还没出过价 */
  bid: number;
  /** 本轮是否已出价 */
  submitted: boolean;
  /** 现在是否可以出价; 剩余资金不够当前拍品的起拍价时不能 */
  can_bid: boolean;
  /** 自己用过的道具和得到的情报, 按使用顺序, 含之前的拍品 */
  uses: UseView[];
  /** 本局还能用几个道具, 练习局为 0; 不考虑阶段和轮次 */
  uses_left: number;
  /** 本局还能买几次情报, 练习局和不卖情报时为 0 */
  intel_left: number;
  /** 本轮是否已买过情报 */
  intel_bought: boolean;
  /** 拍场结算后的盈亏, 不含准入时扣除的入场费和入场代价; 练习局按正式局计算 */
  profit: number;
  /** 拍场结算后的名次, 全部流拍时为 0 */
  rank: number;
  /** 段位分的变化, 只有排位且至少成交一个拍品时有 */
  rating_delta: number;
  /** 结算后的段位分, 同上 */
  rating: number;
}

/** 道具看到的一件藏品, 带占的格子, 以位置 (x, y) 区分 */
export interface PieceIntel {
  x: number;
  y: number;
  width: number;
  height: number;
  /** 0 为只扫出轮廓 */
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
  /** 看到品质或名称的藏品, 按位置从上到下、从左到右; 没有符合条件的藏品时没有 */
  pieces?: PieceIntel[];
  /** 统计或查看的最低品质 */
  min_quality?: number;
  /** 最低品质及以上各品质的件数, 品质从高到低, 没有的品质不列出 */
  counts?: QualityCount[];
  /** 仓库价值的区间, 出价货币 */
  value_min?: number;
  value_max?: number;
}

/** 自己用过的一个道具或买的一次情报 */
export interface UseView {
  /** 使用时的拍品 */
  lot_no: number;
  round: number;
  /** 向主持人买的情报为 0 */
  item_id: number;
  /** 使用时的物品名称 */
  name: string;
  /** 效果代码, 见 AUCTION_EFFECT, 按它展示 result */
  effect: string;
  result: Intel;
  created_at: string;
}

/** 拍品开箱后公开的一个座位在这个拍品上用某种道具的个数 */
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

/** 第 1 轮扫出轮廓的一件藏品占的格子, 不包含品质、名称和回收价, 以位置 (x, y) 区分 */
export interface Footprint {
  x: number;
  y: number;
  width: number;
  height: number;
}

/** 拍品开箱后公开的一个座位的出价 */
export interface SeatResult {
  seat: number;
  /** 最后的出价, 0 表示没出过价 */
  bid: number;
  /** 在这个拍品上用过的道具, 按第一次使用的顺序, 不公开得到的情报 */
  uses: UseCount[];
}

/** 公开的一次出价: 拍品的前 rounds-1 轮结束后公开, 最后一轮和加轮在拍品结算后公开 */
export interface PublicBid {
  /** 拍场内的轮次 */
  round: number;
  seat: number;
  /** 0 表示不加价 */
  amount: number;
}

/**
 * 拍场中的一个拍品(仓库), 对应后端 auction.LotView
 * 格子大小和金额开局时公开, 第 1 轮的轮廓开拍后公开, 价值、藏品和每个座位的出价开箱后公开, 档次拍场结束后公开
 */
export interface LotView {
  /** 出场顺序, 从 1 开始 */
  lot_no: number;
  grid: Grid;
  start_price: number;
  /** 速胜时第一名的出价不能低于它 */
  speed_floor: number;
  /** 向主持人买一次情报的价格, 0 表示不卖情报 */
  intel_price: number;
  /** 开拍时拍场内的轮次, 还没开拍时为 0 */
  first_round: number;
  /** 第 1 轮扫出轮廓的藏品占的格子; 件数开箱前不公开 */
  footprints: Footprint[];
  /** 加轮时可以出价的座位, 为空表示所有人 */
  contenders: number[];
  /** 已公开的出价, 按轮次和座位 */
  bids: PublicBid[];
  /** 结算前为空, 之后见 AUCTION_OUTCOME */
  outcome: string;
  /** 没有成交时为 0 */
  winner_seat: number;
  price: number;
  /** 仓库价值, 开箱后公开 */
  value: number;
  /** 档次名称, 拍场结束后公开 */
  tier: string;
  /** 开箱后公开, 按扫描顺序 */
  pieces: AuctionPiece[] | null;
  /** 开箱后公开, 按座位号排序 */
  seats: SeatResult[] | null;
  /** 开始扫描的时间, 即结算时间 */
  scan_from: string | null;
}

/** 拍场结束后公开的一个座位的成绩 */
export interface SessionSeat {
  seat: number;
  carry: number;
  spent: number;
  /** 拿下的拍品的价值 - 成交价, 不含买情报的钱 */
  profit: number;
  rank: number;
}

/** 拍场结束后公开的成绩 */
export interface SessionView {
  /** 至少成交一个拍品时为 sold */
  outcome: string;
  /** 按座位号排序 */
  seats: SessionSeat[];
}

/** 一局暗拍(拍场)中所有人都看得到的部分, 对应后端 auction.PublicView */
export interface AuctionPublicView {
  match_id: number;
  mode: string;
  status: string;
  phase: string;
  /** 拍场内的轮次, 从 1 开始整场连续编号 */
  round: number;
  /** 当前拍品的第几轮, 大于 venue.rounds 时为加轮 */
  lot_round: number;
  phase_ends_at: string | null;
  /** 与对局的 version 一致, 只采用 version 更大的推送 */
  version: number;
  venue: AuctionVenue;
  lot_count: number;
  /** 当前拍品的序号, 从 1 开始 */
  lot: number;
  /** 按序号 */
  lots: LotView[];
  /** 主持人, 没有主持人提示时为空 */
  host: HostView | null;
  /** 在当前拍品上买过情报的座位, 按座位号 */
  intel_seats: SeatIntel[];
  /** 拍场结束后才有 */
  session: SessionView | null;
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
  /** 人机的风格, 开局时的快照; 真人为空对象 */
  persona: BotSnapshot;
  is_practice: boolean;
  /** 实际扣除的入场代价, 练习局和人机为空 */
  entry_cost: Bundle;
  /** 带入资金, 整个拍场共用 */
  carry: number;
  /** 冻结的带入资金, 练习局和人机为 0 */
  frozen: number;
  /** 已拿下的拍品的成交价之和 */
  spent: number;
  /** 人机对当前拍品按公开信息的最新估值, 真人为 0 */
  estimate: number;
  /** 当前拍品的出价, 0 表示还没出过价 */
  bid: number;
  /** 拍场结算后的盈亏; 练习局按正式局计算 */
  profit: number;
  /** 拍场的名次, 全部流拍时为 0 */
  rank: number;
  /** 本局段位分的变化, 只有排位且至少成交一个拍品时有 */
  rating_delta: number;
  /** 结算后的段位分, 同上 */
  rating: number;
}

/** 管理员看到的一个拍品的完整数据, 对应后端 auction.Lot */
export interface AuctionLotRow {
  lot_no: number;
  /** 档次代码 */
  tier: string;
  grid: Grid;
  start_price: number;
  first_round: number;
  value: number;
  piece_count: number;
  extra_rounds: number;
  outcome: string;
  winner_seat: number;
  price: number;
}

/** 管理员看到的一局暗拍, 对应后端 auction.AdminView */
export interface AuctionAdminView extends AuctionPublicView {
  /** 全部拍品, 含档次代码、价值和件数 */
  lot_rows: AuctionLotRow[];
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
  /** 向主持人买情报时付的出价货币, 用道具时为 0 */
  cost: number;
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

/** 向主持人买情报 */
export interface IntelRequest {
  match_id: number;
  /** 购买时的轮次, 与当前轮次不一致时被拒绝 */
  round: number;
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

/** 自己在一张地图选的带入金额, 对应后端 auction.CarryPreference */
export interface CarryPreference {
  /** 地图代码 */
  venue: string;
  amount: number;
  updated_at: string;
}

/** 选择在一张地图的带入金额, 超出 [起拍价, 带入资金上限] 时按边界保存 */
export interface CarryRequest {
  venue: string;
  amount: number;
}

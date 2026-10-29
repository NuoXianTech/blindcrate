/** 对局状态, 与后端 internal/apps/match/constants.go 对应 */
export const MATCH_STATUS = {
  playing: 'playing',
  finished: 'finished',
  cancelled: 'cancelled',
} as const;

/** 对局相关的 ws 推送 */
export const MATCH_MESSAGES = {
  /** 对局开始、切换阶段或结束, data 为 MatchView */
  updated: 'match:updated',
  /** 创建对局失败, 已移出队列, data 为 MatchmakingFailedMessage */
  matchmakingFailed: 'match:matchmaking_failed',
  /** 开局时不满足玩法的条件, 已移出队列, data 为 MatchmakingRejectedMessage */
  matchmakingRejected: 'match:matchmaking_rejected',
} as const;

/** 排队状态 */
export interface QueueStatus {
  /** 玩法代码, 如 pve:dock */
  mode: string;
  joined_at: string;
}

/** 匹配状态, 排队和对局最多只有一个 */
export interface MatchmakingStatus {
  /** 没有排队时为空 */
  queue: QueueStatus | null;
  /** 进行中的对局, 没有时为 0 */
  match_id: number;
}

/** 玩家看到的座位, 不区分真人和人机 */
export interface MatchPlayer {
  seat: number;
  nickname: string;
  avatar_url: string;
  /** 结束时的名次, 0 表示没有名次 */
  rank: number;
  is_self: boolean;
}

/** 对局记录中暗拍的摘要, 金额为出价货币的最小单位, 对应后端 auction.Summary */
export interface MatchSummary {
  /** 地图代码 */
  venue: string;
  /** 开局时的地图名称 */
  venue_name: string;
  bid_currency: string;
  /** 结算前为空, 之后为 sold、unsold 或 cancelled */
  outcome: string;
  /** 成交价, 没有成交时为 0 */
  price: number;
  /** 是否由自己成交 */
  won: boolean;
  /** 自己的盈亏, 结算前为 0 */
  profit: number;
  /** 自己打的是否为练习局 */
  is_practice: boolean;
  /** 自己段位分的变化, 只有排位成交时有 */
  rating_delta: number;
}

/** 玩家看到的对局, 对应后端 match.MatchView */
export interface MatchView {
  id: number;
  mode: string;
  status: string;
  phase: string;
  round: number;
  /** 当前阶段的截止时间, 不限时或已结束时为空 */
  phase_ends_at: string | null;
  /** 越大越新, 只采用 version 更大的推送 */
  version: number;
  /** 玩家可以离开的时间, 之后不再算作进行中的对局; 为空表示还没释放 */
  released_at: string | null;
  /** 按座位号排序 */
  players: MatchPlayer[];
  created_at: string;
  finished_at: string | null;
  /** 玩法提供的摘要, 只在对局记录中返回 */
  summary?: MatchSummary;
}

export interface ListMatchesParams {
  current: number;
  size: number;
  status?: string;
  /** 玩法代码冒号前的部分, 即模式, 如 pve */
  prefix?: string;
  /** 玩法代码冒号后的部分, 即地图代码 */
  suffix?: string;
}

export interface ListMatchesData {
  total: number;
  matches: MatchView[];
}

export interface MatchmakingFailedMessage {
  mode: string;
}

export interface MatchmakingRejectedMessage {
  mode: string;
  /** 不满足的条件, 如余额不足 */
  reason: string;
}

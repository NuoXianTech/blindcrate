import type {PageParams} from '../core';

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
  /** 队列中的人数变化, 推送给仍在队列中的人, data 为 QueueUpdatedMessage */
  queueUpdated: 'match:queue_updated',
} as const;

/** 排队状态 */
export interface QueueStatus {
  /** 玩法代码, 如 pve:dock */
  mode: string;
  joined_at: string;
  /** 队列中的人数, 含自己 */
  count: number;
  /** 凑够多少人立即开局, 玩法已不存在时为 0 */
  capacity: number;
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
  /** 拍场结算前为空, 之后为 sold(至少成交一个拍品)、unsold 或 cancelled */
  outcome: string;
  /** 拍品数 */
  lot_count: number;
  /** 自己拿下的拍品数 */
  won_lots: number;
  /** 自己拿下的拍品的成交价之和 */
  spent: number;
  /** 自己的盈亏, 结算前为 0 */
  profit: number;
  /** 自己的名次, 全部流拍时为 0 */
  rank: number;
  /** 自己打的是否为练习局 */
  is_practice: boolean;
  /** 自己段位分的变化, 只有排位且至少成交一个拍品时有 */
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

export interface QueueUpdatedMessage {
  mode: string;
  /** 队列中的人数, 含自己 */
  count: number;
}

export interface MatchmakingRejectedMessage {
  mode: string;
  /** 不满足的条件, 如余额不足 */
  reason: string;
}

/** 管理员看到的座位, 对应后端 match.PlayerDetail */
export interface AdminMatchPlayer {
  id: number;
  match_id: number;
  seat: number;
  /** 人机为 0 */
  user_id: number;
  /** 人机的昵称, 真人为空 */
  bot_name: string;
  /** 结束时的名次, 0 表示没有名次 */
  rank: number;
  created_at: string;
  is_bot: boolean;
  /** 真人的昵称, 未设置时为用户名; 人机为人机的昵称 */
  nickname: string;
  avatar_url: string;
}

/** 管理员看到的对局, 对应后端 match.MatchDetail */
export interface AdminMatch {
  id: number;
  mode: string;
  status: string;
  phase: string;
  round: number;
  /** 当前阶段的截止时间, 不限时或已结束时为空 */
  phase_ends_at: string | null;
  version: number;
  /** 玩家可以离开的时间, 为空表示还没释放 */
  released_at: string | null;
  created_at: string;
  updated_at: string;
  finished_at: string | null;
  /** 按座位号排序 */
  players: AdminMatchPlayer[];
}

export interface ListAdminMatchesParams extends PageParams {
  /** 玩法代码冒号前的部分, 如 pve */
  prefix?: string;
  /** 玩法代码冒号后的部分, 如地图代码 */
  suffix?: string;
  status?: string;
  /** 只看该用户参与的对局 */
  user_id?: number;
  /** 只看该用户也参与的对局, 与 user_id 一起用于查看两个人一起打的对局 */
  other_id?: number;
}

export interface ListAdminMatchesData {
  total: number;
  matches: AdminMatch[];
}

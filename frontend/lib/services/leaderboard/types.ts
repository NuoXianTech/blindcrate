/** 榜单的统计周期 */
export const LEADERBOARD_PERIOD = {
  total: 'total',
  day: 'day',
  season: 'season',
} as const;

export type LeaderboardPeriod = (typeof LEADERBOARD_PERIOD)[keyof typeof LEADERBOARD_PERIOD];

/** 段位分指标, 读赛季的段位分而不是统计, 只能用于全部对局范围的赛季榜单 */
export const METRIC_RATING = 'rating';

/** 玩家看到的榜单, 对应后端 leaderboard.BoardView */
export interface Leaderboard {
  code: string;
  name: string;
  description: string;
  metric: string;
  /** 指标的展示名称, 如 "胜利次数" */
  metric_name: string;
  /** 指标为某种货币的净收益时为货币代码, 数值按该货币的最小单位计; 否则为空 */
  currency: string;
  scope: string;
  /** 统计范围的展示名称, 全部对局为空 */
  scope_name: string;
  period: LeaderboardPeriod;
  size: number;
  /** 同一范围和周期内至少完成多少局才上榜 */
  min_matches: number;
}

/** 榜单上的一个名次 */
export interface LeaderboardEntry {
  rank: number;
  /** 未设置昵称时为用户名 */
  nickname: string;
  avatar_url: string;
  value: number;
  is_self: boolean;
}

/** 自己在榜单上的情况 */
export interface LeaderboardMine {
  /** 名次, 可能在前 N 名之外; 0 表示没有上榜 */
  rank: number;
  /** 自己的数值, 没有上榜也会返回 */
  value: number;
}

/** 榜单的前 N 名和自己的名次, 对应后端 leaderboard.BoardDetail */
export interface LeaderboardDetail extends Leaderboard {
  /** 当前的统计周期, 如 total、day:2026-10-28、season:s1; 赛季榜单还没有赛季时为空 */
  current_period: string;
  /** 赛季榜单对应的赛季名称, 其他榜单为空 */
  season_name: string;
  /** 赛季榜单对应的赛季已经结束, 名次不再变化 */
  season_ended: boolean;
  /** 按名次排列; 前 N 名缓存 60 秒 */
  entries: LeaderboardEntry[];
  mine: LeaderboardMine;
}

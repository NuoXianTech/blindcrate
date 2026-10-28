/** 统计范围: 全部对局 */
export const STATS_SCOPE_ALL = 'all';

/** 通用的计数指标, 与后端 internal/apps/stats/constants.go 对应; bid_won 由暗拍注册 */
export const STATS_METRIC = {
  matches: 'matches',
  wins: 'wins',
  bidWon: 'bid_won',
} as const;

/** 净收益指标的前缀, 如 profit:chip */
export const PROFIT_METRIC_PREFIX = 'profit:';

/** 计数指标, 对应后端 stats.Metric */
export interface StatMetric {
  key: string;
  label: string;
}

/** 指标 -> 值, 没有记录的指标不返回 */
export type StatValues = Record<string, number>;

/** 一天的全部指标 */
export interface DayValues {
  /** 日期, 如 2026-10-28 */
  day: string;
  values: StatValues;
}

/** 当前用户在一个统计范围的数据, 对应后端 stats.UserStats */
export interface UserStats {
  scope: string;
  /** 全部计数指标的名称, 按注册顺序 */
  metrics: StatMetric[];
  /** 累计值 */
  total: StatValues;
  /** 最近 7 天, 从早到晚, 最后一天为今天 */
  days: DayValues[];
}

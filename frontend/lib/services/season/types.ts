/** 赛季的状态, 由时间推算 */
export const SEASON_STATUS = {
  upcoming: 'upcoming',
  active: 'active',
  ended: 'ended',
  settled: 'settled',
} as const;

export type SeasonStatus = (typeof SEASON_STATUS)[keyof typeof SEASON_STATUS];

/** 赛季和当前的状态, 对应后端 season.View */
export interface Season {
  id: number;
  /** 自然键, 拼进统计周期 season:<code> */
  code: string;
  name: string;
  /** 开始时间, 包含 */
  starts_at: string;
  /** 结束时间, 不包含 */
  ends_at: string;
  /** 赛季奖励发完的时间, 没有发完时为空 */
  settled_at: string | null;
  /** 段位分继承上一个赛季的百分比, 初始分 = 基础分 + (上赛季的段位分 - 基础分) × 比例 */
  rating_carry: number;
  status: SeasonStatus;
  created_at: string;
  updated_at: string;
}

/** 当前和下一个赛季, 都可能为空 */
export interface CurrentSeasonData {
  current: Season | null;
  next: Season | null;
}

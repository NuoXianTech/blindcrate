import type {Bundle} from '../auction';

/** 赛季的状态, 由时间推算 */
export const SEASON_STATUS = {
  upcoming: 'upcoming',
  active: 'active',
  ended: 'ended',
  settled: 'settled',
} as const;

export type SeasonStatus = (typeof SEASON_STATUS)[keyof typeof SEASON_STATUS];

/** 一档赛季奖励, 赛季中的最高段位分不低于分数线时可以领取; 每人只发满足条件的最高一档 */
export interface SeasonReward {
  /** 分数线, 包含 */
  min_rating: number;
  bundle: Bundle;
}

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
  /** 赛季奖励, 按分数线从低到高 */
  rewards: SeasonReward[];
  status: SeasonStatus;
  created_at: string;
  updated_at: string;
}

/** 玩家看到的一档赛季奖励, 对应后端 season.RewardView */
export interface SeasonRewardView {
  min_rating: number;
  /** 货币在前, 物品在后 */
  lines: SeasonRewardLine[];
}

/** 奖励中的一种货币或物品 */
export interface SeasonRewardLine {
  /** 与 item_id 二选一 */
  currency?: string;
  item_id?: number;
  name: string;
  image_url: string;
  /** 货币的展示用小数位, 物品为 0 */
  decimals: number;
  /** 货币金额或物品数量 */
  amount: number;
}

/** 当前和下一个赛季, 都可能为空 */
export interface CurrentSeasonData {
  current: Season | null;
  next: Season | null;
  /** 进行中的赛季的奖励, 没有进行中的赛季时为空 */
  rewards: SeasonRewardView[];
}

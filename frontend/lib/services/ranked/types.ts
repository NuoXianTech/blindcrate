import type {Season} from '../season';

/** 自己在当前赛季的段位分, 对应后端 ranked.View */
export interface RatingView {
  /** 进行中的赛季, 赛季之外为空 */
  season: Season | null;
  rating: number;
  /** 本赛季的最高分 */
  peak: number;
  /** 本赛季计分的排位局数, 0 表示还没打过, 段位分为初始分 */
  matches: number;
  /** 当前的段位, 低于最低的分数线时为空 */
  tier: Tier | null;
  /** 下一个段位, 已是最高段位时为空 */
  next_tier: Tier | null;
}

/** 段位, 段位分不低于分数线时属于分数线最高的那个段位, 对应后端 ranked.Tier */
export interface Tier {
  id: number;
  name: string;
  /** 分数线, 包含 */
  min_rating: number;
  /** #rrggbb */
  color: string;
  created_at: string;
  updated_at: string;
}

export interface ListTiersData {
  /** 按分数线从低到高 */
  tiers: Tier[];
}

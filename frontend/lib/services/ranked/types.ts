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
}

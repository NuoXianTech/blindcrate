import type {PageParams} from '../core';
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

/** 新建或修改段位, 修改时也需提交全部字段 */
export type TierRequest = Pick<Tier, 'name' | 'min_rating' | 'color'>;

/** 可疑名单中的一个账号, 对应后端 auction.SuspectUser */
export interface SuspectUser {
  id: number;
  username: string;
  nickname: string;
  is_disabled: boolean;
  banned_until: string | null;
}

/** 一对可疑账号的处理记录, 对应后端 auction.SuspectReview */
export interface SuspectReview {
  user_a: number;
  user_b: number;
  /** 处理时两人最近一起打的对局 */
  last_match_id: number;
  note: string;
  reviewed_by: number;
  created_at: string;
  updated_at: string;
}

/** 可疑名单中的一对账号, 对应后端 auction.Suspect */
export interface Suspect {
  /** 用户 ID 较小的一个 */
  user_a: SuspectUser;
  user_b: SuspectUser;
  /** 一起打的排位局数 */
  matches: number;
  /** 其中一人成交的拍品数 */
  wins: number;
  /** 其中一人低价成交的拍品数 */
  low_wins: number;
  /** 其中一人成交时成交价占仓库价值的平均比例, 基点; 没有成交时为 0 */
  price_rate: number;
  last_match_id: number;
  last_match_at: string;
  /** 最近一次处理记录, 没处理过时为空 */
  review: SuspectReview | null;
}

export interface ListSuspectsParams extends PageParams {
  /** 不传返回全部; false 只返回待处理的, 即没处理过或处理后又一起打过排位; true 只返回其余的 */
  reviewed?: boolean;
}

export interface ListSuspectsData {
  total: number;
  suspects: Suspect[];
}

/** 把一对可疑账号标记为已处理 */
export interface ReviewSuspectRequest {
  user_a: number;
  /** 大于 user_a */
  user_b: number;
  /** 处理到的对局, 即名单中的 last_match_id */
  last_match_id: number;
  note: string;
}

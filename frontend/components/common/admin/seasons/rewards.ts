import {nanoid} from 'nanoid';
import {fromBundleDrafts, toBundleDrafts, type BundleDraft} from '@/components/common/admin/editors';
import {check} from '@/hooks/use-submit';
import type {Currency} from '@/lib/services/admin';
import type {SeasonReward} from '@/lib/services/season';
import {parseAmount} from '@/lib/utils';
import {MAX_REWARD_RATING} from './constants';

/** 编辑中的一档奖励, 分数线保留输入框里的原文 */
export interface RewardDraft {
  key: string;
  minRating: string;
  bundle: BundleDraft[];
}

/** 新建一档奖励, 分数线默认比已有的最高一档高 100 */
export function newRewardDraft(drafts: RewardDraft[]): RewardDraft {
  const ratings = drafts.map((d) => parseAmount(d.minRating) ?? 0);
  return {key: nanoid(), minRating: String(ratings.length ? Math.max(...ratings) + 100 : 0), bundle: []};
}

export function toRewardDrafts(rewards: SeasonReward[], currencies: Currency[]): RewardDraft[] {
  return rewards.map((r) => ({key: nanoid(), minRating: String(r.min_rating), bundle: toBundleDrafts(r.bundle, currencies)}));
}

/** 转为请求中的赛季奖励, 与后端 season.NormalizeRewards 的校验一致 */
export function fromRewardDrafts(drafts: RewardDraft[], currencies: Currency[]): SeasonReward[] {
  const seen = new Set<number>();
  return drafts.map((d, i) => {
    const label = `第 ${i + 1} 档奖励`;
    const rating = parseAmount(d.minRating);
    check(rating === null || rating < 0 || rating > MAX_REWARD_RATING ? `${label}: 分数线需为 0-${MAX_REWARD_RATING}` : null);
    check(seen.has(rating!) ? `${label}: 分数线不能与其他档重复` : null);
    seen.add(rating!);
    check(d.bundle.length === 0 ? `${label}: 至少一项` : null);
    return {min_rating: rating!, bundle: fromBundleDrafts(d.bundle, currencies, `${label}的`)};
  });
}

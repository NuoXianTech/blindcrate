import type {Tier} from '@/lib/services/ranked';

/** 段位分所在的段位, 与后端 ranked.TierOf 一致; tiers 需按分数线从低到高, 低于最低的分数线时为 null */
export function tierOf(tiers: Tier[], rating: number): Tier | null {
  let current: Tier | null = null;
  for (const tier of tiers) {
    if (tier.min_rating > rating) break;
    current = tier;
  }
  return current;
}

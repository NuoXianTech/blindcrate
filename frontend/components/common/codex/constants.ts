import type {CodexEntry, CodexSet} from '@/lib/services/codex';

/** 没有收集的藏品显示为剪影: 图片压成纯黑再反色, 所有主题都是暗色系, 浅色剪影更清楚 */
export const SILHOUETTE = 'brightness-0 invert opacity-20';

/** 没有收集的藏品, 后端不返回名称, 显示为问号 */
export const UNKNOWN_NAME = '???';

/** 藏品显示的名称 */
export function entryName(entry: CodexEntry) {
  return entry.collected ? entry.name : UNKNOWN_NAME;
}

/** 收集组卡片上最多显示的藏品缩略图数, 其余显示为 +N */
export const SET_THUMBNAILS = 16;

/** 收集状态筛选 */
export type CollectedFilter = 'all' | 'collected' | 'missing';

export const COLLECTED_FILTERS: { value: CollectedFilter; label: string }[] = [
  {value: 'all', label: '全部'},
  {value: 'collected', label: '已收集'},
  {value: 'missing', label: '未收集'},
];

/** 列表中的顺序: 可领取的在前, 已领取的在后, 其余保持后端的排序 */
export function setRank(set: CodexSet) {
  if (set.claimable) return 0;
  return set.claimed ? 2 : 1;
}

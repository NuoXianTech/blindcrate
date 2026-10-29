import {ROUTES} from '@/lib/access';
import type {Suspect} from '@/lib/services/admin';

/** 可疑名单每页条数 */
export const PAGE_SIZE = 20;

/** 处理状态筛选 */
export type ReviewFilter = 'pending' | 'reviewed' | 'all';

export const REVIEW_FILTERS: { value: ReviewFilter; label: string }[] = [
  {value: 'pending', label: '待处理'},
  {value: 'reviewed', label: '已处理'},
  {value: 'all', label: '全部'},
];

/** 处理状态筛选对应的查询参数 */
export function reviewedParam(filter: ReviewFilter) {
  return filter === 'all' ? undefined : filter === 'reviewed';
}

/** 是否已处理, 处理后又一起打过排位时算待处理 */
export function isReviewed(s: Suspect) {
  return !!s.review && s.review.last_match_id >= s.last_match_id;
}

/** 基点格式化为百分比, 如 4250 -> "42.5%" */
export function formatRate(bp: number) {
  return `${Number((bp / 100).toFixed(1))}%`;
}

/** 对局管理页, 只看两个人一起打的对局 */
export function pairMatchesHref(s: Suspect) {
  return `${ROUTES.adminMatches}?user_id=${s.user_a.id}&other_id=${s.user_b.id}`;
}

/** 用户管理页, 按用户名搜索 */
export function userHref(username: string) {
  return `${ROUTES.adminUsers}?keyword=${encodeURIComponent(username)}`;
}

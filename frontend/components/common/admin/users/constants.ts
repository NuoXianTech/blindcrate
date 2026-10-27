import type {AdminUser, ListUsersParams} from '@/lib/services/admin';
import {ROUTES} from '@/lib/access';

/** 用户列表每页条数 */
export const PAGE_SIZE = 20;

/** 用户状态筛选 */
export type StatusFilter = 'all' | 'admin' | 'banned' | 'disabled';

export const STATUS_FILTERS: { value: StatusFilter; label: string }[] = [
  {value: 'all', label: '全部用户'},
  {value: 'admin', label: '管理员'},
  {value: 'banned', label: '封禁中'},
  {value: 'disabled', label: '已停用'},
];

/** 可选的封禁时长 */
export const BAN_DURATIONS = [
  {value: '1h', label: '1 小时', ms: 3600_000},
  {value: '1d', label: '1 天', ms: 86400_000},
  {value: '7d', label: '7 天', ms: 7 * 86400_000},
  {value: '30d', label: '30 天', ms: 30 * 86400_000},
];

/** 与后端 wallet.MaxAmount、inventory.MaxQuantity 一致 */
export const MAX_AMOUNT = 1_000_000_000_000;
export const MAX_QUANTITY = 1_000_000;
/** 一次拉取的物品种类上限, 与后端分页上限一致 */
export const ITEM_LIMIT = 100;

/** 加款/扣款、发放/扣除 */
export type Direction = 'add' | 'sub';

/** 从现在起 ms 毫秒后的时间 */
export function banUntil(ms: number) {
  return new Date(Date.now() + ms).toISOString();
}

/** 是否在封禁期内 */
export function isBanned(user: AdminUser) {
  return !!user.banned_until && new Date(user.banned_until) > new Date();
}

/** 状态筛选对应的查询参数 */
export function statusParams(status: StatusFilter): Partial<ListUsersParams> {
  switch (status) {
    case 'admin':
      return {is_admin: true};
    case 'banned':
      return {is_banned: true};
    case 'disabled':
      return {is_disabled: true};
    default:
      return {};
  }
}

/** 流水查询页, 带上用户筛选 */
export function transactionsHref(tab: 'wallet' | 'inventory', userID: number) {
  return `${ROUTES.adminTransactions}?tab=${tab}&user_id=${userID}`;
}
